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

// LocalVolumeDiscoveryBuilder provides a struct for localVolumeDiscovery object from the cluster
// and a localVolumeDiscovery definition.
type LocalVolumeDiscoveryBuilder struct {
	// localVolumeDiscovery definition, used to create the localVolumeDiscovery object.
	Definition *lsoV1alpha1.LocalVolumeDiscovery
	// Created localVolumeDiscovery object.
	Object *lsoV1alpha1.LocalVolumeDiscovery
	// Used in functions that define or mutate localVolumeDiscovery definition. errorMsg is processed
	// before the localVolumeDiscovery object is created
	errorMsg string
	// api client to interact with the cluster.
	apiClient *clients.Settings
}

// NewLocalVolumeDiscoveryBuilder creates new instance of LocalVolumeDiscoveryBuilder.
func NewLocalVolumeDiscoveryBuilder(apiClient *clients.Settings, name, nsname string) *LocalVolumeDiscoveryBuilder {
	glog.V(100).Infof("Initializing new localVolumeDiscovery structure with the following params: name: "+
		"%s, namespace: %s", name, nsname)

	builder := LocalVolumeDiscoveryBuilder{
		apiClient: apiClient,
		Definition: &lsoV1alpha1.LocalVolumeDiscovery{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the localVolumeDiscovery is empty")

		builder.errorMsg = "localVolumeDiscovery 'name' cannot be empty"
	}

	if nsname == "" {
		glog.V(100).Infof("The nsname of the localVolumeDiscovery is empty")

		builder.errorMsg = "localVolumeDiscovery 'nsname' cannot be empty"
	}

	return &builder
}

// PullLocalVolumeDiscovery retrieves an existing localVolumeDiscovery object from the cluster.
func PullLocalVolumeDiscovery(apiClient *clients.Settings, name, nsname string) (*LocalVolumeDiscoveryBuilder, error) {
	glog.V(100).Infof(
		"Pulling localVolumeDiscovery object name: %s in namespace: %s", name, nsname)

	builder := LocalVolumeDiscoveryBuilder{
		apiClient: apiClient,
		Definition: &lsoV1alpha1.LocalVolumeDiscovery{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the localVolumeDiscovery is empty")

		builder.errorMsg = "localVolumeDiscovery 'name' cannot be empty"
	}

	if nsname == "" {
		glog.V(100).Infof("The namespace of the localVolumeDiscovery is empty")

		builder.errorMsg = "localVolumeDiscovery 'nsname' cannot be empty"
	}

	if !builder.Exists() {
		return nil, fmt.Errorf("localVolumeDiscovery object %s does not exist in namespace %s", name, nsname)
	}

	builder.Definition = builder.Object

	return &builder, nil
}

// Get fetches existing localVolumeDiscovery from cluster.
func (builder *LocalVolumeDiscoveryBuilder) Get() (*lsoV1alpha1.LocalVolumeDiscovery, error) {
	if valid, err := builder.validate(); !valid {
		return nil, err
	}

	glog.V(100).Infof("Pulling existing localVolumeDiscovery with name %s under namespace %s from cluster",
		builder.Definition.Name, builder.Definition.Namespace)

	lvd := &lsoV1alpha1.LocalVolumeDiscovery{}
	err := builder.apiClient.Get(context.TODO(), goclient.ObjectKey{
		Name:      builder.Definition.Name,
		Namespace: builder.Definition.Namespace,
	}, lvd)

	if err != nil {
		return nil, err
	}

	return lvd, nil
}

// Exists checks whether the given localVolumeDiscovery exists.
func (builder *LocalVolumeDiscoveryBuilder) Exists() bool {
	if valid, _ := builder.validate(); !valid {
		return false
	}

	glog.V(100).Infof("Checking if localVolumeDiscovery %s exists in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	builder.Object, err = builder.Get()

	return err == nil || !k8serrors.IsNotFound(err)
}

// IsDiscovering check if the localVolumeDiscovery is Discovering.
func (builder *LocalVolumeDiscoveryBuilder) IsDiscovering() (bool, error) {
	if valid, err := builder.validate(); !valid {
		return false, err
	}

	glog.V(100).Infof("Verify localVolumeDiscovery %s in namespace %s is in Discovering phase",
		builder.Definition.Name, builder.Definition.Namespace)

	if !builder.Exists() {
		return false, fmt.Errorf("localVolumeDiscovery %s not found in %s namespace",
			builder.Definition.Name, builder.Definition.Namespace)
	}

	phase, err := builder.GetPhase()

	if err != nil {
		return false, fmt.Errorf("failed to get phase value for localVolumeDiscovery %s in namespace %s due to %w",
			builder.Definition.Name, builder.Definition.Namespace, err)
	}

	if phase == "Discovering" {
		return true, nil
	}

	return false, fmt.Errorf("invalid %s localVolumeDiscovery phase in %s namespace phase: %s",
		builder.Definition.Name, builder.Definition.Namespace, phase)
}

// Delete removes localVolumeDiscovery from a cluster.
func (builder *LocalVolumeDiscoveryBuilder) Delete() error {
	if valid, err := builder.validate(); !valid {
		return err
	}

	glog.V(100).Infof("Deleting the localVolumeDiscovery %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	if !builder.Exists() {
		return fmt.Errorf("localVolumeDiscovery cannot be deleted because it does not exist")
	}

	err := builder.apiClient.Delete(context.TODO(), builder.Definition)

	if err != nil {
		return fmt.Errorf("can not delete localVolumeDiscovery: %w", err)
	}

	builder.Object = nil

	return nil
}

// Create makes a localVolumeDiscovery in the cluster and stores the created object in struct.
func (builder *LocalVolumeDiscoveryBuilder) Create() (*LocalVolumeDiscoveryBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Creating the localVolumeDiscovery %s in namespace %s",
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

// Update renovates a localVolumeDiscovery in the cluster and stores the created object in struct.
func (builder *LocalVolumeDiscoveryBuilder) Update() (*LocalVolumeDiscoveryBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Updating the localVolumeDiscovery %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	if !builder.Exists() {
		return nil, fmt.Errorf("localVolumeDiscovery object %s does not exist in namespace %s",
			builder.Definition.Name, builder.Definition.Namespace)
	}

	builder.Definition.CreationTimestamp = metav1.Time{}
	builder.Definition.ResourceVersion = ""

	err := builder.apiClient.Update(context.TODO(), builder.Definition)

	return builder, err
}

// GetPhase get current localVolumeDiscovery phase.
func (builder *LocalVolumeDiscoveryBuilder) GetPhase() (lsoV1alpha1.DiscoveryPhase, error) {
	if valid, err := builder.validate(); !valid {
		return "", err
	}

	glog.V(100).Infof("Get %s localVolumeDiscovery in %s namespace phase",
		builder.Definition.Name, builder.Definition.Namespace)

	if !builder.Exists() {
		return "", fmt.Errorf("%s localVolumeDiscovery not found in %s namespace",
			builder.Definition.Name, builder.Definition.Namespace)
	}

	return builder.Object.Status.Phase, nil
}

// validate will check that the builder and builder definition are properly initialized before
// accessing any member fields.
func (builder *LocalVolumeDiscoveryBuilder) validate() (bool, error) {
	resourceCRD := "LocalVolumeDiscovery"

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
