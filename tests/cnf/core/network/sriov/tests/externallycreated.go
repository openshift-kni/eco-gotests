package tests

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/nmstate"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-goinfra/pkg/sriov"

	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/cmd"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netinittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netnmstate"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netparam"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/sriov/internal/sriovenv"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/sriov/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
)

const sriovAndResourceNameExCreatedTrue = "extcreated"

var _ = Describe("ExternallyCreated", Ordered, Label(tsparams.LabelExternallyCreatedTestCases),
	ContinueOnFailure, func() {
		const (
			configureNMStatePolicyName = "configurevfs"
			removeNMStatePolicyName    = "removevfs"
		)
		var (
			sriovInterfacesUnderTest []string
		)
		BeforeAll(func() {
			By("Creating a new instance of NMstate instance")
			err := netnmstate.CreateNewNMStateAndWaitUntilItsRunning(netparam.DefaultTimeout)
			Expect(err).ToNot(HaveOccurred(), "Failed to create NMState instance")

			By("Validating SR-IOV interfaces")
			workerNodeList = nodes.NewBuilder(APIClient, NetConfig.WorkerLabelMap)
			Expect(workerNodeList.Discover()).ToNot(HaveOccurred(), "Failed to discover worker nodes")
			Expect(sriovenv.ValidateSriovInterfaces(workerNodeList, 2)).ToNot(HaveOccurred(),
				"Failed to get required SR-IOV interfaces")
			sriovInterfacesUnderTest, err = NetConfig.GetSriovInterfaces(2)
			Expect(err).ToNot(HaveOccurred(), "Failed to retrieve SR-IOV interfaces for testing")

			By("Creating SR-IOV VFs via NMState")
			err = netnmstate.ConfigureVFsAndWaitUntilItsConfigured(
				configureNMStatePolicyName,
				sriovInterfacesUnderTest[0],
				NetConfig.WorkerLabelMap,
				5,
				netparam.DefaultTimeout)
			Expect(err).ToNot(HaveOccurred(), "Failed to create VFs via NMState")

			err = sriovenv.WaitUntilVfsCreated(workerNodeList, sriovInterfacesUnderTest[0], 5, netparam.DefaultTimeout)
			Expect(err).ToNot(HaveOccurred(), "Expected number of VFs are not created")

			By("Configure SR-IOV with flag ExternallyCreated true")
			createSriovConfiguration(sriovAndResourceNameExCreatedTrue, sriovInterfacesUnderTest[0], true)
		})

		AfterAll(func() {
			By("Remove all SR-IOV networks")
			sriovNs, err := namespace.Pull(APIClient, NetConfig.SriovOperatorNamespace)
			Expect(err).ToNot(HaveOccurred(), "Failed to pull SR-IOV operator namespace")
			err = sriovNs.CleanObjects(
				netparam.DefaultTimeout,
				sriov.GetSriovNetworksGVR())
			Expect(err).ToNot(HaveOccurred(), "Failed to remove SR-IOV networks from SR-IOV operator namespace")

			By("Remove all SR-IOV policies")
			err = sriovenv.RemoveAllPoliciesAndWaitForSriovAndMCPStable()
			Expect(err).ToNot(HaveOccurred(), "Failed to remove all SR-IOV policies")

			By("Verifying that VFs still exist")
			err = sriovenv.WaitUntilVfsCreated(workerNodeList, sriovInterfacesUnderTest[0], 5, netparam.DefaultTimeout)
			Expect(err).ToNot(HaveOccurred(), "Unexpected amount of VF")

			err = netnmstate.AreVFsCreated(workerNodeList.Objects[0].Object.Name, sriovInterfacesUnderTest[0], 5)
			Expect(err).ToNot(HaveOccurred(), "VFs were removed during the test")

			By("Removing SR-IOV VFs via NMState")
			err = netnmstate.ConfigureVFsAndWaitUntilItsConfigured(
				removeNMStatePolicyName,
				sriovInterfacesUnderTest[0],
				NetConfig.WorkerLabelMap,
				0,
				netparam.DefaultTimeout)
			Expect(err).ToNot(HaveOccurred(), "Failed to remove VFs via NMState")

			By("Removing NMState policies")
			err = nmstate.CleanAllNMStatePolicies(APIClient)
			Expect(err).ToNot(HaveOccurred(), "Failed to remove all NMState policies")
		})

		AfterEach(func() {
			By("Cleaning test namespace")
			err := namespace.NewBuilder(APIClient, tsparams.TestNamespaceName).CleanObjects(
				netparam.DefaultTimeout, pod.GetGVR())

			Expect(err).ToNot(HaveOccurred(), "Failed to clean test namespace")
		})

		DescribeTable("Verifying connectivity with different IP protocols", polarion.ID("63527"),
			func(ipStack string) {
				By("Defining test parameters")
				clientIPs, serverIPs, err := defineIterationParams(ipStack)
				Expect(err).ToNot(HaveOccurred(), "Failed to define test parameters")

				By("Creating test pods and checking connectivity")
				createPodsAndRunTraffic(workerNodeList.Objects[0].Object.Name, workerNodeList.Objects[1].Object.Name,
					"", "", clientIPs, serverIPs)
			},

			Entry("", netparam.IPV4Family, polarion.SetProperty("IPStack", netparam.IPV4Family)),
			Entry("", netparam.IPV6Family, polarion.SetProperty("IPStack", netparam.IPV6Family)),
			Entry("", netparam.DualIPFamily, polarion.SetProperty("IPStack", netparam.DualIPFamily)),
		)

		It("Recreate VFs when SR-IOV policy is applied", polarion.ID("63533"), func() {
			By("Creating test pods and checking connectivity")
			createPodsAndRunTraffic(workerNodeList.Objects[0].Object.Name, workerNodeList.Objects[0].Object.Name,
				tsparams.ClientMacAddress, tsparams.ServerMacAddress,
				[]string{tsparams.ClientIPv4IPAddress}, []string{tsparams.ServerIPv4IPAddress})

			By("Removing created SR-IOV VFs via NMState")
			err := netnmstate.ConfigureVFsAndWaitUntilItsConfigured(removeNMStatePolicyName,
				sriovInterfacesUnderTest[0], NetConfig.WorkerLabelMap, 0, netparam.DefaultTimeout)
			Expect(err).ToNot(HaveOccurred(), "Failed to remove VFs via NMState")

			By("Removing NMState policies")
			err = nmstate.CleanAllNMStatePolicies(APIClient)
			Expect(err).ToNot(HaveOccurred(), "Failed to remove all NMState policies")

			By("Removing all test pods")
			err = namespace.NewBuilder(APIClient, tsparams.TestNamespaceName).CleanObjects(
				netparam.DefaultTimeout, pod.GetGVR())
			Expect(err).ToNot(HaveOccurred(), "Failed to clean all test pods")

			By("Creating SR-IOV VFs again via NMState")
			err = netnmstate.ConfigureVFsAndWaitUntilItsConfigured(configureNMStatePolicyName,
				sriovInterfacesUnderTest[0], NetConfig.WorkerLabelMap, 5, netparam.DefaultTimeout)
			Expect(err).ToNot(HaveOccurred(), "Failed to recreate VFs via NMState")

			err = sriovenv.WaitUntilVfsCreated(workerNodeList, sriovInterfacesUnderTest[0], 5, netparam.DefaultTimeout)
			Expect(err).ToNot(HaveOccurred(), "Expected number of VFs are not created")

			By("Re-create test pods and verify connectivity after recreating the VFs")
			createPodsAndRunTraffic(workerNodeList.Objects[0].Object.Name, workerNodeList.Objects[0].Object.Name,
				tsparams.ClientMacAddress, tsparams.ServerMacAddress,
				[]string{tsparams.ClientIPv4IPAddress}, []string{tsparams.ServerIPv4IPAddress})
		})

		It("SR-IOV network with options", polarion.ID("63534"), func() {
			By("Collecting default MaxTxRate and Vlan values")
			defaultMaxTxRate, defaultVlanID := getVlanIDAndMaxTxRateForVf(workerNodeList.Objects[0].Object.Name,
				sriovInterfacesUnderTest[0])

			By("Updating Vlan and MaxTxRate configurations in the SriovNetwork")
			newMaxTxRate := defaultMaxTxRate + 1
			newVlanID := defaultVlanID + 1
			sriovNetwork, err := sriov.PullNetwork(APIClient, sriovAndResourceNameExCreatedTrue,
				NetConfig.SriovOperatorNamespace)
			Expect(err).ToNot(HaveOccurred(), "Failed to pull SR-IOV network object")
			_, err = sriovNetwork.WithMaxTxRate(uint16(newMaxTxRate)).WithVLAN(uint16(newVlanID)).Update(false)
			Expect(err).ToNot(HaveOccurred(), "Failed to update SR-IOV network with new configuration")

			By("Creating test pods and checking connectivity")
			createPodsAndRunTraffic(workerNodeList.Objects[0].Object.Name, workerNodeList.Objects[0].Object.Name,
				tsparams.ClientMacAddress, tsparams.ServerMacAddress,
				[]string{tsparams.ClientIPv4IPAddress}, []string{tsparams.ServerIPv4IPAddress})

			By("Checking that VF configured with new VLAN and MaxTxRate values")
			Eventually(func() []int {
				currentmaxTxRate, currentVlanID := getVlanIDAndMaxTxRateForVf(workerNodeList.Objects[0].Object.Name,
					sriovInterfacesUnderTest[0])

				return []int{currentmaxTxRate, currentVlanID}
			}, time.Minute, tsparams.DefaultRetryInterval).Should(Equal([]int{newMaxTxRate, newVlanID}),
				"MaxTxRate and VlanId have been not configured properly")

			By("Removing all test pods")
			err = namespace.NewBuilder(APIClient, tsparams.TestNamespaceName).CleanObjects(
				netparam.DefaultTimeout, pod.GetGVR())
			Expect(err).ToNot(HaveOccurred(), "Failed to clean all test pods")

			By("Checking that VF has initial configuration")

			Eventually(func() []int {
				currentmaxTxRate, currentVlanID := getVlanIDAndMaxTxRateForVf(workerNodeList.Objects[0].Object.Name,
					sriovInterfacesUnderTest[0])

				return []int{currentmaxTxRate, currentVlanID}
			}, netparam.DefaultTimeout, tsparams.DefaultRetryInterval).
				Should(Equal([]int{defaultMaxTxRate, defaultVlanID}),
					"MaxTxRate and VlanId configuration have not been reverted to the initial one")

			By("Remove all SR-IOV networks")
			sriovNs, err := namespace.Pull(APIClient, NetConfig.SriovOperatorNamespace)
			Expect(err).ToNot(HaveOccurred(), "Failed to pull SR-IOV operator namespace")
			err = sriovNs.CleanObjects(netparam.DefaultTimeout, sriov.GetSriovNetworksGVR())
			Expect(err).ToNot(HaveOccurred(), "Failed to remove object's from SR-IOV operator namespace")

			By("Remove all SR-IOV policies")
			err = sriovenv.RemoveAllPoliciesAndWaitForSriovAndMCPStable()
			Expect(err).ToNot(HaveOccurred(), "Failed to remove all SR-IOV policies")

			By("Checking that VF has initial configuration")
			Eventually(func() []int {
				currentmaxTxRate, currentVlanID := getVlanIDAndMaxTxRateForVf(workerNodeList.Objects[0].Object.Name,
					sriovInterfacesUnderTest[0])

				return []int{currentmaxTxRate, currentVlanID}
			}, time.Minute, tsparams.DefaultRetryInterval).Should(And(Equal([]int{defaultMaxTxRate, defaultVlanID})),
				"MaxTxRate and VlanId configurations have not been reverted to the initial one")
		})
	})

