package nodes

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-gotests/pkg/clients"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	labels "k8s.io/apimachinery/pkg/labels"
)

// Builder provides struct for Node object containing connection to the cluster and the list of Node definitions.
type Builder struct {
	Objects   []*NodeBuilder
	apiClient *clients.Settings
	selector  string
	errorMsg  string
}

// NewBuilder method creates new instance of Builder.
func NewBuilder(apiClient *clients.Settings, selector map[string]string) *Builder {
	glog.V(100).Infof(
		"Initializing new node structure with labels: %s", selector)

	// Serialize selector
	serialSelector := labels.Set(selector).String()

	builder := &Builder{
		apiClient: apiClient,
		selector:  serialSelector,
	}

	if serialSelector == "" {
		glog.V(100).Infof("The list of labels is empty")

		builder.errorMsg = "The list of labels cannot be empty"
	}

	return builder
}

// Discover method gets the node items and stores them in the Builder struct.
func (builder *Builder) Discover() error {
	glog.V(100).Infof("Discovering nodes")

	if builder.errorMsg != "" {
		return fmt.Errorf(builder.errorMsg)
	}

	nodes, err := builder.apiClient.Nodes().List(context.TODO(), metaV1.ListOptions{LabelSelector: builder.selector})
	if err != nil {
		glog.V(100).Infof("Failed to discover nodes")

		return err
	}

	for _, node := range nodes.Items {
		copiedNode := node
		nodeBuilder := &NodeBuilder{
			apiClient:  builder.apiClient,
			Object:     &copiedNode,
			Definition: &copiedNode,
		}

		builder.Objects = append(builder.Objects, nodeBuilder)
	}

	return err
}

// ExternalIPv4Networks returns a list of node's external ipv4 addresses.
func (builder *Builder) ExternalIPv4Networks() ([]string, error) {
	glog.V(100).Infof("Collecting node's external ipv4 addresses")

	if builder.errorMsg != "" {
		return nil, fmt.Errorf(builder.errorMsg)
	}

	if builder.Objects == nil {
		return nil, fmt.Errorf("error to collect external networks from nodes")
	}

	var ipV4ExternalAddresses []string

	for _, node := range builder.Objects {
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
