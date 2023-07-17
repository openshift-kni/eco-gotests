package tests

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"

	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/nmstate"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-goinfra/pkg/sriov"

	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/cmd"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netinittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netparam"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/sriov/internal/nmstateenv"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/sriov/internal/sriovenv"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/sriov/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
)

var _ = Describe("ExternallyCreated", Ordered, Label(tsparams.LabelExternallyCreatedTestCases),
	ContinueOnFailure, func() {
		const (
			configureNMStatePolicyName = "configurevfs"
			removeNMStatePolicyName    = "removevfs"
			sriovAndResourceName       = "extcreated"
		)
		var (
			sriovInterfacesUnderTest []string
		)
		BeforeAll(func() {
			By("Creating a new instance of NMstate instance")
			err := nmstateenv.CreateNewNMStateAndWaitUntilItsRunning(tsparams.DefaultTimeout)
			Expect(err).ToNot(HaveOccurred(), "Failed to create NMState instance")

			By("Validating SR-IOV interfaces")
			workerNodeList = nodes.NewBuilder(APIClient, NetConfig.WorkerLabelMap)
			Expect(workerNodeList.Discover()).ToNot(HaveOccurred(), "Failed to discover worker nodes")
			Expect(sriovenv.ValidateSriovInterfaces(workerNodeList, 2)).ToNot(HaveOccurred(),
				"Failed to get required SR-IOV interfaces")
			sriovInterfacesUnderTest, err = NetConfig.GetSriovInterfaces(2)
			Expect(err).ToNot(HaveOccurred(), "Failed to retrieve SR-IOV interfaces for testing")

			By("Creating SR-IOV VFs via NMState")
			err = nmstateenv.ConfigureVFsAndWaitUntilItsConfigured(
				configureNMStatePolicyName,
				sriovInterfacesUnderTest[0],
				NetConfig.WorkerLabelMap,
				5,
				tsparams.DefaultTimeout)
			Expect(err).ToNot(HaveOccurred(), "Failed to create VFs via NMState")

			err = sriovenv.WaitUntilVfsCreated(workerNodeList, sriovInterfacesUnderTest[0], 5, tsparams.DefaultTimeout)
			Expect(err).ToNot(HaveOccurred(), "Expected number of VFs are not created")

			By("Configure SR-IOV with flag ExternallyCreated true")
			createSriovConfiguration(sriovAndResourceName, sriovInterfacesUnderTest[0], true)
		})

		AfterAll(func() {
			By("Remove all SR-IOV networks")
			sriovNs, err := namespace.Pull(APIClient, NetConfig.SriovOperatorNamespace)
			Expect(err).ToNot(HaveOccurred(), "Failed to pull SR-IOV operator namespace")
			err = sriovNs.CleanObjects(
				tsparams.DefaultTimeout,
				sriov.GetSriovNetworksGVR())
			Expect(err).ToNot(HaveOccurred(), "Failed to remove SR-IOV networks from SR-IOV operator namespace")

			By("Remove all SR-IOV policies")
			err = sriovenv.RemoveAllPoliciesAndWaitForSriovAndMCPStable()
			Expect(err).ToNot(HaveOccurred(), "Failed to remove all SR-IOV policies")

			By("Verifying that VFs still exist")
			err = sriovenv.WaitUntilVfsCreated(workerNodeList, sriovInterfacesUnderTest[0], 5, tsparams.DefaultTimeout)
			Expect(err).ToNot(HaveOccurred(), "Unexpected amount of VF")

			err = nmstateenv.AreVFsCreated(workerNodeList.Objects[0].Object.Name, sriovInterfacesUnderTest[0], 5)
			Expect(err).ToNot(HaveOccurred(), "VFs were removed during the test")

			By("Removing SR-IOV VFs via NMState")
			err = nmstateenv.ConfigureVFsAndWaitUntilItsConfigured(
				removeNMStatePolicyName,
				sriovInterfacesUnderTest[0],
				NetConfig.WorkerLabelMap,
				0,
				tsparams.DefaultTimeout)
			Expect(err).ToNot(HaveOccurred(), "Failed to remove VFs via NMState")

			By("Removing NMState policies")
			err = nmstate.CleanAllNMStatePolicies(APIClient)
			Expect(err).ToNot(HaveOccurred(), "Failed to remove all NMState policies")
		})

		AfterEach(func() {
			By("Cleaning test namespace")
			err := namespace.NewBuilder(APIClient, tsparams.TestNamespaceName).CleanObjects(
				tsparams.DefaultTimeout, pod.GetGVR())

			Expect(err).ToNot(HaveOccurred(), "Failed to clean test namespace")
		})

		DescribeTable("SR-IOV: ExternallyCreated: Verifying connectivity with different IP protocols", polarion.ID("63527"),
			func(ipStack string) {
				By("Defining test parameters")
				clientIPs, serverIPs, err := defineIterationParams(ipStack)
				Expect(err).ToNot(HaveOccurred(), "Failed to define test parameters")

				By("Creating test pods")
				clientPod, _ := createAndWaitTestPods(workerNodeList.Objects[0].Object.Name, workerNodeList.Objects[1].Object.Name,
					sriovAndResourceName, sriovAndResourceName, "", "", clientIPs, serverIPs)

				By("Checking connectivity between test pods")
				err = cmd.ICMPConnectivityCheck(clientPod, serverIPs)
				Expect(err).ToNot(HaveOccurred(), "Connectivity check failed")
			},

			Entry("", netparam.IPV4Family, polarion.SetProperty("IPStack", netparam.IPV4Family)),
			Entry("", netparam.IPV6Family, polarion.SetProperty("IPStack", netparam.IPV6Family)),
			Entry("", netparam.DualIPFamily, polarion.SetProperty("IPStack", netparam.DualIPFamily)),
		)

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
		return []string{"192.168.0.1/24"}, []string{"192.168.0.2/24"}, nil
	case netparam.IPV6Family:
		return []string{"2001::1/64"}, []string{"2001::2/64"}, nil
	case netparam.DualIPFamily:
		return []string{"192.168.0.1/24", "2001::1/64"}, []string{"192.168.0.2/24", "2001::2/64"}, nil
	}

	return nil, nil, fmt.Errorf(fmt.Sprintf(
		"ipStack parameter %s is invalid; allowed values are %s, %s, %s ",
		ipFamily, netparam.IPV4Family, netparam.IPV6Family, netparam.DualIPFamily))
}

// createAndWaitTestPods creates test pods and waits until they are in the ready state.
func createAndWaitTestPods(
	clientNodeName string,
	serverNodeName string,
	sriovResNameClient string,
	sriovResNameServer string,
	clientMac string,
	serverMac string,
	clientIPs []string,
	serverIPs []string) (client *pod.Builder, server *pod.Builder) {
	By("Creating client test pod")

	clientPod, err := createAndWaitTestPodWithSecondaryNetwork("client", clientNodeName,
		sriovResNameClient, clientMac, clientIPs)
	Expect(err).ToNot(HaveOccurred(), "Failed to create client pod")

	By("Creating server test pod")

	serverPod, err := createAndWaitTestPodWithSecondaryNetwork("server", serverNodeName,
		sriovResNameServer, serverMac, serverIPs)
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
		WithSecondaryNetwork(secNetwork).CreateAndWaitUntilRunning(tsparams.DefaultTimeout)

	return testPod, err
}
