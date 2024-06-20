package upgrade_test

import (
	"time"

	"k8s.io/utils/ptr"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/cgu"
	"github.com/openshift-kni/eco-goinfra/pkg/ocm"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/cnf/internal/cnfclusterinfo"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/cnf/internal/cnfhelper"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/cnf/internal/cnfinittools"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/cnf/upgrade-talm/internal/tsparams"

	policiesv1 "open-cluster-management.io/governance-policy-propagator/api/v1"
)

var _ = Describe(
	"Validating abort at IBU upgrade stage",
	Label(tsparams.LabelUpgradeAbortFlow), func() {

		BeforeEach(func() {
			By("Fetching target sno cluster name", func() {
				err := cnfclusterinfo.PreUpgradeClusterInfo.SaveClusterInfo()
				Expect(err).ToNot(HaveOccurred(), "Failed to extract target sno cluster name")

				tsparams.TargetSnoClusterName = cnfclusterinfo.PreUpgradeClusterInfo.Name
			})
		})

		AfterEach(func() {
			// Deleting CGUs created for validating the test case.
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

			By("Deleting finalize cgu created on target hub cluster", func() {
				err := cnfhelper.DeleteIbuTestCguOnTargetHub(cnfinittools.TargetHubAPIClient, tsparams.FinalizeCguName,
					tsparams.IbuCguNamespace)
				Expect(err).ToNot(HaveOccurred(), "Failed to delete finalize cgu on target hub cluster")
			})
		})

		It("Abort at IBU upgrade stage", reportxml.ID("69055"), func() {
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

			By("Creating, and enabling ibu upgrade CGU", func() {
				upgradeCguBuilder := cgu.NewCguBuilder(cnfinittools.TargetHubAPIClient,
					tsparams.UpgradeCguName, tsparams.IbuCguNamespace, 1).
					WithCluster(tsparams.TargetSnoClusterName).
					WithManagedPolicy(tsparams.UpgradePolicyName).
					WithCanary(tsparams.TargetSnoClusterName)
				upgradeCguBuilder.Definition.Spec.Enable = ptr.To(true)

				_, err := upgradeCguBuilder.Create()
				Expect(err).ToNot(HaveOccurred(), "Failed to create upgrade CGU.")
			})

			By("Waiting for the upgrade-stage-policy to report NonCompliant state", func() {
				upgradeStagePolicy, err := ocm.PullPolicy(cnfinittools.TargetHubAPIClient,
					tsparams.UpgradePolicyName,
					tsparams.IbuPolicyNamespace)
				Expect(err).ToNot(HaveOccurred(), "Failed to pull upgrade stage policy")

				err = upgradeStagePolicy.WaitUntilComplianceState(policiesv1.NonCompliant, 5*time.Minute)
				Expect(err).ToNot(HaveOccurred(), "Upgrade-stage-policy failed to report NonCompliant state")
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
		})
	})
