package ocm

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/msg"
	kacv1 "github.com/stolostron/klusterlet-addon-controller/pkg/apis/agent/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// KACBuilder provides a struct for the KlusterletAddonConfig resource containing a connection to the cluster and the
// KlusterletAddonConfig definition.
type KACBuilder struct {
	// Definition of the KlusterletAddonConfig used to create the object.
	Definition *kacv1.KlusterletAddonConfig
	// Object of the KlusterletAddonConfig as it is on the cluster.
	Object *kacv1.KlusterletAddonConfig
	// apiClient used to interact with the cluster.
	apiClient runtimeclient.Client
	// errorMsg used to store latest error message from functions that do not return errors.
	errorMsg string
}

// NewKACBuilder creates a new instance of a KlusterletAddonConfig builder.
func NewKACBuilder(apiClient *clients.Settings, name, nsname string) *KACBuilder {
	glog.V(100).Infof(
		"Initializing new KlusterletAddonConfig structure with the following params: name: %s, nsname: %s", name, nsname)

	builder := &KACBuilder{
		Definition: &kacv1.KlusterletAddonConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
		},
	}

	if apiClient == nil {
		glog.V(100).Info("The apiClient for the KlusterletAddonConfig is nil")

		builder.errorMsg = "klusterletAddonConfig 'apiClient' cannot be nil"

		return builder
	}

	builder.apiClient = apiClient.Client

	if name == "" {
		glog.V(100).Info("The name of the KlusterletAddonConfig is empty")

		builder.errorMsg = "klusterletAddonConfig 'name' cannot be empty"

		return builder
	}

	if nsname == "" {
		glog.V(100).Info("The namespace of the KlusterletAddonConfig is empty")

		builder.errorMsg = "klusterletAddonConfig 'nsname' cannot be empty"

		return builder
	}

	return builder
}

// PullKAC pulls an existing KlusterletAddonConfig into a Builder struct.
func PullKAC(apiClient *clients.Settings, name, nsname string) (*KACBuilder, error) {
	glog.V(100).Infof("Pulling existing KlusterletAddonConfig %s under namespace %s from cluster", name, nsname)

	if apiClient == nil {
		glog.V(100).Info("The apiClient is empty")

		return nil, fmt.Errorf("klusterletAddonConfig 'apiClient' cannot be nil")
	}

	builder := &KACBuilder{
		apiClient: apiClient.Client,
		Definition: &kacv1.KlusterletAddonConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
		},
	}

	if name == "" {
		glog.V(100).Info("The name of the KlusterletAddonConfig is empty")

		return nil, fmt.Errorf("klusterletAddonConfig 'name' cannot be empty")
	}

	if nsname == "" {
		glog.V(100).Info("The namespace of the KlusterletAddonConfig is empty")

		return nil, fmt.Errorf("klusterletAddonConfig 'nsname' cannot be empty")
	}

	if !builder.Exists() {
		glog.V(100).Info("The KlusterletAddonConfig %s does not exist in namespace %s", name, nsname)

		return nil, fmt.Errorf("klusterletAddonConfig object %s does not exist in namespace %s", name, nsname)
	}

	builder.Definition = builder.Object

	return builder, nil
}

// Exists checks whether the given KlusterletAddonConfig exists on the cluster.
func (builder *KACBuilder) Exists() bool {
	if valid, _ := builder.validate(); !valid {
		return false
	}

	glog.V(100).Infof(
		"Checking if KlusterletAddonConfig %s exists in namespace %s", builder.Definition.Name, builder.Definition.Namespace)

	klusterletAddonConfig := &kacv1.KlusterletAddonConfig{}
	err := builder.apiClient.Get(context.TODO(), runtimeclient.ObjectKey{
		Name:      builder.Definition.Name,
		Namespace: builder.Definition.Namespace,
	}, klusterletAddonConfig)

	if err == nil {
		builder.Object = klusterletAddonConfig
	}

	return err == nil || !k8serrors.IsNotFound(err)
}

// Create makes a KlusterletAddonConfig on the cluster if it does not already exist.
func (builder *KACBuilder) Create() (*KACBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return nil, err
	}

	glog.V(100).Infof(
		"Creating KlusterletAddonConfig %s in namespace %s", builder.Definition.Name, builder.Definition.Namespace)

	if builder.Exists() {
		return builder, nil
	}

	err := builder.apiClient.Create(context.TODO(), builder.Definition)
	if err != nil {
		return nil, err
	}

	builder.Object = builder.Definition

	return builder, err
}

// Update changes the existing KlusterletAddonConfig resource on the cluster, falling back to deleting and recreating if
// the update fails when force is set.
func (builder *KACBuilder) Update(force bool) (*KACBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return nil, err
	}

	glog.V(100).Infof(
		"Updating KlusterletAddonConfig %s in namespace %s", builder.Definition.Name, builder.Definition.Namespace)

	if !builder.Exists() {
		glog.V(100).Infof(
			"KlusterletAddonConfig %s does not exist in namespace %s", builder.Definition.Name, builder.Definition.Namespace)

		return nil, fmt.Errorf("cannot update non-existent klusterletAddonConfig")
	}

	err := builder.apiClient.Update(context.TODO(), builder.Definition)
	if err != nil {
		if force {
			glog.V(100).Infof(msg.FailToUpdateNotification("klusterletAddonConfig", builder.Definition.Name))

			err := builder.Delete()
			if err != nil {
				glog.V(100).Infof(msg.FailToUpdateError("klusterletAddonConfig", builder.Definition.Name))

				return nil, err
			}

			return builder.Create()
		}

		return nil, err
	}

	builder.Object = builder.Definition

	return builder, nil
}

// Delete removes a KlusterletAddonConfig from the cluster if it exists.
func (builder *KACBuilder) Delete() error {
	if valid, err := builder.validate(); !valid {
		return err
	}

	glog.V(100).Infof(
		"Deleting KlusterletAddonConfig %s in namespace %s", builder.Definition.Name, builder.Definition.Namespace)

	if !builder.Exists() {
		glog.V(100).Infof(
			"KlusterletAddonConfig %s in namespace %s does not exist",
			builder.Definition.Name, builder.Definition.Namespace)

		builder.Object = nil

		return nil
	}

	err := builder.apiClient.Delete(context.TODO(), builder.Object)
	if err != nil {
		return err
	}

	builder.Object = nil

	return nil
}

// validate checks that the builder, definition, and apiClient are properly initialized and there is no errorMsg.
func (builder *KACBuilder) validate() (bool, error) {
	resourceCRD := "klusterletAddonConfig"

	if builder == nil {
		glog.V(100).Infof("The %s builder is uninitialized", resourceCRD)

		return false, fmt.Errorf("error: received nil %s builder", resourceCRD)
	}

	if builder.Definition == nil {
		glog.V(100).Infof("The %s is uninitialized", resourceCRD)

		return false, fmt.Errorf(msg.UndefinedCrdObjectErrString(resourceCRD))
	}

	if builder.apiClient == nil {
		glog.V(100).Infof("The %s builder apiClient is nil", resourceCRD)

		return false, fmt.Errorf("%s builder cannot have nil apiClient", resourceCRD)
	}

	if builder.errorMsg != "" {
		glog.V(100).Infof("The %s builder has error message %s", resourceCRD, builder.errorMsg)

		return false, fmt.Errorf(builder.errorMsg)
	}

	return true, nil
}
