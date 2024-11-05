package webhook

import (
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"golang.org/x/net/context"

	admregv1 "k8s.io/api/admissionregistration/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	goclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// MutatingConfigurationBuilder provides struct for MutatingWebhookConfiguration object which contains connection
// to cluster and MutatingWebhookConfiguration definition.
type MutatingConfigurationBuilder struct {
	// MutatingWebhookConfiguration definition. Used to create MutatingWebhookConfiguration object.
	Definition *admregv1.MutatingWebhookConfiguration
	// Created MutatingWebhookConfiguration object.
	Object *admregv1.MutatingWebhookConfiguration
	// Used in functions that define or mutate MutatingWebhookConfiguration definitions.
	// errorMsg is processed before MutatingWebhookConfiguration object is created.
	errorMsg string
	// apiClient opens api connection to the cluster.
	apiClient *clients.Settings
}

// PullMutatingConfiguration pulls existing MutatingWebhookConfiguration from cluster.
func PullMutatingConfiguration(apiClient *clients.Settings, name string) (*MutatingConfigurationBuilder, error) {
	glog.V(100).Infof("Pulling existing MutatingWebhookConfiguration name %s from cluster", name)

	builder := MutatingConfigurationBuilder{
		apiClient: apiClient,
		Definition: &admregv1.MutatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the MutatingWebhookConfiguration is empty")

		builder.errorMsg = "MutatingWebhookConfiguration 'name' cannot be empty"
	}

	if !builder.Exists() {
		return nil, fmt.Errorf("MutatingWebhook object %s does not exist", name)
	}

	builder.Definition = builder.Object

	return &builder, nil
}

// Exists checks whether the given MutatingWebhookConfiguration object exists in the cluster.
func (builder *MutatingConfigurationBuilder) Exists() bool {
	if valid, _ := builder.validate(); !valid {
		return false
	}

	var err error
	builder.Object, err = builder.Get()

	return err == nil || !k8serrors.IsNotFound(err)
}

// Get returns MutatingWebhookConfiguration object.
func (builder *MutatingConfigurationBuilder) Get() (*admregv1.MutatingWebhookConfiguration, error) {
	if valid, err := builder.validate(); !valid {
		return &admregv1.MutatingWebhookConfiguration{}, err
	}

	mutatingWebhookConfiguration := &admregv1.MutatingWebhookConfiguration{}
	err := builder.apiClient.Get(context.TODO(), goclient.ObjectKey{
		Name: builder.Definition.Name,
	}, mutatingWebhookConfiguration)

	if err != nil {
		glog.V(100).Infof("Failed to get MutatingConfigurationBuilder %s: %v", builder.Definition.Name, err)

		return &admregv1.MutatingWebhookConfiguration{}, err
	}

	return mutatingWebhookConfiguration, err
}

// Delete removes a MutatingWebhookConfiguration from a cluster.
func (builder *MutatingConfigurationBuilder) Delete() (*MutatingConfigurationBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Deleting the MutatingWebhookConfiguration %s", builder.Definition.Name)

	if !builder.Exists() {
		glog.V(100).Infof("MutatingWebhookConfiguration %s cannot be deleted because it does not exist",
			builder.Definition.Name)

		builder.Object = nil

		return builder, nil
	}

	err := builder.apiClient.Delete(context.TODO(), builder.Definition)

	if err != nil {
		return builder, fmt.Errorf("can not delete MutatingWebhookConfiguration: %w", err)
	}

	builder.Object = nil

	return builder, nil
}

// Update renovates the existing MutatingWebhookConfiguration object
// with the MutatingWebhookConfiguration definition in builder.
func (builder *MutatingConfigurationBuilder) Update() (*MutatingConfigurationBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Updating MutatingWebhookConfiguration %s", builder.Definition.Name)

	err := builder.apiClient.Update(context.TODO(), builder.Definition)
	if err == nil {
		builder.Object = builder.Definition
	}

	return builder, err
}

// validate will check that the builder and builder definition are properly initialized before
// accessing any member fields.
func (builder *MutatingConfigurationBuilder) validate() (bool, error) {
	resourceCRD := "MutatingWebhookConfiguration"

	if builder == nil {
		glog.V(100).Infof("The %s builder is uninitialized", resourceCRD)

		return false, fmt.Errorf("error: received nil %s builder", resourceCRD)
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
