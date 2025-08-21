package imageregistry

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	imageregistryv1 "github.com/openshift/api/imageregistry/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/clients"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/msg"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	goclient "sigs.k8s.io/controller-runtime/pkg/client"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

// Builder provides a struct for imageRegistry object from the cluster and a imageRegistry definition.
type Builder struct {
	// imageRegistry definition, used to create the imageRegistry object.
	Definition *imageregistryv1.Config
	// Created imageRegistry object.
	Object *imageregistryv1.Config
	// api client to interact with the cluster.
	apiClient goclient.Client
	// Used in functions that define or mutate clusterOperator definition. errorMsg is processed before the
	// ClusterOperator object is created.
	errorMsg string
}

// Pull retrieves an existing imageRegistry object from the cluster.
func Pull(apiClient *clients.Settings, imageRegistryObjName string) (*Builder, error) {
	if apiClient == nil {
		glog.V(100).Infof("The apiClient is empty")

		return nil, fmt.Errorf("imageRegistry Config 'apiClient' cannot be empty")
	}

	if imageRegistryObjName == "" {
		glog.V(100).Infof("The name of the imageRegistry is empty")

		return nil, fmt.Errorf("imageRegistry 'imageRegistryObjName' cannot be empty")
	}

	glog.V(100).Infof(
		"Pulling imageRegistry object name: %s", imageRegistryObjName)

	builder := Builder{
		apiClient: apiClient.Client,
		Definition: &imageregistryv1.Config{
			ObjectMeta: metav1.ObjectMeta{
				Name: imageRegistryObjName,
			},
		},
	}

	if !builder.Exists() {
		return nil, fmt.Errorf("imageRegistry object %s does not exist", imageRegistryObjName)
	}

	builder.Definition = builder.Object

	return &builder, nil
}

// Get fetches existing imageRegistry from cluster.
func (builder *Builder) Get() (*imageregistryv1.Config, error) {
	if valid, err := builder.validate(); !valid {
		return nil, err
	}

	glog.V(100).Infof("Getting existing imageRegistry with name %s from cluster", builder.Definition.Name)

	imageRegistry := &imageregistryv1.Config{}
	err := builder.apiClient.Get(context.TODO(), goclient.ObjectKey{
		Name: builder.Definition.Name,
	}, imageRegistry)

	if err != nil {
		glog.V(100).Infof("ImageRegistry object %s does not exist", builder.Definition.Name)

		return nil, err
	}

	return imageRegistry, nil
}

// Exists checks whether the given imageRegistry exists.
func (builder *Builder) Exists() bool {
	if valid, _ := builder.validate(); !valid {
		return false
	}

	glog.V(100).Infof("Checking if imageRegistry %s exists", builder.Definition.Name)

	var err error
	builder.Object, err = builder.Get()

	return err == nil || !k8serrors.IsNotFound(err)
}

// Update renovates the imageRegistry in the cluster and stores the created object in struct.
func (builder *Builder) Update() (*Builder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Updating the imageRegistry %s", builder.Definition.Name)

	if !builder.Exists() {
		return nil, fmt.Errorf("imageRegistry object %s does not exist", builder.Definition.Name)
	}

	err := builder.apiClient.Update(context.TODO(), builder.Definition)
	if err == nil {
		builder.Object = builder.Definition
	}

	return builder, err
}

// GetManagementState fetches imageRegistry ManagementState.
func (builder *Builder) GetManagementState() (*operatorv1.ManagementState, error) {
	if valid, err := builder.validate(); !valid {
		return nil, err
	}

	glog.V(100).Infof("Getting imageRegistry ManagementState configuration")

	if !builder.Exists() {
		return nil, fmt.Errorf("imageRegistry object does not exist")
	}

	return &builder.Object.Spec.ManagementState, nil
}

// GetStorageConfig fetches imageRegistry Storage configuration.
func (builder *Builder) GetStorageConfig() (*imageregistryv1.ImageRegistryConfigStorage, error) {
	if valid, err := builder.validate(); !valid {
		return nil, err
	}

	glog.V(100).Infof("Getting imageRegistry Storage configuration")

	if !builder.Exists() {
		return nil, fmt.Errorf("imageRegistry object does not exist")
	}

	return &builder.Object.Spec.Storage, nil
}

// WithManagementState sets the imageRegistry operator's management state.
func (builder *Builder) WithManagementState(expectedManagementState operatorv1.ManagementState) *Builder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof(
		"Setting imageRegistry %s with ManagementState: %v",
		builder.Definition.Name, expectedManagementState)

	builder.Definition.Spec.ManagementState = expectedManagementState

	return builder
}

// WithStorage sets the imageRegistry operator's storage.
func (builder *Builder) WithStorage(expectedStorage imageregistryv1.ImageRegistryConfigStorage) *Builder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof(
		"Setting imageRegistry %s with Storage: %v",
		builder.Definition.Name, expectedStorage)

	builder.Definition.Spec.Storage = expectedStorage

	return builder
}

// validate will check that the builder and builder definition are properly initialized before
// accessing any member fields.
func (builder *Builder) validate() (bool, error) {
	resourceCRD := "Configs.ImageRegistry"

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

	if builder.errorMsg != "" {
		glog.V(100).Infof("The %s builder has error message: %s", resourceCRD, builder.errorMsg)

		return false, fmt.Errorf(builder.errorMsg)
	}

	return true, nil
}
