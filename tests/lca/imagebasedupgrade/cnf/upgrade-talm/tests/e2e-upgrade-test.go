package upgrade_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/cgu"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/cnf/internal/cnfcluster"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/cnf/internal/cnfclusterinfo"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/cnf/internal/cnfhelper"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/cnf/internal/cnfinittools"
	cnfibuvalidations "github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/cnf/internal/validations"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/cnf/upgrade-talm/internal/tsparams"
	"k8s.io/utils/ptr"
)

var _ = Describe(
	"Performing happy path image based upgrade",
	Ordered,
	ContinueOnFailure,
	Label(tsparams.LabelEndToEndUpgrade), func() {

		var (
			clusterList []*clients.Settings
		)

		BeforeAll(func() {
			// Initialize cluster list.
			clusterList = cnfhelper.GetAllTestClients()

			// Check that the required clusters are present.
			err := cnfcluster.CheckClustersPresent(clusterList)
			if err != nil {
				Skip(fmt.Sprintf("error occurred validating required clusters are present: %s", err.Error()))
			}

			By("Saving target sno cluster info before upgrade", func() {
				err := cnfclusterinfo.PreUpgradeClusterInfo.SaveClusterInfo()
				Expect(err).ToNot(HaveOccurred(), "Failed to collect and save target sno cluster info before upgrade")

				tsparams.TargetSnoClusterName = cnfclusterinfo.PreUpgradeClusterInfo.Name

			})
		})

		AfterAll(func() {
			// Deleting CGUs created for validating IBU stages.
			By("Deleting pre-prep cgu created on target hub cluster", func() {
				err := cnfhelper.DeleteIbuTestCguOnTargetHub(cnfinittools.TargetHubAPIClient, tsparams.PrePrepCguName,
					tsparams.IbuCguNamespace)
				Expect(err).ToNot(HaveOccurred(), "Failed to delete pre-prep cgu on target hub cluster")
			})

			By("Deleting prep cgu created on target hub cluster", func() {
				err := cnfhelper.DeleteIbuTestCguOnTargetHub(cnfinittools.TargetHubAPIClient, tsparams.PrepCguName,
					tsparams.IbuCguNamespace)
				Expect(err).ToNot(HaveOccurred(), "Failed to delete prep cgu on target hub cluster")
			})

			By("Deleting upgrade cgu created on target hub cluster", func() {
				err := cnfhelper.DeleteIbuTestCguOnTargetHub(cnfinittools.TargetHubAPIClient, tsparams.UpgradeCguName,
					tsparams.IbuCguNamespace)
				Expect(err).ToNot(HaveOccurred(), "Failed to delete upgrade cgu on target hub cluster")
			})

			By("Creating, enabling rollback CGU and waiting for CGU status to report completed", func() {
				rollbackCguBuilder := cgu.NewCguBuilder(cnfinittools.TargetHubAPIClient,
					tsparams.RollbackCguName, tsparams.IbuCguNamespace, 1).
					WithCluster(tsparams.TargetSnoClusterName).
					WithManagedPolicy(tsparams.RollbackPolicyName).
					WithCanary(tsparams.TargetSnoClusterName)
				rollbackCguBuilder.Definition.Spec.Enable = ptr.To(true)

				rollbackCguBuilder, err := rollbackCguBuilder.Create()
				Expect(err).ToNot(HaveOccurred(), "Failed to create rollback CGU.")

				_, err = rollbackCguBuilder.WaitUntilComplete(35 * time.Minute)
				Expect(err).ToNot(HaveOccurred(), "Rollback CGU did not complete in time.")
			})

			By("Creating, enabling ibu finalize CGU and waiting for CGU status to report completed", func() {
				finalizeCguBuilder := cgu.NewCguBuilder(cnfinittools.TargetHubAPIClient,
					tsparams.FinalizeCguName, tsparams.IbuCguNamespace, 1).
					WithCluster(tsparams.TargetSnoClusterName).
					WithManagedPolicy(tsparams.FinalizePolicyName).
					WithCanary(tsparams.TargetSnoClusterName)
				finalizeCguBuilder.Definition.Spec.Enable = ptr.To(true)

				finalizeCguBuilder, err := finalizeCguBuilder.Create()
				Expect(err).ToNot(HaveOccurred(), "Failed to create finalize CGU.")

				_, err = finalizeCguBuilder.WaitUntilComplete(5 * time.Minute)
				Expect(err).ToNot(HaveOccurred(), "Finalize CGU did not complete in time.")
			})

			By("Deleting rollback cgu created on target hub cluster", func() {
				err := cnfhelper.DeleteIbuTestCguOnTargetHub(cnfinittools.TargetHubAPIClient, tsparams.RollbackCguName,
					tsparams.IbuCguNamespace)
				Expect(err).ToNot(HaveOccurred(), "Failed to delete rollback cgu on target hub cluster")
			})

			By("Deleting finalize cgu created on target hub cluster", func() {
				err := cnfhelper.DeleteIbuTestCguOnTargetHub(cnfinittools.TargetHubAPIClient, tsparams.FinalizeCguName,
					tsparams.IbuCguNamespace)
				Expect(err).ToNot(HaveOccurred(), "Failed to delete finalize cgu on target hub cluster")
			})
		})

		It("Upgrade end to end", reportxml.ID("68954"), func() {
			By("Creating, enabling ibu pre-prep CGU and waiting for CGU status to report completed", func() {
				prePrepCguBuilder := cgu.NewCguBuilder(cnfinittools.TargetHubAPIClient,
					tsparams.PrePrepCguName, tsparams.IbuCguNamespace, 1).
					WithCluster(tsparams.TargetSnoClusterName).
					WithManagedPolicy(tsparams.PrePrepPolicyName).
					WithCanary(tsparams.TargetSnoClusterName)
				prePrepCguBuilder.Definition.Spec.Enable = ptr.To(true)

				prePrepCguBuilder, err := prePrepCguBuilder.Create()
				Expect(err).ToNot(HaveOccurred(), "Failed to create pre-prep CGU.")

				_, err = prePrepCguBuilder.WaitUntilComplete(10 * time.Minute)
				Expect(err).ToNot(HaveOccurred(), "Pre-prep CGU did not complete in time.")
			})

			By("Creating, enabling ibu prep CGU and waiting for CGU status to report completed", func() {
				prepCguBuilder := cgu.NewCguBuilder(cnfinittools.TargetHubAPIClient,
					tsparams.PrepCguName, tsparams.IbuCguNamespace, 1).
					WithCluster(tsparams.TargetSnoClusterName).
					WithManagedPolicy(tsparams.PrepPolicyName).
					WithCanary(tsparams.TargetSnoClusterName)
				prepCguBuilder.Definition.Spec.Enable = ptr.To(true)

				prepCguBuilder, err := prepCguBuilder.Create()
				Expect(err).ToNot(HaveOccurred(), "Failed to create prep CGU.")

				_, err = prepCguBuilder.WaitUntilComplete(25 * time.Minute)
				Expect(err).ToNot(HaveOccurred(), "Prep CGU did not complete in time.")
			})

			By("Creating, enabling ibu upgrade CGU and waiting for CGU status to report completed", func() {
				upgradeCguBuilder := cgu.NewCguBuilder(cnfinittools.TargetHubAPIClient,
					tsparams.UpgradeCguName, tsparams.IbuCguNamespace, 1).
					WithCluster(tsparams.TargetSnoClusterName).
					WithManagedPolicy(tsparams.UpgradePolicyName).
					WithCanary(tsparams.TargetSnoClusterName)
				upgradeCguBuilder.Definition.Spec.Enable = ptr.To(true)

				upgradeCguBuilder, err := upgradeCguBuilder.Create()
				Expect(err).ToNot(HaveOccurred(), "Failed to create upgrade CGU.")

				_, err = upgradeCguBuilder.WaitUntilComplete(25 * time.Minute)
				Expect(err).ToNot(HaveOccurred(), "Upgrade CGU did not complete in time.")
			})

			By("Saving target sno cluster info after upgrade", func() {
				err := cnfclusterinfo.PostUpgradeClusterInfo.SaveClusterInfo()
				Expect(err).ToNot(HaveOccurred(), "Failed to collect and save target sno cluster info after upgrade")
			})

		})

		cnfibuvalidations.PostUpgradeValidations()

	})
