package ocm

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/msg"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	policiesv1 "open-cluster-management.io/governance-policy-propagator/api/v1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type placementBindingBuilder struct {
	// placementBinding Definition, used to create the placementBinding object.
	Definition *policiesv1.PlacementBinding
	// created placementBinding object.
	Object *policiesv1.PlacementBinding
	// api client to interact with the cluster.
	apiClient *clients.Settings
	// used to store latest error message upon defining or mutating placementBinding definition.
	errorMsg string
}

// PullPlacementBinding pulls existing placementBinding into Builder struct.
func PullPlacementBinding(apiClient *clients.Settings, name, nsname string) (*placementBindingBuilder, error) {
	glog.V(100).Infof("Pulling existing placementBinding name %s under namespace %s from cluster", name, nsname)

	builder := placementBindingBuilder{
		apiClient: apiClient,
		Definition: &policiesv1.PlacementBinding{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the placementBinding is empty")

		builder.errorMsg = "placementBinding's 'name' cannot be empty"
	}

	if nsname == "" {
		glog.V(100).Infof("The namespace of the placementBinding is empty")

		builder.errorMsg = "placementBinding's 'namespace' cannot be empty"
	}

	if !builder.Exists() {
		return nil, fmt.Errorf("placementBinding object %s doesn't exist in namespace %s", name, nsname)
	}

	builder.Definition = builder.Object

	return &builder, nil
}

// Exists checks whether the given placementBinding exists.
func (builder *placementBindingBuilder) Exists() bool {
	if valid, _ := builder.validate(); !valid {
		return false
	}

	glog.V(100).Infof("Checking if placementBinding %s exists in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	builder.Object, err = builder.Get()

	return err == nil || !k8serrors.IsNotFound(err)
}

// Get returns a placementBinding object if found.
func (builder *placementBindingBuilder) Get() (*policiesv1.PlacementBinding, error) {
	if valid, err := builder.validate(); !valid {
		return nil, err
	}

	glog.V(100).Infof("Getting placementBinding %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	placementBinding := &policiesv1.PlacementBinding{}

	err := builder.apiClient.Get(context.TODO(), runtimeclient.ObjectKey{
		Name:      builder.Definition.Name,
		Namespace: builder.Definition.Namespace,
	}, placementBinding)

	if err != nil {
		return nil, err
	}

	return placementBinding, err
}

// Create makes a placementBinding in the cluster and stores the created object in struct.
func (builder *placementBindingBuilder) Create() (*placementBindingBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Creating the placementBinding %s in namespace %s",
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

// Delete removes a placementBinding from a cluster.
func (builder *placementBindingBuilder) Delete() (*placementBindingBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Deleting the placementBinding %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	if !builder.Exists() {
		return builder, fmt.Errorf("placementBinding cannot be deleted because it does not exist")
	}

	err := builder.apiClient.Delete(context.TODO(), builder.Definition)

	if err != nil {
		return builder, fmt.Errorf("can not delete placementBinding: %w", err)
	}

	builder.Object = nil

	return builder, nil
}

// Update renovates the existing placementBinding object with the placementBinding definition in builder.
func (builder *placementBindingBuilder) Update(force bool) (*placementBindingBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Updating the placementBinding object: %s in namespace: %s",
		builder.Definition.Name, builder.Definition.Namespace)

	err := builder.apiClient.Update(context.TODO(), builder.Definition)

	if err != nil {
		if force {
			glog.V(100).Infof(
				"Failed to update the placementBinding object %s. "+
					"Note: Force flag set, executed delete/create methods instead", builder.Definition.Name)

			builder, err := builder.Delete()

			if err != nil {
				glog.V(100).Infof(
					"Failed to update the placementBinding object %s, "+
						"due to error in delete function", builder.Definition.Name)

				return nil, err
			}

			return builder.Create()
		}
	}

	if err == nil {
		builder.Object = builder.Definition
	}

	return builder, err
}

// validate will check that the builder and builder definition are properly initialized before
// accessing any member fields.
func (builder *placementBindingBuilder) validate() (bool, error) {
	resourceCRD := "PlacementBinding"

	if builder == nil {
		glog.V(100).Infof("The %s builder is uninitialized", resourceCRD)

		return false, fmt.Errorf("error: received nil %s builder", resourceCRD)
	}

	if builder.Definition == nil {
		glog.V(100).Infof("The %s is undefined", resourceCRD)

		builder.errorMsg = msg.UndefinedCrdObjectErrString(resourceCRD)
	}

	if builder.apiClient == nil {
		glog.V(100).Infof("The %s builder apiclient is nil", resourceCRD)

		builder.errorMsg = fmt.Sprintf("%s builder cannot have nil apiClient", resourceCRD)
	}

	if builder.errorMsg != "" {
		glog.V(100).Infof("The %s builder has error message: %s", resourceCRD, builder.errorMsg)

		return false, fmt.Errorf(builder.errorMsg)
	}

	return true, nil
}
