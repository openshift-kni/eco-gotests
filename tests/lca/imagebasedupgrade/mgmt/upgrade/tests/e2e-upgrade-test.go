package upgrade_test

import (
	"fmt"
	"regexp"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/utils/strings/slices"

	"github.com/openshift-kni/eco-goinfra/pkg/clusteroperator"
	"github.com/openshift-kni/eco-goinfra/pkg/clusterversion"
	"github.com/openshift-kni/eco-goinfra/pkg/configmap"
	"github.com/openshift-kni/eco-goinfra/pkg/lca"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/olm"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/internal/ibuparams"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/internal/nodestate"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/internal/safeapirequest"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/mgmt/internal/configmapgenerator"
	. "github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/mgmt/internal/mgmtinittools"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/mgmt/upgrade/internal/tsparams"
	"github.com/openshift-kni/lifecycle-agent/api/v1alpha1"
	oplmV1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
)

var (
	ibu *lca.ImageBasedUpgradeBuilder
	err error
)

var _ = Describe(
	"Performing image based upgrade",
	Ordered,
	Label(tsparams.LabelEndToEndUpgrade), func() {

		BeforeAll(func() {
			By("Pull the imagebasedupgrade from the cluster")
			ibu, err = lca.PullImageBasedUpgrade(APIClient, ibuparams.IBUName)
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

					configmapData, err := configmapgenerator.DataFromDefinition(APIClient,
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
		})

		AfterAll(func() {
			if !MGMTConfig.IdlePostUpgrade {
				By("Revert IBU resource back to Idle stage")
				ibu, err = lca.PullImageBasedUpgrade(APIClient, ibuparams.IBUName)
				Expect(err).NotTo(HaveOccurred(), "error pulling imagebasedupgrade resource")

				if ibu.Object.Spec.Stage == "Upgrade" {
					By("Set IBU stage to Rollback")
					_, err = ibu.WithStage("Rollback").Update()
					Expect(err).NotTo(HaveOccurred(), "error setting ibu to rollback stage")

					By("Wait for IBU resource to be available")
					ibu, err = nodestate.WaitForIBUToBeAvailable(APIClient, time.Minute*10)
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
			}
		})

		It("upgrades the cluster", func() {

			By("Updating the seed image reference")
			ibu, err = ibu.WithSeedImage(MGMTConfig.SeedImage).WithSeedImageVersion(MGMTConfig.SeedImageVersion).Update()
			Expect(err).NotTo(HaveOccurred(), "error updating ibu with image and version")

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
				unreachable, err := nodestate.WaitForNodeToBeUnreachable(node.Object.Name, "6443", time.Minute*5)

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
			ibu, err = nodestate.WaitForIBUToBeAvailable(APIClient, time.Minute*10)
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

			if MGMTConfig.IdlePostUpgrade {
				By("Set the IBU stage to Idle")
				_, err = ibu.WithStage("Idle").Update()
				Expect(err).NotTo(HaveOccurred(), "error setting ibu to idle stage")
			}
		})
	})
