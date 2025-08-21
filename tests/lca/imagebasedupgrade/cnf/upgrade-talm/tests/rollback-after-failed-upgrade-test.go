package upgrade_test

import (
	"strings"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	lcav1 "github.com/openshift-kni/lifecycle-agent/api/imagebasedupgrade/v1"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/ibgu"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/lca"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/nodes"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/reportxml"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/internal/cluster"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/lca/imagebasedupgrade/cnf/internal/cnfclusterinfo"
	. "github.com/rh-ecosystem-edge/eco-gotests/tests/lca/imagebasedupgrade/cnf/internal/cnfinittools"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/lca/imagebasedupgrade/cnf/upgrade-talm/internal/tsparams"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/lca/imagebasedupgrade/internal/nodestate"
)

var (
	ibu              *lca.ImageBasedUpgradeBuilder
	seedImageVersion string
	err              error
)

var _ = Describe(
	"Validating rollback stage after a failed upgrade",
	Label(tsparams.LabelRollbackFlow), func() {
		var newIbguBuilder *ibgu.IbguBuilder

		BeforeEach(func() {
			By("Saving target sno cluster info before the test", func() {
				err := cnfclusterinfo.PreUpgradeClusterInfo.SaveClusterInfo()
				Expect(err).NotTo(HaveOccurred(), "Failed to collect and save target sno cluster info before the test")
			})

			By("Fetching target sno cluster name", func() {
				err = cnfclusterinfo.PreUpgradeClusterInfo.SaveClusterInfo()
				Expect(err).NotTo(HaveOccurred(), "Failed to extract target sno cluster name")

				tsparams.TargetSnoClusterName = cnfclusterinfo.PreUpgradeClusterInfo.Name
			})

			By("Retrieve seed image version and updating LCA init-monitor watchdog timer ", func() {
				ibu, err = lca.PullImageBasedUpgrade(TargetSNOAPIClient)
				Expect(err).NotTo(HaveOccurred(), "error pulling ibu resource from cluster")

				seedImageVersion = ibu.Definition.Spec.SeedImageRef.Version

				By("Setting LCA init-monitor watchdog timer to 5 minutes")
				ibu.Definition.Spec.AutoRollbackOnFailure = &lcav1.AutoRollbackOnFailure{}
				ibu.Definition.Spec.AutoRollbackOnFailure.InitMonitorTimeoutSeconds = 300
				ibu, err = ibu.Update()
				Expect(err).NotTo(HaveOccurred(), "error updating ibu resource with custom lca init-monitor timeout value")
			})
		})

		AfterEach(func() {
			By("Deleting IBGU on target hub cluster", func() {
				_, err := newIbguBuilder.DeleteAndWait(1 * time.Minute)
				Expect(err).ToNot(HaveOccurred(), "Failed to delete IBGU on target hub cluster")

				// Sleep for 10 seconds to allow talm to reconcile state.
				// Sometimes if the next test re-creates the IBGUs too quickly,
				// the policies compliance status is not updated correctly.
				time.Sleep(10 * time.Second)
			})

			By("Creating, enabling ibu finalize", func() {

				_, err = ibu.WaitUntilStageComplete("Idle")
				Expect(err).NotTo(HaveOccurred(), "error waiting for idle stage to complete")

			})

			// Sleep for 10 seconds to allow talm to reconcile state.
			// Sometimes if the next test re-creates the CGUs too quickly,
			// the policies compliance status is not updated correctly.
			time.Sleep(10 * time.Second)
		})

		It("Rollback after a failed upgrade", reportxml.ID("69054"), func() {

			By("Creating Prep->Upgrade->FinalizeUpgrae IBGU and waiting for node rebooted into stateroot B", func() {

				newIbguBuilder = ibgu.NewIbguBuilder(TargetHubAPIClient,
					tsparams.IbguName, tsparams.IbguNamespace).
					WithClusterLabelSelectors(tsparams.ClusterLabelSelector).
					WithOadpContent(CNFConfig.IbguOadpCmName, CNFConfig.IbguOadpCmNamespace).
					WithSeedImageRef(CNFConfig.IbguSeedImage, CNFConfig.IbguSeedImageVersion).
					WithPlan([]string{"Prep"}, 20, 20).
					WithPlan([]string{"Upgrade"}, 20, 20).
					WithPlan([]string{"FinalizeUpgrade"}, 20, 20)

				newIbguBuilder, err = newIbguBuilder.Create()
				Expect(err).ToNot(HaveOccurred(), "Failed to create IBGU")

				By("Get list of node to be upgraded")

				ibuNode, err := nodes.List(TargetSNOAPIClient)
				Expect(err).NotTo(HaveOccurred(), "error listing node")

				By("Wait for node to become unreachable")

				for _, node := range ibuNode {
					unreachable, err := nodestate.WaitForNodeToBeUnreachable(node.Object.Name, "6443", time.Minute*10)

					Expect(err).To(BeNil(), "error waiting for %s node to shutdown", node.Object.Name)
					Expect(unreachable).To(BeTrue(), "error: node %s is still reachable", node.Object.Name)
				}

				By("Wait for node to become reachable")

				for _, node := range ibuNode {
					reachable, err := nodestate.WaitForNodeToBeReachable(node.Object.Name, "6443", time.Minute*30)

					Expect(err).To(BeNil(), "error waiting for %s node to become reachable", node.Object.Name)
					Expect(reachable).To(BeTrue(), "error: node %s is still unreachable", node.Object.Name)
				}

				By("Wait for IBU resource to be available")

				err = nodestate.WaitForIBUToBeAvailable(TargetSNOAPIClient, ibu, time.Minute*15)
				Expect(err).NotTo(HaveOccurred(), "error waiting for ibu resource to become available")
			})

			By("Verifying booted stateroot name on target sno cluster node", func() {
				var seedVersionFound bool

				getDeploymentIndexCmd := "rpm-ostree status --json | jq '.deployments[0].osname'"
				getDesiredStaterootName, err := cluster.ExecCmdWithStdout(TargetSNOAPIClient, getDeploymentIndexCmd)
				Expect(err).NotTo(HaveOccurred(), "could not execute command: %s", err)

				for _, stdout := range getDesiredStaterootName {
					bootedStaterootNameRes := strings.ReplaceAll(stdout, "_", "-")
					if bootedStaterootNameRes != "" {
						if strings.Contains(bootedStaterootNameRes, seedImageVersion) {
							glog.V(100).Infof("Found "+seedImageVersion+" in %s", bootedStaterootNameRes)

							seedVersionFound = true
						}
					}
				}

				Expect(seedVersionFound).To(BeTrue(), "Target cluster node booted into stateroot B")
			})

			By("Simulate a fault to make upgrade fail", func() {
				faultInjectCmd := "echo a > /etc/mco/proxy.env"
				faultInjectCmdRes, err := cluster.ExecCmdWithStdout(TargetSNOAPIClient, faultInjectCmd)
				Expect(err).NotTo(HaveOccurred(), "could not execute command: %s", faultInjectCmdRes)
			})

			By("Verifying auto rollback triggered upon upgrade failure", func() {

				By("Waiting for node rebooted into stateroot A and cluster become available", func() {

					By("Get list of node to be upgraded")

					ibuNode, err := nodes.List(TargetSNOAPIClient)
					Expect(err).NotTo(HaveOccurred(), "error listing node")

					By("Wait for node to become unreachable")

					for _, node := range ibuNode {
						unreachable, err := nodestate.WaitForNodeToBeUnreachable(node.Object.Name, "6443", time.Minute*10)

						Expect(err).To(BeNil(), "error waiting for %s node to shutdown", node.Object.Name)
						Expect(unreachable).To(BeTrue(), "error: node %s is still reachable", node.Object.Name)
					}

					By("Wait for node to become reachable")

					for _, node := range ibuNode {
						reachable, err := nodestate.WaitForNodeToBeReachable(node.Object.Name, "6443", time.Minute*30)

						Expect(err).To(BeNil(), "error waiting for %s node to become reachable", node.Object.Name)
						Expect(reachable).To(BeTrue(), "error: node %s is still unreachable", node.Object.Name)
					}

					By("Wait for IBU resource to be available")

					err = nodestate.WaitForIBUToBeAvailable(TargetSNOAPIClient, ibu, time.Minute*15)
					Expect(err).NotTo(HaveOccurred(), "error waiting for ibu resource to become available")
				})
			})

			By("Saving target sno cluster info after the test", func() {
				err := cnfclusterinfo.PostUpgradeClusterInfo.SaveClusterInfo()
				Expect(err).NotTo(HaveOccurred(), "Failed to collect and save target sno cluster info after the test")
			})

			By("Validating target sno cluster version after auto rollback", func() {
				Expect(cnfclusterinfo.PreUpgradeClusterInfo.Version).
					To(Equal(cnfclusterinfo.PostUpgradeClusterInfo.Version),
						"Target sno cluster reports old cluster version")
			})
		})
	})
