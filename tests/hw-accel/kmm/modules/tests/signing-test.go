package tests

import (
	"fmt"
	"time"

	"github.com/openshift-kni/eco-goinfra/pkg/configmap"
	"github.com/openshift-kni/eco-goinfra/pkg/kmm"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/secret"
	"github.com/openshift-kni/eco-goinfra/pkg/serviceaccount"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/internal/kmmparams"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/modules/internal/await"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/modules/internal/check"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/modules/internal/define"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/modules/internal/get"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/modules/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
	v1 "k8s.io/api/core/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/openshift-kni/eco-gotests/tests/internal/inittools"
)

var _ = Describe("KMM", Ordered, Label(tsparams.LabelSuite), func() {

	Context("Module", Label("build-sign"), func() {

		moduleName := tsparams.ModuleBuildAndSignNamespace
		kmodName := "module-signing"
		serviceAccountName := "build-and-sign-module-manager-rolebinding"
		image := fmt.Sprintf("%s/%s/%s:$KERNEL_FULL_VERSION",
			tsparams.LocalImageRegistry, tsparams.ModuleBuildAndSignNamespace, kmodName)
		buildArgValue := fmt.Sprintf("%s.o", kmodName)
		filesToSign := []string{fmt.Sprintf("/opt/lib/modules/$KERNEL_FULL_VERSION/%s.ko", kmodName)}

		AfterEach(func() {

			By("Delete Module")
			_, err := kmm.NewModuleBuilder(APIClient, moduleName, tsparams.ModuleBuildAndSignNamespace).Delete()
			Expect(err).ToNot(HaveOccurred(), "error creating test namespace")
			svcAccount := serviceaccount.NewBuilder(APIClient, serviceAccountName, tsparams.ModuleBuildAndSignNamespace)
			svcAccount.Exists()

			By("Delete ClusterRoleBinding")
			crb := define.ModuleCRB(*svcAccount, kmodName)
			err = crb.Delete()
			Expect(err).ToNot(HaveOccurred(), "error creating test namespace")

			By("Delete Namespace")
			err = namespace.NewBuilder(APIClient, tsparams.ModuleBuildAndSignNamespace).Delete()
			Expect(err).ToNot(HaveOccurred(), "error creating test namespace")

		})

		It("should use build and sign a module", polarion.ID("56252"), func() {

			By("Create Namespace")
			testNamespace, err := namespace.NewBuilder(APIClient, tsparams.ModuleBuildAndSignNamespace).Create()
			Expect(err).ToNot(HaveOccurred(), "error creating test namespace")

			By("Creating my-signing-key-pub")
			signKey := get.SigningData("cert", kmmparams.SigningCertBase64)

			_, err = secret.NewBuilder(APIClient, "my-signing-key-pub",
				tsparams.ModuleBuildAndSignNamespace, v1.SecretTypeOpaque).WithData(signKey).Create()
			Expect(err).ToNot(HaveOccurred(), "failed creating secret")

			By("Creating my-signing-key")
			signCert := get.SigningData("key", kmmparams.SigningKeyBase64)

			_, err = secret.NewBuilder(APIClient, "my-signing-key",
				tsparams.ModuleBuildAndSignNamespace, v1.SecretTypeOpaque).WithData(signCert).Create()
			Expect(err).ToNot(HaveOccurred(), "failed creating secret")

			By("Create ConfigMap")
			configmapContents := define.MultiStageConfigMapContent(kmodName)

			dockerfileConfigMap, err := configmap.
				NewBuilder(APIClient, kmodName, testNamespace.Object.Name).
				WithData(configmapContents).Create()
			Expect(err).ToNot(HaveOccurred(), "error creating configmap")

			By("Create ServiceAccount")
			svcAccount, err := serviceaccount.
				NewBuilder(APIClient, serviceAccountName, tsparams.ModuleBuildAndSignNamespace).Create()
			Expect(err).ToNot(HaveOccurred(), "error creating serviceaccount")

			By("Create ClusterRoleBinding")
			crb := define.ModuleCRB(*svcAccount, kmodName)
			_, err = crb.Create()
			Expect(err).ToNot(HaveOccurred(), "error creating clusterrolebinding")

			By("Create KernelMapping")
			kernelMapping := kmm.NewRegExKernelMappingBuilder("^.+$")

			kernelMapping.WithContainerImage(image).
				WithBuildArg(tsparams.BuildArgName, buildArgValue).
				WithBuildDockerCfgFile(dockerfileConfigMap.Object.Name).
				WithSign("my-signing-key-pub", "my-signing-key", filesToSign)
			kerMapOne, err := kernelMapping.BuildKernelMappingConfig()
			Expect(err).ToNot(HaveOccurred(), "error creating kernel mapping")

			By("Create ModuleLoaderContainer")
			moduleLoaderContainer := kmm.NewModLoaderContainerBuilder(kmodName)
			moduleLoaderContainer.WithKernelMapping(kerMapOne)
			moduleLoaderContainer.WithImagePullPolicy("Always")
			moduleLoaderContainerCfg, err := moduleLoaderContainer.BuildModuleLoaderContainerCfg()
			Expect(err).ToNot(HaveOccurred(), "error creating moduleloadercontainer")

			By("Create Module")
			module := kmm.NewModuleBuilder(APIClient, moduleName, tsparams.ModuleBuildAndSignNamespace).
				WithNodeSelector(GeneralConfig.WorkerLabelMap)
			module = module.WithModuleLoaderContainer(moduleLoaderContainerCfg).
				WithLoadServiceAccount(svcAccount.Object.Name)
			_, err = module.Create()
			Expect(err).ToNot(HaveOccurred(), "error creating module")

			By("Await build pod to complete build")
			err = await.BuildPodCompleted(APIClient, tsparams.ModuleBuildAndSignNamespace, 5*time.Minute)
			Expect(err).ToNot(HaveOccurred(), "error while building module")

			By("Await driver container deployment")
			err = await.ModuleDeployment(APIClient, moduleName, tsparams.ModuleBuildAndSignNamespace, time.Minute,
				GeneralConfig.WorkerLabelMap)
			Expect(err).ToNot(HaveOccurred(), "error while waiting on driver deployment")

			By("Check module is loaded on node")
			err = check.ModuleLoaded(APIClient, kmodName, tsparams.ModuleBuildAndSignNamespace, time.Minute)
			Expect(err).ToNot(HaveOccurred(), "error while checking the module is loaded")

			By("Check module is signed")
			err = check.ModuleSigned(APIClient, kmodName, "cdvtest signing key",
				tsparams.ModuleBuildAndSignNamespace, time.Minute)
			Expect(err).ToNot(HaveOccurred(), "error while checking the module is signed")

			By("Check label is set on all nodes")
			_, err = check.NodeLabel(APIClient, moduleName, GeneralConfig.WorkerLabelMap)
			Expect(err).ToNot(HaveOccurred(), "error while checking the module is loaded")
		})
	})
})
