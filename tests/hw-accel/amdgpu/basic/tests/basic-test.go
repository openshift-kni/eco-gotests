package tests

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/amdgpu/internal/deviceconfig"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/amdgpu/internal/get"

	"github.com/openshift-kni/eco-goinfra/pkg/amdgpu"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	tsp "github.com/openshift-kni/eco-gotests/tests/hw-accel/amdgpu/basic/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/internal/inittools"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("AMD GPU Basic Tests", Ordered, Label(tsp.LabelSuite), func() {

	Context("AMD GPU Basic 01", Label(tsp.LabelSuite+"-01"), func() {

		apiClient := inittools.APIClient

		amdListOptions := metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", tsp.AMDNFDLabelKey, tsp.AMDNFDLabelValue),
		}

		amdNodeBuilders, amdNodeBuildersErr := nodes.List(apiClient, amdListOptions)

		BeforeAll(func() {

			Expect(amdNodeBuildersErr).To(BeNil(),
				fmt.Sprintf("Failed to get Builders for AMD GPU Worker Nodes. Error:\n%v\n", amdNodeBuildersErr))

			Expect(amdNodeBuilders).ToNot(BeEmpty(),
				"'amdNodeBuilders' can't be empty")

		})

		It("Check AMD label was added by NFD", func() {

			// The assumption is that all Worker Nodes are equipped with AMD GPUs.
			amdNFDLabelFound, amdNFDLabelFoundErr := get.LabelPresentOnAllNodes(
				apiClient, tsp.AMDNFDLabelKey, tsp.AMDNFDLabelValue, inittools.GeneralConfig.WorkerLabelMap)

			Expect(amdNFDLabelFoundErr).To(BeNil(),
				"An error occurred while attempting to verify the AMD label by NFD: %v ", amdNFDLabelFoundErr)
			Expect(amdNFDLabelFound).To(BeTrue(),
				"AMD label check failed to match label %s and label value %s on all nodes",
				tsp.AMDNFDLabelKey, tsp.AMDNFDLabelValue)
		})

		It("Node Labeller", func() {

			deviceConfigBuilder, deviceConfigBuilderErr := amdgpu.Pull(apiClient, tsp.DeviceConfigName, tsp.AMDGPUNamespace)
			Expect(deviceConfigBuilderErr).To(BeNil(),
				fmt.Sprintf("Failed to get DeviceConfig Builder. Error:\n%v\n", deviceConfigBuilderErr))

			nodeLabellerEnabled := deviceconfig.IsNodeLabellerEnabled(deviceConfigBuilder)
			glog.V(tsp.AMDGPULogLevel).Infof("nodeLabellerEnabled: %t", nodeLabellerEnabled)

			nodeLabellerPodNamePrefix := fmt.Sprintf("%s-node-labeller-", tsp.DeviceConfigName)

			enableNodeLabellerErr := deviceconfig.SetEnableNodeLabeller(true, deviceConfigBuilder, false)
			Expect(enableNodeLabellerErr).To(BeNil(),
				fmt.Sprintf("Failed to enable NodeLabeller. Error:\n%v\n", enableNodeLabellerErr))

			// Once the Node Labeller is enabled, each AMD GPU Worker Node must have a
			// running Node Labeller Pod. The Node Labeller Pods will add AMD labels on each AMD Worker Node.
			// But, for this to happen, we need to *wait* for them to reach to 'Running' status.
			// Because a random postfix is added to the name of each Pod, we need to find their name first, and then
			// to wait for them to reach that status.

			var nodeLabellerPods []*pod.Builder
			allNodeLabellerPodsFound := false
			for _, amdNodeBuilder := range amdNodeBuilders {
				nodeName := amdNodeBuilder.Object.Name
				podListFieldSelector := fmt.Sprintf("spec.nodeName=%s", nodeName)

				for attempt := 1; attempt <= tsp.MaxAttempts && !allNodeLabellerPodsFound; attempt++ {

					time.Sleep(1 * time.Second)
					glog.V(tsp.AMDGPULogLevel).Infof("Trying to get NodeLabeller Pod from node '%s'.\n", nodeName)
					podsBuilder, podsListErr := pod.List(
						apiClient, tsp.AMDGPUNamespace, metav1.ListOptions{FieldSelector: podListFieldSelector})
					Expect(podsListErr).To(BeNil(),
						fmt.Sprintf("Failed to list Pods on node '%s'. Error:\n%v\n", nodeName, podsListErr))

					for _, pod := range podsBuilder {
						podName := pod.Object.Name

						if strings.HasPrefix(podName, nodeLabellerPodNamePrefix) {
							glog.V(tsp.AMDGPULogLevel).Infof("Node Labeller Pod is found. Name: %s\n", podName)
							nodeLabellerPods = append(nodeLabellerPods, pod)

							// If Node Labeller Pods were found on all nodes, no need to keep searching
							if len(nodeLabellerPods) == len(amdNodeBuilders) {
								allNodeLabellerPodsFound = true
							}

							break
						}
					}
				}
			}

			Expect(allNodeLabellerPodsFound).To(BeTrue(), "Node Labller Pods haven't been found on all nodes")

			nonRunningPodsCnt := len(nodeLabellerPods)
			for _, nodeLabellerPod := range nodeLabellerPods {
				for attempt := 1; attempt <= tsp.MaxAttempts; attempt++ {
					refreshedNodeLabellerPod, refreshedNodeLabellerPodErr := pod.Pull(
						apiClient, nodeLabellerPod.Object.Name, nodeLabellerPod.Object.Namespace)
					Expect(refreshedNodeLabellerPodErr).To(BeNil(), fmt.Sprintf(
						"Failed to pull refreshed Pod name '%v'. Error:\n%v\n",
						nodeLabellerPod.Object.Name, refreshedNodeLabellerPodErr))
					glog.V(tsp.AMDGPULogLevel).Infof(
						"Refreshed Node Labeller Pod '%s' status is '%v'",
						refreshedNodeLabellerPod.Object.Name, refreshedNodeLabellerPod.Object.Status.Phase)
					if refreshedNodeLabellerPod.Object.Status.Phase == "Running" {
						nonRunningPodsCnt--

						break
					}
					time.Sleep(1 * time.Second)
				}
			}

			Expect(nonRunningPodsCnt).To(BeZero(), "Expecting all Node Labeller Nodes to be in 'Running' status")

			// Now that all pods are ready & running, we can check for the labels on each node
			for _, amdNodeBuilder := range amdNodeBuilders {
				nodeName := amdNodeBuilder.Object.Name

				for _, label := range tsp.NodeLabellerLabels {

					labelFound := false
					value := ""

					for attempt := 1; attempt <= tsp.MaxAttempts; attempt++ {

						glog.V(tsp.AMDGPULogLevel).Infof("Checking for label '%s' on node '%s' (attempt #%v)\n", label, nodeName, attempt)

						v, lFound := amdNodeBuilder.Object.Labels[label]
						if lFound {
							labelFound = true
							value = v

							break
						}

						refreshedAMDNodeBuilder, refreshedAMDNodeBuilderErr := nodes.Pull(apiClient, nodeName)
						Expect(refreshedAMDNodeBuilderErr).To(BeNil(),
							fmt.Sprintf("Failed to pull Refreshed AMD Node Builder for node '%v'. Error:\n%v\n",
								nodeName, refreshedAMDNodeBuilderErr))
						amdNodeBuilder = refreshedAMDNodeBuilder

						time.Sleep(1 * time.Second)

					}

					Expect(labelFound).To(BeTrue(), fmt.Sprintf("Label %v not found on node %v\n", label, nodeName))

					if strings.HasSuffix(label, "device-id") {
						deviceName, deviceFound := tsp.DeviceIDsMap[value]
						Expect(deviceFound).To(BeTrue(),
							fmt.Sprintf("The device '%v' isn't found in the list of supported devices\n", value))
						glog.V(tsp.AMDGPULogLevel).Infof(
							"%v ('%v') device found on node '%v'", deviceName, value, nodeName)
					}
				}
			}

			// After done verifying all labels on all nodes, we can now unset the enableNodeLabeller
			// and make sure those labels are removed

			// An updated version of DeviceConfig is needed after applying some changes above in
			// order to be able to disable the Node Labeller
			deviceConfigBuilderNew, deviceConfigBuilderNewErr := amdgpu.Pull(
				apiClient, tsp.DeviceConfigName, tsp.AMDGPUNamespace)
			Expect(deviceConfigBuilderNewErr).To(BeNil(), fmt.Sprintf(
				"Failed to get DeviceConfigNew Builder. Error:\n%v\n", deviceConfigBuilderNewErr))
			disableNodeLabellerErr := deviceconfig.SetEnableNodeLabeller(false, deviceConfigBuilderNew, false)
			Expect(disableNodeLabellerErr).To(BeNil(),
				fmt.Sprintf("Failed to disable NodeLabeller. Error:\n%v\n", disableNodeLabellerErr))

			for _, amdNodeBuilder := range amdNodeBuilders {
				nodeName := amdNodeBuilder.Object.Name

				for _, label := range tsp.NodeLabellerLabels {

					labelFound := true

					for attempt := 1; attempt <= tsp.MaxAttempts; attempt++ {

						glog.V(tsp.AMDGPULogLevel).Infof(
							"Checking that the label '%s' was removed on node '%s' (attempt #%v)\n", label, nodeName, attempt)

						_, lFound := amdNodeBuilder.Object.Labels[label]
						if !lFound {
							labelFound = false

							break
						}

						refreshedAMDNodeBuilder, refreshedAMDNodeBuilderErr := nodes.Pull(apiClient, nodeName)
						Expect(refreshedAMDNodeBuilderErr).To(BeNil(),
							fmt.Sprintf("Failed to pull Refreshed AMD Node Builder (labels removal) for node '%v'. Error:\n%v\n",
								nodeName, refreshedAMDNodeBuilderErr))
						amdNodeBuilder = refreshedAMDNodeBuilder

						time.Sleep(1 * time.Second)

					}

					Expect(labelFound).To(BeFalse(), fmt.Sprintf("Label %v found on node %v\n", label, nodeName))

				}
			}

			// Check if 'enableNodeLabeller' is different then at the beginning & restore id needed
			if nodeLabellerEnabled != deviceconfig.IsNodeLabellerEnabled(deviceConfigBuilderNew) {
				restoringNodeLabellerErr := deviceconfig.SetEnableNodeLabeller(nodeLabellerEnabled, deviceConfigBuilderNew, false)
				Expect(restoringNodeLabellerErr).To(BeNil(),
					fmt.Sprintf("Failed to restore enableNodeLabeller to '%t'. Error:\n%v\n",
						nodeLabellerEnabled, restoringNodeLabellerErr))

			}

		})
	})
})
