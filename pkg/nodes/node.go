package nodes

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-gotests/pkg/msg"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/openshift-kni/eco-gotests/pkg/clients"
	v1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NodeBuilder provides struct for Node object containing connection to the cluster and the list of Node definitions.
type NodeBuilder struct {
	Definition *v1.Node
	Object     *v1.Node
	apiClient  *clients.Settings
	errorMsg   string
}

// PullNode pulls existing node from cluster.
func PullNode(apiClient *clients.Settings, nodeName string) (*NodeBuilder, error) {
	glog.V(100).Infof("Pulling existing node object: %s", nodeName)

	builder := NodeBuilder{
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
func (builder *NodeBuilder) Update() (*NodeBuilder, error) {
	glog.V(100).Infof("Updating configuration of node %s", builder.Definition.Name)

	if builder.errorMsg != "" {
		return nil, fmt.Errorf(builder.errorMsg)
	}

	if !builder.Exists() {
		return nil, fmt.Errorf("node object doesn't exist")
	}

	builder.Definition.CreationTimestamp = metaV1.Time{}
	builder.Definition.ResourceVersion = ""

	var err error
	builder.Object, err = builder.apiClient.Nodes().Update(context.TODO(), builder.Definition, metaV1.UpdateOptions{})

	return builder, err
}

// Exists checks whether the given node exists.
func (builder *NodeBuilder) Exists() bool {
	glog.V(100).Infof("Checking if node %s exists", builder.Definition.Name)

	var err error
	builder.Object, err = builder.apiClient.Nodes().Get(
		context.Background(), builder.Definition.Name, metaV1.GetOptions{})

	return err == nil || !k8serrors.IsNotFound(err)
}

// WithNewLabel defines the new label placed in the Node metadata.
func (builder *NodeBuilder) WithNewLabel(key, value string) *NodeBuilder {
	glog.V(100).Infof("Adding label %s=%s to node %s ", key, value, builder.Definition.Name)

	if builder.Definition == nil {
		builder.errorMsg = msg.UndefinedCrdObjectErrString("Node")
	}

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

// RemoveLabel removes given label from Node metadata.
func (builder *NodeBuilder) RemoveLabel(key, value string) *NodeBuilder {
	glog.V(100).Infof("Removing label %s=%s from node %s", key, value, builder.Definition.Name)

	if builder.Definition == nil {
		builder.errorMsg = msg.UndefinedCrdObjectErrString("Node")
	}

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
func (builder *NodeBuilder) ExternalIPv4Network() (string, error) {
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
