package nodes

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/msg"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	v1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Builder provides struct for Node object containing connection to the cluster and the list of Node definitions.
type Builder struct {
	Definition *v1.Node
	Object     *v1.Node
	apiClient  *clients.Settings
	errorMsg   string
}

// AdditionalOptions additional options for node object.
type AdditionalOptions func(builder *Builder) (*Builder, error)

// Pull gathers existing node from cluster.
func Pull(apiClient *clients.Settings, nodeName string) (*Builder, error) {
	glog.V(100).Infof("Pulling existing node object: %s", nodeName)

	builder := Builder{
		apiClient: apiClient,
		Definition: &v1.Node{
			ObjectMeta: metaV1.ObjectMeta{
				Name: nodeName,
			},
		},
	}

	if !builder.Exists() {
		return nil, fmt.Errorf("node object %s doesn't exist", nodeName)
	}

	builder.Definition = builder.Object

	return &builder, nil
}

// Update renovates the existing node object with the node definition in builder.
func (builder *Builder) Update() (*Builder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Updating configuration of node %s", builder.Definition.Name)

	if !builder.Exists() {
		return nil, fmt.Errorf("node object doesn't exist")
	}

	builder.Definition.CreationTimestamp = metaV1.Time{}
	builder.Definition.ResourceVersion = ""

	var err error
	builder.Object, err = builder.apiClient.CoreV1Interface.Nodes().Update(
		context.TODO(), builder.Definition, metaV1.UpdateOptions{})

	return builder, err
}

// Exists checks whether the given node exists.
func (builder *Builder) Exists() bool {
	if valid, _ := builder.validate(); !valid {
		return false
	}

	glog.V(100).Infof("Checking if node %s exists", builder.Definition.Name)

	var err error
	builder.Object, err = builder.apiClient.CoreV1Interface.Nodes().Get(
		context.Background(), builder.Definition.Name, metaV1.GetOptions{})

	return err == nil || !k8serrors.IsNotFound(err)
}

// WithNewLabel defines the new label placed in the Node metadata.
func (builder *Builder) WithNewLabel(key, value string) *Builder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof("Adding label %s=%s to node %s ", key, value, builder.Definition.Name)

	if key == "" {
		glog.V(100).Infof("Failed to apply label with an empty key to node %s", builder.Definition.Name)
		builder.errorMsg = "error to set empty key to node"
	}

	if builder.errorMsg != "" {
		return builder
	}

	if builder.Definition.Labels == nil {
		builder.Definition.Labels = map[string]string{key: value}
	} else {
		_, labelExist := builder.Definition.Labels[key]
		if !labelExist {
			builder.Definition.Labels[key] = value
		} else {
			builder.errorMsg = fmt.Sprintf("cannot overwrite existing node label: %s", key)
		}
	}

	return builder
}

// WithOptions creates node with generic mutation options.
func (builder *Builder) WithOptions(options ...AdditionalOptions) *Builder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof("Setting node additional options")

	for _, option := range options {
		if option != nil {
			builder, err := option(builder)

			if err != nil {
				glog.V(100).Infof("Error occurred in mutation function")

				builder.errorMsg = err.Error()

				return builder
			}
		}
	}

	return builder
}

// RemoveLabel removes given label from Node metadata.
func (builder *Builder) RemoveLabel(key, value string) *Builder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof("Removing label %s=%s from node %s", key, value, builder.Definition.Name)

	if key == "" {
		glog.V(100).Infof("Failed to remove empty label's key from node %s", builder.Definition.Name)
		builder.errorMsg = "error to remove empty key from node"
	}

	if builder.errorMsg != "" {
		return builder
	}

	delete(builder.Definition.Labels, key)

	return builder
}

// ExternalIPv4Network returns nodes external ip address.
func (builder *Builder) ExternalIPv4Network() (string, error) {
	if valid, err := builder.validate(); !valid {
		return "", err
	}

	glog.V(100).Infof("Collecting node's external ipv4 addresses")

	if builder.Object == nil {
		builder.errorMsg = "error to collect external networks from node"
	}

	if builder.errorMsg != "" {
		return "", fmt.Errorf(builder.errorMsg)
	}

	var extNetwork ExternalNetworks
	err := json.Unmarshal([]byte(builder.Object.Annotations[ovnExternalAddresses]), &extNetwork)

	if err != nil {
		return "",
			fmt.Errorf("error to unmarshal node %s, annotation %s due to %w", builder.Object.Name, ovnExternalAddresses, err)
	}

	return extNetwork.IPv4, nil
}

// validate will check that the builder and builder definition are properly initialized before
// accessing any member fields.
func (builder *Builder) validate() (bool, error) {
	resourceCRD := "Node"

	if builder == nil {
		glog.V(100).Infof("The %s builder is uninitialized", resourceCRD)

		return false, fmt.Errorf("error: received nil %s builder", resourceCRD)
	}

	if builder.Definition == nil {
		glog.V(100).Infof("The %s is undefined", resourceCRD)

		builder.errorMsg = msg.UndefinedCrdObjectErrString(resourceCRD)
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
