package tests

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/oran"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"github.com/openshift-kni/eco-goinfra/pkg/secret"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/raninittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/oran/internal/helper"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/oran/internal/tsparams"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Some test cases are marked as pending from the code. This is used for tests that cannot be used with the loopback
// adaptor and are guaranteed to fail until the production-ready DTIAS is available.

var _ = Describe("ORAN Pre-provision Tests", Label(tsparams.LabelPreProvision), func() {
	// 77386 - Failed authentication with hardware manager
	PIt("fails to authenticate with hardware manager using invalid credentials", reportxml.ID("77386"), func() {
		By("getting a valid Dell HardwareManager")
		hwmgr, err := helper.GetValidDellHwmgr(HubAPIClient)
		Expect(err).ToNot(HaveOccurred(), "Failed to get a valid Dell HardwareManager")

		By("getting the Dell AuthSecret")
		authSecret, err := secret.Pull(
			HubAPIClient, hwmgr.Definition.Spec.DellData.AuthSecret, tsparams.HardwareManagerNamespace)
		Expect(err).ToNot(HaveOccurred(), "Failed to get the HardwareManager AuthSecret")
		Expect(authSecret.Definition.Data).ToNot(BeNil(), "HardwareManager AuthSecret must have data")

		By("copying the secret and updating the password")
		authSecret.Definition.Name += "-test"
		authSecret.Definition.Data["password"] = []byte(tsparams.TestBase64Credential) // wrongpassword

		authSecret, err = authSecret.Create()
		Expect(err).ToNot(HaveOccurred(), "Failed to create the new AuthSecret")

		By("copying the HardwareManager and updating the AuthSecret")
		hwmgr.Definition.Name += "-test"
		hwmgr.Definition.Spec.DellData.AuthSecret = authSecret.Definition.Name

		By("creating the copied HardwareManager")
		hwmgr, err = hwmgr.Create()
		Expect(err).ToNot(HaveOccurred(), "Failed to create the new HardwareManager")

		By("waiting for the authentication to fail")
		hwmgr, err = hwmgr.WaitForCondition(tsparams.HwmgrFailedAuthCondition, time.Minute)
		Expect(err).ToNot(HaveOccurred(), "Failed to wait for the HardwareManager to fail authentication")

		By("deleting the invalid HardwareManager")
		err = hwmgr.Delete()
		Expect(err).ToNot(HaveOccurred(), "Failed to delete the invalid HardwareManager")

		By("deleting the invalid AuthSecret")
		err = authSecret.Delete()
		Expect(err).ToNot(HaveOccurred(), "Failed to delete the invalid AuthSecret")
	})

	// 77392 - Apply a ProvisioningRequest referencing an invalid ClusterTemplate
	It("fails to create ProvisioningRequest with invalid ClusterTemplate", reportxml.ID("77392"), func() {
		By("attempting to create a ProvisioningRequest")
		prBuilder := helper.NewProvisioningRequest(HubAPIClient, tsparams.TemplateInvalid)
		_, err := prBuilder.Create()
		Expect(err).To(HaveOccurred(), "Creating a ProvisioningRequest with an invalid ClusterTemplate should fail")
	})

	DescribeTable("ClusterTemplate failed validations",
		func(templateVersion, message string) {
			By("verifying the ClusterTemplate validation failed with proper message")
			clusterTemplateName := fmt.Sprintf("%s.%s-%s",
				tsparams.ClusterTemplateName, RANConfig.ClusterTemplateAffix, templateVersion)
			clusterTemplateNamespace := tsparams.ClusterTemplateName + "-" + RANConfig.ClusterTemplateAffix
			clusterTemplate, err := oran.PullClusterTemplate(HubAPIClient, clusterTemplateName, clusterTemplateNamespace)
			Expect(err).ToNot(HaveOccurred(), "Failed to pull ClusterTemplate with version %s", templateVersion)

			condition := tsparams.CTValidationFailedCondition
			condition.Message = message

			_, err = clusterTemplate.WaitForCondition(condition, time.Minute)
			Expect(err).ToNot(HaveOccurred(), "Failed to verify the ClusterTemplate validation failed with message %s", message)
		},
		// 77389 - Failed provisioning with missing interface labels
		Entry("fails to provision with missing interface labels",
			reportxml.ID("77389"), tsparams.TemplateMissingLabels, tsparams.CTMissingLabelMessage),
		// 78245 - Missing schema while provisioning without hardware template
		Entry("fails to provision without a HardwareTemplate when required schema is missing",
			reportxml.ID("78245"), tsparams.TemplateMissingSchema, tsparams.CTMissingSchemaMessage),
	)

	When("a ProvisioningRequest is created", func() {
		AfterEach(func() {
			By("deleting the ProvisioningRequest if it exists")
			prBuilder, err := oran.PullPR(HubAPIClient, tsparams.TestPRName)
			if err == nil {
				err := prBuilder.DeleteAndWait(10 * time.Minute)
				Expect(err).ToNot(HaveOccurred(), "Failed to delete the ProvisioningRequest")
			}
		})

		DescribeTable("ProvisionRequest pre-provision validations",
			func(templateVersion string, condition metav1.Condition) {
				By("creating a ProvisioningRequest")
				prBuilder := helper.NewProvisioningRequest(HubAPIClient, templateVersion)
				prBuilder, err := prBuilder.Create()
				Expect(err).ToNot(HaveOccurred(), "Failed to create a ProvisioningRequest")

				By("verifying the ProvisioningRequest has the expected condition")
				_, err = prBuilder.WaitForCondition(condition, time.Minute)
				Expect(err).ToNot(HaveOccurred(), "Failed to verify the ProvisioningRequest status")
			},
			// 77387 - Failed provisioning with nonexistent hardware profile
			PEntry("fails to provision with nonexistent hardware profile",
				reportxml.ID("77387"), tsparams.TemplateNonexistentProfile, tsparams.PRHardwareProvisionFailedCondition),
			// 77388 - Failed provisioning with no hardware available
			Entry("fails to provision with no hardware available",
				reportxml.ID("77388"), tsparams.TemplateNoHardware, tsparams.PRHardwareProvisionFailedCondition),
			// 77390 - Failed provisioning with incorrect boot interface label
			Entry("fails to provision with incorrect boot interface label",
				reportxml.ID("77390"), tsparams.TemplateIncorrectLabel, tsparams.PRNodeConfigFailedCondition),
		)

		// 78246 - Successful provisioning without hardware template
		It("successfully generates ClusterInstance provisioning without HardwareTemplate", reportxml.ID("78246"), func() {
			By("creating a ProvisioningRequest")
			prBuilder := helper.NewNoTemplatePR(HubAPIClient, tsparams.TemplateNoHWTemplate)
			prBuilder, err := prBuilder.Create()
			Expect(err).ToNot(HaveOccurred(), "Failed to create a ProvisioningRequest")

			By("waiting for its ClusterInstance to be processed")
			_, err = prBuilder.WaitForCondition(tsparams.PRCIProcesssedCondition, time.Minute)
			Expect(err).ToNot(HaveOccurred(), "Failed to wait for ClusterInstance to be processed")
		})
	})

})
