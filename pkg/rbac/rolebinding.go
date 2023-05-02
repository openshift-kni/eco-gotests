package rbac

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-gotests/pkg/clients"
	"golang.org/x/exp/slices"
	v1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RoleBindingBuilder provides struct for RoleBinding object containing connection
// to the cluster RoleBinding definition.
type RoleBindingBuilder struct {
	// Rolebinding definition. Used to create rolebinding object
	Definition *v1.RoleBinding
	// Created rolebinding object
	Object *v1.RoleBinding

	// Used in functions that define or mutate rolebinding definition. errorMsg is processed
	// before the rolebinding object is created
	errorMsg  string
	apiClient *clients.Settings
}

// NewRoleBindingBuilder creates new instance of RoleBindingBuilder.
func NewRoleBindingBuilder(apiClient *clients.Settings,
	name, nsname, role string,
	subject v1.Subject) *RoleBindingBuilder {
	glog.V(100).Infof(
		"Initializing new rolebinding structure with the following params: "+
			"name: %s, namespace: %s, role: %s, subject %v", name, nsname, role, subject)

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
		glog.V(100).Infof("The name of the rolebinding is empty")

		builder.errorMsg = "RoleBinding 'name' cannot be empty"
	}

	if nsname == "" {
		glog.V(100).Infof("The namespace of the rolebinding is empty")

		builder.errorMsg = "RoleBinding 'nsname' cannot be empty"
	}

	builder.WithSubjects([]v1.Subject{subject})

	return &builder
}

// WithSubjects adds specified Subject to the RoleBinding.
func (builder *RoleBindingBuilder) WithSubjects(subjects []v1.Subject) *RoleBindingBuilder {
	if builder.Definition == nil {
		glog.V(100).Infof("The rolebinding is undefined")

		builder.errorMsg = "cannot redefine undefined rolebinding"
	}

	glog.V(100).Infof("Adding to the rolebinding %s these specified subjects: %v",
		builder.Definition.Name, subjects)

	if len(subjects) == 0 {
		glog.V(100).Infof("The list of subjects is empty")

		builder.errorMsg = "cannot create rolebinding with empty subject"
	}

	if builder.errorMsg != "" {
		return builder
	}

	for _, subject := range subjects {
		if !slices.Contains(allowedSubjectKinds(), subject.Kind) {
			glog.V(100).Infof("The rolebinding subject kind must be one of 'ServiceAccount', 'User', or 'Group'")

			builder.errorMsg = "rolebinding subject kind must be one of 'ServiceAccount', 'User', 'Group'"
		}

		if subject.Name == "" {
			glog.V(100).Infof("The rolebinding subject name cannot be empty")

			builder.errorMsg = "rolebinding subject name cannot be empty"
		}

		if builder.errorMsg != "" {
			return builder
		}
	}
	builder.Definition.Subjects = append(builder.Definition.Subjects, subjects...)

	return builder
}

// Create generates a RoleBinding and stores the created object in struct.
func (builder *RoleBindingBuilder) Create() (*RoleBindingBuilder, error) {
	glog.V(100).Infof("Creating rolebinding %s under namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

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

// Delete removes a RoleBinding.
func (builder *RoleBindingBuilder) Delete() error {
	glog.V(100).Infof("Removing rolebinding %s under namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	if !builder.Exists() {
		return nil
	}

	err := builder.apiClient.RoleBindings(builder.Definition.Namespace).Delete(
		context.TODO(), builder.Object.Name, metaV1.DeleteOptions{})

	builder.Object = nil

	return err
}

// Update modifies an existing RoleBinding in the cluster.
func (builder *RoleBindingBuilder) Update() (*RoleBindingBuilder, error) {
	glog.V(100).Infof("Updating rolebinding %s under namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	if builder.errorMsg != "" {
		return nil, fmt.Errorf(builder.errorMsg)
	}

	var err error
	builder.Object, err = builder.apiClient.RoleBindings(builder.Definition.Namespace).Update(
		context.TODO(), builder.Definition, metaV1.UpdateOptions{})

	return builder, err
}

// Exists checks whether the given RoleBinding exists.
func (builder *RoleBindingBuilder) Exists() bool {
	glog.V(100).Infof("Checking if rolebinding %s exists under namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	builder.Object, err = builder.apiClient.RoleBindings(builder.Definition.Namespace).Get(
		context.Background(), builder.Definition.Name, metaV1.GetOptions{})

	return err == nil || !k8serrors.IsNotFound(err)
}
