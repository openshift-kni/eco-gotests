package configmap

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-gotests/pkg/clients"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Builder provides struct for configmap object containing connection to the cluster and the configmap definitions.
type Builder struct {
	// ConfigMap definition. Used to create configmap object.
	Definition *v1.ConfigMap
	// Created configmap object.
	Object *v1.ConfigMap
	// Used in functions that defines or mutates configmap definition. errorMsg is processed before the configmap
	// object is created.
	errorMsg  string
	apiClient *clients.Settings
}

// NewBuilder creates a new instance of Builder.
func NewBuilder(apiClient *clients.Settings, name, nsname string) *Builder {
	glog.V(100).Infof(
		"Initializing new configmap structure with the following params: %s, %s", name, nsname)

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
		glog.V(100).Infof("The name of the configmap is empty")

		builder.errorMsg = "configmap 'name' cannot be empty"
	}

	if nsname == "" {
		glog.V(100).Infof("The namespace of the configmap is empty")

		builder.errorMsg = "configmap 'nsname' cannot be empty"
	}

	return &builder
}

// Create makes a configmap in cluster and stores the created object in struct.
func (builder *Builder) Create() (*Builder, error) {
	glog.V(100).Infof("Creating the configmap %s in namespace %s", builder.Definition.Name, builder.Definition.Namespace)

	if builder.errorMsg != "" {
		return nil, fmt.Errorf(builder.errorMsg)
	}

	var err error
	if !builder.Exists() {
		builder.Object, err = builder.apiClient.ConfigMaps(builder.Definition.Namespace).Create(
			context.TODO(), builder.Definition, metaV1.CreateOptions{})
	}

	return builder, err
}

// Delete removes a configmap.
func (builder *Builder) Delete() error {
	glog.V(100).Infof("Deleting the configmap %s from namespace %s", builder.Definition.Name, builder.Definition.Namespace)

	if !builder.Exists() {
		return nil
	}

	err := builder.apiClient.ConfigMaps(builder.Definition.Namespace).Delete(
		context.TODO(), builder.Object.Name, metaV1.DeleteOptions{})

	if err != nil {
		return err
	}

	builder.Object = nil

	return err
}

// Exists checks whether the given configmap exists.
func (builder *Builder) Exists() bool {
	glog.V(100).Infof(
		"Checking if configmap %s exists in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	builder.Object, err = builder.apiClient.ConfigMaps(builder.Definition.Namespace).Get(
		context.Background(), builder.Definition.Name, metaV1.GetOptions{})

	return err == nil || !k8serrors.IsNotFound(err)
}

// WithData defines the data placed in the configmap.
func (builder *Builder) WithData(data map[string]string) *Builder {
	glog.V(100).Infof(
		"Creating configmap %s in namespace %s with this data: %s",
		builder.Definition.Name, builder.Definition.Namespace, data)

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
