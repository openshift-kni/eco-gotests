package rdscorecommon

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/reboot"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/remote"

	. "github.com/openshift-kni/eco-gotests/tests/system-tests/rdscore/internal/rdscoreinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/rdscore/internal/rdscoreparams"
)

func mountNamespaceEncapsulation(nodeLabel string) {
	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Check propagation of directory containing namespace mount pin")

	var (
		nodeList []*nodes.Builder
		err      error
		ctx      SpecContext
	)

	By("Retrieve nodes list")

	Eventually(func() bool {
		nodeList, err = nodes.List(
			APIClient,
			metav1.ListOptions{LabelSelector: nodeLabel},
		)

		if err != nil {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Failed to list nodes: %w", err)

			return false
		}

		return len(nodeList) > 0
	}).WithContext(ctx).WithTimeout(1*time.Minute).WithPolling(5*time.Second).Should(BeTrue(),
		fmt.Sprintf("Failed to find pods matching label: %q", nodeLabel))

	for _, node := range nodeList {
		By("Trigger kernel crash")
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Trigerring kernel crash on %q",
			node.Definition.Name)

		err = reboot.KernelCrashKdump(node.Definition.Name)
		Expect(err).ToNot(HaveOccurred(), "Error triggering a kernel crash on the node.")

		By("Waiting for node to go into NotReady state")

		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Checking node %q got into Ready state",
			node.Definition.Name)

		Eventually(func() bool {
			currentNode, err := nodes.Pull(APIClient, node.Definition.Name)
			if err != nil {
				glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Failed to pull in node %q due to %v",
					node.Definition.Name, err)

				return false
			}

			for _, condition := range currentNode.Object.Status.Conditions {
				if condition.Type == rdscoreparams.ConditionTypeReadyString {
					if condition.Status == rdscoreparams.ConstantTrueString {
						glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Node %q is Ready", currentNode.Definition.Name)
						glog.V(rdscoreparams.RDSCoreLogLevel).Infof("  Reason: %s", condition.Reason)

						return true
					}
				}
			}

			return false
		}).WithTimeout(5*time.Minute).WithPolling(15*time.Second).WithContext(ctx).Should(BeTrue(),
			"Node hasn't reached Ready state")

		By("Assert vmcore dump was generated")

		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Checking if vmcore dump was generated")

		cmdToExec := []string{"chroot", "/rootfs", "ls", "/var/crash"}

		Eventually(func() bool {
			coreDumps, err := remote.ExecuteOnNodeWithDebugPod(cmdToExec, node.Definition.Name)

			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Executing command: %q", strings.Join(cmdToExec, " "))

			if err != nil {
				glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Failed to execute command: %v", err)

				return false
			}

			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("\tGenerated VMCore dumps: %v", coreDumps)

			return len(strings.Fields(coreDumps)) >= 1
		}).WithContext(ctx).WithTimeout(1*time.Minute).WithPolling(5*time.Second).Should(BeTrue(),
			"error: vmcore dump was not generated")
	}
}

// VerifyMountNamespaceOnControlPlane check mount namespace on Control Plane nodes.
func VerifyMountNamespaceOnControlPlane(ctx SpecContext) {
	mountNamespaceEncapsulation(RDSCoreConfig.KDumpCPNodeLabel)
}

// VerifyMountNamespaceOnWorkerMCP check mount namespace service on nodes in "Worker" MCP.
func VerifyMountNamespaceOnWorkerMCP(ctx SpecContext) {
	mountNamespaceEncapsulation(RDSCoreConfig.KDumpWorkerMCPNodeLabel)
}

// VerifyMountNamespaceOnCNFMCP check mount namespace service on nodes in "CNF" MCP.
func VerifyMountNamespaceOnCNFMCP(ctx SpecContext) {
	mountNamespaceEncapsulation(RDSCoreConfig.KDumpCNFMCPNodeLabel)
}
