package negative_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/lca"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	. "github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/mgmt/internal/mgmtinittools"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/mgmt/negative/internal/tsparams"
)

var _ = Describe(
	"Patching ibu with a not supported next stage",
	Ordered,
	Label(tsparams.LabelStageTransition), func() {
		var (
			ibu *lca.ImageBasedUpgradeBuilder
			err error
		)

		BeforeAll(func() {
			By("Pull the imagebasedupgrade from the cluster")
			ibu, err = lca.PullImageBasedUpgrade(APIClient)
			Expect(err).NotTo(HaveOccurred(), "error pulling ibu resource from cluster")

			By("Ensure that imagebasedupgrade stage is set to Idle")
			Expect(string(ibu.Object.Spec.Stage)).To(Equal("Idle"), "error: ibu resource contains unexpected state")

			ibu, err = ibu.WithSeedImage(MGMTConfig.SeedImage).
				WithSeedImageVersion(MGMTConfig.SeedClusterInfo.SeedClusterOCPVersion).Update()
			Expect(err).NotTo(HaveOccurred(), "error updating ibu with image and version")
		})

		AfterEach(func() {
			By("Pull the imagebasedupgrade from the cluster")
			ibu, err = lca.PullImageBasedUpgrade(APIClient)
			Expect(err).NotTo(HaveOccurred(), "error pulling imagebasedupgrade resource")

			if ibu.Object.Spec.Stage != "Idle" {
				By("Set IBU stage to Idle")
				_, err = ibu.WithStage("Idle").Update()
				Expect(err).NotTo(HaveOccurred(), "error setting ibu to idle stage")

				By("Wait until IBU has become Idle")
				_, err = ibu.WaitUntilStageComplete("Idle")
				Expect(err).NotTo(HaveOccurred(), "error waiting for idle stage to complete")
			}

			Expect(string(ibu.Object.Spec.Stage)).To(Equal("Idle"), "error: ibu resource contains unexpected state")
		})

		It("fails because from Idle it's not possible to move to Rollback stage", reportxml.ID("71738"), func() {

			By("Setting the IBU stage to Rollback")

			_, err := ibu.WithStage("Rollback").Update()
			Expect(err.Error()).To(ContainSubstring("the stage transition is not permitted"),
				"error: ibu seedimage updated with wrong next stage")

		})

		It("fails because from Idle it's not possible to move to Upgrade stage", reportxml.ID("71739"), func() {

			By("Setting the IBU stage to Upgrade")

			_, err := ibu.WithStage("Upgrade").Update()
			Expect(err.Error()).To(ContainSubstring("the stage transition is not permitted"),
				"error: ibu seedimage updated with wrong next stage")

		})
	})
