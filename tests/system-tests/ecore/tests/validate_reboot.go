package ecore_system_test

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	bmclib "github.com/bmc-toolbox/bmclib/v2"
	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/clusteroperator"
	"github.com/openshift-kni/eco-goinfra/pkg/deployment"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	. "github.com/openshift-kni/eco-gotests/tests/system-tests/ecore/internal/ecoreinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/ecore/internal/ecoreparams"
)

var _ = Describe(
	"ECore Reboots",
	Ordered,
	ContinueOnFailure,
	Label(ecoreparams.LabelEcoreValidateReboots), func() {
		It("Verifies Hard Reboot", Label("validate_ungraceful_reboot"), func(ctx SpecContext) {
			if len(ECoreConfig.NodesCredentialsMap) == 0 {
				glog.V(ecoreparams.ECoreLogLevel).Infof("BMC Details not specified")
				Skip("BMC Details not specified. Skipping...")
			}

			clientOpts := []bmclib.Option{}

			glog.V(ecoreparams.ECoreLogLevel).Infof(
				fmt.Sprintf("BMC options %v", clientOpts))

			glog.V(ecoreparams.ECoreLogLevel).Infof(
				fmt.Sprintf("NodesCredentialsMap:\n\t%#v", ECoreConfig.NodesCredentialsMap))

			var bmcMap = make(map[string]*bmclib.Client)

			for node, auth := range ECoreConfig.NodesCredentialsMap {
				glog.V(ecoreparams.ECoreLogLevel).Infof(
					fmt.Sprintf("Creating BMC client for node %s", node))
				glog.V(ecoreparams.ECoreLogLevel).Infof(
					fmt.Sprintf("BMC Auth %#v", auth))

				bmcClient := bmclib.NewClient(auth.BMCAddress, auth.Username, auth.Password, clientOpts...)
				bmcMap[node] = bmcClient

			}

			var waitGroup sync.WaitGroup

			for node, client := range bmcMap {
				waitGroup.Add(1)
				go func(wg *sync.WaitGroup, nodeName string, client *bmclib.Client) {
					glog.V(ecoreparams.ECoreLogLevel).Infof(
						fmt.Sprintf("Starting go routine for %s", nodeName))

					defer GinkgoRecover()
					defer wg.Done()

					glog.V(ecoreparams.ECoreLogLevel).Infof(
						fmt.Sprintf("[%s] Setting timeout for context", nodeName))
					bmcCtx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
					defer cancel()

					glog.V(ecoreparams.ECoreLogLevel).Infof(
						fmt.Sprintf("[%s] Starting BMC session", nodeName))
					err := client.Open(bmcCtx)

					Expect(err).ToNot(HaveOccurred(),
						fmt.Sprintf("Failed to login to %s", nodeName))

					defer client.Close(bmcCtx)

					By(fmt.Sprintf("Querying power state on %s", nodeName))
					glog.V(ecoreparams.ECoreLogLevel).Infof(
						fmt.Sprintf("Checking power state on %s", nodeName))

					state, err := client.GetPowerState(bmcCtx)
					msgRegex := `(?i)chassis power is on|(?i)^on$`

					glog.V(ecoreparams.ECoreLogLevel).Infof(
						fmt.Sprintf("Power state on %s -> %s", nodeName, state))

					Expect(err).ToNot(HaveOccurred(),
						fmt.Sprintf("Failed to login to %s", nodeName))
					Expect(strings.TrimSpace(state)).To(MatchRegexp(msgRegex),
						fmt.Sprintf("Unexpected power state %s", state))

					err = wait.PollUntilContextTimeout(ctx, 5*time.Second, 5*time.Minute, true,
						func(ctx context.Context) (bool, error) {
							if _, err := client.SetPowerState(bmcCtx, "cycle"); err != nil {
								glog.V(ecoreparams.ECoreLogLevel).Infof(
									fmt.Sprintf("Failed to power cycle %s -> %v", nodeName, err))

								return false, err
							}

							glog.V(ecoreparams.ECoreLogLevel).Infof(
								fmt.Sprintf("Successfully powered cycle %s", nodeName))

							return true, nil
						})

					Expect(err).ToNot(HaveOccurred(),
						fmt.Sprintf("Failed to reboot node %s", nodeName))

				}(&waitGroup, node, client)
			}

			By("Wait for all reboots to finish")
			waitGroup.Wait()
			glog.V(ecoreparams.ECoreLogLevel).Infof("Finished waiting for go routines to finish")
			time.Sleep(1 * time.Minute)

			By("Checking all nodes are Ready")
			Eventually(func(ctx SpecContext) bool {
				allNodes, err := nodes.List(APIClient, metav1.ListOptions{})
				if err != nil {
					glog.V(ecoreparams.ECoreLogLevel).Infof("Failed to list all nodes: %s", err)

					return false
				}

				for _, _node := range allNodes {
					glog.V(ecoreparams.ECoreLogLevel).Infof("Processing node %q", _node.Definition.Name)

					for _, condition := range _node.Object.Status.Conditions {
						if condition.Type == ecoreparams.ConditionTypeReadyString {
							if condition.Status != ecoreparams.ConstantTrueString {
								glog.V(ecoreparams.ECoreLogLevel).Infof("Node %q is notReady", _node.Definition.Name)
								glog.V(ecoreparams.ECoreLogLevel).Infof("  Reason: %s", condition.Reason)

								return false
							}
						}
					}
				}

				return true
			}).WithTimeout(25*time.Minute).WithPolling(15*time.Second).WithContext(ctx).Should(BeTrue(),
				"Some nodes are notReady")

			By("Checking all deployments")
			Eventually(func() bool {
				allDeployments, err := deployment.ListInAllNamespaces(APIClient, metav1.ListOptions{})

				if err != nil {
					glog.V(ecoreparams.ECoreLogLevel).Infof("Failed to list all deployments: %s", err)

					return false
				}

				glog.V(ecoreparams.ECoreLogLevel).Infof(
					fmt.Sprintf("Found %d deployments", len(allDeployments)))

				var nonAvailableDeployments []*deployment.Builder

				for _, deploy := range allDeployments {
					glog.V(ecoreparams.ECoreLogLevel).Infof(
						"Processing deployment %q in %q namespace", deploy.Definition.Name, deploy.Definition.Namespace)

					for _, condition := range deploy.Object.Status.Conditions {
						if condition.Type == "Available" {
							if condition.Status != ecoreparams.ConstantTrueString {
								glog.V(ecoreparams.ECoreLogLevel).Infof(
									"Deployment %q in %q namespace is NotAvailable", deploy.Definition.Name, deploy.Definition.Namespace)
								glog.V(ecoreparams.ECoreLogLevel).Infof("\tReason: %s", condition.Reason)
								glog.V(ecoreparams.ECoreLogLevel).Infof("\tMessage: %s", condition.Message)
								nonAvailableDeployments = append(nonAvailableDeployments, deploy)
							}
						}
					}
				}

				return len(nonAvailableDeployments) == 0
			}).WithTimeout(25*time.Minute).WithPolling(15*time.Second).WithContext(ctx).Should(BeTrue(),
				"There are non-available deployments") // end Eventually

			By("Checking all cluster operators")
			ok, err := clusteroperator.WaitForAllClusteroperatorsAvailable(
				APIClient, 15*time.Minute, metav1.ListOptions{})

			Expect(err).ToNot(HaveOccurred(), "Failed to get cluster operator status")
			Expect(ok).To(BeTrue(), "Some cluster operators not Available")

		})

		It("Verifies Soft Reboot", Label("validate_graceful_reboot"), func(ctx SpecContext) {
			if len(ECoreConfig.NodesCredentialsMap) == 0 {
				glog.V(ecoreparams.ECoreLogLevel).Infof("BMC Details not specified")
				Skip("BMC Details not specified. Skipping...")
			}

			By("Getting list of all nodes")
			allNodes, err := nodes.List(APIClient, metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred(),
				fmt.Sprintf("Error listing pods in the cluster: %v", err))
			Expect(len(allNodes)).ToNot(Equal(0), "0 nodes found in the cluster")

			for _, _node := range allNodes {
				glog.V(ecoreparams.ECoreLogLevel).Infof("Processing node %q", _node.Definition.Name)

				glog.V(ecoreparams.ECoreLogLevel).Infof("Cordoning node %q", _node.Definition.Name)
				err := _node.Cordon()
				Expect(err).ToNot(HaveOccurred(),
					fmt.Sprintf("Failed to cordon %q due to %v", _node.Definition.Name, err))
				time.Sleep(5 * time.Second)

				glog.V(ecoreparams.ECoreLogLevel).Infof("Draining node %q", _node.Definition.Name)
				err = _node.Drain()
				Expect(err).ToNot(HaveOccurred(),
					fmt.Sprintf("Failed to drain %q due to %v", _node.Definition.Name, err))

				clientOpts := []bmclib.Option{}

				glog.V(ecoreparams.ECoreLogLevel).Infof(
					fmt.Sprintf("BMC options %v", clientOpts))

				glog.V(ecoreparams.ECoreLogLevel).Infof(
					fmt.Sprintf("NodesCredentialsMap:\n\t%#v", ECoreConfig.NodesCredentialsMap))

				var bmcClient *bmclib.Client

				glog.V(ecoreparams.ECoreLogLevel).Infof(
					fmt.Sprintf("Creating BMC client for node %s", _node.Definition.Name))

				if auth, ok := ECoreConfig.NodesCredentialsMap[_node.Definition.Name]; !ok {
					glog.V(ecoreparams.ECoreLogLevel).Infof(
						fmt.Sprintf("BMC Details for %q not found", _node.Definition.Name))
					Fail(fmt.Sprintf("BMC Details for %q not found", _node.Definition.Name))
				} else {
					bmcClient = bmclib.NewClient(auth.BMCAddress, auth.Username, auth.Password, clientOpts...)
				}

				glog.V(ecoreparams.ECoreLogLevel).Infof(
					fmt.Sprintf("[%s] Setting timeout for context", _node.Definition.Name))
				bmcCtx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
				defer cancel()

				glog.V(ecoreparams.ECoreLogLevel).Infof(
					fmt.Sprintf("[%s] Starting BMC session", _node.Definition.Name))

				err = bmcClient.Open(bmcCtx)

				Expect(err).ToNot(HaveOccurred(),
					fmt.Sprintf("Failed to login to %s", _node.Definition.Name))

				defer bmcClient.Close(bmcCtx)

				err = wait.PollUntilContextTimeout(ctx, 5*time.Second, 5*time.Minute, true,
					func(ctx context.Context) (bool, error) {
						if _, err := bmcClient.SetPowerState(bmcCtx, "cycle"); err != nil {
							glog.V(ecoreparams.ECoreLogLevel).Infof(
								fmt.Sprintf("Failed to power cycle %s -> %v", _node.Definition.Name, err))

							return false, err
						}

						glog.V(ecoreparams.ECoreLogLevel).Infof(
							fmt.Sprintf("Successfully powered cycle %s", _node.Definition.Name))

						return true, nil
					})

				Expect(err).ToNot(HaveOccurred(),
					fmt.Sprintf("Failed to reboot node %s", _node.Definition.Name))

				By(fmt.Sprintf("Checking node %q got into NotReady", _node.Definition.Name))

				Eventually(func(ctx SpecContext) bool {
					currentNode, err := nodes.Pull(APIClient, _node.Definition.Name)
					if err != nil {
						glog.V(ecoreparams.ECoreLogLevel).Infof("Failed to pull node: %v", err)

						return false
					}

					for _, condition := range currentNode.Object.Status.Conditions {
						if condition.Type == ecoreparams.ConditionTypeReadyString {
							if condition.Status != ecoreparams.ConstantTrueString {
								glog.V(ecoreparams.ECoreLogLevel).Infof("Node %q is notReady", currentNode.Definition.Name)
								glog.V(ecoreparams.ECoreLogLevel).Infof("  Reason: %s", condition.Reason)

								return true
							}
						}
					}

					return false
				}).WithTimeout(25*time.Minute).WithPolling(15*time.Second).WithContext(ctx).Should(BeTrue(),
					"Node hasn't reached notReady state")

				By(fmt.Sprintf("Checking node %q got into Ready", _node.Definition.Name))

				Eventually(func(ctx SpecContext) bool {
					currentNode, err := nodes.Pull(APIClient, _node.Definition.Name)
					if err != nil {
						glog.V(ecoreparams.ECoreLogLevel).Infof("Error pulling in node: %v", err)

						return false
					}

					for _, condition := range currentNode.Object.Status.Conditions {
						if condition.Type == ecoreparams.ConditionTypeReadyString {
							if condition.Status == ecoreparams.ConstantTrueString {
								glog.V(ecoreparams.ECoreLogLevel).Infof("Node %q is notReady", currentNode.Definition.Name)
								glog.V(ecoreparams.ECoreLogLevel).Infof("  Reason: %s", condition.Reason)

								return true
							}
						}
					}

					return false
				}).WithTimeout(25*time.Minute).WithPolling(15*time.Second).WithContext(ctx).Should(BeTrue(),
					"Node hasn't reached Ready state")

				glog.V(ecoreparams.ECoreLogLevel).Infof("Uncordoning node %q", _node.Definition.Name)
				err = _node.Uncordon()
				Expect(err).ToNot(HaveOccurred(),
					fmt.Sprintf("Failed to uncordon %q due to %v", _node.Definition.Name, err))

				time.Sleep(15 * time.Second)
			}

			By("Checking all deployments are Available")
			Eventually(func() bool {
				allDeployments, err := deployment.ListInAllNamespaces(APIClient, metav1.ListOptions{})

				if err != nil {
					glog.V(ecoreparams.ECoreLogLevel).Infof("Failed to list all deployments: %s", err)

					return false
				}

				glog.V(ecoreparams.ECoreLogLevel).Infof(
					fmt.Sprintf("Found %d deployments", len(allDeployments)))

				var nonAvailableDeployments []*deployment.Builder

				for _, deploy := range allDeployments {
					glog.V(ecoreparams.ECoreLogLevel).Infof(
						"Processing deployment %q in %q namespace", deploy.Definition.Name, deploy.Definition.Namespace)

					for _, condition := range deploy.Object.Status.Conditions {
						if condition.Type == "Available" {
							if condition.Status != ecoreparams.ConstantTrueString {
								glog.V(ecoreparams.ECoreLogLevel).Infof(
									"Deployment %q in %q namespace is NotAvailable", deploy.Definition.Name, deploy.Definition.Namespace)
								glog.V(ecoreparams.ECoreLogLevel).Infof("\tReason: %s", condition.Reason)
								glog.V(ecoreparams.ECoreLogLevel).Infof("\tMessage: %s", condition.Message)
								nonAvailableDeployments = append(nonAvailableDeployments, deploy)
							}
						}
					}
				}

				return len(nonAvailableDeployments) == 0
			}).WithTimeout(25*time.Minute).WithPolling(15*time.Second).WithContext(ctx).Should(BeTrue(),
				"There are non-available deployments") // end Eventually

			By("Checking all cluster operators")
			ok, err := clusteroperator.WaitForAllClusteroperatorsAvailable(
				APIClient, 15*time.Minute, metav1.ListOptions{})

			Expect(err).ToNot(HaveOccurred(), "Failed to get cluster operator status")
			Expect(ok).To(BeTrue(), "Some cluster operators not Available")

		})

	}) //
