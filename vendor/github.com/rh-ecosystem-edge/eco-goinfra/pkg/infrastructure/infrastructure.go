package infrastructure

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	v1 "github.com/openshift/api/config/v1"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/clients"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/msg"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	infrastructureName = "cluster"
)

// Builder provides a struct for infrastructure object from the cluster and a infrastructure definition.
type Builder struct {
	// infrastructure definition, used to create the infrastructure object.
	Definition *v1.Infrastructure
	// Created infrastructure object.
	Object *v1.Infrastructure
	// api client to interact with the cluster.
	apiClient *clients.Settings
}

// Pull loads an existing infrastructure into Builder struct.
func Pull(apiClient *clients.Settings) (*Builder, error) {
	glog.V(100).Infof("Pulling existing infrastructure name: %s", infrastructureName)

	builder := Builder{
		apiClient: apiClient,
		Definition: &v1.Infrastructure{
			ObjectMeta: metaV1.ObjectMeta{
				Name: infrastructureName,
			},
		},
	}

	if !builder.Exists() {
		return nil, fmt.Errorf("infrastructure object %s doesn't exist", infrastructureName)
	}

	builder.Definition = builder.Object

	return &builder, nil
}

// Exists checks whether the given infrastructure exists.
func (builder *Builder) Exists() bool {
	if valid, _ := builder.validate(); !valid {
		return false
	}

	glog.V(100).Infof("Checking if infrastructure %s exists", builder.Definition.Name)

	var err error
	builder.Object, err = builder.apiClient.ConfigV1Interface.Infrastructures().Get(
		context.Background(), builder.Definition.Name, metaV1.GetOptions{})

	return err == nil || !k8serrors.IsNotFound(err)
}

// validate will check that the builder and builder definition are properly initialized before
// accessing any member fields.
func (builder *Builder) validate() (bool, error) {
	resourceCRD := "Infrastructure"

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
