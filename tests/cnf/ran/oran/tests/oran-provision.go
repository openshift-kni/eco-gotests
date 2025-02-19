package tests

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang/glog"
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
	// 77393 - Apply a ProvisioningRequest with missing required input parameter
	It("recovers provisioning when invalid ProvisioningRequest is updated", reportxml.ID("77393"), func() {
		By("verifying the ProvisioningRequest does not already exist")
		_, err := oran.PullPR(HubAPIClient, tsparams.TestPRName)
		if err == nil {
			Skip("cannot run provisioning tests if the ProvisioningRequest already exists")
		}

		By("creating a ProvisioningRequest with invalid policyTemplateParameters")
		prBuilder := helper.NewProvisioningRequest(HubAPIClient, tsparams.TemplateValid).
			WithTemplateParameter(tsparams.PolicyTemplateParamsKey, map[string]any{
				// By using an integer when the schema specifies a string we can create an invalid
				// ProvisioningRequest without being stopped by the webhook.
				tsparams.TestName: 1,
			})

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
		_, err = prBuilder.WaitForCondition(tsparams.PRValidationSucceededCondition, time.Minute)
		Expect(err).ToNot(HaveOccurred(), "Failed to wait for ProvisioningRequest validation to succeed")
	})

	When("provisioning with a valid ProvisioningRequest", func() {
		AfterEach(func() {
			if RANConfig.Spoke1Kubeconfig != "" {
				By("saving the spoke 1 admin kubeconfig")
				err := saveSpoke1Secret("-admin-kubeconfig", "kubeconfig", RANConfig.Spoke1Kubeconfig)
				Expect(err).ToNot(HaveOccurred(), "Failed to save spoke 1 admin kubeconfig")
			}

			if RANConfig.Spoke1Password != "" {
				By("saving the spoke 1 admin password")
				err := saveSpoke1Secret("-admin-password", "password", RANConfig.Spoke1Password)
				Expect(err).ToNot(HaveOccurred(), "Failed to save the spoke 1 admin password")
			}
		})

		// 77394 - Apply a valid ProvisioningRequest
		It("successfully provisions and generates the correct resources", reportxml.ID("77394"), func() {
			By("pulling the ProvisioningRequest")
			prBuilder, err := oran.PullPR(HubAPIClient, tsparams.TestPRName)
			if err != nil {
				By("creating the ProvisioningRequest since it does not exist")
				prBuilder, err = helper.NewProvisioningRequest(HubAPIClient, tsparams.TemplateValid).Create()
				Expect(err).ToNot(HaveOccurred(), "Failed to create ProvisioningRequest since it does not exist")
			}

			By("waiting for the ProvisioningRequest to apply configuration")
			prBuilder, err = prBuilder.WaitForCondition(tsparams.PRConfigurationAppliedCondition, 2*time.Hour)
			Expect(err).ToNot(HaveOccurred(), "Failed to wait for the ProvisioningRequest to apply configuration")

			By("waiting for the ProvisioningRequest to be fulfilled")
			_, err = prBuilder.WaitUntilFulfilled(time.Minute)
			Expect(err).ToNot(HaveOccurred(), "Failed to wait for the ProvisioningRequest to be fulfilled")

			By("verifying provisioning succeeded")
			err = verifySpokeProvisioning()
			Expect(err).ToNot(HaveOccurred(), "Failed to verify spoke provisioning succeeded")
		})
	})
})

// saveSpoke1Secret will write the value of key in secret RANConfig.Spoke1Name+suffix to fileName, truncating if the
// file exists, otherwise saving with permissions 644.
func saveSpoke1Secret(suffix, key, fileName string) error {
	spokeSecret, err := secret.Pull(HubAPIClient, RANConfig.Spoke1Name+suffix, RANConfig.Spoke1Name)
	if err != nil {
		return err
	}

	value, exists := spokeSecret.Object.Data[key]
	if !exists {
		return fmt.Errorf("unable to save key %s in secret %s: key does not exist", key, RANConfig.Spoke1Name+suffix)
	}

	return os.WriteFile(fileName, value, 0644)
}

