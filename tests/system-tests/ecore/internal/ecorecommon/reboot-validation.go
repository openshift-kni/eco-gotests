package ecorecommon

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"

	bmclib "github.com/bmc-toolbox/bmclib/v2"
	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/openshift-kni/eco-gotests/tests/system-tests/ecore/internal/ecoreinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/ecore/internal/ecoreparams"

	"github.com/openshift-kni/eco-goinfra/pkg/clusteroperator"
	"github.com/openshift-kni/eco-goinfra/pkg/deployment"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

// WaitAllNodesAreReady waits for all the nodes in the cluster to report Ready state.
func WaitAllNodesAreReady(ctx SpecContext) {
	By("Checking all nodes are Ready")

	Eventually(func(ctx SpecContext) bool {
		allNodes, err := nodes.List(APIClient, metav1.ListOptions{})
		if err != nil {
			glog.V(ecoreparams.ECoreLogLevel).Infof("Failed to list all nodes: %s", err)

			return false
		}

		for _, _node := range allNodes {
			glog.V(ecoreparams.ECoreLogLevel).Infof("Processing node %q", _node.Definition.Name)

			for _, condition := range _node.Object.Status.Conditions {
				if condition.Type == ecoreparams.ConditionTypeReadyString {
					if condition.Status != ecoreparams.ConstantTrueString {
						glog.V(ecoreparams.ECoreLogLevel).Infof("Node %q is notReady", _node.Definition.Name)
						glog.V(ecoreparams.ECoreLogLevel).Infof("  Reason: %s", condition.Reason)

						return false
					}
				}
			}
		}

		return true
	}).WithTimeout(25*time.Minute).WithPolling(15*time.Second).WithContext(ctx).Should(BeTrue(),
		"Some nodes are notReady")
}

// VerifyUngracefulReboot performs ungraceful reboot of the cluster
//
//nolint:funlen
func VerifyUngracefulReboot(ctx SpecContext) {
	glog.V(ecoreparams.ECoreLogLevel).Infof("\t*** VerifyUngracefulReboot started ***")

	if len(ECoreConfig.NodesCredentialsMap) == 0 {
		glog.V(ecoreparams.ECoreLogLevel).Infof("BMC Details not specified")
		Skip("BMC Details not specified. Skipping...")
	}

	clientOpts := []bmclib.Option{}

	glog.V(ecoreparams.ECoreLogLevel).Infof(
		fmt.Sprintf("BMC options %v", clientOpts))

	glog.V(ecoreparams.ECoreLogLevel).Infof(
		fmt.Sprintf("NodesCredentialsMap:\n\t%#v", ECoreConfig.NodesCredentialsMap))

	var bmcMap = make(map[string]*bmclib.Client)

	for node, auth := range ECoreConfig.NodesCredentialsMap {
		glog.V(ecoreparams.ECoreLogLevel).Infof(
			fmt.Sprintf("Creating BMC client for node %s", node))
		glog.V(ecoreparams.ECoreLogLevel).Infof(
			fmt.Sprintf("BMC Auth %#v", auth))

		bmcClient := bmclib.NewClient(auth.BMCAddress, auth.Username, auth.Password, clientOpts...)
		bmcMap[node] = bmcClient
	}

	var waitGroup sync.WaitGroup

	for node, client := range bmcMap {
		waitGroup.Add(1)

		go func(wg *sync.WaitGroup, nodeName string, client *bmclib.Client) {
			glog.V(ecoreparams.ECoreLogLevel).Infof(
				fmt.Sprintf("Starting go routine for %s", nodeName))

			defer GinkgoRecover()
			defer wg.Done()

			glog.V(ecoreparams.ECoreLogLevel).Infof(
				fmt.Sprintf("[%s] Setting timeout for context", nodeName))

			bmcCtx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)

			defer cancel()

			glog.V(ecoreparams.ECoreLogLevel).Infof(
				fmt.Sprintf("[%s] Starting BMC session", nodeName))

			err := client.Open(bmcCtx)

			Expect(err).ToNot(HaveOccurred(),
				fmt.Sprintf("Failed to login to %s", nodeName))

			defer client.Close(bmcCtx)

			By(fmt.Sprintf("Querying power state on %s", nodeName))

			glog.V(ecoreparams.ECoreLogLevel).Infof(
				fmt.Sprintf("Checking power state on %s", nodeName))

			state, err := client.GetPowerState(bmcCtx)
			msgRegex := `(?i)chassis power is on|(?i)^on$`

			glog.V(ecoreparams.ECoreLogLevel).Infof(
				fmt.Sprintf("Power state on %s -> %s", nodeName, state))

			Expect(err).ToNot(HaveOccurred(),
				fmt.Sprintf("Failed to login to %s", nodeName))
			Expect(strings.TrimSpace(state)).To(MatchRegexp(msgRegex),
				fmt.Sprintf("Unexpected power state %s", state))

			err = wait.PollUntilContextTimeout(ctx, 5*time.Second, 5*time.Minute, true,
				func(ctx context.Context) (bool, error) {
					if _, err := client.SetPowerState(bmcCtx, "cycle"); err != nil {
						glog.V(ecoreparams.ECoreLogLevel).Infof(
							fmt.Sprintf("Failed to power cycle %s -> %v", nodeName, err))

						return false, err
					}

					glog.V(ecoreparams.ECoreLogLevel).Infof(
						fmt.Sprintf("Successfully powered cycle %s", nodeName))

					return true, nil
				})

			Expect(err).ToNot(HaveOccurred(),
				fmt.Sprintf("Failed to reboot node %s", nodeName))
		}(&waitGroup, node, client)
	}

	By("Wait for all reboots to finish")

	waitGroup.Wait()
	glog.V(ecoreparams.ECoreLogLevel).Infof("Finished waiting for go routines to finish")
	time.Sleep(1 * time.Minute)

	WaitAllNodesAreReady(ctx)
}