func createSriovConfiguration(sriovAndResName, sriovInterfaceName string, externallyCreated bool) {
	By("Creating SR-IOV policy with flag ExternallyCreated true")

	sriovPolicy := sriov.NewPolicyBuilder(APIClient, sriovAndResName, NetConfig.SriovOperatorNamespace, sriovAndResName,
		5, []string{sriovInterfaceName + "#0-1"}, NetConfig.WorkerLabelMap).WithExternallyCreated(externallyCreated)

	err := sriovenv.CreateSriovPolicyAndWaitUntilItsApplied(sriovPolicy, tsparams.SriovStableTimeout)
	Expect(err).ToNot(HaveOccurred(), "Failed to configure SR-IOV policy")

	By("Creating SR-IOV network")

	_, err = sriov.NewNetworkBuilder(APIClient, sriovAndResName, NetConfig.SriovOperatorNamespace,
		tsparams.TestNamespaceName, sriovAndResName).WithStaticIpam().WithMacAddressSupport().WithIPAddressSupport().Create()
	Expect(err).ToNot(HaveOccurred(), "Failed to create SR-IOV network")
}

// defineIterationParams defines ip settings for iteration based on ipFamily parameter.
func defineIterationParams(ipFamily string) (clientIPs, serverIPs []string, err error) {
	switch ipFamily {
	case netparam.IPV4Family:
		return []string{tsparams.ClientIPv4IPAddress}, []string{tsparams.ServerIPv4IPAddress}, nil
	case netparam.IPV6Family:
		return []string{tsparams.ClientIPv6IPAddress}, []string{tsparams.ServerIPv6IPAddress}, nil
	case netparam.DualIPFamily:
		return []string{tsparams.ClientIPv4IPAddress, tsparams.ClientIPv6IPAddress},
			[]string{tsparams.ServerIPv4IPAddress, tsparams.ServerIPv6IPAddress}, nil
	}

	return nil, nil, fmt.Errorf(fmt.Sprintf(
		"ipStack parameter %s is invalid; allowed values are %s, %s, %s ",
		ipFamily, netparam.IPV4Family, netparam.IPV6Family, netparam.DualIPFamily))
}

