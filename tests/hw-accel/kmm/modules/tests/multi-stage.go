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

	Context("Module", Label("multi-stage"), func() {

		moduleName := tsparams.UseLocalMultiStageTestNamespace
		kmodName := "multi-stage"
		serviceAccountName := "multi-stage-manager"
		image := fmt.Sprintf("%s/%s/%s:$KERNEL_FULL_VERSION",
			tsparams.LocalImageRegistry, tsparams.UseLocalMultiStageTestNamespace, kmodName)
		buildArgValue := fmt.Sprintf("%s.o", kmodName)

		AfterEach(func() {
			By("Delete Module")
			_, err := kmm.NewModuleBuilder(APIClient, moduleName, tsparams.UseLocalMultiStageTestNamespace).Delete()
			Expect(err).ToNot(HaveOccurred(), "error creating test namespace")

			svcAccount := serviceaccount.NewBuilder(APIClient, serviceAccountName, tsparams.UseLocalMultiStageTestNamespace)
			svcAccount.Exists()

			By("Delete ClusterRoleBinding")
			crb := define.ModuleCRB(*svcAccount, kmodName)
			err = crb.Delete()
			Expect(err).ToNot(HaveOccurred(), "error creating test namespace")

			By("Delete Namespace")
			err = namespace.NewBuilder(APIClient, tsparams.UseLocalMultiStageTestNamespace).Delete()
			Expect(err).ToNot(HaveOccurred(), "error creating test namespace")
		})

		It("should use internal image-stream", polarion.ID("53651"), func() {

			By("Create Namespace")
			testNamespace, err := namespace.NewBuilder(APIClient, tsparams.UseLocalMultiStageTestNamespace).Create()
			Expect(err).ToNot(HaveOccurred(), "error creating test namespace")

			configmapContents := define.LocalMultiStageConfigMapContent(kmodName)

			By("Create ConfigMap")
			dockerfileConfigMap, err := configmap.
				NewBuilder(APIClient, kmodName, testNamespace.Object.Name).
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
			err = check.ModuleLoaded(APIClient, kmodName, tsparams.UseLocalMultiStageTestNamespace, time.Minute)
			Expect(err).ToNot(HaveOccurred(), "error while checking the module is loaded")

			By("Check label is set on all nodes")
			_, err = check.NodeLabel(APIClient, moduleName, GeneralConfig.WorkerLabelMap)
			Expect(err).ToNot(HaveOccurred(), "error while checking the module is loaded")
		})
	})
})