// WaitAllDeploymentsAreAvailable wait for all deployments in all namespaces to be Available.
func WaitAllDeploymentsAreAvailable(ctx SpecContext) {
	By("Checking all deployments")

	Eventually(func() bool {
		allDeployments, err := deployment.ListInAllNamespaces(APIClient, metav1.ListOptions{})

		if err != nil {
			glog.V(ecoreparams.ECoreLogLevel).Infof("Failed to list all deployments: %s", err)

			return false
		}

		glog.V(ecoreparams.ECoreLogLevel).Infof(
			fmt.Sprintf("Found %d deployments", len(allDeployments)))

		var nonAvailableDeployments []*deployment.Builder

		for _, deploy := range allDeployments {
			glog.V(ecoreparams.ECoreLogLevel).Infof(
				"Processing deployment %q in %q namespace", deploy.Definition.Name, deploy.Definition.Namespace)

			for _, condition := range deploy.Object.Status.Conditions {
				if condition.Type == "Available" {
					if condition.Status != ecoreparams.ConstantTrueString {
						glog.V(ecoreparams.ECoreLogLevel).Infof(
							"Deployment %q in %q namespace is NotAvailable", deploy.Definition.Name, deploy.Definition.Namespace)
						glog.V(ecoreparams.ECoreLogLevel).Infof("\tReason: %s", condition.Reason)
						glog.V(ecoreparams.ECoreLogLevel).Infof("\tMessage: %s", condition.Message)
						nonAvailableDeployments = append(nonAvailableDeployments, deploy)
					}
				}
			}
		}

		return len(nonAvailableDeployments) == 0
	}).WithTimeout(25*time.Minute).WithPolling(15*time.Second).WithContext(ctx).Should(BeTrue(),
		"There are non-available deployments") // end Eventually
}

