package network

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-gotests/pkg/clients"
	v1 "github.com/openshift/api/config/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	clusterNetworkName = "cluster"
)

// ConfigBuilder provides a struct for network object from the cluster and a network definition.
type ConfigBuilder struct {
	// network definition, used to create the network object.
	Definition *v1.Network
	// Created network object.
	Object *v1.Network
	// api client to interact with the cluster.
	apiClient *clients.Settings
}

// PullConfig loads an existing network into ConfigBuilder struct.
func PullConfig(apiClient *clients.Settings) (*ConfigBuilder, error) {
	glog.V(100).Infof("Pulling existing network name: %s", clusterNetworkName)

	builder := ConfigBuilder{
		apiClient: apiClient,
		Definition: &v1.Network{
			ObjectMeta: metaV1.ObjectMeta{
				Name: clusterNetworkName,
			},
		},
	}

	if !builder.Exists() {
		return nil, fmt.Errorf("network object %s doesn't exist", clusterNetworkName)
	}

	builder.Definition = builder.Object

	return &builder, nil
}

// Exists checks whether the given network exists.
func (builder *ConfigBuilder) Exists() bool {
	glog.V(100).Infof(
		"Checking if network %s exists",
		builder.Definition.Name)

	var err error
	builder.Object, err = builder.apiClient.ConfigV1Interface.Networks().Get(
		context.Background(), builder.Definition.Name, metaV1.GetOptions{})

	return err == nil || !k8serrors.IsNotFound(err)
}
