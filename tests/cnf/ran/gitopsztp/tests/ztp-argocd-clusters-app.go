package tests

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/assisted"
	"github.com/openshift-kni/eco-goinfra/pkg/ocm"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/gitopsztp/internal/gitdetails"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/gitopsztp/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/ranhelper"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/raninittools"
)

var _ = Describe("ZTP Argo CD Clusters Tests", Label(tsparams.LabelArgoCdClustersAppTestCases), func() {
	BeforeEach(func() {
		By("verifying that ZTP meets the minimum version")
		versionInRange, err := ranhelper.IsVersionStringInRange(RANConfig.ZTPVersion, "4.11", "")
		Expect(err).ToNot(HaveOccurred(), "Failed to compare ZTP version string")

		if !versionInRange {
			Skip("ZTP Argo CD clusters app tests require ZTP 4.11 or later")
		}

	})

	AfterEach(func() {
		By("resetting the clusters app back to the original settings")
		err := gitdetails.SetGitDetailsInArgoCd(
			tsparams.ArgoCdClustersAppName, tsparams.ArgoCdAppDetails[tsparams.ArgoCdClustersAppName], true, false)
		Expect(err).ToNot(HaveOccurred(), "Failed to reset clusters app git details")
	})

	// 54238 - User modification of klustletaddonconfig via gitops
	It("should override the KlusterletAddonConfiguration and verify the change", reportxml.ID("54238"), func() {
		exists, err := gitdetails.UpdateArgoCdAppGitPath(
			tsparams.ArgoCdClustersAppName, tsparams.ZtpTestPathClustersApp, true)
		if !exists {
			Skip(err.Error())
		}

		Expect(err).ToNot(HaveOccurred(), "Failed to update Argo CD git path")

		By("validating the klusterlet addon change occurred")
		kac, err := ocm.PullKAC(HubAPIClient, RANConfig.Spoke1Name, RANConfig.Spoke1Name)
		Expect(err).ToNot(HaveOccurred(), "Failed to pull klusterlet addon config")

		_, err = kac.WaitUntilSearchCollectorEnabled(tsparams.ArgoCdChangeTimeout)
		Expect(err).ToNot(HaveOccurred(), "Failed to wait for klusterlet addon config to have search collector enabled")
	})

	// 60619 - Image creation fails when NMstateConfig CR is empty
	It("should not have NMStateConfig CR when nodeNetwork section not in siteConfig", reportxml.ID("60619"), func() {
		// Update the git path manually so we can potentially skip the test before checking if the NM State
		// Config exists.
		gitDetails := tsparams.ArgoCdAppDetails[tsparams.ArgoCdClustersAppName]
		testGitPath := gitdetails.JoinGitPaths([]string{
			gitDetails.Path,
			tsparams.ZtpTestPathRemoveNmState,
		})

		By("checking if the git path exists")
		if !gitdetails.DoesGitPathExist(gitDetails.Repo, gitDetails.Branch, testGitPath+tsparams.ZtpKustomizationPath) {
			Skip(fmt.Sprintf("git path '%s' could not be found", testGitPath))
		}

		By("checking if the NM state config exists on hub")
		nmStateConfigList, err := assisted.ListNmStateConfigsInAllNamespaces(HubAPIClient)
		Expect(err).ToNot(HaveOccurred(), "Failed to list NM state configs")
		Expect(nmStateConfigList).ToNot(BeEmpty(), "Failed to find NM state config")

		gitDetails.Path = testGitPath

		By("updating the Argo CD clusters app with the remove NM state git path")
		err = gitdetails.SetGitDetailsInArgoCd(
			tsparams.ArgoCdClustersAppName,
			gitDetails,
			true,
			true)
		Expect(err).ToNot(HaveOccurred(), "Failed to update the Argo CD app with new git details")

		By("validate the NM state config is gone on hub")
		nmStateConfigList, err = assisted.ListNmStateConfigsInAllNamespaces(HubAPIClient)
		Expect(err).ToNot(HaveOccurred(), "Failed to list NM state configs")
		Expect(nmStateConfigList).To(BeEmpty(), "Found NM state config when it should be gone")
	})
})
