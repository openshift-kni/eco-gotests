package upgrade_test

import (
	"context"
	"fmt"
	"regexp"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/strings/slices"

	"github.com/openshift-kni/eco-goinfra/pkg/clusteroperator"
	"github.com/openshift-kni/eco-goinfra/pkg/clusterversion"
	"github.com/openshift-kni/eco-goinfra/pkg/configmap"
	"github.com/openshift-kni/eco-goinfra/pkg/deployment"
	"github.com/openshift-kni/eco-goinfra/pkg/lca"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/olm"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-goinfra/pkg/route"
	"github.com/openshift-kni/eco-goinfra/pkg/service"
	"github.com/openshift-kni/eco-goinfra/pkg/velero"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/internal/nodestate"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/internal/safeapirequest"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/mgmt/internal/configmapgenerator"
	. "github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/mgmt/internal/mgmtinittools"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/mgmt/upgrade/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/lca/internal/url"
	"github.com/openshift-kni/lifecycle-agent/api/v1alpha1"
	oplmV1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	veleroScheme "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned/scheme"
)

const (
	oadpContentConfigmap = "oadp-cm"
	ibuKlusterletName    = "acm-klusterlet"
)

var (
	ibu *lca.ImageBasedUpgradeBuilder
	err error

	ibuWorkloadNamespace *namespace.Builder
	ibuWorkloadRoute     *route.Builder
	ibuWorkloadBackup    *velero.BackupBuilder
	ibuWorkloadRestore   *velero.RestoreBuilder

	ibuKlusterLetBackup  *velero.BackupBuilder
	ibuKlusterLetRestore *velero.RestoreBuilder
)

