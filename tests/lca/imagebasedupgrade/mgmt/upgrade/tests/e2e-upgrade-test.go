package upgrade_test

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/golang/glog"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/strings/slices"

	"github.com/openshift-kni/eco-goinfra/pkg/clusteroperator"
	"github.com/openshift-kni/eco-goinfra/pkg/clusterversion"
	"github.com/openshift-kni/eco-goinfra/pkg/configmap"
	"github.com/openshift-kni/eco-goinfra/pkg/deployment"
	"github.com/openshift-kni/eco-goinfra/pkg/kmm"
	"github.com/openshift-kni/eco-goinfra/pkg/lca"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/olm"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-goinfra/pkg/proxy"
	"github.com/openshift-kni/eco-goinfra/pkg/rbac"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"github.com/openshift-kni/eco-goinfra/pkg/route"
	"github.com/openshift-kni/eco-goinfra/pkg/service"
	"github.com/openshift-kni/eco-goinfra/pkg/serviceaccount"
	"github.com/openshift-kni/eco-gotests/tests/internal/cluster"
	"github.com/openshift-kni/eco-gotests/tests/internal/url"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/internal/nodestate"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/internal/safeapirequest"
	. "github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/mgmt/internal/mgmtinittools"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/mgmt/internal/mgmtparams"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/mgmt/upgrade/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/lca/internal/brutil"
	lcav1 "github.com/openshift-kni/lifecycle-agent/api/imagebasedupgrade/v1"
	oplmV1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	kmmv1beta1 "github.com/rh-ecosystem-edge/kernel-module-management/api/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8sScheme "k8s.io/client-go/kubernetes/scheme"
)

const (
	oadpContentConfigmap = "oadp-cm"

	extraManifestNamespace = "extranamespace"
	extraManifestConfigmap = "extra-configmap"

	extraManifestNamespaceConfigmapName = "extra-manifests-cm0"
	extraManifesConfigmapConfigmapName  = "extra-manifests-cm1"

	kmmManifestsConfigmapName  = "kmm-manifests-cm0"
	kmmModuleName              = "simple-kmod-existing-quay"
	kmmModuleNamespaceName     = "simple-kmod"
	kmmModuleServiceAccoutName = "simple-kmod-manager"
	kmmClusterRoleBindingName  = "simple-kmod-module-manager-rolebinding"
	kmmClusterRoleName         = "system:openshift:scc:privileged"
)

var (
	ibu *lca.ImageBasedUpgradeBuilder
	err error

	ibuWorkloadNamespace *namespace.Builder
	ibuWorkloadRoute     *route.Builder
)

