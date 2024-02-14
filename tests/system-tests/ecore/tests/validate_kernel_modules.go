package ecore_system_test

import (
	"fmt"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/openshift-kni/eco-gotests/tests/system-tests/ecore/internal/ecoreinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/ecore/internal/ecoreparams"
)

var _ = Describe(
	"Validate Kernel Modules are loaded",
	Ordered,
	ContinueOnFailure,
	Label("validate_kernel_modules"), func() {
		DescribeTable("Verify kernel modules are loaded",
			func(nodeSelectorString string, kernelModules []string) {
				if len(ECoreConfig.KernelModulesMap) == 0 {
					glog.V(ecoreparams.ECoreLogLevel).Infof("Kernel Modules Map parameter is empty, skipping...")
					Skip("Kernel Modules Map parameter is empty")
				}

				if nodeSelectorString == "" {
					glog.V(ecoreparams.ECoreLogLevel).Infof("NodeSelector parameter is empty, skipping...")
					Skip("NodeSelector parameter is empty")
				}

				nodesSelector := metav1.ListOptions{
					LabelSelector: nodeSelectorString,
				}

				matchingNodes, err := nodes.List(APIClient, nodesSelector)
				Expect(err).ToNot(HaveOccurred(), "Failed to get nodes")
				Expect(len(matchingNodes)).NotTo(Equal(0), "0 nodes matching nodeSelector found")

				By(fmt.Sprintf("Getting pods in %s NS", ECoreConfig.MCONamespace))

				for _, _node := range matchingNodes {
					glog.V(ecoreparams.ECoreLogLevel).Infof("Found Node: %q", _node.Definition.ObjectMeta.Name)

					nodeName := _node.Definition.Name

					glog.V(ecoreparams.ECoreLogLevel).Infof(
						"Looking for machine-config-daemon pod running on node: %v", nodeName)

					podSelector := metav1.ListOptions{
						LabelSelector: ecoreparams.MachineConfidDaemonPodSelector,
						FieldSelector: fmt.Sprintf("spec.nodeName=%s", nodeName),
					}

					tmpPods, err := pod.List(APIClient, ECoreConfig.MCONamespace, podSelector)
					Expect(err).ToNot(HaveOccurred(), "Failed to find pods")
					Expect(len(tmpPods)).To(Equal(1), "Unexpected amount of machine-config-daemon pods found")

					pod := tmpPods[0]

					glog.V(ecoreparams.ECoreLogLevel).Infof(
						"Pod: %q runs on node %q", tmpPods[0].Definition.Name, nodeName)

					By("Executing command from within the pod(s)")

					for _, moduleName := range kernelModules {
						glog.V(ecoreparams.ECoreLogLevel).Infof("Checking module: %q on %s", moduleName, nodeName)

						grepCmd := fmt.Sprintf("chroot /rootfs bash -c lsmod | grep -e ^%v || echo qe_not_found", moduleName)
						lsmodCmd := []string{"/bin/bash", "-c"}

						lsmodCmd = append(lsmodCmd, grepCmd)

						glog.V(ecoreparams.ECoreLogLevel).Infof("Running command %q from pod %q",
							lsmodCmd, pod.Definition.Name)

						cmdOutput, err := pod.ExecCommand(lsmodCmd, ecoreparams.MachineConfigDaemonContainerName)
						Expect(err).ToNot(HaveOccurred(), fmt.Sprintf(
							"Failed to execute command %q on node %q", lsmodCmd, pod.Definition.Name))
						Expect(&cmdOutput).To(ContainSubstring(moduleName), fmt.Sprintf(
							"%q not loaded on %q", moduleName, nodeName))

						glog.V(ecoreparams.ECoreLogLevel).Infof("Command's output:\n\t%v", &cmdOutput)
					}
				}

			},
			Entry("Verify kernel modules on control-plane nodes", "node-role.kubernetes.io/master",
				ECoreConfig.KernelModulesMap["node-role.kubernetes.io/master"],
				Label("validate_kernel_modules_control_plane"), polarion.ID("67036")),
			Entry("Verify kernle modules on standard nodes", "node-role.kubernetes.io/standard",
				ECoreConfig.KernelModulesMap["node-role.kubernetes.io/standard"],
				Label("validate_kernel_modules_standard"), polarion.ID("67034")),
		)
	})
