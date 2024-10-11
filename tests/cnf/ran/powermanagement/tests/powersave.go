package tests

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/mco"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/nto"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/raninittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/ranparam"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/powermanagement/internal/collect"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/powermanagement/internal/helper"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/powermanagement/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/internal/cluster"
	performancev2 "github.com/openshift/cluster-node-tuning-operator/pkg/apis/performanceprofile/v2"
	"github.com/openshift/cluster-node-tuning-operator/pkg/performanceprofile/controller/performanceprofile/components"
	mcov1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/cpuset"
)

var _ = Describe("Per-core runtime power states tuning", Label(tsparams.LabelPowerSaveTestCases), Ordered, func() {
	var (
		nodeList                []*nodes.Builder
		nodeName                string
		perfProfile             *nto.Builder
		originalPerfProfileSpec performancev2.PerformanceProfileSpec
		err                     error
	)

	BeforeAll(func() {
		nodeList, err = nodes.List(Spoke1APIClient)
		Expect(err).ToNot(HaveOccurred(), "Failed to get nodes")
		Expect(len(nodeList)).To(Equal(1), "Currently only SNO clusters are supported")

		nodeName = nodeList[0].Object.Name
		perfProfile, err = helper.GetPerformanceProfileWithCPUSet()
		Expect(err).ToNot(HaveOccurred(), "Failed to get performance profile")

		originalPerfProfileSpec = perfProfile.Object.Spec
	})

	AfterAll(func() {
		perfProfile, err = helper.GetPerformanceProfileWithCPUSet()
		Expect(err).ToNot(HaveOccurred(), "Failed to get performance profile")

		if reflect.DeepEqual(perfProfile.Object.Spec, originalPerfProfileSpec) {
			glog.V(tsparams.LogLevel).Info("Performance profile did not change, exiting")

			return
		}

		By("restoring performance profile to original specs")
		perfProfile.Definition.Spec = originalPerfProfileSpec

		_, err = perfProfile.Update(true)
		Expect(err).ToNot(HaveOccurred())
		mcp, err := mco.Pull(Spoke1APIClient, "master")
		Expect(err).ToNot(HaveOccurred(), "Failed to get machineconfigpool")

		err = mcp.WaitToBeInCondition(mcov1.MachineConfigPoolUpdating, corev1.ConditionTrue, 2*tsparams.PowerSaveTimeout)
		Expect(err).ToNot(HaveOccurred(), "Failed to wait for machineconfigpool to be updating")

		err = mcp.WaitForUpdate(3 * tsparams.PowerSaveTimeout)
		Expect(err).ToNot(HaveOccurred(), "Failed to wait for machineconfigpool to be updated")
	})

	// 54571 - Install SNO node with standard DU profile that does not include WorkloadHints
	It("verifies expected kernel parameters with no workload hints "+
		"specified in PerformanceProfile", reportxml.ID("54571"), func() {
		workloadHints := perfProfile.Definition.Spec.WorkloadHints
		if workloadHints != nil {
			Skip("WorkloadHints already present in perfProfile.Spec")
		}

		By("checking for expected kernel parameters")
		cmdline, err := cluster.ExecCommandOnSNOWithRetries(Spoke1APIClient,
			ranparam.RetryCount, ranparam.RetryInterval, "cat /proc/cmdline")
		Expect(err).ToNot(HaveOccurred(), "Failed to cat /proc/cmdline")

		// Expected default set of kernel parameters when no WorkloadHints are specified in PerformanceProfile
		requiredKernelParms := []string{
			"nohz_full=[0-9,-]+",
			"tsc=nowatchdog",
			"nosoftlockup",
			"nmi_watchdog=0",
			"mce=off",
			"skew_tick=1",
			"intel_pstate=disable",
		}
		for _, parameter := range requiredKernelParms {
			By(fmt.Sprintf("checking /proc/cmdline for %s", parameter))
			rePattern := regexp.MustCompile(parameter)
			Expect(rePattern.FindStringIndex(cmdline)).
				ToNot(BeNil(), "Kernel parameter %s is missing from cmdline", parameter)
		}
	})

	// 54572 - Enable powersave at node level and then enable performance at node level
	It("enables powersave at node level and then enables performance at node level", reportxml.ID("54572"), func() {
		By("patching the performance profile with the workload hints")
		err := helper.SetPowerModeAndWaitForMcpUpdate(perfProfile, *nodeList[0], true, false, true)
		Expect(err).ToNot(HaveOccurred(), "Failed to set power mode")

		cmdline, err := cluster.ExecCommandOnSNOWithRetries(Spoke1APIClient,
			ranparam.RetryCount, ranparam.RetryInterval, "cat /proc/cmdline")
		Expect(err).ToNot(HaveOccurred(), "Failed to cat /proc/cmdline")
		Expect(cmdline).
			To(ContainSubstring("intel_pstate=passive"), "Kernel parameter intel_pstate=passive missing from /proc/cmdline")
		Expect(cmdline).
			ToNot(ContainSubstring("intel_pstate=disable"), "Kernel parameter intel_pstate=disable found on /proc/cmdline")
	})

	// 54574 - Enable powersave at node level and then enable high performance at node level, check power
	// consumption with no workload pods.
	It("enables powersave, enables high performance at node level, "+
		"and checks power consumption with no workload pods", reportxml.ID("54574"), func() {
		testPodAnnotations := map[string]string{
			"cpu-load-balancing.crio.io": "disable",
			"cpu-quota.crio.io":          "disable",
			"irq-load-balancing.crio.io": "disable",
			"cpu-c-states.crio.io":       "disable",
			"cpu-freq-governor.crio.io":  "performance",
		}

		cpuLimit := resource.MustParse("2")
		memLimit := resource.MustParse("100Mi")

		By("defining the test pod")
		testpod, err := helper.DefineQoSTestPod(
			tsparams.TestingNamespace, nodeName, cpuLimit.String(), cpuLimit.String(), memLimit.String(), memLimit.String())
		Expect(err).ToNot(HaveOccurred(), "Failed to define test pod")

		testpod.Definition.Annotations = testPodAnnotations
		runtimeClass := fmt.Sprintf("%s-%s", components.ComponentNamePrefix, perfProfile.Definition.Name)
		testpod.Definition.Spec.RuntimeClassName = &runtimeClass

		DeferCleanup(func() {
			// Delete the test pod if it's still around when the function returns, like in a test case failure.
			if testpod.Exists() {
				By("deleting the test pod in case of a failure")
				_, err = testpod.DeleteAndWait(tsparams.PowerSaveTimeout)
				Expect(err).ToNot(HaveOccurred(), "Failed to delete test pod in case of failure")
			}
		})

		By("creating the test pod")
		testpod, err = testpod.CreateAndWaitUntilRunning(tsparams.PowerSaveTimeout)
		Expect(err).ToNot(HaveOccurred(), "Failed to create pod")
		Expect(testpod.Object.Status.QOSClass).To(Equal(corev1.PodQOSGuaranteed),
			"Test pod does not have QoS class of Guaranteed")

		cpusetOutput, err := testpod.ExecCommand([]string{"sh", `-c`, "taskset -c -p $$ | cut -d: -f2"})
		Expect(err).ToNot(HaveOccurred(), "Failed to get cpuset")

		By("verifying powersetting of cpus used by the test pod")
		trimmedOutput := strings.TrimSpace(cpusetOutput.String())
		cpusUsed, err := cpuset.Parse(trimmedOutput)
		Expect(err).ToNot(HaveOccurred(), "Failed to parse cpuset output")

		targetCpus := cpusUsed.List()
		checkCPUGovernorsAndResumeLatency(targetCpus, "n/a", "performance")

		By("verifying the rest of cpus have default power setting")
		allCpus := nodeList[0].Object.Status.Capacity.Cpu()
		cpus, err := cpuset.Parse(fmt.Sprintf("0-%d", allCpus.Value()-1))
		Expect(err).ToNot(HaveOccurred(), "Failed to parse cpuset")

		otherCPUs := cpus.Difference(cpusUsed)
		// Verify cpus not assigned to the pod have default power settings.
		checkCPUGovernorsAndResumeLatency(otherCPUs.List(), "0", "performance")

		By("deleting the test pod")
		_, err = testpod.DeleteAndWait(tsparams.PowerSaveTimeout)
		Expect(err).ToNot(HaveOccurred(), "Failed to delete test pod")

		By("verifying after the test pod was deleted cpus assigned to container have default powersave settings")
		checkCPUGovernorsAndResumeLatency(targetCpus, "0", "performance")
	})

	When("collecting power usage metrics", Ordered, func() {
		var (
			samplingInterval time.Duration
			powerState       string
		)

		BeforeAll(func() {
			if BMCClient == nil {
				Skip("Collecting power usage metrics requires the BMC configuration be set.")
			}

			samplingInterval, err = time.ParseDuration(RANConfig.MetricSamplingInterval)
			Expect(err).ToNot(HaveOccurred(), "Failed to parse metric sampling interval")

			// Determine power state to be used as a tag for the metric
			powerState, err = collect.GetPowerState(perfProfile)
			Expect(err).ToNot(HaveOccurred(), "Failed to get power state for the performance profile")
		})

		It("checks power usage for 'noworkload' scenario", func() {
			duration, err := time.ParseDuration(RANConfig.NoWorkloadDuration)
			Expect(err).ToNot(HaveOccurred(), "Failed to parse no workload duration")

			compMap, err := collect.CollectPowerMetricsWithNoWorkload(duration, samplingInterval, powerState)
			Expect(err).ToNot(HaveOccurred(), "Failed to collect power metrics with no workload")

			// Persist power usage metric to ginkgo report for further processing in pipeline.
			for metricName, metricValue := range compMap {
				GinkgoWriter.Printf("%s: %s\n", metricName, metricValue)
			}
		})

		It("checks power usage for 'steadyworkload' scenario", func() {
			duration, err := time.ParseDuration(RANConfig.WorkloadDuration)
			Expect(err).ToNot(HaveOccurred(), "Failed to parse steady workload duration")

			compMap, err := collect.CollectPowerMetricsWithSteadyWorkload(
				duration, samplingInterval, powerState, perfProfile, nodeName)
			Expect(err).ToNot(HaveOccurred(), "Failed to collect power metrics with steady workload")

			// Persist power usage metric to ginkgo report for further processing in pipeline.
			for metricName, metricValue := range compMap {
				GinkgoWriter.Printf("%s: %s\n", metricName, metricValue)
			}
		})
	})
})

