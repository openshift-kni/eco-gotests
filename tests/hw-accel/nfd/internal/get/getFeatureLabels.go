package get

import (
	"fmt"
	"strings"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/openshift-kni/eco-gotests/tests/hw-accel/nfd/nfdparams"
)

const feature = "feature"

// NodeFeatureLabels return a map of all labels found in specific node.
func NodeFeatureLabels(apiClient *clients.Settings, nodeSelector map[string]string) (map[string][]string, error) {
	nodes, err := nodes.List(apiClient, metav1.ListOptions{LabelSelector: labels.Set(nodeSelector).String()})
	if err != nil {
		glog.V(nfdparams.LogLevel).Infof("failed retrieving node list")

		return nil, err
	}

	nodeLabels := make(map[string][]string)
	for _, node := range nodes {
		nodeLabels[node.Object.Name] = []string{}

		for label, labelvalue := range node.Object.Labels {
			if strings.Contains(label, feature) {
				nodeLabels[node.Object.Name] = append(nodeLabels[node.Object.Name], fmt.Sprintf("%s=%s", label, labelvalue))
			}
		}
	}

	return nodeLabels, nil
}
