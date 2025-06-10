package tests

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/amdgpu"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/amdgpu/internal/deviceconfig"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/amdgpu/internal/labels"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/amdgpu/internal/pods"
	amdparams "github.com/openshift-kni/eco-gotests/tests/hw-accel/amdgpu/params"
	"github.com/openshift-kni/eco-gotests/tests/internal/inittools"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("AMD GPU Basic Tests", Ordered, Label(amdparams.LabelSuite), func() {

	Context("AMD GPU Basic 01", Label(amdparams.LabelSuite+"-01"), func() {

		apiClient := inittools.APIClient

		amdListOptions := metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", amdparams.AMDNFDLabelKey, amdparams.AMDNFDLabelValue),
		}

		amdNodeBuilders, amdNodeBuildersErr := nodes.List(apiClient, amdListOptions)

		BeforeAll(func() {

			Expect(amdNodeBuildersErr).To(BeNil(),
				fmt.Sprintf("Failed to get Builders for AMD GPU Worker Nodes. Error:\n%v\n", amdNodeBuildersErr))

			Expect(amdNodeBuilders).ToNot(BeEmpty(),
				"'amdNodeBuilders' can't be empty")

		})

		It("Check AMD label was added by NFD", func() {

			By("Checking AMD label was added to all AMD GPU Worker Nodes by NFD")
			amdNFDLabelFound, amdNFDLabelFoundErr := labels.LabelPresentOnAllNodes(
				apiClient, amdparams.AMDNFDLabelKey, amdparams.AMDNFDLabelValue, inittools.GeneralConfig.WorkerLabelMap)

			Expect(amdNFDLabelFoundErr).To(BeNil(),
				"An error occurred while attempting to verify the AMD label by NFD: %v ", amdNFDLabelFoundErr)
			Expect(amdNFDLabelFound).To(BeTrue(),
				"AMD label check failed to match label %s and label value %s on all nodes",
				amdparams.AMDNFDLabelKey, amdparams.AMDNFDLabelValue)
		})

		It("Node Labeller", func() {

			By("Getting Device Config Builder")
			deviceConfigBuilder, deviceConfigBuilderErr := amdgpu.Pull(
				apiClient, amdparams.DeviceConfigName, amdparams.AMDGPUNamespace)
			Expect(deviceConfigBuilderErr).To(BeNil(),
				fmt.Sprintf("Failed to get DeviceConfig Builder. Error:\n%v\n", deviceConfigBuilderErr))

			By("Saving the Node Labeller state for post-test restoration")
			nodeLabellerEnabled := deviceconfig.IsNodeLabellerEnabled(deviceConfigBuilder)
			glog.V(amdparams.AMDGPULogLevel).Infof("nodeLabellerEnabled: %t", nodeLabellerEnabled)

			By("Enabling the Node Labeller")
			enableNodeLabellerErr := deviceconfig.SetEnableNodeLabeller(true, deviceConfigBuilder, false)
			Expect(enableNodeLabellerErr).To(BeNil(),
				fmt.Sprintf("Failed to enable NodeLabeller. Error:\n%v\n", enableNodeLabellerErr))

			By("Getting Node Labeller Pods from all AMD GPU Worker Nodes")
			nodeLabellerPodBuilders, err := pods.NodeLabellerPodsFromNodes(apiClient, amdNodeBuilders)
			Expect(err).To(BeNil(), fmt.Sprintf("Failed to get Node Labeller Pods: %v", err))

			By("Waiting for all Node Labeller Nodes to be in 'Running' state")
			for _, nodeLabellerPod := range nodeLabellerPodBuilders {
				err := nodeLabellerPod.WaitUntilRunning(amdparams.DefaultTimeout * time.Second)
				Expect(err).To(BeNil(), fmt.Sprintf("Got the following error while waiting for "+
					"Pod '%s' to be in 'Running' state:\n%v", nodeLabellerPod.Object.Name, err))
			}

			By("Validating all AMD labels are added to each AMD GPU Worker Node by the Node Labeller Pod")
			labelsCheckErr := labels.LabelsExistOnAllNodes(amdNodeBuilders, amdparams.NodeLabellerLabels,
				amdparams.DefaultTimeout*time.Second, amdparams.DefaultSleepInterval*time.Second)
			Expect(labelsCheckErr).To(BeNil(), fmt.Sprintf("Node Labeller labels don't "+
				"exist on all AMD GPU Worker Nodes: %v\n", labelsCheckErr))

			By("Getting a new Device Config Builder with all thr changes")
			deviceConfigBuilderNew, deviceConfigBuilderNewErr := amdgpu.Pull(
				apiClient, amdparams.DeviceConfigName, amdparams.AMDGPUNamespace)
			Expect(deviceConfigBuilderNewErr).To(BeNil(), fmt.Sprintf(
				"Failed to get DeviceConfigNew Builder. Error:\n%v\n", deviceConfigBuilderNewErr))

			By("Disabling the Node Labeller")
			disableNodeLabellerErr := deviceconfig.SetEnableNodeLabeller(false, deviceConfigBuilderNew, false)
			Expect(disableNodeLabellerErr).To(BeNil(),
				fmt.Sprintf("Failed to disable NodeLabeller. Error:\n%v\n", disableNodeLabellerErr))

			By("Make sure there are no Node Labeller Pods")
			noNodeLabellerPodsErr := pods.WaitUntilNoNodeLabellerPodes(apiClient)
			Expect(noNodeLabellerPodsErr).To(BeNil(), fmt.Sprintf("Got an error while waiting for "+
				"all Node Labeller Pods to be deleted. %v", noNodeLabellerPodsErr))

			By("Ensuring that all labels added by the Node Labeller are removed")
			missingNodeLabellerLabelsErr := labels.NodeLabellersLabelsMissingOnAllAMDGPUNode(amdNodeBuilders)
			Expect(missingNodeLabellerLabelsErr).To(BeNil(), fmt.Sprintf("failure occurred while checking "+
				"that Node Labeller Labels were removed from all AMD GPU Worker Nodes. %v", missingNodeLabellerLabelsErr))

			By("Ensuring that the Node Labeller state is the same as it was before the test")
			deviceConfigBuilderNew.Exists()
			if nodeLabellerEnabled != deviceconfig.IsNodeLabellerEnabled(deviceConfigBuilderNew) {
				restoringNodeLabellerErr := deviceconfig.SetEnableNodeLabeller(nodeLabellerEnabled, deviceConfigBuilderNew, false)
				Expect(restoringNodeLabellerErr).To(BeNil(),
					fmt.Sprintf("Failed to restore enableNodeLabeller to '%t'. Error:\n%t\n",
						nodeLabellerEnabled, *deviceconfig.GetEnableNodeLabeller(deviceConfigBuilderNew)))
			}
		})
	})
})
