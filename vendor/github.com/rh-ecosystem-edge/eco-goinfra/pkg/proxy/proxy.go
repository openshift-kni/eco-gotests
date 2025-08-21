package proxy

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	configv1 "github.com/openshift/api/config/v1"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/clients"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/msg"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	clusterProxyName = "cluster"
)

// Builder provides a struct for proxy object from the cluster and a proxy definition.
type Builder struct {
	// proxy definition, used to create the proxy object.
	Definition *configv1.Proxy
	// Created proxy object.
	Object *configv1.Proxy
	// api client to interact with the cluster.
	apiClient *clients.Settings
}

// Pull loads an existing proxy into Builder struct.
func Pull(apiClient *clients.Settings) (*Builder, error) {
	glog.V(100).Infof("Pulling existing proxy name: %s", clusterProxyName)

	builder := Builder{
		apiClient: apiClient,
		Definition: &configv1.Proxy{
			ObjectMeta: metav1.ObjectMeta{
				Name: clusterProxyName,
			},
		},
	}

	if !builder.Exists() {
		return nil, fmt.Errorf("proxy object %s does not exist", clusterProxyName)
	}

	builder.Definition = builder.Object

	return &builder, nil
}

// Exists checks whether the given proxy exists.
func (builder *Builder) Exists() bool {
	if valid, _ := builder.validate(); !valid {
		return false
	}

	glog.V(100).Infof(
		"Checking if proxy %s exists",
		builder.Definition.Name)

	var err error
	builder.Object, err = builder.apiClient.ConfigV1Interface.Proxies().Get(
		context.TODO(), builder.Definition.Name, metav1.GetOptions{})

	return err == nil || !k8serrors.IsNotFound(err)
}

// validate will check that the builder and builder definition are properly initialized before
// accessing any member fields.
func (builder *Builder) validate() (bool, error) {
	resourceCRD := "Proxy"

	if builder == nil {
		glog.V(100).Infof("The %s builder is uninitialized", resourceCRD)

		return false, fmt.Errorf("error: received nil %s builder", resourceCRD)
	}

	if builder.Definition == nil {
		glog.V(100).Infof("The %s is undefined", resourceCRD)

		return false, fmt.Errorf(msg.UndefinedCrdObjectErrString(resourceCRD))
	}

	if builder.apiClient == nil {
		glog.V(100).Infof("The %s builder apiclient is nil", resourceCRD)

		return false, fmt.Errorf("%s builder cannot have nil apiClient", resourceCRD)
	}

	return true, nil
}
