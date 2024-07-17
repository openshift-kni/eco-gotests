package helper

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/mco"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/nto" //nolint:misspell
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/cluster"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/raninittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/powermanagement/internal/tsparams"
	performancev2 "github.com/openshift/cluster-node-tuning-operator/pkg/apis/performanceprofile/v2"
	mcov1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/cpuset"
	"k8s.io/utils/ptr"
)

// GetPerformanceProfileWithCPUSet returns the first performance profile found with reserved and isolated cpuset.
func GetPerformanceProfileWithCPUSet() (*nto.Builder, error) {
	profileBuilders, err := nto.ListProfiles(Spoke1APIClient)
	if err != nil {
		return nil, err
	}

	for _, profileBuilder := range profileBuilders {
		if profileBuilder.Object.Spec.CPU != nil &&
			profileBuilder.Object.Spec.CPU.Reserved != nil &&
			profileBuilder.Object.Spec.CPU.Isolated != nil {
			return profileBuilder, nil
		}
	}

	return nil, errors.New("failed to find performance profile with reserved and isolated CPU set")
}

// SetPowerModeAndWaitForMcpUpdate updates the performance profile with the given workload hints,
// and waits for the mcp update.
func SetPowerModeAndWaitForMcpUpdate(perfProfile *nto.Builder, node nodes.Builder, perPodPowerManagement,
	highPowerConsumption, realTime bool) error {
	glog.V(tsparams.LogLevel).Infof("Set powersave mode on performance profile")

	perfProfile.Definition.Spec.WorkloadHints = &performancev2.WorkloadHints{
		PerPodPowerManagement: ptr.To[bool](perPodPowerManagement),
		HighPowerConsumption:  ptr.To[bool](highPowerConsumption),
		RealTime:              ptr.To[bool](realTime),
	}

	_, err := perfProfile.Update(true)
	if err != nil {
		return err
	}

	mcp, err := mco.Pull(Spoke1APIClient, "master")
	if err != nil {
		return err
	}

	err = mcp.WaitToBeInCondition(mcov1.MachineConfigPoolUpdating, corev1.ConditionTrue, 2*tsparams.PowerSaveTimeout)
	if err != nil {
		return err
	}

	err = mcp.WaitForUpdate(3 * tsparams.PowerSaveTimeout)
	if err != nil {
		return err
	}

	err = node.WaitUntilReady(tsparams.PowerSaveTimeout)

	return err
}

// SetCPUFreq updates the performance profile with the given isolated and reserved
// core frequencies and verifies that the frequencies have been updated on the spoke cluster.
func SetCPUFreq(
	perfProfile *nto.Builder,
	desiredIsolatedCoreFreq *performancev2.CPUfrequency,
	desiredReservedCoreFreq *performancev2.CPUfrequency) error {
	glog.V(tsparams.LogLevel).Infof("Set Reserved and Isolated CPU Frequency on performance profile")

	if perfProfile.Definition.Spec.HardwareTuning == nil {
		perfProfile.Definition.Spec.HardwareTuning = &performancev2.HardwareTuning{}
	}

	// Update PerfProfile with new CPU Frequencies
	perfProfile.Definition.Spec.HardwareTuning.IsolatedCpuFreq = desiredIsolatedCoreFreq
	perfProfile.Definition.Spec.HardwareTuning.ReservedCpuFreq = desiredReservedCoreFreq
	_, err := perfProfile.Update(true)

	if err != nil {
		return err
	}

	// Determine isolated CPU number from IsolatedCPUset
	isolatedCPUSet, err := cpuset.Parse(string(*perfProfile.Object.Spec.CPU.Isolated))
	if err != nil {
		return err
	}

	// Determine reserved CPU number from reservedCPUset
	reservedCPUSet, err := cpuset.Parse(string(*perfProfile.Object.Spec.CPU.Reserved))
	if err != nil {
		return err
	}

	isolatedCPUsList := isolatedCPUSet.List()
	isolatedCPUNumber := isolatedCPUsList[0]
	reservedCPUsList := reservedCPUSet.List()
	reservedCCPUNumber := reservedCPUsList[0]

	spokeCommandIsolatedCPUs := fmt.Sprintf("cat /sys/devices/system/cpu/cpufreq/policy%v/scaling_max_freq",
		isolatedCPUNumber)

	spokeCommandReservedCPUs := fmt.Sprintf("cat /sys/devices/system/cpu/cpufreq/policy%v/scaling_max_freq",
		reservedCCPUNumber)

	// Wait for Isolated CPU Frequency to be updated.
	err = wait.PollUntilContextTimeout(
		context.TODO(), 2*time.Second, 1*time.Minute, true, func(ctx context.Context) (bool, error) {
			// Get current isolated core frequency from spoke cluster and compare to desired frequency
			cmdOut, err := cluster.ExecCommandOnSNO(Spoke1APIClient, 3, spokeCommandIsolatedCPUs)

			if err != nil {
				return false, fmt.Errorf("command failed: %s", cmdOut)
			}

			cmdOut = strings.TrimSpace(cmdOut)
			currIsolatedCoreFreq, err := strconv.Atoi(cmdOut)

			if err != nil {
				glog.V(tsparams.LogLevel).Infof("converting cpu frequency %s to an int failed: %w", cmdOut, err)

				return false, nil
			}

			if currIsolatedCoreFreq != int(*desiredIsolatedCoreFreq) {
				return false, nil
			}

			// Get current isolated core frequency from spoke cluster and compare to desired frequency
			cmdOut, err = cluster.ExecCommandOnSNO(Spoke1APIClient, 3, spokeCommandReservedCPUs)

			if err != nil {
				return false, fmt.Errorf("command failed: %s", cmdOut)
			}

			cmdOut = strings.TrimSpace(cmdOut)
			currReservedFreq, err := strconv.Atoi(cmdOut)

			if err != nil {
				glog.V(tsparams.LogLevel).Infof("converting cpu frequency %s to an int failed: %w", cmdOut, err)

				return false, nil
			}

			if currReservedFreq != int(*desiredReservedCoreFreq) {
				return false, nil
			}

			return true, nil
		})

	return err
}

