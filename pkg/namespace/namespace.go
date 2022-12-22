package namespace

import (
	"context"
	"fmt"
	"time"

	"github.com/openshift-kni/eco-gotests/pkg/clients"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

// Builder provides struct for namespace object containing connection to the cluster and the namespace definitions.
type Builder struct {
	// Namespace definition. Used to create namespace object.
	Definition *v1.Namespace
	// Created namespace object
	Object *v1.Namespace
	// Used in functions that define or mutate namespace definition. errorMsg is processed before the namespace
	// object is created
	errorMsg  string
	apiClient *clients.Settings
}

// NewBuilder creates new instance of Builder.
func NewBuilder(apiClient *clients.Settings, name string) *Builder {
	builder := Builder{
		apiClient: apiClient,
		Definition: &v1.Namespace{
			ObjectMeta: metaV1.ObjectMeta{
				Name: name,
			},
		},
	}

	if name == "" {
		builder.errorMsg = "namespace 'name' cannot be empty"
	}

	return &builder
}

// WithLabel redefines namespace definition with the given label.
func (builder *Builder) WithLabel(key string, value string) *Builder {
	if builder.errorMsg != "" {
		return builder
	}

	if key == "" {
		builder.errorMsg = "'key' cannot be empty"

		return builder
	}

	// Make sure NewBuilder was already called to set builder.Definition.
	if builder.Definition == nil {
		builder.errorMsg = "can not redefine undefined namespace"

		return builder
	}

	if builder.Definition.Labels == nil {
		builder.Definition.Labels = map[string]string{}
	}

	builder.Definition.Labels[key] = value

	return builder
}

// WithMultipleLabels redefines namespace definition with the given labels.
func (builder *Builder) WithMultipleLabels(labels map[string]string) *Builder {
	for k, v := range labels {
		builder.WithLabel(k, v)
	}

	return builder
}

// Create makes a namespace in the cluster and stores the created object in struct.
func (builder *Builder) Create() (*Builder, error) {
	if builder.errorMsg != "" {
		return nil, fmt.Errorf(builder.errorMsg)
	}

	var err error
	if !builder.Exists() {
		builder.Object, err = builder.apiClient.Namespaces().Create(
			context.TODO(), builder.Definition, metaV1.CreateOptions{})
	}

	return builder, err
}

// Update renovates the existing namespace object with the namespace definition in builder.
func (builder *Builder) Update() (*Builder, error) {
	if builder.errorMsg != "" {
		return nil, fmt.Errorf(builder.errorMsg)
	}

	var err error
	builder.Object, err = builder.apiClient.Namespaces().Update(
		context.TODO(), builder.Definition, metaV1.UpdateOptions{})

	return builder, err
}

// Delete removes a namespace.
func (builder *Builder) Delete() error {
	if !builder.Exists() {
		return nil
	}

	err := builder.apiClient.Namespaces().Delete(context.TODO(), builder.Object.Name, metaV1.DeleteOptions{})

	if err != nil {
		return err
	}

	builder.Object = nil

	return err
}

// DeleteAndWait deletes a namespace and waits until it's removed from the cluster.
func (builder *Builder) DeleteAndWait(timeout time.Duration) error {
	if err := builder.Delete(); err != nil {
		return err
	}

	return wait.PollImmediate(time.Second, timeout, func() (bool, error) {
		_, err := builder.apiClient.Namespaces().Get(context.Background(), builder.Definition.Name, metaV1.GetOptions{})
		if k8serrors.IsNotFound(err) {

			return true, nil
		}

		return false, nil
	})
}

// Exists checks whether the given namespace exists.
func (builder *Builder) Exists() bool {
	var err error
	builder.Object, err = builder.apiClient.Namespaces().Get(
		context.Background(), builder.Definition.Name, metaV1.GetOptions{})

	return err == nil || !k8serrors.IsNotFound(err)
}
