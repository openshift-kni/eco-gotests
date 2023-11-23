package tests

import (
	"fmt"
	apiUrl "net/url"
	"os"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/bmh"
	"github.com/openshift-kni/eco-goinfra/pkg/clusteroperator"
	"github.com/openshift-kni/eco-goinfra/pkg/clusterversion"
	"github.com/openshift-kni/eco-goinfra/pkg/mco"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
	. "github.com/openshift-kni/eco-gotests/tests/system-tests/samsung-vcore/internal/samsunginittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/samsung-vcore/internal/samsungparams"
)

var _ = Describe(
	"Initial Cluster Deployment Verification",
	Label(samsungparams.Label), func() {
		It("Verify healthy cluster status", polarion.ID("59441"),
			Label(samsungparams.LabelSamsungVCoreDeployment), func() {
				kubeConfigURL := os.Getenv("KUBECONFIG")

				By("Checking if API URL available")
				_, err := apiUrl.Parse(kubeConfigURL)
				Expect(err).ToNot(HaveOccurred(), "Error getting API URL: %s", err)

				glog.V(100).Infof("Checking if all BareMetalHosts in good OperationalState")
				var bmhList []*bmh.BmhBuilder
				bmhList, err = bmh.List(APIClient, samsungparams.OpenshiftMachineAPINamespace)
				Expect(err).ToNot(HaveOccurred(), "Error getting BareMetaHosts list: %s", err)
				Expect(len(bmhList)).ToNot(Equal(0), "Empty bareMetalHosts list received")

				_, err = bmh.WaitForAllBareMetalHostsInGoodOperationalState(APIClient,
					samsungparams.OpenshiftMachineAPINamespace,
					5*time.Second)
				Expect(err).ToNot(HaveOccurred(), "Error waiting for all BareMetalHosts in good OperationalState: %s", err)

				glog.V(100).Infof("Checking available control-plane nodes count")
				var nodesList []*nodes.Builder
				nodesList, err = nodes.List(APIClient, SamsungConfig.ControlPlaneLabelListOption)
				Expect(err).ToNot(HaveOccurred(), "Failed to get master nodes list; %s", err)

				masterNodesCount := len(nodesList)
				Expect(masterNodesCount).To(Equal(3),
					"Error in master nodes count; found master nodes count is %s", masterNodesCount)

				glog.V(100).Infof("Checking all master nodes are Ready")
				var isReady bool
				isReady, err = nodes.WaitForAllNodesAreReady(
					APIClient,
					5*time.Second,
					SamsungConfig.ControlPlaneLabelListOption)
				Expect(err).ToNot(HaveOccurred(), "Error getting master nodes list: %s", err)
				Expect(isReady).To(Equal(true), "Error in master nodes status, not all Master node are Ready; %s", isReady)

				glog.V(100).Infof("Checking that the clusterversion is available")
				_, err = clusterversion.Pull(APIClient)
				Expect(err).ToNot(HaveOccurred(), "Error accessing csv: %s", err)

				glog.V(100).Infof("Asserting clusteroperators availability")
				var coBuilder []*clusteroperator.Builder
				coBuilder, err = clusteroperator.List(APIClient)
				Expect(err).To(BeNil(), fmt.Sprintf("ClusterOperator List not found: %s", err))
				Expect(len(coBuilder)).ToNot(Equal(0), "Empty clusterOperators list received")

				_, err = clusteroperator.WaitForAllClusteroperatorsAvailable(APIClient, 60*time.Second)
				Expect(err).ToNot(HaveOccurred(), "Error waiting for all available clusteroperators: %s", err)
			})

		It("Asserts time sync was successfully applied for master nodes", polarion.ID("60028"),
			Label(samsungparams.LabelSamsungVCoreDeployment), func() {
				isChronyApplied := false

				mcp := mco.NewMCPBuilder(APIClient, samsungparams.MasterNodeRole)
				Expect(mcp.Exists()).To(BeTrue(), "Error find master mcp")

				for _, source := range mcp.Object.Status.Configuration.Source {
					if source.Name == samsungparams.MasterChronyConfigName {
						isChronyApplied = true

						break
					}
				}
				Expect(isChronyApplied).To(BeTrue(), "Error assert time sync was applied")
			})

		It("Asserts time sync was successfully applied for worker nodes", polarion.ID("60029"),
			Label(samsungparams.LabelSamsungVCoreDeployment), func() {
				isChronyApplied := false

				mcp := mco.NewMCPBuilder(APIClient, samsungparams.WorkerNodeRole)
				Expect(mcp.Exists()).To(BeTrue(), "Error find worker mcp")

				for _, source := range mcp.Object.Status.Configuration.Source {
					if source.Name == samsungparams.WorkerChronyConfigName {
						isChronyApplied = true

						break
					}
				}
				Expect(isChronyApplied).To(BeTrue(), "Error assert time sync was applied")
			})

		It("Asserts full set of ODF nodes was deployed", polarion.ID("59442"),
			Label("samsungvcoreodf"), func() {

				mcp := mco.NewMCPBuilder(APIClient, samsungparams.SamsungOdfMcpName)
				Expect(mcp.Exists()).To(BeTrue(), "Error to find ODF mcp")

				glog.V(100).Infof("Checking available ODF nodes count")
				nodesList, err := nodes.List(APIClient, SamsungConfig.OdfLabelListOption)
				Expect(err).ToNot(HaveOccurred(), "Failed to get ODF nodes list; %s", err)
				Expect(len(nodesList)).ToNot(Equal(0), "ODF nodes list is empty")
			})

		It("Asserts samsung-cnf mcp found", polarion.ID("60049"),
			Label("samsungvcoredeployment"), func() {

				mcp := mco.NewMCPBuilder(APIClient, samsungparams.SamsungCnfMcpName)
				Expect(mcp.Exists()).To(BeTrue(), "Error to find samsung-cnf mcp")

				glog.V(100).Infof("Checking samsung-cnf mcp condition state")
				Expect(mcp.IsInCondition("Updated")).To(BeTrue(), "samsung-cnf mcp failed to update")
			})

		It("Asserts full set of samsung-cnf nodes was deployed", polarion.ID("59505"),
			Label("samsungvcoredeployment"), func() {

				mcp := mco.NewMCPBuilder(APIClient, samsungparams.SamsungCnfMcpName)
				Expect(mcp.Exists()).To(BeTrue(), "Error to find samsung-cnf mcp")

				glog.V(100).Infof("Checking available samsung-cnf nodes count")
				nodesList, err := nodes.List(APIClient, SamsungConfig.SamsungCnfLabelListOption)
				Expect(err).ToNot(HaveOccurred(), "Failed to get samsung-cnf nodes list; %s", err)
				Expect(len(nodesList)).ToNot(Equal(0), "samsung-cnf nodes list is empty")
			})

		It("Asserts samsung-pp mcp found", polarion.ID("60050"),
			Label("samsungvcoredeployment"), func() {

				mcp := mco.NewMCPBuilder(APIClient, samsungparams.SamsungPpMcpName)
				Expect(mcp.Exists()).To(BeTrue(), "Error to find samsung-pp mcp")

				glog.V(100).Infof("Checking samsung-pp mcp condition state")
				Expect(mcp.IsInCondition("Updated")).To(BeTrue(), "samsung-ppf mcp failed to update")
			})

		It("Asserts full set of samsung-pp nodes was deployed", polarion.ID("59506"),
			Label("samsungvcoredeployment"), func() {

				mcp := mco.NewMCPBuilder(APIClient, samsungparams.SamsungPpMcpName)
				Expect(mcp.Exists()).To(BeTrue(), "Error to find samsung-pp mcp")

				glog.V(100).Infof("Checking available samsung-pp nodes count")
				nodesList, err := nodes.List(APIClient, SamsungConfig.SamsungPpLabelListOption)
				Expect(err).ToNot(HaveOccurred(), "Failed to get samsung-pp nodes list; %s", err)
				Expect(len(nodesList)).ToNot(Equal(0), "samsung-pp nodes list is empty")
			})
	})
