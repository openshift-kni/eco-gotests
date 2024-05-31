package network

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/msg"
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
	if valid, _ := builder.validate(); !valid {
		return false
	}

	glog.V(100).Infof(
		"Checking if network %s exists",
		builder.Definition.Name)

	var err error
	builder.Object, err = builder.apiClient.ConfigV1Interface.Networks().Get(
		context.Background(), builder.Definition.Name, metaV1.GetOptions{})

	return err == nil || !k8serrors.IsNotFound(err)
}

// validate will check that the builder and builder definition are properly initialized before
// accessing any member fields.
func (builder *ConfigBuilder) validate() (bool, error) {
	resourceCRD := "Network.Config"

	if builder == nil {
		glog.V(100).Infof("The %s builder is uninitialized", resourceCRD)

		return false, fmt.Errorf("error: received nil %s builder", resourceCRD)
	}

	if builder.Definition == nil {
		glog.V(100).Infof("The %s is undefined", resourceCRD)

		return false, fmt.Errorf(msg.UndefinedCrdObjectErrString(resourceCRD))
	}

	if builder.apiClient == nil {
		glog.V(100).Infof("The %s builder apiclient is nil", resourceCRD)

		return false, fmt.Errorf(fmt.Sprintf("%s builder cannot have nil apiClient", resourceCRD))
	}

	return true, nil
}
