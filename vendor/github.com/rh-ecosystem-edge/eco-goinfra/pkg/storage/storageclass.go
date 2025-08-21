package storage

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/clients"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/msg"
	v1 "k8s.io/api/core/v1"
	storageV1 "k8s.io/api/storage/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClassBuilder provides struct for storageclass object containing
// connection to the cluster and the storageclass definitions.
type ClassBuilder struct {
	// Storageclass definition. Used to create the storageclass object.
	Definition *storageV1.StorageClass
	// Created storageclass object
	Object *storageV1.StorageClass
	// Used in functions that define or mutate storageclass definition. errorMsg is processed before the storageclass
	// object is created.
	errorMsg  string
	apiClient *clients.Settings
}

// AdditionalOptions additional options for storageclass object.
type AdditionalOptions func(builder *ClassBuilder) (*ClassBuilder, error)

// NewClassBuilder creates a new instance of ClassBuilder.
func NewClassBuilder(apiClient *clients.Settings, name, provisioner string) *ClassBuilder {
	glog.V(100).Infof(
		"Initializing new storageclass structure with the following params: "+
			"name: %s, provisioner: %s", name, provisioner)

	builder := ClassBuilder{
		apiClient: apiClient,
		Definition: &storageV1.StorageClass{
			ObjectMeta: metaV1.ObjectMeta{
				Name: name,
			},
			Provisioner: provisioner,
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the storageclass is empty")

		builder.errorMsg = "storageclass 'name' cannot be empty"
	}

	if provisioner == "" {
		glog.V(100).Infof("The provisioner of the storageclass is empty")

		builder.errorMsg = "storageclass 'provisioner' cannot be empty"
	}

	return &builder
}

// WithReclaimPolicy adds a reclaimPolicy to the storageclass definition.
func (builder *ClassBuilder) WithReclaimPolicy(
	reclaimPolicy v1.PersistentVolumeReclaimPolicy) *ClassBuilder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	if reclaimPolicy == "" {
		glog.V(100).Infof("The reclaimPolicy of the storageclass is empty")

		builder.errorMsg = "storageclass 'reclaimPolicy' cannot be empty"
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.ReclaimPolicy = &reclaimPolicy

	return builder
}

// WithVolumeBindingMode adds a volumeBindingMode to the storage class definition.
func (builder *ClassBuilder) WithVolumeBindingMode(
	bindingMode storageV1.VolumeBindingMode) *ClassBuilder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	if bindingMode == "" {
		glog.V(100).Infof("The bindingMode of the storageclass is empty")

		builder.errorMsg = "storageclass 'bindingMode' cannot be empty"
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.VolumeBindingMode = &bindingMode

	return builder
}

// WithParameter adds a parameter to the storage class definition.
func (builder *ClassBuilder) WithParameter(parameterKey, parameterValue string) *ClassBuilder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	if parameterKey == "" {
		glog.V(100).Infof("The parameter key of the storageclass is empty")

		builder.errorMsg = "storageclass parameter key cannot be empty"
	}

	if parameterValue == "" {
		glog.V(100).Infof("The parameter value of the storageclass is empty")

		builder.errorMsg = "storageclass parameter value cannot be empty"
	}

	if builder.errorMsg != "" {
		return builder
	}

	if builder.Definition.Parameters == nil {
		builder.Definition.Parameters = make(map[string]string)
	}

	builder.Definition.Parameters[parameterKey] = parameterValue

	return builder
}

// WithOptions creates a storageclass with generic mutation options.
func (builder *ClassBuilder) WithOptions(options ...AdditionalOptions) *ClassBuilder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof("Setting storageclass additional options")

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

// Exists checks whether the given storageclass exists.
func (builder *ClassBuilder) Exists() bool {
	if valid, _ := builder.validate(); !valid {
		return false
	}

	glog.V(100).Infof("Checking if storageclass %s exists",
		builder.Definition.Name)

	var err error
	builder.Object, err = builder.apiClient.StorageClasses().Get(
		context.TODO(), builder.Definition.Name, metaV1.GetOptions{})

	return err == nil || !k8serrors.IsNotFound(err)
}

// Create generates a storageclass in cluster and stores the created object in struct.
func (builder *ClassBuilder) Create() (*ClassBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Creating storageclass %s", builder.Definition.Name)

	var err error
	if !builder.Exists() {
		builder.Object, err = builder.apiClient.StorageClasses().Create(
			context.TODO(), builder.Definition, metaV1.CreateOptions{})
	}

	return builder, err
}

// Delete removes a storageclass.
func (builder *ClassBuilder) Delete() error {
	if valid, err := builder.validate(); !valid {
		return err
	}

	glog.V(100).Infof("Deleting storageclass %s", builder.Definition.Name)

	if !builder.Exists() {
		return nil
	}

	err := builder.apiClient.StorageClasses().Delete(
		context.TODO(), builder.Object.Name, metaV1.DeleteOptions{})

	if err != nil {
		return err
	}

	builder.Object = nil

	return err
}

// Update renovates the existing storageclass object with the storageclass definition in builder.
func (builder *ClassBuilder) Update(force bool) (*ClassBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Updating storageclass %s",
		builder.Definition.Name)

	if !builder.Exists() {
		glog.V(100).Infof("storageclass %s does not exist",
			builder.Definition.Name)

		builder.errorMsg = "Cannot update non-existent storageclass"
	}

	if builder.errorMsg != "" {
		return nil, fmt.Errorf(builder.errorMsg)
	}

	err := builder.apiClient.Update(context.TODO(), builder.Definition)

	if err != nil {
		if force {
			glog.V(100).Infof(
				msg.FailToUpdateNotification("storageclass", builder.Definition.Name))

			err = builder.Delete()
			builder.Definition.ResourceVersion = ""

			if err != nil {
				glog.V(100).Infof(
					"Failed to update the storageclass object %s, "+
						"due to error in delete function",
					builder.Definition.Name,
				)

				return nil, err
			}

			return builder.Create()
		}
	}

	if err == nil {
		builder.Object = builder.Definition
	}

	return builder, err
}

// validate will check that the builder and builder definition are properly initialized before
// accessing any member fields.
func (builder *ClassBuilder) validate() (bool, error) {
	resourceCRD := "StorageClass"

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
