package tests

import (
	"fmt"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	corev1 "k8s.io/api/core/v1"
	"time"

	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/internal/await"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/internal/check"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/internal/define"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/internal/kmmparams"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/modules/internal/tsparams"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/configmap"
	"github.com/openshift-kni/eco-goinfra/pkg/kmm"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"github.com/openshift-kni/eco-goinfra/pkg/serviceaccount"
	. "github.com/openshift-kni/eco-gotests/tests/internal/inittools"
)

var _ = Describe("KMM", Ordered, Label(kmmparams.LabelSuite, kmmparams.LabelSanity), func() {

	Context("Module", Label("tolerations"), func() {

		var testNamespace *namespace.Builder

		moduleName := kmmparams.UseDtkModuleTestNamespace
		kmodName := "use-dtk"
		serviceAccountName := "dtk-manager"
		image := fmt.Sprintf("%s/%s/%s:$KERNEL_FULL_VERSION",
			tsparams.LocalImageRegistry, kmmparams.UseDtkModuleTestNamespace, kmodName)
		buildArgValue := fmt.Sprintf("%s.o", kmodName)

		BeforeAll(func() {

			By("Create Namespace")
			var err error
			testNamespace, err = namespace.NewBuilder(APIClient, kmmparams.UseDtkModuleTestNamespace).Create()
			Expect(err).ToNot(HaveOccurred(), "error creating test namespace")

		})

		AfterAll(func() {

			By("Delete Module")
			_, err := kmm.NewModuleBuilder(APIClient, moduleName, kmmparams.UseDtkModuleTestNamespace).Delete()
			Expect(err).ToNot(HaveOccurred(), "error deleting module")

			By("Await module to be deleted")
			err = await.ModuleObjectDeleted(APIClient, moduleName, kmmparams.UseDtkModuleTestNamespace, time.Minute)
			Expect(err).ToNot(HaveOccurred(), "error while waiting module to be deleted")

			svcAccount := serviceaccount.NewBuilder(APIClient, serviceAccountName, kmmparams.UseDtkModuleTestNamespace)
			svcAccount.Exists()

			By("Delete ClusterRoleBinding")
			crb := define.ModuleCRB(*svcAccount, kmodName)
			err = crb.Delete()
			Expect(err).ToNot(HaveOccurred(), "error deleting test namespace")

			By("Delete Namespace")
			err = namespace.NewBuilder(APIClient, kmmparams.UseDtkModuleTestNamespace).Delete()
			Expect(err).ToNot(HaveOccurred(), "error creating test namespace")

		})

		It("should use DTK_AUTO parameter", reportxml.ID("54283"), func() {

			configmapContents := define.MultiStageConfigMapContent(kmodName)

			By("Create ConfigMap")
			dockerfileConfigMap, err := configmap.
				NewBuilder(APIClient, kmodName, testNamespace.Object.Name).
				WithData(configmapContents).Create()
			Expect(err).ToNot(HaveOccurred(), "error creating configmap")

			By("Create ServiceAccount")
			svcAccount, err := serviceaccount.
				NewBuilder(APIClient, serviceAccountName, kmmparams.UseDtkModuleTestNamespace).Create()
			Expect(err).ToNot(HaveOccurred(), "error creating serviceaccount")

			By("Create ClusterRoleBinding")
			crb := define.ModuleCRB(*svcAccount, kmodName)
			_, err = crb.Create()
			Expect(err).ToNot(HaveOccurred(), "error creating clusterrolebinding")

			By("Create KernelMapping")
			kernelMapping := kmm.NewRegExKernelMappingBuilder("^.+$")

			kernelMapping.WithContainerImage(image).
				WithBuildArg(kmmparams.BuildArgName, buildArgValue).
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
			module := kmm.NewModuleBuilder(APIClient, moduleName, kmmparams.UseDtkModuleTestNamespace).
				WithNodeSelector(GeneralConfig.WorkerLabelMap)
			module = module.WithModuleLoaderContainer(moduleLoaderContainerCfg).
				WithLoadServiceAccount(svcAccount.Object.Name).WithToleration(
				kmmparams.TolerationNoExecuteK8sUnreachable.Key,
				string(kmmparams.TolerationNoExecuteK8sUnreachable.Operator),
				kmmparams.TolerationNoExecuteK8sUnreachable.Value,
				string(kmmparams.TolerationNoExecuteK8sUnreachable.Effect), nil)

			module = module.WithToleration(
				kmmparams.TolerationNoScheduleKeyValue.Key,
				string(kmmparams.TolerationNoScheduleKeyValue.Operator),
				kmmparams.TolerationNoScheduleKeyValue.Value,
				string(kmmparams.TolerationNoScheduleKeyValue.Effect), nil)
			_, err = module.Create()
			Expect(err).ToNot(HaveOccurred(), "error creating module")

			By("Await build pod to complete build")
			err = await.BuildPodCompleted(APIClient, kmmparams.UseDtkModuleTestNamespace, 5*time.Minute)
			Expect(err).ToNot(HaveOccurred(), "error while building module")

			By("Await driver container deployment")
			err = await.ModuleDeployment(APIClient, moduleName, kmmparams.UseDtkModuleTestNamespace, time.Minute,
				GeneralConfig.WorkerLabelMap)
			Expect(err).ToNot(HaveOccurred(), "error while waiting on driver deployment")

			By("Check module is loaded on node")
			err = check.ModuleLoaded(APIClient, kmodName, time.Minute)
			Expect(err).ToNot(HaveOccurred(), "error while checking the module is loaded")

			By("Check label is set on all nodes")
			_, err = check.NodeLabel(APIClient, moduleName, kmmparams.UseDtkModuleTestNamespace,
				GeneralConfig.WorkerLabelMap)
			Expect(err).ToNot(HaveOccurred(), "error while checking the module is loaded")

			By("Taint node with test taint")
			node, err := nodes.Pull(APIClient, "worker-0-0")

			var newTaint []corev1.Taint

			newTaint = append(newTaint, corev1.Taint{
				Key:    kmmparams.TolerationNoScheduleKeyValue.Key,
				Value:  kmmparams.TolerationNoScheduleKeyValue.Value,
				Effect: kmmparams.TolerationNoScheduleKeyValue.Effect,
			})

			node.Definition.Spec.Taints = newTaint
			node.Update()
		})
	})
})
