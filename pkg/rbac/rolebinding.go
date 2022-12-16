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

// RoleBindingBuilder provides struct for RoleBinding object which contains connection
// to cluster RoleBinding definition.
type RoleBindingBuilder struct {
	// rolebinding definition. Used to create rolebinding object
	Definition *v1.RoleBinding
	// created rolebinding object
	Object *v1.RoleBinding

	// used in functions that defines or mutates rolebinding definition. errorMsg is processed
	// before rolebinding object is created
	errorMsg  string
	apiClient *clients.Settings
}

// NewRoleBindingBuilder creates new instance of RoleBindingBuilder.
func NewRoleBindingBuilder(apiClient *clients.Settings,
	name, nsname, role string,
	subject v1.Subject) *RoleBindingBuilder {
	builder := RoleBindingBuilder{
		apiClient: apiClient,
		Definition: &v1.RoleBinding{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
			RoleRef: v1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Name:     role,
				Kind:     "Role",
			},
		},
	}

	if name == "" {
		builder.errorMsg = "RoleBinding 'name' cannot be empty"
	}

	if nsname == "" {
		builder.errorMsg = "RoleBinding 'nsname' cannot be empty"
	}

	builder.WithSubjects([]v1.Subject{subject})

	return &builder
}

// WithSubjects adds specified Subject to the RoleBinding.
func (builder *RoleBindingBuilder) WithSubjects(subjects []v1.Subject) *RoleBindingBuilder {
	if builder.Definition == nil {
		builder.errorMsg = "cannot redefine undefined rolebinding"
	}

	if len(subjects) == 0 {
		builder.errorMsg = "cannot create rolebinding with empty subject"
	}

	if builder.errorMsg != "" {
		return builder
	}

	for _, subject := range subjects {
		if !slice.Contains(allowedSubjectKinds(), subject.Kind) {
			builder.errorMsg = "rolebinding subject kind must be one of 'ServiceAccount', 'User', 'Group'"
		}

		if subject.Name == "" {
			builder.errorMsg = "rolebinding subject name cannot be empty"
		}

		if builder.errorMsg != "" {
			return builder
		}
	}
	builder.Definition.Subjects = append(builder.Definition.Subjects, subjects...)

	return builder
}

// Create generates the RoleBinding and stores created object in struct.
func (builder *RoleBindingBuilder) Create() (*RoleBindingBuilder, error) {
	if builder.errorMsg != "" {
		return nil, fmt.Errorf(builder.errorMsg)
	}

	var err error
	if !builder.Exists() {
		builder.Object, err = builder.apiClient.RoleBindings(builder.Definition.Namespace).Create(
			context.TODO(), builder.Definition, metaV1.CreateOptions{})
	}

	return builder, err
}

// Delete removes the RoleBinding.
func (builder *RoleBindingBuilder) Delete() error {
	if !builder.Exists() {
		return nil
	}

	err := builder.apiClient.RoleBindings(builder.Definition.Namespace).Delete(
		context.TODO(), builder.Object.Name, metaV1.DeleteOptions{})

	builder.Object = nil

	return err
}

// Update modifies existing RoleBinding on cluster.
func (builder *RoleBindingBuilder) Update() (*RoleBindingBuilder, error) {
	if builder.errorMsg != "" {
		return nil, fmt.Errorf(builder.errorMsg)
	}

	var err error
	builder.Object, err = builder.apiClient.RoleBindings(builder.Definition.Namespace).Update(
		context.TODO(), builder.Definition, metaV1.UpdateOptions{})

	return builder, err
}

// Exists tells whether the given RoleBinding exists.
func (builder *RoleBindingBuilder) Exists() bool {
	var err error
	builder.Object, err = builder.apiClient.RoleBindings(builder.Definition.Namespace).Get(
		context.Background(), builder.Definition.Name, metaV1.GetOptions{})

	return err == nil || !k8serrors.IsNotFound(err)
}
