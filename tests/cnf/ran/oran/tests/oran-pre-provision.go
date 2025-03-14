package tests

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/oran"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/raninittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/oran/internal/helper"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/oran/internal/tsparams"
)

var _ = Describe("ORAN Pre-provision Tests", Label(tsparams.LabelPreProvision), func() {
	// 77392 - Apply a ProvisioningRequest referencing an invalid ClusterTemplate
	It("fails to create ProvisioningRequest with invalid ClusterTemplate", reportxml.ID("77392"), func() {
		By("attempting to create a ProvisioningRequest")
		prBuilder := helper.NewProvisioningRequest(HubAPIClient, tsparams.TemplateInvalid)
		_, err := prBuilder.Create()
		Expect(err).To(HaveOccurred(), "Creating a ProvisioningRequest with an invalid ClusterTemplate should fail")
	})

	// 78245 - Missing schema while provisioning without hardware template
	It("fails to provision without a HardwareTemplate when required schema is missing", reportxml.ID("78245"), func() {
		By("verifying the ClusterTemplate validation failed with invalid schema message")
		clusterTemplateName := fmt.Sprintf("%s.%s-%s",
			tsparams.ClusterTemplateName, RANConfig.ClusterTemplateAffix, tsparams.TemplateMissingSchema)
		clusterTemplateNamespace := tsparams.ClusterTemplateName + "-" + RANConfig.ClusterTemplateAffix

		clusterTemplate, err := oran.PullClusterTemplate(HubAPIClient, clusterTemplateName, clusterTemplateNamespace)
		Expect(err).ToNot(HaveOccurred(), "Failed to pull ClusterTemplate with missing schema")

		_, err = clusterTemplate.WaitForCondition(tsparams.CTInvalidSchemaCondition, time.Minute)
		Expect(err).ToNot(HaveOccurred(), "Failed to verify the ClusterTemplate validation failed due to invalid schema")
	})

	When("a ProvisioningRequest is created", func() {
		AfterEach(func() {
			By("deleting the ProvisioningRequest if it exists")
			prBuilder, err := oran.PullPR(HubAPIClient, tsparams.TestPRName)
			if err == nil {
				err := prBuilder.DeleteAndWait(10 * time.Minute)
				Expect(err).ToNot(HaveOccurred(), "Failed to delete the ProvisioningRequest")
			}
		})

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
