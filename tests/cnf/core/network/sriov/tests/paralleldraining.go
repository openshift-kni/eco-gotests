package tests

import (
	"fmt"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"github.com/openshift-kni/eco-goinfra/pkg/sriov"

	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netenv"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netinittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netparam"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/sriov/internal/sriovenv"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/sriov/internal/tsparams"
)

const sriovAndResourceNameParallelDrain = "paralleldraining"

var _ = Describe("ParallelDraining", Ordered, Label(tsparams.LabelParallelDrainingTestCases),
	ContinueOnFailure, func() {
		var (
			sriovInterfacesUnderTest []string
			workerNodeList           []*nodes.Builder
			err                      error
			poolConfigName           = "pool1"
		)

		BeforeAll(func() {
			By("Validating SR-IOV interfaces")
			workerNodeList, err = nodes.List(APIClient,
				metav1.ListOptions{LabelSelector: labels.Set(NetConfig.WorkerLabelMap).String()})
			Expect(err).ToNot(HaveOccurred(), "Failed to discover worker nodes")
			Expect(sriovenv.ValidateSriovInterfaces(workerNodeList, 1)).ToNot(HaveOccurred(),
				"Failed to get required SR-IOV interfaces")
			sriovInterfacesUnderTest, err = NetConfig.GetSriovInterfaces(1)
			Expect(err).ToNot(HaveOccurred(), "Failed to retrieve SR-IOV interfaces for testing")

			By("Verifying if parallel draining tests can be executed on given cluster")
			err = netenv.DoesClusterHasEnoughNodes(APIClient, NetConfig, 1, 2)
			Expect(err).ToNot(HaveOccurred(),
				"Cluster doesn't support parallel draining test cases - doesn't have enough nodes")
		})
		BeforeEach(func() {
			By("Configuring SR-IOV")
			createSriovConfigurationParallelDrain(sriovInterfacesUnderTest[0])

			By("Creating test pods and checking connectivity between the them")
			err := sriovenv.CreatePodsAndRunTraffic(workerNodeList[0].Object.Name, workerNodeList[0].Object.Name,
				sriovAndResourceNameParallelDrain, sriovAndResourceNameParallelDrain,
				tsparams.ClientMacAddress, tsparams.ServerMacAddress,
				[]string{tsparams.ClientIPv4IPAddress}, []string{tsparams.ServerIPv4IPAddress})
			Expect(err).ToNot(HaveOccurred(), "Connectivity check between test pods failed")

			By("Adding pods with terminationGracePeriodSeconds on each worker node")
			createPodWithVFOnEachWorker(workerNodeList)
		})

		AfterEach(func() {
			By("Removing SR-IOV configuration")
			err := sriovenv.RemoveSriovConfigurationAndWaitForSriovAndMCPStable()
			Expect(err).ToNot(HaveOccurred(), "Failed to remove SR-IOV configration")

			err = sriov.CleanAllNonDefaultPoolConfigs(APIClient, NetConfig.SriovOperatorNamespace)
			Expect(err).ToNot(HaveOccurred(), "Failed to remove SriovNetworkPoolConfigs")

			By("Cleaning test namespace")
			err = namespace.NewBuilder(APIClient, tsparams.TestNamespaceName).CleanObjects(
				netparam.DefaultTimeout, pod.GetGVR())
			Expect(err).ToNot(HaveOccurred(), "Failed to clean test namespace")
		})

		It("without SriovNetworkPoolConfig", reportxml.ID("68640"), func() {
			By("Removing test configuration to call draining mechanism")
			removeTestConfiguration()

			By("Validating that nodes are drained one by one")
			Eventually(isDrainingRunningAsExpected, time.Minute, tsparams.RetryInterval).WithArguments(1).
				Should(BeTrue(), "draining runs not as expected")

			err = netenv.WaitForSriovStable(APIClient, tsparams.MCOWaitTimeout, NetConfig.SriovOperatorNamespace)
			Expect(err).ToNot(HaveOccurred(), "Failed to wait for stable cluster.")
		})

		It("without maxUnavailable field", reportxml.ID("68661"), func() {
			By("Creating SriovNetworkPoolConfig without maxUnavailable field")
			_, err = sriov.NewPoolConfigBuilder(APIClient, poolConfigName, NetConfig.SriovOperatorNamespace).
				WithNodeSelector(NetConfig.WorkerLabelMap).Create()
			Expect(err).ToNot(HaveOccurred(), "Failed to create SriovNetworkPoolConfig without maxUnavailable field.")

			By("Removing test configuration to call draining mechanism")
			removeTestConfiguration()

			By("Validating that nodes are drained all together")
			Eventually(isDrainingRunningAsExpected, time.Minute, tsparams.RetryInterval).WithArguments(len(workerNodeList)).
				Should(BeTrue(), "draining runs not as expected")

			err = netenv.WaitForSriovStable(APIClient, tsparams.MCOWaitTimeout, NetConfig.SriovOperatorNamespace)
			Expect(err).ToNot(HaveOccurred(), "Failed to wait for stable cluster.")
		})

		It("1 SriovNetworkPoolconfig: maxUnavailable value is 2", reportxml.ID("68662"), func() {
			By("Validating that the cluster has more 2 worker nodes")
			if len(workerNodeList) < 3 {
				Skip(fmt.Sprintf("The cluster has less than 3 workers: %d", len(workerNodeList)))
			}

			By("Creating SriovNetworkPoolConfig with maxUnavailable 2")
			_, err = sriov.NewPoolConfigBuilder(APIClient, poolConfigName, NetConfig.SriovOperatorNamespace).
				WithMaxUnavailable(intstr.FromInt32(2)).
				WithNodeSelector(NetConfig.WorkerLabelMap).Create()
			Expect(err).ToNot(HaveOccurred(), "Failed to create SriovNetworkPoolConfig with maxUnavailable 2.")

			By("Removing test configuration to call draining mechanism")
			removeTestConfiguration()

			By("Validating that nodes are drained by 2")
			Eventually(isDrainingRunningAsExpected, time.Minute, tsparams.RetryInterval).WithArguments(2).
				Should(BeTrue(), "draining runs not as expected")

			err = netenv.WaitForSriovStable(APIClient, tsparams.MCOWaitTimeout, NetConfig.SriovOperatorNamespace)
			Expect(err).ToNot(HaveOccurred(), "Failed to wait for stable cluster.")
		})
	})

