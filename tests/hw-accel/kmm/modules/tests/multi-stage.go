package tests

import (
	"fmt"
	"time"

	"github.com/openshift-kni/eco-goinfra/pkg/secret"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/internal/kmmparams"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/modules/internal/get"
	v1 "k8s.io/api/core/v1"

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

	Context("Module", Label("multi-stage"), func() {

		moduleName := tsparams.UseLocalMultiStageTestNamespace
		kmodName := "multi-stage"
		serviceAccountName := "multi-stage-manager"
		plainImage := fmt.Sprintf("%s/%s/%s:$KERNEL_FULL_VERSION-plain",
			tsparams.LocalImageRegistry, tsparams.UseLocalMultiStageTestNamespace, kmodName)
		signedImage := fmt.Sprintf("%s/%s/%s:$KERNEL_FULL_VERSION-signed",
			tsparams.LocalImageRegistry, tsparams.UseLocalMultiStageTestNamespace, kmodName)
		buildArgValue := fmt.Sprintf("%s.o", kmodName)
		filesToSign := []string{fmt.Sprintf("/opt/lib/modules/$KERNEL_FULL_VERSION/%s.ko", kmodName)}

		BeforeAll(func() {
			By("Create Namespace")
			_, err := namespace.NewBuilder(APIClient, tsparams.UseLocalMultiStageTestNamespace).Create()
			Expect(err).ToNot(HaveOccurred(), "error creating test namespace")
		})

		AfterEach(func() {
			By("Delete Module")
			_, err := kmm.NewModuleBuilder(APIClient, moduleName, tsparams.UseLocalMultiStageTestNamespace).Delete()
			Expect(err).ToNot(HaveOccurred(), "error creating test namespace")

			By("Await pods deletion")
			err = await.ModuleUndeployed(APIClient, tsparams.UseLocalMultiStageTestNamespace, time.Minute)
			Expect(err).ToNot(HaveOccurred(), "error while waiting pods to be deleted")
		})

		AfterAll(func() {
			svcAccount := serviceaccount.NewBuilder(APIClient, serviceAccountName, tsparams.UseLocalMultiStageTestNamespace)
			svcAccount.Exists()

			By("Delete ClusterRoleBinding")
			crb := define.ModuleCRB(*svcAccount, kmodName)
			err := crb.Delete()
			Expect(err).ToNot(HaveOccurred(), "error creating test namespace")

			By("Delete Namespace")
			err = namespace.NewBuilder(APIClient, tsparams.UseLocalMultiStageTestNamespace).Delete()
			Expect(err).ToNot(HaveOccurred(), "error creating test namespace")
		})

		It("should use internal image-stream", polarion.ID("53651"), func() {

			configmapContents := define.LocalMultiStageConfigMapContent(kmodName)

			By("Create ConfigMap")
			dockerfileConfigMap, err := configmap.
				NewBuilder(APIClient, kmodName, tsparams.UseLocalMultiStageTestNamespace).
				WithData(configmapContents).Create()
			Expect(err).ToNot(HaveOccurred(), "error creating configmap")

			By("Create ServiceAccount")
			svcAccount, err := serviceaccount.
				NewBuilder(APIClient, serviceAccountName, tsparams.UseLocalMultiStageTestNamespace).Create()
			Expect(err).ToNot(HaveOccurred(), "error creating serviceaccount")

			By("Create ClusterRoleBinding")
			crb := define.ModuleCRB(*svcAccount, kmodName)
			_, err = crb.Create()
			Expect(err).ToNot(HaveOccurred(), "error creating clusterrolebinding")

			By("Create KernelMapping")
			kernelMapping := kmm.NewRegExKernelMappingBuilder("^.+$")

			kernelMapping.WithContainerImage(plainImage).
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
			module := kmm.NewModuleBuilder(APIClient, moduleName, tsparams.UseLocalMultiStageTestNamespace).
				WithNodeSelector(GeneralConfig.WorkerLabelMap)
			module = module.WithModuleLoaderContainer(moduleLoaderContainerCfg).
				WithLoadServiceAccount(svcAccount.Object.Name)
			_, err = module.Create()
			Expect(err).ToNot(HaveOccurred(), "error creating module")

			By("Await build pod to complete build")
			err = await.BuildPodCompleted(APIClient, tsparams.UseLocalMultiStageTestNamespace, 5*time.Minute)
			Expect(err).ToNot(HaveOccurred(), "error while building module")

			By("Await driver container deployment")
			err = await.ModuleDeployment(APIClient, moduleName, tsparams.UseLocalMultiStageTestNamespace, time.Minute,
				GeneralConfig.WorkerLabelMap)
			Expect(err).ToNot(HaveOccurred(), "error while waiting on driver deployment")

			By("Check module is loaded on node")
			err = check.ModuleLoaded(APIClient, kmodName, time.Minute)
			Expect(err).ToNot(HaveOccurred(), "error while checking the module is loaded")

			By("Check label is set on all nodes")
			_, err = check.NodeLabel(APIClient, moduleName, GeneralConfig.WorkerLabelMap)
			Expect(err).ToNot(HaveOccurred(), "error while checking the module is loaded")
		})

		It("should sign an existing image and load the signed module", polarion.ID("53677"), func() {
			By("Creating my-signing-key-pub")
			signKey := get.SigningData("cert", kmmparams.SigningCertBase64)

			_, err := secret.NewBuilder(APIClient, "my-signing-key-pub",
				tsparams.UseLocalMultiStageTestNamespace, v1.SecretTypeOpaque).WithData(signKey).Create()
			Expect(err).ToNot(HaveOccurred(), "failed creating secret")

			By("Creating my-signing-key")
			signCert := get.SigningData("key", kmmparams.SigningKeyBase64)

			_, err = secret.NewBuilder(APIClient, "my-signing-key",
				tsparams.UseLocalMultiStageTestNamespace, v1.SecretTypeOpaque).WithData(signCert).Create()
			Expect(err).ToNot(HaveOccurred(), "failed creating secret")

			By("Reusing previously created ServiceAccount")
			svcAccount, err := serviceaccount.
				NewBuilder(APIClient, serviceAccountName, tsparams.UseLocalMultiStageTestNamespace).Create()
			Expect(err).ToNot(HaveOccurred(), "error creating serviceaccount")

			By("Create KernelMapping")
			kernelMapping := kmm.NewRegExKernelMappingBuilder("^.+$")

			kernelMapping.WithContainerImage(signedImage).
				WithSign("my-signing-key-pub", "my-signing-key", filesToSign)
			kerMapOne, err := kernelMapping.BuildKernelMappingConfig()
			kerMapOne.Sign.UnsignedImage = plainImage
			Expect(err).ToNot(HaveOccurred(), "error creating kernel mapping")

			By("Create ModuleLoaderContainer")
			moduleLoaderContainer := kmm.NewModLoaderContainerBuilder(kmodName)
			moduleLoaderContainer.WithKernelMapping(kerMapOne)
			moduleLoaderContainer.WithImagePullPolicy("Always")
			moduleLoaderContainerCfg, err := moduleLoaderContainer.BuildModuleLoaderContainerCfg()
			Expect(err).ToNot(HaveOccurred(), "error creating moduleloadercontainer")

			By("Create Module")
			module := kmm.NewModuleBuilder(APIClient, moduleName, tsparams.UseLocalMultiStageTestNamespace).
				WithNodeSelector(GeneralConfig.WorkerLabelMap)
			module = module.WithModuleLoaderContainer(moduleLoaderContainerCfg).
				WithLoadServiceAccount(svcAccount.Object.Name)
			_, err = module.Create()
			Expect(err).ToNot(HaveOccurred(), "error creating module")

			By("Await driver container deployment")
			err = await.ModuleDeployment(APIClient, moduleName, tsparams.UseLocalMultiStageTestNamespace, 3*time.Minute,
				GeneralConfig.WorkerLabelMap)
			Expect(err).ToNot(HaveOccurred(), "error while waiting on driver deployment")

			By("Check module is loaded on node")
			err = check.ModuleLoaded(APIClient, kmodName, time.Minute)
			Expect(err).ToNot(HaveOccurred(), "error while checking the module is loaded")

			By("Check module is signed")
			err = check.ModuleSigned(APIClient, kmodName, "cdvtest signing key",
				tsparams.UseLocalMultiStageTestNamespace, signedImage)
			Expect(err).ToNot(HaveOccurred(), "error while checking the module is signed")

			By("Check label is set on all nodes")
			_, err = check.NodeLabel(APIClient, moduleName, GeneralConfig.WorkerLabelMap)
			Expect(err).ToNot(HaveOccurred(), "error while checking the module is loaded")
		})
	})
})
