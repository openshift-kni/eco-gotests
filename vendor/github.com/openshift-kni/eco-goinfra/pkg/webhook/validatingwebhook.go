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

// ValidatingConfigurationBuilder provides struct for ValidatingWebhookConfiguration object
// which contains connection to cluster and ValidatingWebhookConfiguration definition.
type ValidatingConfigurationBuilder struct {
	// ValidatingWebhookConfiguration definition. Used to create ValidatingWebhookConfiguration object.
	Definition *admregv1.ValidatingWebhookConfiguration
	// Created ValidatingWebhookConfiguration object.
	Object *admregv1.ValidatingWebhookConfiguration
	// Used in functions that define or mutate ValidatingWebhookConfiguration definitions.
	// errorMsg is processed before ValidatingWebhookConfiguration object is created.
	errorMsg string
	// apiClient opens api connection to the cluster.
	apiClient *clients.Settings
}

// PullValidatingConfiguration pulls existing ValidatingWebhookConfiguration from cluster.
func PullValidatingConfiguration(apiClient *clients.Settings, name string) (
	*ValidatingConfigurationBuilder, error) {
	glog.V(100).Infof("Pulling existing ValidatingWebhookConfiguration name %s from cluster", name)

	builder := ValidatingConfigurationBuilder{
		apiClient: apiClient,
		Definition: &admregv1.ValidatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the ValidatingWebhookConfiguration is empty")

		builder.errorMsg = "ValidatingWebhookConfiguration 'name' cannot be empty"
	}

	if !builder.Exists() {
		return nil, fmt.Errorf("ValidatingWebhookConfiguration object %s does not exist", name)
	}

	builder.Definition = builder.Object

	return &builder, nil
}

// Exists checks whether the given ValidatingWebhookConfiguration object exists in the cluster.
func (builder *ValidatingConfigurationBuilder) Exists() bool {
	if valid, _ := builder.validate(); !valid {
		return false
	}

	var err error
	builder.Object, err = builder.Get()

	return err == nil || !k8serrors.IsNotFound(err)
}

// Get returns ValidatingWebhookConfiguration object.
func (builder *ValidatingConfigurationBuilder) Get() (*admregv1.ValidatingWebhookConfiguration, error) {
	if valid, err := builder.validate(); !valid {
		return &admregv1.ValidatingWebhookConfiguration{}, err
	}

	validatingWebhookConfiguration := &admregv1.ValidatingWebhookConfiguration{}
	err := builder.apiClient.Get(context.TODO(), goclient.ObjectKey{
		Name: builder.Definition.Name,
	}, validatingWebhookConfiguration)

	if err != nil {
		glog.V(100).Infof("Failed to get ValidatingWebhookConfiguration %s: %v", builder.Definition.Name, err)

		return &admregv1.ValidatingWebhookConfiguration{}, err
	}

	return validatingWebhookConfiguration, err
}

// Delete removes a ValidatingWebhookConfiguration from a cluster.
func (builder *ValidatingConfigurationBuilder) Delete() (*ValidatingConfigurationBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Deleting the ValidatingWebhookConfiguration %s", builder.Definition.Name)

	if !builder.Exists() {
		glog.V(100).Infof("ValidatingWebhookConfiguration %s cannot be deleted"+
			" because it does not exist",
			builder.Definition.Name)

		builder.Object = nil

		return builder, nil
	}

	err := builder.apiClient.Delete(context.TODO(), builder.Definition)

	if err != nil {
		return builder, fmt.Errorf("can not delete ValidatingWebhookConfiguration: %w", err)
	}

	builder.Object = nil

	return builder, nil
}

// Update renovates the existing ValidatingWebhookConfiguration object
// with the ValidatingWebhookConfiguration definition in builder.
func (builder *ValidatingConfigurationBuilder) Update() (*ValidatingConfigurationBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Updating ValidatingWebhookConfiguration %s", builder.Definition.Name)

	err := builder.apiClient.Update(context.TODO(), builder.Definition)
	if err == nil {
		builder.Object = builder.Definition
	}

	return builder, err
}

// validate will check that the builder and builder definition are properly initialized before
// accessing any member fields.
func (builder *ValidatingConfigurationBuilder) validate() (bool, error) {
	resourceCRD := "ValidatingWebhookConfiguration"

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
