package tests

import (
	"fmt"
	"time"

	netattdefv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/namespace"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/nmstate"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/nodes"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/pod"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/reportxml"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/sriov"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/core/network/internal/juniper"
	. "github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/core/network/internal/netinittools"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/core/network/internal/netparam"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/core/network/sriov/internal/sriovenv"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/core/network/sriov/internal/tsparams"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	srIovPolicyNode1Name    = "sriov-policy-lacp-node-1"
	srIovPolicyNode2Name    = "sriov-policy-lacp-node-2"
	srIovPolicyNode0ResName = "sriovpolicylacpnode0"
	srIovPolicyNode1ResName = "sriovpolicylacpnode1"
)

var _ = Describe("Day1Day2", Ordered, Label(tsparams.LabelSuite), ContinueOnFailure, func() {
	var (
		workerNodeList           []*nodes.Builder
		juniperSession           *juniper.JunosSession
		switchInterfaces         []string
		switchLagNames           []string
		srIovInterfacesUnderTest []string
	)

	BeforeAll(func() {
		var err error

		By("Discover worker nodes")
		workerNodeList, err = nodes.List(APIClient,
			metav1.ListOptions{LabelSelector: labels.Set(NetConfig.WorkerLabelMap).String()})
		Expect(err).ToNot(HaveOccurred(), "Fail to discover worker nodes")

		Expect(err).ToNot(HaveOccurred(), "Failed to discover worker nodes")

		By("Collecting SR-IOV interfaces for qinq testing")
		srIovInterfacesUnderTest, err = NetConfig.GetSriovInterfaces(1)
		Expect(err).ToNot(HaveOccurred(), "Failed to retrieve SR-IOV interfaces for testing")

		Expect(sriovenv.ValidateSriovInterfaces(workerNodeList, 2)).ToNot(HaveOccurred(),
			"Failed to get required SR-IOV interfaces")

		By(fmt.Sprintf("Define and create sriov network policy on %s", workerNodeList[0].Definition.Name))
		nodeSelectorWorker0 := map[string]string{
			"kubernetes.io/hostname": workerNodeList[0].Definition.Name,
		}

		_, err = sriov.NewPolicyBuilder(
			APIClient,
			srIovPolicyNode1Name,
			NetConfig.SriovOperatorNamespace,
			srIovPolicyNode0ResName,
			6,
			[]string{fmt.Sprintf("%s#0-5", srIovInterfacesUnderTest[0])},
			nodeSelectorWorker0).WithMTU(9000).WithVhostNet(true).Create()
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to create an sriov policy on %s",
			workerNodeList[0].Definition.Name))

		By(fmt.Sprintf("Define and create sriov network policy on %s", workerNodeList[1].Definition.Name))
		nodeSelectorWorker1 := map[string]string{
			"kubernetes.io/hostname": workerNodeList[1].Definition.Name,
		}

		_, err = sriov.NewPolicyBuilder(
			APIClient,
			srIovPolicyNode2Name,
			NetConfig.SriovOperatorNamespace,
			srIovPolicyNode1ResName,
			6,
			[]string{fmt.Sprintf("%s#0-5", srIovInterfacesUnderTest[0])},
			nodeSelectorWorker1).WithMTU(9000).WithVhostNet(true).Create()
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to create an sriov policy on %s",
			workerNodeList[1].Definition.Name))

		By("Getting switch credentials")
		switchCredentials, err := juniper.NewSwitchCredentials()
		Expect(err).ToNot(HaveOccurred(), "Failed to get switch credentials")

		By("Opening management connection to switch")
		juniperSession, err = juniper.NewSession(
			switchCredentials.SwitchIP, switchCredentials.User, switchCredentials.Password)
		Expect(err).ToNot(HaveOccurred(), "Failed to open a switch session")

		By("Collecting switch interfaces")
		switchInterfaces, err = NetConfig.GetPrimarySwitchInterfaces()
		Expect(err).ToNot(HaveOccurred(), "Failed to get switch interfaces")

		By("Collecting switch LAG names")
		switchLagNames, err = NetConfig.GetSwitchLagNames()
		Expect(err).ToNot(HaveOccurred(), "Failed to get switch LAG names")
	})

	AfterEach(func() {
		if len(juniper.InterfaceConfigs) > 0 {
			By("Reverting initial switch interface configurations")
			recoverSwitchConfiguration(juniperSession, switchInterfaces, switchLagNames)

		}

		By("Cleaning test namespace")
		err := namespace.NewBuilder(APIClient, tsparams.TestNamespaceName).CleanObjects(
			netparam.DefaultTimeout, pod.GetGVR())
		Expect(err).ToNot(HaveOccurred(), "Failed to clean test namespace")

		By("Removing NMState policies")
		err = nmstate.CleanAllNMStatePolicies(APIClient)
		Expect(err).ToNot(HaveOccurred(), "Failed to remove all NMState policies")
	})

	It("Valiadte SRIOV VF LACP on same network card", reportxml.ID("99999"), func() {
		By("")

	})
})

