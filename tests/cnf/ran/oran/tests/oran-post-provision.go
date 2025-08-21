package tests

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	provisioningv1alpha1 "github.com/openshift-kni/oran-o2ims/api/provisioning/v1alpha1"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/configmap"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/ocm"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/oran"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/reportxml"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/siteconfig"
	. "github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/ran/internal/raninittools"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/ran/oran/internal/helper"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/ran/oran/internal/tsparams"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/internal/cluster"
	"github.com/stmcginnis/gofish/redfish"
	"k8s.io/client-go/util/retry"
	policiesv1 "open-cluster-management.io/governance-policy-propagator/api/v1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("ORAN Post-provision Tests", Label(tsparams.LabelPostProvision), func() {
	var (
		prBuilder      *oran.ProvisioningRequestBuilder
		originalPRSpec *provisioningv1alpha1.ProvisioningRequestSpec
	)

	BeforeEach(func() {
		By("saving the original ProvisioningRequest spec")
		var err error
		prBuilder, err = oran.PullPR(HubAPIClient, tsparams.TestPRName)
		Expect(err).ToNot(HaveOccurred(), "Failed to pull spoke 1 ProvisioningRequest")

		copiedSpec := prBuilder.Definition.Spec
		originalPRSpec = &copiedSpec

		By("verifying ProvisioningRequest is fulfilled to start")
		prBuilder, err = prBuilder.WaitUntilFulfilled(time.Minute)
		Expect(err).ToNot(HaveOccurred(), "Failed to verify spoke 1 ProvisioningRequest is fulfilled")
	})

	AfterEach(func() {
		// If saving the original spec failed, skip restoring it to prevent unnecessary panics.
		if originalPRSpec == nil {
			return
		}

		By("checking spoke 1 power state")
		powerState, err := BMCClient.SystemPowerState()
		Expect(err).ToNot(HaveOccurred(), "Failed to get system power state from spoke 1 BMC")

		By("pulling the ProvisioningRequest again to ensure it is valid")
		prBuilder, err = oran.PullPR(HubAPIClient, tsparams.TestPRName)
		Expect(err).ToNot(HaveOccurred(), "Failed to pull the ProvisioningRequest again")

		By("restoring the original ProvisioningRequest spec")
		prBuilder.Definition.Spec = *originalPRSpec
		prBuilder = updatePRUntilNoConflict(prBuilder)
		Expect(err).ToNot(HaveOccurred(), "Failed to restore spoke 1 ProvisioningRequest")

		By("waiting for original ProvisioningRequest to apply")
		waitForPolicies(prBuilder)

		By("waiting for ProvisioningRequest to be fulfilled")
		prBuilder, err = prBuilder.WaitUntilFulfilled(time.Minute)
		Expect(err).ToNot(HaveOccurred(), "Failed to wait for ProvisioningRequest to become fulfilled")

		By("deleting the second test ConfigMap if it exists")
		err = configmap.NewBuilder(Spoke1APIClient, tsparams.TestName2, tsparams.TestName).Delete()
		Expect(err).ToNot(HaveOccurred(), "Failed to delete second test ConfigMap if it exists")

		By("deleting the test label if it exists")
		removeTestLabelIfExists()

		if powerState != "On" {
			By("waiting for spoke 1 to recover")
			err = cluster.WaitForRecover(Spoke1APIClient, []string{}, 45*time.Minute)
			Expect(err).ToNot(HaveOccurred(), "Failed to wait for spoke 1 to recover")
		}
	})

	// 77373 - Successful update to ProvisioningRequest clusterInstanceParameters
	It("successfully updates clusterInstanceParameters", reportxml.ID("77373"), func() {
		By("verifying the test label does not already exist")
		verifyTestLabelDoesNotExist()

		By("updating the extraLabels in clusterInstanceParameters")
		templateParameters, err := prBuilder.GetTemplateParameters()
		Expect(err).ToNot(HaveOccurred(), "Failed to get spoke 1 TemplateParameters")
		Expect(tsparams.ClusterInstanceParamsKey).
			To(BeKeyOf(templateParameters), "Spoke 1 TemplateParameters is missing clusterInstanceParameters")

		clusterInstanceParams, ok := templateParameters[tsparams.ClusterInstanceParamsKey].(map[string]any)
		Expect(ok).To(BeTrue(), "Spoke 1 clusterInstanceParameters is not a map[string]any")

		clusterInstanceParams["extraLabels"] = map[string]any{"ManagedCluster": map[string]string{tsparams.TestName: ""}}
		prBuilder = prBuilder.WithTemplateParameters(templateParameters)

		prBuilder = updatePRUntilNoConflict(prBuilder)
		waitForLabels()
	})

	// 77374 - Successful update to ProvisioningRequest policyTemplateParameters
	It("successfully updates policyTemplateParameters", reportxml.ID("77374"), func() {
		By("verifying the test ConfigMap exists and has the original value")
		verifyCM(tsparams.TestName, tsparams.TestOriginalValue)

		By("updating the policyTemplateParameters")
		prBuilder = prBuilder.WithTemplateParameter(tsparams.PolicyTemplateParamsKey, map[string]string{
			tsparams.TestName: tsparams.TestNewValue,
		})

		prBuilder = updatePRUntilNoConflict(prBuilder)

		By("waiting to ensure the policy status updates")
		// This test case updates a previously compliant policy so if we check the compliance state too soon we
		// risk a situation where the policy has been updated but its compliance state has not been.
		time.Sleep(15 * time.Second)

		DeferCleanup(func() {
			By("waiting to ensure the policy status updates")
			// The same issue that happens on the cleanup side of this test case, so wait again to ensure
			// the next test case is not affected.
			time.Sleep(15 * time.Second)
		})

		waitForPolicies(prBuilder)

		By("verifying the test ConfigMap has the new value")
		verifyCM(tsparams.TestName, tsparams.TestNewValue)
	})

	// 77375 - Successful update to ClusterInstance defaults ConfigMap
	It("successfully updates ClusterInstance defaults", reportxml.ID("77375"), func() {
		By("verifying the test label does not already exist")
		verifyTestLabelDoesNotExist()

		By("updating the ProvisioningRequest TemplateVersion")
		prBuilder.Definition.Spec.TemplateVersion = RANConfig.ClusterTemplateAffix + "-" + tsparams.TemplateUpdateDefaults
		prBuilder = updatePRUntilNoConflict(prBuilder)

		waitForLabels()
	})

	// 77376 - Successful update of existing PG manifest
	It("successfully updates existing PG manifest", reportxml.ID("77376"), func() {
		By("verifying the test ConfigMap exists and has the original value")
		verifyCM(tsparams.TestName, tsparams.TestOriginalValue)

		By("updating the ProvisioningRequest TemplateVersion")
		prBuilder.Definition.Spec.TemplateVersion = RANConfig.ClusterTemplateAffix + "-" + tsparams.TemplateUpdateExisting
		prBuilder = updatePRUntilNoConflict(prBuilder)

		waitForPolicies(prBuilder)

		By("verifying the test ConfigMap has the new value")
		verifyCM(tsparams.TestName, tsparams.TestNewValue)
	})

	// 77377 - Successful addition of new manifest to existing PG
	It("successfully adds new manifest to existing PG", reportxml.ID("77377"), func() {
		By("verifying the test ConfigMap exists and has the original value")
		verifyCM(tsparams.TestName, tsparams.TestOriginalValue)

		By("verifying the second test ConfigMap does not exist")
		_, err := configmap.Pull(Spoke1APIClient, tsparams.TestName2, tsparams.TestName)
		Expect(err).To(HaveOccurred(), "Second test ConfigMap already exists on spoke 1")

		By("updating the ProvisioningRequest TemplateVersion")
		prBuilder.Definition.Spec.TemplateVersion = RANConfig.ClusterTemplateAffix + "-" + tsparams.TemplateAddNew
		prBuilder = updatePRUntilNoConflict(prBuilder)

		waitForPolicies(prBuilder)

		By("verifying the test ConfigMap has the original value")
		verifyCM(tsparams.TestName, tsparams.TestOriginalValue)

		By("verifying the second test ConfigMap exists and has the original value")
		verifyCM(tsparams.TestName2, tsparams.TestOriginalValue)
	})

	// 77378 - Successful update of ClusterTemplate policyTemplateParameters schema
	//
	// This test will update the TemplateVersion and in doing so update the policyTemplateParameters and the
	// policyTemplateDefaults ConfigMap. Though the policyTemplateParameters are not changed directly, the policy
	// ConfigMap gets updated so the changes are can be verified. The second test ConfigMap is also added as part of
	// the change to the policies, using the new key added to the policyTemplateParameters schema.
	It("successfully updates schema of policyTemplateParameters", reportxml.ID("77378"), func() {
		By("verifying the test ConfigMap exists and has the original value")
		verifyCM(tsparams.TestName, tsparams.TestOriginalValue)

		By("verifying the second test ConfigMap does not exist")
		_, err := configmap.Pull(Spoke1APIClient, tsparams.TestName2, tsparams.TestName)
		Expect(err).To(HaveOccurred(), "Second test ConfigMap already exists on spoke 1")

		By("updating the ProvisioningRequest TemplateVersion")
		prBuilder.Definition.Spec.TemplateVersion = RANConfig.ClusterTemplateAffix + "-" + tsparams.TemplateUpdateSchema
		prBuilder = updatePRUntilNoConflict(prBuilder)

		waitForPolicies(prBuilder)

		By("verifying the test ConfigMap has the original value")
		verifyCM(tsparams.TestName, tsparams.TestOriginalValue)

		By("verifying the second test ConfigMap has the new value")
		verifyCM(tsparams.TestName2, tsparams.TestNewValue)
	})

	// 77379 - Failed update to ProvisioningRequest and successful rollback
	It("successfully rolls back failed ProvisioningRequest update", reportxml.ID("77379"), func() {
		By("updating the policyTemplateParameters")
		prBuilder = prBuilder.WithTemplateParameter(tsparams.PolicyTemplateParamsKey, map[string]string{
			tsparams.HugePagesSizeKey: "2G",
		})
		prBuilder = updatePRUntilNoConflict(prBuilder)

		By("waiting for policy to go NonCompliant")
		err := helper.WaitForNoncompliantImmutable(HubAPIClient, RANConfig.Spoke1Name, time.Minute)
		Expect(err).ToNot(HaveOccurred(), "Failed to wait for a spoke 1 policy to go NonCompliant due to immutable field")

		By("fixing the policyTemplateParameters")
		prBuilder = prBuilder.WithTemplateParameter(tsparams.PolicyTemplateParamsKey, map[string]string{})
		prBuilder = updatePRUntilNoConflict(prBuilder)

		waitForPolicies(prBuilder)
	})

	// 77391 - Successful update of hardware profile
	PIt("successfully updates hardware profile", reportxml.ID("77391"), func() {
		By("verifying spoke 1 is powered on")
		powerState, err := BMCClient.SystemPowerState()
		Expect(err).ToNot(HaveOccurred(), "Failed to get system power state from spoke 1 BMC")
		Expect(powerState).To(Equal("On"), "Spoke 1 is not powered on")

		By("updating ProvisioningRequest TemplateVersion")
		prBuilder.Definition.Spec.TemplateVersion = RANConfig.ClusterTemplateAffix + "-" + tsparams.TemplateUpdateProfile
		prBuilder = updatePRUntilNoConflict(prBuilder)

		By("waiting for spoke 1 to be powered off")
		err = BMCClient.WaitForSystemPowerState(redfish.OffPowerState, 5*time.Minute)
		Expect(err).ToNot(HaveOccurred(), "Failed to wait for spoke 1 to power off")
	})
})