func createSriovConfigurationParallelDrain(sriovInterfaceName string) {
	By("Creating SR-IOV policy")

	sriovPolicy := sriov.NewPolicyBuilder(
		APIClient,
		sriovAndResourceNameParallelDrain,
		NetConfig.SriovOperatorNamespace,
		sriovAndResourceNameParallelDrain,
		5,
		[]string{sriovInterfaceName}, NetConfig.WorkerLabelMap)

	err := sriovenv.CreateSriovPolicyAndWaitUntilItsApplied(sriovPolicy, tsparams.MCOWaitTimeout)
	Expect(err).ToNot(HaveOccurred(), "Failed to configure SR-IOV policy")

	By("Creating SR-IOV network")

	_, err = sriov.NewNetworkBuilder(APIClient, sriovAndResourceNameParallelDrain, NetConfig.SriovOperatorNamespace,
		tsparams.TestNamespaceName, sriovAndResourceNameParallelDrain).WithStaticIpam().WithMacAddressSupport().
		WithIPAddressSupport().Create()
	Expect(err).ToNot(HaveOccurred(), "Failed to create SR-IOV network")
}

func removeTestConfiguration() {
	err := sriovenv.RemoveAllSriovNetworks()
	Expect(err).ToNot(HaveOccurred(), "Failed to clean all SR-IOV Networks")
	err = sriov.CleanAllNetworkNodePolicies(APIClient, NetConfig.SriovOperatorNamespace)
	Expect(err).ToNot(HaveOccurred(), "Failed to clean all SR-IOV policies")
}

func isDrainingRunningAsExpected(expectedConcurrentDrains int) bool {
	sriovNodeStateList, err := sriov.ListNetworkNodeState(APIClient, NetConfig.SriovOperatorNamespace)
	Expect(err).ToNot(HaveOccurred(), "Failed to collect all SriovNetworkNodeStates")
	Expect(len(sriovNodeStateList)).ToNot(Equal(0), "SriovNetworkNodeStates list is empty")

	var inProgressWithDrainingCompleteCount int

	for _, sriovNodeState := range sriovNodeStateList {
		// Check if syncStatus is "InProgress" and CurrentSyncState is "DrainComplete"
		if sriovNodeState.Objects.Status.SyncStatus == "InProgress" &&
			sriovNodeState.Objects.Annotations["sriovnetwork.openshift.io/current-state"] == "Draining" {
			inProgressWithDrainingCompleteCount++
		}
	}

	return inProgressWithDrainingCompleteCount == expectedConcurrentDrains
}

func createPodWithVFOnEachWorker(workerList []*nodes.Builder) {
	for i, worker := range workerList {
		// 192.168.0.1 and 192.168.0.2 addresses are busy by client and server pods
		ipaddress := "192.168.0." + strconv.Itoa(i+3) + "/24"
		secNetwork := pod.StaticIPAnnotation(sriovAndResourceNameParallelDrain, []string{ipaddress})
		_, err := pod.NewBuilder(
			APIClient, "testpod"+worker.Object.Name, tsparams.TestNamespaceName, NetConfig.CnfNetTestContainer).
			DefineOnNode(worker.Object.Name).WithSecondaryNetwork(secNetwork).
			WithTerminationGracePeriodSeconds(5).
			CreateAndWaitUntilRunning(netparam.DefaultTimeout)
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to create a pod %s", "testpod-"+worker.Object.Name))
	}
}
