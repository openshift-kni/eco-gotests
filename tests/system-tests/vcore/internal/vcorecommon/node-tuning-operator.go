package vcorecommon

import (
	"fmt"
	"strings"
	"time"

	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"

	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/openshift-kni/eco-goinfra/pkg/clusteroperator"
	"github.com/openshift-kni/eco-goinfra/pkg/mco"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/nto" //nolint:misspell
	"github.com/openshift-kni/eco-goinfra/pkg/service"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/apiobjectshelper"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/ocpcli"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/vcore/internal/vcoreparams"
	tunedv1 "github.com/openshift/cluster-node-tuning-operator/pkg/apis/tuned/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	kubeletconfigv1beta1 "k8s.io/kubelet/config/v1beta1"

	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	v2 "github.com/openshift/cluster-node-tuning-operator/pkg/apis/performanceprofile/v2"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/openshift-kni/eco-gotests/tests/system-tests/vcore/internal/vcoreinittools"
)

// VerifyNTOSuite container that contains tests for Node Tuning Operator verification.
func VerifyNTOSuite() {
	Describe(
		"NTO validation", //nolint:misspell
		Label(vcoreparams.LabelVCoreOperators), func() {
			It(fmt.Sprintf("Verifies %s namespace exists", vcoreparams.NTONamespace),
				Label("nto"), VerifyNTONamespaceExists) //nolint:misspell

			It("Verify Node Tuning Operator successfully installed",
				Label("nto"), reportxml.ID("63656"), VerifyNTODeployment) //nolint:misspell

			It("Create new performanceprofile",
				Label("nto"), reportxml.ID("63741"), CreatePerformanceProfile) //nolint:misspell

			It("Create new nodes tuning",
				Label("nto"), reportxml.ID("63740"), CreateNodesTuning) //nolint:misspell

			It("Verify CPU Manager config",
				Label("nto"), reportxml.ID("63809"), VerifyCPUManagerConfig) //nolint:misspell

			It("Verify Node Tuning Operator Huge Pages configuration",
				Label("nto"), reportxml.ID("60062"), VerifyHugePagesConfig) //nolint:misspell

			It("Verify System Reserved memory for user-plane-worker nodes configuration",
				Label("nto"), //nolint:misspell
				reportxml.ID("60047"), SetSystemReservedMemoryForUserPlaneNodes)
		})
}

// VerifyNTONamespaceExists asserts namespace for Node Tuning Operator exists.
func VerifyNTONamespaceExists(ctx SpecContext) {
	err := apiobjectshelper.VerifyNamespaceExists(APIClient, vcoreparams.NTONamespace, time.Second)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to pull %q namespace", vcoreparams.NTONamespace))
} // func VerifyNTONamespaceExists (ctx SpecContext)

