package tests

import (
	"fmt"
	"time"

	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/modules/internal/await"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/modules/internal/check"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/modules/internal/define"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/modules/internal/tsparams"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/configmap"
	"github.com/openshift-kni/eco-goinfra/pkg/kmm"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/serviceaccount"
	. "github.com/openshift-kni/eco-gotests/tests/internal/inittools"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
)

var _ = Describe("KMM", Ordered, Label(tsparams.LabelSuite), func() {

	Context("Module", Label("use-dtk"), func() {

		var testNamespace *namespace.Builder

		moduleName := tsparams.UseDtkModuleTestNamespace
		kmodName := "use-dtk"
		serviceAccountName := "dtk-manager"
		image := fmt.Sprintf("%s/%s/%s:$KERNEL_FULL_VERSION",
			tsparams.LocalImageRegistry, tsparams.UseDtkModuleTestNamespace, kmodName)
		buildArgValue := fmt.Sprintf("%s.o", kmodName)

		BeforeAll(func() {

			By("Create Namespace")
			var err error
			testNamespace, err = namespace.NewBuilder(APIClient, tsparams.UseDtkModuleTestNamespace).Create()
			Expect(err).ToNot(HaveOccurred(), "error creating test namespace")

		})

		AfterAll(func() {
			By("Delete Module")
			_, err := kmm.NewModuleBuilder(APIClient, moduleName, tsparams.UseDtkModuleTestNamespace).Delete()
			Expect(err).ToNot(HaveOccurred(), "error creating test namespace")

			svcAccount := serviceaccount.NewBuilder(APIClient, serviceAccountName, tsparams.UseDtkModuleTestNamespace)
			svcAccount.Exists()

			By("Delete ClusterRoleBinding")
			crb := define.ModuleCRB(*svcAccount, kmodName)
			err = crb.Delete()
			Expect(err).ToNot(HaveOccurred(), "error creating test namespace")

			By("Delete Namespace")
			err = namespace.NewBuilder(APIClient, tsparams.UseDtkModuleTestNamespace).Delete()
			Expect(err).ToNot(HaveOccurred(), "error creating test namespace")
		})

		It("should use DTK_AUTO parameter", polarion.ID("54283"), func() {

			configmapContents := define.MultiStageConfigMapContent(kmodName)

			By("Create ConfigMap")
			dockerfileConfigMap, err := configmap.
				NewBuilder(APIClient, kmodName, testNamespace.Object.Name).
				WithData(configmapContents).Create()
			Expect(err).ToNot(HaveOccurred(), "error creating configmap")

			By("Create ServiceAccount")
			svcAccount, err := serviceaccount.
				NewBuilder(APIClient, serviceAccountName, tsparams.UseDtkModuleTestNamespace).Create()
			Expect(err).ToNot(HaveOccurred(), "error creating serviceaccount")

			By("Create ClusterRoleBinding")
			crb := define.ModuleCRB(*svcAccount, kmodName)
			_, err = crb.Create()
			Expect(err).ToNot(HaveOccurred(), "error creating clusterrolebinding")

			By("Create KernelMapping")
			kernelMapping := kmm.NewRegExKernelMappingBuilder("^.+$")

			kernelMapping.WithContainerImage(image).
				WithBuildArg(tsparams.BuildArgName, buildArgValue).
				WithBuildDockerCfgFile(dockerfileConfigMap.Object.Name)
			kerMapOne, err := kernelMapping.BuildKernelMappingConfig()
			Expect(err).ToNot(HaveOccurred(), "error creating kernel mapping")

			By("Create ModuleLoaderContainer")
			moduleLoaderContainer := kmm.NewModLoaderContainerBuilder(kmodName)
			moduleLoaderContainer.WithKernelMapping(kerMapOne)
			moduleLoaderContainer.WithImagePullPolicy("Always")
			moduleLoaderContainerCfg, err := moduleLoaderContainer.BuildModuleLoaderContainerCfg()
			Expect(err).ToNot(HaveOccurred(), "error creating moduleloadercontainer")

			By("Create Module")
			module := kmm.NewModuleBuilder(APIClient, moduleName, tsparams.UseDtkModuleTestNamespace).
				WithNodeSelector(GeneralConfig.WorkerLabelMap)
			module = module.WithModuleLoaderContainer(moduleLoaderContainerCfg).
				WithLoadServiceAccount(svcAccount.Object.Name)
			_, err = module.Create()
			Expect(err).ToNot(HaveOccurred(), "error creating module")

			By("Await build pod to complete build")
			err = await.BuildPodCompleted(APIClient, tsparams.UseDtkModuleTestNamespace, 5*time.Minute)
			Expect(err).ToNot(HaveOccurred(), "error while building module")

			By("Await driver container deployment")
			err = await.ModuleDeployment(APIClient, moduleName, tsparams.UseDtkModuleTestNamespace, time.Minute,
				GeneralConfig.WorkerLabelMap)
			Expect(err).ToNot(HaveOccurred(), "error while waiting on driver deployment")

			By("Check module is loaded on node")
			err = check.ModuleLoaded(APIClient, kmodName, tsparams.UseDtkModuleTestNamespace, time.Minute)
			Expect(err).ToNot(HaveOccurred(), "error while checking the module is loaded")

			By("Check label is set on all nodes")
			_, err = check.NodeLabel(APIClient, moduleName, GeneralConfig.WorkerLabelMap)
			Expect(err).ToNot(HaveOccurred(), "error while checking the module is loaded")
		})

		It("should be able to modify a kmod in a module", polarion.ID("53466"), func() {

			const newKmod = "kmm_ci_a"

			By("Getting the module")
			moduleBuilder, err := kmm.Pull(APIClient, moduleName, moduleName)
			Expect(err).ToNot(HaveOccurred(), "error getting the module")

			By("Modifying the kmod in the module and re-applying the module")
			moduleBuilder.Object.Spec.ModuleLoader.Container.Modprobe.ModuleName = newKmod
			_, err = moduleBuilder.Update()
			Expect(err).ToNot(HaveOccurred(), "error updating the module")

			By("Wait for old pods to terminate")
			time.Sleep(time.Minute)

			By("Await new driver container deployment")
			err = await.ModuleDeployment(APIClient, moduleName, tsparams.UseDtkModuleTestNamespace, time.Minute,
				GeneralConfig.WorkerLabelMap)
			Expect(err).ToNot(HaveOccurred(), "error while waiting on driver deployment")

			By("Check new kmod was loaded to node")
			err = check.ModuleLoaded(APIClient, newKmod, tsparams.UseDtkModuleTestNamespace, time.Minute)
			Expect(err).ToNot(HaveOccurred(), "error while checking the module is loaded")

		})
	})
})