// verifyCM verifies that the test ConfigMap name has value for the test key.
func verifyCM(name, value string) {
	testCM, err := configmap.Pull(Spoke1APIClient, name, tsparams.TestName)
	Expect(err).ToNot(HaveOccurred(), "Failed to pull test ConfigMap %s from spoke 1", name)
	Expect(tsparams.TestName).
		To(BeKeyOf(testCM.Definition.Data), "Test ConfigMap %s on spoke 1 does not have test key", name)
	Expect(testCM.Definition.Data[tsparams.TestName]).
		To(Equal(value), "Test ConfigMap %s on spoke 1 does not have value %s", name, value)
}

// removeTestLabelIfExists removes the test label from the ManagedCluster if it is present.
func removeTestLabelIfExists() {
	mcl, err := ocm.PullManagedCluster(HubAPIClient, RANConfig.Spoke1Name)
	Expect(err).ToNot(HaveOccurred(), "Failed to pull spoke 1 ManagedCluster")

	if _, hasLabel := mcl.Definition.Labels[tsparams.TestName]; !hasLabel {
		return
	}

	delete(mcl.Definition.Labels, tsparams.TestName)

	_, err = mcl.Update()
	Expect(err).ToNot(HaveOccurred(), "Failed to update spoke 1 ManagedCluster to remove test label")
}

