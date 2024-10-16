package tests

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/siteconfig"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/gitopsztp/internal/gitdetails"

	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/gitopsztp/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/version"

	. "github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/raninittools"
)

var _ = Describe("ZTP Siteconfig Operator's Negative Tests",
	Label(tsparams.LabelSiteconfigNegativeTestCases), func() {
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

		// 75378 - Validate erroneous/invalid ClusterInstance CR does not block other ClusterInstance CR handling.
		It("Verify erroneous/invalid ClusterInstance CR does not block other ClusterInstance CR handling",
			reportxml.ID("75378"), func() {

				// Deploy a first ClusterInstance CR with invalid template reference(i.e.invalid ClusterInstance CR).
				// Test step-1: Update the ztp-test git path to reference invalid node template configmap.
				// in kind:ClusterInstance to make ClusterInstance CR invalid.
				By("updating the Argo CD clusters app with invalid template reference git path")
				exists, err := gitdetails.UpdateArgoCdAppGitPath(tsparams.ArgoCdClustersAppName,
					tsparams.ZtpTestPathInvalidTemplateRef, true)
				if !exists {
					Skip(err.Error())
				}

				Expect(err).ToNot(HaveOccurred(), "Failed to update Argo CD clusters app with new git path")

				// Make sure that first ClusterInstance CR should show ClusterInstanceValidated false.
				// Test step 1.a expected result validation.
				By("checking cluster instance1 CR reporting validation failed with correct error message")
				clusterInstance1, err := siteconfig.PullClusterInstance(HubAPIClient, RANConfig.Spoke1Name, RANConfig.Spoke1Name)
				Expect(err).ToNot(HaveOccurred(), "Failed to find cluster instance1 custom resource on hub")

				// Test step 1.b expected result validation.
				// i.e. to check the proper message: 'Validation failed: .
				// failed to validate node-level TemplateRef'.
				_, err = clusterInstance1.WaitForCondition(tsparams.CIInvalidTemplateRefCondition,
					tsparams.SiteconfigOperatorDefaultReconcileTime)
				Expect(err).ToNot(HaveOccurred(), "cluster instance1 failed to report an expected error message")

				// Deploy a second ClusterInstance CR with template reference which is valid.
				// Test step-2: Update the ztp-test git path to reference valid cluster & node.
				// template configmap in kind:ClusterInstance to make ClusterInstance CR valid.
				By("updating the Argo CD clusters app with valid template reference git path")
				exists, err = gitdetails.UpdateArgoCdAppGitPath(tsparams.ArgoCdClustersAppName,
					tsparams.ZtpTestPathValidTemplateRef, true)
				if !exists {
					Skip(err.Error())
				}

				Expect(err).ToNot(HaveOccurred(), "Failed to update Argo CD clusters app with new git path")

				// Make sure that second ClusterInstance CR should proceed to provisioning the spoke cluster.
				// Test step 2.a expected result validation.
				By("checking cluster instance2 CR reporting correct message and reason")
				clusterInstance2, err := siteconfig.PullClusterInstance(HubAPIClient, RANConfig.Spoke2Name, RANConfig.Spoke2Name)
				Expect(err).ToNot(HaveOccurred(), "Failed to find cluster instance2 custom resource on hub")

				// Test step 2.b expected result validation.
				// I.e. ClusterInstance2 reporting message: Provisioning cluster.
				// with reason: InProgress.
				_, err = clusterInstance2.WaitForCondition(tsparams.CIValidTemplateRefCondition,
					tsparams.SiteconfigOperatorDefaultReconcileTime)
				Expect(err).ToNot(HaveOccurred(), "cluster instance2 failed to report an expected message and reason")
			})

		// 75379 - Validate two ClusterInstance CR with duplicate cluster name.
		It("Verify two ClusterInstance CR with duplicate cluster name",
			reportxml.ID("75379"), func() {

				// Deploy a first ClusterInstance CR named "site-plan-A" with valid template reference.
				// and field clusterName: "clusterA".
				// Test step-1: Update the ztp-test git path to reference valid template.
				// in kind:ClusterInstance with field clusterName: "clusterA".
				By("updating the Argo CD clusters app with unique cluster name git path")
				exists, err := gitdetails.UpdateArgoCdAppGitPath(tsparams.ArgoCdClustersAppName,
					tsparams.ZtpTestPathUniqueClusterName, true)
				if !exists {
					Skip(err.Error())
				}

				Expect(err).ToNot(HaveOccurred(), "Failed to update Argo CD clusters app with new git path")

				// Make sure that first ClusterInstance CR should proceed to provisioning the spoke cluster.
				// Test step 1.a expected result validation.
				By("checking cluster instance1 CR reporting correct message and reason")
				clusterInstance1, err := siteconfig.PullClusterInstance(HubAPIClient, RANConfig.Spoke1Name, RANConfig.Spoke1Name)
				Expect(err).ToNot(HaveOccurred(), "Failed to find cluster instance1 custom resource on hub")

				// Test step 1.b expected result validation.
				// I.e. ClusterInstance1 should report a message: Provisioning cluster.
				// with reason: InProgress.
				_, err = clusterInstance1.WaitForCondition(tsparams.CIValidTemplateRefCondition,
					tsparams.SiteconfigOperatorDefaultReconcileTime)
				Expect(err).ToNot(HaveOccurred(), "cluster instance1 failed to report an expected message and reason")

				// Deploy a second ClusterInstance CR named "site-plan-B" with valid template reference.
				// and field clusterName: "value" (=>clusterA) should be same as first ClusterInstance CR.
				// Test step-2: Update the ztp-test git path to reference valid template.
				// in kind:ClusterInstance with field clusterName: "clusterA" (duplicate).
				By("updating the Argo CD clusters app with duplicate cluster name git path")
				exists, err = gitdetails.UpdateArgoCdAppGitPath(tsparams.ArgoCdClustersAppName,
					tsparams.ZtpTestPathDuplicateClusterName, true)
				if !exists {
					Skip(err.Error())
				}

				Expect(err).ToNot(HaveOccurred(), "Failed to update Argo CD clusters app with new git path")

				// Make sure that second ClusterInstance(with duplicated clusterName) would fail the dry-run validation.
				// and reports a message: Rendered manifests failed dry-run validation.
				// Test step 2.a expected result validation.
				By("checking cluster instance2 CR reporting correct error message")
				clusterInstance2, err := siteconfig.PullClusterInstance(HubAPIClient, RANConfig.Spoke2Name, RANConfig.Spoke2Name)
				Expect(err).ToNot(HaveOccurred(), "Failed to find cluster instance2 custom resource on hub")

				// Test step 2.b expected result validation.
				// I.e. ClusterInstance2 reporting a message: Rendered manifests failed dry-run validation
				// with reason: Failed.
				_, err = clusterInstance2.WaitForCondition(tsparams.CIDuplicateClusterNameCondition,
					tsparams.SiteconfigOperatorDefaultReconcileTime)
				Expect(err).ToNot(HaveOccurred(), "cluster instance2 failed to report an expected error message")
			})
	})
