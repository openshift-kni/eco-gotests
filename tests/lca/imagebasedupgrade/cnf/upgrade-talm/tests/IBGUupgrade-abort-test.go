package upgrade_test

import (
	"github.com/openshift-kni/eco-goinfra/pkg/ibgu"
	"time"

	"k8s.io/utils/ptr"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/cgu"
	"github.com/openshift-kni/eco-goinfra/pkg/lca"
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
		var newIbguBuilder *ibgu.IbguBuilder

		BeforeEach(func() {
			By("Fetching target sno cluster name", func() {
				err := cnfclusterinfo.PreUpgradeClusterInfo.SaveClusterInfo()
				Expect(err).ToNot(HaveOccurred(), "Failed to extract target sno cluster name")

				tsparams.TargetSnoClusterName = cnfclusterinfo.PreUpgradeClusterInfo.Name

				ibu, err = lca.PullImageBasedUpgrade(cnfinittools.TargetSNOAPIClient)
				Expect(err).NotTo(HaveOccurred(), "error pulling ibu resource from cluster")
			})
		})

		AfterEach(func() {
			By("Deleting IBGU on target hub cluster", func() {
				_, err := newIbguBuilder.DeleteAndWait(1 * time.Minute)
				Expect(err).ToNot(HaveOccurred(), "Failed to delete IBGU cgu on target hub cluster")
			})

			// Sleep for 10 seconds to allow talm to reconcile state.
			// Sometimes if the next test re-creates the IBGUs too quickly,
			// the policies compliance status is not updated correctly.
			time.Sleep(10 * time.Second)
		})

		It("Abort at IBU upgrade stage", reportxml.ID("69055"), func() {
			By("Creating, enabling prep IBGU and waiting for IBU status to report completed", func() {

				newIbguBuilder = ibgu.NewIbguBuilder(cnfinittools.TargetHubAPIClient,
					tsparams.IbguName, tsparams.IbuCguNamespace).
					WithClusterLabelSelectors(tsparams.ClusterLabelSelector).
					WithOadpContent("oadp-cm", "ztp-group").
					WithSeedImageRef("registry.kni-qe-18.lab.eng.tlv2.redhat.com:5000/ibu/seed:4.17.0-rc.1", "4.17.0-rc.1").
					WithPlan([]string{"Prep"}, 20, 20)

				newIbguBuilder, err = newIbguBuilder.Create()
				Expect(err).ToNot(HaveOccurred(), "Failed to create IBGU")

				_, err = ibu.WaitUntilStageComplete("Prep")
				Expect(err).NotTo(HaveOccurred(), "error waiting for prep stage to complete")
			})

			By("Creating, and enabling ibu upgrade CGU", func() {
				newIbguBuilder = ibgu.NewIbguBuilder(cnfinittools.TargetHubAPIClient,
					tsparams.IbguName, tsparams.IbuCguNamespace).
					WithClusterLabelSelectors(tsparams.ClusterLabelSelector).
					WithOadpContent("oadp-cm", "ztp-group").
					WithSeedImageRef("registry.kni-qe-18.lab.eng.tlv2.redhat.com:5000/ibu/seed:4.17.0-rc.1", "4.17.0-rc.1").
					WithPlan([]string{"Upgrade"}, 20, 20)

				newIbguBuilder, err = newIbguBuilder.Create()
				Expect(err).ToNot(HaveOccurred(), "Failed to create IBGU")
				// Wait for 10 seconds to avoid upgrade and finalize CGUs getting created simultaneously.
				time.Sleep(10 * time.Second)
			})

			By("Waiting for the upgrade and finalize policies to report NonCompliant state", func() {
				upgradeStagePolicy, err := ocm.PullPolicy(cnfinittools.TargetHubAPIClient,
					tsparams.UpgradePolicyName,
					tsparams.IbuPolicyNamespace)
				Expect(err).ToNot(HaveOccurred(), "Failed to pull upgrade stage policy")

				err = upgradeStagePolicy.WaitUntilComplianceState(policiesv1.NonCompliant, 5*time.Minute)
				Expect(err).ToNot(HaveOccurred(), "Upgrade-stage-policy failed to report NonCompliant state")

				finalizeStagePolicy, err := ocm.PullPolicy(cnfinittools.TargetHubAPIClient,
					tsparams.FinalizePolicyName,
					tsparams.IbuPolicyNamespace)
				Expect(err).ToNot(HaveOccurred(), "Failed to pull finalize stage policy")

				err = finalizeStagePolicy.WaitUntilComplianceState(policiesv1.NonCompliant, 5*time.Minute)
				Expect(err).ToNot(HaveOccurred(), "Finalize-stage-policy failed to report NonCompliant state")
			})

			By("Creating, enabling ibu finalize CGU and waiting for CGU status to report completed", func() {
				finalizeCguBuilder := cgu.NewCguBuilder(cnfinittools.TargetHubAPIClient,
					tsparams.FinalizeCguName, tsparams.IbuCguNamespace, 1).
					WithCluster(tsparams.TargetSnoClusterName).
					WithManagedPolicy(tsparams.FinalizePolicyName).
					WithCanary(tsparams.TargetSnoClusterName)
				finalizeCguBuilder.Definition.Spec.Enable = ptr.To(true)

				// Delete the upgrade CGU so that it does not interfere with the finalize CGU.
				By("Deleting upgrade cgu created on target hub cluster", func() {
					err := cnfhelper.DeleteIbuTestCguOnTargetHub(cnfinittools.TargetHubAPIClient, tsparams.UpgradeCguName,
						tsparams.IbuCguNamespace)
					Expect(err).ToNot(HaveOccurred(), "Failed to delete upgrade cgu on target hub cluster")
				})

				finalizeCguBuilder, err := finalizeCguBuilder.Create()
				Expect(err).ToNot(HaveOccurred(), "Failed to create finalize CGU.")

				_, err = ibu.WaitUntilStageComplete("Idle")
				Expect(err).NotTo(HaveOccurred(), "error waiting for idle stage to complete")

				_, err = finalizeCguBuilder.WaitUntilComplete(5 * time.Minute)
				Expect(err).ToNot(HaveOccurred(), "Finalize CGU did not complete in time.")
			})
		})
	})
