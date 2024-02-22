package rdscorecommon

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

	. "github.com/openshift-kni/eco-gotests/tests/system-tests/rdscore/internal/rdscoreinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/rdscore/internal/rdscoreparams"

	"github.com/openshift-kni/eco-goinfra/pkg/deployment"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

// WaitAllNodesAreReady waits for all the nodes in the cluster to report Ready state.
func WaitAllNodesAreReady(ctx SpecContext) {
	By("Checking all nodes are Ready")

	Eventually(func(ctx SpecContext) bool {
		allNodes, err := nodes.List(APIClient, metav1.ListOptions{})
		if err != nil {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Failed to list all nodes: %s", err)

			return false
		}

		for _, _node := range allNodes {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Processing node %q", _node.Definition.Name)

			for _, condition := range _node.Object.Status.Conditions {
				if condition.Type == rdscoreparams.ConditionTypeReadyString {
					if condition.Status != rdscoreparams.ConstantTrueString {
						glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Node %q is notReady", _node.Definition.Name)
						glog.V(rdscoreparams.RDSCoreLogLevel).Infof("  Reason: %s", condition.Reason)

						return false
					}
				}
			}
		}

		return true
	}).WithTimeout(25*time.Minute).WithPolling(15*time.Second).WithContext(ctx).Should(BeTrue(),
		"Some nodes are notReady")
}

// VerifyUngracefulReboot performs ungraceful reboot of the cluster
//
//nolint:funlen
func VerifyUngracefulReboot(ctx SpecContext) {
	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("\t*** VerifyUngracefulReboot started ***")

	if len(RDSCoreConfig.NodesCredentialsMap) == 0 {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("BMC Details not specified")
		Skip("BMC Details not specified. Skipping...")
	}

	clientOpts := []bmclib.Option{}

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof(
		fmt.Sprintf("BMC options %v", clientOpts))

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof(
		fmt.Sprintf("NodesCredentialsMap:\n\t%#v", RDSCoreConfig.NodesCredentialsMap))

	var bmcMap = make(map[string]*bmclib.Client)

	for node, auth := range RDSCoreConfig.NodesCredentialsMap {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof(
			fmt.Sprintf("Creating BMC client for node %s", node))
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof(
			fmt.Sprintf("BMC Auth %#v", auth))

		bmcClient := bmclib.NewClient(auth.BMCAddress, auth.Username, auth.Password, clientOpts...)
		bmcMap[node] = bmcClient
	}

	var waitGroup sync.WaitGroup

	for node, client := range bmcMap {
		waitGroup.Add(1)

		go func(wg *sync.WaitGroup, nodeName string, client *bmclib.Client) {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof(
				fmt.Sprintf("Starting go routine for %s", nodeName))

			defer GinkgoRecover()
			defer wg.Done()

			glog.V(rdscoreparams.RDSCoreLogLevel).Infof(
				fmt.Sprintf("[%s] Setting timeout for context", nodeName))

			bmcCtx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)

			defer cancel()

			glog.V(rdscoreparams.RDSCoreLogLevel).Infof(
				fmt.Sprintf("[%s] Starting BMC session", nodeName))

			err := client.Open(bmcCtx)

			Expect(err).ToNot(HaveOccurred(),
				fmt.Sprintf("Failed to login to %s", nodeName))

			defer client.Close(bmcCtx)

			By(fmt.Sprintf("Querying power state on %s", nodeName))

			glog.V(rdscoreparams.RDSCoreLogLevel).Infof(
				fmt.Sprintf("Checking power state on %s", nodeName))

			state, err := client.GetPowerState(bmcCtx)
			msgRegex := `(?i)chassis power is on|(?i)^on$`

			glog.V(rdscoreparams.RDSCoreLogLevel).Infof(
				fmt.Sprintf("Power state on %s -> %s", nodeName, state))

			Expect(err).ToNot(HaveOccurred(),
				fmt.Sprintf("Failed to login to %s", nodeName))
			Expect(strings.TrimSpace(state)).To(MatchRegexp(msgRegex),
				fmt.Sprintf("Unexpected power state %s", state))

			err = wait.PollUntilContextTimeout(ctx, 5*time.Second, 5*time.Minute, true,
				func(ctx context.Context) (bool, error) {
					if _, err := client.SetPowerState(bmcCtx, "cycle"); err != nil {
						glog.V(rdscoreparams.RDSCoreLogLevel).Infof(
							fmt.Sprintf("Failed to power cycle %s -> %v", nodeName, err))

						return false, err
					}

					glog.V(rdscoreparams.RDSCoreLogLevel).Infof(
						fmt.Sprintf("Successfully powered cycle %s", nodeName))

					return true, nil
				})

			Expect(err).ToNot(HaveOccurred(),
				fmt.Sprintf("Failed to reboot node %s", nodeName))
		}(&waitGroup, node, client)
	}

	By("Wait for all reboots to finish")

	waitGroup.Wait()
	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Finished waiting for go routines to finish")
	time.Sleep(1 * time.Minute)

	WaitAllNodesAreReady(ctx)
}

// WaitAllDeploymentsAreAvailable wait for all deployments in all namespaces to be Available.
func WaitAllDeploymentsAreAvailable(ctx SpecContext) {
	By("Checking all deployments")

	Eventually(func() bool {
		allDeployments, err := deployment.ListInAllNamespaces(APIClient, metav1.ListOptions{})

		if err != nil {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Failed to list all deployments: %s", err)

			return false
		}

		glog.V(rdscoreparams.RDSCoreLogLevel).Infof(
			fmt.Sprintf("Found %d deployments", len(allDeployments)))

		var nonAvailableDeployments []*deployment.Builder

		for _, deploy := range allDeployments {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof(
				"Processing deployment %q in %q namespace", deploy.Definition.Name, deploy.Definition.Namespace)

			for _, condition := range deploy.Object.Status.Conditions {
				if condition.Type == "Available" {
					if condition.Status != rdscoreparams.ConstantTrueString {
						glog.V(rdscoreparams.RDSCoreLogLevel).Infof(
							"Deployment %q in %q namespace is NotAvailable", deploy.Definition.Name, deploy.Definition.Namespace)
						glog.V(rdscoreparams.RDSCoreLogLevel).Infof("\tReason: %s", condition.Reason)
						glog.V(rdscoreparams.RDSCoreLogLevel).Infof("\tMessage: %s", condition.Message)
						nonAvailableDeployments = append(nonAvailableDeployments, deploy)
					}
				}
			}
		}

		return len(nonAvailableDeployments) == 0
	}).WithTimeout(25*time.Minute).WithPolling(15*time.Second).WithContext(ctx).Should(BeTrue(),
		"There are non-available deployments") // end Eventually
}
