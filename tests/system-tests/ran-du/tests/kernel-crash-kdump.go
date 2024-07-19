package ran_du_system_test

import (
	"fmt"
	"strings"
	"time"

	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/remote"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/reboot"
	. "github.com/openshift-kni/eco-gotests/tests/system-tests/ran-du/internal/randuinittools"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe(
	"KernelCrashKdump",
	Ordered,
	ContinueOnFailure,
	Label("KernelCrashKdump"), func() {
		It("Trigger kernel crash to generate kdump vmcore", reportxml.ID("56216"), Label("KernelCrashKdump"), func() {

			By("Retrieve nodes list")
			nodeList, err := nodes.List(
				APIClient,
				metav1.ListOptions{},
			)
			Expect(err).ToNot(HaveOccurred(), "Error listing nodes.")

			for _, node := range nodeList {
				By("Trigger kernel crash")
				err = reboot.KernelCrashKdump(node.Definition.Name)
				Expect(err).ToNot(HaveOccurred(), "Error triggering a kernel crash on the node.")

				By(fmt.Sprintf("Wait for %d minutes for the cluster resources to reconciliate their state",
					RanDuTestConfig.RebootRecoveryTime))
				time.Sleep(time.Duration(RanDuTestConfig.RebootRecoveryTime) * time.Minute)

				By("Assert vmcore dump was generated")
				cmdToExec := []string{"chroot", "/rootfs", "ls", "/var/crash"}

				coreDumps, err := remote.ExecuteOnNodeWithDebugPod(cmdToExec, node.Definition.Name)
				Expect(err).ToNot(HaveOccurred(), "could not execute command: %s", err)

				Expect(len(strings.Fields(coreDumps))).To(BeNumerically(">=", 1), "error: vmcore dump was not generated")
			}

		})
	})
