package nodes

import (
	"context"
	"fmt"

	"github.com/openshift-kni/eco-gotests/pkg/clients"
	v1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	labels "k8s.io/apimachinery/pkg/labels"
)

// Builder provides struct for Node object which contains connection to cluster and list of Node definitions.
type Builder struct {
	Objects   []v1.Node
	apiClient *clients.Settings
	selector  string
	errorMsg  string
}

// NewBuilder method creates new instance of Builder.
func NewBuilder(apiClient *clients.Settings, selector map[string]string) *Builder {
	// Serialize selector
	serialSelector := labels.Set(selector).String()

	builder := &Builder{
		apiClient: apiClient,
		selector:  serialSelector,
	}

	if serialSelector == "" {
		builder.errorMsg = "error node selector is empty"
	}

	return builder
}

// Discover method gets the node items and stores them in the Builder struct.
func (builder *Builder) Discover() error {
	if builder.errorMsg != "" {
		return fmt.Errorf(builder.errorMsg)
	}

	nodes, err := builder.apiClient.Nodes().List(
		context.TODO(), metaV1.ListOptions{LabelSelector: builder.selector})
	builder.Objects = nodes.Items

	return err
}