var _ = Describe(
	"Performing image based upgrade",
	Ordered,
	Label(tsparams.LabelEndToEndUpgrade), func() {

		BeforeAll(func() {
			By("Pull the imagebasedupgrade from the cluster")
			ibu, err = lca.PullImageBasedUpgrade(APIClient)
			Expect(err).NotTo(HaveOccurred(), "error pulling ibu resource from cluster")

			By("Ensure that imagebasedupgrade values are empty")
			ibu.Definition.Spec.ExtraManifests = []v1alpha1.ConfigMapRef{}
			ibu.Definition.Spec.OADPContent = []v1alpha1.ConfigMapRef{}
			ibu, err := ibu.Update()
			Expect(err).NotTo(HaveOccurred(), "error updating ibu resource with empty values")

			By("Include user-defined catalogsources in IBU extraManifests")
			catalogSources, err := olm.ListCatalogSources(APIClient, "openshift-marketplace")
			Expect(err).NotTo(HaveOccurred(), "error listing catalogsources in openshift-marketplace namespace")

			omitCatalogRegex := regexp.MustCompile(`(redhat|certified|community)-(operators|marketplace)`)

			for _, catalogSource := range catalogSources {
				if !omitCatalogRegex.MatchString(catalogSource.Object.Name) {

					configmapData, err := configmapgenerator.DataFromDefinition(APIClient.Scheme(),
						catalogSource.Object, oplmV1alpha1.SchemeGroupVersion)
					Expect(err).NotTo(HaveOccurred(), "error creating configmap data from catalogsource content")

					By("Create configmap with catalogsource information")
					_, err = configmap.NewBuilder(APIClient,
						fmt.Sprintf("%s-configmap", catalogSource.Object.Name), tsparams.LCANamespace).WithData(
						map[string]string{
							fmt.Sprintf("99-%s-catalogsource", catalogSource.Object.Name): configmapData,
						}).Create()
					Expect(err).NotTo(HaveOccurred(), "error creating configmap from user-defined catalogsource")

					By("Updating IBU to include configmap")
					ibu.WithExtraManifests(fmt.Sprintf("%s-configmap", catalogSource.Object.Name), tsparams.LCANamespace)
				}
			}

			if len(ibu.Definition.Spec.ExtraManifests) > 0 {
				_, err := ibu.Update()
				Expect(err).NotTo(HaveOccurred(), "error updating imagebasedupgrade with extramanifests")
			}

			By("Start test workload on IBU cluster")
			startTestWorkload()

			By("Define test workload backup and restore")
			defineWorkloadBackupRestore()

			By("Define klusterlet backup and restore")
			defineKlusterletBackupRestore()

			By("Create configmap for oadp")
			oadpConfigmap := configmap.NewBuilder(APIClient, oadpContentConfigmap, tsparams.LCAOADPNamespace)
			var oadpConfigmapData = make(map[string]string)

			By("Add workload app backup to configmap")
			configmapData, err := configmapgenerator.DataFromDefinition(
				veleroScheme.Scheme, ibuWorkloadBackup.Definition, velerov1.SchemeGroupVersion)
			Expect(err).NotTo(HaveOccurred(), "error creating configmap data for workload app backup")
			oadpConfigmapData["backup_app.yaml"] = configmapData

			By("Add workload app restore to configmap")
			configmapData, err = configmapgenerator.DataFromDefinition(
				veleroScheme.Scheme, ibuWorkloadRestore.Definition, velerov1.SchemeGroupVersion)
			Expect(err).NotTo(HaveOccurred(), "error creating configmap data for workload app restore")
			oadpConfigmapData["restore_app.yaml"] = configmapData

			if ibuKlusterLetBackup != nil {
				By("Add klusterlet backup to configmap")
				configmapData, err := configmapgenerator.DataFromDefinition(
					veleroScheme.Scheme, ibuKlusterLetBackup.Definition, velerov1.SchemeGroupVersion)
				Expect(err).NotTo(HaveOccurred(), "error creating configmap data for workload app backup")
				oadpConfigmapData["backup_klusterlet.yaml"] = configmapData

				By("Add klusterlet restore to configmap")
				configmapData, err = configmapgenerator.DataFromDefinition(
					veleroScheme.Scheme, ibuKlusterLetRestore.Definition, velerov1.SchemeGroupVersion)
				Expect(err).NotTo(HaveOccurred(), "error creating configmap data for workload app restore")
				oadpConfigmapData["restore_klusterlet.yaml"] = configmapData

			}

			By("Create oadpContent configmap")
			_, err = oadpConfigmap.WithData(oadpConfigmapData).Create()
			Expect(err).NotTo(HaveOccurred(), "error creating oadp configmap")
		})

		AfterAll(func() {
			if !MGMTConfig.IdlePostUpgrade {
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
			}
		})

		It("upgrades the cluster", func() {

			By("Updating the seed image reference")
			ibu, err = ibu.WithSeedImage(MGMTConfig.SeedImage).WithSeedImageVersion(MGMTConfig.SeedImageVersion).Update()
			Expect(err).NotTo(HaveOccurred(), "error updating ibu with image and version")

			By("Updating the oadpContent")
			ibu, err = ibu.WithOadpContent(oadpContentConfigmap, tsparams.LCAOADPNamespace).Update()
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
			Expect(MGMTConfig.SeedImageVersion).To(
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
			workloadPods, err := pod.List(APIClient, tsparams.LCAWorkloadName)
			Expect(err).NotTo(HaveOccurred(), "error listing pods in workload namespace %s", tsparams.LCAWorkloadName)
			Expect(len(workloadPods) > 0).To(BeTrue(),
				"error: found no running pods in workload namespace %s", tsparams.LCAWorkloadName)
			for _, workloadPod := range workloadPods {
				err := workloadPod.WaitUntilReady(time.Minute * 2)
				Expect(err).To(BeNil(), "error waiting for workload pod to become ready")
			}
			verifyIBUWorkloadReachable()

			if ibuKlusterLetBackup != nil {
				By("Check that all pods are running in klusterlet namespace")
				klusterletPods, err := pod.List(APIClient, tsparams.LCAKlusterletNamespace)
				Expect(err).NotTo(HaveOccurred(), "error listing pods in kusterlet namespace %s", tsparams.LCAKlusterletNamespace)
				Expect(len(klusterletPods) > 0).To(BeTrue(),
					"error: found no running pods in klusterlet namespace %s", tsparams.LCAKlusterletNamespace)
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

			if MGMTConfig.IdlePostUpgrade {
				By("Set the IBU stage to Idle")
				_, err = ibu.WithStage("Idle").Update()
				Expect(err).NotTo(HaveOccurred(), "error setting ibu to idle stage")
			}
		})
	})

func startTestWorkload() {
	By("Check if workload app namespace exists")

	if ibuWorkloadNamespace, err = namespace.Pull(APIClient, tsparams.LCAWorkloadName); err == nil {
		deleteTestWorkload()
	}

	By("Create workload app namespace")

	ibuWorkloadNamespace, err = namespace.NewBuilder(APIClient, tsparams.LCAWorkloadName).Create()
	Expect(err).NotTo(HaveOccurred(), "error creating namespace for ibu workload app")

	By("Create workload app deployment")

	_, err = deployment.NewBuilder(
		APIClient, tsparams.LCAWorkloadName, tsparams.LCAWorkloadName, map[string]string{
			"app": tsparams.LCAWorkloadName,
		}, &v1.Container{
			Name:  tsparams.LCAWorkloadName,
			Image: MGMTConfig.IBUWorkloadImage,
			Ports: []v1.ContainerPort{
				{
					Name:          "http",
					ContainerPort: 8080,
				},
			},
		}).WithLabel("app", tsparams.LCAWorkloadName).CreateAndWaitUntilReady(time.Second * 30)
	Expect(err).NotTo(HaveOccurred(), "error creating ibu workload deployment")

	By("Create workload app service")

	_, err = service.NewBuilder(
		APIClient, tsparams.LCAWorkloadName, tsparams.LCAWorkloadName, map[string]string{
			"app": tsparams.LCAWorkloadName,
		}, v1.ServicePort{
			Protocol: v1.ProtocolTCP,
			Port:     8080,
		}).Create()
	Expect(err).NotTo(HaveOccurred(), "error creating ibu workload service")

	By("Create workload app route")

	ibuWorkloadRoute, err = route.NewBuilder(
		APIClient, tsparams.LCAWorkloadName, tsparams.LCAWorkloadName, tsparams.LCAWorkloadName).Create()
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

func defineWorkloadBackupRestore() {
	By("Define workload backup")

	ibuWorkloadBackup = velero.NewBackupBuilder(
		APIClient, tsparams.LCAWorkloadName, tsparams.LCAOADPNamespace).WithStorageLocation("default").
		WithIncludedNamespace(tsparams.LCAWorkloadName).
		WithIncludedNamespaceScopedResource("deployments").
		WithIncludedNamespaceScopedResource("services").
		WithIncludedNamespaceScopedResource("routes").
		WithExcludedClusterScopedResource("persistentVolumes")

	By("Define workload restore")

	ibuWorkloadRestore = velero.NewRestoreBuilder(APIClient, tsparams.LCAWorkloadName,
		tsparams.LCAOADPNamespace, tsparams.LCAWorkloadName).WithStorageLocation("default")
}

func defineKlusterletBackupRestore() {
	By("Check if klusterlet namespace exists")

	if _, err := namespace.Pull(APIClient, tsparams.LCAKlusterletNamespace); err == nil {
		By("Define klusterlet backup")

		ibuKlusterLetBackup = velero.NewBackupBuilder(
			APIClient, ibuKlusterletName, tsparams.LCAOADPNamespace).WithStorageLocation("default").
			WithIncludedNamespace(tsparams.LCAKlusterletNamespace).
			WithIncludedClusterScopedResource("klusterlets.operator.open-cluster-management.io").
			WithIncludedClusterScopedResource("clusterclaims.cluster.open-cluster-management.io").
			WithIncludedClusterScopedResource("clusterroles.rbac.authorization.k8s.io").
			WithIncludedClusterScopedResource("clusterrolebindings.rbac.authorization.k8s.io").
			WithIncludedNamespaceScopedResource("deployments").
			WithIncludedNamespaceScopedResource("serviceaccounts").
			WithIncludedNamespaceScopedResource("secrets")

		By("Define klusterlet restore")

		ibuKlusterLetRestore = velero.NewRestoreBuilder(
			APIClient, ibuKlusterletName, tsparams.LCAOADPNamespace, ibuKlusterletName).WithStorageLocation("default")
	}
}