// verifyTestLabelDoesNotExist asserts that the spoke 1 ManagedCluster does not have the test label.
func verifyTestLabelDoesNotExist() {
	mcl, err := ocm.PullManagedCluster(HubAPIClient, RANConfig.Spoke1Name)
	Expect(err).ToNot(HaveOccurred(), "Failed to pull spoke 1 ManagedCluster")

	_, hasLabel := mcl.Definition.Labels[tsparams.TestName]
	Expect(hasLabel).To(BeFalse(), "Spoke 1 ManagedCluster has test label when it should not")
}

// waitForLabels waits for the test label to appear on the ClusterInstance then on the ManagedCluster.
func waitForLabels() {
	By("waiting for ClusterInstance to have label")

	clusterInstance, err := siteconfig.PullClusterInstance(HubAPIClient, RANConfig.Spoke1Name, RANConfig.Spoke1Name)
	Expect(err).ToNot(HaveOccurred(), "Failed to pull spoke 1 ClusterInstance")

	_, err = clusterInstance.WaitForExtraLabel("ManagedCluster", tsparams.TestName, time.Minute)
	Expect(err).ToNot(HaveOccurred(), "Failed to wait for spoke 1 ClusterInstance to have the extraLabel")

	By("waiting for ManagedCluster to have label")

	mcl, err := ocm.PullManagedCluster(HubAPIClient, RANConfig.Spoke1Name)
	Expect(err).ToNot(HaveOccurred(), "Failed to pull spoke 1 ManagedCluster")

	_, err = mcl.WaitForLabel(tsparams.TestName, time.Minute)
	Expect(err).ToNot(HaveOccurred(), "Failed to wait for spoke 1 ManagedCluster to have the label")
}

