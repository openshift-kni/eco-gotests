package nodes

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-gotests/pkg/clients"
	v1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	labels "k8s.io/apimachinery/pkg/labels"
)

// Builder provides struct for Node object containing connection to the cluster and the list of Node definitions.
type Builder struct {
	Objects   []v1.Node
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

	nodes, err := builder.apiClient.Nodes().List(
		context.TODO(), metaV1.ListOptions{LabelSelector: builder.selector})
	builder.Objects = nodes.Items

	return err
}
