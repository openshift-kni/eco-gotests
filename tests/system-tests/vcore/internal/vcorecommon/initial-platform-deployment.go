package vcorecommon

import (
	"fmt"
	apiUrl "net/url"
	"os"
	"regexp"
	"time"

	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/ocpcli"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/cgroup"
	configv1 "github.com/openshift/api/config/v1"

	"github.com/openshift-kni/eco-goinfra/pkg/bmh"
	"github.com/openshift-kni/eco-goinfra/pkg/clusteroperator"
	"github.com/openshift-kni/eco-goinfra/pkg/clusterversion"
	"github.com/openshift-kni/eco-goinfra/pkg/mco"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/nodesconfig"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	. "github.com/openshift-kni/eco-gotests/tests/system-tests/vcore/internal/vcoreinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/vcore/internal/vcoreparams"
)

// VerifyHealthyClusterStatus asserts healthy cluster status.
func VerifyHealthyClusterStatus(ctx SpecContext) {
	glog.V(vcoreparams.VCoreLogLevel).Infof("Verify healthy cluster status")

	kubeConfigURL := os.Getenv("KUBECONFIG")

	glog.V(vcoreparams.VCoreLogLevel).Infof("Checking if API URL available")

	_, err := apiUrl.Parse(kubeConfigURL)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Error getting API URL: %v", err))

	glog.V(100).Infof("Checking if all BareMetalHosts in good OperationalState")

	var bmhList []*bmh.BmhBuilder
	bmhList, err = bmh.List(APIClient, vcoreparams.OpenshiftMachineAPINamespace)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Error getting BareMetaHosts list: %v", err))
	Expect(len(bmhList)).ToNot(Equal(0), "Empty bareMetalHosts list received")

	_, err = bmh.WaitForAllBareMetalHostsInGoodOperationalState(APIClient,
		vcoreparams.OpenshiftMachineAPINamespace,
		5*time.Second)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Error waiting for all BareMetalHosts in good OperationalState: %v", err))

	glog.V(100).Infof("Checking available control-plane nodes count")

	var nodesList []*nodes.Builder
	nodesList, err = nodes.List(APIClient, VCoreConfig.ControlPlaneLabelListOption)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to get master nodes list; %v", err))

	masterNodesCount := len(nodesList)
	Expect(masterNodesCount).To(Equal(3),
		fmt.Sprintf("Error in master nodes count; found master nodes count is %d", masterNodesCount))

	glog.V(100).Infof("Checking all master nodes are Ready")

	var isReady bool
	isReady, err = nodes.WaitForAllNodesAreReady(
		APIClient,
		5*time.Second,
		VCoreConfig.ControlPlaneLabelListOption)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Error getting master nodes list: %v", err))
	Expect(isReady).To(Equal(true),
		fmt.Sprintf("Failed master nodes status, not all Master node are Ready; %v", isReady))

	glog.V(100).Infof("Checking that the clusterversion is available")

	_, err = clusterversion.Pull(APIClient)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Error accessing csv: %v", err))

	glog.V(100).Infof("Asserting clusteroperators availability")

	var coBuilder []*clusteroperator.Builder
	coBuilder, err = clusteroperator.List(APIClient)
	Expect(err).To(BeNil(), fmt.Sprintf("ClusterOperator List not found: %v", err))
	Expect(len(coBuilder)).ToNot(Equal(0), "Empty clusterOperators list received")

	_, err = clusteroperator.WaitForAllClusteroperatorsAvailable(APIClient, 60*time.Second)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Error waiting for all available clusteroperators: %v", err))
} // func VerifyHealthyClusterStatus (ctx SpecContext)

// verifyEtcChronyRun assert that time sync config was successfully applied and running.
func verifyEtcChronyRun(nodesRole string, nodeLabelOption metav1.ListOptions) {
	glog.V(vcoreparams.VCoreLogLevel).Infof(fmt.Sprintf("Asserts that time sync config was successfully applied "+
		"and running on %s nodes", nodesRole))

	isChronyApplied := false

	chronyConfigNameRegex := fmt.Sprintf("\\d+-%s\\w*-*\\w*-chrony-conf\\w*", nodesRole)
	mcp, err := mco.Pull(APIClient, nodesRole)
	Expect(err).To(BeNil(), fmt.Sprintf("%q MCP was not found", nodesRole))

	for _, source := range mcp.Object.Status.Configuration.Source {
		reg, _ := regexp.Compile(chronyConfigNameRegex)

		if reg.MatchString(source.Name) {
			isChronyApplied = true

			break
		}
	}

	Expect(isChronyApplied).To(BeTrue(), fmt.Sprintf("Error assert time sync was applied for %s nodes", nodesRole))

	glog.V(vcoreparams.VCoreLogLevel).Infof(fmt.Sprintf("Verify the chronyd status for %s nodes", nodesRole))

	chronydStatusCmd := "sudo systemctl status chronyd | grep Active"

	nodesList, err := nodes.List(APIClient, nodeLabelOption)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to get %s nodes list; %v", nodesRole, err))

	for _, node := range nodesList {
		output, err := ocpcli.ExecuteViaDebugPodOnNode(node.Object.Name, chronydStatusCmd)
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to execute %v cmd on the %s node due to %v",
			chronydStatusCmd, nodesRole, err))
		Expect(output).To(ContainSubstring("running"),
			fmt.Sprintf("chronyd failed to start on the node %s, current status: %v",
				node.Definition.Name, output))
	}
}

