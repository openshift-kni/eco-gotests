package storage

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-gotests/pkg/clients"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PVBuilder provides struct for persistentvolume object containing connection
// to the cluster and the persistentvolume definitions.
type PVBuilder struct {
	// PersistentVolume definition. Used to create a persistentvolume object
	Definition *v1.PersistentVolume
	// Created persistentvolume object
	Object *v1.PersistentVolume

	apiClient *clients.Settings
}

// PullPersistentVolume gets an existing PersistentVolume from the cluster.
func PullPersistentVolume(apiClient *clients.Settings, persistentVolume string) (*PVBuilder, error) {
	glog.V(100).Infof("Pulling existing PersistentVolume object: %s", persistentVolume)

	builder := PVBuilder{
		apiClient: apiClient,
		Definition: &v1.PersistentVolume{
			ObjectMeta: metaV1.ObjectMeta{
				Name: persistentVolume,
			},
		},
	}

	if !builder.Exists() {
		return nil, fmt.Errorf("PersistentVolume object %s doesn't exist", persistentVolume)
	}

	builder.Definition = builder.Object

	return &builder, nil
}

// Exists checks whether the given PersistentVolume exists.
func (builder *PVBuilder) Exists() bool {
	glog.V(100).Infof("Checking if PersistentVolume %s exists", builder.Definition.Name)

	var err error
	builder.Object, err = builder.apiClient.PersistentVolumes().Get(
		context.Background(), builder.Definition.Name, metaV1.GetOptions{})

	return err == nil || !k8serrors.IsNotFound(err)
}
