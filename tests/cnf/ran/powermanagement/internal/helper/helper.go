package helper

import (
	"context"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/mco"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/nto" //nolint:misspell
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/raninittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/powermanagement/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/internal/cluster"
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
	profileBuilders, err := nto.ListProfiles(raninittools.APIClient)
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

	mcp, err := mco.Pull(raninittools.APIClient, "master")
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

// DefineQoSTestPod defines test pod with given cpu and memory resources.
func DefineQoSTestPod(
	namespace namespace.Builder, nodeName, cpuReq, cpuLimit, memReq, memLimit string) (*pod.Builder, error) {
	_, err := namespace.Create()
	if err != nil {
		return nil, err
	}

	pod := pod.NewBuilder(
		raninittools.APIClient, "qos-test-pod", namespace.Definition.Name, raninittools.RANConfig.CnfTestImage,
	).DefineOnNode(nodeName)
	pod, err = redefineContainerResources(pod, cpuReq, cpuLimit, memReq, memLimit)

	return pod, err
}

// ExecCommandOnSNO executes a command on a single node cluster and returns the stdout.
func ExecCommandOnSNO(shellCmd string) (string, error) {
	outputs, err := cluster.ExecCmdWithStdout(raninittools.APIClient, shellCmd)
	if err != nil {
		return "", err
	}

	if len(outputs) != 1 {
		return "", errors.New("expected results from only one node")
	}

	for _, output := range outputs {
		return output, nil
	}

	// unreachable
	return "", nil
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

// GetIpmiPod sets up a pod to use ipmitool in the privileged namespace.
func GetIpmiPod(nodeName string) (*pod.Builder, error) {
	ipmiPod := pod.NewBuilder(
		raninittools.APIClient, "ipmi-test-pod", tsparams.PrivPodNamespace, raninittools.RANConfig.IpmiToolImage,
	).DefineOnNode(nodeName).WithPrivilegedFlag().RedefineDefaultCMD([]string{"sleep", "86400"})

	return ipmiPod.CreateAndWaitUntilRunning(tsparams.PowerSaveTimeout)
}

// CollectPowerMetricsWithNoWorkload collects metrics with no workload.
func CollectPowerMetricsWithNoWorkload(duration, samplingInterval time.Duration,
	tag string, ipmiPod *pod.Builder) (map[string]string, error) {
	glog.V(tsparams.LogLevel).Infof("Wait for %s for noworkload scenario\n", duration.String())

	return collectPowerUsageMetrics(duration, samplingInterval, "noworkload", tag, ipmiPod)
}

// CollectPowerMetricsWithSteadyWorkload collects power metrics with steady workload scenario.
func CollectPowerMetricsWithSteadyWorkload(duration, samplingInterval time.Duration, tag string,
	perfProfile *nto.Builder, ipmiPod *pod.Builder, nodeName string) (map[string]string, error) {
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
	result, collectErr := collectPowerUsageMetrics(duration, samplingInterval, "steadyworkload", tag, ipmiPod)

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
func collectPowerUsageMetrics(duration, samplingInterval time.Duration, scenario,
	tag string, ipmiPod *pod.Builder) (map[string]string, error) {
	startTime := time.Now()
	expectedEndTime := startTime.Add(duration)
	powerMeasurements := make(map[time.Time]map[string]float64)

	stopSampling := func(t time.Time) bool {
		if t.After(startTime) && t.Before(expectedEndTime) {
			return false
		}

		return true
	}

	var sampleGroup sync.WaitGroup

	err := wait.PollUntilContextTimeout(
		context.TODO(), samplingInterval, duration, true, func(context.Context) (bool, error) {
			sampleGroup.Add(1)

			timestamp := time.Now()
			go func(t time.Time) {
				defer sampleGroup.Done()

				out, err := getHostPowerUsage(ipmiPod)
				if err == nil {
					powerMeasurements[t] = out
				}
			}(timestamp)

			return stopSampling(timestamp.Add(samplingInterval)), nil
		})

	// Wait for all the tasks to complete.
	sampleGroup.Wait()

	if err != nil {
		return nil, err
	}

	glog.V(tsparams.LogLevel).Infof("Power usage test started: %v\nPower usage test ended: %v\n", startTime, time.Now())

	if len(powerMeasurements) < 1 {
		return nil, errors.New("no power usage metrics were retrieved")
	}

	// Compute power metrics.
	return computePowerUsageStatistics(powerMeasurements, samplingInterval, scenario, tag)
}

// getHostPowerUsage retrieve host power utilization metrics queried via ipmitool command.
func getHostPowerUsage(ipmiPod *pod.Builder) (map[string]float64, error) {
	output, err := ipmiPod.ExecCommand([]string{"ipmitool", "dcmi", "power", "reading"})
	if err != nil {
		glog.V(tsparams.LogLevel).Infof("failed to get power reading with ipmitool: %w", err)

		return nil, fmt.Errorf("failed to get power reading with ipmitool: %w", err)
	}

	// Parse the ipmitool string output and return the result.
	return parseIpmiPowerOutput(output.String())
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
		tempPod, err := pod.Pull(raninittools.APIClient, singlePod.Definition.Name,
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
	stressngImage := raninittools.RANConfig.StressngTestImage
	envVars := []corev1.EnvVar{{Name: "INITIAL_DELAY_SEC", Value: "60"}}
	cpuLimit := strconv.Itoa(cpus)
	memoryLimit := "100M"

	if !guaranteed {
		// Override CMDLINE for non-guaranteed pod to avoid specifying taskset
		envVars = append(envVars, corev1.EnvVar{Name: "CMDLINE", Value: fmt.Sprintf("--cpu %d --cpu-load 50", cpus)})
		cpuLimit = fmt.Sprintf("%dm", cpus*1200)
		memoryLimit = "200M"
	}

	stressPod := pod.NewBuilder(raninittools.APIClient, name, tsparams.TestingNamespace, stressngImage)
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

// parseIpmiPowerOutput parses the ipmitool host power usage and returns a map of corresponding float values.
func parseIpmiPowerOutput(result string) (map[string]float64, error) {
	powerMeasurements := make(map[string]float64)
	powerMeasurementExpr := make(map[string]string)

	powerMeasurementExpr[tsparams.IpmiDcmiPowerInstantaneous] =
		`Instantaneous power reading: (\s*[0-9]+) Watts`
	powerMeasurementExpr[tsparams.IpmiDcmiPowerMinimumDuringSampling] =
		`Minimum during sampling period: (\s*[0-9]+) Watts`
	powerMeasurementExpr[tsparams.IpmiDcmiPowerMaximumDuringSampling] =
		`Maximum during sampling period: (\s*[0-9]+) Watts`
	powerMeasurementExpr[tsparams.IpmiDcmiPowerAverageDuringSampling] =
		`Average power reading over sample period: (\s*[0-9]+) Watts`

	// Extract power measurements.
	for key, pattern := range powerMeasurementExpr {
		re := regexp.MustCompile(pattern)
		res := re.FindStringSubmatch(result)

		if len(res) > 0 {
			var value float64

			_, err := fmt.Sscan(res[1], &value)
			if err != nil {
				return nil, err
			}

			powerMeasurements[key] = value
		}
	}

	return powerMeasurements, nil
}

// computePowerUsageStatistics computes the power usage summary statistics.
func computePowerUsageStatistics(powerMeasurements map[time.Time]map[string]float64,
	samplingInterval time.Duration, scenario, tag string) (map[string]string, error) {
	/*
		Compute power measurement statistics

		Sample power measurement data:
		map[
		2023-03-08 10:17:46.629599 -0500 EST m=+132.341222733:map[avgPower:251 instantaneousPower:326 maxPower:503 minPower:8]
		2023-03-08 10:18:46.630737 -0500 EST m=+192.341245075:map[avgPower:251 instantaneousPower:324 maxPower:503 minPower:8]
		2023-03-08 10:19:46.563857 -0500 EST m=+252.341201729:map[avgPower:251 instantaneousPower:329 maxPower:503 minPower:8]
		2023-03-08 10:20:46.563313 -0500 EST m=+312.340308977:map[avgPower:251 instantaneousPower:332 maxPower:503 minPower:8]
		2023-03-08 10:21:46.564469 -0500 EST m=+372.341065314:map[avgPower:251 instantaneousPower:329 maxPower:503 minPower:8]
		]

		The following power measurement summary statistics are computed:

		numberSamples: count(powerMeasurements)
		samplingInterval: <samplingInterval>
		minInstantaneousPower: min(instantaneousPower)
		maxInstantaneousPower: max(instantaneousPower)
		meanInstantaneousPower: mean(instantaneousPower)
		stdDevInstantaneousPower: standard-deviation(instantaneousPower)
		medianInstantaneousPower: median(instantaneousPower)
	*/
	glog.V(tsparams.LogLevel).Infof("Power usage measurements for %s:\n%v\n", scenario, powerMeasurements)

	compMap := make(map[string]string)

	numberSamples := len(powerMeasurements)

	compMap[fmt.Sprintf("%s_%s_%s", tsparams.RanPowerMetricTotalSamples, scenario, tag)] =
		fmt.Sprintf("%d", numberSamples)
	compMap[fmt.Sprintf("%s_%s_%s", tsparams.RanPowerMetricSamplingIntervalSeconds, scenario, tag)] =
		fmt.Sprintf("%.0f", samplingInterval.Seconds())

	instantPowerData := make([]float64, numberSamples)
	index := 0

	for _, row := range powerMeasurements {
		instantPowerData[index] = row[tsparams.IpmiDcmiPowerInstantaneous]
		index++
	}

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
