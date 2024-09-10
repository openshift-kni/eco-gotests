package tests

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/gitopsztp/internal/gitdetails"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/gitopsztp/internal/tsparams"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/raninittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/version"
)

var _ = Describe("ZTP Siteconfig Operator Tests", Label(tsparams.LabelSiteconfigOperatorTestCases), func() {
	BeforeEach(func() {
		By("verifying that ZTP meets the minimum version")
		versionInRange, err := version.IsVersionStringInRange(RANConfig.ZTPVersion, "4.17", "")
		Expect(err).ToNot(HaveOccurred(), "Failed to compare ZTP version string")

		if !versionInRange {
			Skip("ZTP Siteconfig operator tests require ZTP 4.17 or later")
		}

	})

	AfterEach(func() {
		By("resetting the clusters app back to the original settings")
		err := gitdetails.SetGitDetailsInArgoCd(
			tsparams.ArgoCdClustersAppName, tsparams.ArgoCdAppDetails[tsparams.ArgoCdClustersAppName], true, false)
		Expect(err).ToNot(HaveOccurred(), "Failed to reset clusters app git details")
	})

	// 75382 - Recovery test by referencing non-existent cluster template configmap.
	It("Verify new siteconfig operator’s recovery mechanism by referencing non-existent "+
		"cluster template configmap", reportxml.ID("75382"), func() {
		// Test validation steps automation code implementation is in-progress.
	})

	// 75383 - Recovery test by referencing non-existent extra manifests configmap.
	It("Verify new siteconfig operator’s recovery mechanism by referencing non-existent "+
		"extra manifests configmap", reportxml.ID("75383"), func() {
		// Test validation steps automation code implementation is in-progress.
	})
})
