package sriovfec

import (
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/msg"
	sriovfectypes "github.com/openshift-kni/eco-goinfra/pkg/schemes/fec/fectypes"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	sriovfecnodeconfig = "SriovFecNodeConfig"
)

// NodeConfigBuilder provides struct for the SriovFecNodeConfig object containing connection to
// the cluster and the SriovFecNodeConfig definitions.
type NodeConfigBuilder struct {
	// SriovFecNodeConfig definition. Used to create SriovFecNodeConfig object.
	Definition *sriovfectypes.SriovFecNodeConfig
	// Create SriovFecNodeConfig object.
	Object *sriovfectypes.SriovFecNodeConfig
	// apiClient opens a connection to the cluster.
	apiClient *clients.Settings
	// Used in functions that define SriovFecNodeConfig definitions. errorMsg is processed before SriovFecNodeConfig
	// object is created.
	errorMsg string
}

// AdditionalOptions additional options for sriovfecnodeconfig object.
type AdditionalOptions func(builder *NodeConfigBuilder) (*NodeConfigBuilder, error)

// NewNodeConfigBuilder creates a new instance of NodeConfigBuilder.
func NewNodeConfigBuilder(
	apiClient *clients.Settings,
	name, nsname string,
	label map[string]string) *NodeConfigBuilder {
	glog.V(100).Infof(
		"Initializing new SriovFecNodeConfig structure with the following params: %s, %s, %v",
		name, nsname, label)

	builder := NodeConfigBuilder{
		apiClient: apiClient,
		Definition: &sriovfectypes.SriovFecNodeConfig{
			TypeMeta: metaV1.TypeMeta{
				Kind:       sriovfecnodeconfig,
				APIVersion: fmt.Sprintf("%s/%s", APIGroup, APIVersion),
			},
			ObjectMeta: metaV1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the SriovFecNodeConfig is empty")

		builder.errorMsg = "SriovFecNodeConfig 'name' cannot be empty"
	}

	if nsname == "" {
		glog.V(100).Infof("The namespace of the SriovFecNodeConfig is empty")

		builder.errorMsg = "SriovFecNodeConfig 'nsname' cannot be empty"
	}

	return &builder
}

// Pull retrieves an existing SriovFecNodeConfig.io object from the cluster.
func Pull(apiClient *clients.Settings, name, nsname string) (*NodeConfigBuilder, error) {
	glog.V(100).Infof(
		"Pulling SriovFecNodeConfig.io object name: %s in namespace: %s", name, nsname)

	builder := NodeConfigBuilder{
		apiClient: apiClient,
		Definition: &sriovfectypes.SriovFecNodeConfig{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
		},
	}

	if name == "" {
		return nil, fmt.Errorf("the name of the SriovFecNodeConfig is empty")
	}

	if nsname == "" {
		return nil, fmt.Errorf("the namespace of the SriovFecNodeConfig is empty")
	}

	if !builder.Exists() {
		return nil, fmt.Errorf("SriovFecNodeConfig object %s does not exist in namespace %s", name, nsname)
	}

	builder.Definition = builder.Object

	return &builder, nil
}

// Exists checks whether the given SriovFecNodeConfig exists.
func (builder *NodeConfigBuilder) Exists() bool {
	if valid, _ := builder.validate(); !valid {
		return false
	}

	glog.V(100).Infof(
		"Checking if SriovFecNodeConfig %s exists in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	builder.Object, err = builder.Get()

	if err != nil {
		glog.V(100).Infof("Failed to collect SriovFecNodeConfig object due to %s", err.Error())
	}

	return err == nil || !k8serrors.IsNotFound(err)
}

// Create makes a SriovFecNodeConfig in the cluster and stores the created object in struct.
func (builder *NodeConfigBuilder) Create() (*NodeConfigBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Creating the sriovfecnodeconfig %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace,
	)

	var err error
	if !builder.Exists() {
		unstructuredSriovFecNodeConfig, err := runtime.DefaultUnstructuredConverter.ToUnstructured(builder.Definition)

		if err != nil {
			glog.V(100).Infof("Failed to convert structured SriovFecNodeConfig to unstructured object")

			return nil, err
		}

		unsObject, err := builder.apiClient.Resource(
			GetSriovFecNodeConfigIoGVR()).Namespace(builder.Definition.Namespace).Create(
			context.TODO(), &unstructured.Unstructured{Object: unstructuredSriovFecNodeConfig}, metaV1.CreateOptions{})

		if err != nil {
			glog.V(100).Infof("Failed to create SriovFecNodeConfig")

			return nil, err
		}

		builder.Object, err = builder.convertToStructured(unsObject)

		if err != nil {
			return nil, err
		}
	}

	return builder, err
}

// Get returns SriovFecNodeConfig object if found.
func (builder *NodeConfigBuilder) Get() (*sriovfectypes.SriovFecNodeConfig, error) {
	if valid, err := builder.validate(); !valid {
		return nil, err
	}

	glog.V(100).Infof(
		"Collecting SriovFecNodeConfig object %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	unsObject, err := builder.apiClient.Resource(GetSriovFecNodeConfigIoGVR()).Namespace(builder.Definition.Namespace).Get(
		context.TODO(), builder.Definition.Name, metaV1.GetOptions{})

	if err != nil {
		glog.V(100).Infof(
			"SriovFecNodeConfig object %s does not exist in namespace %s",
			builder.Definition.Name, builder.Definition.Namespace)

		return nil, err
	}

	return builder.convertToStructured(unsObject)
}

// Delete removes SriovFecNodeConfig object from a cluster.
func (builder *NodeConfigBuilder) Delete() (*NodeConfigBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Deleting the SriovFecNodeConfig object %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace,
	)

	if !builder.Exists() {
		glog.V(100).Infof("SriovFecNodeConfig %s in namespace %s cannot be deleted because it does not exist",
			builder.Definition.Name, builder.Definition.Namespace)

		builder.Object = nil

		return builder, nil
	}

	err := builder.apiClient.Resource(
		GetSriovFecNodeConfigIoGVR()).Namespace(builder.Definition.Namespace).Delete(
		context.TODO(), builder.Definition.Name, metaV1.DeleteOptions{})

	if err != nil {
		return builder, fmt.Errorf("can not delete SriovFecNodeConfig: %w", err)
	}

	builder.Object = nil

	return builder, nil
}

