package rds_core_system_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift-kni/eco-goinfra/pkg/clusteroperator"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"

	"github.com/openshift-kni/eco-gotests/tests/system-tests/rdscore/internal/rdscorecommon"
	. "github.com/openshift-kni/eco-gotests/tests/system-tests/rdscore/internal/rdscoreinittools"
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

	})
