package tests

import (
	"time"

	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/internal/get"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/kmm"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/internal/await"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/internal/check"
	. "github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/internal/kmminittools"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/internal/kmmparams"
	. "github.com/openshift-kni/eco-gotests/tests/internal/inittools"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
)

var _ = Describe("KMM", Ordered, Label(kmmparams.LabelSuite, kmmparams.LabelSanity), func() {

	Context("Module", Label("use-global-pull-secret"), func() {

		moduleName := "intel-dgpu"
		kmodName := "i915"

		BeforeAll(func() {
			By("Checking cluster architecture")
			arch, err := get.ClusterArchitecture(APIClient, GeneralConfig.WorkerLabelMap)
			if err != nil {
				Skip("could not detect cluster architecture")
			}
			if arch != "amd64" {
				Skip("test only supported on amd64 cpu architecture")
			}

			By("Check if IMAGE is specified")
			if ModulesConfig.GlobalPullSecretImage == "" {
				Skip("No GlobalPullSecretImage defined. Skipping test")
			}

		})

		AfterAll(func() {

			By("Delete Module")
			_, err := kmm.NewModuleBuilder(APIClient, moduleName, kmmparams.KmmOperatorNamespace).Delete()
			Expect(err).ToNot(HaveOccurred(), "error deleting module")

			By("Await module to be deleted")
			err = await.ModuleObjectDeleted(APIClient, moduleName, kmmparams.KmmOperatorNamespace, time.Minute)
			Expect(err).ToNot(HaveOccurred(), "error while waiting module to be deleted")

		})

		It("should use global pull-secret to pull images", polarion.ID("71694"), func() {

			By("Create KernelMapping")
			kernelMapping := kmm.NewRegExKernelMappingBuilder("^.*\\.x86_64$")

			kernelMapping.WithContainerImage(ModulesConfig.GlobalPullSecretImage)
			kerMapOne, err := kernelMapping.BuildKernelMappingConfig()
			Expect(err).ToNot(HaveOccurred(), "error creating kernel mapping")

			By("Create ModuleLoaderContainer")
			moduleLoaderContainer := kmm.NewModLoaderContainerBuilder(kmodName)
			moduleLoaderContainer.WithKernelMapping(kerMapOne)
			moduleLoaderContainer.WithModprobeSpec("", "/firmware", []string{}, []string{}, []string{}, []string{})
			moduleLoaderContainer.WithImagePullPolicy("Always")
			moduleLoaderContainerCfg, err := moduleLoaderContainer.BuildModuleLoaderContainerCfg()
			Expect(err).ToNot(HaveOccurred(), "error creating moduleloadercontainer")

			By("Create Module")
			module := kmm.NewModuleBuilder(APIClient, moduleName, kmmparams.KmmOperatorNamespace).
				WithNodeSelector(GeneralConfig.WorkerLabelMap)
			module = module.WithModuleLoaderContainer(moduleLoaderContainerCfg)
			_, err = module.Create()
			Expect(err).ToNot(HaveOccurred(), "error creating module")

			By("Await driver container deployment")
			err = await.ModuleDeployment(APIClient, moduleName, kmmparams.KmmOperatorNamespace, 5*time.Minute,
				GeneralConfig.WorkerLabelMap)
			Expect(err).ToNot(HaveOccurred(), "error while waiting on driver deployment")

			By("Check module is loaded on node")
			err = check.ModuleLoaded(APIClient, kmodName, time.Minute)
			Expect(err).ToNot(HaveOccurred(), "error while checking the module is loaded")

			By("Check label is set on all nodes")
			_, err = check.NodeLabel(APIClient, moduleName, kmmparams.KmmOperatorNamespace,
				GeneralConfig.WorkerLabelMap)
			Expect(err).ToNot(HaveOccurred(), "error while checking the module is loaded")
		})
	})
})
