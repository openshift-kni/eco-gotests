package tests

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/containernshide/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/cluster"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/raninittools"
)

var _ = Describe("Container Namespace Hiding", Label(tsparams.LabelContainerNSHideTestCases), func() {
	It("should not have kubelet and crio using the same inode as systemd", reportxml.ID("53681"), func() {
		By("Getting systemd inodes on cluster nodes")
		systemdInodes, err := cluster.ExecCmdWithStdout(raninittools.Spoke1APIClient, 3, "readlink /proc/1/ns/mnt")
		Expect(err).ToNot(HaveOccurred(), "Failed to check systemd inodes")

		By("Getting kubelet inodes on cluster nodes")
		kubeletInodes, err := cluster.ExecCmdWithStdout(
			raninittools.Spoke1APIClient, 3, "readlink /proc/$(pidof kubelet)/ns/mnt")
		Expect(err).ToNot(HaveOccurred(), "Failed to check kubelet inodes")

		By("Getting crio inodes on cluster nodes")
		crioInodes, err := cluster.ExecCmdWithStdout(raninittools.Spoke1APIClient, 3, "readlink /proc/$(pidof crio)/ns/mnt")
		Expect(err).ToNot(HaveOccurred(), "Failed to check crio inodes")

		Expect(len(systemdInodes)).To(Equal(len(kubeletInodes)),
			"Collected systemd inodes from different number of nodes than kubelet inodes")
		Expect(len(systemdInodes)).To(Equal(len(crioInodes)),
			"Collected systemd inodes from different number of nodes than crio inodes")

		for host, systemdInode := range systemdInodes {
			By(fmt.Sprintf("Checking inodes on host %s", host))
			kubeletInode, ok := kubeletInodes[host]
			Expect(ok).To(BeTrue(), "Found systemd inode but not kubelet inode on node %s", host)

			crioInode, ok := crioInodes[host]
			Expect(ok).To(BeTrue(), "Found systemd inode but not crio inode on node %s", host)

			glog.V(tsparams.LogLevel).Infof("systemd inode: %s", systemdInode)
			glog.V(tsparams.LogLevel).Infof("kubelet inode: %s", kubeletInode)
			glog.V(tsparams.LogLevel).Infof("crio inode: %s", crioInode)
			Expect(kubeletInode).To(Equal(crioInode), "kubelet and crio inodes do not match on node %s", host)
			Expect(systemdInode).NotTo(Equal(kubeletInode),
				"systemd and kubelet inodes match - namespace is not hidden on node %s", host)
			Expect(systemdInode).NotTo(Equal(crioInode),
				"systemd and crio inodes match - namespace is not hidden on node %s", host)
		}
	})
})
