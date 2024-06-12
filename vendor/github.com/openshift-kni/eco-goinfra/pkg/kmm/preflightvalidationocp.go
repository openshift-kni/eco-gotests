package kmm

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/msg"
	moduleV1Beta1 "github.com/rh-ecosystem-edge/kernel-module-management/api/v1beta1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	goclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// PreflightValidationOCPBuilder provides struct for the preflightvalidationocp object
// containing connection to the cluster and the preflightvalidationocp definitions.
type PreflightValidationOCPBuilder struct {
	// PreflightValidationOCP definition. Used to create a PreflightValidationOCP object.
	Definition *moduleV1Beta1.PreflightValidationOCP
	// Created PreflightValidationOCP object.
	Object *moduleV1Beta1.PreflightValidationOCP
	// ApiClient to interact with the cluster.
	apiClient *clients.Settings
	// errorMsg is processed before the object is created or updated.
	errorMsg string
}

// PreflightValidationOCPAdditionalOptions additional options for preflightvalidationocp object.
type PreflightValidationOCPAdditionalOptions func(
	builder *PreflightValidationOCPBuilder) (*PreflightValidationOCPBuilder, error)

// NewPreflightValidationOCPBuilder creates a new instance of PreflightValidationOCPBuilder.
func NewPreflightValidationOCPBuilder(
	apiClient *clients.Settings, name, nsname string) *PreflightValidationOCPBuilder {
	glog.V(100).Infof("Initializing new PreflightValidationOCP structure with following params: %s, %s",
		name, nsname)

	builder := PreflightValidationOCPBuilder{
		apiClient: apiClient,
		Definition: &moduleV1Beta1.PreflightValidationOCP{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the PreflightValidationOCP is empty")

		builder.errorMsg = "PreflightValidationOCP 'name' cannot be empty"
	}

	if nsname == "" {
		glog.V(100).Infof("The namespace of the PreflightValidationOCP is empty")

		builder.errorMsg = "PreflightValidationOCP 'nsname' cannot be empty"
	}

	return &builder
}

// WithReleaseImage sets the image for which the preflightvalidationocp checks the module.
func (builder *PreflightValidationOCPBuilder) WithReleaseImage(image string) *PreflightValidationOCPBuilder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	if image == "" {
		builder.errorMsg = "invald 'image' argument can not be nil"

		return builder
	}

	glog.V(100).Infof("Creating new PreflightValidationOCP with release image: %s",
		image)

	builder.Definition.Spec.ReleaseImage = image

	return builder
}

// WithUseRTKernel specifies if the kernel is realtime.
func (builder *PreflightValidationOCPBuilder) WithUseRTKernel(flag bool) *PreflightValidationOCPBuilder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof("Creating new PreflightValidationOCP with UseRTKernel set to: %s", flag)

	builder.Definition.Spec.UseRTKernel = flag

	return builder
}

// WithPushBuiltImage configures the build to be pushed to the registry.
func (builder *PreflightValidationOCPBuilder) WithPushBuiltImage(push bool) *PreflightValidationOCPBuilder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof("Creating new PreflightValidaionOCP with PushBuiltImage set to: %s", push)

	builder.Definition.Spec.PushBuiltImage = push

	return builder
}

// WithOptions creates Module with generic mutation options.
func (builder *PreflightValidationOCPBuilder) WithOptions(
	options ...PreflightValidationOCPAdditionalOptions) *PreflightValidationOCPBuilder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof("Setting Module additional options")

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

// PullPreflightValidationOCP fetches existing PreflightValidationOCP from the cluster.
func PullPreflightValidationOCP(apiClient *clients.Settings,
	name, nsname string) (*PreflightValidationOCPBuilder, error) {
	glog.V(100).Infof("Pulling existing preflightvalidationocp name % under namespace %s",
		name, nsname)

	if apiClient == nil {
		glog.V(100).Infof("The apiClient is empty")

		return nil, fmt.Errorf("preflightvalidation 'apiClient' cannot be empty")
	}

	builder := PreflightValidationOCPBuilder{
		apiClient: apiClient,
		Definition: &moduleV1Beta1.PreflightValidationOCP{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the preflightvalidationocp is empty")

		builder.errorMsg = "preflightvalidationocp 'name' cannot be empty"

		return &builder, fmt.Errorf(builder.errorMsg)
	}

	if nsname == "" {
		glog.V(100).Infof("The namespace of the preflightvalidationocp is empty")

		builder.errorMsg = "preflightvalidationocp 'nsname' cannot be empty"

		return &builder, fmt.Errorf(builder.errorMsg)
	}

	if !builder.Exists() {
		return nil, fmt.Errorf("preflightvalidationocp object %s doesn't exist in namespace %s",
			name, nsname)
	}

	builder.Definition = builder.Object

	return &builder, nil
}

// Create builds preflightvalidationocp in the cluster and stores object in struct.
func (builder *PreflightValidationOCPBuilder) Create() (*PreflightValidationOCPBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Creating preflightvalidationocp %s in namespace %s",
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

// Update modifies an existing preflightvalidationocp on the cluster.
func (builder *PreflightValidationOCPBuilder) Update() (*PreflightValidationOCPBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Updating preflightvalidationocp %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	err := builder.apiClient.Update(context.TODO(), builder.Definition)

	if err == nil {
		builder.Object = builder.Definition
	}

	return builder, err
}

// Exists checks if the defined preflightvalidationocp has already need created.
func (builder *PreflightValidationOCPBuilder) Exists() bool {
	if valid, _ := builder.validate(); !valid {
		return false
	}

	glog.V(100).Infof("Checking if preflightvalidationocp %s exists in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	builder.Object, err = builder.Get()

	return err == nil || !k8serrors.IsNotFound(err)
}

// Delete removes a preflightvalidationocp from the cluster.
func (builder *PreflightValidationOCPBuilder) Delete() (*PreflightValidationOCPBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Deleting preflightvalidationocp %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	if !builder.Exists() {
		return builder, nil
	}

	err := builder.apiClient.Delete(context.TODO(), builder.Definition)

	if err != nil {
		return builder, fmt.Errorf("cannot delete preflightvalidationocp: %w", err)
	}

	builder.Object = nil
	builder.Definition.ResourceVersion = ""

	return builder, err
}

// Get fetches the defined preflightvalidationocp from the cluster.
func (builder *PreflightValidationOCPBuilder) Get() (*moduleV1Beta1.PreflightValidationOCP, error) {
	if valid, err := builder.validate(); !valid {
		return nil, err
	}

	glog.V(100).Infof("Getting preflightvalidationocp %s from namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	preflightvalidationocp := &moduleV1Beta1.PreflightValidationOCP{}

	err := builder.apiClient.Get(context.TODO(), goclient.ObjectKey{
		Name:      builder.Definition.Name,
		Namespace: builder.Definition.Namespace,
	}, preflightvalidationocp)

	if err != nil {
		return nil, err
	}

	return preflightvalidationocp, nil
}

// validate will check that the builder and builder definition are properly initialized before
// accessing any member fields.
func (builder *PreflightValidationOCPBuilder) validate() (bool, error) {
	resourceCRD := "PreflightValidationOCP"

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
