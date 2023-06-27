package nodes

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
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
	if valid, err := builder.validate(); !valid {
		return err
	}

	glog.V(100).Infof("Discovering nodes")

	builder.Objects = nil

	nodes, err := builder.apiClient.CoreV1Interface.Nodes().List(
		context.TODO(), metaV1.ListOptions{LabelSelector: builder.selector})
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
	if valid, err := builder.validate(); !valid {
		return nil, err
	}

	glog.V(100).Infof("Collecting node's external ipv4 addresses")

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

// validate will check that the builder and builder definition are properly initialized before
// accessing any member fields.
func (builder *Builder) validate() (bool, error) {
	resourceCRD := "Nodes"

	if builder == nil {
		glog.V(100).Infof("The %s builder is uninitialized", resourceCRD)

		return false, fmt.Errorf("error: received nil %s builder", resourceCRD)
	}

	if builder.apiClient == nil {
		glog.V(100).Infof("The %s builder apiclient is nil", resourceCRD)

		builder.errorMsg = fmt.Sprintf("%s builder cannot have nil apiClient", resourceCRD)
	}

	if builder.errorMsg != "" {
		glog.V(100).Infof("The %s builder has error message: %s", resourceCRD, builder.errorMsg)

		return false, fmt.Errorf(builder.errorMsg)
	}

	return true, nil
}
