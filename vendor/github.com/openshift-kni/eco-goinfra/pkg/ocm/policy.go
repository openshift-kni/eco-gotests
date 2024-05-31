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

// PolicyBuilder provides struct for the policy object containing connection to
// the cluster and the policy definitions.
type PolicyBuilder struct {
	// policy Definition, used to create the policy object.
	Definition *policiesv1.Policy
	// created policy object.
	Object *policiesv1.Policy
	// api client to interact with the cluster.
	apiClient *clients.Settings
	// used to store latest error message upon defining or mutating application definition.
	errorMsg string
}

// PullPolicy pulls existing policy into Builder struct.
func PullPolicy(apiClient *clients.Settings, name, nsname string) (*PolicyBuilder, error) {
	glog.V(100).Infof("Pulling existing policy name %s under namespace %s from cluster", name, nsname)

	builder := PolicyBuilder{
		apiClient: apiClient,
		Definition: &policiesv1.Policy{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the policy is empty")

		builder.errorMsg = "policy's 'name' cannot be empty"
	}

	if nsname == "" {
		glog.V(100).Infof("The namespace of the policy is empty")

		builder.errorMsg = "policy's 'namespace' cannot be empty"
	}

	if !builder.Exists() {
		return nil, fmt.Errorf("policy object %s doesn't exist in namespace %s", name, nsname)
	}

	builder.Definition = builder.Object

	return &builder, nil
}

// Exists checks whether the given policy exists.
func (builder *PolicyBuilder) Exists() bool {
	if valid, _ := builder.validate(); !valid {
		return false
	}

	glog.V(100).Infof("Checking if policy %s exists in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	builder.Object, err = builder.Get()

	return err == nil || !k8serrors.IsNotFound(err)
}

// Get returns a policy object if found.
func (builder *PolicyBuilder) Get() (*policiesv1.Policy, error) {
	if valid, err := builder.validate(); !valid {
		return nil, err
	}

	glog.V(100).Infof("Getting policy %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	policy := &policiesv1.Policy{}

	err := builder.apiClient.Get(context.TODO(), runtimeclient.ObjectKey{
		Name:      builder.Definition.Name,
		Namespace: builder.Definition.Namespace,
	}, policy)

	if err != nil {
		return nil, err
	}

	return policy, err
}

// Create makes a policy in the cluster and stores the created object in struct.
func (builder *PolicyBuilder) Create() (*PolicyBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Creating the policy %s in namespace %s",
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

// Delete removes a policy from a cluster.
func (builder *PolicyBuilder) Delete() (*PolicyBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Deleting the policy %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	if !builder.Exists() {
		return builder, fmt.Errorf("policy cannot be deleted because it does not exist")
	}

	err := builder.apiClient.Delete(context.TODO(), builder.Definition)

	if err != nil {
		return builder, fmt.Errorf("can not delete policy: %w", err)
	}

	builder.Object = nil

	return builder, nil
}

// Update renovates the existing policy object with the policy definition in builder.
func (builder *PolicyBuilder) Update(force bool) (*PolicyBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Updating the policy object: %s in namespace: %s",
		builder.Definition.Name, builder.Definition.Namespace)

	err := builder.apiClient.Update(context.TODO(), builder.Definition)

	if err != nil {
		if force {
			glog.V(100).Infof(
				msg.FailToUpdateNotification("policy", builder.Definition.Name))

			builder, err := builder.Delete()

			if err != nil {
				glog.V(100).Infof(
					msg.FailToUpdateError("policy", builder.Definition.Name))

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
func (builder *PolicyBuilder) validate() (bool, error) {
	resourceCRD := "policy"

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