var _ = Describe(
	"Performing image based upgrade",
	Ordered,
	Label(tsparams.LabelEndToEndUpgrade), func() {
		var (
			originalTargetProxy *proxy.Builder
		)

		BeforeAll(func() {

			By("Get target cluster proxy configuration")
			originalTargetProxy, err = proxy.Pull(APIClient)
			Expect(err).NotTo(HaveOccurred(), "error pulling target cluster proxy")

			By("Pull the imagebasedupgrade from the cluster")
			ibu, err = lca.PullImageBasedUpgrade(APIClient)
			Expect(err).NotTo(HaveOccurred(), "error pulling ibu resource from cluster")

			By("Ensure that imagebasedupgrade values are empty")
			ibu.Definition.Spec.ExtraManifests = []lcav1.ConfigMapRef{}
			ibu.Definition.Spec.OADPContent = []lcav1.ConfigMapRef{}
			ibu, err := ibu.Update()
			Expect(err).NotTo(HaveOccurred(), "error updating ibu resource with empty values")

			if findInstalledCSV("kernel-module-management") {
				glog.V(mgmtparams.MGMTLogLevel).Infof("KMM was installed")

				By("Create namespace definition for KMM module")
				kmmNamespace := namespace.NewBuilder(APIClient, kmmModuleNamespaceName)
				kmmNamespace.Definition.Annotations = map[string]string{
					"lca.openshift.io/apply-wave": "4",
				}

				By("Create string from the KMM module namespace definition")
				kmmNamespaceString, err := brutil.NewBackRestoreObject(
					kmmNamespace.Definition, k8sScheme.Scheme, v1.SchemeGroupVersion).String()
				Expect(err).NotTo(HaveOccurred(), "error creating configmap data for KMM namespace")

				By("Create serviceaccount definition for KMM module")
				kmmServiceAccount := serviceaccount.NewBuilder(APIClient, kmmModuleServiceAccoutName,
					kmmModuleNamespaceName)
				kmmServiceAccount.Definition.Annotations = map[string]string{
					"lca.openshift.io/apply-wave": "5",
				}

				By("Create string from the KMM module serviceaccount definition")
				kmmServiceAccountString, err := brutil.NewBackRestoreObject(
					kmmServiceAccount.Definition, k8sScheme.Scheme, v1.SchemeGroupVersion).String()
				Expect(err).NotTo(HaveOccurred(), "error creating configmap data for KMM serviceaccount")

				By("Create clusterrolebinding definition for KMM module")
				kmmClusterRoleBinding := rbac.NewClusterRoleBindingBuilder(APIClient, kmmClusterRoleBindingName,
					kmmClusterRoleName,
					rbacv1.Subject{
						Name:      kmmModuleServiceAccoutName,
						Kind:      "ServiceAccount",
						Namespace: kmmModuleNamespaceName,
					},
				)
				kmmClusterRoleBinding.Definition.Annotations = map[string]string{
					"lca.openshift.io/apply-wave": "6",
				}

				By("Create string from the KMM module clusterrolebinding definition")
				kmmClusterRoleBindingString, err := brutil.NewBackRestoreObject(
					kmmClusterRoleBinding.Definition, k8sScheme.Scheme, rbacv1.SchemeGroupVersion).String()
				Expect(err).NotTo(HaveOccurred(), "error creating configmap data for KMM clusterrolebinding")

				By("Create module definition for KMM")
				kmmKernelMappings := kmmv1beta1.KernelMapping{Regexp: "^.+$",
					ContainerImage: "quay.io/ocp-edge-qe/simple-kmod:$KERNEL_FULL_VERSION"}
				var kmmKernelMappingsList []kmmv1beta1.KernelMapping
				kmmMappings := append(kmmKernelMappingsList, kmmKernelMappings)

				kmmModule := kmm.NewModuleBuilder(APIClient, kmmModuleName, kmmModuleNamespaceName)
				kmmModule.Definition.Spec.ModuleLoader.ServiceAccountName = kmmModuleServiceAccoutName
				kmmModule.Definition.Spec.ModuleLoader.Container.Modprobe.ModuleName = kmmModuleNamespaceName
				kmmModule.Definition.Spec.ModuleLoader.Container.ImagePullPolicy = "Always"
				kmmModule.Definition.Spec.ModuleLoader.Container.KernelMappings = kmmMappings
				kmmModule.Definition.Spec.Selector = map[string]string{"node-role.kubernetes.io/worker": ""}

				By("Create string from the KMM module namespace definition")
				kmmModuleString, err := brutil.NewBackRestoreObject(
					kmmModule.Definition, APIClient.Scheme(), kmmv1beta1.GroupVersion).String()
				Expect(err).NotTo(HaveOccurred(), "error creating configmap data for KMM module")

				By("Create configmap with the KMM module manifests")
				kmmManifestsConfigmap, err := configmap.NewBuilder(
					APIClient, kmmManifestsConfigmapName, mgmtparams.LCANamespace).WithData(map[string]string{
					"namespace.yaml":          kmmNamespaceString,
					"serviceaccount.yaml":     kmmServiceAccountString,
					"clusterrolebinding.yaml": kmmClusterRoleBindingString,
					"kmmmodule.yaml":          kmmModuleString,
				}).Create()
				Expect(err).NotTo(HaveOccurred(), "error creating configmap with KMM module manifests")

				By("Update IBU with KMM module manifests")
				_, err = ibu.WithExtraManifests(
					kmmManifestsConfigmap.Object.Name, kmmManifestsConfigmap.Object.Namespace).Update()
				Expect(err).NotTo(HaveOccurred(), "error updating image based upgrade with kmm module manifests")
			}

			if MGMTConfig.ExtraManifests {
				By("Include user-defined catalogsources in IBU extraManifests")
				updateIBUWithCustomCatalogSources(ibu)

				By("Create namespace for extramanifests")
				extraNamespace := namespace.NewBuilder(APIClient, extraManifestNamespace)
				extraNamespace.Definition.Annotations = make(map[string]string)
				extraNamespace.Definition.Annotations["lca.openshift.io/apply-wave"] = "1"

				By("Create configmap for extra manifests namespace")
				extraNamespaceString, err := brutil.NewBackRestoreObject(
					extraNamespace.Definition, k8sScheme.Scheme, v1.SchemeGroupVersion).String()
				Expect(err).NotTo(HaveOccurred(), "error creating configmap data for extramanifest namespace")
				extraManifestsNamespaceConfigmap, err := configmap.NewBuilder(
					APIClient, extraManifestNamespaceConfigmapName, mgmtparams.LCANamespace).WithData(map[string]string{
					"namespace.yaml": extraNamespaceString,
				}).Create()
				Expect(err).NotTo(HaveOccurred(), "error creating configmap for extra manifests namespace")

				By("Create configmap for extramanifests")
				extraConfigmap := configmap.NewBuilder(
					APIClient, extraManifestConfigmap, extraManifestNamespace).WithData(map[string]string{
					"hello": "world",
				})
				extraConfigmap.Definition.Annotations = make(map[string]string)
				extraConfigmap.Definition.Annotations["lca.openshift.io/apply-wave"] = "2"

				By("Create configmap for extramanifests configmap")
				extraConfigmapString, err := brutil.NewBackRestoreObject(
					extraConfigmap.Definition, k8sScheme.Scheme, v1.SchemeGroupVersion).String()
				Expect(err).NotTo(HaveOccurred(), "error creating configmap data for extramanifest configmap")
				extraManifestsConfigmapConfigmap, err := configmap.NewBuilder(
					APIClient, extraManifesConfigmapConfigmapName, mgmtparams.LCANamespace).WithData(map[string]string{
					"configmap.yaml": extraConfigmapString,
				}).Create()
				Expect(err).NotTo(HaveOccurred(), "error creating configmap for extra manifests configmap")

				By("Update IBU with extra manifests")
				_, err = ibu.WithExtraManifests(
					extraManifestsNamespaceConfigmap.Object.Name, extraManifestsNamespaceConfigmap.Object.Namespace).
					WithExtraManifests(
						extraManifestsConfigmapConfigmap.Object.Name, extraManifestsConfigmapConfigmap.Object.Namespace).Update()
				Expect(err).NotTo(HaveOccurred(), "error updating image based upgrade with extra manifests")
			}

			By("Start test workload on IBU cluster")
			startTestWorkload()

			By("Create configmap for oadp")
			oadpConfigmap := configmap.NewBuilder(APIClient, oadpContentConfigmap, mgmtparams.LCAOADPNamespace)
			var oadpConfigmapData = make(map[string]string)

			By("Add workload app backup oadp configmap")
			workloadBackup, err := mgmtparams.WorkloadBackup.String()
			Expect(err).NotTo(HaveOccurred(), "error creating configmap data for workload app backup")
			oadpConfigmapData["workload_app_backup.yaml"] = workloadBackup

			By("Add workload app restore to oadp configmap")
			workloadRestore, err := mgmtparams.WorkloadRestore.String()
			Expect(err).NotTo(HaveOccurred(), "error creating configmap data for workload app restore")
			oadpConfigmapData["workload_app_restore.yaml"] = workloadRestore

			_, err = namespace.Pull(APIClient, mgmtparams.LCAKlusterletNamespace)
			if err == nil {
				By("Add klusterlet backup oadp configmap")
				klusterletBackup, err := mgmtparams.KlusterletBackup.String()
				Expect(err).NotTo(HaveOccurred(), "error creating configmap data for klusterlet backup content")
				oadpConfigmapData["klusterlet_backup.yaml"] = klusterletBackup

				By("Add klusterlet restore oadp configmap")
				klusterletRestore, err := mgmtparams.KlusterletRestore.String()
				Expect(err).NotTo(HaveOccurred(), "error creating configmap data for klusterlet restire content")
				oadpConfigmapData["klusterlet_restore.yaml"] = klusterletRestore
			}

			By("Create oadpContent configmap")
			_, err = oadpConfigmap.WithData(oadpConfigmapData).Create()
			Expect(err).NotTo(HaveOccurred(), "error creating oadp configmap")
		})

		AfterAll(func() {
			if MGMTConfig.IdlePostUpgrade && !MGMTConfig.RollbackAfterUpgrade {
				By("Set the IBU stage to Idle")

				_, err = ibu.WithStage("Idle").Update()
				Expect(err).NotTo(HaveOccurred(), "error setting ibu to idle stage")
			}

			if !MGMTConfig.IdlePostUpgrade && MGMTConfig.RollbackAfterUpgrade {
				By("Revert IBU resource back to Idle stage")
				ibu, err = lca.PullImageBasedUpgrade(APIClient)
				Expect(err).NotTo(HaveOccurred(), "error pulling imagebasedupgrade resource")

				if ibu.Object.Spec.Stage == "Upgrade" {
					By("Set IBU stage to Rollback")
					_, err = ibu.WithStage("Rollback").Update()
					Expect(err).NotTo(HaveOccurred(), "error setting ibu to rollback stage")

					By("Wait for IBU resource to be available")
					err = nodestate.WaitForIBUToBeAvailable(APIClient, ibu, time.Minute*10)
					Expect(err).NotTo(HaveOccurred(), "error waiting for ibu resource to become available")

					By("Wait until Rollback stage has completed")
					_, err = ibu.WaitUntilStageComplete("Rollback")
					Expect(err).NotTo(HaveOccurred(), "error waiting for rollback stage to complete")
				}

				if slices.Contains([]string{"Prep", "Rollback"}, string(ibu.Object.Spec.Stage)) {
					By("Set IBU stage to Idle")
					_, err = ibu.WithStage("Idle").Update()
					Expect(err).NotTo(HaveOccurred(), "error setting ibu to idle stage")

					By("Wait until IBU has become Idle")
					_, err = ibu.WaitUntilStageComplete("Idle")
					Expect(err).NotTo(HaveOccurred(), "error waiting for idle stage to complete")
				}

				Expect(string(ibu.Object.Spec.Stage)).To(Equal("Idle"), "error: ibu resource contains unexpected state")

				deleteTestWorkload()

				if MGMTConfig.ExtraManifests {
					By("Pull namespace extra manifests namespace")
					extraNamespace, err := namespace.Pull(APIClient, extraManifestNamespace)
					Expect(err).NotTo(HaveOccurred(), "error pulling namespace created by extra manifests")

					By("Delete extra manifest namespace")
					err = extraNamespace.DeleteAndWait(time.Minute * 1)
					Expect(err).NotTo(HaveOccurred(), "error deleting extra manifest namespace")

					By("Pull extra manifests namespace configmap")
					extraManifestsNamespaceConfigmap, err := configmap.Pull(
						APIClient, extraManifestNamespaceConfigmapName, mgmtparams.LCANamespace)
					Expect(err).NotTo(HaveOccurred(), "error pulling extra manifest namespace configmap")

					By("Delete extra manifests namespace configmap")
					err = extraManifestsNamespaceConfigmap.Delete()
					Expect(err).NotTo(HaveOccurred(), "error deleting extra manifest namespace configmap")

					By("Pull extra manifests configmap configmap")
					extraManifestsConfigmapConfigmap, err := configmap.Pull(
						APIClient, extraManifesConfigmapConfigmapName, mgmtparams.LCANamespace)
					Expect(err).NotTo(HaveOccurred(), "error pulling extra manifest configmap configmap")

					By("Delete extra manifests configmap configmap")
					err = extraManifestsConfigmapConfigmap.Delete()
					Expect(err).NotTo(HaveOccurred(), "error deleting extra manifest configmap configmap")
				}
			}
		})

		It("upgrades the connected cluster", reportxml.ID("71362"), func() {
			By("Check if the target cluster is connected")
			connected, err := cluster.Connected(APIClient)

			if !connected {
				Skip("Target cluster is disconnected")
			}

			if err != nil {
				Skip(fmt.Sprintf("Encountered an error while getting cluster connection info: %s", err.Error()))
			}

			upgrade()
		})

		It("upgrades the disconnected cluster", reportxml.ID("71736"), func() {
			By("Check if the target cluster is disconnected")
			disconnected, err := cluster.Disconnected(APIClient)

			if !disconnected {
				Skip("Target cluster is connected")
			}

			if err != nil {
				Skip(fmt.Sprintf("Encountered an error while getting cluster connection info: %s", err.Error()))
			}

			upgrade()
		})

		It("successfully creates extramanifests", reportxml.ID("71556"), func() {
			if !MGMTConfig.ExtraManifests {
				Skip("Cluster not upgraded with extra manifests")
			}

			By("Pull namespace created by extra manifests")
			extraNamespace, err := namespace.Pull(APIClient, extraManifestNamespace)
			Expect(err).NotTo(HaveOccurred(), "error pulling namespace created by extra manifests")

			By("Pull configmap created by extra manifests")
			extraConfigmap, err := configmap.Pull(APIClient, extraManifestConfigmap, extraNamespace.Object.Name)
			Expect(err).NotTo(HaveOccurred(), "error pulling configmap created by extra manifests")
			Expect(len(extraConfigmap.Object.Data)).To(Equal(1), "error: got unexpected data in configmap")
			Expect(extraConfigmap.Object.Data["hello"]).To(Equal("world"),
				"error: extra manifest configmap has incorrect content")
		})

		It("contains same proxy configuration as seed after upgrade", reportxml.ID("73103"), func() {
			if originalTargetProxy.Object.Spec.HTTPProxy == "" &&
				originalTargetProxy.Object.Spec.HTTPSProxy == "" &&
				originalTargetProxy.Object.Spec.NoProxy == "" {
				Skip("Target was not installed with proxy")
			}

			if originalTargetProxy.Object.Spec.HTTPProxy != MGMTConfig.SeedClusterInfo.Proxy.HTTPProxy ||
				originalTargetProxy.Object.Spec.HTTPSProxy != MGMTConfig.SeedClusterInfo.Proxy.HTTPSProxy {
				Skip("Target was not installed with the same proxy as seed")
			}

			targetProxyPostUpgrade, err := proxy.Pull(APIClient)
			Expect(err).NotTo(HaveOccurred(), "error pulling target proxy")
			Expect(originalTargetProxy.Object.Spec.HTTPProxy).To(Equal(targetProxyPostUpgrade.Object.Spec.HTTPProxy),
				"HTTP_PROXY postupgrade config does not match pre upgrade config")
			Expect(originalTargetProxy.Object.Spec.HTTPSProxy).To(Equal(targetProxyPostUpgrade.Object.Spec.HTTPSProxy),
				"HTTPS_PROXY postupgrade config does not match pre upgrade config")
			Expect(originalTargetProxy.Object.Spec.NoProxy).To(Equal(targetProxyPostUpgrade.Object.Spec.NoProxy),
				"NO_PROXY postupgrade config does not match pre upgrade config")
		})

		It("contains different proxy configuration than seed after upgrade", reportxml.ID("73369"), func() {
			if originalTargetProxy.Object.Spec.HTTPProxy == "" &&
				originalTargetProxy.Object.Spec.HTTPSProxy == "" &&
				originalTargetProxy.Object.Spec.NoProxy == "" {
				Skip("Target was not installed with proxy")
			}

			if originalTargetProxy.Object.Spec.HTTPProxy == MGMTConfig.SeedClusterInfo.Proxy.HTTPProxy &&
				originalTargetProxy.Object.Spec.HTTPSProxy == MGMTConfig.SeedClusterInfo.Proxy.HTTPSProxy {
				Skip("Target was installed with the same proxy as seed")
			}

			targetProxyPostUpgrade, err := proxy.Pull(APIClient)
			Expect(err).NotTo(HaveOccurred(), "error pulling target proxy")
			Expect(originalTargetProxy.Object.Spec.HTTPProxy).To(Equal(targetProxyPostUpgrade.Object.Spec.HTTPProxy),
				"HTTP_PROXY postupgrade config does not match pre upgrade config")
			Expect(originalTargetProxy.Object.Spec.HTTPSProxy).To(Equal(targetProxyPostUpgrade.Object.Spec.HTTPSProxy),
				"HTTPS_PROXY postupgrade config does not match pre upgrade config")
			Expect(originalTargetProxy.Object.Spec.NoProxy).To(Equal(targetProxyPostUpgrade.Object.Spec.NoProxy),
				"NO_PROXY postupgrade config does not match pre upgrade config")
		})

		It("fails because from Upgrade it's not possible to move to Prep stage", reportxml.ID("71741"), func() {
			By("Pull the imagebasedupgrade from the cluster")
			ibu, err = lca.PullImageBasedUpgrade(APIClient)
			Expect(err).NotTo(HaveOccurred(), "error pulling imagebasedupgrade resource")

			if ibu.Object.Spec.Stage != "Upgrade" {
				Skip("IBU is not in Upgrade stage")
			}

			_, err := ibu.WithStage("Prep").Update()
			Expect(err.Error()).To(ContainSubstring("the stage transition is not permitted"),
				"error: ibu seedimage updated with wrong next stage")
		})

		It("successfully loads kmm module", reportxml.ID("73457"), func() {
			if !findInstalledCSV("kernel-module-management") {
				Skip("Target was not installed with KMM")
			}

			By("Validate KMM module is loaded after upgrade", func() {
				execCmd := "lsmod"
				cmdOutput, err := cluster.ExecCmdWithStdout(APIClient, execCmd)
				Expect(err).ToNot(HaveOccurred(), "could not execute command: %s", err)

				for _, stdout := range cmdOutput {
					Expect(strings.ReplaceAll(stdout, "\n", "")).To(ContainSubstring("simple_kmod"),
						"error: simple_kmod wasn't loaded")

				}
			})
		})
	})

