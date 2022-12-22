package rbac

import (
	"context"
	"fmt"

	"github.com/openshift-kni/eco-gotests/pkg/clients"
	"github.com/openshift-kni/eco-gotests/pkg/slice"
	v1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

/* ClusterRoleBindingBuilder provides struct for clusterrolebinding object
   containing connection to the cluster and the clusterrolebinding definitions.
*/
type ClusterRoleBindingBuilder struct {
	// Clusterrolebinding definition. Used to create a clusterrolebinding object.
	Definition *v1.ClusterRoleBinding
	// Created clusterrolebinding object
	Object *v1.ClusterRoleBinding
	// Used in functions that define or mutate clusterrolebinding definition.
	// errorMsg is processed before the clusterrolebinding object is created.
	errorMsg  string
	apiClient *clients.Settings
}

// NewClusterRoleBindingBuilder creates a new instance of ClusterRoleBindingBuilder.
func NewClusterRoleBindingBuilder(
	apiClient *clients.Settings, name, clusterRole string, subject v1.Subject) *ClusterRoleBindingBuilder {
	builder := ClusterRoleBindingBuilder{
		apiClient: apiClient,
		Definition: &v1.ClusterRoleBinding{
			ObjectMeta: metaV1.ObjectMeta{
				Name: name,
			},
			RoleRef: v1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Name:     clusterRole,
				Kind:     "ClusterRole",
			},
		},
	}

	builder.WithSubjects([]v1.Subject{subject})

	if name == "" {
		builder.errorMsg = "clusterrolebinding 'name' cannot be empty"
	}

	return &builder
}

// WithSubjects appends additional subjects to clusterrolebinding definition.
func (builder *ClusterRoleBindingBuilder) WithSubjects(subjects []v1.Subject) *ClusterRoleBindingBuilder {
	// Make sure NewClusterRoleBindingBuilder was already called to set builder.Definition.
	if builder.Definition == nil {
		builder.errorMsg = "can not redefine undefined clusterrolebinding"
	}

	if len(subjects) == 0 {
		builder.errorMsg = "cannot accept nil or empty slice as subjects"
	}

	if builder.errorMsg != "" {
		return builder
	}

	for _, subject := range subjects {
		if !slice.Contains(allowedSubjectKinds(), subject.Kind) {
			builder.errorMsg = "clusterrolebinding subject kind must be one of 'ServiceAccount', 'User', or 'Group'"
		}

		if subject.Name == "" {
			builder.errorMsg = "clusterrolebinding subject name cannot be empty"
		}

		if builder.errorMsg != "" {
			return builder
		}
	}

	builder.Definition.Subjects = append(builder.Definition.Subjects, subjects...)

	return builder
}

// Create generates a clusterrolebinding in the cluster and stores the created object in struct.
func (builder *ClusterRoleBindingBuilder) Create() (*ClusterRoleBindingBuilder, error) {
	if builder.errorMsg != "" {
		return nil, fmt.Errorf(builder.errorMsg)
	}

	var err error
	if !builder.Exists() {
		builder.Object, err = builder.apiClient.ClusterRoleBindings().Create(
			context.TODO(), builder.Definition, metaV1.CreateOptions{})
	}

	return builder, err
}

// Delete removes a clusterrolebinding from the cluster.
func (builder *ClusterRoleBindingBuilder) Delete() error {
	if !builder.Exists() {
		return nil
	}

	err := builder.apiClient.ClusterRoleBindings().Delete(
		context.TODO(), builder.Object.Name, metaV1.DeleteOptions{})

	if err != nil {
		return err
	}

	builder.Object = nil

	return err
}

// Update modifies a clusterrolebinding object in the cluster.
func (builder *ClusterRoleBindingBuilder) Update() (*ClusterRoleBindingBuilder, error) {
	if builder.errorMsg != "" {
		return nil, fmt.Errorf(builder.errorMsg)
	}

	var err error
	builder.Object, err = builder.apiClient.ClusterRoleBindings().Update(
		context.TODO(), builder.Definition, metaV1.UpdateOptions{})

	return builder, err
}

// Exists checks if clusterrolebinding exists in the cluster.
func (builder *ClusterRoleBindingBuilder) Exists() bool {
	var err error
	builder.Object, err = builder.apiClient.ClusterRoleBindings().Get(
		context.Background(), builder.Definition.Name, metaV1.GetOptions{})

	return err == nil || !k8serrors.IsNotFound(err)
}
