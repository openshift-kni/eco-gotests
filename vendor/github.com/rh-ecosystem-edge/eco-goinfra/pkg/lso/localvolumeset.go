package lso

import (
	"context"
	"fmt"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/golang/glog"
	lsoV1alpha1 "github.com/openshift/local-storage-operator/api/v1alpha1"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/clients"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/msg"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	goclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// LocalVolumeSetBuilder provides a struct for localVolumeSet object from the cluster and a localVolumeSet definition.
type LocalVolumeSetBuilder struct {
	// localVolumeSet definition, used to create the localVolumeSet object.
	Definition *lsoV1alpha1.LocalVolumeSet
	// Created localVolumeSet object.
	Object *lsoV1alpha1.LocalVolumeSet
	// Used in functions that define or mutate localVolumeSet definition. errorMsg is processed
	// before the localVolumeSet object is created
	errorMsg string
	// api client to interact with the cluster.
	apiClient *clients.Settings
}

// NewLocalVolumeSetBuilder creates new instance of LocalVolumeSetBuilder.
func NewLocalVolumeSetBuilder(apiClient *clients.Settings, name, nsname string) *LocalVolumeSetBuilder {
	glog.V(100).Infof("Initializing new %s localVolumeSet structure in %s namespace", name, nsname)

	builder := LocalVolumeSetBuilder{
		apiClient: apiClient,
		Definition: &lsoV1alpha1.LocalVolumeSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the localVolumeSet is empty")

		builder.errorMsg = "localVolumeSet 'name' cannot be empty"
	}

	if nsname == "" {
		glog.V(100).Infof("The nsname of the localVolumeSet is empty")

		builder.errorMsg = "localVolumeSet 'nsname' cannot be empty"
	}

	return &builder
}

// PullLocalVolumeSet retrieves an existing localVolumeSet object from the cluster.
func PullLocalVolumeSet(apiClient *clients.Settings, name, nsname string) (*LocalVolumeSetBuilder, error) {
	glog.V(100).Infof(
		"Pulling localVolumeSet object name: %s in namespace: %s", name, nsname)

	builder := LocalVolumeSetBuilder{
		apiClient: apiClient,
		Definition: &lsoV1alpha1.LocalVolumeSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the localVolumeSet is empty")

		builder.errorMsg = "localVolumeSet 'name' cannot be empty"
	}

	if nsname == "" {
		glog.V(100).Infof("The namespace of the localVolumeSet is empty")

		builder.errorMsg = "localVolumeSet 'nsname' cannot be empty"
	}

	if !builder.Exists() {
		return nil, fmt.Errorf("localVolumeSet object %s does not exist in namespace %s", name, nsname)
	}

	builder.Definition = builder.Object

	return &builder, nil
}

// Get fetches existing localVolumeSet from cluster.
func (builder *LocalVolumeSetBuilder) Get() (*lsoV1alpha1.LocalVolumeSet, error) {
	if valid, err := builder.validate(); !valid {
		return nil, err
	}

	glog.V(100).Infof("Pulling existing localVolumeSet with name %s under namespace %s from cluster",
		builder.Definition.Name, builder.Definition.Namespace)

	lvs := &lsoV1alpha1.LocalVolumeSet{}
	err := builder.apiClient.Get(context.TODO(), goclient.ObjectKey{
		Name:      builder.Definition.Name,
		Namespace: builder.Definition.Namespace,
	}, lvs)

	if err != nil {
		return nil, err
	}

	return lvs, nil
}

// Exists checks whether the given localVolumeSet exists.
func (builder *LocalVolumeSetBuilder) Exists() bool {
	if valid, _ := builder.validate(); !valid {
		return false
	}

	glog.V(100).Infof("Checking if localVolumeSet %s exists in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	builder.Object, err = builder.Get()

	return err == nil || !k8serrors.IsNotFound(err)
}

// Delete removes localVolumeSet from a cluster.
func (builder *LocalVolumeSetBuilder) Delete() error {
	if valid, err := builder.validate(); !valid {
		return err
	}

	glog.V(100).Infof("Deleting the localVolumeSet %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	if !builder.Exists() {
		return fmt.Errorf("localVolumeSet cannot be deleted because it does not exist")
	}

	err := builder.apiClient.Delete(context.TODO(), builder.Definition)

	if err != nil {
		return fmt.Errorf("can not delete localVolumeSet: %w", err)
	}

	builder.Object = nil

	return nil
}

// Create makes a LocalVolumeSetBuilder in the cluster and stores the created object in struct.
func (builder *LocalVolumeSetBuilder) Create() (*LocalVolumeSetBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Creating the LocalVolumeSetBuilder %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	if !builder.Exists() {
		err = builder.apiClient.Create(context.TODO(), builder.Definition)
		if err == nil {
			builder.Object = builder.Definition
		}
	}

	return builder, err
}

// Update renovates a LocalVolumeSetBuilder in the cluster and stores the created object in struct.
func (builder *LocalVolumeSetBuilder) Update() (*LocalVolumeSetBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Updating the LocalVolumeSetBuilder %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	if !builder.Exists() {
		return nil, fmt.Errorf("LocalVolumeSetBuilder object %s does not exist in namespace %s",
			builder.Definition.Name, builder.Definition.Namespace)
	}

	builder.Definition.CreationTimestamp = metav1.Time{}
	builder.Definition.ResourceVersion = ""

	err := builder.apiClient.Update(context.TODO(), builder.Definition)

	return builder, err
}

// validate will check that the builder and builder definition are properly initialized before
// accessing any member fields.
func (builder *LocalVolumeSetBuilder) validate() (bool, error) {
	resourceCRD := "LocalVolumeSet"

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
