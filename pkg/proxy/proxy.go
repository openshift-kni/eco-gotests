package proxy

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-gotests/pkg/clients"
	v1 "github.com/openshift/api/config/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	clusterProxyName = "cluster"
)

// Builder provides a struct for proxy object from the cluster and a proxy definition.
type Builder struct {
	// proxy definition, used to create the proxy object.
	Definition *v1.Proxy
	// Created proxy object.
	Object *v1.Proxy
	// api client to interact with the cluster.
	apiClient *clients.Settings
}

// Pull loads an existing proxy into Builder struct.
func Pull(apiClient *clients.Settings) (*Builder, error) {
	glog.V(100).Infof("Pulling existing proxy name: %s", clusterProxyName)

	builder := Builder{
		apiClient: apiClient,
		Definition: &v1.Proxy{
			ObjectMeta: metaV1.ObjectMeta{
				Name: clusterProxyName,
			},
		},
	}

	if !builder.Exists() {
		return nil, fmt.Errorf("proxy object %s doesn't exist", clusterProxyName)
	}

	builder.Definition = builder.Object

	return &builder, nil
}

// Exists checks whether the given proxy exists.
func (builder *Builder) Exists() bool {
	glog.V(100).Infof(
		"Checking if proxy %s exists",
		builder.Definition.Name)

	var err error
	builder.Object, err = builder.apiClient.ConfigV1Interface.Proxies().Get(
		context.Background(), builder.Definition.Name, metaV1.GetOptions{})

	return err == nil || !k8serrors.IsNotFound(err)
}