// VerifyNTODeployment asserts Node Tuning Operator successfully installed.
func VerifyNTODeployment(ctx SpecContext) {
	ntoServiceName := "node-tuning-operator"
	paoServiceName := "performance-addon-operator-service"

	err := apiobjectshelper.VerifyOperatorDeployment(APIClient,
		"",
		vcoreparams.NTODeploymentName,
		vcoreparams.NTONamespace,
		time.Minute)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("operator deployment %s failure in the namespace %s; %v",
			vcoreparams.NTODeploymentName, vcoreparams.NTONamespace, err))

	glog.V(vcoreparams.VCoreLogLevel).Infof(
		"Confirm that NTO %s pod was deployed and running in %s namespace", //nolint:misspell
		vcoreparams.NTODeploymentName, vcoreparams.NTONamespace)

	ntoPods, err := pod.ListByNamePattern(APIClient, vcoreparams.NTODeploymentName, vcoreparams.NTONamespace)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("No %s pods were found in %s namespace due to %s",
		vcoreparams.NTODeploymentName, vcoreparams.NTONamespace, err))
	Expect(len(ntoPods)).ToNot(Equal(0), fmt.Sprintf("The list of pods %s found in namespace %s is empty",
		vcoreparams.NTODeploymentName, vcoreparams.NTONamespace))

	ntoPod := ntoPods[0]
	ntoPodName := ntoPod.Object.Name

	err = ntoPod.WaitUntilReady(time.Second)
	if err != nil {
		ntoPodLog, _ := ntoPod.GetLog(3*time.Second, vcoreparams.NTODeploymentName)
		glog.Fatalf("%s pod in %s namespace in a bad state: %s",
			ntoPodName, vcoreparams.NTONamespace, ntoPodLog)
	}

	glog.V(vcoreparams.VCoreLogLevel).Infof("Verify NTO service %s created in the namespace %s", //nolint:misspell
		ntoServiceName, vcoreparams.NTONamespace)

	ntoService, err := service.Pull(APIClient, ntoServiceName, vcoreparams.NTONamespace)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to get service %s from the namespace %s due to %s",
		ntoServiceName, vcoreparams.NTONamespace, err))
	Expect(ntoService.Exists()).To(Equal(true), fmt.Sprintf("no service %s was found in the namespace %s",
		ntoServiceName, vcoreparams.NTONamespace))

	glog.V(vcoreparams.VCoreLogLevel).Infof("Verify NTO service %s created in the namespace %s", //nolint:misspell
		ntoServiceName, vcoreparams.NTONamespace)

	paoService, err := service.Pull(APIClient, paoServiceName, vcoreparams.NTONamespace)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to get service %s from the namespace %s due to %s",
		paoServiceName, vcoreparams.NTONamespace, err))
	Expect(paoService.Exists()).To(Equal(true), fmt.Sprintf("no service %s was found in the namespace %s",
		paoServiceName, vcoreparams.NTONamespace))
} // func VerifyNTODeployment (ctx SpecContext)

// CreatePerformanceProfile asserts performanceprofile can be created and successfully applied.
func CreatePerformanceProfile(ctx SpecContext) {
	glog.V(vcoreparams.VCoreLogLevel).Infof("Verify performanceprofile %s exists", VCoreConfig.VCorePpMCPName)

	hugePagesNodeOne := int32(0)
	hugePagesNodeTwo := int32(1)
	hugePages := []v2.HugePage{
		{
			Size:  vcoreparams.HugePagesSize,
			Count: 32768,
			Node:  &hugePagesNodeOne,
		},
		{
			Size:  vcoreparams.HugePagesSize,
			Count: 32768,
			Node:  &hugePagesNodeTwo,
		},
	}
	performanceKubeletConfigName := fmt.Sprintf("performance-%s", VCoreConfig.VCorePpMCPName)
	ppAnnotations := map[string]string{"performance.openshift.io/ignore-cgroups-version": "true",
		"kubeletconfig.experimental": fmt.Sprintf("{\"systemReserved\":{\"cpu\":\"%s\",\"memory\":\"%s\"}}",
			vcoreparams.SystemReservedCPU, vcoreparams.SystemReservedMemory)}
	netInterfaceName := "ens2f(0|1)"
	netDevices := []v2.Device{{
		InterfaceName: &netInterfaceName,
	}}

	var err error

	ppObj := nto.NewBuilder(APIClient,
		VCoreConfig.VCorePpMCPName,
		VCoreConfig.CPUIsolated,
		VCoreConfig.CPUReserved,
		VCoreConfig.VCorePpLabelMap).
		WithAnnotations(ppAnnotations).
		WithNet(true, netDevices).
		WithGloballyDisableIrqLoadBalancing().
		WithHugePages(vcoreparams.HugePagesSize, hugePages).
		WithNumaTopology(vcoreparams.TopologyConfig).
		WithWorkloadHints(false, true, false)

	if !ppObj.Exists() {
		glog.V(vcoreparams.VCoreLogLevel).Infof("Create new performanceprofile %s", VCoreConfig.VCorePpMCPName)

		ppObj, err = ppObj.Create()
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to create performanceprofile %s; %v",
			VCoreConfig.VCorePpMCPName, err))
		Expect(ppObj.Exists()).To(Equal(true),
			fmt.Sprintf("performanceprofile %s not found", VCoreConfig.VCorePpMCPName))

		glog.V(vcoreparams.VCoreLogLevel).Infof("Wait for all nodes rebooting after applying performanceprofile %s",
			VCoreConfig.VCorePpMCPName)

		_, err = nodes.WaitForAllNodesToReboot(
			APIClient,
			40*time.Minute,
			VCoreConfig.VCorePpLabelListOption)
		Expect(err).ToNot(HaveOccurred(),
			fmt.Sprintf("Nodes failed to reboot after applying performanceprofile %s config; %v",
				VCoreConfig.VCorePpMCPName, err))

		glog.V(vcoreparams.VCoreLogLevel).Info("Wait for all clusteroperators availability after nodes reboot")

		_, err = clusteroperator.WaitForAllClusteroperatorsAvailable(APIClient, 60*time.Second)
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Error waiting for all available clusteroperators: %v", err))
	}

	glog.V(vcoreparams.VCoreLogLevel).Info("Verify NUMA Topology Manager")

	performanceKubeletConfigObj, err := mco.PullKubeletConfig(APIClient, performanceKubeletConfigName)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to get kubeletconfigs %s due to %v",
		performanceKubeletConfigName, err))

	kubeletConfig := performanceKubeletConfigObj.Object.Spec.KubeletConfig.Raw

	currentTopologyManagerPolicy :=
		unmarshalRaw[kubeletconfigv1beta1.KubeletConfiguration](kubeletConfig).TopologyManagerPolicy
	Expect(currentTopologyManagerPolicy).To(Equal(vcoreparams.TopologyConfig),
		fmt.Sprintf("incorrect topology manager policy found; expected: %s, found: %s",
			vcoreparams.TopologyConfig, currentTopologyManagerPolicy))
} // func CreatePerformanceProfile (ctx SpecContext)

