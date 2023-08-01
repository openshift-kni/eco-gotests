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
)

var _ = Describe("Day1Day2", Ordered, Label(tsparams.LabelSuite), ContinueOnFailure, func() {
	BeforeAll(func() {
		By("Creating a new instance of NMState instance")
		err := netnmstate.CreateNewNMStateAndWaitUntilItsRunning(netparam.DefaultTimeout)
		Expect(err).ToNot(HaveOccurred(), "Failed to create NMState instance")

		By("Verifying that the cluster deployed via bond interface" +
			" with enslaved Vlan interfaces which based on SR-IOV VFs")
		isClusterDeployedWithBondVlanVfs()
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
		Skip("ToDo")
	})
})

func isClusterDeployedWithBondVlanVfs() {
	By("Verifying that the cluster deployed via bond interface")

	var (
		bondName string
		err      error
	)

	workerNodeList := nodes.NewBuilder(APIClient, NetConfig.WorkerLabelMap)
	Expect(workerNodeList.Discover()).ToNot(HaveOccurred(), "Failed to discover worker nodes")

	for _, worker := range workerNodeList.Objects {
		var isBondPrimaryInterface bool
		isBondPrimaryInterface, bondName, err = netnmstate.IsPrimaryInterfaceBond(worker.Definition.Name)
		Expect(err).ToNot(HaveOccurred(), "Failed to check primary interface")

		if !isBondPrimaryInterface {
			Skip("The test skipped because of primary interface is not a bond interface")
		}
	}

	By("Gathering enslave interfaces for the bond interface")

	bondInterfaceVlanSlaves, err := netnmstate.GetBondSlaves(bondName, workerNodeList.Objects[0].Definition.Name)
	Expect(err).ToNot(HaveOccurred(), "Failed to get bond slave interfaces")

	By("Verifying that enslave interfaces are vlan interfaces and base-interface for Vlan interface is a VF interface")

	for _, bondSlave := range bondInterfaceVlanSlaves {
		baseInterface, err := netnmstate.GetBaseVlanInterface(bondSlave, workerNodeList.Objects[0].Definition.Name)
		if err != nil && strings.Contains(err.Error(), "it is not a vlan type") {
			Skip("The test skipped because of bond slave interfaces are not vlan interfaces")
		}

		Expect(err).ToNot(HaveOccurred(), "Failed to get Vlan base interface")

		// If a Vlan baseInterface has SR-IOV PF, it means that the baseInterface is VF.
		_, err = day1day2env.GetSrIovPf(baseInterface, workerNodeList.Objects[0].Definition.Name)
		if err != nil && strings.Contains(err.Error(), "No such file or directory") {
			Skip("The test skipped because of bond slave interfaces are not vlan interfaces")
		}

		Expect(err).ToNot(HaveOccurred(), "Failed to get SR-IOV PF interface")
	}
}