// waitForPolicies waits first for the policies to compliant then for prBuilder to have the ConfigurationApplied
// condition and be fulfilled.
func waitForPolicies(prBuilder *oran.ProvisioningRequestBuilder) {
	// If we do not wait for the policies to propagate to the spoke, we risk checking the old, already compliant
	// policies on the spoke. This will fail tests that rely on new policies updating resources on the spoke.
	By("waiting for policies to propagate to spoke")

	templateName := tsparams.ClusterTemplateName + "." + prBuilder.Definition.Spec.TemplateVersion
	templateNamespace := tsparams.ClusterTemplateName + "-" + RANConfig.ClusterTemplateAffix
	clusterTemplate, err := oran.PullClusterTemplate(HubAPIClient, templateName, templateNamespace)
	Expect(err).ToNot(HaveOccurred(), "Failed to pull ClusterTemplate corresponding to the ProvisioningRequest")

	policyVersion, err := helper.GetPolicyVersionForTemplate(HubAPIClient, clusterTemplate)
	Expect(err).ToNot(HaveOccurred(),
		"Failed to get policy version for TemplateVersion %s", prBuilder.Definition.Spec.TemplateVersion)

	err = helper.WaitForPolicyVersion(Spoke1APIClient, RANConfig.Spoke1Name, policyVersion, 2*time.Minute)
	Expect(err).ToNot(HaveOccurred(), "Failed to wait for policies to propagate to the spoke")

	By("waiting for ProvisioningRequest to have the correct policies")

	err = helper.WaitForPRPolicyVersion(prBuilder, policyVersion, 2*time.Minute)
	Expect(err).ToNot(HaveOccurred(),
		"Failed to wait for the ProvisioningRequest to have the correct policy version %s", policyVersion)

	By("waiting for policies to be compliant")

	err = ocm.WaitForAllPoliciesComplianceState(
		Spoke1APIClient, policiesv1.Compliant, time.Minute, runtimeclient.ListOptions{Namespace: RANConfig.Spoke1Name})
	Expect(err).ToNot(HaveOccurred(), "Failed to wait for spoke 1 policies to be compliant")

	By("verifying the ProvisioningRequest status is updated")

	prBuilder, err = prBuilder.WaitForCondition(tsparams.PRConfigurationAppliedCondition, time.Minute)
	Expect(err).ToNot(HaveOccurred(), "Failed to wait for spoke 1 ProvisioningRequest to have ConfigurationApplied")

	By("verifying the ProvisioningRequest is fulfilled")

	_, err = prBuilder.WaitUntilFulfilled(time.Minute)
	Expect(err).ToNot(HaveOccurred(), "Failed to wait for spoke 1 ProvisioningRequest to be fulfilled")
}

// updatePRUntilNoConflict retries updating the prBuilder until it does not return an error due to conflict. This
// usually happens due to duplicate updates to provisioningStatus by the operator after the ConfigurationApplied
// condition is true.
func updatePRUntilNoConflict(prBuilder *oran.ProvisioningRequestBuilder) *oran.ProvisioningRequestBuilder {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Any error will cause the returned prBuilder to be nil, so ignore the returned builder while retrying.
		_, err := prBuilder.Update()

		return err
	})
	Expect(err).ToNot(HaveOccurred(), "Failed to update ProvisioningRequest until no conflict encountered")

	return prBuilder
}
