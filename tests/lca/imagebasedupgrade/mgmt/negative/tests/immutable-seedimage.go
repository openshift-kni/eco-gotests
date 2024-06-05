package negative_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/lca"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/internal/nodestate"
	. "github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/mgmt/internal/mgmtinittools"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/mgmt/negative/internal/tsparams"
	lcav1 "github.com/openshift-kni/lifecycle-agent/api/imagebasedupgrade/v1"
	"golang.org/x/exp/slices"
)

var _ = Describe(
	"Patching ibu seedimage ref while prepping images",
	Ordered,
	Label(tsparams.LabelImmutableSeedImage), func() {
		var (
			ibu *lca.ImageBasedUpgradeBuilder
			err error

			negativeSeedImage        = "quay.io/ibutest/negative:latest"
			negativeSeedImageVersion = "4.16.0-negative"
		)

		BeforeAll(func() {
			By("Pull the imagebasedupgrade from the cluster")
			ibu, err = lca.PullImageBasedUpgrade(APIClient)
			Expect(err).NotTo(HaveOccurred(), "error pulling ibu resource from cluster")

			By("Ensure that imagebasedupgrade values are empty")
			ibu.Definition.Spec.ExtraManifests = []lcav1.ConfigMapRef{}
			ibu.Definition.Spec.OADPContent = []lcav1.ConfigMapRef{}
			_, err = ibu.Update()
			Expect(err).NotTo(HaveOccurred(), "error updating ibu resource with empty values")
		})

		AfterAll(func() {
			By("Revert IBU resource back to Idle stage")
			ibu, err = lca.PullImageBasedUpgrade(APIClient)
			Expect(err).NotTo(HaveOccurred(), "error pulling imagebasedupgrade resource")

			if ibu.Object.Spec.Stage == "Upgrade" {
				By("Set IBU stage to Rollback")
				_, err = ibu.WithStage("Rollback").Update()
				Expect(err).NotTo(HaveOccurred(), "error setting ibu to rollback stage")

				By("Wait for IBU resource to be available")
				err = nodestate.WaitForIBUToBeAvailable(APIClient, ibu, time.Minute*10)
				Expect(err).NotTo(HaveOccurred(), "error waiting for ibu resource to become available")

				By("Wait until Rollback stage has completed")
				_, err = ibu.WaitUntilStageComplete("Rollback")
				Expect(err).NotTo(HaveOccurred(), "error waiting for rollback stage to complete")
			}

			if slices.Contains([]string{"Prep", "Rollback"}, string(ibu.Object.Spec.Stage)) {
				By("Set IBU stage to Idle")
				_, err = ibu.WithStage("Idle").Update()
				Expect(err).NotTo(HaveOccurred(), "error setting ibu to idle stage")

				By("Wait until IBU has become Idle")
				_, err = ibu.WaitUntilStageComplete("Idle")
				Expect(err).NotTo(HaveOccurred(), "error waiting for idle stage to complete")
			}

			Expect(string(ibu.Object.Spec.Stage)).To(Equal("Idle"), "error: ibu resource contains unexpected state")

		})

		It("fails because seedImageRef is immutable while progressing", reportxml.ID("71383"), func() {
			ibu, err = ibu.WithSeedImage(MGMTConfig.SeedImage).
				WithSeedImageVersion(MGMTConfig.SeedClusterInfo.SeedClusterOCPVersion).Update()
			Expect(err).NotTo(HaveOccurred(), "error updating ibu with image and version")

			By("Setting the IBU stage to Prep")

			_, err := ibu.WithStage("Prep").Update()
			Expect(err).NotTo(HaveOccurred(), "error setting ibu to prep stage")

			ibu.Definition, err = ibu.Get()
			Expect(err).To(BeNil(), "error: getting updated ibu")

			By("Updating IBU with different seedimage")
			_, err = ibu.WithSeedImage(negativeSeedImage).Update()
			Expect(err.Error()).To(ContainSubstring("can not change spec.seedImageRef while ibu is in progress"),
				"error: ibu seedimage updated while in prep phase")

			By("Updating IBU with different seedimage version")
			_, err = ibu.WithSeedImageVersion(negativeSeedImageVersion).Update()
			Expect(err.Error()).To(ContainSubstring("can not change spec.seedImageRef while ibu is in progress"),
				"error: ibu seedimage version updated while in prep phase")
		})
	})
