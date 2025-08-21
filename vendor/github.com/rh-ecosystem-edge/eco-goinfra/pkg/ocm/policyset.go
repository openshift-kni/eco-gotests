package ocm

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/clients"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/msg"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	policiesv1beta1 "open-cluster-management.io/governance-policy-propagator/api/v1beta1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// PolicySetBuilder provides struct for the policySet object containing connection to
// the cluster and the policySet definitions.
type PolicySetBuilder struct {
	// policySet Definition, used to create the policySet object.
	Definition *policiesv1beta1.PolicySet
	// created policySet object.
	Object *policiesv1beta1.PolicySet
	// api client to interact with the cluster.
	apiClient *clients.Settings
	// used to store latest error message upon defining or mutating policySet definition.
	errorMsg string
}

// PullPolicySet pulls existing policySet into Builder struct.
func PullPolicySet(apiClient *clients.Settings, name, nsname string) (*PolicySetBuilder, error) {
	glog.V(100).Infof("Pulling existing policySet name %s under namespace %s from cluster", name, nsname)

	builder := PolicySetBuilder{
		apiClient: apiClient,
		Definition: &policiesv1beta1.PolicySet{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the policyset is empty")

		builder.errorMsg = "policyset's 'name' cannot be empty"
	}

	if nsname == "" {
		glog.V(100).Infof("The namespace of the policyset is empty")

		builder.errorMsg = "policyset's 'namespace' cannot be empty"
	}

	if !builder.Exists() {
		return nil, fmt.Errorf("policyset object %s doesn't exist in namespace %s", name, nsname)
	}

	builder.Definition = builder.Object

	return &builder, nil
}

// Exists checks whether the given policySet exists.
func (builder *PolicySetBuilder) Exists() bool {
	if valid, _ := builder.validate(); !valid {
		return false
	}

	glog.V(100).Infof("Checking if policySet %s exists in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	builder.Object, err = builder.Get()

	return err == nil || !k8serrors.IsNotFound(err)
}

// Get returns a policySet object if found.
func (builder *PolicySetBuilder) Get() (*policiesv1beta1.PolicySet, error) {
	if valid, err := builder.validate(); !valid {
		return nil, err
	}

	glog.V(100).Infof("Getting policySet %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	policySet := &policiesv1beta1.PolicySet{}

	err := builder.apiClient.Get(context.Background(), runtimeclient.ObjectKey{
		Name:      builder.Definition.Name,
		Namespace: builder.Definition.Namespace,
	}, policySet)

	if err != nil {
		return nil, err
	}

	return policySet, err
}

// Create makes a policySet in the cluster and stores the created object in struct.
func (builder *PolicySetBuilder) Create() (*PolicySetBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Creating the policySet %s in namespace %s",
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

// Delete removes a policySet from a cluster.
func (builder *PolicySetBuilder) Delete() (*PolicySetBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Deleting the policySet %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	if !builder.Exists() {
		return builder, fmt.Errorf("policySet cannot be deleted because it does not exist")
	}

	err := builder.apiClient.Delete(context.TODO(), builder.Definition)

	if err != nil {
		return builder, fmt.Errorf("can not delete policySet: %w", err)
	}

	builder.Object = nil

	return builder, nil
}

// Update renovates the existing policySet object with the policySet's definition in builder.
func (builder *PolicySetBuilder) Update(force bool) (*PolicySetBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Updating the policySet object: %s in namespace: %s",
		builder.Definition.Name, builder.Definition.Namespace)

	err := builder.apiClient.Update(context.TODO(), builder.Definition)

	if err != nil {
		if force {
			glog.V(100).Infof(
				msg.FailToUpdateNotification("policySet", builder.Definition.Name))

			builder, err := builder.Delete()

			if err != nil {
				glog.V(100).Infof(
					msg.FailToUpdateError("policySet", builder.Definition.Name))

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
func (builder *PolicySetBuilder) validate() (bool, error) {
	resourceCRD := "policySet"

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