// VerifySoftReboot performs graceful reboot of a cluster with cordoning and draining of individual nodes.
//
//nolint:gocognit,funlen
func VerifySoftReboot(ctx SpecContext) {
	glog.V(ecoreparams.ECoreLogLevel).Infof("\t*** Starting Soft Reboot Test Suite ***")

	if len(ECoreConfig.NodesCredentialsMap) == 0 {
		glog.V(ecoreparams.ECoreLogLevel).Infof("BMC Details not specified")
		Skip("BMC Details not specified. Skipping...")
	}

	By("Getting list of all nodes")

	allNodes, err := nodes.List(APIClient, metav1.ListOptions{})

	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Error listing pods in the cluster: %v", err))
	Expect(len(allNodes)).ToNot(Equal(0), "0 nodes found in the cluster")

	for _, _node := range allNodes {
		glog.V(ecoreparams.ECoreLogLevel).Infof("Processing node %q", _node.Definition.Name)

		glog.V(ecoreparams.ECoreLogLevel).Infof("Cordoning node %q", _node.Definition.Name)
		err := _node.Cordon()
		Expect(err).ToNot(HaveOccurred(),
			fmt.Sprintf("Failed to cordon %q due to %v", _node.Definition.Name, err))
		time.Sleep(5 * time.Second)

		glog.V(ecoreparams.ECoreLogLevel).Infof("Draining node %q", _node.Definition.Name)
		err = _node.Drain()
		Expect(err).ToNot(HaveOccurred(),
			fmt.Sprintf("Failed to drain %q due to %v", _node.Definition.Name, err))

		clientOpts := []bmclib.Option{}

		glog.V(ecoreparams.ECoreLogLevel).Infof(
			fmt.Sprintf("BMC options %v", clientOpts))

		glog.V(ecoreparams.ECoreLogLevel).Infof(
			fmt.Sprintf("NodesCredentialsMap:\n\t%#v", ECoreConfig.NodesCredentialsMap))

		var bmcClient *bmclib.Client

		glog.V(ecoreparams.ECoreLogLevel).Infof(
			fmt.Sprintf("Creating BMC client for node %s", _node.Definition.Name))

		if auth, ok := ECoreConfig.NodesCredentialsMap[_node.Definition.Name]; !ok {
			glog.V(ecoreparams.ECoreLogLevel).Infof(
				fmt.Sprintf("BMC Details for %q not found", _node.Definition.Name))
			Fail(fmt.Sprintf("BMC Details for %q not found", _node.Definition.Name))
		} else {
			bmcClient = bmclib.NewClient(auth.BMCAddress, auth.Username, auth.Password, clientOpts...)
		}

		glog.V(ecoreparams.ECoreLogLevel).Infof(
			fmt.Sprintf("[%s] Setting timeout for context", _node.Definition.Name))

		bmcCtx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)

		defer cancel()

		glog.V(ecoreparams.ECoreLogLevel).Infof(
			fmt.Sprintf("[%s] Starting BMC session", _node.Definition.Name))

		err = bmcClient.Open(bmcCtx)

		Expect(err).ToNot(HaveOccurred(),
			fmt.Sprintf("Failed to login to %s", _node.Definition.Name))

		defer bmcClient.Close(bmcCtx)

		err = wait.PollUntilContextTimeout(ctx, 5*time.Second, 5*time.Minute, true,
			func(ctx context.Context) (bool, error) {
				if _, err := bmcClient.SetPowerState(bmcCtx, "cycle"); err != nil {
					glog.V(ecoreparams.ECoreLogLevel).Infof(
						fmt.Sprintf("Failed to power cycle %s -> %v", _node.Definition.Name, err))

					return false, err
				}

				glog.V(ecoreparams.ECoreLogLevel).Infof(
					fmt.Sprintf("Successfully powered cycle %s", _node.Definition.Name))

				return true, nil
			})

		Expect(err).ToNot(HaveOccurred(),
			fmt.Sprintf("Failed to reboot node %s", _node.Definition.Name))

		By(fmt.Sprintf("Checking node %q got into NotReady", _node.Definition.Name))

		Eventually(func(ctx SpecContext) bool {
			currentNode, err := nodes.Pull(APIClient, _node.Definition.Name)
			if err != nil {
				glog.V(ecoreparams.ECoreLogLevel).Infof("Failed to pull node: %v", err)

				return false
			}

			for _, condition := range currentNode.Object.Status.Conditions {
				if condition.Type == ecoreparams.ConditionTypeReadyString {
					if condition.Status != ecoreparams.ConstantTrueString {
						glog.V(ecoreparams.ECoreLogLevel).Infof("Node %q is notReady", currentNode.Definition.Name)
						glog.V(ecoreparams.ECoreLogLevel).Infof("  Reason: %s", condition.Reason)

						return true
					}
				}
			}

			return false
		}).WithTimeout(25*time.Minute).WithPolling(15*time.Second).WithContext(ctx).Should(BeTrue(),
			"Node hasn't reached notReady state")

		By(fmt.Sprintf("Checking node %q got into Ready", _node.Definition.Name))

		Eventually(func(ctx SpecContext) bool {
			currentNode, err := nodes.Pull(APIClient, _node.Definition.Name)
			if err != nil {
				glog.V(ecoreparams.ECoreLogLevel).Infof("Error pulling in node: %v", err)

				return false
			}

			for _, condition := range currentNode.Object.Status.Conditions {
				if condition.Type == ecoreparams.ConditionTypeReadyString {
					if condition.Status == ecoreparams.ConstantTrueString {
						glog.V(ecoreparams.ECoreLogLevel).Infof("Node %q is Ready", currentNode.Definition.Name)
						glog.V(ecoreparams.ECoreLogLevel).Infof("  Reason: %s", condition.Reason)

						return true
					}
				}
			}

			return false
		}).WithTimeout(25*time.Minute).WithPolling(15*time.Second).WithContext(ctx).Should(BeTrue(),
			"Node hasn't reached Ready state")

		glog.V(ecoreparams.ECoreLogLevel).Infof("Uncordoning node %q", _node.Definition.Name)
		err = _node.Uncordon()
		Expect(err).ToNot(HaveOccurred(),
			fmt.Sprintf("Failed to uncordon %q due to %v", _node.Definition.Name, err))

		time.Sleep(15 * time.Second)
	}
}

