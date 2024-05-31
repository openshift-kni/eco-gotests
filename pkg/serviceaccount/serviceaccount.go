package serviceaccount

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-gotests/pkg/clients"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Builder provides struct for serviceaccount object containing connection to the cluster and the
// serviceaccount definitions.
type Builder struct {
	// ServiceAccount definition. Used to create serviceaccount object.
	Definition *v1.ServiceAccount
	// Created serviceaccount object.
	Object *v1.ServiceAccount
	// Used in functions that defines or mutates configmap definition. errorMsg is processed before the configmap
	// object is created.
	errorMsg  string
	apiClient *clients.Settings
}

// NewBuilder creates a new instance of Builder.
func NewBuilder(apiClient *clients.Settings, name, nsname string) *Builder {
	glog.V(100).Infof("Initializing new serviceaccount structure with the following params: %s, %s", name, nsname)

	builder := Builder{
		apiClient: apiClient,
		Definition: &v1.ServiceAccount{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the serviceaccount is empty")

		builder.errorMsg = "serviceaccount 'name' cannot be empty"
	}

	if nsname == "" {
		glog.V(100).Infof("The namespace of the serviceaccount is empty")

		builder.errorMsg = "serviceaccount 'nsname' cannot be empty"
	}

	return &builder
}

// Create makes a serviceaccount in cluster and stores the created object in struct.
func (builder *Builder) Create() (*Builder, error) {
	glog.V(100).Infof(
		"Creating serviceaccount %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	if builder.errorMsg != "" {
		return nil, fmt.Errorf(builder.errorMsg)
	}

	var err error
	if !builder.Exists() {
		builder.Object, err = builder.apiClient.ServiceAccounts(builder.Definition.Namespace).Create(
			context.TODO(), builder.Definition, metaV1.CreateOptions{})
	}

	return builder, err
}

// Delete removes a serviceaccount.
func (builder *Builder) Delete() error {
	glog.V(100).Infof(
		"Deleting serviceaccount %s from namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	if !builder.Exists() {
		return nil
	}

	err := builder.apiClient.ConfigMaps(builder.Definition.Namespace).Delete(
		context.TODO(), builder.Definition.Name, metaV1.DeleteOptions{})

	if err != nil {
		return err
	}

	builder.Object = nil

	return err
}

// Exists checks whether the given serviceaccount exists.
func (builder *Builder) Exists() bool {
	glog.V(100).Infof(
		"Checking if serviceaccount %s exists in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	builder.Object, err = builder.apiClient.ServiceAccounts(builder.Definition.Namespace).Get(
		context.Background(), builder.Definition.Name, metaV1.GetOptions{})

	return err == nil || !k8serrors.IsNotFound(err)
}
