package tests

import (
	"fmt"

	"strings"
	"time"

	"github.com/openshift-kni/eco-goinfra/pkg/configmap"
	"github.com/openshift-kni/eco-goinfra/pkg/kmm"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"github.com/openshift-kni/eco-goinfra/pkg/secret"
	"github.com/openshift-kni/eco-goinfra/pkg/serviceaccount"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/internal/await"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/internal/check"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/internal/define"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/internal/get"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/internal/kmmparams"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/modules/internal/tsparams"
	corev1 "k8s.io/api/core/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/openshift-kni/eco-gotests/tests/internal/inittools"
)

var _ = Describe("KMM", Ordered, Label(kmmparams.LabelSuite, kmmparams.LabelSanity), func() {

	Context("Module", Label("build-sign"), func() {

		moduleName := kmmparams.ModuleBuildAndSignNamespace
		kmodName := "module-signing"
		serviceAccountName := "build-and-sign-sa"
		image := fmt.Sprintf("%s/%s/%s:$KERNEL_FULL_VERSION",
			tsparams.LocalImageRegistry, kmmparams.ModuleBuildAndSignNamespace, kmodName)
		buildArgValue := fmt.Sprintf("%s.o", kmodName)
		filesToSign := []string{fmt.Sprintf("/opt/lib/modules/$KERNEL_FULL_VERSION/%s.ko", kmodName)}

		AfterAll(func() {
			By("Delete Module")
			_, err := kmm.NewModuleBuilder(APIClient, moduleName, kmmparams.ModuleBuildAndSignNamespace).Delete()
			Expect(err).ToNot(HaveOccurred(), "error deleting module")

			By("Await module to be deleted")
			err = await.ModuleObjectDeleted(APIClient, moduleName, kmmparams.ModuleBuildAndSignNamespace, time.Minute)
			Expect(err).ToNot(HaveOccurred(), "error while waiting module to be deleted")

			svcAccount := serviceaccount.NewBuilder(APIClient, serviceAccountName, kmmparams.ModuleBuildAndSignNamespace)
			svcAccount.Exists()

			By("Delete ClusterRoleBinding")
			crb := define.ModuleCRB(*svcAccount, kmodName)
			err = crb.Delete()
			Expect(err).ToNot(HaveOccurred(), "error creating test namespace")

			By("Delete preflightvalidationocp")
			_, err = kmm.NewPreflightValidationOCPBuilder(APIClient, kmmparams.PreflightName,
				kmmparams.ModuleBuildAndSignNamespace).Delete()
			Expect(err).ToNot(HaveOccurred(), "error deleting preflightvalidationocp")

			By("Delete Namespace")
			err = namespace.NewBuilder(APIClient, kmmparams.ModuleBuildAndSignNamespace).Delete()
			Expect(err).ToNot(HaveOccurred(), "error creating test namespace")

		})

		It("should use build and sign a module", reportxml.ID("56252"), func() {

			By("Create Namespace")
			testNamespace, err := namespace.NewBuilder(APIClient, kmmparams.ModuleBuildAndSignNamespace).Create()
			Expect(err).ToNot(HaveOccurred(), "error creating test namespace")

			By("Creating my-signing-key-pub")
			signKey := get.SigningData("cert", kmmparams.SigningCertBase64)

			_, err = secret.NewBuilder(APIClient, "my-signing-key-pub",
				kmmparams.ModuleBuildAndSignNamespace, corev1.SecretTypeOpaque).WithData(signKey).Create()
			Expect(err).ToNot(HaveOccurred(), "failed creating secret")

			By("Creating my-signing-key")
			signCert := get.SigningData("key", kmmparams.SigningKeyBase64)

			_, err = secret.NewBuilder(APIClient, "my-signing-key",
				kmmparams.ModuleBuildAndSignNamespace, corev1.SecretTypeOpaque).WithData(signCert).Create()
			Expect(err).ToNot(HaveOccurred(), "failed creating secret")

			By("Create ConfigMap")
			configmapContents := define.MultiStageConfigMapContent(kmodName)

			dockerfileConfigMap, err := configmap.
				NewBuilder(APIClient, kmodName, testNamespace.Object.Name).
				WithData(configmapContents).Create()
			Expect(err).ToNot(HaveOccurred(), "error creating configmap")

			By("Create ServiceAccount")
			svcAccount, err := serviceaccount.
				NewBuilder(APIClient, serviceAccountName, kmmparams.ModuleBuildAndSignNamespace).Create()
			Expect(err).ToNot(HaveOccurred(), "error creating serviceaccount")

			By("Create ClusterRoleBinding")
			crb := define.ModuleCRB(*svcAccount, kmodName)
			_, err = crb.Create()
			Expect(err).ToNot(HaveOccurred(), "error creating clusterrolebinding")

			By("Create KernelMapping")
			kernelMapping := kmm.NewRegExKernelMappingBuilder("^.+$")

			kernelMapping.WithContainerImage(image).
				WithBuildArg(kmmparams.BuildArgName, buildArgValue).
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
			module := kmm.NewModuleBuilder(APIClient, moduleName, kmmparams.ModuleBuildAndSignNamespace).
				WithNodeSelector(GeneralConfig.WorkerLabelMap)
			module = module.WithModuleLoaderContainer(moduleLoaderContainerCfg).
				WithLoadServiceAccount(svcAccount.Object.Name)
			_, err = module.Create()
			Expect(err).ToNot(HaveOccurred(), "error creating module")

			By("Await build pod to complete build")
			err = await.BuildPodCompleted(APIClient, kmmparams.ModuleBuildAndSignNamespace, 5*time.Minute)
			Expect(err).ToNot(HaveOccurred(), "error while building module")

			By("Await driver container deployment")
			err = await.ModuleDeployment(APIClient, moduleName, kmmparams.ModuleBuildAndSignNamespace, time.Minute,
				GeneralConfig.WorkerLabelMap)
			Expect(err).ToNot(HaveOccurred(), "error while waiting on driver deployment")

			By("Check module is loaded on node")
			err = check.ModuleLoaded(APIClient, kmodName, time.Minute)
			Expect(err).ToNot(HaveOccurred(), "error while checking the module is loaded")

			By("Check module is signed")
			err = check.ModuleSigned(APIClient, kmodName, "cdvtest signing key",
				kmmparams.ModuleBuildAndSignNamespace, image)
			Expect(err).ToNot(HaveOccurred(), "error while checking the module is signed")

			By("Check label is set on all nodes")
			_, err = check.NodeLabel(APIClient, moduleName, kmmparams.ModuleBuildAndSignNamespace,
				GeneralConfig.WorkerLabelMap)
			Expect(err).ToNot(HaveOccurred(), "error while checking the module is loaded")
		})

		It("should be able to run preflightvalidation with no push", reportxml.ID("56329"), func() {
			By("Detecting cluster architecture")

			arch, err := get.ClusterArchitecture(APIClient, GeneralConfig.WorkerLabelMap)
			if err != nil {
				Skip("could not detect cluster architecture")
			}
			preflightImage := get.PreflightImage(arch)

			By("Create preflightvalidationocp")
			pre, err := kmm.NewPreflightValidationOCPBuilder(APIClient, kmmparams.PreflightName,
				kmmparams.ModuleBuildAndSignNamespace).
				WithReleaseImage(preflightImage).
				WithPushBuiltImage(false).
				Create()
			Expect(err).ToNot(HaveOccurred(), "error while creating preflight")

			By("Await build pod to complete build")
			err = await.BuildPodCompleted(APIClient, kmmparams.ModuleBuildAndSignNamespace, 5*time.Minute)
			Expect(err).ToNot(HaveOccurred(), "error while building module")

			By("Await preflightvalidationocp checks")
			err = await.PreflightStageDone(APIClient, kmmparams.PreflightName, moduleName,
				kmmparams.ModuleBuildAndSignNamespace, time.Minute)
			Expect(err).To(HaveOccurred(), "preflightvalidationocp did not complete")

			By("Get status of the preflightvalidationocp checks")
			status, _ := get.PreflightReason(APIClient, kmmparams.PreflightName, moduleName,
				kmmparams.ModuleBuildAndSignNamespace)
			Expect(strings.Contains(status, "Failed to verify signing for module")).
				To(BeTrue(), "expected message not found")

			By("Delete preflight validation")
			_, err = pre.Delete()
			Expect(err).ToNot(HaveOccurred(), "error deleting preflightvalidation")
		})

		It("should be able to run preflightvalidation and push to registry", reportxml.ID("56327"), func() {
			By("Detecting cluster architecture")

			arch, err := get.ClusterArchitecture(APIClient, GeneralConfig.WorkerLabelMap)
			if err != nil {
				Skip("could not detect cluster architecture")
			}
			preflightImage := get.PreflightImage(arch)

			By("Create preflightvalidationocp")
			_, err = kmm.NewPreflightValidationOCPBuilder(APIClient, kmmparams.PreflightName,
				kmmparams.ModuleBuildAndSignNamespace).
				WithReleaseImage(preflightImage).
				WithPushBuiltImage(true).
				Create()
			Expect(err).ToNot(HaveOccurred(), "error while creating preflight")

			By("Await build pod to complete build")
			err = await.BuildPodCompleted(APIClient, kmmparams.ModuleBuildAndSignNamespace, 5*time.Minute)
			Expect(err).ToNot(HaveOccurred(), "error while building module")

			By("Await preflightvalidationocp checks")
			err = await.PreflightStageDone(APIClient, kmmparams.PreflightName, moduleName,
				kmmparams.ModuleBuildAndSignNamespace, 3*time.Minute)
			Expect(err).NotTo(HaveOccurred(), "preflightvalidationocp did not complete")

			By("Get status of the preflightvalidationocp checks")
			status, _ := get.PreflightReason(APIClient, kmmparams.PreflightName, moduleName,
				kmmparams.ModuleBuildAndSignNamespace)
			Expect(strings.Contains(status, "Verification successful (sign completes and image pushed)")).
				To(BeTrue(), "expected message not found")
		})
	})
})
