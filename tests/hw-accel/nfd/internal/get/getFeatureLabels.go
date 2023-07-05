package get

import (
	"fmt"
	"strings"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
)

const feature = "feature"

// NodeFeatureLabels return a map of all labels found in specific node.
func NodeFeatureLabels(apiClient *clients.Settings, nodeSelector map[string]string) (map[string][]string, error) {
	nodeBuilder := nodes.NewBuilder(apiClient, nodeSelector)
	nodeLabels := make(map[string][]string)

	if err := nodeBuilder.Discover(); err != nil {
		glog.V(100).Infof("could not discover %v nodes", nodeSelector)

		return nil, err
	}

	for _, node := range nodeBuilder.Objects {
		nodeLabels[node.Object.Name] = []string{}

		for label, labelvalue := range node.Object.Labels {
			if strings.Contains(label, feature) {
				nodeLabels[node.Object.Name] = append(nodeLabels[node.Object.Name], fmt.Sprintf("%s=%s", label, labelvalue))
			}
		}
	}

	return nodeLabels, nil
}
