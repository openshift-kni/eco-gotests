// The content of this file is taken from 'tests/hw-accel/nvidiagpu/internal/nvidiagpuconfig/device-config.go'. The full
// content should be moved to 'tests/hw-accel/internal'
// LabelPresentOnAllNodes
package get

import (
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/amdgpu/internal/amdgpuparams"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// LabelPresentOnAllNodes checks if label is present on all nodes matching nodeSelector.
func LabelPresentOnAllNodes(apiClient *clients.Settings, nodeLabel, nodeLabelValue string,
	nodeSelector map[string]string) (bool, error) {
	nodeBuilder, err := nodes.List(apiClient, metav1.ListOptions{LabelSelector: labels.Set(nodeSelector).String()})

	// in all the nodes that match the nodeSelectors, look for specific label
	// For example, look in all the worker nodes for a specific label with specific value
	if err != nil {
		glog.V(amdgpuparams.AmdGpuLogLevel).Infof("could not discover %v nodes, error encountered: '%v'",
			nodeSelector, err)

		return false, err
	}

	// Sample label: feature.node.kubernetes.io/system-os_release.ID=rhcos.
	foundLabels := 0

	for _, node := range nodeBuilder {
		labelValue := node.Object.Labels[nodeLabel]

		if labelValue == nodeLabelValue {
			glog.V(amdgpuparams.AmdGpuLogLevel).Infof("Found label %v that contains %v with label value %s on "+
				"node %v", nodeLabel, nodeLabel, nodeLabelValue, node.Object.Name)

			foundLabels++
			// if all nodes matching nodeSelector have this label with label value.
			if foundLabels == len(nodeBuilder) {
				return true, nil
			}
		}
	}

	err = fmt.Errorf("not all (%v) nodes have the label '%s' with value '%s'", len(nodeBuilder),
		nodeLabel, nodeLabelValue)

	return false, err
}

// LabelPresentOnAtLeastOneNode checks if label is present on at least one node matching nodeSelector.
func LabelPresentOnAtLeastOneNode(apiClient *clients.Settings,
	nodeLabel string, nodeSelector map[string]string) (bool, error) {
	nodeBuilder, err := nodes.List(apiClient, metav1.ListOptions{LabelSelector: labels.Set(nodeSelector).String()})

	// Check if at least one node matching the nodeSelector has the specific nodeLabel label set to true
	// For example, look in all the worker nodes for specific label
	if err != nil {
		glog.V(amdgpuparams.AmdGpuLogLevel).Infof("could not discover %v nodes", nodeSelector)

		return false, err
	}

	for _, node := range nodeBuilder {
		labelValue, ok := node.Object.Labels[nodeLabel]

		if ok {
			glog.V(amdgpuparams.AmdGpuLogLevel).Infof("Found label '%v' with label value '%v' on node '%v'",
				nodeLabel, labelValue, node.Object.Name)

			return true, nil
		}
	}

	err = fmt.Errorf("could not find one node with label '%s' set to true", nodeLabel)

	return false, err
}