// DefineQoSTestPod defines test pod with given cpu and memory resources.
func DefineQoSTestPod(namespace, nodeName, cpuReq, cpuLimit, memReq, memLimit string) (*pod.Builder, error) {
	var err error

	pod := pod.NewBuilder(
		Spoke1APIClient, "qos-test-pod", namespace, RANConfig.CnfTestImage,
	).DefineOnNode(nodeName)
	pod, err = redefineContainerResources(pod, cpuReq, cpuLimit, memReq, memLimit)

	return pod, err
}

// GetPowerState determines the power state from the workloadHints object of the PerformanceProfile.
func GetPowerState(perfProfile *nto.Builder) (string, error) {
	workloadHints := perfProfile.Definition.Spec.WorkloadHints

	if workloadHints == nil {
		// No workloadHints object -> default is Performance mode
		return tsparams.PerformanceMode, nil
	}

	realTime := *workloadHints.RealTime
	highPowerConsumption := *workloadHints.HighPowerConsumption
	perPodPowerManagement := *workloadHints.PerPodPowerManagement

	switch {
	case realTime && !highPowerConsumption && !perPodPowerManagement:
		return tsparams.PerformanceMode, nil
	case realTime && highPowerConsumption && !perPodPowerManagement:
		return tsparams.HighPerformanceMode, nil
	case realTime && !highPowerConsumption && perPodPowerManagement:
		return tsparams.PowerSavingMode, nil
	default:
		return "", errors.New("unknown workloadHints power state configuration")
	}
}

// CollectPowerMetricsWithNoWorkload collects metrics with no workload.
func CollectPowerMetricsWithNoWorkload(duration, interval time.Duration, tag string) (map[string]string, error) {
	glog.V(tsparams.LogLevel).Infof("Wait for %s for noworkload scenario", duration)

	return collectPowerUsageMetrics(duration, interval, "noworkload", tag)
}

// CollectPowerMetricsWithSteadyWorkload collects power metrics with steady workload scenario.
func CollectPowerMetricsWithSteadyWorkload(
	duration, interval time.Duration, tag string, perfProfile *nto.Builder, nodeName string) (map[string]string, error) {
	// stressNg cpu count is roughly 75% of total isolated cores.
	// 1 cpu will be used by other consumer pods, such as process-exporter, cnf-ran-gotests-priv.
	isolatedCPUSet, err := cpuset.Parse(string(*perfProfile.Object.Spec.CPU.Isolated))
	if err != nil {
		return nil, err
	}

	stressNgCPUCount := (isolatedCPUSet.Size() - 1) * 300 / 400
	stressngMaxPodCount := 50

	stressNgPods, err := deployStressNgPods(stressNgCPUCount, stressngMaxPodCount, nodeName)
	if err != nil {
		return nil, err
	}

	if len(stressNgPods) < 1 {
		return nil, errors.New("not enough stress-ng pods to run test")
	}

	glog.V(tsparams.LogLevel).Infof("Wait for %s for steadyworkload scenario\n", duration.String())
	result, collectErr := collectPowerUsageMetrics(duration, interval, "steadyworkload", tag)

	// Delete stress-ng pods regardless of whether collectPowerUsageMetrics failed.
	for _, stressPod := range stressNgPods {
		_, err = stressPod.DeleteAndWait(tsparams.PowerSaveTimeout)
		if err != nil {
			return nil, err
		}
	}

	// If deleting stress-ng pods was successful, still return error from collectPowerUsageMetrics.
	return result, collectErr
}

