package tests

import (
	"fmt"
	"time"

	"github.com/openshift-kni/eco-goinfra/pkg/configmap"
	"github.com/openshift-kni/eco-goinfra/pkg/kmm"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"github.com/openshift-kni/eco-goinfra/pkg/secret"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/internal/await"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/internal/check"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/internal/define"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/internal/kmmparams"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/mcm/internal/tsparams"
	corev1 "k8s.io/api/core/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/internal/kmminittools"
	. "github.com/openshift-kni/eco-gotests/tests/internal/inittools"
)

var _ = Describe("KMM-Hub", Ordered, Label(tsparams.LabelSuite), func() {

	Context("MCM", Label("hub-multi-stage"), func() {

		moduleName := "multi-build"
		secretName := "registry-secret"
		plainImage := fmt.Sprintf("%s/%s:$KERNEL_FULL_VERSION-%v",
			ModulesConfig.Registry, moduleName, time.Now().Unix())
		buildArgValue := fmt.Sprintf("%s.o", moduleName)

		BeforeAll(func() {
			if ModulesConfig.SpokeClusterName == "" || ModulesConfig.SpokeKubeConfig == "" {
				Skip("Skipping test. No Spoke environment variables defined.")
			}

			if ModulesConfig.Registry == "" || ModulesConfig.PullSecret == "" {
				Skip("Skipping test. No Registry or PullSecret environment variables defined.")
			}
		})

		AfterAll(func() {
			By("Delete ManagedClusterModule")
			_, err := kmm.NewManagedClusterModuleBuilder(APIClient, moduleName, kmmparams.KmmHubOperatorNamespace).Delete()
			Expect(err).ToNot(HaveOccurred(), "error deleting managedclustermodule")

			By("Await module to be deleted")
			err = await.ModuleObjectDeleted(ModulesConfig.SpokeAPIClient, moduleName, kmmparams.KmmOperatorNamespace,
				time.Minute)
			Expect(err).ToNot(HaveOccurred(), "error while waiting for module to be deleted on spoke")

			By("Delete Hub Secret")
			err = secret.NewBuilder(APIClient, secretName,
				kmmparams.KmmHubOperatorNamespace, corev1.SecretTypeDockerConfigJson).Delete()
			Expect(err).ToNot(HaveOccurred(), "error deleting hub registry secret")

			By("Delete Spoke Secret")
			err = secret.NewBuilder(ModulesConfig.SpokeAPIClient, secretName,
				kmmparams.KmmOperatorNamespace, corev1.SecretTypeDockerConfigJson).Delete()
			Expect(err).ToNot(HaveOccurred(), "error deleting spoke registry secret")

		})

		It("should deploy image on Spoke cluster", reportxml.ID("54004"), func() {

			By("Creating registry secret on Hub")
			secretContent := define.SecretContent(ModulesConfig.Registry, ModulesConfig.PullSecret)

			_, err := secret.NewBuilder(APIClient, secretName,
				kmmparams.KmmHubOperatorNamespace, corev1.SecretTypeDockerConfigJson).WithData(secretContent).Create()
			Expect(err).ToNot(HaveOccurred(), "error creating secret on hub")

			By("Creating registry secret on Spoke")
			_, err = secret.NewBuilder(ModulesConfig.SpokeAPIClient, secretName,
				kmmparams.KmmOperatorNamespace, corev1.SecretTypeDockerConfigJson).WithData(secretContent).Create()
			Expect(err).ToNot(HaveOccurred(), "error creating secret on spoke")

			By("Create ConfigMap")
			configmapContents := define.LocalMultiStageConfigMapContent(moduleName)
			dockerfileConfigMap, err := configmap.
				NewBuilder(APIClient, moduleName, kmmparams.KmmHubOperatorNamespace).
				WithData(configmapContents).Create()
			Expect(err).ToNot(HaveOccurred(), "error creating configmap")

			By("Create KernelMapping")
			kernelMapping := kmm.NewRegExKernelMappingBuilder("^.+$")

			kernelMapping.WithContainerImage(plainImage).
				WithBuildArg("MY_MODULE", buildArgValue).
				WithBuildDockerCfgFile(dockerfileConfigMap.Object.Name).
				WithBuildImageRegistryTLS(true, true).
				RegistryTLS(true, true)
			kerMapOne, err := kernelMapping.BuildKernelMappingConfig()
			Expect(err).ToNot(HaveOccurred(), "error creating kernel mapping")

			By("Create ModuleLoaderContainer")
			moduleLoaderContainer := kmm.NewModLoaderContainerBuilder(moduleName)
			moduleLoaderContainer.WithKernelMapping(kerMapOne)
			moduleLoaderContainer.WithImagePullPolicy("Always")
			moduleLoaderContainerCfg, err := moduleLoaderContainer.BuildModuleLoaderContainerCfg()
			Expect(err).ToNot(HaveOccurred(), "error creating moduleloadercontainer")

			By("Build Module Spec")
			moduleSpec, err := kmm.NewModuleBuilder(APIClient, moduleName, kmmparams.KmmOperatorNamespace).
				WithNodeSelector(GeneralConfig.ControlPlaneLabelMap).
				WithModuleLoaderContainer(moduleLoaderContainerCfg).
				WithImageRepoSecret(secretName).
				BuildModuleSpec()
			Expect(err).ToNot(HaveOccurred(), "error creating module spec")

			By("Create ManagedClusterModule")
			selector := map[string]string{"name": ModulesConfig.SpokeClusterName}
			_, err = kmm.NewManagedClusterModuleBuilder(APIClient, moduleName, kmmparams.KmmHubOperatorNamespace).
				WithModuleSpec(moduleSpec).
				WithSpokeNamespace(kmmparams.KmmOperatorNamespace).
				WithSelector(selector).
				Create()
			Expect(err).ToNot(HaveOccurred(), "error creating managedclustermodule")

			By("Await build pod to complete build")
			err = await.BuildPodCompleted(APIClient, kmmparams.KmmHubOperatorNamespace, 5*time.Minute)
			Expect(err).ToNot(HaveOccurred(), "error while building module")

			By("Await driver container deployment on Spoke")
			err = await.ModuleDeployment(ModulesConfig.SpokeAPIClient, moduleName, kmmparams.KmmOperatorNamespace,
				time.Minute, GeneralConfig.ControlPlaneLabelMap)
			Expect(err).ToNot(HaveOccurred(), "error while waiting on driver deployment")

			By("Check label is set on all nodes")
			_, err = check.NodeLabel(ModulesConfig.SpokeAPIClient, moduleName, kmmparams.KmmOperatorNamespace,
				GeneralConfig.ControlPlaneLabelMap)
			Expect(err).ToNot(HaveOccurred(), "error while checking label on all nodes")
		})

		It("should use SHA format for the module deployed on spoke", reportxml.ID("63102"), func() {
			By("Getting deployed Module on Spoke")
			moduleSpec, err := kmm.Pull(ModulesConfig.SpokeAPIClient, moduleName, kmmparams.KmmOperatorNamespace)
			Expect(err).ToNot(HaveOccurred(), "error while checking the module is loaded")
			Expect(moduleSpec.Definition.Spec.ModuleLoader.Container.KernelMappings[0].ContainerImage).
				Should(MatchRegexp(`:[A-Fa-f0-9]{64}$`))
		})
	})
})
