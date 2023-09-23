package tests

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/nmstate"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"

	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/day1day2/internal/day1day2env"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/day1day2/internal/tsparams"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netinittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netnmstate"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netparam"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

var _ = Describe("Day1Day2", Ordered, Label(tsparams.LabelSuite), ContinueOnFailure, func() {
	var (
		workerNodeList []*nodes.Builder
	)

	BeforeAll(func() {
		var err error
		By("Discovering worker nodes")
		workerNodeList, err = nodes.List(APIClient,
			metav1.ListOptions{LabelSelector: labels.Set(NetConfig.WorkerLabelMap).String()},
		)
		Expect(err).ToNot(HaveOccurred(), "Failed to discover worker nodes")

		By("Creating a new instance of NMState instance")
		err = netnmstate.CreateNewNMStateAndWaitUntilItsRunning(netparam.DefaultTimeout)
		Expect(err).ToNot(HaveOccurred(), "Failed to create NMState instance")

		By("Verifying that the cluster deployed via bond interface" +
			" with enslaved Vlan interfaces which based on SR-IOV VFs")
		checkThatWorkersDeployedWithBondVlanVfs(workerNodeList)

	})

	AfterAll(func() {
		By("Cleaning test namespace")
		err := namespace.NewBuilder(APIClient, tsparams.TestNamespaceName).CleanObjects(
			netparam.DefaultTimeout, pod.GetGVR())
		Expect(err).ToNot(HaveOccurred(), "Failed to clean test namespace")

		By("Removing NMState policies")
		err = nmstate.CleanAllNMStatePolicies(APIClient)
		Expect(err).ToNot(HaveOccurred(), "Failed to remove all NMState policies")
	})

	It("Day2 Bond: change miimon configuration", polarion.ID("63881"), func() {
		By("Collecting information about test interfaces")
		bondName, err := netnmstate.GetPrimaryInterfaceBond(workerNodeList[0].Definition.Name)
		Expect(err).ToNot(HaveOccurred(), "Failed to get Bond primary interface name")
		bondInterfaceVlanSlaves, err := netnmstate.GetBondSlaves(bondName, workerNodeList[0].Definition.Name)
		Expect(err).ToNot(HaveOccurred(), "Failed to get bond slave interfaces")

		By("Saving miimon value on the bond interface before the test")
		defaultMiimonValue, err := day1day2env.GetBondInterfaceMiimon(workerNodeList[0].Definition.Name, bondName)
		Expect(err).ToNot(HaveOccurred(), "Failed to get miimon configuration")
		newExpectedMiimonValue := defaultMiimonValue + 10

		By("Configuring miimon on the bond interface")

		nmstatePolicy := nmstate.NewPolicyBuilder(APIClient, "miimon", NetConfig.WorkerLabelMap).
			WithBondInterface(bondInterfaceVlanSlaves, bondName, "active-backup").
			WithOptions(netnmstate.WithOptionMiimon(uint64(newExpectedMiimonValue), bondName))
		err = netnmstate.CreatePolicyAndWaitUntilItsAvailable(netparam.DefaultTimeout, nmstatePolicy)
		Expect(err).ToNot(HaveOccurred(), "Failed to create NMState network policy")

		By("Verifying that expected miimon value is configured")

		for _, workerNode := range workerNodeList {
			currentMiimonValue, err := day1day2env.GetBondInterfaceMiimon(workerNode.Object.Name, bondName)
			Expect(err).ToNot(HaveOccurred(), "Failed to get miimon configuration")
			Expect(currentMiimonValue).To(Equal(newExpectedMiimonValue), "Miimon has unexpected value")
		}

		By("Verifying workers are available over the bond interface after miimon re-config")
		err = day1day2env.CheckConnectivityBetweenMasterAndWorkers()
		Expect(err).ToNot(HaveOccurred(), "Connectivity check failed")

		By("Restoring miimon configuration")
		nmstatePolicy = nmstate.NewPolicyBuilder(APIClient, "restoremiimon", NetConfig.WorkerLabelMap).
			WithBondInterface(bondInterfaceVlanSlaves, bondName, "active-backup").
			WithOptions(netnmstate.WithOptionMiimon(uint64(defaultMiimonValue), bondName))
		err = netnmstate.CreatePolicyAndWaitUntilItsAvailable(netparam.DefaultTimeout, nmstatePolicy)
		Expect(err).ToNot(HaveOccurred(), "Failed to create NMState network policy")

		By("Verifying that miimon is restored")

		for _, workerNode := range workerNodeList {
			currentMiimon, err := day1day2env.GetBondInterfaceMiimon(workerNode.Object.Name, bondName)
			Expect(err).ToNot(HaveOccurred(), "Failed to get miimon configuration")
			Expect(currentMiimon).To(Equal(defaultMiimonValue), "miimon has unexpected value")
		}

		By("Verifying workers are available over the bond interface after miimon reverted to default")
		err = day1day2env.CheckConnectivityBetweenMasterAndWorkers()
		Expect(err).ToNot(HaveOccurred(), "Connectivity check failed")
	})
})

func checkThatWorkersDeployedWithBondVlanVfs(workerNodes []*nodes.Builder) {
	By("Verifying that the cluster deployed via bond interface")

	var (
		bondName string
		err      error
	)

	for _, worker := range workerNodes {
		bondName, err = netnmstate.GetPrimaryInterfaceBond(worker.Definition.Name)
		Expect(err).ToNot(HaveOccurred(), "Failed to check primary interface")

		if bondName == "" {
			Skip("The test skipped because of primary interface is not a bond interface")
		}
	}

	By("Gathering enslave interfaces for the bond interface")

	bondInterfaceVlanSlaves, err := netnmstate.GetBondSlaves(bondName, workerNodes[0].Definition.Name)
	Expect(err).ToNot(HaveOccurred(), "Failed to get bond slave interfaces")

	By("Verifying that enslave interfaces are vlan interfaces and base-interface for Vlan interface is a VF interface")

	for _, bondSlave := range bondInterfaceVlanSlaves {
		baseInterface, err := netnmstate.GetBaseVlanInterface(bondSlave, workerNodes[0].Definition.Name)
		if err != nil && strings.Contains(err.Error(), "it is not a vlan type") {
			Skip("The test skipped because of bond slave interfaces are not vlan interfaces")
		}

		Expect(err).ToNot(HaveOccurred(), "Failed to get Vlan base interface")

		// If a Vlan baseInterface has SR-IOV PF, it means that the baseInterface is VF.
		_, err = day1day2env.GetSrIovPf(baseInterface, workerNodes[0].Definition.Name)
		if err != nil && strings.Contains(err.Error(), "No such file or directory") {
			Skip("The test skipped because of bond slave interfaces are not vlan interfaces")
		}

		Expect(err).ToNot(HaveOccurred(), "Failed to get SR-IOV PF interface")
	}
}
