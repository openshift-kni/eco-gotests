package mco

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/msg"
	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MCBuilder provides struct for MachineConfig Object which contains connection to cluster
// and MachineConfig definitions.
type MCBuilder struct {
	// MachineConfig definition. Used to create MachineConfig object with minimum set of required elements.
	Definition *mcv1.MachineConfig
	// Created MachineConfig object on the cluster.
	Object *mcv1.MachineConfig
	// api client to interact with the cluster.
	apiClient *clients.Settings
	// errorMsg is processed before MachineConfig object is created.
	errorMsg string
}

// MCAdditionalOptions for machineconfig object.
type MCAdditionalOptions func(builder *MCBuilder) (*MCBuilder, error)

// NewMCBuilder provides struct for MachineConfig object which contains connection to cluster
// and MachineConfig definition.
func NewMCBuilder(apiClient *clients.Settings, name string) *MCBuilder {
	glog.V(100).Infof("Initializing new MCBuilder structure with following params: %s", name)

	builder := MCBuilder{
		apiClient: apiClient,
		Definition: &mcv1.MachineConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the MachineConfig is empty")

		builder.errorMsg = "MachineConfig 'name' cannot be empty"
	}

	return &builder
}

// PullMachineConfig fetches existing machineconfig from cluster.
func PullMachineConfig(apiClient *clients.Settings, name string) (*MCBuilder, error) {
	glog.V(100).Infof("Pulling existing machineconfig name %s from cluster", name)

	builder := MCBuilder{
		apiClient: apiClient,
		Definition: &mcv1.MachineConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the machineconfig is empty")

		builder.errorMsg = "machineconfig 'name' cannot be empty"
	}

	if !builder.Exists() {
		return nil, fmt.Errorf("machineconfig object %s doesn't exist", name)
	}

	builder.Definition = builder.Object

	return &builder, nil
}

// Create generates a machineconfig in the cluster and stores the created object in struct.
func (builder *MCBuilder) Create() (*MCBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Creating MachineConfig %s", builder.Definition.Name)

	var err error
	if !builder.Exists() {
		builder.Object, err = builder.apiClient.MachineConfigs().Create(
			context.TODO(), builder.Definition, metav1.CreateOptions{})
	}

	return builder, err
}

// Delete removes the machineconfig.
func (builder *MCBuilder) Delete() error {
	if valid, err := builder.validate(); !valid {
		return err
	}

	glog.V(100).Infof("Deleting the MachineConfig object %s", builder.Definition.Name)

	if !builder.Exists() {
		return fmt.Errorf("MachineConfig cannot be deleted because it does not exist")
	}

	err := builder.apiClient.MachineConfigs().Delete(
		context.TODO(), builder.Object.Name, metav1.DeleteOptions{})

	if err != nil {
		return fmt.Errorf("cannot delete MachineConfig: %w", err)
	}

	builder.Object = nil

	return err
}

// Update renovates the existing machineconfig object with machineconfig definition in builder.
func (builder *MCBuilder) Update() (*MCBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Updating machineconfig %s", builder.Definition.Name)

	var err error
	builder.Object, err = builder.apiClient.MachineConfigs().Update(
		context.TODO(), builder.Definition, metav1.UpdateOptions{})

	return builder, err
}

// Exists checks whether the given machineconfig exists.
func (builder *MCBuilder) Exists() bool {
	if valid, _ := builder.validate(); !valid {
		return false
	}

	glog.V(100).Infof("Checking if the MachineConfig object %s exists", builder.Definition.Name)

	var err error
	builder.Object, err = builder.apiClient.MachineConfigs().Get(
		context.TODO(), builder.Definition.Name, metav1.GetOptions{})

	return err == nil || !k8serrors.IsNotFound(err)
}

// WithLabel redefines machineconfig definition with the given label.
func (builder *MCBuilder) WithLabel(key, value string) *MCBuilder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof("Labeling the machineconfig %s with %s=%s", builder.Definition.Name, key, value)

	if key == "" {
		glog.V(100).Infof("The key cannot be empty")

		builder.errorMsg = "'key' cannot be empty"

		return builder
	}

	if builder.Definition.Labels == nil {
		builder.Definition.Labels = map[string]string{}
	}

	builder.Definition.Labels[key] = value

	return builder
}

// WithOptions creates the machineconfig with generic mutation options.
func (builder *MCBuilder) WithOptions(options ...MCAdditionalOptions) *MCBuilder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof("Setting machineconfig additional options")

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

// WithKernelArguments sets the specified KernelArguments to the MachineConfig.
func (builder *MCBuilder) WithKernelArguments(kernelArgs []string) *MCBuilder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	if len(kernelArgs) == 0 {
		glog.V(100).Infof("The kernelArgs cannot be empty")

		builder.errorMsg = "'kernelArgs' cannot be empty"

		return builder
	}

	glog.V(100).Infof("Setting KernelArguments: %v", kernelArgs)

	builder.Definition.Spec.KernelArguments = kernelArgs

	return builder
}

// WithExtensions sets the specified Extensions to the MachineConfig.
func (builder *MCBuilder) WithExtensions(extensions []string) *MCBuilder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	if len(extensions) == 0 {
		glog.V(100).Infof("The extensions cannot be empty")

		builder.errorMsg = "'extensions' cannot be empty"

		return builder
	}

	glog.V(100).Infof("Setting Extensions: %v", extensions)

	builder.Definition.Spec.Extensions = extensions

	return builder
}

// WithFIPS sets the specified FIPS value to the MachineConfig.
func (builder *MCBuilder) WithFIPS(fips bool) *MCBuilder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof("Setting FIPS: %v", fips)

	builder.Definition.Spec.FIPS = fips

	return builder
}

// WithKernelType sets the specified kernelType to the MachineConfig.
func (builder *MCBuilder) WithKernelType(kernelType string) *MCBuilder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	if kernelType == "" {
		glog.V(100).Infof("The kernelType cannot be empty")

		builder.errorMsg = "'kernelType' cannot be empty"

		return builder
	}

	glog.V(100).Infof("Setting KernelType: %v", kernelType)

	builder.Definition.Spec.KernelType = kernelType

	return builder
}

func (builder *MCBuilder) validate() (bool, error) {
	resourceCRD := "MachineConfig"

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