// verifySpokeProvisioning ensures that for a provisioned spoke, its NodePool exists, its BMC details are correct, the
// pull-secret and extra-manifests exist, and finally that its policies are compliant, in that order. Errors are
// accumulated for each validation and returned so that every one of the validations will run and be logged.
func verifySpokeProvisioning() error {
	var accumulatedErrors []error

	By("verifying a NodePool was created")

	_, err := oran.PullNodePool(HubAPIClient, RANConfig.Spoke1Name, tsparams.HardwareManagerNamespace)
	if err != nil {
		glog.V(tsparams.LogLevel).Infof("Failed to verify a NodePool was created: %v", err)

		accumulatedErrors = append(accumulatedErrors, fmt.Errorf("failed to verify a NodePool was created: %w", err))
	}

	By("verifying the ClusterInstance has the correct BMC details")

	err = verifyBMCDetails()
	if err != nil {
		glog.V(tsparams.LogLevel).Infof("Failed to verify the ClusterInstance BMC details: %v", err)

		accumulatedErrors = append(accumulatedErrors,
			fmt.Errorf("failed to verify the ClusterInstance BMC details: %w", err))
	}

	By("verifying spoke 1 pull-secret was created")

	_, err = secret.Pull(HubAPIClient, "pull-secret", RANConfig.Spoke1Name)
	if err != nil {
		glog.V(tsparams.LogLevel).Infof("Failed to verify the pull-secret was created: %v", err)

		accumulatedErrors = append(accumulatedErrors, fmt.Errorf("failed to verify the pull-secret was created: %w", err))
	}

	By("verifying spoke 1 extra-manifests was created")

	_, err = configmap.Pull(HubAPIClient, tsparams.ExtraManifestsName, RANConfig.Spoke1Name)
	if err != nil {
		glog.V(tsparams.LogLevel).Infof("Failed to verify the extra-manifests ConfigMap was created: %v", err)

		accumulatedErrors = append(accumulatedErrors,
			fmt.Errorf("failed to verify the extra-manifests ConfigMap was created: %w", err))
	}

	By("verifying spoke 1 policy ConfigMap was created")

	ztpNamespace := fmt.Sprintf("ztp-%s-%s", tsparams.ClusterTemplateName, RANConfig.ClusterTemplateAffix)
	_, err = configmap.Pull(HubAPIClient, RANConfig.Spoke1Name+"-pg", ztpNamespace)

	if err != nil {
		glog.V(tsparams.LogLevel).Infof("Failed to verify spoke 1 policy ConfigMap was created: %v", err)

		accumulatedErrors = append(accumulatedErrors,
			fmt.Errorf("failed to verify spoke 1 policy ConfigMap was created: %w", err))
	}

	By("verifying all the policies are compliant")

	err = ocm.WaitForAllPoliciesComplianceState(
		HubAPIClient, policiesv1.Compliant, time.Minute, runtimeclient.ListOptions{Namespace: RANConfig.Spoke1Name})
	if err != nil {
		glog.V(tsparams.LogLevel).Infof("Failed to verify all policies are compliant: %v", err)

		accumulatedErrors = append(accumulatedErrors, fmt.Errorf("failed to verify all policies are compliant: %w", err))
	}

	return errors.Join(accumulatedErrors...)
}

// verifyBMCDetails ensures that the BMC address, username, and password for the spoke 1 ClusterInstance match the
// configured values.
func verifyBMCDetails() error {
	clusterInstance, err := siteconfig.PullClusterInstance(HubAPIClient, RANConfig.Spoke1Name, RANConfig.Spoke1Name)
	if err != nil {
		return fmt.Errorf("failed to pull ClusterInstance for spoke 1: %w", err)
	}

	clusterInstanceNode := clusterInstance.Definition.Spec.Nodes[0]
	if !strings.Contains(clusterInstanceNode.BmcAddress, RANConfig.BMCHosts[0]) {
		return fmt.Errorf("clusterInstance has incorrect BMC address: %s", clusterInstanceNode.BmcAddress)
	}

	bmcSecret, err := secret.Pull(HubAPIClient, clusterInstanceNode.BmcCredentialsName.Name, RANConfig.Spoke1Name)
	if err != nil {
		return fmt.Errorf("failed to pull spoke 1 BMC secret: %w", err)
	}

	bmcUsername, exists := bmcSecret.Definition.Data["username"]
	if !exists {
		return fmt.Errorf("username key does not appear in ClusterInstance BMC secret data")
	}

	bmcUsername = bytes.TrimSpace(bmcUsername)
	if string(bmcUsername) != RANConfig.BMCUsername {
		return fmt.Errorf("clusterInstance BMC username %s does not match expected username %s",
			string(bmcUsername), RANConfig.BMCUsername)
	}

	bmcPassword, exists := bmcSecret.Definition.Data["password"]
	if !exists {
		return fmt.Errorf("password key does not appear in ClusterInstance BMC secret data")
	}

	bmcPassword = bytes.TrimSpace(bmcPassword)
	if string(bmcPassword) != RANConfig.BMCPassword {
		return fmt.Errorf("clusterInstance BMC password does not match the expected password")
	}

	return nil
}
