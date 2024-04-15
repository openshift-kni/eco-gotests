package ecore_system_test

import (
	"fmt"
	"strings"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/nto" //nolint:misspell
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/openshift-kni/eco-gotests/tests/system-tests/ecore/internal/ecoreinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/ecore/internal/ecoreparams"
)

var _ = Describe(
	"ECore PerformanceProfile Validation",
	Ordered,
	ContinueOnFailure,
	Label(ecoreparams.LabelEcoreValidatePerformanceProfile), func() {

		var htNodes []*nodes.Builder

		BeforeAll(func() {
			By("Pulling in PerformanceProfile")
			glog.V(ecoreparams.ECoreLogLevel).Infof("Pulling PerformanceProfile %q ", ECoreConfig.PerformanceProfileHTName)

			perfProfile, err := nto.Pull(APIClient, ECoreConfig.PerformanceProfileHTName)

			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to pull in PerformanceProfile %q",
				ECoreConfig.PerformanceProfileHTName))

			By("Getting HT nodes nodeSelector")
			var nodeSelectorPP map[string]string

			if perfProfile.Definition.Spec.NodeSelector != nil {
				nodeSelectorPP = perfProfile.Definition.Spec.NodeSelector
				glog.V(ecoreparams.ECoreLogLevel).Infof("NodeSelectorDefined in performance profile: %q", nodeSelectorPP)
			} else {
				nodeSelectorPP = ECoreConfig.NodeSelectorHTNodes
			}

			glog.V(ecoreparams.ECoreLogLevel).Infof("Using next node selector for HT nodes: %v", nodeSelectorPP)

			By("Getting nodes matching HT nodeSelector")
			var tmpNodeSelector []string

			for k, v := range nodeSelectorPP {
				tmpNodeSelector = append(tmpNodeSelector, fmt.Sprintf("%s=%s", k, v))
			}

			nodeSelectorString := strings.Join(tmpNodeSelector, ",")
			glog.V(ecoreparams.ECoreLogLevel).Infof("NodeSelectorLabels: %v", nodeSelectorString)

			nodesSelector := metav1.ListOptions{
				LabelSelector: nodeSelectorString,
			}

			htNodes, err = nodes.List(APIClient, nodesSelector)
			Expect(err).ToNot(HaveOccurred(), "Failed to get nodes")
			Expect(len(htNodes)).NotTo(Equal(0), "HT nodes not found")

			for _, htNode := range htNodes {
				glog.V(ecoreparams.ECoreLogLevel).Infof("HT Node: %v", htNode.Definition.ObjectMeta.Name)
			}
		})

		It("Assert PerformanceProfile exists", func() {
			glog.V(ecoreparams.ECoreLogLevel).Infof("\t*** Assert PerformanceProfile exists")
			glog.V(ecoreparams.ECoreLogLevel).Infof("Pulling PerformanceProfile %q ", ECoreConfig.PerformanceProfileHTName)

			perfProfile, err := nto.Pull(APIClient, ECoreConfig.PerformanceProfileHTName)

			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to pull in PerformanceProfile %q",
				ECoreConfig.PerformanceProfileHTName))

			Expect(perfProfile.Definition.Spec.HugePages).NotTo(BeNil(), "HugePages not configured or defined")

			glog.V(ecoreparams.ECoreLogLevel).Infof("Debug profile: %#v", perfProfile.Definition.Spec)

		})

		It("Asserts CPU pinning on HT nodes", reportxml.ID("67038"), func() {
			glog.V(ecoreparams.ECoreLogLevel).Infof("\t*** Assert CPU pinning")

			By("Pulling in PerformanceProfile")
			glog.V(ecoreparams.ECoreLogLevel).Infof("Pulling PerformanceProfile %q ", ECoreConfig.PerformanceProfileHTName)

			perfProfile, err := nto.Pull(APIClient, ECoreConfig.PerformanceProfileHTName)

			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to pull in PerformanceProfile %q",
				ECoreConfig.PerformanceProfileHTName))

			Expect(perfProfile.Definition.Spec.CPU).NotTo(BeNil(), "CPU config not configured or defined")
			Expect(perfProfile.Definition.Spec.CPU.Isolated).NotTo(BeNil(), "Isolated cores not configured")
			Expect(perfProfile.Definition.Spec.CPU.Reserved).NotTo(BeNil(), "Reserved cores not configured")

			glog.V(ecoreparams.ECoreLogLevel).Infof("\t* Isolated CPU config: %#v", *perfProfile.Definition.Spec.CPU.Isolated)
			glog.V(ecoreparams.ECoreLogLevel).Infof("\t* Reserved CPU config: %#v", *perfProfile.Definition.Spec.CPU.Reserved)

			By(fmt.Sprintf("Getting pods in %s NS", ECoreConfig.MCONamespace))
			var mcPods []*pod.Builder

			for _, htNode := range htNodes {
				nodeName := htNode.Definition.ObjectMeta.Name
				glog.V(ecoreparams.ECoreLogLevel).Infof(
					"Looking for machine-config-daemon pod running on node: %v", nodeName)

				podSelector := metav1.ListOptions{
					LabelSelector: ecoreparams.MachineConfidDaemonPodSelector,
					FieldSelector: fmt.Sprintf("spec.nodeName=%s", nodeName),
				}

				tmpPods, err := pod.List(APIClient, ECoreConfig.MCONamespace, podSelector)
				Expect(err).ToNot(HaveOccurred(), "Failed to find pods")
				Expect(len(tmpPods)).To(Equal(1), "More then 1 pod found")

				glog.V(ecoreparams.ECoreLogLevel).Infof(
					"Pod: %q runs on node %q", tmpPods[0].Definition.Name, nodeName)
				mcPods = append(mcPods, tmpPods...)

			}

			By("Executing command from within the pod(s)")
			cpuCmd := []string{"/bin/bash", "-c", "cat /proc/cmdline"}
			affinityCPU := fmt.Sprintf("systemd.cpu_affinity=%s", *perfProfile.Definition.Spec.CPU.Reserved)
			isolCPUs := fmt.Sprintf("isolcpus=managed_irq,%s", *perfProfile.Definition.Spec.CPU.Isolated)

			for _, pod := range mcPods {
				glog.V(ecoreparams.ECoreLogLevel).Infof("Running command %q from pod %q",
					cpuCmd, pod.Definition.Name)

				cmdOutput, err := pod.ExecCommand(cpuCmd, ecoreparams.MachineConfigDaemonContainerName)
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf(
					"Failed to execute command %q on node %q", cpuCmd, pod.Definition.Name))

				glog.V(ecoreparams.ECoreLogLevel).Infof("Command's output:\n\t%v", &cmdOutput)
				Expect(&cmdOutput).To(ContainSubstring(affinityCPU), fmt.Sprintf(
					"%q not found in command's output", affinityCPU))
				Expect(&cmdOutput).To(ContainSubstring(isolCPUs), fmt.Sprintf(
					"%q not found in command's output", isolCPUs))

			}

		})

		It("Asserts HugePages configured on the HT nodes", reportxml.ID("67039"),
			Label("ecore_performance_huge_pages"), func() {
				glog.V(ecoreparams.ECoreLogLevel).Infof("\t*** Assert HugePages")

				By("Pulling in PerformanceProfile")
				glog.V(ecoreparams.ECoreLogLevel).Infof("Pulling PerformanceProfile %q ", ECoreConfig.PerformanceProfileHTName)

				perfProfile, err := nto.Pull(APIClient, ECoreConfig.PerformanceProfileHTName)

				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to pull in PerformanceProfile %q",
					ECoreConfig.PerformanceProfileHTName))

				Expect(perfProfile.Definition.Spec.HugePages).NotTo(BeNil(), "HugePages not configured or defined")
				Expect(perfProfile.Definition.Spec.HugePages.Pages).NotTo(BeNil(), "HugePages list not configured")
				Expect(len(perfProfile.Definition.Spec.HugePages.Pages)).NotTo(Equal(0), "HugePages not configured")

				var hugePagesRequested int

				for hugePageKey, hugePageValue := range perfProfile.Definition.Spec.HugePages.Pages {
					glog.V(ecoreparams.ECoreLogLevel).Infof("\t* HugePage: %v", hugePageKey)
					glog.V(ecoreparams.ECoreLogLevel).Infof("\t* HugePage: Size : %v", hugePageValue.Size)
					glog.V(ecoreparams.ECoreLogLevel).Infof("\t* HugePage: Count: %v", hugePageValue.Count)
					glog.V(ecoreparams.ECoreLogLevel).Infof("\t* HugePage: Node : %v", *hugePageValue.Node)
					glog.V(ecoreparams.ECoreLogLevel).Infof("\t*** *** ***")
					hugePagesRequested += int(hugePageValue.Count)
				}

				By(fmt.Sprintf("Getting pods in %s NS", ECoreConfig.MCONamespace))
				var mcPods []*pod.Builder

				for _, htNode := range htNodes {
					nodeName := htNode.Definition.ObjectMeta.Name
					glog.V(ecoreparams.ECoreLogLevel).Infof(
						"Looking for machine-config-daemon pod running on node: %v", nodeName)

					podSelector := metav1.ListOptions{
						LabelSelector: ecoreparams.MachineConfidDaemonPodSelector,
						FieldSelector: fmt.Sprintf("spec.nodeName=%s", nodeName),
					}

					tmpPods, err := pod.List(APIClient, ECoreConfig.MCONamespace, podSelector)
					Expect(err).ToNot(HaveOccurred(), "Failed to find pods")
					Expect(len(tmpPods)).To(Equal(1), "More then 1 pod found")

					glog.V(ecoreparams.ECoreLogLevel).Infof(
						"Pod: %q runs on node %q", tmpPods[0].Definition.Name, nodeName)
					mcPods = append(mcPods, tmpPods...)

				}

				By("Executing command from within the pod(s)")
				hugePagesCmd := []string{"/bin/bash", "-c", "cat /proc/meminfo"}
				hugePagesTotalRegexp := fmt.Sprintf("HugePages_Total:[[:space:]]+%d", hugePagesRequested)

				glog.V(ecoreparams.ECoreLogLevel).Infof("Regexp is %q", hugePagesTotalRegexp)

				for _, pod := range mcPods {
					glog.V(ecoreparams.ECoreLogLevel).Infof("Running command %q from pod %q",
						hugePagesCmd, pod.Definition.Name)

					cmdOutput, err := pod.ExecCommand(hugePagesCmd, ecoreparams.MachineConfigDaemonContainerName)
					Expect(err).ToNot(HaveOccurred(), fmt.Sprintf(
						"Failed to execute command %q on node %q", hugePagesCmd, pod.Definition.Name))

					glog.V(ecoreparams.ECoreLogLevel).Infof("Command's output:\n\t%v", &cmdOutput)
					Expect(&cmdOutput).To(MatchRegexp(hugePagesTotalRegexp), fmt.Sprintf(
						"%q not found in command's output", hugePagesTotalRegexp))

				}

			})

	})