// CreateNodesTuning creates new Node Tuning configuration.
//
//nolint:funlen
func CreateNodesTuning(ctx SpecContext) {
	tunedInstanceName := fmt.Sprintf("configuration-nic-%s", VCoreConfig.VCorePpMCPName)
	sysctlConf := map[string]string{
		"net.core.netdev_max_backlog": "5000",
		"net.core.optmem_max":         "81920",
		"net.core.rmem_default":       "33554432",
		"net.core.rmem_max":           "33554432",
		"net.core.wmem_default":       "33554432",
		"net.core.wmem_max":           "33554432",
		"net.ipv4.udp_rmem_min":       "8192",
	}
	tunedProfileData := fmt.Sprintf(""+
		"[main]"+
		"summary=Configuration changes profile inherited from performance created tuned"+
		"include=openshift-node-performance-%s"+
		""+
		"[sysctl]"+
		"net.core.netdev_max_backlog = %s"+
		"net.core.optmem_max = %s"+
		"net.core.rmem_default = %s"+
		"net.core.rmem_max = %s"+
		"net.core.wmem_default = %s"+
		"net.core.wmem_max = %s"+
		"net.ipv4.udp_rmem_min = %s"+
		""+
		"[net]"+
		"type=net"+
		"devices_udev_regex=^INTERFACE=ens2f(0|1)"+
		"channels=combined 32",
		VCoreConfig.VCorePpMCPName,
		sysctlConf["net.core.netdev_max_backlog"],
		sysctlConf["net.core.optmem_max"],
		sysctlConf["net.core.rmem_default"],
		sysctlConf["net.core.rmem_max"],
		sysctlConf["net.core.wmem_default"],
		sysctlConf["net.core.wmem_max"],
		sysctlConf["net.ipv4.udp_rmem_min"])
	tunedProfile := tunedv1.TunedProfile{
		Name: &tunedInstanceName,
		Data: &tunedProfileData,
	}
	recommendPriority := uint64(19)
	recommendLabel := VCoreConfig.VCorePpLabel
	tunedRecommend := tunedv1.TunedRecommend{
		Profile:  &tunedInstanceName,
		Priority: &recommendPriority,
		Match: []tunedv1.TunedMatch{{
			Label: &recommendLabel,
		}},
	}

	glog.V(vcoreparams.VCoreLogLevel).Infof("Verify nodes tuning %s already exists in namespace %s",
		VCoreConfig.VCorePpMCPName, vcoreparams.NTONamespace)

	var err error

	ntoObj := nto.NewTunedBuilder(APIClient,
		tunedInstanceName,
		vcoreparams.NTONamespace).
		WithProfile(tunedProfile).
		WithRecommend(tunedRecommend)

	if !ntoObj.Exists() {
		glog.V(vcoreparams.VCoreLogLevel).Infof("Create new node tuning profile %s in namespace %s",
			tunedInstanceName, vcoreparams.NTONamespace)

		ntoObj, err = ntoObj.Create()
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to create tuned %s in namespace %s; %v",
			tunedInstanceName, vcoreparams.NTONamespace, err))
		Expect(ntoObj.Exists()).To(Equal(true),
			fmt.Sprintf("tuned %s not found in namespace %s", tunedInstanceName, vcoreparams.NTONamespace))

		glog.V(vcoreparams.VCoreLogLevel).Info("The short sleep to update new values before the following change")

		time.Sleep(10 * time.Second)
	}

	nodesList, err := nodes.List(APIClient, VCoreConfig.VCorePpLabelListOption)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to get nodes list with the label %v; %v",
		VCoreConfig.VCorePpLabelListOption, err))

	for _, node := range nodesList {
		glog.V(vcoreparams.VCoreLogLevel).Infof("Check nohz_full is removed from the node %s",
			node.Definition.Name)

		nohzFullCmd := "cat /proc/cmdline"
		output, err := ocpcli.ExecuteViaDebugPodOnNode(node.Object.Name, nohzFullCmd)
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to execute %s cmd on the node %s due to %v",
			nohzFullCmd, node.Object.Name, err))
		Expect(output).NotTo(ContainSubstring("nohz_full"),
			fmt.Sprintf("failed to remove nohz_full on the node %s; %v", node.Definition.Name, output))

		for param, value := range sysctlConf {
			glog.V(vcoreparams.VCoreLogLevel).Infof("Check net.core.netdev_max_backlog value set on the node %s",
				node.Definition.Name)

			netCmd := fmt.Sprintf("sudo sysctl --all | grep %s", param)
			output, err = ocpcli.ExecuteViaDebugPodOnNode(node.Object.Name, netCmd)
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to execute %s cmd on the node %s due to %v",
				netCmd, node.Object.Name, err))

			Expect(strings.TrimSuffix(output[strings.LastIndex(output, "=")+1:], "\n")).To(Equal(value),
				fmt.Sprintf("failed to change %s value to %s on the node %s; %v",
					param, value, node.Definition.Name, output))
		}
	}
} // func CreateNodesTuning (ctx SpecContext)

