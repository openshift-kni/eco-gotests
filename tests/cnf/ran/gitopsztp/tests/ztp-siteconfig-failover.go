package tests

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/reportxml"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/siteconfig"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/ran/gitopsztp/internal/gitdetails"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/ran/gitopsztp/internal/tsparams"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/ran/internal/version"

	. "github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/ran/internal/raninittools"
)

var _ = Describe("ZTP Siteconfig Operator's Failover Tests",
	Label(tsparams.LabelSiteconfigFailoverTestCases), func() {
		// These tests use the hub and spoke architecture.
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
				tsparams.ArgoCdClustersAppName, tsparams.ArgoCdAppDetails[tsparams.ArgoCdClustersAppName],
				true, false)
			Expect(err).ToNot(HaveOccurred(), "Failed to reset clusters app git details")
		})

		// 75382 - Validate recovery mechanism by referencing non-existent cluster template configmap custom resource.
		It("Verify siteconfig operator’s recovery mechanism by referencing non-existent cluster template configmap CR",
			reportxml.ID("75382"), func() {

				// Test step 1-Update the ztp-test git path to reference non-existent cluster template configmap.
				// in clusterinstance.yaml.
				By("updating the Argo CD clusters app with the non-existent cluster template configmap reference git path")
				exists, err := gitdetails.UpdateArgoCdAppGitPath(tsparams.ArgoCdClustersAppName,
					tsparams.ZtpTestPathNoClusterTemplateCm, true)
				if !exists {
					Skip(err.Error())
				}

				Expect(err).ToNot(HaveOccurred(), "Failed to update Argo CD clusters app with new git path")

				// Test step 1 expected result validation.
				By("checking cluster instance CR reporting validation failed with correct error message")
				clusterInstance, err := siteconfig.PullClusterInstance(HubAPIClient, RANConfig.Spoke1Name, RANConfig.Spoke1Name)
				Expect(err).ToNot(HaveOccurred(), "Failed to find cluster instance custom resource on hub")

				// Test step 1.b expected result validation.
				// i.e. to check the proper message: 'Validation failed: .
				// failed to validate cluster-level TemplateRef'.
				_, err = clusterInstance.WaitForCondition(tsparams.CINonExistentClusterTemplateConfigMapCondition,
					tsparams.SiteconfigOperatorDefaultReconcileTime)
				Expect(err).ToNot(HaveOccurred(), "Cluster instance failed to report an expected error message")
			})

		// 75383 - Validate recovery mechanism by referencing non-existent extra manifests configmap custom resource.
		It("Verify siteconfig operator’s recovery mechanism by referencing non-existent extra manifests configmap CR",
			reportxml.ID("75383"), func() {

				// Test step 1-Update the ztp-test git path to reference non-existent extra manifests configmap.
				// in clusterinstance.yaml.
				By("updating the Argo CD clusters app with the non-existent extra manifests configmap reference git path")
				exists, err := gitdetails.UpdateArgoCdAppGitPath(tsparams.ArgoCdClustersAppName,
					tsparams.ZtpTestPathNoExtraManifestsCm, true)
				if !exists {
					Skip(err.Error())
				}

				Expect(err).ToNot(HaveOccurred(), "Failed to update Argo CD clusters app with new git path")

				// Test step 1 expected result validation.
				By("checking cluster instance CR reporting validation failed with correct error message")
				clusterInstance, err := siteconfig.PullClusterInstance(HubAPIClient, RANConfig.Spoke1Name, RANConfig.Spoke1Name)
				Expect(err).ToNot(HaveOccurred(), "Failed to find cluster instance custom resource on hub")

				// Test step 1.b expected result validation.
				// i.e. to check the proper message: 'Validation failed: failed to retrieve ExtraManifest'.
				_, err = clusterInstance.WaitForCondition(tsparams.CINonExistentExtraManifestConfigMapCondition,
					tsparams.SiteconfigOperatorDefaultReconcileTime)
				Expect(err).ToNot(HaveOccurred(), "Cluster instance failed to report an expected error message")
			})
	})
