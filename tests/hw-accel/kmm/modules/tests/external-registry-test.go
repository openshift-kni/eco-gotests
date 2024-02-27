package tests

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/openshift-kni/eco-gotests/tests/internal/cluster"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/configmap"
	"github.com/openshift-kni/eco-goinfra/pkg/kmm"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/secret"
	"github.com/openshift-kni/eco-goinfra/pkg/serviceaccount"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/internal/await"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/internal/check"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/internal/define"
	. "github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/internal/kmminittools"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/internal/kmmparams"
	. "github.com/openshift-kni/eco-gotests/tests/internal/inittools"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
	v1 "k8s.io/api/core/v1"
)

var _ = Describe("KMM", Ordered, Label(kmmparams.LabelSuite, kmmparams.LabelSanity), func() {

	Context("Module", Label("simple-kmod"), func() {

		moduleName := "simple-kmod"
		kmodName := "simple-kmod"
		localNsName := kmmparams.SimpleKmodModuleTestNamespace
		serviceAccountName := "simple-kmod-manager"
		secretName := "ocp-edge-qe-build-secret"
		image := fmt.Sprintf("%s/%s:$KERNEL_FULL_VERSION-%v",
			ModulesConfig.Registry, moduleName, time.Now().Unix())
		buildArgValue := fmt.Sprintf("%s.o", kmodName)

		var module *kmm.ModuleBuilder
		var svcAccount *serviceaccount.Builder
		var originalSecretMap map[string]map[string]interface{}
		var secretMap map[string]map[string]interface{}

		BeforeAll(func() {
			if ModulesConfig.PullSecret == "" || ModulesConfig.Registry == "" {
				Skip("No external registry secret found in environment, Skipping test")
			}

			By("Create Namespace")
			_, err := namespace.NewBuilder(APIClient, localNsName).Create()
			Expect(err).NotTo(HaveOccurred(), "error creating test namespace")

			By("Creating registry secret")
			secretContent := define.SecretContent(ModulesConfig.Registry, ModulesConfig.PullSecret)
			_, err = secret.NewBuilder(APIClient, secretName,
				localNsName, v1.SecretTypeDockerConfigJson).WithData(secretContent).Create()
			Expect(err).ToNot(HaveOccurred(), "failed creating secret")

			By("Get cluster's global pull-secret")
			globalSecret, err := cluster.GetOCPPullSecret(APIClient)
			Expect(err).ToNot(HaveOccurred(), "error fetching cluster's pull-secret")

			err = json.Unmarshal(globalSecret.Object.Data[".dockerconfigjson"], &secretMap)
			Expect(err).ToNot(HaveOccurred(), "error unmarshal pull-secret")
			err = json.Unmarshal(globalSecret.Object.Data[".dockerconfigjson"], &originalSecretMap)
			Expect(err).ToNot(HaveOccurred(), "error unmarshal pull-secret")

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
				WithBuildArg(kmmparams.BuildArgName, buildArgValue).
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
			err = await.ModuleDeployment(APIClient, moduleName, localNsName, 5*time.Minute, GeneralConfig.WorkerLabelMap)
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

			By("Await module to be deleted")
			err = await.ModuleObjectDeleted(APIClient, kmodName, localNsName, 3*time.Minute)
			Expect(err).ToNot(HaveOccurred(), "error while waiting module to be deleted")

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
			err = await.ModuleDeployment(APIClient, moduleName, localNsName, 3*time.Minute, GeneralConfig.WorkerLabelMap)
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

			By("Await module to be deleted")
			err = await.ModuleObjectDeleted(APIClient, kmodName, localNsName, 3*time.Minute)
			Expect(err).ToNot(HaveOccurred(), "error while waiting module to be deleted")

			By("Await pods deletion")
			err = await.ModuleUndeployed(APIClient, localNsName, time.Minute)
			Expect(err).ToNot(HaveOccurred(), "error while waiting pods to be deleted")

			By("Check labels are removed on all nodes")
			_, err = check.NodeLabel(APIClient, kmodName, localNsName, GeneralConfig.WorkerLabelMap)
			log.Printf("error is: %v", err)
			Expect(err).To(HaveOccurred(), "error while checking the module is loaded")

		})

		It("should deploy prebuild image with global secret", polarion.ID("71694"), func() {

			By("Update global pull-secret")
			if secretMap["auths"][ModulesConfig.Registry] == nil {
				secretMap["auths"][ModulesConfig.Registry] = map[string]string{
					"auth":  ModulesConfig.PullSecret,
					"email": "",
				}

				ps, err := json.Marshal(secretMap)
				Expect(err).ToNot(HaveOccurred(), "error encoding pull secret")
				secretContents := map[string][]byte{".dockerconfigjson": ps}

				pullSecret, _ := secret.Pull(APIClient, "pull-secret", "openshift-config")
				_, err = pullSecret.WithData(secretContents).Update()
				Expect(err).ToNot(HaveOccurred(), "error updating global pull secret")
			}

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

			module = module.WithModuleLoaderContainer(moduleLoaderContainerCfg).
				WithLoadServiceAccount(svcAccount.Object.Name)
			_, err = module.Create()
			Expect(err).ToNot(HaveOccurred(), "error creating module")

			By("Await driver container deployment")
			err = await.ModuleDeployment(APIClient, moduleName, localNsName, 3*time.Minute, GeneralConfig.WorkerLabelMap)
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
			_, err := kmm.NewModuleBuilder(APIClient, kmodName, localNsName).Delete()
			Expect(err).ToNot(HaveOccurred(), "error creating test namespace")

			By("Await module to be deleted")
			err = await.ModuleObjectDeleted(APIClient, kmodName, localNsName, 3*time.Minute)
			Expect(err).ToNot(HaveOccurred(), "error while waiting module to be deleted")

			svcAccount := serviceaccount.NewBuilder(APIClient, serviceAccountName, moduleName)
			svcAccount.Exists()

			By("Delete ClusterRoleBinding")
			crb := define.ModuleCRB(*svcAccount, moduleName)
			err = crb.Delete()
			Expect(err).ToNot(HaveOccurred(), "error creating test namespace")

			By("Delete Namespace")
			err = namespace.NewBuilder(APIClient, moduleName).Delete()
			Expect(err).ToNot(HaveOccurred(), "error creating test namespace")

			By("Restore original global pull-secret")
			if originalSecretMap["auths"][ModulesConfig.Registry] == nil {
				pullSecret, err := secret.Pull(APIClient, "pull-secret", "openshift-config")
				Expect(err).ToNot(HaveOccurred(), "error pulling global pull secret")
				ps, err := json.Marshal(originalSecretMap)
				Expect(err).ToNot(HaveOccurred(), "error encoding pull-secret")

				origSecretContents := map[string][]byte{".dockerconfigjson": ps}
				_, err = pullSecret.WithData(origSecretContents).Update()
				Expect(err).ToNot(HaveOccurred(), "error restoring global pull secret")
			}
		})

	})
})