// VerifyEtcChronyMasters assert that time sync config was successfully applied for master nodes.
func VerifyEtcChronyMasters(ctx SpecContext) {
	verifyEtcChronyRun(vcoreparams.MasterNodeRole, VCoreConfig.ControlPlaneLabelListOption)
} // func VerifyEtcChronyMasters (ctx SpecContext)

// VerifyEtcChronyWorkers assert that time sync config was successfully applied for workers nodes.
func VerifyEtcChronyWorkers(ctx SpecContext) {
	verifyEtcChronyRun(vcoreparams.WorkerNodeRole, metav1.ListOptions{LabelSelector: vcoreparams.WorkerNodeRole})
} // func VerifyEtcChronyWorkers (ctx SpecContext)

// verifyNodesAvailability assert MCP nodes availability.
func verifyNodesAvailability(nodesRole string, nodeLabelOption metav1.ListOptions) {
	glog.V(vcoreparams.VCoreLogLevel).Infof("Verify %s nodes availability", nodesRole)

	nodesList, err := nodes.List(APIClient, nodeLabelOption)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to get %s nodes list; %v", nodesRole, err))
	Expect(len(nodesList)).ToNot(Equal(0), fmt.Sprintf("%s nodes list is empty", nodesRole))
}

// verifyNodesInMCP assert MCP was deployed.
func verifyNodesInMCP(nodesRole string) {
	glog.V(vcoreparams.VCoreLogLevel).Infof("Verify %s MCP was deployed", nodesRole)

	mcp, err := mco.Pull(APIClient, nodesRole)
	Expect(err).To(BeNil(), fmt.Sprintf("%q MCP was not found", nodesRole))

	glog.V(100).Infof("Checking %s MCP condition state", nodesRole)
	Expect(mcp.IsInCondition("Updated")).To(BeTrue(),
		fmt.Sprintf("%s MCP failed to update", nodesRole))
}

// VerifyODFMCPAvailability assert odf MCP was deployed.
func VerifyODFMCPAvailability(ctx SpecContext) {
	verifyNodesInMCP(vcoreparams.VCoreOdfMcpName)
} // func VerifyODFMCPAvailability (ctx SpecContext)

// VerifyODFNodesAvailability assert full set of ODF nodes was deployed.
func VerifyODFNodesAvailability(ctx SpecContext) {
	verifyNodesAvailability(vcoreparams.VCoreOdfMcpName, VCoreConfig.OdfLabelListOption)
} // func VerifyODFNodesAvailability (ctx SpecContext)

// VerifyControlPlaneWorkerMCPAvailability assert control-plane-worker MCP was deployed.
func VerifyControlPlaneWorkerMCPAvailability(ctx SpecContext) {
	verifyNodesInMCP(vcoreparams.VCoreCpMcpName)
} // func VerifyControlPlaneWorkerMCPAvailability (ctx SpecContext)

// VerifyControlPlaneWorkerNodesAvailability assert control-plane-worker nodes availability.
func VerifyControlPlaneWorkerNodesAvailability(ctx SpecContext) {
	verifyNodesAvailability(vcoreparams.VCoreCpMcpName, VCoreConfig.VCoreCpLabelListOption)
} // func VerifyControlPlaneWorkerNodesAvailability (ctx SpecContext)

// VerifyUserPlaneWorkerMCPAvailability assert user-plane-worker MCP was deployed.
func VerifyUserPlaneWorkerMCPAvailability(ctx SpecContext) {
	verifyNodesInMCP(vcoreparams.VCorePpMcpName)
} // func VerifyUserPlaneWorkerMCPAvailability (ctx SpecContext)

// VerifyUserPlaneWorkerNodesAvailability assert user-plane-worker nodes availability.
func VerifyUserPlaneWorkerNodesAvailability(ctx SpecContext) {
	verifyNodesAvailability(vcoreparams.VCorePpMcpName, VCoreConfig.VCorePpLabelListOption)
} // func VerifyUserPlaneWorkerNodesAvailability (ctx SpecContext)

