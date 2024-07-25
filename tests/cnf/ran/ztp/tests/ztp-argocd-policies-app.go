package tests

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/imageregistry"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/ocm"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"github.com/openshift-kni/eco-goinfra/pkg/serviceaccount"
	"github.com/openshift-kni/eco-goinfra/pkg/storage"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/cluster"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/ranhelper"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/raninittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/ranparam"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/ztp/internal/helper"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/ztp/internal/tsparams"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	policiesv1 "open-cluster-management.io/governance-policy-propagator/api/v1"
)

var _ = Describe("ZTP Argo CD Policies Tests", Label(tsparams.LabelArgoCdPoliciesAppTestCases), func() {
	BeforeEach(func() {
		By("checking the ZTP version")
		versionInRange, err := ranhelper.IsVersionStringInRange(RANConfig.ZTPVersion, "4.10", "")
		Expect(err).ToNot(HaveOccurred(), "Failed to check if ZTP version is in range")

		if !versionInRange {
			Skip("ZTP policies app tests require ZTP version of at least 4.10")
		}
	})

	AfterEach(func() {
		By("resetting the policies app to the original settings")
		err := helper.SetGitDetailsInArgoCd(
			tsparams.ArgoCdPoliciesAppName, tsparams.ArgoCdAppDetails[tsparams.ArgoCdPoliciesAppName], true, false)
		Expect(err).ToNot(HaveOccurred(), "Failed to reset policies app git details")
	})

	When("overriding the PGT policy's compliance and non-compliance intervals", func() {
		// 54241 - User override of policy intervals
		It("should specify new intervals and verify they were applied", reportxml.ID("54241"), func() {
			By("updating Argo CD policies app")
			exists, err := helper.UpdateArgoCdAppGitPath(
				tsparams.ArgoCdPoliciesAppName, tsparams.ZtpTestPathCustomInterval, true)
			if !exists {
				Skip(err.Error())
			}

			Expect(err).ToNot(HaveOccurred(), "Failed to update Argo CD git path")

			By("waiting for policies to be created")
			defaultPolicy, err := helper.WaitForPolicyToExist(
				HubAPIClient,
				tsparams.CustomIntervalDefaultPolicyName,
				tsparams.TestNamespace,
				tsparams.ArgoCdChangeTimeout)
			Expect(err).ToNot(HaveOccurred(), "Failed to wait for default policy to exist")

			overridePolicy, err := helper.WaitForPolicyToExist(
				HubAPIClient,
				tsparams.CustomIntervalOverridePolicyName,
				tsparams.TestNamespace,
				tsparams.ArgoCdChangeTimeout)
			Expect(err).ToNot(HaveOccurred(), "Failed to wait for override policy to exist")

			By("validating the interval on the default policy")
			defaultComplianceInterval, defaultNonComplianceInterval, err := helper.GetPolicyEvaluationIntervals(defaultPolicy)
			Expect(err).ToNot(HaveOccurred(), "Failed to get default policy evaluation intervals")

			Expect(defaultComplianceInterval).To(Equal("1m"))
			Expect(defaultNonComplianceInterval).To(Equal("1m"))

			By("validating the interval on the overridden policy")
			overrideComplianceInterval, overrideNonComplianceInterval, err := helper.GetPolicyEvaluationIntervals(overridePolicy)
			Expect(err).ToNot(HaveOccurred(), "Failed to get override policy evaluation intervals")

			Expect(overrideComplianceInterval).To(Equal("2m"))
			Expect(overrideNonComplianceInterval).To(Equal("2m"))
		})

		// 54242 - Invalid time duration string for user override of policy intervals
		It("should specify an invalid interval format and verify the app error", reportxml.ID("54242"), func() {
			By("updating Argo CD policies app")
			exists, err := helper.UpdateArgoCdAppGitPath(
				tsparams.ArgoCdPoliciesAppName, tsparams.ZtpTestPathInvalidInterval, false)
			if !exists {
				Skip(err.Error())
			}

			Expect(err).ToNot(HaveOccurred(), "Failed to update Argo CD git path")

			By("checking the Argo CD conditions for the expected error")
			expectedMessage := "evaluationInterval.compliant 'time: invalid duration"
			err = helper.WaitForConditionInArgoCdApp(
				HubAPIClient,
				tsparams.ArgoCdPoliciesAppName,
				ranparam.OpenshiftGitOpsNamespace,
				expectedMessage,
				tsparams.ArgoCdChangeTimeout)
			Expect(err).ToNot(HaveOccurred(), "Failed to check Argo CD conditions for expected error")
		})
	})

	When("an image registry is configured on the DU profile", func() {
		var imageRegistryConfig *imageregistry.Builder

		AfterEach(func() {
			// Reset the policies app before doing later restore actions so that they're not affected.
			By("resetting the policies app to the original settings")
			err := helper.SetGitDetailsInArgoCd(
				tsparams.ArgoCdPoliciesAppName, tsparams.ArgoCdAppDetails[tsparams.ArgoCdPoliciesAppName], true, false)
			Expect(err).ToNot(HaveOccurred(), "Failed to reset policies app git details")

			if imageRegistryConfig == nil {
				return
			}

			By("restoring the image registry configs")
			err = helper.RestoreImageRegistry(Spoke1APIClient, tsparams.ImageRegistryName, imageRegistryConfig)
			Expect(err).ToNot(HaveOccurred(), "Failed to restore image registry config")

			By("removing the image registry leftovers if they exist")
			err = helper.CleanupImageRegistryConfig(Spoke1APIClient)
			Expect(err).ToNot(HaveOccurred(), "Failed to clean up image registry leftovers")
		})

		// 54354 - Ability to configure local registry via du profile
		It("verifies the image registry exists", reportxml.ID("54354"), func() {
			By("updating Argo CD policies app")
			exists, err := helper.UpdateArgoCdAppGitPath(
				tsparams.ArgoCdPoliciesAppName, tsparams.ZtpTestPathImageRegistry, true)
			if !exists {
				Skip(err.Error())
			}

			Expect(err).ToNot(HaveOccurred(), "Failed to update Argo CD git path")

			// This test requires that the spoke be configured with the ImageRegistry capability enabled in
			// the ClusterVersion as a precondition. If the ZTP test path exists but the capability is not
			// enabled, this test will fail.
			By("checking if the image registry directory is present on spoke 1")
			_, err = cluster.ExecCommandOnSNO(Spoke1APIClient, 3, fmt.Sprintf("ls %s", tsparams.ImageRegistryPath))
			Expect(err).ToNot(HaveOccurred(), "Image registry directory '%s' does not exist", tsparams.ImageRegistryPath)

			imageRegistryConfig, err = imageregistry.Pull(Spoke1APIClient, tsparams.ImageRegistryName)
			Expect(err).ToNot(HaveOccurred(), "Failed to pull image registry config")

			By("waiting for the policies to exist and be compliant")
			for _, policyName := range tsparams.ImageRegistryPolicies {
				policy, err := helper.WaitForPolicyToExist(
					HubAPIClient, policyName, tsparams.TestNamespace, tsparams.ArgoCdChangeTimeout)
				Expect(err).ToNot(HaveOccurred(), "Failed to wait for policys %s to exist", policyName)

				err = policy.WaitUntilComplianceState(policiesv1.Compliant, tsparams.ArgoCdChangeTimeout)
				Expect(err).ToNot(HaveOccurred(), "Failed to wait for policy %s to be Compliant", policyName)
			}

			By("waiting for the image registry config to be Available")
			imageRegistryBuilder, err := imageregistry.Pull(Spoke1APIClient, tsparams.ImageRegistryName)
			Expect(err).ToNot(HaveOccurred(), "Failed to pull image registry config")

			err = helper.WaitForConditionInImageRegistry(
				imageRegistryBuilder,
				metav1.Condition{Type: "Available", Reason: "Ready", Status: metav1.ConditionTrue},
				tsparams.ArgoCdChangeTimeout)
			Expect(err).ToNot(HaveOccurred(), "Failed to wait for image registry config to be Available")
		})
	})

	When("applying and validating custom source CRs on the DU policies", func() {
		AfterEach(func() {
			By("deleting the policy from spoke if it exists")
			policy, err := ocm.PullPolicy(
				Spoke1APIClient, tsparams.CustomSourceCrPolicyName, tsparams.TestNamespace)
			if err == nil {
				_, err = policy.Delete()
				Expect(err).ToNot(HaveOccurred(), "Failed to delete policy")
			}

			By("deleting the service account from spoke if it exists")
			serviceAccount, err := serviceaccount.Pull(
				Spoke1APIClient, tsparams.CustomSourceCrName, tsparams.CustomSourceTestNamespace)
			if err == nil {
				err := serviceAccount.Delete()
				Expect(err).ToNot(HaveOccurred(), "Failed to delete service account")
			}

			By("deleting custom namespace from spoke if it exists")
			customNamespace, err := namespace.Pull(Spoke1APIClient, tsparams.CustomSourceCrName)
			if err == nil {
				err = customNamespace.DeleteAndWait(3 * time.Minute)
				Expect(err).ToNot(HaveOccurred(), "Failed to delete custom namespace")
			}

			By("deleting storage class from spoke if it exists")
			storageClass, err := storage.PullClass(Spoke1APIClient, tsparams.CustomSourceStorageClass)
			if err == nil {
				err = storageClass.DeleteAndWait(3 * time.Minute)
				Expect(err).ToNot(HaveOccurred(), "Failed to delete storage class")
			}
		})

		// 61978 - Create a new source CR in the user GIT repository
		It("verifies new CR kind that does not exist in ztp container image can be created "+
			"via custom source-cr", reportxml.ID("61978"), func() {
			By("checking service account does not exist on spoke")
			_, err := serviceaccount.Pull(
				Spoke1APIClient, tsparams.CustomSourceCrName, tsparams.CustomSourceTestNamespace)
			Expect(err).To(HaveOccurred(), "Service account already exists before test")

			By("updating Argo CD policies app")
			exists, err := helper.UpdateArgoCdAppGitPath(
				tsparams.ArgoCdPoliciesAppName, tsparams.ZtpTestPathCustomSourceNewCr, true)
			if !exists {
				Skip(err.Error())
			}

			Expect(err).ToNot(HaveOccurred(), "Failed to update Argo CD git path")

			By("waiting for policy to exist")
			policy, err := helper.WaitForPolicyToExist(
				HubAPIClient, tsparams.CustomSourceCrPolicyName, tsparams.TestNamespace, tsparams.ArgoCdChangeTimeout)
			Expect(err).ToNot(HaveOccurred(), "Failed to wait for policy to exist")

			By("waiting for the policy to be Compliant")
			err = policy.WaitUntilComplianceState(policiesv1.Compliant, tsparams.ArgoCdChangeTimeout)
			Expect(err).ToNot(HaveOccurred(), "Failed to wait for policy to be Compliant")

			By("waiting for service account to exist")
			err = helper.WaitForServiceAccountToExist(
				Spoke1APIClient,
				tsparams.CustomSourceCrName,
				tsparams.CustomSourceTestNamespace,
				tsparams.ArgoCdChangeTimeout)
			Expect(err).ToNot(HaveOccurred(), "Failed to wait for service account to exist")
		})

		// 62260 - Same source CR file name
		It("verifies the custom source CR takes precedence over the default source CR with "+
			"the same file name", reportxml.ID("62260"), func() {
			By("checking the ZTP version")
			versionInRange, err := ranhelper.IsVersionStringInRange(RANConfig.ZTPVersion, "4.14", "")
			Expect(err).ToNot(HaveOccurred(), "Failed to check if ZTP version is in range")

			if !versionInRange {
				Skip("This test requires a ZTP version of at least 4.14")
			}

			By("updating Argo CD policies app")
			exists, err := helper.UpdateArgoCdAppGitPath(
				tsparams.ArgoCdPoliciesAppName, tsparams.ZtpTestPathCustomSourceReplaceExisting, true)
			if !exists {
				Skip(err.Error())
			}

			Expect(err).ToNot(HaveOccurred(), "Failed to update Argo CD git path")

			By("waiting for policy to exist")
			policy, err := helper.WaitForPolicyToExist(
				HubAPIClient, tsparams.CustomSourceCrPolicyName, tsparams.TestNamespace, tsparams.ArgoCdChangeTimeout)
			Expect(err).ToNot(HaveOccurred(), "Failed to wait for policy to exist")

			By("waiting for the policy to be Compliant")
			err = policy.WaitUntilComplianceState(policiesv1.Compliant, tsparams.ArgoCdChangeTimeout)
			Expect(err).ToNot(HaveOccurred(), "Failed to wait for policy to be Compliant")

			By("checking the custom namespace exists")
			_, err = namespace.Pull(Spoke1APIClient, tsparams.CustomSourceCrName)
			Expect(err).ToNot(HaveOccurred(), "Failed to pull namespace that should exist")
		})

		// 63516 - Reference non-existence source CR yaml file
		It("verifies a proper error is returned in ArgoCD app when a non-existent "+
			"source-cr is used in PGT", reportxml.ID("63516"), func() {
			By("checking the ZTP version")
			versionInRange, err := ranhelper.IsVersionStringInRange(RANConfig.ZTPVersion, "4.14", "")
			Expect(err).ToNot(HaveOccurred(), "Failed to check if ZTP version is in range")

			if !versionInRange {
				Skip("This test requires a ZTP version of at least 4.14")
			}

			By("updating Argo CD policies app")
			exists, err := helper.UpdateArgoCdAppGitPath(
				tsparams.ArgoCdPoliciesAppName, tsparams.ZtpTestPathCustomSourceNoCrFile, false)
			if !exists {
				Skip(err.Error())
			}

			Expect(err).ToNot(HaveOccurred(), "Failed to update Argo CD git path")

			By("checking the Argo CD conditions for the expected error")
			err = helper.WaitForConditionInArgoCdApp(
				HubAPIClient,
				tsparams.ArgoCdPoliciesAppName,
				ranparam.OpenshiftGitOpsNamespace,
				"test/NoCustomCr.yaml is not found",
				tsparams.ArgoCdChangeTimeout)
			Expect(err).ToNot(HaveOccurred(), "Failed to check Argo CD conditions for expected error")
		})

		// 64407 - Verify source CR search path implementation
		It("verifies custom and default source CRs can be included in the same policy", reportxml.ID("64407"), func() {
			By("checking the ZTP version")
			versionInRange, err := ranhelper.IsVersionStringInRange(RANConfig.ZTPVersion, "4.14", "")
			Expect(err).ToNot(HaveOccurred(), "Failed to check if ZTP version is in range")

			if !versionInRange {
				Skip("This test requires a ZTP version of at least 4.14")
			}

			By("checking service account does not exist on spoke")
			_, err = serviceaccount.Pull(Spoke1APIClient, tsparams.CustomSourceCrName, tsparams.CustomSourceTestNamespace)
			Expect(err).To(HaveOccurred(), "Service account already exists before test")

			By("checking storage class does not exist on spoke")
			_, err = storage.PullClass(Spoke1APIClient, tsparams.CustomSourceStorageClass)
			Expect(err).To(HaveOccurred(), "Storage class already exists before test")

			By("updating Argo CD policies app")
			exists, err := helper.UpdateArgoCdAppGitPath(
				tsparams.ArgoCdPoliciesAppName, tsparams.ZtpTestPathCustomSourceSearchPath, true)
			if !exists {
				Skip(err.Error())
			}

			Expect(err).ToNot(HaveOccurred(), "Failed to update Argo CD git path")

			By("waiting for policy to exist")
			policy, err := helper.WaitForPolicyToExist(
				HubAPIClient, tsparams.CustomSourceCrPolicyName, tsparams.TestNamespace, tsparams.ArgoCdChangeTimeout)
			Expect(err).ToNot(HaveOccurred(), "Failed to wait for policy to exist")

			By("waiting for the policy to be Compliant")
			err = policy.WaitUntilComplianceState(policiesv1.Compliant, tsparams.ArgoCdChangeTimeout)
			Expect(err).ToNot(HaveOccurred(), "Failed to wait for policy to be Compliant")

			By("checking service account exists")
			_, err = serviceaccount.Pull(Spoke1APIClient, tsparams.CustomSourceCrName, tsparams.CustomSourceTestNamespace)
			Expect(err).ToNot(HaveOccurred(), "Failed to check that service account exists")

			By("checking storage class exists")
			_, err = storage.PullClass(Spoke1APIClient, tsparams.CustomSourceStorageClass)
			Expect(err).ToNot(HaveOccurred(), "Failed to check that storage class exists")
		})
	})
})
