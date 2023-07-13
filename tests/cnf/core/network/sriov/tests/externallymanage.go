package tests

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"

	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/nmstate"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-goinfra/pkg/sriov"

	. "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netinittools"
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

			err = sriovenv.WaitWhenVfsCreated(workerNodeList, sriovInterfacesUnderTest[0], 5, tsparams.DefaultTimeout)
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

		It("SR-IOV: ExternallyCreated: Verifying connectivity with different IP protocols", polarion.ID("63527"), func() {
			Skip("TODO")
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
