package siteconfig

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/msg"
	siteconfigv1alpha1 "github.com/openshift-kni/eco-goinfra/pkg/schemes/siteconfig/v1alpha1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	goclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterInstanceBuilder provides struct for the ClusterInstance object.
type ClusterInstanceBuilder struct {
	// ClusterInstance definition. Used to create a clusterinstance object.
	Definition *siteconfigv1alpha1.ClusterInstance
	// Created clusterinstance object.
	Object *siteconfigv1alpha1.ClusterInstance
	// apiClient opens api connection to interact with the cluster.
	apiClient goclient.Client
	// Used in functions that define or mutate clusterinstance definition.
	// errorMsg is processed before the clusterinstance object is created.
	errorMsg string
}

// PullClusterInstance retrieves an existing ClusterInstance from the cluster.
func PullClusterInstance(apiClient *clients.Settings, name, nsname string) (*ClusterInstanceBuilder, error) {
	glog.V(100).Infof(
		"Pulling existing clusterinstance with name %s from namespace %s", name, nsname)

	if apiClient == nil {
		glog.V(100).Infof("The apiClient is nil")

		return nil, fmt.Errorf("apiClient cannot be nil")
	}

	err := apiClient.AttachScheme(siteconfigv1alpha1.AddToScheme)
	if err != nil {
		glog.V(100).Infof(
			"Failed to add siteconfigv1alpha1 scheme to client schemes")

		return nil, fmt.Errorf("failed to add siteconfigv1alpha1 to client schemes")
	}

	builder := &ClusterInstanceBuilder{
		apiClient: apiClient.Client,
		Definition: &siteconfigv1alpha1.ClusterInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the clusterinstance is empty")

		return nil, fmt.Errorf("clusterinstance 'name' cannot be empty")
	}

	if nsname == "" {
		glog.V(100).Infof("The namespace of the clusterinstance is empty")

		return nil, fmt.Errorf("clusterinstance 'nsname' cannot be empty")
	}

	if !builder.Exists() {
		return nil, fmt.Errorf("clusterinstance object %s does not exist in namespace %s", name, nsname)
	}

	builder.Definition = builder.Object

	return builder, nil
}

// Get fetches the defined ClusterInstance from the cluster.
func (builder *ClusterInstanceBuilder) Get() (*siteconfigv1alpha1.ClusterInstance, error) {
	if valid, err := builder.validate(); !valid {
		return nil, err
	}

	glog.V(100).Infof("Getting clusterinstance %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	ClusterInstance := &siteconfigv1alpha1.ClusterInstance{}
	err := builder.apiClient.Get(context.TODO(), goclient.ObjectKey{
		Name:      builder.Definition.Name,
		Namespace: builder.Definition.Namespace,
	}, ClusterInstance)

	if err != nil {
		return nil, err
	}

	return ClusterInstance, err
}

// Create generates an ClusterInstance on the cluster.
func (builder *ClusterInstanceBuilder) Create() (*ClusterInstanceBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Creating the clusterinstance %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	if !builder.Exists() {
		err = builder.apiClient.Create(context.TODO(), builder.Definition)
		if err == nil {
			builder.Object = builder.Definition
		}
	}

	return builder, err
}

// Delete removes an ClusterInstance from the cluster.
func (builder *ClusterInstanceBuilder) Delete() error {
	if valid, err := builder.validate(); !valid {
		return err
	}

	glog.V(100).Infof("Deleting the clusterinstance %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	if !builder.Exists() {
		glog.V(100).Infof("clusterinstance %s cannot be deleted because it does not exist in namespace %s",
			builder.Definition.Name, builder.Definition.Namespace)

		builder.Object = nil

		return nil
	}

	err := builder.apiClient.Delete(context.TODO(), builder.Definition)

	if err != nil {
		return fmt.Errorf("cannot delete clusterinstance: %w", err)
	}

	builder.Object = nil

	return nil
}

// Exists checks if the defined ClusterInstance has already been created.
func (builder *ClusterInstanceBuilder) Exists() bool {
	if valid, _ := builder.validate(); !valid {
		return false
	}

	glog.V(100).Infof("Checking if clusterinstance %s exists in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	builder.Object, err = builder.Get()

	return err == nil || !k8serrors.IsNotFound(err)
}

// validate will check that the builder and builder definition are properly initialized before
// accessing any member fields.
func (builder *ClusterInstanceBuilder) validate() (bool, error) {
	resourceCRD := "ClusterInstance"

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

	if builder.errorMsg != "" {
		glog.V(100).Infof("The %s builder has error message: %s", resourceCRD, builder.errorMsg)

		return false, fmt.Errorf(builder.errorMsg)
	}

	return true, nil
}
