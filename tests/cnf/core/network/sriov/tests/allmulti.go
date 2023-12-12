package allmulti

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/nad"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/sriov"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netenv"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netinittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/sriov/internal/tsparams"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	srIovPolicyOneName         = "sriov-policy-node-one"
	srIovPolicyPFOneResName    = "sriovpolicyone"
	srIovNetworkAllMultiSamePF = "sriovnet-allmulti-same"
	srIovNetworkDefaultSamePF  = "sriovnet-default-same"
)

var workerNodes []*nodes.Builder

var _ = Describe("allmulti", Ordered, Label(tsparams.LabelSuite), ContinueOnFailure, func() {

	BeforeAll(func() {
		By("Discover worker nodes")
		var err error

		workerNodes, err = nodes.List(APIClient,
			metaV1.ListOptions{LabelSelector: labels.Set(NetConfig.WorkerLabelMap).String()})
		Expect(err).ToNot(HaveOccurred(), "Fail to discover nodes")

		By("Collecting SR-IOV interfaces for allmulti testing")
		srIovInterfacesUnderTest, err := NetConfig.GetSriovInterfaces(1)
		Expect(err).ToNot(HaveOccurred(), "Failed to retrieve SR-IOV interfaces for testing")

		By("Define and create sriov network polices")
		nodeSelectorWorker0 := map[string]string{
			"kubernetes.io/hostname": workerNodes[0].Definition.Name,
		}

		_, err = sriov.NewPolicyBuilder(
			APIClient,
			srIovPolicyOneName,
			NetConfig.SriovOperatorNamespace,
			srIovPolicyPFOneResName,
			10,
			[]string{fmt.Sprintf("%s#0-9", srIovInterfacesUnderTest[0])},
			nodeSelectorWorker0).WithMTU(9000).WithVhostNet(true).Create()
		Expect(err).ToNot(HaveOccurred(), "failed to create an sriov policy")

		By("Define and create sriov network with allmuti enabled")
		defineAndCreateSrIovNetwork(srIovNetworkAllMultiSamePF, srIovPolicyPFOneResName, true)

		By("Define and create sriov network with allmuti disabled")
		defineAndCreateSrIovNetwork(srIovNetworkDefaultSamePF, srIovPolicyPFOneResName, false)

		By("Waiting until cluster MCP and SR-IOV are stable")
		err = netenv.WaitForSriovAndMCPStable(
			APIClient, tsparams.MCOWaitTimeout, time.Minute, NetConfig.CnfMcpLabel, NetConfig.SriovOperatorNamespace)
		Expect(err).ToNot(HaveOccurred(), "fail cluster is not stable")
	})

	AfterAll(func() {
		By("Removing all SR-IOV Policy")
		err := sriov.CleanAllNetworkNodePolicies(APIClient, NetConfig.SriovOperatorNamespace, metaV1.ListOptions{})
		Expect(err).ToNot(HaveOccurred(), "Fail to clean srIovPolicy")

		By("Removing all srIovNetworks")
		err = sriov.CleanAllNetworksByTargetNamespace(
			APIClient, NetConfig.SriovOperatorNamespace, tsparams.TestNamespaceName, metaV1.ListOptions{})
		Expect(err).ToNot(HaveOccurred(), "Fail to clean sriov networks")
	})
})

func defineAndCreateSrIovNetwork(srIovNetwork, resName string, allMulti bool) {
	srIovNetworkObject := sriov.NewNetworkBuilder(
		APIClient, srIovNetwork, NetConfig.SriovOperatorNamespace, tsparams.TestNamespaceName, resName).
		WithStaticIpam().WithIPAddressSupport().WithMacAddressSupport()

	if allMulti {
		srIovNetworkObject.WithTrustFlag(true).WithMetaPluginAllMultiFlag(true)
	}

	srIovNetworkObject, err := srIovNetworkObject.Create()
	Expect(err).ToNot(HaveOccurred(), "Fail to create sriov network")

	Eventually(func() bool {
		_, err := nad.Pull(APIClient, srIovNetworkObject.Object.Name, tsparams.TestNamespaceName)

		return err == nil
	}, tsparams.WaitTimeout, tsparams.RetryInterval).Should(BeTrue(), "Fail to pull "+
		"NetworkAttachmentDefinition")
}
