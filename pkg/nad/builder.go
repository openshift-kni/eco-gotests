package nad

import (
	"github.com/golang/glog"
	nadV1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	"github.com/openshift-kni/eco-gotests/pkg/clients"
	"github.com/openshift-kni/eco-gotests/pkg/msg"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"context"
	"encoding/json"
	"fmt"
)

// Builder provides struct for NAD object which contains connection to cluster and the NAD object itself.
type Builder struct {
	Definition        *nadV1.NetworkAttachmentDefinition
	Object            *nadV1.NetworkAttachmentDefinition
	metaPluginConfigs []Plugin
	apiClient         *clients.Settings
	errorMsg          string
}

// NewBuilder creates a new instance of NetworkAttachmentDefinition Builder.
// arguments:       "apiClient" -       the nad network client.
//
//	"name"      -       the name of the nad network.
//	"nsname"    -       the nad network namespace.
//
// return value:    the created Builder.
func NewBuilder(apiClient *clients.Settings, name, nsname string) *Builder {
	glog.V(100).Infof(
		"Initializing new NetworkAttachmentDefinition structure with the following params: "+
			"name: %s, namespace: %s",
		name, nsname)

	builder := Builder{
		apiClient: apiClient,
		Definition: &nadV1.NetworkAttachmentDefinition{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
		},
	}

	if builder.Definition.Name == "" {
		glog.V(100).Infof("The name of the NetworkAttachmentDefinition is empty")

		builder.errorMsg = "NAD name is empty"
	}

	if builder.Definition.Namespace == "" {
		glog.V(100).Infof("The namespace of the NetworkAttachmentDefinition is empty")

		builder.errorMsg = "NAD namespace is empty"
	}

	return &builder
}

// Create builds a NetworkAttachmentDefinition resource with the builder configuration.
//
//	if the creation failed, the builder errorMsg will be updated.
//
// return value:    the builder itself with the NAD object if the creation succeeded.
//
//	an error if any occurred.
func (builder *Builder) Create() (*Builder, error) {
	glog.V(100).Infof("Creating NetworkAttachmentDefinition %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	if builder.errorMsg != "" {
		return nil, fmt.Errorf(builder.errorMsg)
	}

	err := builder.fillConfigureString()

	if err != nil {
		return builder, fmt.Errorf("failed create NAD object, could not marshal configuration " + err.Error())
	}

	if !builder.Exists() {
		builder.Object, err = builder.apiClient.NetworkAttachmentDefinitions(builder.Definition.Namespace).
			Create(context.Background(), builder.Definition, metaV1.CreateOptions{})
		if err != nil {
			return builder, fmt.Errorf("fail to create NAD object due to: " + err.Error())
		}
	}

	return builder, nil
}

// Delete removes NetworkAttachmentDefinition resource with the builder definition.
// (If NAD doesn't exist, nothing is done) and a nil error is returned.
// return value:    an error if any occurred.
func (builder *Builder) Delete() error {
	glog.V(100).Infof("Deleting NetworkAttachmentDefinition %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	if !builder.Exists() {
		return nil
	}

	err := builder.apiClient.NetworkAttachmentDefinitions(builder.Definition.Namespace).Delete(
		context.Background(), builder.Definition.Namespace, metaV1.DeleteOptions{})

	if err != nil {
		return fmt.Errorf("fail to delete NAD object due to: %w", err)
	}

	builder.Object = nil

	return nil
}

// Exists checks if a NAD is exists in the builder.
// return value:    true    - NAD exists.
//
//	false   - NAD doesn't exist.
func (builder *Builder) Exists() bool {
	glog.V(100).Infof("Checking if NetworkAttachmentDefinition %s exists in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	_, err := builder.apiClient.NetworkAttachmentDefinitions(builder.Definition.Namespace).Get(context.Background(),
		builder.Definition.Name, metaV1.GetOptions{})

	return nil == err || !k8serrors.IsNotFound(err)
}

// GetString prints NetworkAttachmentDefinition resource.
// return value:    the builder details in json string format, and an error if any occurred.
func (builder *Builder) GetString() (string, error) {
	glog.V(100).Infof("Returning NetworkAttachmentDefinition resource in json format")

	nadByte, err := json.MarshalIndent(builder.Definition, "", "    ")
	if err != nil {
		return "", err
	}

	return string(nadByte), err
}

// fillConfigureString adds a configuration string to builder definition specs configuration if needed.
// return value:    an error if any occurred.
func (builder *Builder) fillConfigureString() error {
	glog.V(100).Infof("Adding configuration to NetworkAttachmentDefinition builder if needed")

	if builder.metaPluginConfigs == nil {
		return nil
	}

	nadConfig := &MasterPlugin{
		CniVersion: "0.4.0",
		Name:       builder.Definition.Name,
		Plugins:    &builder.metaPluginConfigs,
	}

	var nadConfigJSONString []byte

	nadConfigJSONString, err := json.Marshal(nadConfig)
	if err != nil {
		return err
	}

	if string(nadConfigJSONString) != "" {
		builder.Definition.Spec.Config = string(nadConfigJSONString)
	}

	return nil
}

// WithMasterPlugin defines master plugin configuration in the NetworkAttachmentDefinition spec.
func (builder *Builder) WithMasterPlugin(masterPlugin *MasterPlugin) *Builder {
	glog.V(100).Infof("Adding masterPlugin %v to NAD %s", masterPlugin, builder.Definition.Name)

	if builder.Definition == nil {
		builder.errorMsg = msg.UndefinedCrdObjectErrString("NAD")
	}

	emptyNadConfig := nadV1.NetworkAttachmentDefinitionSpec{}

	if builder.Definition.Spec != emptyNadConfig {
		builder.errorMsg = "error to redefine predefine NAD"
	}

	masterPluginSting, err := json.Marshal(masterPlugin)

	if err != nil {
		builder.errorMsg = err.Error()
	}

	builder.Definition.Spec.Config = string(masterPluginSting)

	return builder
}

// GetGVR returns nad's GroupVersionResource which could be used for Clean function.
func GetGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group: "k8s.cni.cncf.io", Version: "v1", Resource: "network-attachment-definitions",
	}
}
