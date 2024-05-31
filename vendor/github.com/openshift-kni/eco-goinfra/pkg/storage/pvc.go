package storage

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/msg"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PVCBuilder provides struct for persistentvolumeclaim object containing connection
// to the cluster and the persistentvolumeclaim definitions.
type PVCBuilder struct {
	// PersistentVolumeClaim definition. Used to create a persistentvolumeclaim object
	Definition *v1.PersistentVolumeClaim
	// Created persistentvolumeclaim object
	Object *v1.PersistentVolumeClaim

	apiClient *clients.Settings
}

// PullPersistentVolumeClaim gets an existing PersistentVolumeClaim
// from the cluster.
func PullPersistentVolumeClaim(
	apiClient *clients.Settings, persistentVolumeClaim string, nsname string) (
	*PVCBuilder, error) {
	glog.V(100).Infof("Pulling existing PersistentVolumeClaim object: %s from namespace %s",
		persistentVolumeClaim, nsname)

	builder := PVCBuilder{
		apiClient: apiClient,
		Definition: &v1.PersistentVolumeClaim{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      persistentVolumeClaim,
				Namespace: nsname,
			},
		},
	}

	if !builder.Exists() {
		return nil, fmt.Errorf("PersistentVolumeClaim object %s doesn't exist in namespace %s",
			persistentVolumeClaim, nsname)
	}

	builder.Definition = builder.Object

	return &builder, nil
}

// Exists checks whether the given PersistentVolumeClaim exists.
func (builder *PVCBuilder) Exists() bool {
	if valid, _ := builder.validate(); !valid {
		return false
	}

	glog.V(100).Infof("Checking if PersistentVolumeClaim %s exists in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	builder.Object, err = builder.apiClient.PersistentVolumeClaims(builder.Definition.Namespace).Get(
		context.Background(), builder.Definition.Name, metaV1.GetOptions{})

	return err == nil || !k8serrors.IsNotFound(err)
}

// validate will check that the builder and builder definition are properly initialized before
// accessing any member fields.
func (builder *PVCBuilder) validate() (bool, error) {
	resourceCRD := "PersistentVolumeClaim"

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
