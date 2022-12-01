package secret

import (
	"context"
	"fmt"

	"github.com/openshift-kni/eco-gotests/pkg/clients"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Builder provides struct for secret object which contains connection to cluster and secret definition.
type Builder struct {
	// secret definition. Used to store secret object.
	Definition *v1.Secret
	// Created secret object.
	Object *v1.Secret
	// Used in functions that defines or mutates secret definition. errorMsg is processed before secret
	// object is created.
	errorMsg  string
	apiClient *clients.Settings
}

// NewBuilder creates new instance of Builder.
func NewBuilder(apiClient *clients.Settings, name, nsname string, secretType v1.SecretType) *Builder {
	builder := Builder{
		apiClient: apiClient,
		Definition: &v1.Secret{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
			Type: secretType,
		},
	}

	if name == "" {
		builder.errorMsg = "secret 'name' cannot be empty"
	}

	if nsname == "" {
		builder.errorMsg = "secret 'nsname' cannot be empty"
	}

	return &builder
}

// Create creates secret on cluster and stores created object in struct.
func (builder *Builder) Create() (*Builder, error) {
	if builder.errorMsg != "" {
		return nil, fmt.Errorf(builder.errorMsg)
	}

	var err error
	if !builder.Exists() {
		builder.Object, err = builder.apiClient.Secrets(builder.Definition.Namespace).Create(
			context.TODO(), builder.Definition, metaV1.CreateOptions{})
	}

	return builder, err
}

// Delete removes secret from a cluster.
func (builder *Builder) Delete() error {
	if !builder.Exists() {
		return nil
	}

	err := builder.apiClient.Secrets(builder.Definition.Namespace).Delete(
		context.TODO(), builder.Object.Name, metaV1.DeleteOptions{})

	if err != nil {
		return err
	}

	builder.Object = nil

	return err
}

// Exists tells whether the given secret exists.
func (builder *Builder) Exists() bool {
	var err error
	builder.Object, err = builder.apiClient.Secrets(builder.Definition.Namespace).Get(
		context.Background(), builder.Definition.Name, metaV1.GetOptions{})

	return err == nil || !k8serrors.IsNotFound(err)
}

// WithData defines the data placed in the secret.
func (builder *Builder) WithData(data map[string][]byte) *Builder {
	if len(data) == 0 {
		builder.errorMsg = "'data' cannot be empty"
	}

	// Make sure NewBuilder was already called to set builder.Definition.
	if builder.Definition == nil {
		builder.errorMsg = "can not redefine undefined secret"
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Data = data

	return builder
}
