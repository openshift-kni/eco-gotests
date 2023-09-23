package nodes

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// List returns node inventory.
func List(apiClient *clients.Settings, options v1.ListOptions) ([]*Builder, error) {
	glog.V(100).Infof("Listing all node resources with the options %v", options)

	nodeList, err := apiClient.CoreV1Interface.Nodes().List(context.Background(), options)
	if err != nil {
		glog.V(100).Infof("Failed to list nodes due to %s", err.Error())

		return nil, err
	}

	var nodeObjects []*Builder

	for _, runningNode := range nodeList.Items {
		copiedNode := runningNode
		nodeBuilder := &Builder{
			apiClient:  apiClient,
			Object:     &copiedNode,
			Definition: &copiedNode,
		}

		nodeObjects = append(nodeObjects, nodeBuilder)
	}

	return nodeObjects, nil
}

// ListExternalIPv4Networks returns a list of node's external ipv4 addresses.
func ListExternalIPv4Networks(apiClient *clients.Settings, options v1.ListOptions) ([]string, error) {
	glog.V(100).Infof("Collecting node's external ipv4 addresses")

	var ipV4ExternalAddresses []string

	nodeBuilders, err := List(apiClient, options)
	if err != nil {
		return nil, err
	}

	for _, node := range nodeBuilders {
		extNodeNetwork, err := node.ExternalIPv4Network()
		if err != nil {
			glog.V(100).Infof("Failed to collect external ip address from node %s", node.Object.Name)

			return nil, fmt.Errorf(
				"error getting external IPv4 address from node %s due to %w", node.Definition.Name, err)
		}
		ipV4ExternalAddresses = append(ipV4ExternalAddresses, extNodeNetwork)
	}

	return ipV4ExternalAddresses, nil
}
