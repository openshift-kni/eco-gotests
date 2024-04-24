package tests

import (
	"fmt"
	apiUrl "net/url"
	"os"
	"regexp"
	"time"

	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/imageregistryconfig"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/bmh"
	"github.com/openshift-kni/eco-goinfra/pkg/clusteroperator"
	"github.com/openshift-kni/eco-goinfra/pkg/clusterversion"
	"github.com/openshift-kni/eco-goinfra/pkg/mco"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	. "github.com/openshift-kni/eco-gotests/tests/system-tests/vcore/internal/vcoreinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/vcore/internal/vcoreparams"
	v1 "github.com/openshift/api/operator/v1"
)

var _ = Describe(
	"Initial Cluster Deployment Verification",
	Label(vcoreparams.Label), func() {
		It("Verify healthy cluster status", reportxml.ID("59441"),
			Label(vcoreparams.LabelVCoreDeployment), func() {
				kubeConfigURL := os.Getenv("KUBECONFIG")

				By("Checking if API URL available")
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
			})

		It("Asserts time sync was successfully applied for master nodes", reportxml.ID("60028"),
			Label(vcoreparams.LabelVCoreDeployment), func() {
				isChronyApplied := false

				chronyConfigNameRegex := "\\d+-master\\w*-*\\w*-chrony-conf\\w*"
				mcp := mco.NewMCPBuilder(APIClient, vcoreparams.MasterNodeRole)
				Expect(mcp.Exists()).To(BeTrue(), "Error find master mcp")

				for _, source := range mcp.Object.Status.Configuration.Source {
					reg, _ := regexp.Compile(chronyConfigNameRegex)

					if reg.MatchString(source.Name) {
						isChronyApplied = true

						break
					}
				}
				Expect(isChronyApplied).To(BeTrue(), "Error assert time sync was applied")
			})

		It("Asserts time sync was successfully applied for worker nodes", reportxml.ID("60029"),
			Label(vcoreparams.LabelVCoreDeployment), func() {
				isChronyApplied := false

				chronyConfigNameRegex := "\\d+-worker\\w*-*\\w*-chrony-conf\\w*"

				mcp := mco.NewMCPBuilder(APIClient, vcoreparams.WorkerNodeRole)
				Expect(mcp.Exists()).To(BeTrue(), "Error find worker mcp")

				for _, source := range mcp.Object.Status.Configuration.Source {
					reg, _ := regexp.Compile(chronyConfigNameRegex)

					if reg.MatchString(source.Name) {
						isChronyApplied = true

						break
					}
				}
				Expect(isChronyApplied).To(BeTrue(), "Error assert time sync was applied")
			})

		It("Asserts full set of ODF nodes was deployed", reportxml.ID("59442"),
			Label(vcoreparams.LabelVCoreDeployment), func() {

				mcp := mco.NewMCPBuilder(APIClient, vcoreparams.VCoreOdfMcpName)
				Expect(mcp.Exists()).To(BeTrue(), "Error to find ODF mcp")

				glog.V(100).Infof("Checking available ODF nodes count")
				nodesList, err := nodes.List(APIClient, VCoreConfig.OdfLabelListOption)
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to get ODF nodes list; %v", err))
				Expect(len(nodesList)).ToNot(Equal(0), "ODF nodes list is empty")
			})

		It("Asserts control-plane-worker mcp found", reportxml.ID("60049"),
			Label(vcoreparams.LabelVCoreDeployment), func() {

				mcp := mco.NewMCPBuilder(APIClient, vcoreparams.VCoreCpMcpName)
				Expect(mcp.Exists()).To(BeTrue(), "Error to find control-plane-worker mcp")

				glog.V(100).Infof("Checking control-plane-worker mcp condition state")
				Expect(mcp.IsInCondition("Updated")).To(BeTrue(), "control-plane-worker mcp failed to update")
			})

		It("Asserts full set of control-plane-worker nodes was deployed", reportxml.ID("59505"),
			Label(vcoreparams.LabelVCoreDeployment), func() {

				mcp := mco.NewMCPBuilder(APIClient, vcoreparams.VCoreCpMcpName)
				Expect(mcp.Exists()).To(BeTrue(), "Error to find control-plane-worker mcp")

				glog.V(100).Infof("Checking available control-plane-worker nodes count")
				nodesList, err := nodes.List(APIClient, VCoreConfig.VCoreCpLabelListOption)
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to get control-plane-worker nodes list; %v", err))
				Expect(len(nodesList)).ToNot(Equal(0), "control-plane-worker nodes list is empty")
			})

		It("Asserts user-plane-worker mcp found", reportxml.ID("60050"),
			Label(vcoreparams.LabelVCoreDeployment), func() {

				mcp := mco.NewMCPBuilder(APIClient, vcoreparams.VCorePpMcpName)
				Expect(mcp.Exists()).To(BeTrue(), "Error to find user-plane-worker mcp")

				glog.V(100).Infof("Checking user-plane-worker mcp condition state")
				Expect(mcp.IsInCondition("Updated")).To(BeTrue(), "user-plane-worker mcp failed to update")
			})

		It("Asserts full set of user-plane-worker nodes was deployed", reportxml.ID("59506"),
			Label(vcoreparams.LabelVCoreDeployment), func() {

				mcp := mco.NewMCPBuilder(APIClient, vcoreparams.VCorePpMcpName)
				Expect(mcp.Exists()).To(BeTrue(), "Error to find user-plane-worker mcp")

				glog.V(100).Infof("Checking available user-plane-worker nodes count")
				nodesList, err := nodes.List(APIClient, VCoreConfig.VCorePpLabelListOption)
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to get user-plane-worker nodes list; %v", err))
				Expect(len(nodesList)).ToNot(Equal(0), "user-plane-worker nodes list is empty")
			})

		It("Asserts Image Registry management state is Enabled", reportxml.ID("72812"),
			Label(vcoreparams.LabelVCoreDebug), func() {

				glog.V(100).Infof("Enable local imageregistryconfig; change ManagementState to the Managed")

				err := imageregistryconfig.SetManagementState(APIClient, v1.Managed)
				Expect(err).ToNot(HaveOccurred(),
					fmt.Sprintf("Failed to change imageRegistry state to the Managed; %v", err))

				glog.V(100).Infof("Setup imageRegistry storage")
				err = imageregistryconfig.SetStorageToTheEmptyDir(APIClient)
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to setup imageRegistry storage; %v", err))
			})
	})
