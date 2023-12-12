package tests

import (
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/nmstate"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"

	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/day1day2/internal/day1day2env"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/day1day2/internal/juniper"
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
		workerNodeList          []*nodes.Builder
		bondName                string
		bondInterfaceVlanSlaves []string
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
		bondName, bondInterfaceVlanSlaves = checkThatWorkersDeployedWithBondVlanVfs(workerNodeList)

	})

	AfterEach(func() {
		By("Cleaning test namespace")
		err := namespace.NewBuilder(APIClient, tsparams.TestNamespaceName).CleanObjects(
			netparam.DefaultTimeout, pod.GetGVR())
		Expect(err).ToNot(HaveOccurred(), "Failed to clean test namespace")

		By("Removing NMState policies")
		err = nmstate.CleanAllNMStatePolicies(APIClient)
		Expect(err).ToNot(HaveOccurred(), "Failed to remove all NMState policies")
	})

	It("VF: change QOS configuration", polarion.ID("63926"), func() {
		By("Collecting information about test interfaces")
		vfInterface, err := netnmstate.GetBaseVlanInterface(bondInterfaceVlanSlaves[0], workerNodeList[0].Definition.Name)
		Expect(err).ToNot(HaveOccurred(), "Failed to get VF base interface")
		pfUnderTest, err := day1day2env.GetSrIovPf(vfInterface, workerNodeList[0].Definition.Name)
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to get SR-IOV PF for VF %s", vfInterface))

		By(fmt.Sprintf("Saving MaxTxRate value on the first VF of interface %s before the test", pfUnderTest))
		defaultMaxTxRate, err := day1day2env.GetFirstVfInterfaceMaxTxRate(workerNodeList[0].Definition.Name, pfUnderTest)
		Expect(err).ToNot(HaveOccurred(), "Failed to get default MaxTxRate value")

		By("Configuring MaxTxRate on the first VF")
		newExpectedMaxTxRateValue := 200
		nmstatePolicy := nmstate.NewPolicyBuilder(APIClient, "qos", NetConfig.WorkerLabelMap).
			WithInterfaceAndVFs(pfUnderTest, 3).
			WithOptions(netnmstate.WithOptionMaxTxRateOnFirstVf(uint64(newExpectedMaxTxRateValue), pfUnderTest))
		err = netnmstate.CreatePolicyAndWaitUntilItsAvailable(netparam.DefaultTimeout, nmstatePolicy)
		Expect(err).ToNot(HaveOccurred(), "Failed to create NMState network policy")

		By("Verifying that expected MaxTxRate value is configured")
		for _, workerNode := range workerNodeList {
			currentMaxTxRateValue, err := day1day2env.GetFirstVfInterfaceMaxTxRate(workerNode.Object.Name, pfUnderTest)
			Expect(err).ToNot(HaveOccurred(), "Failed to get MaxTxRate configuration")
			Expect(currentMaxTxRateValue).To(Equal(newExpectedMaxTxRateValue), "MaxTxRate has unexpected value")
		}

		By("Verifying workers are available over the bond interface after MaxTxRate re-config")
		err = day1day2env.CheckConnectivityBetweenMasterAndWorkers()
		Expect(err).ToNot(HaveOccurred(), "Connectivity check failed")

		By("Restoring MaxTxRate configuration")
		nmstatePolicy = nmstate.NewPolicyBuilder(APIClient, "restoreqos", NetConfig.WorkerLabelMap).
			WithInterfaceAndVFs(pfUnderTest, 3).
			WithOptions(netnmstate.WithOptionMaxTxRateOnFirstVf(uint64(defaultMaxTxRate), pfUnderTest))
		err = netnmstate.CreatePolicyAndWaitUntilItsAvailable(netparam.DefaultTimeout, nmstatePolicy)
		Expect(err).ToNot(HaveOccurred(), "Failed to create NMState network policy")

		By("Verifying that MaxTxRate is restored")
		for _, workerNode := range workerNodeList {
			currentMaxTxRateValue, err := day1day2env.GetFirstVfInterfaceMaxTxRate(workerNode.Object.Name, pfUnderTest)
			Expect(err).ToNot(HaveOccurred(), "Failed to get MaxTxRate configuration")
			Expect(currentMaxTxRateValue).To(Equal(defaultMaxTxRate), "MaxTxRate has unexpected value")
		}

		By("Verifying workers are available over the bond interface after MaxTxRate reverted to default")
		err = day1day2env.CheckConnectivityBetweenMasterAndWorkers()
		Expect(err).ToNot(HaveOccurred(), "Connectivity check failed")
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
			WithOptions(netnmstate.WithBondOptionMiimon(uint64(newExpectedMiimonValue), bondName))
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
			WithOptions(netnmstate.WithBondOptionMiimon(uint64(defaultMiimonValue), bondName))
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

	Context("", func() {
		const policyNameBondMode = "changebondmode"

		var (
			juniperSession   *juniper.JunosSession
			switchInterfaces []string
			clusterVlan      string
			switchLagNames   []string
		)

		BeforeAll(func() {
			By("Getting switch credentials")
			switchCredentials, err := juniper.NewSwitchCredentials()
			Expect(err).ToNot(HaveOccurred(), "Failed to get switch credentials")

			By("Opening management connection to switch")
			juniperSession, err = juniper.NewSession(
				switchCredentials.SwitchIP, switchCredentials.User, switchCredentials.Password)
			Expect(err).ToNot(HaveOccurred(), "Failed to open a switch session")

			By("Collecting switch interfaces")
			switchInterfaces, err = NetConfig.GetSwitchInterfaces()
			Expect(err).ToNot(HaveOccurred(), "Failed to get switch interfaces")

			By("Collecting vlan id")
			clusterVlan, err = NetConfig.GetClusterVlan()
			Expect(err).ToNot(HaveOccurred(), "Failed to get cluster vlan")

			By("Collecting switch LAG names")
			switchLagNames, err = NetConfig.GetSwitchLagNames()
			Expect(err).ToNot(HaveOccurred(), "Failed to get switch LAG names")
		})

		AfterEach(func() {
			if len(juniper.InterfaceConfigs) > 0 {
				By("Reverting initial switch interface configurations")
				recoverSwitchConfiguration(juniperSession, switchInterfaces, switchLagNames)

				By("Verifying workers are still available over the bond interface")
				err := day1day2env.CheckConnectivityBetweenMasterAndWorkers()
				Expect(err).ToNot(HaveOccurred(), "Connectivity check failed")
			}

			By("Reverting active-backup bond mode on the bond interfaces")
			nmstatePolicy := nmstate.NewPolicyBuilder(APIClient, policyNameBondMode, NetConfig.WorkerLabelMap).
				WithBondInterface(bondInterfaceVlanSlaves, bondName, "active-backup").
				WithOptions(netnmstate.WithBondOptionFailOverMac("none", bondName))
			err := netnmstate.UpdatePolicyAndWaitUntilItsAvailable(netparam.DefaultTimeout, nmstatePolicy)
			Expect(err).ToNot(HaveOccurred(), "Failed to update NMState network policy")

			By("Checking that Bond mode is restored")
			validateBondType("active-backup", bondName, workerNodeList[0].Object.Name)
		})

		It("Day2 Bond: change mode configuration", polarion.ID("63882"), func() {
			By("Creating NMState policy to change a bond mode")
			nmstatePolicy := nmstate.NewPolicyBuilder(APIClient, policyNameBondMode, NetConfig.WorkerLabelMap).
				WithBondInterface(bondInterfaceVlanSlaves, bondName, "balance-rr").
				WithOptions(netnmstate.WithBondOptionFailOverMac("active", bondName))
			err := netnmstate.CreatePolicyAndWaitUntilItsAvailable(netparam.DefaultTimeout, nmstatePolicy)
			Expect(err).ToNot(HaveOccurred(), "Failed to create NMState network policy")

			By("Checking that Bond mode is configured")
			validateBondType("balance-rr", bondName, workerNodeList[0].Object.Name)

			By(fmt.Sprintf("Removing all configuration from the switch interfaces %v", switchInterfaces))
			err = juniper.DumpInterfaceConfigs(juniperSession, switchInterfaces)
			Expect(err).ToNot(HaveOccurred(), "Failed to save initial switch interfaces configs")
			err = juniper.RemoveAllConfigurationFromInterfaces(juniperSession, switchInterfaces)
			Expect(err).ToNot(HaveOccurred(), "Failed to remove configuration from the switch interfaces")

			By("Configuring aggregated interface on a switch")
			configureLAGsOnSwitch(juniperSession, clusterVlan, switchInterfaces, switchLagNames)

			By("Verifying workers are still available over the bond interface")
			err = day1day2env.CheckConnectivityBetweenMasterAndWorkers()
			Expect(err).ToNot(HaveOccurred(), "Connectivity check failed")

			By("Disabling one bond slave interface on the switch and check the traffic again via secondary bond interface")
			err = juniper.DisableSwitchInterface(juniperSession, switchInterfaces[0])
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to shutdown switch interface %s", switchInterfaces[0]))

			err = day1day2env.CheckConnectivityBetweenMasterAndWorkers()
			Expect(err).ToNot(HaveOccurred(), "Connectivity check failed")

			By(fmt.Sprintf("Disabling secondary LAG slave interface %s, bring first LAG slave interface %s back"+
				" and check the traffic again", switchInterfaces[1], switchInterfaces[0]))

			err = juniper.EnableSwitchInterface(juniperSession, switchInterfaces[0])
			Expect(err).ToNot(HaveOccurred(),
				fmt.Sprintf("Failed to turn on the switch interface %s", switchInterfaces[0]))

			err = juniper.DisableSwitchInterface(juniperSession, switchInterfaces[1])
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to shutdown switch interface %s", switchInterfaces[1]))

			waitForSwitchInterfaceUp(juniperSession, switchLagNames[0])

			By("Verifying workers are still available over the bond interface")
			err = day1day2env.CheckConnectivityBetweenMasterAndWorkers()
			Expect(err).ToNot(HaveOccurred(), "Connectivity check failed")
		})
	})
})

