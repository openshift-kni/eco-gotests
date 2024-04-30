package rds_core_system_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clusteroperator"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"

	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/rdscore/internal/rdscorecommon"
	. "github.com/openshift-kni/eco-gotests/tests/system-tests/rdscore/internal/rdscoreinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/rdscore/internal/rdscoreparams"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe(
	"RDS Core Top Level Suite",
	Ordered,
	ContinueOnFailure,
	Label("rds-core-workflow"), func() {
		Context("Configured Cluster", Label("clean-cluster"), func() {
			It("Verify MACVLAN", Label("macvlan", "validate-new-macvlan-different-nodes"), reportxml.ID("72566"),
				rdscorecommon.VerifyMacVlanOnDifferentNodes)

			It("Verify MACVLAN", Label("macvlan", "validate-new-macvlan-same-node"), reportxml.ID("72567"),
				rdscorecommon.VerifyMacVlanOnSameNode)

			It("Verifices SR-IOV workloads on the same node",
				Label("sriov", "sriov-same-node"), reportxml.ID("71949"), MustPassRepeatedly(3),
				rdscorecommon.VerifySRIOVWorkloadsOnSameNode)

			It("Verifices SR-IOV workloads on different nodes",
				Label("sriov", "sriov-different-node"), reportxml.ID("71950"), MustPassRepeatedly(3),
				rdscorecommon.VerifySRIOVWorkloadsOnDifferentNodes)

			It(fmt.Sprintf("Verifies %s namespace exists", RDSCoreConfig.NMStateOperatorNamespace),
				Label("nmstate", "nmstate-ns"),
				rdscorecommon.VerifyNMStateNamespaceExists)

			It("Verifies NMState instance exists",
				Label("nmstate", "nmstate-instance"), reportxml.ID("67027"),
				rdscorecommon.VerifyNMStateInstanceExists)

			It("Verifies all NodeNetworkConfigurationPolicies are Available",
				Label("nmstate", "nmstate-nncp"), reportxml.ID("71846"),
				rdscorecommon.VerifyAllNNCPsAreOK)

			It("Verifies CephFS",
				Label("persistent-storage", "odf-cephfs-pvc"), reportxml.ID("71850"), MustPassRepeatedly(3),
				rdscorecommon.VerifyCephFSPVC)

			It("Verifies CephRBD",
				Label("persistent-storage", "odf-cephrbd-pvc"), reportxml.ID("71989"), MustPassRepeatedly(3),
				rdscorecommon.VerifyCephRBDPVC)
		})

		Context("Ungraceful Cluster Reboot", Label("ungraceful-cluster-reboot"), func() {
			BeforeAll(func(ctx SpecContext) {
				By("Creating a workload with CephFS PVC")
				rdscorecommon.VerifyCephFSPVC(ctx)

				By("Creating a workload with CephFS PVC")
				rdscorecommon.VerifyCephRBDPVC(ctx)

				By("Creating SR-IOV workloads on the same node")
				rdscorecommon.VerifySRIOVWorkloadsOnSameNode(ctx)

				By("Creating SR-IOV workloads on different nodes")
				rdscorecommon.VerifySRIOVWorkloadsOnDifferentNodes(ctx)

				By("Creating MACVLAN workloads on the same node")
				rdscorecommon.VerifyMacVlanOnSameNode()

				By("Creating MACVLAN workloads on different nodes")
				rdscorecommon.VerifyMacVlanOnDifferentNodes()
			})

			It("Verifies ungraceful cluster reboot",
				Label("rds-core-hard-reboot"), reportxml.ID("30020"),
				rdscorecommon.VerifyUngracefulReboot)

			It("Verifies all ClusterOperators are Available after ungraceful reboot",
				Label("verify-cos"), reportxml.ID("71868"), func() {
					By("Checking all cluster operators")

					glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Waiting for all ClusterOperators to be Available")
					glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Sleeping for 3 minutes")

					time.Sleep(3 * time.Minute)

					ok, err := clusteroperator.WaitForAllClusteroperatorsAvailable(
						APIClient, 15*time.Minute, metav1.ListOptions{})
					Expect(err).ToNot(HaveOccurred(), "Failed to get cluster operator status")
					Expect(ok).To(BeTrue(), "Some cluster operators not Available")
				})

			It("Removes all pods with UnexpectedAdmissionError", Label("sriov-unexpected-pods"),
				MustPassRepeatedly(3), func(ctx SpecContext) {
					By("Remove any pods in UnexpectedAdmissionError state")

					glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Remove pods with UnexpectedAdmissionError status")

					glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Sleeping for 3 minutes")

					time.Sleep(3 * time.Minute)

					listOptions := metav1.ListOptions{
						FieldSelector: "status.phase=Failed",
					}

					var podsList []*pod.Builder

					var err error

					Eventually(func() bool {
						podsList, err = pod.ListInAllNamespaces(APIClient, listOptions)
						if err != nil {
							glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Failed to list pods: %v", err)

							return false
						}

						glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Found %d pods matching search criteria",
							len(podsList))

						for _, failedPod := range podsList {
							glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Pod %q in %q ns matches search criteria",
								failedPod.Definition.Name, failedPod.Definition.Namespace)
						}

						return true
					}).WithContext(ctx).WithPolling(5*time.Second).WithTimeout(1*time.Minute).Should(BeTrue(),
						"Failed to search for pods with UnexpectedAdmissionError status")

					for _, failedPod := range podsList {
						if failedPod.Definition.Status.Reason == "UnexpectedAdmissionError" {
							glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Deleting pod %q in %q ns",
								failedPod.Definition.Name, failedPod.Definition.Namespace)

							_, err := failedPod.DeleteAndWait(5 * time.Minute)
							Expect(err).ToNot(HaveOccurred(), "could not delete pod in UnexpectedAdmissionError state")
						}
					}
				})

			It("Verifies all deploymentes are available",
				Label("verify-deployments"), reportxml.ID("71872"),
				rdscorecommon.WaitAllDeploymentsAreAvailable)

			It("Verifies CephFS PVC is still accessible",
				Label("persistent-storage", "verify-cephfs"), reportxml.ID("71873"),
				rdscorecommon.VerifyDataOnCephFSPVC)

			It("Verifies CephRBD PVC is still accessible",
				Label("persistent-storage", "verify-cephrbd"), reportxml.ID("71990"),
				rdscorecommon.VerifyDataOnCephRBDPVC)

			It("Verifies CephFS workload is deployable after hard reboot",
				Label("persistent-storage", "deploy-cephfs-pvc"), reportxml.ID("71851"), MustPassRepeatedly(3),
				rdscorecommon.VerifyCephFSPVC)

			It("Verifies CephRBD workload is deployable after hard reboot",
				Label("persistent-storage", "deploy-cephrbd-pvc"), reportxml.ID("71992"), MustPassRepeatedly(3),
				rdscorecommon.VerifyCephRBDPVC)

			It("Verifies SR-IOV workloads on different nodes post reboot",
				Label("sriov", "verify-sriov-different-node"), reportxml.ID("71952"),
				rdscorecommon.VerifySRIOVConnectivityBetweenDifferentNodes)

			It("Verifies SR-IOV workloads on the same node post reboot",
				Label("sriov", "verify-sriov-same-node"), reportxml.ID("71951"),
				rdscorecommon.VerifySRIOVConnectivityOnSameNode)

			It("Verifies MACVLAN workloads on the same node post hard reboot",
				Label("macvlan", "verify-macvlan-same-node"), reportxml.ID("72569"),
				rdscorecommon.VerifyMACVLANConnectivityOnSameNode)

			It("Verifies MACVLAN workloads on different nodes post hard reboot",
				Label("macvlan", "verify-macvlan-different-nodes"), reportxml.ID("72568"),
				rdscorecommon.VerifyMACVLANConnectivityOnSameNode)
		})

		Context("Graceful Cluster Reboot", Label("graceful-cluster-reboot"), func() {
			BeforeAll(func(ctx SpecContext) {
				By("Creating a workload with CephFS PVC")
				rdscorecommon.VerifyCephFSPVC(ctx)

				By("Creating a workload with CephFS PVC")
				rdscorecommon.VerifyCephRBDPVC(ctx)

				By("Creating SR-IOV worklods that run on same node")
				rdscorecommon.VerifySRIOVWorkloadsOnSameNode(ctx)

				By("Verifying SR-IOV workloads on different nodes")
				rdscorecommon.VerifySRIOVWorkloadsOnDifferentNodes(ctx)

				By("Creating MACVLAN workloads on the same node")
				rdscorecommon.VerifyMacVlanOnSameNode()

				By("Creating MACVLAN workloads on different nodes")
				rdscorecommon.VerifyMacVlanOnDifferentNodes()
			})

			It("Verifies graceful cluster reboot",
				Label("rds-core-soft-reboot"), reportxml.ID("30021"), rdscorecommon.VerifySoftReboot)

			It("Verifies all ClusterOperators are Available after ungraceful reboot",
				Label("verify-cos"), reportxml.ID("72040"), func() {
					By("Checking all cluster operators")

					glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Waiting for all ClusterOperators to be Available")
					glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Sleeping for 3 minutes")

					time.Sleep(3 * time.Minute)

					ok, err := clusteroperator.WaitForAllClusteroperatorsAvailable(
						APIClient, 15*time.Minute, metav1.ListOptions{})
					Expect(err).ToNot(HaveOccurred(), "Failed to get cluster operator status")
					Expect(ok).To(BeTrue(), "Some cluster operators not Available")
				})

			It("Verifies all deploymentes are available",
				Label("verify-deployments"), reportxml.ID("72041"),
				rdscorecommon.WaitAllDeploymentsAreAvailable)

			It("Verifies CephFS PVC is still accessible",
				Label("persistent-storage", "verify-cephfs"), reportxml.ID("72042"),
				rdscorecommon.VerifyDataOnCephFSPVC)

			It("Verifies CephRBD PVC is still accessible",
				Label("persistent-storage", "verify-cephrbd"), reportxml.ID("72044"),
				rdscorecommon.VerifyDataOnCephRBDPVC)

			It("Verifies CephFS workload is deployable after graceful reboot",
				Label("persistent-storage", "deploy-cephfs-pvc"), reportxml.ID("72045"), MustPassRepeatedly(3),
				rdscorecommon.VerifyCephFSPVC)

			It("Verifies CephRBD workload is deployable after graceful reboot",
				Label("persistent-storage", "deploy-cephrbd-pvc"), reportxml.ID("72046"), MustPassRepeatedly(3),
				rdscorecommon.VerifyCephRBDPVC)

			It("Verifies SR-IOV workloads on different nodes post graceful reboot",
				Label("sriov", "verify-sriov-different-node"), reportxml.ID("72039"),
				rdscorecommon.VerifySRIOVConnectivityBetweenDifferentNodes)

			It("Verifies SR-IOV workloads on the same node post graceful reboot",
				Label("sriov", "verify-sriov-same-node"), reportxml.ID("72038"),
				rdscorecommon.VerifySRIOVConnectivityOnSameNode)

			It("Verifies SR-IOV workloads deployable on the same node after graceful reboot",
				Label("sriov", "deploy-sriov-same-node"), reportxml.ID("72048"), MustPassRepeatedly(3),
				rdscorecommon.VerifySRIOVWorkloadsOnSameNode)

			It("Verifies SR-IOV workloads deployable on different nodes after graceful reboot",
				Label("sriov", "deploy-sriov-different-node"), reportxml.ID("72049"), MustPassRepeatedly(3),
				rdscorecommon.VerifySRIOVWorkloadsOnDifferentNodes)

			It("Verifies MACVLAN workloads on the same node post soft reboot",
				Label("macvlan", "verify-macvlan-same-node"), reportxml.ID("72571"),
				rdscorecommon.VerifyMACVLANConnectivityOnSameNode)

			It("Verifies MACVLAN workloads on different nodes post soft reboot",
				Label("macvlan", "verify-macvlan-different-nodes"), reportxml.ID("72570"),
				rdscorecommon.VerifyMACVLANConnectivityOnSameNode)
		})
	})