//nolint:funlen
func upgrade() {
	By("Updating the seed image reference")

	ibu, err = ibu.WithSeedImage(MGMTConfig.SeedImage).
		WithSeedImageVersion(MGMTConfig.SeedClusterInfo.SeedClusterOCPVersion).Update()
	Expect(err).NotTo(HaveOccurred(), "error updating ibu with image and version")

	By("Updating the oadpContent")

	ibu, err = ibu.WithOadpContent(oadpContentConfigmap, mgmtparams.LCAOADPNamespace).Update()
	Expect(err).NotTo(HaveOccurred(), "error updating ibu oadp content")

	By("Setting the IBU stage to Prep")

	_, err := ibu.WithStage("Prep").Update()
	Expect(err).NotTo(HaveOccurred(), "error setting ibu to prep stage")

	By("Wait until Prep stage has completed")

	_, err = ibu.WaitUntilStageComplete("Prep")
	Expect(err).NotTo(HaveOccurred(), "error waiting for prep stage to complete")

	By("Get list of nodes to be upgraded")

	ibuNodes, err := nodes.List(APIClient)
	Expect(err).NotTo(HaveOccurred(), "error listing nodes")

	By("Set the IBU stage to Upgrade")

	_, err = ibu.WithStage("Upgrade").Update()
	Expect(err).NotTo(HaveOccurred(), "error setting ibu to upgrade stage")

	By("Wait for nodes to become unreachable")

	for _, node := range ibuNodes {
		unreachable, err := nodestate.WaitForNodeToBeUnreachable(node.Object.Name, "6443", time.Minute*15)

		Expect(err).To(BeNil(), "error waiting for %s node to shutdown", node.Object.Name)
		Expect(unreachable).To(BeTrue(), "error: node %s is still reachable", node.Object.Name)
	}

	By("Wait for nodes to become reachable")

	for _, node := range ibuNodes {
		reachable, err := nodestate.WaitForNodeToBeReachable(node.Object.Name, "6443", time.Minute*20)

		Expect(err).To(BeNil(), "error waiting for %s node to become reachable", node.Object.Name)
		Expect(reachable).To(BeTrue(), "error: node %s is still unreachable", node.Object.Name)
	}

	By("Wait until all nodes are reporting as Ready")

	err = safeapirequest.Do(func() error {
		_, err := nodes.WaitForAllNodesAreReady(APIClient, time.Minute*10)

		return err
	})
	Expect(err).To(BeNil(), "error waiting for nodes to become ready")

	By("Wait for IBU resource to be available")

	err = nodestate.WaitForIBUToBeAvailable(APIClient, ibu, time.Minute*10)
	Expect(err).NotTo(HaveOccurred(), "error waiting for ibu resource to become available")

	By("Wait until Upgrade stage has completed")

	ibu, err = ibu.WaitUntilStageComplete("Upgrade")
	Expect(err).NotTo(HaveOccurred(), "error waiting for upgrade stage to complete")

	By("Check the clusterversion matches seedimage version")

	clusterVersion, err := clusterversion.Pull(APIClient)
	Expect(err).NotTo(HaveOccurred(), "error pulling clusterversion")
	Expect(MGMTConfig.SeedClusterInfo.SeedClusterOCPVersion).To(
		Equal(clusterVersion.Object.Status.Desired.Version), "error: clusterversion does not match seedimageversion")

	By("Check that no cluster operators are progressing")

	cosStoppedProgressing, err := clusteroperator.WaitForAllClusteroperatorsStopProgressing(APIClient, time.Minute*5)
	Expect(err).NotTo(HaveOccurred(), "error while waiting for cluster operators to stop progressing")
	Expect(cosStoppedProgressing).To(BeTrue(), "error: some cluster operators are still progressing")

	By("Check that all cluster operators are available")

	cosAvailable, err := clusteroperator.WaitForAllClusteroperatorsAvailable(APIClient, time.Minute*5)
	Expect(err).NotTo(HaveOccurred(), "error while waiting for cluster operators to become available")
	Expect(cosAvailable).To(BeTrue(), "error: some cluster operators are not available")

	By("Check that all pods are running in workload namespace")

	workloadPods, err := pod.List(APIClient, mgmtparams.LCAWorkloadName)
	Expect(err).NotTo(HaveOccurred(), "error listing pods in workload namespace %s", mgmtparams.LCAWorkloadName)
	Expect(len(workloadPods) > 0).To(BeTrue(),
		"error: found no running pods in workload namespace %s", mgmtparams.LCAWorkloadName)

	for _, workloadPod := range workloadPods {
		err := workloadPod.WaitUntilReady(time.Minute * 2)
		Expect(err).To(BeNil(), "error waiting for workload pod to become ready")
	}

	verifyIBUWorkloadReachable()

	_, err = namespace.Pull(APIClient, mgmtparams.LCAKlusterletNamespace)
	if err == nil {
		By("Check that all pods are running in klusterlet namespace")

		klusterletPods, err := pod.List(APIClient, mgmtparams.LCAKlusterletNamespace)
		Expect(err).NotTo(HaveOccurred(), "error listing pods in kusterlet namespace %s", mgmtparams.LCAKlusterletNamespace)
		Expect(len(klusterletPods) > 0).To(BeTrue(),
			"error: found no running pods in klusterlet namespace %s", mgmtparams.LCAKlusterletNamespace)

		for _, klusterletPod := range klusterletPods {
			// We check if the pod is terminataing or if it still exists to
			// mitigate situations where a leftover pod is still found but gets removed.
			if klusterletPod.Object.Status.Phase != "Terminating" {
				err := klusterletPod.WaitUntilReady(time.Minute * 2)
				if klusterletPod.Exists() {
					Expect(err).To(BeNil(), "error waiting for klusterlet pod to become ready")
				}
			}
		}
	}
}

