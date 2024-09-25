package upgrade_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/ibgu"
	"github.com/openshift-kni/eco-goinfra/pkg/lca"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/cnf/internal/cnfclusterinfo"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/cnf/internal/cnfinittools"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/cnf/upgrade-talm/internal/tsparams"
)

var _ = Describe(
	"Validating abort at IBU upgrade stage",
	Label(tsparams.LabelUpgradeAbortFlow), func() {
		var newIbguBuilder *ibgu.IbguBuilder
		var abortIbguBuilder *ibgu.IbguBuilder

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
			By("Deleting IBGUs on target hub cluster", func() {
				_, err := newIbguBuilder.DeleteAndWait(1 * time.Minute)
				Expect(err).ToNot(HaveOccurred(), "Failed to delete IBGU cgu on target hub cluster")

				_, err = abortIbguBuilder.DeleteAndWait(1 * time.Minute)
				Expect(err).ToNot(HaveOccurred(), "Failed to delete Abort IBGU cgu on target hub cluster")
			})

			// Sleep for 10 seconds to allow talm to reconcile state.
			// Sometimes if the next test re-creates the IBGUs too quickly,
			// the policies compliance status is not updated correctly.
			time.Sleep(10 * time.Second)
		})

		It("Aborts an upgrade at IBU upgrade stage", reportxml.ID("69055"), func() {
			By("Creating an upgrade IBGU", func() {

				newIbguBuilder = ibgu.NewIbguBuilder(cnfinittools.TargetHubAPIClient,
					tsparams.IbguName, tsparams.IbguNamespace).
					WithClusterLabelSelectors(tsparams.ClusterLabelSelector).
					WithOadpContent("oadp-cm", "ztp-group").
					WithSeedImageRef("registry.kni-qe-18.lab.eng.tlv2.redhat.com:5000/ibu/seed:4.17.0-rc.1", "4.17.0-rc.1").
					WithPlan([]string{"Prep"}, 20, 20).
					WithPlan([]string{"Upgrade"}, 20, 20).
					WithPlan([]string{"FinalizeUpgrade"}, 20, 20)

				newIbguBuilder, err = newIbguBuilder.Create()
				Expect(err).ToNot(HaveOccurred(), "Failed to create IBGU")

			})

			By("Aborting the upgrade phase once prep phase has finished", func() {

				_, err = ibu.WaitUntilStageComplete("Prep")
				Expect(err).NotTo(HaveOccurred(), "error waiting for prep stage to complete")

				newIbguBuilder := ibgu.NewIbguBuilder(cnfinittools.TargetHubAPIClient,
					tsparams.AbortIbguName, tsparams.IbguNamespace).
					WithClusterLabelSelectors(tsparams.ClusterLabelSelector).
					WithOadpContent("oadp-cm", "ztp-group").
					WithSeedImageRef("registry.kni-qe-18.lab.eng.tlv2.redhat.com:5000/ibu/seed:4.17.0-rc.1", "4.17.0-rc.1").
					WithPlan([]string{"Abort"}, 20, 20)

				abortIbguBuilder, err = newIbguBuilder.Create()
				Expect(err).ToNot(HaveOccurred(), "Failed to create IBGU")
				// Wait for 10 seconds to avoid upgrade and finalize CGUs getting created simultaneously.
				time.Sleep(10 * time.Second)
			})

			By("Waiting until the IBU has reached the idle stage", func() {

				_, err = ibu.WaitUntilStageComplete("Idle")
				Expect(err).NotTo(HaveOccurred(), "error waiting for idle stage to complete")

			})
		})
	})