func recoverSwitchConfiguration(juniperSession *juniper.JunosSession, switchInterfaces, lagInterfaces []string) {
	err := juniper.RestoreSwitchInterfacesConfiguration(juniperSession, switchInterfaces)
	Expect(err).ToNot(HaveOccurred(), "Failed to restore initial switch interfaces configurations")

	err = juniper.DeleteInterfaces(juniperSession, lagInterfaces)
	Expect(err).ToNot(HaveOccurred(), "Failed to delete switch LAG interfaces")
}

func waitForSwitchInterfaceUp(juniperSession *juniper.JunosSession, switchLagName string) {
	Eventually(func() bool {
		isBondInterfaceUp, err := juniper.IsSwitchInterfaceUp(juniperSession, switchLagName)
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to get status of switch LAG interface %s", switchLagName))

		return isBondInterfaceUp
	}, 1*time.Minute, 5*time.Second).Should(BeTrue(), "Bond interface is not Up on the switch")
}

func testBondFailOver(juniperSession *juniper.JunosSession, switchInterfaces []string) {
	By("Disabling one bond slave interface on the switch and check the traffic again via secondary bond interface")

	err := juniper.DisableSwitchInterface(juniperSession, switchInterfaces[0])
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to shutdown switch interface %s", switchInterfaces[0]))

	By(fmt.Sprintf("Disabling secondary LAG slave interface %s, bring first LAG slave interface %s back"+
		" and check the traffic again", switchInterfaces[1], switchInterfaces[0]))

	err = juniper.EnableSwitchInterface(juniperSession, switchInterfaces[0])
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to turn on the switch interface %s", switchInterfaces[0]))

	err = juniper.DisableSwitchInterface(juniperSession, switchInterfaces[1])
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to shutdown switch interface %s", switchInterfaces[1]))

	waitForSwitchInterfaceUp(juniperSession, switchInterfaces[0])

	By("Verifying workers are still available over the bond interface")

}

// DefineBondNad returns network attachment definition for a Bond interface.
func DefineBondNad(nadName string,
	bondType string,
	mtu int,
	numberSlaveInterfaces int, ipam string) (*netattdefv1.NetworkAttachmentDefinition, error) {
	slaveInterfaces := bondNADSlaveInterfaces(numberSlaveInterfaces)
	bondNad := &netattdefv1.NetworkAttachmentDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nadName,
			Namespace: tsparams.TestNamespaceName,
		},
		Spec: netattdefv1.NetworkAttachmentDefinitionSpec{
			Config: fmt.Sprintf(
				`{"type": "bond", "cniVersion": "0.3.1", "name": "%s",
"mode": "%s", "failOverMac": 1, "linksInContainer": true, "miimon": "100", "mtu": %d,
"links": [%s], "capabilities": {"ips": true}, `,
				nadName, bondType, mtu, slaveInterfaces),
		}}

	switch ipam {
	case "static":
		bondNad.Spec.Config += fmt.Sprintf(`"ipam": {"type": "%s"}}`, ipam)
	case "whereabouts":
		bondNad.Spec.Config += fmt.Sprintf(`"ipam": {"type": "%s", "range": "%s"}}`,
			ipam, "2001:1db8:85a3::0/126")
	default:
		return nil, fmt.Errorf("wrong ipam type %s", ipam)
	}

	return bondNad, nil
}

// bondNADSlaveInterfaces returns string with slave interfaces for Bond interface Network Attachment Definition.
func bondNADSlaveInterfaces(numberInterfaces int) string {
	slaveInterfaces := `{"name": "net1"}`

	for i := 2; i <= numberInterfaces; i++ {
		slaveInterfaces += fmt.Sprintf(`,{"name": "net%d"}`, i)
	}

	return slaveInterfaces
}
