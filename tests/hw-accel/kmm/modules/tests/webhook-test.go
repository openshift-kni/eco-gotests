package tests

import (
	"fmt"

	"github.com/rh-ecosystem-edge/eco-gotests/tests/hw-accel/kmm/modules/internal/tsparams"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/kmm"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/namespace"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/reportxml"
	. "github.com/rh-ecosystem-edge/eco-gotests/tests/internal/inittools"
)

var _ = Describe("KMM", Ordered, Label(tsparams.LabelSuite, tsparams.LabelSanity), func() {

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

		Context("KernelMapping", Label("webhook"), func() {

			It("should fail if no container image is specified in the module", reportxml.ID("62601"), func() {

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

			It("should fail if the regexp isn't valid in the module", reportxml.ID("62602"), func() {

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

			It("should fail if no regexp nor literal are set in a kernel mapping", reportxml.ID("62603"), func() {

				By("Create KernelMapping")
				kernelMapping, err := kmm.NewRegExKernelMappingBuilder("willBeRemoved").BuildKernelMappingConfig()
				Expect(err).ToNot(HaveOccurred(), "error creating kernel mapping")
				kernelMapping.Regexp = ""

				By("Create ModuleLoaderContainer")
				moduleLoaderContainerCfg, err := kmm.NewModLoaderContainerBuilder("webhook").
					WithKernelMapping(kernelMapping).
					BuildModuleLoaderContainerCfg()
				Expect(err).ToNot(HaveOccurred(), "error creating moduleloadercontainer")

				By("Create Module")
				_, err = kmm.NewModuleBuilder(APIClient, "webhook-regexp-and-literal", nSpace).
					WithNodeSelector(GeneralConfig.WorkerLabelMap).
					WithModuleLoaderContainer(moduleLoaderContainerCfg).
					Create()
				Expect(err).To(HaveOccurred(), "error creating module")
				Expect(err.Error()).To(ContainSubstring("regexp or literal must be set"))
			})

			It("should fail if both regexp and literal are set in a kernel mapping", reportxml.ID("62604"), func() {

				By("Create KernelMapping")
				kernelMapping, err := kmm.NewRegExKernelMappingBuilder("^.+$").BuildKernelMappingConfig()
				Expect(err).ToNot(HaveOccurred(), "error creating kernel mapping")
				kernelMapping.Literal = "5.14.0-284.28.1.el9_2.x86_64"

				By("Create ModuleLoaderContainer")
				moduleLoaderContainerCfg, err := kmm.NewModLoaderContainerBuilder("webhook").
					WithKernelMapping(kernelMapping).
					BuildModuleLoaderContainerCfg()
				Expect(err).ToNot(HaveOccurred(), "error creating moduleloadercontainer")

				By("Create Module")
				_, err = kmm.NewModuleBuilder(APIClient, "webhook-regexp-and-literal", nSpace).
					WithNodeSelector(GeneralConfig.WorkerLabelMap).
					WithModuleLoaderContainer(moduleLoaderContainerCfg).
					Create()
				Expect(err).To(HaveOccurred(), "error creating module")
				Expect(err.Error()).To(ContainSubstring("regexp and literal are mutually exclusive properties"))
			})
		})

		Context("Modprobe", Label("webhook"), func() {

			It("should fail if there are duplications in modulesLoaindOrder", reportxml.ID("62742"), func() {

				By("Create KernelMapping")
				image := fmt.Sprintf("%s/%s/%s:$KERNEL_FULL_VERSION",
					tsparams.LocalImageRegistry, tsparams.WebhookModuleTestNamespace, "my-kmod")
				kernelMapping, err := kmm.NewRegExKernelMappingBuilder("^.+$").
					WithContainerImage(image).
					BuildKernelMappingConfig()
				Expect(err).ToNot(HaveOccurred(), "error creating kernel mapping")

				By("Create moduleLoader container")
				moduleLoader, err := kmm.NewModLoaderContainerBuilder("kmod-a").
					WithModprobeSpec("", "", nil, nil, nil, []string{"kmod-a", "kmod-b", "kmod-a"}).
					WithKernelMapping(kernelMapping).
					BuildModuleLoaderContainerCfg()
				Expect(err).ToNot(HaveOccurred(), "error creating moduleloadercontainer")

				By("Create Module")
				_, err = kmm.NewModuleBuilder(APIClient, "webhook-module-loading-order-dups", nSpace).
					WithNodeSelector(GeneralConfig.WorkerLabelMap).
					WithModuleLoaderContainer(moduleLoader).
					Create()
				Expect(err).To(HaveOccurred(), "error creating module")
				Expect(err.Error()).To(ContainSubstring("duplicate value in the loading order list"))
			})

			It("should fail if the 'main' kmod isn't the first one in modulesLoadingOrder", reportxml.ID("64227"), func() {

				By("Create KernelMapping")
				image := fmt.Sprintf("%s/%s/%s:$KERNEL_FULL_VERSION",
					tsparams.LocalImageRegistry, tsparams.WebhookModuleTestNamespace, "my-kmod")
				kernelMapping, err := kmm.NewRegExKernelMappingBuilder("^.+$").
					WithContainerImage(image).
					BuildKernelMappingConfig()
				Expect(err).ToNot(HaveOccurred(), "error creating kernel mapping")

				By("Create moduleLoader container")
				moduleLoader, err := kmm.NewModLoaderContainerBuilder("kmod-a").
					WithModprobeSpec("", "", nil, nil, nil, []string{"kmod-b", "kmod-a", "kmod-c"}).
					WithKernelMapping(kernelMapping).
					BuildModuleLoaderContainerCfg()
				Expect(err).ToNot(HaveOccurred(), "error creating moduleloadercontainer")

				By("Create Module")
				_, err = kmm.NewModuleBuilder(APIClient, "main-module-not-first-in-list", nSpace).
					WithNodeSelector(GeneralConfig.WorkerLabelMap).
					WithModuleLoaderContainer(moduleLoader).
					Create()
				Expect(err).To(HaveOccurred(), "error creating module")
				Expect(err.Error()).To(ContainSubstring("if a loading order is defined, the first element must be moduleName"))
			})
		})
	})
})