// Update renovates the existing SriovFecNodeConfig object with the SriovFecNodeConfig definition in builder.
func (builder *NodeConfigBuilder) Update(force bool) (*NodeConfigBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	if !builder.Exists() {
		return nil, fmt.Errorf("failed to update SriovFecNodeConfig, object does not exist on cluster")
	}

	glog.V(100).Infof("Updating the SriovFecNodeConfig object %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace,
	)

	if builder.errorMsg != "" {
		return nil, fmt.Errorf(builder.errorMsg)
	}

	builder.Definition.ResourceVersion = builder.Object.ResourceVersion
	builder.Definition.ObjectMeta.ResourceVersion = builder.Object.ObjectMeta.ResourceVersion

	unstructuredSriovFecNodeConfig, err := runtime.DefaultUnstructuredConverter.ToUnstructured(builder.Definition)
	if err != nil {
		glog.V(100).Infof("Failed to convert structured SriovFecNodeConfig to unstructured object")

		return nil, err
	}

	_, err = builder.apiClient.Resource(
		GetSriovFecNodeConfigIoGVR()).Namespace(builder.Definition.Namespace).Update(
		context.TODO(), &unstructured.Unstructured{Object: unstructuredSriovFecNodeConfig}, metaV1.UpdateOptions{})

	if err != nil {
		if force {
			glog.V(100).Infof(
				msg.FailToUpdateNotification("SriovFecNodeConfig", builder.Definition.Name, builder.Definition.Namespace))

			builder, err := builder.Delete()

			if err != nil {
				glog.V(100).Infof(
					msg.FailToUpdateError("SriovFecNodeConfig", builder.Definition.Name, builder.Definition.Namespace))

				return nil, err
			}

			return builder.Create()
		}
	}

	return builder, err
}

// WithOptions creates SriovFecNodeConfig with generic mutation options.
func (builder *NodeConfigBuilder) WithOptions(options ...AdditionalOptions) *NodeConfigBuilder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof("Setting SriovFecNodeConfig additional options")

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

// GetSriovFecNodeConfigIoGVR returns SriovFecNodeConfig's GroupVersionResource which could be used for Clean function.
func GetSriovFecNodeConfigIoGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group: APIGroup, Version: APIVersion, Resource: "sriovfecnodeconfigs",
	}
}

// validate will check that the builder and builder definition are properly initialized before
// accessing any member fields.
func (builder *NodeConfigBuilder) validate() (bool, error) {
	resourceCRD := "SriovFecNodeConfig"

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

func (builder *NodeConfigBuilder) convertToStructured(unsObject *unstructured.Unstructured) (
	*sriovfectypes.SriovFecNodeConfig, error) {
	SriovFecNodeConfig := &sriovfectypes.SriovFecNodeConfig{}

	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unsObject.Object, SriovFecNodeConfig)
	if err != nil {
		glog.V(100).Infof(
			"Failed to convert from unstructured to SriovFecNodeConfig object in namespace %s",
			builder.Definition.Name, builder.Definition.Namespace)

		return nil, err
	}

	return SriovFecNodeConfig, err
}
