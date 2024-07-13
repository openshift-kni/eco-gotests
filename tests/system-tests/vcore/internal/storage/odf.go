package storage

import (
	"context"
	"fmt"

	odfoperatorv1alpha1 "github.com/red-hat-storage/odf-operator/api/v1alpha1"

	goclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/msg"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StorageSystemBuilder provides struct for storageSystem object containing connection
// to the cluster and the storageSystem definitions.
type StorageSystemBuilder struct {
	// StorageSystem definition. Used to create a storageSystem object
	Definition *odfoperatorv1alpha1.StorageSystem
	// Created storageSystem object
	Object *odfoperatorv1alpha1.StorageSystem
	// api client to interact with the cluster.
	apiClient goclient.Client
	// Used in functions that define or mutate storageSystem definition. errorMsg is processed before the
	// storageSystem object is created.
	errorMsg string
}

// NewStorageSystemBuilder creates a new instance of Builder.
func NewStorageSystemBuilder(apiClient *clients.Settings, name, nsname string) *StorageSystemBuilder {
	glog.V(100).Infof(
		"Initializing new storageSystem structure with the following params: %s, %s", name, nsname)

	if apiClient == nil {
		glog.V(100).Infof("storageSystem 'apiClient' cannot be empty")

		return nil
	}

	builder := &StorageSystemBuilder{
		apiClient: apiClient.Client,
		Definition: &odfoperatorv1alpha1.StorageSystem{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the storageSystem is empty")

		builder.errorMsg = "storageSystem 'name' cannot be empty"

		return builder
	}

	if nsname == "" {
		glog.V(100).Infof("The namespace of the storageSystem is empty")

		builder.errorMsg = "storageSystem 'nsname' cannot be empty"

		return builder
	}

	return builder
}

// PullStorageSystem gets an existing storageSystem object from the cluster.
func PullStorageSystem(apiClient *clients.Settings, name, namespace string) (*StorageSystemBuilder, error) {
	glog.V(100).Infof("Pulling existing storageSystem object %s from namespace %s",
		name, namespace)

	if apiClient == nil {
		glog.V(100).Infof("The apiClient is empty")

		return nil, fmt.Errorf("storageSystem 'apiClient' cannot be empty")
	}

	builder := StorageSystemBuilder{
		apiClient: apiClient.Client,
		Definition: &odfoperatorv1alpha1.StorageSystem{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the storageSystem is empty")

		return nil, fmt.Errorf("storageSystem 'name' cannot be empty")
	}

	if namespace == "" {
		glog.V(100).Infof("The namespace of the storageSystem is empty")

		return nil, fmt.Errorf("storageSystem 'namespace' cannot be empty")
	}

	if !builder.Exists() {
		return nil, fmt.Errorf("storageSystem object %s does not exist in namespace %s",
			name, namespace)
	}

	builder.Definition = builder.Object

	return &builder, nil
}

// Get fetches existing storageSystem from cluster.
func (builder *StorageSystemBuilder) Get() (*odfoperatorv1alpha1.StorageSystem, error) {
	if valid, err := builder.validate(); !valid {
		return nil, err
	}

	glog.V(100).Infof("Getting existing storageSystem with name %s from the namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	storageSystemObj := &odfoperatorv1alpha1.StorageSystem{}
	err := builder.apiClient.Get(context.TODO(), goclient.ObjectKey{
		Name:      builder.Definition.Name,
		Namespace: builder.Definition.Namespace,
	}, storageSystemObj)

	if err != nil {
		glog.V(100).Infof("storageSystem object %s does not exist in namespace %s",
			builder.Definition.Name, builder.Definition.Namespace)

		return nil, err
	}

	return storageSystemObj, nil
}

// Exists checks whether the given storageSystem exists.
func (builder *StorageSystemBuilder) Exists() bool {
	if valid, _ := builder.validate(); !valid {
		return false
	}

	glog.V(100).Infof("Checking if storageSystem %s exists in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	builder.Object, err = builder.Get()

	return err == nil || !k8serrors.IsNotFound(err)
}

// Create makes a storageSystem in the cluster and stores the created object in struct.
func (builder *StorageSystemBuilder) Create() (*StorageSystemBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Creating the storageSystem %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace,
	)

	var err error
	if !builder.Exists() {
		err = builder.apiClient.Create(context.TODO(), builder.Definition)
		if err == nil {
			builder.Object = builder.Definition
		}
	}

	return builder, err
}

// Delete removes storageSystem object from a cluster.
func (builder *StorageSystemBuilder) Delete() error {
	if valid, err := builder.validate(); !valid {
		return err
	}

	glog.V(100).Infof("Deleting the storageSystem object %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	err := builder.apiClient.Delete(context.TODO(), builder.Definition)

	if err != nil {
		return fmt.Errorf("can not delete storageSystem: %w", err)
	}

	builder.Object = nil

	return nil
}

// Update renovates the storageSystem in the cluster and stores the created object in struct.
func (builder *StorageSystemBuilder) Update() (*StorageSystemBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Updating the storageSystem %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	if !builder.Exists() {
		return nil, fmt.Errorf("storageSystem object %s does not exist in namespace %s",
			builder.Definition.Name, builder.Definition.Namespace)
	}

	err := builder.apiClient.Update(context.TODO(), builder.Definition)

	if err != nil {
		glog.V(100).Infof(
			msg.FailToUpdateError("storageSystem", builder.Definition.Name, builder.Definition.Namespace))

		return nil, err
	}

	builder.Object = builder.Definition

	return builder, nil
}

// WithStorageClusterSpec sets the storageSystem with storageCluster spec values.
func (builder *StorageSystemBuilder) WithStorageClusterSpec(
	kind odfoperatorv1alpha1.StorageKind, name, nsname string) *StorageSystemBuilder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof(
		"Setting storageSystem %s in namespace %s with storageCluster spec; \n"+
			"kind: %v, name: %s, namespace %s",
		builder.Definition.Name, builder.Definition.Namespace, kind, name, nsname)

	if kind == "" {
		glog.V(100).Infof("The kind of the storageCluster spec is empty")

		builder.errorMsg = "storageCluster spec 'kind' cannot be empty"

		return builder
	}

	if name == "" {
		glog.V(100).Infof("The name of the storageCluster spec is empty")

		builder.errorMsg = "storageCluster spec 'name' cannot be empty"

		return builder
	}

	if nsname == "" {
		glog.V(100).Infof("The namespace of the storageCluster spec is empty")

		builder.errorMsg = "storageCluster spec 'nsname' cannot be empty"

		return builder
	}

	builder.Definition.Spec.Kind = kind
	builder.Definition.Spec.Name = name
	builder.Definition.Spec.Namespace = nsname

	return builder
}

// validate will check that the builder and builder definition are properly initialized before
// accessing any member fields.
func (builder *StorageSystemBuilder) validate() (bool, error) {
	resourceCRD := "StorageSystem"

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