// redefineContainerResources redefines a pod builder with CPU and Memory resources in first container
// Use empty string to skip a resource. e.g., cpuLimit="".
func redefineContainerResources(
	pod *pod.Builder, cpuRequest string, cpuLimit string, memoryRequest string, memoryLimit string) (*pod.Builder, error) {
	if len(pod.Definition.Spec.Containers) < 1 {
		return pod, errors.New("pod has no containers to redefine resources for")
	}

	pod.Definition.Spec.Containers[0].Resources.Requests = corev1.ResourceList{}
	pod.Definition.Spec.Containers[0].Resources.Limits = corev1.ResourceList{}

	if cpuLimit != "" {
		pod.Definition.Spec.Containers[0].Resources.Limits["cpu"] = resource.MustParse(cpuLimit)
	}

	if cpuRequest != "" {
		pod.Definition.Spec.Containers[0].Resources.Requests["cpu"] = resource.MustParse(cpuRequest)
	}

	if memoryLimit != "" {
		pod.Definition.Spec.Containers[0].Resources.Limits["memory"] = resource.MustParse(memoryLimit)
	}

	if memoryRequest != "" {
		pod.Definition.Spec.Containers[0].Resources.Requests["memory"] = resource.MustParse(memoryRequest)
	}

	return pod, nil
}

// collectPowerUsageMetrics collects power usage metrics.
func collectPowerUsageMetrics(duration, interval time.Duration, scenario, tag string) (map[string]string, error) {
	var powerMeasurements []float64

	endTime := time.Now().Add(duration)
	for time.Now().Before(endTime) {
		power, err := BMCClient.PowerUsage()
		if err != nil {
			glog.V(tsparams.LogLevel).Infof("error getting power usage: %w", err)

			continue
		}

		powerMeasurements = append(powerMeasurements, float64(power))

		time.Sleep(interval)
	}

	glog.V(tsparams.LogLevel).Info("Finished collecting power usage, waiting for results")

	if len(powerMeasurements) < 1 {
		return nil, errors.New("no power usage metrics were retrieved")
	}

	// Compute power metrics.
	return computePowerUsageStatistics(powerMeasurements, interval, scenario, tag)
}

// deployStressNgPods deploys the stress-ng workload pods.
func deployStressNgPods(stressNgCPUCount, stressngMaxPodCount int, nodeName string) ([]*pod.Builder, error) {
	// Determine cpu requests for stress-ng pods.
	stressngPodsCPUs := parsePodCountAndCpus(stressngMaxPodCount, stressNgCPUCount)

	// Create and wait for stress-ng pods to be Ready
	glog.V(tsparams.LogLevel).
		Infof("Creating up to %d stress-ng pods with total %d cpus", stressngMaxPodCount, stressNgCPUCount)

	stressngPods := []*pod.Builder{}

	for i, cpuReq := range stressngPodsCPUs {
		pod := defineStressPod(nodeName, cpuReq, false, fmt.Sprintf("ran-stressng-pod-%d", i))

		_, err := pod.Create()
		if err != nil {
			return nil, err
		}

		stressngPods = append(stressngPods, pod)
	}

	err := waitForPodsHealthy(stressngPods, 2*tsparams.PowerSaveTimeout)
	if err != nil {
		return nil, err
	}

	glog.V(tsparams.LogLevel).
		Infof("%d stress-ng pods with total %d cpus are created and running", len(stressngPods), stressNgCPUCount)

	return stressngPods, nil
}

// waitForPodsHealthy waits for given pods to appear and healthy.
func waitForPodsHealthy(pods []*pod.Builder, timeout time.Duration) error {
	for _, singlePod := range pods {
		tempPod, err := pod.Pull(Spoke1APIClient, singlePod.Definition.Name,
			singlePod.Object.Namespace)
		if err != nil {
			return err
		}

		err = tempPod.WaitUntilCondition(corev1.ContainersReady, timeout)
		if err != nil {
			return err
		}
	}

	return nil
}

