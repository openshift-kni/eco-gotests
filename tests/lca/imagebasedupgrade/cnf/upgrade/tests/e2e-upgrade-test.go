package upgrade_test

import (
	. "github.com/onsi/ginkgo/v2"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/cnf/upgrade/internal/tsparams"
)

var _ = Describe(
	"Performing happy path image based upgrade",
	Ordered,
	Label(tsparams.LabelEndToEndUpgrade), func() {
		BeforeAll(func() {
			By("Saving pre upgrade cluster info", func() {
				// Test Automation Code Implementation under development.
			})

			AfterAll(func() {
				// Rollback to pre-upgrade state and reset ACM policies back to initial state.
			})

			It("Upgrade end to end", reportxml.ID("68954"), func() {
				// The ibu test data is stored in a nested directory within the gitops ztp repo
				// Test Automation Code Implementation under development.
				By("Checking if the git path exists", func() {
					// Test Automation Code Implementation under development.
				})

				By("Updating the Argocd app", func() {
					// Update the Argo app to point to the new test kustomization
					// Test Automation Code Implementation under development.
				})

				By("Creating and applying ibu pre-prep CGU and wait until it report completed", func() {
					// Test Automation Code Implementation under development.
				})

				By("Creating and applying ibu prep CGU and wait until it report completed", func() {
					// Test Automation Code Implementation under development.
				})

				By("Creating and applying ibu upgrade CGU and wait until it report completed", func() {
					// Test Automation Code Implementation under development.
				})

				By(" Verifying target SNO cluster ACM registration post upgrade", func() {
					// Test Automation Code Implementation under development.
				})

				By("Creating and applying ibu finalize CGU and wait until it report completed", func() {
					// Test Automation Code Implementation under development.
				})
			})
		})
	})