// VerifyCGroupV2IsADefault assert cGroupV2 is a default for the cluster deployment.
func VerifyCGroupV2IsADefault(ctx SpecContext) {
	glog.V(vcoreparams.VCoreLogLevel).Infof("Verify cgroupv2 is a default for the cluster deployment")

	glog.V(vcoreparams.VCoreLogLevel).Infof("Get current cgroup mode configured for the cluster")

	nodesConfigObj, err := nodesconfig.Pull(APIClient, "cluster")
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to get nodes.config 'cluster' object due to %v", err))

	cgroupMode, err := nodesConfigObj.GetCGroupMode()
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to get cluster cgroup mode due to %v", err))
	Expect(cgroupMode).ToNot(Equal(configv1.CgroupModeV1), "wrong cgroup mode default found, v1 instead of v2")

	glog.V(vcoreparams.VCoreLogLevel).Infof("Verify actual cgroup version configured for the nodes")

	nodesList, err := nodes.List(APIClient)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to get cluster nodes list due to %v", err))

	for _, node := range nodesList {
		glog.V(vcoreparams.VCoreLogLevel).Infof("Verify actual cgroup version configured for node %s",
			node.Definition.Name)

		currentCgroupMode, err := cgroup.GetNodeLinuxCGroupVersion(APIClient, node.Definition.Name)
		Expect(err).ToNot(HaveOccurred(),
			fmt.Sprintf("failed to get cgroup mode value for the node %s due to %v",
				node.Definition.Name, err))
		Expect(currentCgroupMode).To(Equal(configv1.CgroupModeV2), fmt.Sprintf("wrong cgroup mode default "+
			"found configured for the node %s; expected cgroupMode is %v, found cgroupMode is %v",
			node.Definition.Name, configv1.CgroupModeV2, currentCgroupMode))
	}
} // func VerifyCGroupV2IsADefault (ctx SpecContext)

// VerifySwitchBetweenCGroupVersions assert that the cluster can be moved to the cgroupv1 and back.
func VerifySwitchBetweenCGroupVersions(ctx SpecContext) {
	glog.V(vcoreparams.VCoreLogLevel).Infof("Verify that the cluster can be moved to the cgroupv1 and back")

	err := cgroup.SetLinuxCGroupVersion(APIClient, configv1.CgroupModeV1)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to change cluster cgroup mode to the %v due to %v",
		configv1.CgroupModeV1, err))

	glog.V(vcoreparams.VCoreLogLevel).Infof("The short sleep to update new values before the following change.")

	time.Sleep(2 * time.Minute)

	err = cgroup.SetLinuxCGroupVersion(APIClient, configv1.CgroupModeV2)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to change cluster cgroup mode to the %v due to %v",
		configv1.CgroupModeV2, err))
} // func VerifySwitchBetweenCGroupVersions (ctx SpecContext)

// VerifyInitialDeploymentConfig container that contains tests for initial cluster deployment verification.
func VerifyInitialDeploymentConfig() {
	Describe(
		"Initial deployment config validation",
		Label(vcoreparams.LabelVCoreDeployment), func() {
			BeforeAll(func() {
				By("Insure cgroupv2 configured for the cluster")

				err := cgroup.SetLinuxCGroupVersion(APIClient, configv1.CgroupModeV2)
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to change cluster cgroup mode to the %v due to %v",
					configv1.CgroupModeV2, err))
			})

			It("Verifies healthy cluster status",
				Label("healthy-cluster"), reportxml.ID("59441"), VerifyHealthyClusterStatus)

			It("Asserts time sync was successfully applied for master nodes",
				Label("chrony"), reportxml.ID("60028"), VerifyEtcChronyMasters)

			It("Asserts time sync was successfully applied for workers nodes",
				Label("chrony"), reportxml.ID("60029"), VerifyEtcChronyWorkers)

			It("Verifies odf MCP was deployed",
				Label("odf"), reportxml.ID("73673"), VerifyODFMCPAvailability)

			It("Verifies full set of ODF nodes was deployed",
				Label("odf"), reportxml.ID("59442"), VerifyODFNodesAvailability)

			It("Verifies control-plane-worker MCP was deployed",
				Label("cp-mcp"), reportxml.ID("60049"), VerifyControlPlaneWorkerMCPAvailability)

			It("Verifies control-plane-worker nodes availability",
				Label("cp-nodes"), reportxml.ID("59505"), VerifyControlPlaneWorkerNodesAvailability)

			It("Verifies user-plane-worker MCP was deployed",
				Label("pp-mcp"), reportxml.ID("60050"), VerifyUserPlaneWorkerMCPAvailability)

			It("Verifies user-plane-worker nodes availability",
				Label("pp-nodes"), reportxml.ID("59506"), VerifyUserPlaneWorkerNodesAvailability)

			It("Verifies cgroupv2 is a default for the cluster deployment",
				Label("cgroupv2"), reportxml.ID("73370"), VerifyCGroupV2IsADefault)

			It("Verifies that the cluster can be moved to the cgroupv1 and back",
				Label("cgroupv2"), reportxml.ID("73371"), VerifySwitchBetweenCGroupVersions)

			AfterAll(func() {
				By("Restore cgroupv2 cluster configuration")

				err := cgroup.SetLinuxCGroupVersion(APIClient, configv1.CgroupModeV2)
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to change cluster cgroup mode to the %v due to %v",
					configv1.CgroupModeV2, err))
			})
		})
}
