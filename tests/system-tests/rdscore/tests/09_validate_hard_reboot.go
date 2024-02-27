package rds_core_system_test

import (
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift-kni/eco-goinfra/pkg/clusteroperator"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"

	"github.com/openshift-kni/eco-gotests/tests/system-tests/rdscore/internal/rdscorecommon"
	. "github.com/openshift-kni/eco-gotests/tests/system-tests/rdscore/internal/rdscoreinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/rdscore/internal/rdscoreparams"
)

var _ = Describe(
	"Ungraceful reboot validation",
	Ordered,
	ContinueOnFailure,
	Label("rds-core-ungraceful-reboot"), func() {

		BeforeAll(func(ctx SpecContext) {
			By("Creating a workload with CephFS PVC")
			rdscorecommon.VerifyCephFSPVC(ctx)

			By("Creating a workload with CephFS PVC")
			rdscorecommon.VerifyCephRBDPVC(ctx)
		})

		It("Verifies ungraceful cluster reboot",
			Label("rds-core-hard-reboot"), polarion.ID("30020"), rdscorecommon.VerifyUngracefulReboot)

		It("Removes all pods with UnexpectedAdmissionError", Label("sriov-unexpected-pods"), func(ctx SpecContext) {
			By("Remove any pods in UnexpectedAdmissionError state")

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

		It("Verifies all ClusterOperators are Available after ungraceful reboot",
			Label("rds-core-hard-reboot"), polarion.ID("71868"), func() {
				By("Checking all cluster operators")
				ok, err := clusteroperator.WaitForAllClusteroperatorsAvailable(
					APIClient, 15*time.Minute, metav1.ListOptions{})
				Expect(err).ToNot(HaveOccurred(), "Failed to get cluster operator status")
				Expect(ok).To(BeTrue(), "Some cluster operators not Available")

			})

		It("Verifies all deploymentes are available",
			Label("rds-core-hard-reboot"), polarion.ID("71872"), rdscorecommon.WaitAllDeploymentsAreAvailable)

		It("Verifies CephFS PVC is still accessible",
			Label("rds-core-hard-reboot-cephfs"), polarion.ID("71873"), rdscorecommon.VerifyDataOnCephFSPVC)

		It("Verifices SR-IOV workloads on different nodes post reboot",
			Label("rds-core-hard-reboot-sriov-different-node"), polarion.ID("71952"),
			rdscorecommon.VerifySRIOVConnectivityBetweenDifferentNodes)

		It("Verifices SR-IOV workloads on the same node post reboot",
			Label("rds-core-hard-reboot-sriov-same-node"), polarion.ID("71951"),
			rdscorecommon.VerifySRIOVConnectivityOnSameNode)

	})
