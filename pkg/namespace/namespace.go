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

// Builder provides struct for namespace object which contains connection to cluster and namespace definition.
type Builder struct {
	// Namespace definition. Used to create namespace object.
	Definition *v1.Namespace
	// Created namespace object
	Object *v1.Namespace
	// Used in functions that defines or mutates namespace definition. errorMsg is processed before namespace
	// object is created
	errorMsg  string
	apiClient *clients.Settings
}

// NewBuilder creates new instance of Builder.
func NewBuilder(name string, apiClient *clients.Settings) *Builder {
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

// WithLabel redefines namespace definition with given label.
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

// WithMultipleLabels redefines namespace definition with given labels.
func (builder *Builder) WithMultipleLabels(labels map[string]string) *Builder {
	for k, v := range labels {
		builder.WithLabel(k, v)
	}

	return builder
}

// Create creates namespace on cluster and stores created object in struct.
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

// Update updates existing namespace object with namespace definition in builder.
func (builder *Builder) Update() (*Builder, error) {
	if builder.errorMsg != "" {
		return nil, fmt.Errorf(builder.errorMsg)
	}

	var err error
	builder.Object, err = builder.apiClient.Namespaces().Update(
		context.TODO(), builder.Definition, metaV1.UpdateOptions{})

	return builder, err
}

// Delete deletes namespace.
func (builder *Builder) Delete() error {
	if !builder.Exists() {
		return nil
	}

	return builder.apiClient.Namespaces().Delete(context.TODO(), builder.Object.Name, metaV1.DeleteOptions{})
}

// DeleteAndWait deletes and waits until namespace is removed from the cluster.
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

// Exists tells whether the given namespace exists.
func (builder *Builder) Exists() bool {
	var err error
	builder.Object, err = builder.apiClient.Namespaces().Get(
		context.Background(), builder.Definition.Name, metaV1.GetOptions{})

	return err == nil || !k8serrors.IsNotFound(err)
}