func checkThatWorkersDeployedWithBondVlanVfs(workerNodes []*nodes.Builder) (string, []string) {
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

	return bondName, bondInterfaceVlanSlaves
}

func configureLAGsOnSwitch(
	juniperSession *juniper.JunosSession, clusterVlan string, switchInterfaces, lagInterfaces []string) {
	err := juniper.SetNonLacpLag(juniperSession, []string{switchInterfaces[0], switchInterfaces[1]},
		lagInterfaces[0])
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to create switch LAG interface %s with enslave itnerfaces: %s, %s",
			lagInterfaces[0], switchInterfaces[0], switchInterfaces[1]))

	err = juniper.SetNonLacpLag(juniperSession, []string{switchInterfaces[2], switchInterfaces[3]},
		lagInterfaces[1])
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to create switch LAG interface %s with enslave itnerfaces: %s, %s",
			lagInterfaces[1], switchInterfaces[2], switchInterfaces[3]))

	for _, lagInterface := range lagInterfaces {
		err = juniper.SetVlanOnTrunkInterface(juniperSession, clusterVlan, lagInterface)
		Expect(err).ToNot(HaveOccurred(), "Failed to configure VLAN on switch LAG interfaces")
	}
}

func recoverSwitchConfiguration(juniperSession *juniper.JunosSession, switchInterfaces, lagInterfaces []string) {
	err := juniper.RestoreSwitchInterfacesConfiguration(juniperSession, switchInterfaces)
	Expect(err).ToNot(HaveOccurred(), "Failed to restore initial switch interfaces configurations")

	err = juniper.DeleteInterfaces(juniperSession, lagInterfaces)
	Expect(err).ToNot(HaveOccurred(), "Failed to delete switch LAG interfaces")
}

func validateBondType(bondTypeName, bondName, workerName string) {
	By("Checking Bond mode on a worker")

	bondModeViaCmd, err := day1day2env.GetBondModeViaCmd(bondName, workerName)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to get bond mode interface %s on worker %s via cmd", bondName, workerName))
	Expect(bondModeViaCmd).To(ContainSubstring(bondTypeName), "Bond mode is not expected one")

	By("Checking Bond mode via NMState")

	bondModeViaNmstate, err := netnmstate.GetBondMode(bondName, workerName)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to get bond mode interface %s on worker %s via nmstate", bondName, workerName))
	Expect(bondModeViaNmstate).To(Equal(bondTypeName), "Bond mode is not expected one")
}

func waitForSwitchInterfaceUp(juniperSession *juniper.JunosSession, switchLagName string) {
	Eventually(func() bool {
		isBondInterfaceUp, err := juniper.IsSwitchInterfaceUp(juniperSession, switchLagName)
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to get status of switch LAG interface %s", switchLagName))

		return isBondInterfaceUp
	}, 1*time.Minute, 5*time.Second).Should(BeTrue(), "Bond interface is not Up on the switch")
}