// VerifyCPUManagerConfig verifies CPU Manager configuration.
func VerifyCPUManagerConfig(ctx SpecContext) {
	nodesList, err := nodes.List(APIClient, VCoreConfig.VCorePpLabelListOption)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to get nodes list with the label %v; %v",
		VCoreConfig.VCorePpLabelListOption, err))

	glog.V(vcoreparams.VCoreLogLevel).Info("Verify CPU Manager configuration")

	cpuManagerCmd := "sudo grep cpuManager /etc/kubernetes/kubelet.conf"

	for _, node := range nodesList {
		glog.V(vcoreparams.VCoreLogLevel).Infof("Check CPU Manager activated on the node %s",
			node.Definition.Name)

		output, err := ocpcli.ExecuteViaDebugPodOnNode(node.Object.Name, cpuManagerCmd)
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to execute %s cmd on the node %s due to %v",
			cpuManagerCmd, node.Object.Name, err))
		Expect(output).To(ContainSubstring("cpuManagerPolicy"),
			fmt.Sprintf("failed to activate CPU Manager on the node %s; %v", node.Definition.Name, output))
	}
} // func VerifyCPUManagerConfig (ctx SpecContext)

// VerifyHugePagesConfig verifies correctness of the Huge Pages configuration.
func VerifyHugePagesConfig(ctx SpecContext) {
	nodesList, err := nodes.List(APIClient, VCoreConfig.VCorePpLabelListOption)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to get nodes list with the label %v; %v",
		VCoreConfig.VCorePpLabelListOption, err))

	glog.V(vcoreparams.VCoreLogLevel).Info("Verify Node Tuning instance hugepages configuration")

	for _, node := range nodesList {
		glog.V(vcoreparams.VCoreLogLevel).Infof("Check hugepages config for the node %s",
			node.Definition.Name)

		allocatableHPResources := node.Object.Status.Allocatable
		capacityHPResources := node.Object.Status.Capacity

		for hpResource, value := range allocatableHPResources {
			if hpResource == "hugepages-2Mi" {
				Expect(value).To(Equal(resource.MustParse(vcoreparams.ExpectedHugePagesResource)),
					fmt.Sprintf("allocatable resource config on node %s not as expected; expected: %s, current: %v",
						node.Definition.Name, vcoreparams.ExpectedHugePagesResource, value))
			}
		}

		for hpResource, value := range capacityHPResources {
			if hpResource == "hugepages-2Mi" {
				Expect(value).To(Equal(resource.MustParse(vcoreparams.ExpectedHugePagesResource)),
					fmt.Sprintf("capacity resource config on node %s not as expected; expected: %s, current: %v",
						node.Definition.Name, vcoreparams.ExpectedHugePagesResource, value))
			}
		}
	}
} // func VerifyHugePagesConfig (ctx SpecContext)

