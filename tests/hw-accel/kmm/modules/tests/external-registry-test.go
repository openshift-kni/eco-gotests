package tests

import (
	"fmt"
	"log"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/configmap"
	"github.com/openshift-kni/eco-goinfra/pkg/kmm"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/secret"
	"github.com/openshift-kni/eco-goinfra/pkg/serviceaccount"
	. "github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/internal/kmminittools"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/modules/internal/await"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/modules/internal/check"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/modules/internal/define"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/modules/internal/tsparams"
	. "github.com/openshift-kni/eco-gotests/tests/internal/inittools"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
	v1 "k8s.io/api/core/v1"
)

var _ = Describe("KMM", Ordered, Label(tsparams.LabelSuite, tsparams.LabelSanity), func() {

	Context("Module", Label("simple-kmod"), func() {

		moduleName := "simple-kmod"
		kmodName := "simple-kmod"
		localNsName := tsparams.SimpleKmodModuleTestNamespace
		serviceAccountName := "simple-kmod-manager"
		secretName := "ocp-edge-qe-build-secret"
		image := fmt.Sprintf("%s/%s:$KERNEL_FULL_VERSION-%v",
			ModulesConfig.Registry, moduleName, time.Now().Unix())
		buildArgValue := fmt.Sprintf("%s.o", kmodName)

		var module *kmm.ModuleBuilder

		var svcAccount *serviceaccount.Builder

		BeforeAll(func() {
			if ModulesConfig.PullSecret == "" {
				Skip("No external registry secret found in environment, Skipping test")
			}

			By("Create Namespace")
			_, err := namespace.NewBuilder(APIClient, localNsName).Create()
			Expect(err).NotTo(HaveOccurred(), "error creating test namespace")

			By("creating registry secret")

			secretContent := define.SecretContent(ModulesConfig.Registry, ModulesConfig.PullSecret)

			_, err = secret.NewBuilder(APIClient, secretName,
				localNsName, v1.SecretTypeDockerConfigJson).WithData(secretContent).Create()

			Expect(err).ToNot(HaveOccurred(), "failed creating secret")

		})
		It("should build and push image to quay", polarion.ID("53584"), func() {

			By("Create configmap")
			configmapContent := define.SimpleKmodConfigMapContents()

			dockerfileConfigMap, err := configmap.NewBuilder(APIClient, kmodName, localNsName).
				WithData(configmapContent).Create()

			Expect(err).ToNot(HaveOccurred(), "error creating configmap")

			By("Create service account")
			svcAccount, err = serviceaccount.NewBuilder(APIClient, serviceAccountName, localNsName).Create()

			Expect(err).ToNot(HaveOccurred(), "error creating serviceaccount")

			By("Create clusterrolebinding")
			crb := define.ModuleCRB(*svcAccount, moduleName)

			_, err = crb.Create()
			Expect(err).ToNot(HaveOccurred(), "error creating clusterrolebinding")

			By("Create kernel mapping")
			kernelMapping := kmm.NewRegExKernelMappingBuilder("^.+$")

			kernelMapping.WithContainerImage(image).
				WithBuildArg(tsparams.BuildArgName, buildArgValue).
				WithBuildDockerCfgFile(dockerfileConfigMap.Object.Name)
			kerMapOne, err := kernelMapping.BuildKernelMappingConfig()
			Expect(err).ToNot(HaveOccurred(), "error creating kernel mapping")

			By("Create Module LoaderContainer")
			moduleLoaderContainer := kmm.NewModLoaderContainerBuilder(moduleName)
			moduleLoaderContainer.WithKernelMapping(kerMapOne)
			moduleLoaderContainer.WithImagePullPolicy("Always")
			moduleLoaderContainerCfg, err := moduleLoaderContainer.BuildModuleLoaderContainerCfg()
			Expect(err).ToNot(HaveOccurred(), "error creating moduleloadercontainer")

			By("Create module")
			module = kmm.NewModuleBuilder(APIClient, moduleName, localNsName).
				WithNodeSelector(GeneralConfig.WorkerLabelMap)

			module = module.WithImageRepoSecret(secretName)

			module = module.WithModuleLoaderContainer(moduleLoaderContainerCfg).
				WithLoadServiceAccount(svcAccount.Object.Name)
			_, err = module.Create()
			Expect(err).ToNot(HaveOccurred(), "error creating module")

			By("Await build pod to complete build")
			err = await.BuildPodCompleted(APIClient, localNsName, 5*time.Minute)
			Expect(err).ToNot(HaveOccurred(), "error while building module")

			By("Await driver container deployment")
			err = await.ModuleDeployment(APIClient, moduleName, localNsName, time.Minute, GeneralConfig.WorkerLabelMap)
			Expect(err).ToNot(HaveOccurred(), "error while waiting on driver deployment")

			By("Check module is loaded on node")
			err = check.ModuleLoaded(APIClient, kmodName, time.Minute)
			Expect(err).ToNot(HaveOccurred(), "error while checking the module is loaded")

			By("Check label is set on all nodes")
			_, err = check.NodeLabel(APIClient, kmodName, localNsName, GeneralConfig.WorkerLabelMap)
			Expect(err).ToNot(HaveOccurred(), "error while checking the module is loaded")

		})

		It("should delete simple-kmod module", polarion.ID("53413"), func() {
			By("Deleting the module")
			_, err := module.Delete()
			Expect(err).ToNot(HaveOccurred(), "error deleting the module")

			By("Await pods deletion")
			err = await.ModuleUndeployed(APIClient, localNsName, time.Minute)
			Expect(err).ToNot(HaveOccurred(), "error while waiting pods to be deleted")

			By("Check labels are removed on all nodes")
			_, err = check.NodeLabel(APIClient, kmodName, localNsName, GeneralConfig.WorkerLabelMap)
			log.Printf("error is: %v", err)
			Expect(err).To(HaveOccurred(), "error while checking the module is loaded")

		})

		It("should deploy prebuild image", polarion.ID("53395"), func() {

			By("Create kernel mapping")
			kernelMapping := kmm.NewRegExKernelMappingBuilder("^.+$")

			kernelMapping.WithContainerImage(image)

			kerMapOne, err := kernelMapping.BuildKernelMappingConfig()
			Expect(err).ToNot(HaveOccurred(), "error creating kernel mapping")

			By("Create Module LoaderContainer")
			moduleLoaderContainer := kmm.NewModLoaderContainerBuilder(moduleName)
			moduleLoaderContainer.WithKernelMapping(kerMapOne)
			moduleLoaderContainer.WithImagePullPolicy("Always")
			moduleLoaderContainerCfg, err := moduleLoaderContainer.BuildModuleLoaderContainerCfg()
			Expect(err).ToNot(HaveOccurred(), "error creating moduleloadercontainer")

			By("Create module")
			module = kmm.NewModuleBuilder(APIClient, moduleName, localNsName).
				WithNodeSelector(GeneralConfig.WorkerLabelMap)

			module = module.WithImageRepoSecret(secretName)

			module = module.WithModuleLoaderContainer(moduleLoaderContainerCfg).
				WithLoadServiceAccount(svcAccount.Object.Name)
			_, err = module.Create()
			Expect(err).ToNot(HaveOccurred(), "error creating module")

			By("Await driver container deployment")
			err = await.ModuleDeployment(APIClient, moduleName, localNsName, time.Minute, GeneralConfig.WorkerLabelMap)
			Expect(err).ToNot(HaveOccurred(), "error while waiting on driver deployment")

			By("Check module is loaded on node")
			err = check.ModuleLoaded(APIClient, kmodName, time.Minute)
			Expect(err).ToNot(HaveOccurred(), "error while checking the module is loaded")

			By("Check label is set on all nodes")
			_, err = check.NodeLabel(APIClient, kmodName, localNsName, GeneralConfig.WorkerLabelMap)
			Expect(err).ToNot(HaveOccurred(), "error while checking the module is loaded")
		})

		AfterAll(func() {
			By("Delete Module")
			_, err := kmm.NewModuleBuilder(APIClient, kmodName, moduleName).Delete()
			Expect(err).ToNot(HaveOccurred(), "error creating test namespace")

			svcAccount := serviceaccount.NewBuilder(APIClient, serviceAccountName, moduleName)
			svcAccount.Exists()

			By("Delete ClusterRoleBinding")
			crb := define.ModuleCRB(*svcAccount, moduleName)
			err = crb.Delete()
			Expect(err).ToNot(HaveOccurred(), "error creating test namespace")

			By("Delete Namespace")
			err = namespace.NewBuilder(APIClient, moduleName).Delete()
			Expect(err).ToNot(HaveOccurred(), "error creating test namespace")
		})

	})
})
