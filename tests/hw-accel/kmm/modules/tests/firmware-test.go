package tests

import (
	"fmt"
	"time"

	"github.com/openshift-kni/eco-goinfra/pkg/mco"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/modules/internal/get"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/kmm"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/serviceaccount"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/modules/internal/await"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/modules/internal/check"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/modules/internal/define"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/modules/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"

	. "github.com/openshift-kni/eco-gotests/tests/internal/inittools"

	"github.com/openshift-kni/eco-goinfra/pkg/configmap"
)

var _ = Describe("KMM", Ordered, Label(tsparams.LabelSuite), func() {

	Context("Module", Label("firmware"), func() {

		moduleName := tsparams.FirmwareTestNamespace
		kmodName := "simple-kmod-firmware"
		serviceAccountName := "firmware-manager"
		image := fmt.Sprintf("%s/%s/%s:$KERNEL_FULL_VERSION",
			tsparams.LocalImageRegistry, tsparams.FirmwareTestNamespace, kmodName)
		machineConfigName := "99-worker-kernel-args-firmware-path"
		machineConfigRole := "machineconfiguration.openshift.io/role"
		workerKernelArgs := []string{"firmware_class.path=/var/lib/firmware"}
		mcpName := get.MachineConfigPoolName(APIClient)

		AfterEach(func() {
			By("Delete Module")
			_, _ = kmm.NewModuleBuilder(APIClient, moduleName, tsparams.FirmwareTestNamespace).Delete()

			svcAccount := serviceaccount.NewBuilder(APIClient, serviceAccountName, tsparams.FirmwareTestNamespace)
			svcAccount.Exists()

			By("Delete ClusterRoleBinding")
			crb := define.ModuleCRB(*svcAccount, kmodName)
			_ = crb.Delete()

			By("Delete Namespace")
			_ = namespace.NewBuilder(APIClient, tsparams.FirmwareTestNamespace).Delete()

			By("Delete machine configuration that sets Kernel Arguments on workers")
			kernelArgsMc, err := mco.PullMachineConfig(APIClient, machineConfigName)
			Expect(err).ToNot(HaveOccurred(), "error fetching machine configuration object")
			_ = kernelArgsMc.Delete()

			By("Waiting machine config pool to update")
			mcp, err := mco.Pull(APIClient, mcpName)
			Expect(err).ToNot(HaveOccurred(), "error while pulling machineconfigpool")

			err = mcp.WaitToBeStableFor(time.Minute, 2*time.Minute)
			Expect(err).To(HaveOccurred(), "the machine configuration did not trigger a mcp update")

			err = mcp.WaitForUpdate(30 * time.Minute)
			Expect(err).ToNot(HaveOccurred(), "error while waiting machineconfigpool to get updated")

		})

		It("should properly build a module with firmware support", polarion.ID("56675"), func() {

			By("Create Namespace")
			testNamespace, err := namespace.NewBuilder(APIClient, tsparams.FirmwareTestNamespace).Create()
			Expect(err).ToNot(HaveOccurred(), "error creating test namespace")

			configmapContents := define.SimpleKmodFirmwareConfigMapContents()

			By("Create ConfigMap")
			dockerfileConfigMap, err := configmap.
				NewBuilder(APIClient, kmodName, testNamespace.Object.Name).
				WithData(configmapContents).Create()
			Expect(err).ToNot(HaveOccurred(), "error creating configmap")

			By("Create ServiceAccount")
			svcAccount, err := serviceaccount.
				NewBuilder(APIClient, serviceAccountName, tsparams.FirmwareTestNamespace).Create()
			Expect(err).ToNot(HaveOccurred(), "error creating serviceaccount")

			By("Create ClusterRoleBinding")
			crb := define.ModuleCRB(*svcAccount, kmodName)
			_, err = crb.Create()
			Expect(err).ToNot(HaveOccurred(), "error creating clusterrolebinding")

			By("Creating machine configuration that sets the kernelArguments")
			kernelArgsMc := mco.NewMCBuilder(APIClient, machineConfigName).
				WithLabel(machineConfigRole, mcpName).
				WithKernelArguments(workerKernelArgs)
			_, err = kernelArgsMc.Create()
			Expect(err).ToNot(HaveOccurred(), "error creating machine configuration")

			By("Waiting machine config pool to update")
			mcp, err := mco.Pull(APIClient, "worker")
			Expect(err).ToNot(HaveOccurred(), "error while pulling machineconfigpool")

			err = mcp.WaitToBeStableFor(time.Minute, 2*time.Minute)
			Expect(err).To(HaveOccurred(), "the machineconfiguration did not trigger a mcp update")

			err = mcp.WaitForUpdate(30 * time.Minute)
			Expect(err).ToNot(HaveOccurred(), "error while waiting machineconfigpool to get updated")

			By("Create KernelMapping")
			kernelMapping := kmm.NewRegExKernelMappingBuilder("^.+$")

			kernelMapping.WithContainerImage(image).
				WithBuildArg("KVER", "$KERNEL_VERSION").
				WithBuildArg("KMODVER", "0.0.1").
				WithBuildDockerCfgFile(dockerfileConfigMap.Object.Name)
			kerMapOne, err := kernelMapping.BuildKernelMappingConfig()
			Expect(err).ToNot(HaveOccurred(), "error creating kernel mapping")

			By("Create ModuleLoaderContainer")
			moduleLoaderContainer := kmm.NewModLoaderContainerBuilder(kmodName)
			moduleLoaderContainer.WithModprobeSpec("/opt", "/firmware", []string{}, []string{}, []string{})
			moduleLoaderContainer.WithKernelMapping(kerMapOne)
			moduleLoaderContainer.WithImagePullPolicy("Always")

			moduleLoaderContainerCfg, err := moduleLoaderContainer.BuildModuleLoaderContainerCfg()
			Expect(err).ToNot(HaveOccurred(), "error creating moduleloadercontainer")

			By("Create Module")
			module := kmm.NewModuleBuilder(APIClient, moduleName, tsparams.FirmwareTestNamespace).
				WithNodeSelector(GeneralConfig.WorkerLabelMap)
			module = module.WithModuleLoaderContainer(moduleLoaderContainerCfg).
				WithLoadServiceAccount(svcAccount.Object.Name)
			_, err = module.Create()
			Expect(err).ToNot(HaveOccurred(), "error creating module")

			By("Await build pod to complete build")
			err = await.BuildPodCompleted(APIClient, tsparams.FirmwareTestNamespace, 5*time.Minute)
			Expect(err).ToNot(HaveOccurred(), "error while building module")

			By("Await driver container deployment")
			err = await.ModuleDeployment(APIClient, moduleName, tsparams.FirmwareTestNamespace, time.Minute,
				GeneralConfig.WorkerLabelMap)
			Expect(err).ToNot(HaveOccurred(), "error while waiting on driver deployment")

			By("Check module is loaded on node")
			err = check.ModuleLoaded(APIClient, kmodName, tsparams.FirmwareTestNamespace, time.Minute)
			Expect(err).ToNot(HaveOccurred(), "error while checking the module is loaded")

			By("Check dmesg contains module message")
			err = check.Dmesg(APIClient, "ALL GOOD WITH FIRMWARE", tsparams.FirmwareTestNamespace, time.Minute)
			Expect(err).ToNot(HaveOccurred(), "error while checking dmesg contents")

			By("Check label is set on all nodes")
			_, err = check.NodeLabel(APIClient, moduleName, GeneralConfig.WorkerLabelMap)
			Expect(err).ToNot(HaveOccurred(), "error while checking the module is loaded")

		})
	})
})