// SetSystemReservedMemoryForUserPlaneNodes assert system reserved memory for user-plane-worker nodes succeeded.
func SetSystemReservedMemoryForUserPlaneNodes(ctx SpecContext) {
	glog.V(vcoreparams.VCoreLogLevel).Infof("Verify system reserved memory config for masters succeeded")

	kubeletConfigName := fmt.Sprintf("performance-%s", VCoreConfig.VCorePpMCPName)

	_, err := mco.PullKubeletConfig(APIClient, kubeletConfigName)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to get kubeletconfigs %s due to %v",
		kubeletConfigName, err))

	glog.V(vcoreparams.VCoreLogLevel).Infof("Verify system reserved data updated for all %s nodes",
		VCoreConfig.ControlPlaneLabel)

	nodesList, err := nodes.List(APIClient, VCoreConfig.VCorePpLabelListOption)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to get %v nodes list; %v", VCoreConfig.VCorePpLabelListOption, err))

	systemReservedDataCmd := "cat /etc/node-sizing.env"
	for _, node := range nodesList {
		output, err := ocpcli.ExecuteViaDebugPodOnNode(node.Object.Name, systemReservedDataCmd)
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to execute %v cmd on the %s node due to %v",
			systemReservedDataCmd, VCoreConfig.ControlPlaneLabel, err))
		Expect(output).To(ContainSubstring(fmt.Sprintf("SYSTEM_RESERVED_CPU=%s", vcoreparams.SystemReservedCPU)),
			fmt.Sprintf("reserved CPU configuration did not changed for the node %s; expected value: %s, "+
				"found: %v", node.Definition.Name, vcoreparams.SystemReservedCPU, output))
		Expect(output).To(ContainSubstring(fmt.Sprintf("SYSTEM_RESERVED_MEMORY=%s", vcoreparams.SystemReservedMemory)),
			fmt.Sprintf("reserved memory configuration did not changed for the node %s; expected value: %s, "+
				"found: %v", node.Definition.Name, vcoreparams.SystemReservedMemory, output))
	}
} // func SetSystemReservedMemoryForUserPlaneNodes (ctx SpecContext)

// unmarshalRaw converts raw bytes for a K8s CR into the actual type.
func unmarshalRaw[T any](raw []byte) T {
	untyped := &unstructured.Unstructured{}
	err := untyped.UnmarshalJSON(raw)
	Expect(err).ToNot(HaveOccurred(), "Failed to unmarshal JSON into unstructured")

	var typed T
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(untyped.UnstructuredContent(), &typed)
	Expect(err).ToNot(HaveOccurred(), "Failed to convert unstructed to structured")

	return typed
}