// VerifyHardRebootSuite container that contains tests for ungraceful cluster reboot verification.
//
//nolint:funlen
func VerifyHardRebootSuite() {
	Describe(
		"Ungraceful reboot validation",
		Label("ecore-ungraceful-reboot"), func() {
			BeforeAll(func(ctx SpecContext) {
				By("Creating SR-IOV workloads on same SR-IOV net and the same node")
				VerifySRIOVWorkloadsOnSameNode(ctx)

				By("Creaing SR-IOV workloads on same SR-IOV net and different nodes")
				VerifySRIOVWorkloadsOnDifferentNodes(ctx)

				By("Creating SR-IOV workloads on different SR-IOV nets and same node")
				VerifySRIOVWorkloadsOnSameNodeDifferentNetworks(ctx)

				By("Creating SR-IOV workloads on different SR-IOV nets and different nodes")
				VerifySRIOVWorkloadsDifferentNodesDifferentNetworks(ctx)

				By("Creating a workload with CephFS PVC")
				VerifyCephFSPVC(ctx)

				By("Creating a workload with CephFS PVC")
				VerifyCephRBDPVC(ctx)

				By("Creating MACVLAN workloads that run on different nodes")
				VerifyMacVlanOnDifferentNodes()

				By("Creating MACVLAN workloads that run on the same node")
				VerifyMacVlanOnSameNode()
			})

			It("Verifies ungraceful cluster reboot",
				Label("ecore-hard-reboot"), polarion.ID("30020"), VerifyUngracefulReboot)

			It("Verifies all ClusterOperators are Available after ungraceful reboot",
				Label("ecore-hard-reboot"), polarion.ID("71868"), func() {
					By("Checking all cluster operators")

					glog.V(ecoreparams.ECoreLogLevel).Infof("Waiting for all ClusterOperators to be Available")
					glog.V(ecoreparams.ECoreLogLevel).Infof("Sleeping for 3 minutes")

					time.Sleep(3 * time.Minute)

					ok, err := clusteroperator.WaitForAllClusteroperatorsAvailable(
						APIClient, 15*time.Minute, metav1.ListOptions{})
					Expect(err).ToNot(HaveOccurred(), "Failed to get cluster operator status")
					Expect(ok).To(BeTrue(), "Some cluster operators not Available")
				})

			It("Removes all pods with UnexpectedAdmissionError", Label("sriov-unexpected-pods"),
				MustPassRepeatedly(3), func(ctx SpecContext) {
					By("Remove any pods in UnexpectedAdmissionError state")

					glog.V(ecoreparams.ECoreLogLevel).Infof("Remove pods with UnexpectedAdmissionError status")

					glog.V(ecoreparams.ECoreLogLevel).Infof("Sleeping for 3 minutes")

					time.Sleep(3 * time.Minute)

					listOptions := metav1.ListOptions{
						FieldSelector: "status.phase=Failed",
					}

					var podsList []*pod.Builder

					var err error

					Eventually(func() bool {
						podsList, err = pod.ListInAllNamespaces(APIClient, listOptions)
						if err != nil {
							glog.V(ecoreparams.ECoreLogLevel).Infof("Failed to list pods: %v", err)

							return false
						}

						glog.V(ecoreparams.ECoreLogLevel).Infof("Found %d pods matching search criteria",
							len(podsList))

						for _, failedPod := range podsList {
							glog.V(ecoreparams.ECoreLogLevel).Infof("Pod %q in %q ns matches search criteria",
								failedPod.Definition.Name, failedPod.Definition.Namespace)
						}

						return true
					}).WithContext(ctx).WithPolling(5*time.Second).WithTimeout(1*time.Minute).Should(BeTrue(),
						"Failed to search for pods with UnexpectedAdmissionError status")

					for _, failedPod := range podsList {
						if failedPod.Definition.Status.Reason == "UnexpectedAdmissionError" {
							glog.V(ecoreparams.ECoreLogLevel).Infof("Deleting pod %q in %q ns",
								failedPod.Definition.Name, failedPod.Definition.Namespace)

							_, err := failedPod.DeleteAndWait(5 * time.Minute)
							Expect(err).ToNot(HaveOccurred(), "could not delete pod in UnexpectedAdmissionError state")
						}
					}
				})

			It("Verifies all deploymentes are available",
				Label("ecore-hard-reboot"), polarion.ID("71872"), WaitAllDeploymentsAreAvailable)

			It("Verifies all policies are compliant", polarion.ID("72355"), Label("ecore-hard-reboot-validate-policies"),
				ValidateAllPoliciesCompliant)

			It("Verifies CephFS PVC is still accessible after hard rebot",
				Label("ecore-hard-reboot-cephfs"), polarion.ID("71873"), VerifyDataOnCephFSPVC)

			It("Verifies CephRBD PVC is still accessible after hard reboot",
				Label("ecore-hard-reboot-cephrbd"), polarion.ID("71990"), VerifyDataOnCephRBDPVC)

			It("Verifies CephFS workload is deployable after hard reboot",
				Label("ecore-hard-reboot-odf-cephfs-pvc"), polarion.ID("71851"), MustPassRepeatedly(3),
				VerifyCephFSPVC)

			It("Verifies CephRBD workload is deployable after hard reboot",
				Label("ecore-hard-reboot-odf-cephrbd-pvc"), polarion.ID("71992"), MustPassRepeatedly(3),
				VerifyCephRBDPVC)

			It("Verifies SR-IOV workloads on different nodes post reboot",
				Label("ecore-hard-reboot-sriov-different-node", "ecore-hard-reboot-sriov"), polarion.ID("71952"),
				VerifySRIOVConnectivityBetweenDifferentNodesSameNet)

			It("Verifies SR-IOV workloads on the same node post reboot",
				Label("ecore-hard-reboot-sriov-same-node", "ecore-hard-reboot-sriov"), polarion.ID("71951"),
				VerifySRIOVConnectivityOnSameNodeSameNet)

			It("Verifies SR-IOV workloads on different nodes and different SR-IOV nets post reboot",
				Label("ecore-hard-reboot-sriov-different-node", "ecore-hard-reboot-sriov"), polarion.ID("72254"),
				VerifySRIOVConnectivityOnDifferentNodesDifferentNets)

			It("Verifies SR-IOV workloads on the same node and different SR-IOV nets post reboot",
				Label("ecore-hard-reboot-sriov-same-node", "ecore-hard-reboot-sriov"), polarion.ID("72255"),
				VerifySRIOVConnectivityOnSameNodeDifferentNets)

			It("Verifies SR-IOV workloads with same SR-IOV net on the same node are deployable after hard reboot",
				Label("sriov-same-net-same-node"), polarion.ID("72262"), MustPassRepeatedly(3),
				VerifySRIOVWorkloadsOnSameNode)

			It("Verifies SR-IOV workloads with same SR-IOV net on different nodes are deployable after hard reboot",
				Label("sriov-same-net-different-node"), polarion.ID("72263"), MustPassRepeatedly(3),
				VerifySRIOVWorkloadsOnDifferentNodes)

			It("Verifies SR-IOV workloads on the different SR-IOV nets and same node",
				Label("sriov-different-net-same-node"), polarion.ID("72264"), MustPassRepeatedly(3),
				VerifySRIOVWorkloadsOnSameNodeDifferentNetworks)

			It("Verifies SR-IOV workloads on different SR-IOV nets and different nodes",
				Label("sriov-same-net-different-node"), polarion.ID("72265"), MustPassRepeatedly(3),
				VerifySRIOVWorkloadsDifferentNodesDifferentNetworks)

			It("Verifies MACVLAN workloads on the same net and different nodes after hard reboot",
				Label("macvlan-same-net-different-nodes"), polarion.ID("72568"), MustPassRepeatedly(3),
				VerifyMACVLANConnectivityBetweenDifferentNodes)

			It("Verifies MACVLAN workloads on the same net and the same node after hard reboot",
				Label("macvlan-same-net-different-nodes"), polarion.ID("72569"), MustPassRepeatedly(3),
				VerifyMACVLANConnectivityOnSameNode)
		})
}

