package helper

import (
	"errors"

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