func updateIBUWithCustomCatalogSources(imagebasedupgrade *lca.ImageBasedUpgradeBuilder) {
	catalogSources, err := olm.ListCatalogSources(APIClient, "openshift-marketplace")
	Expect(err).NotTo(HaveOccurred(), "error listing catalogsources in openshift-marketplace namespace")

	omitCatalogRegex := regexp.MustCompile(`(redhat|certified|community)-(operators|marketplace)`)

	for _, catalogSource := range catalogSources {
		if !omitCatalogRegex.MatchString(catalogSource.Object.Name) {
			configmapData, err := brutil.NewBackRestoreObject(
				catalogSource.Object, APIClient.Scheme(), oplmV1alpha1.SchemeGroupVersion).String()
			Expect(err).NotTo(HaveOccurred(), "error creating configmap data from catalogsource content")

			By("Create configmap with catalogsource information")

			_, err = configmap.NewBuilder(APIClient,
				fmt.Sprintf("%s-configmap", catalogSource.Object.Name), mgmtparams.LCANamespace).WithData(
				map[string]string{
					fmt.Sprintf("99-%s-catalogsource", catalogSource.Object.Name): configmapData,
				}).Create()
			Expect(err).NotTo(HaveOccurred(), "error creating configmap from user-defined catalogsource")

			By("Updating IBU to include configmap")
			imagebasedupgrade.WithExtraManifests(fmt.Sprintf("%s-configmap", catalogSource.Object.Name), mgmtparams.LCANamespace)
		}
	}
}