// checkCPUGovernorsAndResumeLatency checks power and latency settings of the cpus.
func checkCPUGovernorsAndResumeLatency(cpus []int, pmQos, governor string) {
	for _, cpu := range cpus {
		command := fmt.Sprintf("cat /sys/devices/system/cpu/cpu%d/power/pm_qos_resume_latency_us", cpu)

		// Eventually allows for retries on malformed output, but we use StopTrying since the command failing is
		// a failure, not just a malformed output.
		Eventually(func() (string, error) {
			output, err := cluster.ExecCommandOnSNOWithRetries(Spoke1APIClient,
				ranparam.RetryCount, ranparam.RetryInterval, command)
			if err != nil {
				return "", StopTrying(fmt.Sprintf("Failed to check cpu %d resume latency", cpu)).Wrap(err)
			}

			return strings.TrimSpace(output), nil
		}, 10*time.Second, time.Second).Should(Equal(pmQos))

		command = fmt.Sprintf("cat /sys/devices/system/cpu/cpu%d/cpufreq/scaling_governor", cpu)

		Eventually(func() (string, error) {
			output, err := cluster.ExecCommandOnSNOWithRetries(Spoke1APIClient,
				ranparam.RetryCount, ranparam.RetryInterval, command)
			if err != nil {
				return "", StopTrying(fmt.Sprintf("Failed to check cpu %d scaling governor", cpu)).Wrap(err)
			}

			return strings.TrimSpace(output), nil
		}, 10*time.Second, time.Second).Should(Equal(governor))
	}
}
