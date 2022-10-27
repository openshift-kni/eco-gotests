package configmap

import (
	"context"
	"fmt"

	"github.com/openshift-kni/eco-gotests/pkg/clients"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Builder provides struct for configmap object which contains connection to cluster and configmap definition.
type Builder struct {
	// configMap definition. Used to create configmap object.
	Definition *v1.ConfigMap
	// Created configmap object
	Object *v1.ConfigMap
	// Used in functions that defines or mutates configmap definition. errorMsg is processed before configmap
	// object is created
	errorMsg  string
	apiClient *clients.Settings
}

// NewBuilder creates new instance of Builder.
func NewBuilder(name, nsname string, apiClient *clients.Settings) *Builder {
	builder := Builder{
		apiClient: apiClient,
		Definition: &v1.ConfigMap{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
		},
	}

	if name == "" {
		builder.errorMsg = "configmap 'name' cannot be empty"
	}
	if nsname == "" {
		builder.errorMsg = "configmap 'nsname' cannot be empty"
	}

	return &builder
}

// Create creates configmap on cluster and stores created object in struct.
func (builder *Builder) Create() (*Builder, error) {
	if builder.errorMsg != "" {
		return nil, fmt.Errorf(builder.errorMsg)
	}

	var err error
	if !builder.Exists() {
		builder.Object, err = builder.apiClient.ConfigMaps(builder.Definition.Namespace).Create(
			context.TODO(), builder.Definition, metaV1.CreateOptions{})

		return builder, err
	}

	return builder, err
}

// Delete deletes configmap.
func (builder *Builder) Delete() error {
	if !builder.Exists() {
		return nil
	}

	return builder.apiClient.ConfigMaps(builder.Definition.Namespace).Delete(context.TODO(), builder.Object.Name, metaV1.DeleteOptions{})
}

// Exists tells whether the given configmap exists.
func (builder *Builder) Exists() bool {
	var err error
	builder.Object, err = builder.apiClient.ConfigMaps(builder.Definition.Namespace).Get(
		context.Background(), builder.Definition.Name, metaV1.GetOptions{})

	return err == nil || !k8serrors.IsNotFound(err)
}

// WithData defines the data placed in the configmap.
func (builder *Builder) WithData(data map[string]string) *Builder {
	if builder.errorMsg != "" {
		return builder
	}

	if len(data) == 0 {
		builder.errorMsg = "'data' cannot be empty"

		return builder
	}

	// Make sure NewBuilder was already called to set builder.Definition.
	if builder.Definition == nil {
		builder.errorMsg = "can not redefine undefined configmap"

		return builder
	}

	builder.Definition.Data = data

	return builder
}
