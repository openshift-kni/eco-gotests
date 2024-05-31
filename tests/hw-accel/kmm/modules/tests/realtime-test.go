package tests

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/configmap"
	"github.com/openshift-kni/eco-goinfra/pkg/kmm"
	"github.com/openshift-kni/eco-goinfra/pkg/mco"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/nto" //nolint:misspell
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"github.com/openshift-kni/eco-goinfra/pkg/serviceaccount"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/modules/internal/await"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/modules/internal/check"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/modules/internal/define"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/modules/internal/get"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/modules/internal/tsparams"
	. "github.com/openshift-kni/eco-gotests/tests/internal/inittools"
)

var _ = Describe("KMM", Ordered, Label(tsparams.LabelSuite, tsparams.LabelLongRun), func() {

	Context("Module", Label("use-rt-kernel"), func() {
		var mcpName string

		moduleName := tsparams.RealtimeKernelNamespace
		kmodName := "module-rt"
		serviceAccountName := "rtkernel-manager"
		image := fmt.Sprintf("%s/%s/%s:$KERNEL_FULL_VERSION",
			tsparams.LocalImageRegistry, tsparams.RealtimeKernelNamespace, kmodName)
		buildArgValue := fmt.Sprintf("%s.o", kmodName)
		performanceProfileName := "rt-profile"
		rtCPUIsolated := "1,3,5,7"
		rtCPUReserved := "0,2,4,6"

		BeforeAll(func() {
			By("Detect if we can run test on architecture")
			arch, err := get.ClusterArchitecture(APIClient, GeneralConfig.WorkerLabelMap)

			if err != nil {
				Skip("could not detect cluster architecture")
			}

			if arch == "arm64" {
				Skip("ARM platform does not support realtime kernel.")
			}

			By("Collect MachineConfigPoolName")
			mcpName = get.MachineConfigPoolName(APIClient)
		})

		AfterEach(func() {
			By("Delete Module")
			_, _ = kmm.NewModuleBuilder(APIClient, moduleName, tsparams.RealtimeKernelNamespace).Delete()

			By("Await module to be deleted")
			err := await.ModuleObjectDeleted(APIClient, moduleName, tsparams.RealtimeKernelNamespace, time.Minute)
			Expect(err).ToNot(HaveOccurred(), "error while waiting module to be deleted")

			svcAccount := serviceaccount.NewBuilder(APIClient, serviceAccountName, tsparams.RealtimeKernelNamespace)
			svcAccount.Exists()

			By("Delete ClusterRoleBinding")
			crb := define.ModuleCRB(*svcAccount, kmodName)
			_ = crb.Delete()

			By("Delete Namespace")
			_ = namespace.NewBuilder(APIClient, tsparams.RealtimeKernelNamespace).Delete()

			By("Delete performance profile that sets Realtime Kernel on workers")
			realtimeProfile := nto.NewBuilder(APIClient, performanceProfileName,
				rtCPUIsolated, rtCPUReserved, GeneralConfig.WorkerLabelMap)
			_, _ = realtimeProfile.Delete()

			By("Waiting machine config pool to update")
			mcp, err := mco.Pull(APIClient, mcpName)
			Expect(err).ToNot(HaveOccurred(), "error while pulling machineconfigpool")

			err = mcp.WaitToBeStableFor(time.Minute, 2*time.Minute)
			Expect(err).To(HaveOccurred(), "the performance profile delete did not triggered a mcp update")

			err = mcp.WaitForUpdate(30 * time.Minute)
			Expect(err).ToNot(HaveOccurred(), "error while waiting machineconfigpool to get updated")

		})

		It("should properly build a module on Realtime Kernel", reportxml.ID("53656"), func() {

			By("Create Namespace")
			testNamespace, err := namespace.NewBuilder(APIClient, tsparams.RealtimeKernelNamespace).Create()
			Expect(err).ToNot(HaveOccurred(), "error creating test namespace")

			configmapContents := define.MultiStageConfigMapContent(kmodName)

			By("Create ConfigMap")
			dockerfileConfigMap, err := configmap.
				NewBuilder(APIClient, kmodName, testNamespace.Object.Name).
				WithData(configmapContents).Create()
			Expect(err).ToNot(HaveOccurred(), "error creating configmap")

			By("Create ServiceAccount")
			svcAccount, err := serviceaccount.
				NewBuilder(APIClient, serviceAccountName, tsparams.RealtimeKernelNamespace).Create()
			Expect(err).ToNot(HaveOccurred(), "error creating serviceaccount")

			By("Create ClusterRoleBinding")
			crb := define.ModuleCRB(*svcAccount, kmodName)
			_, err = crb.Create()
			Expect(err).ToNot(HaveOccurred(), "error creating clusterrolebinding")

			By("Creating performance profile that sets Realtime Kernel on workers")
			realtimeProfile := nto.NewBuilder(APIClient, performanceProfileName,
				rtCPUIsolated, rtCPUReserved, GeneralConfig.WorkerLabelMap).WithRTKernel()
			_, err = realtimeProfile.Create()
			Expect(err).ToNot(HaveOccurred(), "error creating realtime performance profile")

			By("Waiting machine config pool to update")
			mcp, err := mco.Pull(APIClient, mcpName)
			Expect(err).ToNot(HaveOccurred(), "error while pulling machineconfigpool")

			err = mcp.WaitToBeStableFor(time.Minute, 2*time.Minute)
			Expect(err).To(HaveOccurred(), "the performance profile did not triggered a mcp update")

			err = mcp.WaitForUpdate(30 * time.Minute)
			Expect(err).ToNot(HaveOccurred(), "error while waiting machineconfigpool to get updated")

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
			module := kmm.NewModuleBuilder(APIClient, moduleName, tsparams.RealtimeKernelNamespace).
				WithNodeSelector(GeneralConfig.WorkerLabelMap)
			module = module.WithModuleLoaderContainer(moduleLoaderContainerCfg).
				WithLoadServiceAccount(svcAccount.Object.Name)
			_, err = module.Create()
			Expect(err).ToNot(HaveOccurred(), "error creating module")

			By("Await build pod to complete build")
			err = await.BuildPodCompleted(APIClient, tsparams.RealtimeKernelNamespace, 5*time.Minute)
			Expect(err).ToNot(HaveOccurred(), "error while building module")

			By("Await driver container deployment")
			err = await.ModuleDeployment(APIClient, moduleName, tsparams.RealtimeKernelNamespace, time.Minute,
				GeneralConfig.WorkerLabelMap)
			Expect(err).ToNot(HaveOccurred(), "error while waiting on driver deployment")

			By("Check module is loaded on node")
			err = check.ModuleLoaded(APIClient, kmodName, time.Minute)
			Expect(err).ToNot(HaveOccurred(), "error while checking the module is loaded")

			By("Check label is set on all nodes")
			_, err = check.NodeLabel(APIClient, moduleName, tsparams.RealtimeKernelNamespace,
				GeneralConfig.WorkerLabelMap)
			Expect(err).ToNot(HaveOccurred(), "error while checking the module is loaded")

		})
	})
})
