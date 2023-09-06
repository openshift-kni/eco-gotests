package tests

import (
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/modules/internal/tsparams"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/kmm"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	. "github.com/openshift-kni/eco-gotests/tests/internal/inittools"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
)

var _ = Describe("KMM", Ordered, Label(tsparams.LabelSuite), func() {

	Context("Module", Label("webhook"), func() {

		var nSpace = tsparams.WebhookModuleTestNamespace

		BeforeAll(func() {

			By("Create Namespace")
			_, err := namespace.NewBuilder(APIClient, nSpace).Create()
			Expect(err).ToNot(HaveOccurred(), "error creating test namespace")
		})

		AfterAll(func() {

			By("Delete Namespace")
			err := namespace.NewBuilder(APIClient, nSpace).Delete()
			Expect(err).ToNot(HaveOccurred(), "error deleting test namespace")
		})

		It("should fail if no container image is specified in the module", polarion.ID("62601"), func() {

			By("Create KernelMapping")
			kernelMapping, err := kmm.NewRegExKernelMappingBuilder("^.+$").BuildKernelMappingConfig()
			Expect(err).ToNot(HaveOccurred(), "error creating kernel mapping")

			By("Create ModuleLoaderContainer")
			moduleLoaderContainerCfg, err := kmm.NewModLoaderContainerBuilder("webhook").
				WithKernelMapping(kernelMapping).
				BuildModuleLoaderContainerCfg()
			Expect(err).ToNot(HaveOccurred(), "error creating moduleloadercontainer")

			By("Create Module")
			_, err = kmm.NewModuleBuilder(APIClient, "webhook-no-container-image", nSpace).
				WithNodeSelector(GeneralConfig.WorkerLabelMap).
				WithModuleLoaderContainer(moduleLoaderContainerCfg).
				Create()
			Expect(err).To(HaveOccurred(), "error creating module")
			Expect(err.Error()).To(ContainSubstring("missing spec.moduleLoader.container.kernelMappings"))
			Expect(err.Error()).To(ContainSubstring(".containerImage"))
		})

		It("should fail if the regexp isn't valid in the module", polarion.ID("62602"), func() {

			By("Create KernelMapping")
			kernelMapping, err := kmm.NewRegExKernelMappingBuilder("*-invalid-regexp").BuildKernelMappingConfig()
			Expect(err).ToNot(HaveOccurred(), "error creating kernel mapping")

			By("Create ModuleLoaderContainer")
			moduleLoaderContainerCfg, err := kmm.NewModLoaderContainerBuilder("webhook").
				WithKernelMapping(kernelMapping).
				BuildModuleLoaderContainerCfg()
			Expect(err).ToNot(HaveOccurred(), "error creating moduleloadercontainer")

			By("Create Module")
			_, err = kmm.NewModuleBuilder(APIClient, "webhook-invalid-regexp", nSpace).
				WithNodeSelector(GeneralConfig.WorkerLabelMap).
				WithModuleLoaderContainer(moduleLoaderContainerCfg).
				Create()
			Expect(err).To(HaveOccurred(), "error creating module")
			Expect(err.Error()).To(ContainSubstring("invalid regexp"))
		})
	})
})
