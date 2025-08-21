package console

import (
	"context"
	"fmt"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/golang/glog"
	v1 "github.com/openshift/api/config/v1"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/clients"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/msg"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Builder provides a struct for console object from the cluster and a console definition.
type Builder struct {
	// Console definition, used to create the pod object.
	Definition *v1.Console
	// Created console object.
	Object *v1.Console
	// api client to interact with the cluster.
	apiClient *clients.Settings
	// errorMsg is processed before console object is created.
	errorMsg string
}

// NewBuilder creates a new instance of Builder.
func NewBuilder(apiClient *clients.Settings, name string) *Builder {
	glog.V(100).Info("Initializing new console %s structure", name)

	builder := Builder{
		apiClient: apiClient,
		Definition: &v1.Console{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		},
	}

	if name == "" {
		glog.V(100).Info("The name of the Console is empty")

		builder.errorMsg = "console 'name' cannot be empty"
	}

	return &builder
}

// Pull loads an existing console into the Builder struct.
func Pull(apiClient *clients.Settings, name string) (*Builder, error) {
	builder := Builder{
		apiClient: apiClient,
		Definition: &v1.Console{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		},
	}

	if name == "" {
		glog.V(100).Info("The name of the Console is empty")

		builder.errorMsg = "console 'name' cannot be empty"
	}

	glog.V(100).Infof("Pulling cluster console %s", name)

	if !builder.Exists() {
		return nil, fmt.Errorf("the console object %s does not exist", name)
	}

	builder.Definition = builder.Object

	return &builder, nil
}

// Create makes a console in cluster and stores the created object in struct.
func (builder *Builder) Create() (*Builder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Creating the console %s", builder.Definition.Name)

	var err error
	if !builder.Exists() {
		builder.Object, err = builder.apiClient.Consoles().Create(
			context.TODO(), builder.Definition, metav1.CreateOptions{})
	}

	return builder, err
}

// Exists checks whether the given console exists.
func (builder *Builder) Exists() bool {
	if valid, _ := builder.validate(); !valid {
		return false
	}

	glog.V(100).Infof("Checking if console %s exists", builder.Definition.Name)

	var err error
	builder.Object, err = builder.apiClient.Consoles().Get(
		context.TODO(), builder.Definition.Name, metav1.GetOptions{})

	return err == nil || !k8serrors.IsNotFound(err)
}

// Delete removes a console object from a cluster.
func (builder *Builder) Delete() error {
	if valid, err := builder.validate(); !valid {
		return err
	}

	glog.V(100).Infof("Deleting the console object %s", builder.Definition.Name)

	if !builder.Exists() {
		return fmt.Errorf("console cannot be deleted because it does not exist")
	}

	err := builder.apiClient.Consoles().Delete(context.TODO(), builder.Definition.Name, metav1.DeleteOptions{})

	if err != nil {
		return fmt.Errorf("cannot delete console: %w", err)
	}

	builder.Object = nil

	return err
}

// Update renovates the existing cluster console object with cluster console definition in builder.
func (builder *Builder) Update() (*Builder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Info("Updating cluster console %s", builder.Definition.Name)

	var err error
	builder.Object, err = builder.apiClient.Consoles().Update(context.TODO(), builder.Definition,
		metav1.UpdateOptions{})

	return builder, err
}

// validate will check that the builder and builder definition are properly initialized before
// accessing any member fields.
func (builder *Builder) validate() (bool, error) {
	resourceCRD := "Console"

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
