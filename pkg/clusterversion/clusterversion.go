package clusterversion

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
	clusterVersionName = "version"
)

// Builder provides a struct for clusterversion object from the cluster and a clusterversion definition.
type Builder struct {
	// clusterversion definition, used to create the clusterversion object.
	Definition *v1.ClusterVersion
	// Created clusterversion object.
	Object *v1.ClusterVersion
	// api client to interact with the cluster.
	apiClient *clients.Settings
}

// Pull loads an existing clusterversion into Builder struct.
func Pull(apiClient *clients.Settings) (*Builder, error) {
	glog.V(100).Infof("Pulling existing clusterversion name: %s", clusterVersionName)

	builder := Builder{
		apiClient: apiClient,
		Definition: &v1.ClusterVersion{
			ObjectMeta: metaV1.ObjectMeta{
				Name: clusterVersionName,
			},
		},
	}

	if !builder.Exists() {
		return nil, fmt.Errorf("clusterversion object %s doesn't exist", clusterVersionName)
	}

	builder.Definition = builder.Object

	return &builder, nil
}

// Exists checks whether the given clusterversion exists.
func (builder *Builder) Exists() bool {
	glog.V(100).Infof(
		"Checking if clusterversion %s exists",
		builder.Definition.Name)

	var err error
	builder.Object, err = builder.apiClient.ConfigV1Interface.ClusterVersions().Get(
		context.Background(), builder.Definition.Name, metaV1.GetOptions{})

	return err == nil || !k8serrors.IsNotFound(err)
}
