package clusterversion

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/msg"
	v1 "github.com/openshift/api/config/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
			ObjectMeta: metav1.ObjectMeta{
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
	if valid, _ := builder.validate(); !valid {
		return false
	}

	glog.V(100).Infof(
		"Checking if clusterversion %s exists",
		builder.Definition.Name)

	var err error
	builder.Object, err = builder.apiClient.ConfigV1Interface.ClusterVersions().Get(
		context.Background(), builder.Definition.Name, metav1.GetOptions{})

	return err == nil || !k8serrors.IsNotFound(err)
}

// validate will check that the builder and builder definition are properly initialized before
// accessing any member fields.
func (builder *Builder) validate() (bool, error) {
	resourceCRD := "ClusterVersion"

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

		return false, fmt.Errorf("%s builder cannot have nil apiClient", resourceCRD)
	}

	return true, nil
}
