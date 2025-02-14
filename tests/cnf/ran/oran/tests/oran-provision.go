package tests

import (
	"bytes"
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/configmap"
	"github.com/openshift-kni/eco-goinfra/pkg/ocm"
	"github.com/openshift-kni/eco-goinfra/pkg/oran"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"github.com/openshift-kni/eco-goinfra/pkg/secret"
	"github.com/openshift-kni/eco-goinfra/pkg/siteconfig"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/raninittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/oran/internal/helper"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/oran/internal/tsparams"
	policiesv1 "open-cluster-management.io/governance-policy-propagator/api/v1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ContinueOnError is deliberately left out of this Ordered container. If the invalid ProvisioningRequest does not
// become valid again, we cannot test provisioning with a valid ProvisioningRequest.
var _ = Describe("ORAN Provision Tests", Label(tsparams.LabelProvision), Ordered, func() {
	var prBuilder *oran.ProvisioningRequestBuilder

	AfterAll(func() {
		By("verifying the spoke 1 kubeconfig path is set")
		Expect(RANConfig.Spoke1Kubeconfig).
			ToNot(BeEmpty(), "KUBECONFIG environment variable must be set to save spoke 1 kubeconfig")

		By("saving the spoke 1 admin kubeconfig")
		kubeconfigSecret, err := secret.Pull(HubAPIClient, RANConfig.Spoke1Name+"-admin-kubeconfig", RANConfig.Spoke1Name)
		Expect(err).ToNot(HaveOccurred(), "Failed to get the spoke 1 kubeconfig secret")

		kubeconfig, exists := kubeconfigSecret.Definition.Data["kubeconfig"]
		Expect(exists).To(BeTrue(), "Kubeconfig key does not exist in kubeconfig secret")

		err = os.WriteFile(RANConfig.Spoke1Kubeconfig, kubeconfig, 0644)
		Expect(err).ToNot(HaveOccurred(), "Failed to save the spoke 1 admin kubeconfig")
	})

	// 77393 - Apply a ProvisioningRequest with missing required input parameter
	It("recovers provisioning when invalid ProvisioningRequest is updated", reportxml.ID("77393"), func() {
		By("creating a ProvisioningRequest with invalid policyTemplateParameters")
		prBuilder = helper.NewProvisioningRequest(HubAPIClient, tsparams.TemplateValid).
			WithTemplateParameter(tsparams.PolicyTemplateParamsKey, map[string]any{
				// By using an integer when the schema specifies a string we can create an invalid
				// ProvisioningRequest without being stopped by the webhook.
				tsparams.TestName: 1,
			})

		var err error
		prBuilder, err = prBuilder.Create()
		Expect(err).ToNot(HaveOccurred(), "Failed to create an invalid ProvisioningRequest")

		By("checking the ProvisioningRequest status for a failure")
		prBuilder, err = prBuilder.WaitForCondition(tsparams.PRValidationFailedCondition, time.Minute)
		Expect(err).ToNot(HaveOccurred(), "Failed to wait for the ProvisioningRequest to fail")

		By("updating the ProvisioningRequest with valid policyTemplateParameters")
		prBuilder = prBuilder.WithTemplateParameter(tsparams.PolicyTemplateParamsKey, map[string]any{})
		prBuilder, err = prBuilder.Update()
		Expect(err).ToNot(HaveOccurred(), "Failed to update the ProvisioningRequest to add nodeClusterName")

		By("waiting for ProvisioningRequest validation to succeed")
		prBuilder, err = prBuilder.WaitForCondition(tsparams.PRValidationSucceededCondition, time.Minute)
		Expect(err).ToNot(HaveOccurred(), "Failed to wait for ProvisioningRequest validation to succeed")
	})

	// 77394 - Apply a valid ProvisioningRequest
	It("successfully provisions with a valid ProvisioningRequest", reportxml.ID("77394"), func() {
		By("waiting for the ProvisioningRequest to apply configuration")
		var err error
		prBuilder, err = prBuilder.WaitForCondition(tsparams.PRConfigurationAppliedCondition, 2*time.Hour)
		Expect(err).ToNot(HaveOccurred(), "Failed to wait for the ProvisioningRequest to apply configuration")

		By("waiting for the ProvisioningRequest to be fulfilled")
		prBuilder, err = prBuilder.WaitUntilFulfilled(time.Minute)
		Expect(err).ToNot(HaveOccurred(), "Failed to wait for the ProvisioningRequest to be fulfilled")

		By("verifying a NodePool was created")
		nodePool, err := oran.PullNodePool(HubAPIClient, RANConfig.Spoke1Name, tsparams.HardwareManagerNamespace)
		Expect(err).ToNot(HaveOccurred(), "Failed to pull NodePool for spoke 1")
		Expect(nodePool.Object).ToNot(BeNil(), "Failed to get NodePool object for spoke 1")

		By("verifying the ClusterInstance has the correct BMC details")
		clusterInstance, err := siteconfig.PullClusterInstance(HubAPIClient, RANConfig.Spoke1Name, RANConfig.Spoke1Name)
		Expect(err).ToNot(HaveOccurred(), "Failed to pull ClusterInstance for spoke 1")

		clusterInstanceNode := clusterInstance.Definition.Spec.Nodes[0]
		Expect(clusterInstanceNode.BmcAddress).
			To(ContainSubstring(RANConfig.BMCHosts[0]), "ClusterInstance has incorrect BMC address")

		bmcSecret, err := secret.Pull(HubAPIClient, clusterInstanceNode.BmcCredentialsName.Name, RANConfig.Spoke1Name)
		Expect(err).ToNot(HaveOccurred(), "Failed to pull spoke 1 BMC secret")

		bmcUsername, exists := bmcSecret.Definition.Data["username"]
		Expect(exists).To(BeTrue(), "Username does not exist in BMC secret")

		bmcUsername = bytes.TrimSpace(bmcUsername)
		Expect(string(bmcUsername)).To(Equal(RANConfig.BMCUsername), "ClusterInstance has incorrect BMC username")

		bmcPassword, exists := bmcSecret.Definition.Data["password"]
		Expect(exists).To(BeTrue(), "Passowrd does not exist in BMC secret")

		bmcPassword = bytes.TrimSpace(bmcPassword)
		Expect(string(bmcPassword)).To(Equal(RANConfig.BMCPassword), "ClusterInstance has incorrect BMC password")

		By("verifying spoke 1 pull-secret was created")
		pullSecret, err := secret.Pull(HubAPIClient, "pull-secret", RANConfig.Spoke1Name)
		Expect(err).ToNot(HaveOccurred(), "Failed to pull pull-secret for spoke 1")
		Expect(pullSecret.Object).ToNot(BeNil(), "Failed to get pull-secret object for spoke 1")

		By("verifying spoke 1 extra-manifests was created")
		extraManifests, err := configmap.Pull(HubAPIClient, tsparams.ExtraManifestsName, RANConfig.Spoke1Name)
		Expect(err).ToNot(HaveOccurred(), "Failed to pull extra-manifests ConfigMap for spoke 1")
		Expect(extraManifests.Object).ToNot(BeNil(), "Failed to get extra-manifests ConfigMap object for spoke 1")

		By("verifying spoke 1 policy ConfigMap was created")
		ztpNamespace := fmt.Sprintf("ztp-%s-%s", tsparams.ClusterTemplateName, RANConfig.ClusterTemplateAffix)
		pgMap, err := configmap.Pull(
			HubAPIClient, RANConfig.Spoke1Name+"-pg", ztpNamespace)
		Expect(err).ToNot(HaveOccurred(), "Failed to pull policy ConfigMap for spoke 1")
		Expect(pgMap.Object).ToNot(BeNil(), "Failed to get policy ConfigMap object for spoke 1")

		By("verifying all the policies are compliant")
		err = ocm.WaitForAllPoliciesComplianceState(
			HubAPIClient, policiesv1.Compliant, time.Minute, runtimeclient.ListOptions{Namespace: RANConfig.Spoke1Name})
		Expect(err).ToNot(HaveOccurred(), "Failed to verify all spoke 1 policies are compliant")

	})
})
