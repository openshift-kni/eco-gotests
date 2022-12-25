package nad

import (
	nadV1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	"github.com/openshift-kni/eco-gotests/pkg/clients"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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

// NewBuilder creates new instance of NAD Builder.
// arguments:       "apiClient" -       the nad network client.
//                  "name"      -       the name of the nad network.
//                  "nsname"    -       the nad network namespace.
// return value:    the created Builder.
func NewBuilder(apiClient *clients.Settings, name, nsname string) *Builder {
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
		builder.errorMsg = "NAD name is empty"
	}

	if builder.Definition.Namespace == "" {
		builder.errorMsg = "NAD namespace is empty"
	}

	return &builder
}

// Create creates NAD resource with the builder configuration.
//  if the creation failed, the builder errorMsg will be updated.
// return value:    the builder itself with the NAD object if the creation succeeded.
//                  an error if any occurred.
func (builder *Builder) Create() (*Builder, error) {
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

	return builder, fmt.Errorf(builder.errorMsg)
}

// Delete removes NetworkAttachmentDefinition resource with the builder definition.
// (If NAD doesn't exist, nothing is done) and a nil error is returned.
// return value:    an error if any occurred.
func (builder *Builder) Delete() error {
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
//                  false   - NAD doesn't exist.
func (builder *Builder) Exists() bool {
	_, err := builder.apiClient.NetworkAttachmentDefinitions(builder.Definition.Namespace).Get(context.Background(),
		builder.Definition.Name, metaV1.GetOptions{})

	return nil == err || !k8serrors.IsNotFound(err)
}

// GetString prints NetworkAttachmentDefinition resource.
// return value:    the builder details in json string format, and an error if any occurred.
func (builder *Builder) GetString() (string, error) {
	nadByte, err := json.MarshalIndent(builder.Definition, "", "    ")
	if err != nil {
		return "", err
	}

	return string(nadByte), err
}

// fillConfigureString adds a configuration string to builder definition specs configuration if needed.
// return value:    an error if any occurred.
func (builder *Builder) fillConfigureString() error {
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
