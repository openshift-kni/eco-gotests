package secret

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-gotests/pkg/clients"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Builder provides struct for secret object containing connection to the cluster and the secret definitions.
type Builder struct {
	// Secret definition. Used to store the secret object.
	Definition *v1.Secret
	// Created secret object.
	Object *v1.Secret
	// Used in functions that define or mutate secret definitions. errorMsg is processed before the secret
	// object is created.
	errorMsg  string
	apiClient *clients.Settings
}

// NewBuilder creates a new instance of Builder.
func NewBuilder(apiClient *clients.Settings, name, nsname string, secretType v1.SecretType) *Builder {
	glog.V(100).Infof(
		"Initializing new secret structure with the following params: %s, %s, %s",
		name, nsname, string(secretType))

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
		glog.V(100).Infof("The name of the secret is empty")

		builder.errorMsg = "secret 'name' cannot be empty"
	}

	if nsname == "" {
		glog.V(100).Infof("The namespace of the secret is empty")

		builder.errorMsg = "secret 'nsname' cannot be empty"
	}

	return &builder
}

// Create makes a secret in the cluster and stores the created object in struct.
func (builder *Builder) Create() (*Builder, error) {
	glog.V(100).Infof("Creating the secret %s in namespace %s", builder.Definition.Name, builder.Definition.Namespace)

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

// Delete removes a secret from the cluster.
func (builder *Builder) Delete() error {
	glog.V(100).Infof("Deleting the secret %s from namespace %s", builder.Definition.Name, builder.Definition.Namespace)

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

// Exists checks whether the given secret exists.
func (builder *Builder) Exists() bool {
	glog.V(100).Infof(
		"Checking if secret %s exists in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	builder.Object, err = builder.apiClient.Secrets(builder.Definition.Namespace).Get(
		context.Background(), builder.Definition.Name, metaV1.GetOptions{})

	return err == nil || !k8serrors.IsNotFound(err)
}

// WithData defines the data placed in the secret.
func (builder *Builder) WithData(data map[string][]byte) *Builder {
	glog.V(100).Infof(
		"Creating secret %s in namespace %s with this data: %s",
		builder.Definition.Name, builder.Definition.Namespace, data)

	if len(data) == 0 {
		glog.V(100).Infof("The data of the secret is empty")

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