// VerifyGracefulRebootSuite container that contains tests for graceful reboot verification.
//
//nolint:funlen
func VerifyGracefulRebootSuite() {
	Describe(
		"Graceful reboot validation",
		Label("ecore-graceful-reboot"), func() {
			BeforeAll(func(ctx SpecContext) {
				By("Creating SR-IOV workloads on same SR-IOV net and the same node")
				VerifySRIOVWorkloadsOnSameNode(ctx)

				By("Creaing SR-IOV workloads on same SR-IOV net and different nodes")
				VerifySRIOVWorkloadsOnDifferentNodes(ctx)

				By("Creating SR-IOV workloads on different SR-IOV nets and same node")
				VerifySRIOVWorkloadsOnSameNodeDifferentNetworks(ctx)

				By("Creating SR-IOV workloads on different SR-IOV nets and different nodes")
				VerifySRIOVWorkloadsDifferentNodesDifferentNetworks(ctx)

				By("Creating a workload with CephFS PVC")
				VerifyCephFSPVC(ctx)

				By("Creating a workload with CephFS PVC")
				VerifyCephRBDPVC(ctx)

				By("Creating MACVLAN workloads that run on different nodes")
				VerifyMacVlanOnDifferentNodes()

				By("Creating MACVLAN workloads that run on the same node")
				VerifyMacVlanOnSameNode()
			})

			It("Verifies graceful cluster reboot",
				Label("ecore-soft-reboot"), polarion.ID("30021"), VerifySoftReboot)

			It("Verifies all ClusterOperators are Available after ungraceful reboot",
				Label("ecore-soft-reboot-cluster-operators"), polarion.ID("72040"), func() {
					By("Checking all cluster operators")

					glog.V(ecoreparams.ECoreLogLevel).Infof("Waiting for all ClusterOperators to be Available")
					glog.V(ecoreparams.ECoreLogLevel).Infof("Sleeping for 3 minutes")

					time.Sleep(3 * time.Minute)

					ok, err := clusteroperator.WaitForAllClusteroperatorsAvailable(
						APIClient, 15*time.Minute, metav1.ListOptions{})
					Expect(err).ToNot(HaveOccurred(), "Failed to get cluster operator status")
					Expect(ok).To(BeTrue(), "Some cluster operators not Available")
				})

			It("Verifies all deploymentes are available",
				Label("ecore-soft-reboot-all-deployments"), polarion.ID("72041"), WaitAllDeploymentsAreAvailable)

			It("Verifies all policies are compliant", polarion.ID("72357"), Label("ecore-soft-reboot-validate-policies"),
				ValidateAllPoliciesCompliant)

			It("Verifies CephFS PVC is still accessible after graceful reboot",
				Label("ecore-soft-reboot-cephfs"), polarion.ID("72042"), VerifyDataOnCephFSPVC)

			It("Verifies CephRBD PVC is still accessible after graceful reboot",
				Label("ecore-soft-reboot-cephrbd"), polarion.ID("72044"), VerifyDataOnCephRBDPVC)

			It("Verifies CephFS workload is deployable after graceful reboot",
				Label("ecore-soft-rebboot-odf-cephfs-pvc"), polarion.ID("72045"), MustPassRepeatedly(3),
				VerifyCephFSPVC)

			It("Verifies CephRBD workload is deployable after graceful reboot",
				Label("ecore-soft-reboot-odf-cephrbd-pvc"), polarion.ID("72046"), MustPassRepeatedly(3),
				VerifyCephRBDPVC)

			It("Verifies SR-IOV workloads on different nodes post graceful reboot",
				Label("ecore-soft-reboot-sriov-different-nodes-same-net", "ecore-soft-reboot-sriov"),
				polarion.ID("72039"), VerifySRIOVConnectivityBetweenDifferentNodesSameNet)

			It("Verifies SR-IOV workloads on the same node post graceful reboot",
				Label("ecore-soft-reboot-sriov-same-node-same-net", "ecore-soft-reboot-sriov"),
				polarion.ID("72038"), VerifySRIOVConnectivityOnSameNodeSameNet)

			It("Verifies SR-IOV workloads on different nodes and different SR-IOV nets post reboot",
				Label("ecore-soft-reboot-sriov-different-nodes-different-nets", "ecore-soft-reboot-sriov"),
				polarion.ID("72256"), VerifySRIOVConnectivityOnDifferentNodesDifferentNets)

			It("Verifies SR-IOV workloads on the same node and different SR-IOV nets post reboot",
				Label("ecore-soft-reboot-sriov-same-node-different-nets", "ecore-soft-reboot-sriov"),
				polarion.ID("72257"), VerifySRIOVConnectivityOnSameNodeDifferentNets)

			It("Verifices SR-IOV workloads with same SR-IOV net on the same node",
				Label("ecore-soft-reboot-sriov-same-net-same-node"), MustPassRepeatedly(3),
				polarion.ID("72048"), VerifySRIOVWorkloadsOnSameNode)

			It("Verifices SR-IOV workloads on same SR-IOV net and different nodes",
				Label("ecore-soft-reboot-sriov-same-net-different-nodes"), MustPassRepeatedly(3),
				polarion.ID("72049"), VerifySRIOVWorkloadsOnDifferentNodes)

			It("Verifies SR-IOV workloads on the different SR-IOV nets and different nodes",
				Label("sriov-soft-reboot-different-nets-same-node"), MustPassRepeatedly(3),
				polarion.ID("72260"), VerifySRIOVWorkloadsOnSameNodeDifferentNetworks)

			It("Verifies SR-IOV workloads on the same net and same node",
				Label("sriov-same-net-different-node"), MustPassRepeatedly(3),
				polarion.ID("72261"), VerifySRIOVWorkloadsDifferentNodesDifferentNetworks)

			It("Verifies MACVLAN workloads on the same net and different nodes after graceful reboot",
				Label("macvlan-same-net-different-nodes"), polarion.ID("72570"), MustPassRepeatedly(3),
				VerifyMACVLANConnectivityBetweenDifferentNodes)

			It("Verifies MACVLAN workloads on the same net and the same node after graceful reboot",
				Label("macvlan-same-net-different-nodes"), polarion.ID("72571"), MustPassRepeatedly(3),
				VerifyMACVLANConnectivityOnSameNode)
		})
}