func startTestWorkload() {
	By("Check if workload app namespace exists")

	if ibuWorkloadNamespace, err = namespace.Pull(APIClient, mgmtparams.LCAWorkloadName); err == nil {
		deleteTestWorkload()
	}

	By("Create workload app namespace")

	ibuWorkloadNamespace, err = namespace.NewBuilder(APIClient, mgmtparams.LCAWorkloadName).Create()
	Expect(err).NotTo(HaveOccurred(), "error creating namespace for ibu workload app")

	By("Create workload app deployment")

	_, err = deployment.NewBuilder(
		APIClient, mgmtparams.LCAWorkloadName, mgmtparams.LCAWorkloadName, map[string]string{
			"app": mgmtparams.LCAWorkloadName,
		}, v1.Container{
			Name:  mgmtparams.LCAWorkloadName,
			Image: MGMTConfig.IBUWorkloadImage,
			Ports: []v1.ContainerPort{
				{
					Name:          "http",
					ContainerPort: 8080,
				},
			},
		}).WithLabel("app", mgmtparams.LCAWorkloadName).CreateAndWaitUntilReady(time.Second * 60)
	Expect(err).NotTo(HaveOccurred(), "error creating ibu workload deployment")

	By("Create workload app service")

	_, err = service.NewBuilder(
		APIClient, mgmtparams.LCAWorkloadName, mgmtparams.LCAWorkloadName, map[string]string{
			"app": mgmtparams.LCAWorkloadName,
		}, v1.ServicePort{
			Protocol: v1.ProtocolTCP,
			Port:     8080,
		}).Create()
	Expect(err).NotTo(HaveOccurred(), "error creating ibu workload service")

	By("Create workload app route")

	ibuWorkloadRoute, err = route.NewBuilder(
		APIClient, mgmtparams.LCAWorkloadName, mgmtparams.LCAWorkloadName, mgmtparams.LCAWorkloadName).Create()
	Expect(err).NotTo(HaveOccurred(), "error creating ibu workload route")

	verifyIBUWorkloadReachable()
}

func deleteTestWorkload() {
	By("Delete ibu workload namespace")

	err := ibuWorkloadNamespace.DeleteAndWait(time.Second * 30)
	Expect(err).NotTo(HaveOccurred(), "error deleting ibu workload namespace")
}

func verifyIBUWorkloadReachable() {
	By("Verify IBU workload is reachable")

	err := wait.PollUntilContextTimeout(
		context.TODO(), time.Second*2, time.Second*10, true, func(ctx context.Context) (bool, error) {
			_, rc, _ := url.Fetch(fmt.Sprintf("http://%s", ibuWorkloadRoute.Object.Spec.Host), "get", false)

			return rc == 200, nil
		},
	)

	Expect(err).NotTo(HaveOccurred(), "error reaching ibu workload")
}

func findInstalledCSV(expectedCSV string) bool {
	csvList, err := olm.ListClusterServiceVersionInAllNamespaces(APIClient)
	Expect(err).NotTo(HaveOccurred(), "error retrieving the list of CSV")

	for _, csv := range csvList {
		if strings.Contains(csv.Object.Name, expectedCSV) {
			return true
		}
	}

	return false
}