// createAndWaitTestPods creates test pods and waits until they are in the ready state.
func createAndWaitTestPods(
	clientNodeName string,
	serverNodeName string,
	clientMac string,
	serverMac string,
	clientIPs []string,
	serverIPs []string) (client *pod.Builder, server *pod.Builder) {
	By("Creating client test pod")

	clientPod, err := createAndWaitTestPodWithSecondaryNetwork("client", clientNodeName,
		sriovAndResourceNameExCreatedTrue, clientMac, clientIPs)
	Expect(err).ToNot(HaveOccurred(), "Failed to create client pod")

	By("Creating server test pod")

	serverPod, err := createAndWaitTestPodWithSecondaryNetwork("server", serverNodeName,
		sriovAndResourceNameExCreatedTrue, serverMac, serverIPs)
	Expect(err).ToNot(HaveOccurred(), "Failed to create server pod")

	return clientPod, serverPod
}

// createAndWaitTestPodWithSecondaryNetwork creates test pod with secondary network
// and waits until it is in the ready state.
func createAndWaitTestPodWithSecondaryNetwork(
	podName string,
	testNodeName string,
	sriovResNameTest string,
	testMac string,
	testIPs []string) (*pod.Builder, error) {
	By("Creating test pod")

	secNetwork := pod.StaticIPAnnotationWithMacAddress(sriovResNameTest, testIPs, testMac)
	testPod, err := pod.NewBuilder(APIClient, podName, tsparams.TestNamespaceName, NetConfig.CnfNetTestContainer).
		DefineOnNode(testNodeName).WithPrivilegedFlag().
		WithSecondaryNetwork(secNetwork).CreateAndWaitUntilRunning(netparam.DefaultTimeout)

	return testPod, err
}

// createPodsAndRunTraffic creates test pods and verifies connectivity between them.
func createPodsAndRunTraffic(
	clientNodeName string,
	serverNodeName string,
	clientMac string,
	serverMac string,
	clientIPs []string,
	serverIPs []string) {
	By("Creating test pods")

	clientPod, _ := createAndWaitTestPods(clientNodeName, serverNodeName, clientMac, serverMac, clientIPs, serverIPs)

	By("Checking connectivity between test pods")

	err := cmd.ICMPConnectivityCheck(clientPod, serverIPs)
	Expect(err).ToNot(HaveOccurred(), "Connectivity check failed")
}

func getVlanIDAndMaxTxRateForVf(nodeName, sriovInterfaceName string) (maxTxRate, vlanID int) {
	nmstateState, err := nmstate.PullNodeNetworkState(APIClient, nodeName)
	Expect(err).ToNot(HaveOccurred(), "Failed to discover NMState network state")
	sriovVfs, err := nmstateState.GetSriovVfs(sriovInterfaceName)
	Expect(err).ToNot(HaveOccurred(), "Failed to get all SR-IOV VFs")

	return sriovVfs[0].MaxTxRate, sriovVfs[0].VlanID
}
