package rdscorecommon

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/remote"

	. "github.com/openshift-kni/eco-gotests/tests/system-tests/rdscore/internal/rdscoreinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/rdscore/internal/rdscoreparams"
)

func mountNamespaceEncapsulation(nodeLabel string) {
	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Check encapsulation for nodes %s", nodeLabel)

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
		By("Open debug pod and check the systemd, kubelet and CRI-O mount namespaces")

		nodeName := node.Definition.Name

		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Check the systemd mount namespace on node %s", nodeName)

		systemdMountNsCmd := []string{"chroot", "/rootfs", "/bin/sh", "-c", "readlink /proc/1/ns/mnt"}

		systemdMountNsOutput, err := remote.ExecuteOnNodeWithDebugPod(systemdMountNsCmd, nodeName)
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to execute %s cmd on the node %s due to %v",
			systemdMountNsCmd, nodeName, err))

		systemdMountNs := strings.Split(systemdMountNsOutput, ":")[1]

		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Check the kubelet mount namespace on node %s", nodeName)

		kubeletMountNsCmd := []string{"chroot", "/rootfs", "/bin/sh", "-c", "readlink /proc/$(pgrep kubelet)/ns/mnt"}

		kubeletMountNsOutput, err := remote.ExecuteOnNodeWithDebugPod(kubeletMountNsCmd, nodeName)
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to execute %s cmd on the node %s due to %v",
			kubeletMountNsCmd, nodeName, err))

		kubeletMountNs := strings.Split(kubeletMountNsOutput, ":")[1]

		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Check the CRI-O mount namespace on node %s", nodeName)

		crioMountNsCmd := []string{"chroot", "/rootfs", "/bin/sh", "-c", "readlink /proc/$(pgrep crio)/ns/mnt"}

		crioMountNsOutput, err := remote.ExecuteOnNodeWithDebugPod(crioMountNsCmd, nodeName)
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to execute %s cmd on the node %s due to %v",
			crioMountNsCmd, nodeName, err))

		crioMountNs := strings.Split(crioMountNsOutput, ":")[1]

		By("Check that encapsulation is in effect")

		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Checking if kubelet and CRI-O are in the same mount namespace")

		Expect(kubeletMountNs == crioMountNs).To(Equal(true),
			fmt.Sprintf("General mount namespace failure; kubelet and CRI-O have to be in the same mount namespace;"+
				"kubelet mount namespace: %s; CRI-O mount namespace: %s", kubeletMountNs, crioMountNs))

		Expect(systemdMountNs != crioMountNs).To(Equal(true),
			fmt.Sprintf("Encapsulation is not in effect; systemd have to be in a different mount namespace to "+
				"kubelet and CRI-O; systemd mount namespace: %s; kubelet mount namespace: %s; CRI-O mount namespace: %s",
				systemdMountNs, kubeletMountNs, crioMountNs))

		By("Inspecting encapsulated namespaces")
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Check the systemd mount namespace on node %s", nodeName)

		inspectingCmd := []string{"chroot", "/rootfs", "/bin/sh", "-c", "findmnt -n -oPROPAGATION /run/kubens"}

		out, err := remote.ExecuteOnNodeWithDebugPod(inspectingCmd, nodeName)
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to execute %s cmd on the node %s due to %v",
			inspectingCmd, nodeName, err))
		Expect(strings.Contains(out, "private,slave")).To(BeTrue(),
			fmt.Sprintf("propagation of the directory does not contain a namespace mount pin for the node %s; %s",
				nodeName, out))
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