// defineStressPod returns stress-ng pod definition.
func defineStressPod(nodeName string, cpus int, guaranteed bool, name string) *pod.Builder {
	stressngImage := RANConfig.StressngTestImage
	envVars := []corev1.EnvVar{{Name: "INITIAL_DELAY_SEC", Value: "60"}}
	cpuLimit := strconv.Itoa(cpus)
	memoryLimit := "100M"

	if !guaranteed {
		// Override CMDLINE for non-guaranteed pod to avoid specifying taskset
		envVars = append(envVars, corev1.EnvVar{Name: "CMDLINE", Value: fmt.Sprintf("--cpu %d --cpu-load 50", cpus)})
		cpuLimit = fmt.Sprintf("%dm", cpus*1200)
		memoryLimit = "200M"
	}

	stressPod := pod.NewBuilder(Spoke1APIClient, name, tsparams.TestingNamespace, stressngImage)
	stressPod = stressPod.DefineOnNode(nodeName)
	stressPod.RedefineDefaultContainer(corev1.Container{
		Name:            "stress-ng",
		Image:           stressngImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Env:             envVars,
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(cpuLimit),
				corev1.ResourceMemory: resource.MustParse(memoryLimit),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(strconv.Itoa(cpus)),
				corev1.ResourceMemory: resource.MustParse("100M"),
			},
		},
	})

	return stressPod
}

func parsePodCountAndCpus(maxPodCount, cpuCount int) []int {
	podCount := int(math.Min(float64(cpuCount), float64(maxPodCount)))
	cpuPerPod := cpuCount / podCount
	cpus := []int{}

	for i := 1; i <= podCount-1; i++ {
		cpus = append(cpus, cpuPerPod)
	}
	cpus = append(cpus, cpuCount-cpuPerPod*(podCount-1))

	return cpus
}

// computePowerUsageStatistics computes the power usage summary statistics.
func computePowerUsageStatistics(
	instantPowerData []float64,
	samplingInterval time.Duration,
	scenario,
	tag string) (map[string]string, error) {
	glog.V(tsparams.LogLevel).Infof("Power usage measurements for %s:\n%v\n", scenario, instantPowerData)

	compMap := make(map[string]string)

	compMap[fmt.Sprintf("%s_%s_%s", tsparams.RanPowerMetricTotalSamples, scenario, tag)] =
		fmt.Sprintf("%d", len(instantPowerData))
	compMap[fmt.Sprintf("%s_%s_%s", tsparams.RanPowerMetricSamplingIntervalSeconds, scenario, tag)] =
		fmt.Sprintf("%.0f", samplingInterval.Seconds())

	minInstantaneousPower, err := min(instantPowerData)
	if err != nil {
		return compMap, err
	}

	compMap[fmt.Sprintf("%s_%s_%s", tsparams.RanPowerMetricMinInstantPower, scenario, tag)] =
		fmt.Sprintf("%.7f", minInstantaneousPower)

	maxInstantaneousPower, err := max(instantPowerData)
	if err != nil {
		return compMap, err
	}

	compMap[fmt.Sprintf("%s_%s_%s", tsparams.RanPowerMetricMaxInstantPower, scenario, tag)] =
		fmt.Sprintf("%.7f", maxInstantaneousPower)

	meanInstantaneousPower, err := mean(instantPowerData)
	if err != nil {
		return compMap, err
	}

	compMap[fmt.Sprintf("%s_%s_%s", tsparams.RanPowerMetricMeanInstantPower, scenario, tag)] =
		fmt.Sprintf("%.7f", meanInstantaneousPower)

	stdDevInstantaneousPower, err := stdDev(instantPowerData)
	if err != nil {
		return compMap, err
	}

	compMap[fmt.Sprintf("%s_%s_%s", tsparams.RanPowerMetricStdDevInstantPower, scenario, tag)] =
		fmt.Sprintf("%.7f", stdDevInstantaneousPower)

	medianInstantaneousPower, err := median(instantPowerData)
	if err != nil {
		return compMap, err
	}

	compMap[fmt.Sprintf("%s_%s_%s", tsparams.RanPowerMetricMedianInstantPower, scenario, tag)] =
		fmt.Sprintf("%.7f", medianInstantaneousPower)

	return compMap, nil
}
