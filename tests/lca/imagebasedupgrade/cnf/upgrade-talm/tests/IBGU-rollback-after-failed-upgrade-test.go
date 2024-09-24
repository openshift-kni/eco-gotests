package upgrade_test

import (
	"strings"
	"time"

	"k8s.io/utils/ptr"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/cgu"
	"github.com/openshift-kni/eco-goinfra/pkg/ibgu"
	"github.com/openshift-kni/eco-goinfra/pkg/lca"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"github.com/openshift-kni/eco-gotests/tests/internal/cluster"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/cnf/internal/cnfclusterinfo"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/cnf/internal/cnfhelper"
	. "github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/cnf/internal/cnfinittools"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/cnf/upgrade-talm/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/internal/nodestate"
	lcav1 "github.com/openshift-kni/lifecycle-agent/api/imagebasedupgrade/v1"
)

var (
// ibu              *lca.ImageBasedUpgradeBuilder
// seedImageVersion string
// err              error
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
				Expect(err).ToNot(HaveOccurred(), "Failed to delete IBGU cgu on target hub cluster")

				// Sleep for 10 seconds to allow talm to reconcile state.
				// Sometimes if the next test re-creates the IBGUs too quickly,
				// the policies compliance status is not updated correctly.
				time.Sleep(10 * time.Second)
			})

			By("Creating, enabling ibu finalize CGU and waiting for CGU status to report completed", func() {
				finalizeCguBuilder := cgu.NewCguBuilder(TargetHubAPIClient,
					tsparams.FinalizeCguName, tsparams.IbuCguNamespace, 1).
					WithCluster(tsparams.TargetSnoClusterName).
					WithManagedPolicy(tsparams.FinalizePolicyName).
					WithCanary(tsparams.TargetSnoClusterName)
				finalizeCguBuilder.Definition.Spec.Enable = ptr.To(true)

				finalizeCguBuilder, err := finalizeCguBuilder.Create()
				Expect(err).ToNot(HaveOccurred(), "Failed to create finalize CGU.")

				_, err = ibu.WaitUntilStageComplete("Idle")
				Expect(err).NotTo(HaveOccurred(), "error waiting for idle stage to complete")

				_, err = finalizeCguBuilder.WaitUntilComplete(5 * time.Minute)
				Expect(err).ToNot(HaveOccurred(), "Finalize CGU did not complete in time.")
			})

			By("Deleting finalize cgu created on target hub cluster", func() {
				err := cnfhelper.DeleteIbuTestCguOnTargetHub(TargetHubAPIClient, tsparams.FinalizeCguName,
					tsparams.IbuCguNamespace)
				Expect(err).ToNot(HaveOccurred(), "Failed to delete finalize cgu on target hub cluster")
			})

			// Sleep for 10 seconds to allow talm to reconcile state.
			// Sometimes if the next test re-creates the CGUs too quickly,
			// the policies compliance status is not updated correctly.
			time.Sleep(10 * time.Second)
		})

		It("Rollback after a failed upgrade", reportxml.ID("69054"), func() {

			By("Creating Prep->Upgrade->FinalizeUpgrae IBGU and waiting for node rebooted into stateroot B", func() {

				newIbguBuilder = ibgu.NewIbguBuilder(TargetHubAPIClient,
					tsparams.IbguName, tsparams.IbuCguNamespace).
					WithClusterLabelSelectors(tsparams.ClusterLabelSelector).
					WithOadpContent("oadp-cm", "ztp-group").
					WithSeedImageRef("registry.kni-qe-18.lab.eng.tlv2.redhat.com:5000/ibu/seed:4.17.0-rc.1", "4.17.0-rc.1").
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
