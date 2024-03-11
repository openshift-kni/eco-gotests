package ecorecommon

import (
	"bytes"
	"fmt"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/openshift-kni/eco-gotests/tests/system-tests/ecore/internal/ecoreinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/ecore/internal/ecoreparams"
)

//nolint:funlen
func validateKernelModulesOnNodes(nodeSelectorString string, kernelModules []string) {
	if nodeSelectorString == "" {
		glog.V(ecoreparams.ECoreLogLevel).Infof("NodeSelector parameter is empty, skipping...")
		Skip("NodeSelector parameter is empty")
	}

	nodesSelector := metav1.ListOptions{
		LabelSelector: nodeSelectorString,
	}

	var (
		matchingNodes []*nodes.Builder
		err           error
		ctx           SpecContext
		tmpPods       []*pod.Builder
		cmdOutput     bytes.Buffer
	)

	Eventually(func() bool {
		glog.V(ecoreparams.ECoreLogLevel).Infof("Looking for node(s) matching %q selector", nodeSelectorString)

		matchingNodes, err = nodes.List(APIClient, nodesSelector)

		if err != nil {
			glog.V(ecoreparams.ECoreLogLevel).Infof("Error listing nodes: %v", err)

			return false
		}

		glog.V(ecoreparams.ECoreLogLevel).Infof("Found %d nodes matching label %q",
			len(matchingNodes), nodeSelectorString)

		return true
	}).WithContext(ctx).WithPolling(5*time.Second).WithTimeout(1*time.Minute).Should(BeTrue(),
		fmt.Sprintf("Error listing nodes matching selector %q", nodeSelectorString))

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

		Eventually(func() bool {
			tmpPods, err = pod.List(APIClient, ECoreConfig.MCONamespace, podSelector)

			if err != nil {
				glog.V(ecoreparams.ECoreLogLevel).Infof("Error listing pods: %v", err)

				return false
			}

			glog.V(ecoreparams.ECoreLogLevel).Infof("Found %d pods matching selector", len(tmpPods))

			return true
		}).WithContext(ctx).WithPolling(5*time.Second).WithTimeout(1*time.Minute).Should(BeTrue(),
			fmt.Sprintf("Failed to find machine-config-daemon pod on %q node", nodeName))

		Expect(len(tmpPods)).To(Equal(1), "Unexpected amount of machine-config-daemon pods found")

		podOne := tmpPods[0]

		glog.V(ecoreparams.ECoreLogLevel).Infof(
			"Pod: %q runs on node %q", podOne.Definition.Name, nodeName)

		By("Executing command from within the pod(s)")

		for _, moduleName := range kernelModules {
			glog.V(ecoreparams.ECoreLogLevel).Infof("Checking module: %q on %s", moduleName, nodeName)

			glog.V(ecoreparams.ECoreLogLevel).Infof("Reseting buffer's output")
			cmdOutput.Reset()

			grepCmd := fmt.Sprintf("chroot /rootfs bash -c lsmod | grep -e ^%v || echo qe_not_found", moduleName)
			lsmodCmd := []string{"/bin/bash", "-c"}

			lsmodCmd = append(lsmodCmd, grepCmd)

			glog.V(ecoreparams.ECoreLogLevel).Infof("Running command %q from pod %q",
				lsmodCmd, podOne.Definition.Name)

			Eventually(func() bool {
				cmdOutput, err = podOne.ExecCommand(lsmodCmd, ecoreparams.MachineConfigDaemonContainerName)

				if err != nil {
					glog.V(ecoreparams.ECoreLogLevel).Infof("Error running command: %v", err)

					return false
				}

				glog.V(ecoreparams.ECoreLogLevel).Infof("Command's output:\n\t%s", &cmdOutput)

				return true
			}).WithContext(ctx).WithPolling(5*time.Second).WithTimeout(1*time.Minute).Should(BeTrue(),
				fmt.Sprintf("Failed to run command from %q pod", podOne.Definition.Name))

			Expect(&cmdOutput).To(ContainSubstring(moduleName), fmt.Sprintf(
				"%q not loaded on %q", moduleName, nodeName))
		}
	}
}

// ValidateKernelModulesOnControlPlane verifies kernel modules on control-plane nodes.
func ValidateKernelModulesOnControlPlane() {
	if len(ECoreConfig.KernelModulesMap["node-role.kubernetes.io/master"]) == 0 {
		glog.V(ecoreparams.ECoreLogLevel).Infof("Kernel Modules Map parameter is empty, skipping...")
		Skip("Kernel Modules Map parameter is empty")
	}

	validateKernelModulesOnNodes("node-role.kubernetes.io/master",
		ECoreConfig.KernelModulesMap["node-role.kubernetes.io/master"])
}

// ValidateKernelModulesOnStandardNodes verifies kernel modules on standard nodes.
func ValidateKernelModulesOnStandardNodes() {
	if len(ECoreConfig.KernelModulesMap["node-role.kubernetes.io/standard"]) == 0 {
		glog.V(ecoreparams.ECoreLogLevel).Infof("Kernel Modules Map parameter is empty, skipping...")
		Skip("Kernel Modules Map parameter is empty")
	}

	validateKernelModulesOnNodes("node-role.kubernetes.io/standard",
		ECoreConfig.KernelModulesMap["node-role.kubernetes.io/standard"])
}
