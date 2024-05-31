package kmm

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-gotests/pkg/clients"
	"github.com/openshift-kni/eco-gotests/pkg/msg"
	moduleV1Beta1 "github.com/rh-ecosystem-edge/kernel-module-management/api/v1beta1"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	goclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ModuleBuilder provides struct for the module object containing connection to
// the cluster and the module definitions.
type ModuleBuilder struct {
	// Module definition. Used to create a Module object.
	Definition *moduleV1Beta1.Module
	// Created Module object.
	Object *moduleV1Beta1.Module
	// Used in functions that define or mutate Module definition. errorMsg is processed before the Module
	// object is created.
	apiClient *clients.Settings
	errorMsg  string
}

// NewModuleBuilder creates a new instance of ModuleBuilder.
func NewModuleBuilder(
	apiClient *clients.Settings, name, nsname string) *ModuleBuilder {
	glog.V(100).Infof(
		"Initializing new Module structure with following params: %s, %s", name, nsname)

	builder := ModuleBuilder{
		apiClient: apiClient,
		Definition: &moduleV1Beta1.Module{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the Module is empty")

		builder.errorMsg = "Module 'name' cannot be empty"
	}

	if nsname == "" {
		glog.V(100).Infof("The namespace of the module is empty")

		builder.errorMsg = "Module 'namespace' cannot be empty"
	}

	return &builder
}

// WithNodeSelector adds the specified NodeSelector to the Module.
func (builder *ModuleBuilder) WithNodeSelector(nodeSelector map[string]string) *ModuleBuilder {
	glog.V(100).Infof(
		"Creating Module %s in namespace %s with this nodeSelector: %s",
		builder.Definition.Name, builder.Definition.Namespace, nodeSelector)

	if builder.Definition == nil {
		builder.errorMsg = msg.UndefinedCrdObjectErrString("Module")
	}

	if len(nodeSelector) == 0 {
		glog.V(100).Infof("Can not redefine Module with empty nodeSelector map")

		builder.errorMsg = "Module 'nodeSelector' cannot be empty map"
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.Selector = nodeSelector

	return builder
}

// WithLoadServiceAccount adds the specified Load ServiceAccount to the Module.
func (builder *ModuleBuilder) WithLoadServiceAccount(srvAccountName string) *ModuleBuilder {
	glog.V(100).Infof(
		"Creating Module %s in namespace %s with ModuleLoad ServiceAccount: %s",
		builder.Definition.Name, builder.Definition.Namespace, srvAccountName)

	return builder.withServiceAccount(srvAccountName, "module")
}

// WithDevicePluginServiceAccount adds the specified Device Plugin ServiceAccount to the Module.
func (builder *ModuleBuilder) WithDevicePluginServiceAccount(srvAccountName string) *ModuleBuilder {
	glog.V(100).Infof(
		"Creating Module %s in namespace %s with DevicePlugin ServiceAccount: %s",
		builder.Definition.Name, builder.Definition.Namespace, srvAccountName)

	return builder.withServiceAccount(srvAccountName, "device")
}

// WithImageRepoSecret adds the specific ImageRepoSecret to the Module.
func (builder *ModuleBuilder) WithImageRepoSecret(imageRepoSecret string) *ModuleBuilder {
	if imageRepoSecret == "" {
		builder.errorMsg = "can not redefine module with empty imageRepoSecret"
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.ImageRepoSecret.Name = imageRepoSecret

	return builder
}

// WithDevicePluginVolume adds the specified DevicePlugin volume to the Module.
func (builder *ModuleBuilder) WithDevicePluginVolume(name string, configMapName string) *ModuleBuilder {
	if name == "" {
		builder.errorMsg = "cannot redefine with empty volume 'name'"
	}

	if configMapName == "" {
		builder.errorMsg = "cannot redefine with empty 'configMapName'"
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.DevicePlugin.Volumes = append(builder.Definition.Spec.DevicePlugin.Volumes, v1.Volume{
		Name: name,
		VolumeSource: v1.VolumeSource{
			ConfigMap: &v1.ConfigMapVolumeSource{
				LocalObjectReference: v1.LocalObjectReference{
					Name: configMapName},
			},
		},
	})

	return builder
}

// WithModuleLoaderContainer adds the specified ModuleLoader container to the Module.
func (builder *ModuleBuilder) WithModuleLoaderContainer(
	container *moduleV1Beta1.ModuleLoaderContainerSpec) *ModuleBuilder {
	if builder.Definition == nil {
		builder.errorMsg = msg.UndefinedCrdObjectErrString("Module")
	}

	if container == nil {
		builder.errorMsg = "invalid 'container' argument can not be nil"
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.ModuleLoader.Container = *container

	return builder
}

// WithDevicePluginContainer adds the specified DevicePlugin container to the Module.
func (builder *ModuleBuilder) WithDevicePluginContainer(
	container *moduleV1Beta1.DevicePluginContainerSpec) *ModuleBuilder {
	if builder.Definition == nil {
		builder.errorMsg = msg.UndefinedCrdObjectErrString("Module")
	}

	if container == nil {
		builder.errorMsg = "invalid 'container' argument can not be nil"
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.DevicePlugin.Container = *container

	return builder
}

// Create builds module in the cluster and stores object in struct.
func (builder *ModuleBuilder) Create() (*ModuleBuilder, error) {
	glog.V(100).Infof("Creating module %s in namespace %s",
		builder.Definition.Name,
		builder.Definition.Namespace)

	if builder.errorMsg != "" {
		return nil, fmt.Errorf(builder.errorMsg)
	}

	var err error
	if !builder.Exists() {
		err = builder.apiClient.Create(context.TODO(), builder.Definition)
	}

	return builder, err
}

// Exists checks whether the given module exists.
func (builder *ModuleBuilder) Exists() bool {
	glog.V(100).Infof("Checking if module %s exists in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	builder.Object, err = builder.Get()

	return err == nil || !k8serrors.IsNotFound(err)
}

// Delete removes the module.
func (builder *ModuleBuilder) Delete() (*ModuleBuilder, error) {
	glog.V(100).Infof("Deleting module %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	if !builder.Exists() {
		return builder, fmt.Errorf("module cannot be deleted because it does not exist")
	}

	err := builder.apiClient.Delete(context.TODO(), builder.Definition)

	if err != nil {
		return builder, err
	}

	builder.Object = nil

	return builder, err
}

// Get fetches the defined module from the cluster.
func (builder *ModuleBuilder) Get() (*moduleV1Beta1.Module, error) {
	glog.V(100).Infof("Getting module %s from namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	module := &moduleV1Beta1.Module{}

	err := builder.apiClient.Get(context.TODO(), goclient.ObjectKey{
		Name:      builder.Definition.Name,
		Namespace: builder.Definition.Namespace,
	}, module)

	if err != nil {
		return nil, err
	}

	return module, err
}

func (builder *ModuleBuilder) withServiceAccount(srvAccountName string, accountType string) *ModuleBuilder {
	if builder.Definition == nil {
		builder.errorMsg = msg.UndefinedCrdObjectErrString("Module")
	}

	if srvAccountName == "" {
		builder.errorMsg = "can not redefine module with empty ServiceAccount"
	}

	if builder.errorMsg != "" {
		return builder
	}

	switch accountType {
	case "module":
		builder.Definition.Spec.ModuleLoader.ServiceAccountName = srvAccountName
	case "device":
		builder.Definition.Spec.DevicePlugin.ServiceAccountName = srvAccountName
	default:
		builder.errorMsg = "invalid account type parameter. Supported parameters are: 'module', 'device'"
	}

	return builder
}
