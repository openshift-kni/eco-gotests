package ocm

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/msg"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterV1Client "open-cluster-management.io/api/client/cluster/clientset/versioned/typed/cluster/v1"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
)

// ManagedClusterBuilder provides a struct for the ManagedCluster object containing connection to the cluster and the
// ManagedCluster definitions.
type ManagedClusterBuilder struct {
	Definition *clusterv1.ManagedCluster
	Object     *clusterv1.ManagedCluster
	errorMsg   string
	apiClient  clusterV1Client.ClusterV1Interface
}

// ManagedClusterAdditionalOptions additional options for ManagedCluster object.
type ManagedClusterAdditionalOptions func(builder *ManagedClusterBuilder) (*ManagedClusterBuilder, error)

// NewManagedClusterBuilder creates a new instance of ManagedClusterBuilder.
func NewManagedClusterBuilder(apiClient *clients.Settings, name string) *ManagedClusterBuilder {
	glog.V(100).Infof(
		"Initializing new ManagedCluster structure with the following params: name: %s", name)

	builder := &ManagedClusterBuilder{
		Definition: &clusterv1.ManagedCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		},
	}

	if apiClient == nil {
		glog.V(100).Info("The apiClient for the ManagedCluster is nil")

		builder.errorMsg = "managedCluster 'apiClient' cannot be nil"

		return builder
	}

	builder.apiClient = apiClient.ClusterV1Interface

	if name == "" {
		glog.V(100).Infof("The name of the ManagedCluster is empty")

		builder.errorMsg = "managedCluster 'name' cannot be empty"

		return builder
	}

	return builder
}

// WithOptions creates ManagedCluster with generic mutation options.
func (builder *ManagedClusterBuilder) WithOptions(options ...ManagedClusterAdditionalOptions) *ManagedClusterBuilder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof("Setting ManagedCluster additional options")

	for _, option := range options {
		if option != nil {
			builder, err := option(builder)

			if err != nil {
				glog.V(100).Infof("Error occurred in mutation function")

				builder.errorMsg = err.Error()

				return builder
			}
		}
	}

	return builder
}

// PullManagedCluster loads an existing ManagedCluster into ManagedClusterBuilder struct.
func PullManagedCluster(apiClient *clients.Settings, name string) (*ManagedClusterBuilder, error) {
	glog.V(100).Infof("Pulling existing ManagedCluster name: %s", name)

	if apiClient == nil {
		glog.V(100).Infof("The apiClient for the ManagedCluster is empty")

		return nil, fmt.Errorf("managedCluster 'apiClient' cannot be empty")
	}

	builder := &ManagedClusterBuilder{
		apiClient: apiClient.ClusterV1Interface,
		Definition: &clusterv1.ManagedCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		},
	}

	if name == "" {
		glog.V(100).Info("The name of the ManagedCluster is empty")

		return nil, fmt.Errorf("managedCluster 'name' cannot be empty")
	}

	if !builder.Exists() {
		glog.V(100).Info("The ManagedCluster does not exist")

		return nil, fmt.Errorf("managedCluster object %s does not exist", name)
	}

	builder.Definition = builder.Object

	return builder, nil
}

// Update modifies an existing ManagedCluster on the cluster.
func (builder *ManagedClusterBuilder) Update() (*ManagedClusterBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Updating ManagedCluster %s", builder.Definition.Name)

	var err error
	builder.Object, err = builder.apiClient.ManagedClusters().
		Update(context.TODO(), builder.Definition, metav1.UpdateOptions{})

	return builder, err
}

// Delete removes a ManagedCluster from the cluster.
func (builder *ManagedClusterBuilder) Delete() error {
	if valid, err := builder.validate(); !valid {
		return err
	}

	glog.V(100).Infof("Deleting the ManagedCluster %s", builder.Definition.Name)

	if !builder.Exists() {
		builder.Object = nil

		return nil
	}

	err := builder.apiClient.ManagedClusters().Delete(context.TODO(), builder.Definition.Name, metav1.DeleteOptions{})

	if err != nil {
		return fmt.Errorf("cannot delete managedCluster: %w", err)
	}

	builder.Object = nil
	builder.Definition.ResourceVersion = ""
	builder.Definition.CreationTimestamp = metav1.Time{}

	return nil
}

// Exists checks if the defined ManagedCluster has already been created.
func (builder *ManagedClusterBuilder) Exists() bool {
	if valid, _ := builder.validate(); !valid {
		return false
	}

	glog.V(100).Infof("Checking if ManagedCluster %s exists", builder.Definition.Name)

	var err error
	builder.Object, err = builder.apiClient.ManagedClusters().
		Get(context.TODO(), builder.Definition.Name, metav1.GetOptions{})

	return err == nil || !k8serrors.IsNotFound(err)
}

// validate will check that the builder and builder definition are properly initialized before
// accessing any member fields.
func (builder *ManagedClusterBuilder) validate() (bool, error) {
	resourceCRD := "managedCluster"

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
