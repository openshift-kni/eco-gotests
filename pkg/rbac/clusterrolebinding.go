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

// ClusterRoleBindingBuilder provides struct for clusterrolebinding object
// containing connection to the cluster and the clusterrolebinding definitions.
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
	glog.V(100).Infof(
		"Initializing new clusterrolebinding structure with the following params: "+
			"name: %s, clusterrole: %s, subject %v",
		name, clusterRole, subject)

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
		glog.V(100).Infof("The name of the clusterrolebinding is empty")

		builder.errorMsg = "clusterrolebinding 'name' cannot be empty"
	}

	return &builder
}

// WithSubjects appends additional subjects to clusterrolebinding definition.
func (builder *ClusterRoleBindingBuilder) WithSubjects(subjects []v1.Subject) *ClusterRoleBindingBuilder {
	glog.V(100).Infof("Appending to the definition of clusterrolebinding %s these additional subjects %v",
		builder.Definition.Name, subjects)

	// Make sure NewClusterRoleBindingBuilder was already called to set builder.Definition.
	if builder.Definition == nil {
		glog.V(100).Infof("The clusterrolebinding is undefined")

		builder.errorMsg = "can not redefine undefined clusterrolebinding"
	}

	if len(subjects) == 0 {
		glog.V(100).Infof("The list of subjects is empty")

		builder.errorMsg = "cannot accept nil or empty slice as subjects"
	}

	if builder.errorMsg != "" {
		return builder
	}

	for _, subject := range subjects {
		if !slices.Contains(allowedSubjectKinds(), subject.Kind) {
			glog.V(100).Infof("The clusterrolebinding subject kind must be one of 'ServiceAccount', 'User', or 'Group'")

			builder.errorMsg = "clusterrolebinding subject kind must be one of 'ServiceAccount', 'User', or 'Group'"
		}

		if subject.Name == "" {
			glog.V(100).Infof("The clusterrolebinding subject name cannot be empty")

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
	glog.V(100).Infof("Creating clusterrolebinding %s",
		builder.Definition.Name)

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
	glog.V(100).Infof("Removing clusterrolebinding %s",
		builder.Definition.Name)

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
	glog.V(100).Infof("Updating clusterrolebinding %s",
		builder.Definition.Name)

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
	glog.V(100).Infof("Checking if clusterrolebinding %s exists",
		builder.Definition.Name)

	var err error
	builder.Object, err = builder.apiClient.ClusterRoleBindings().Get(
		context.Background(), builder.Definition.Name, metaV1.GetOptions{})

	return err == nil || !k8serrors.IsNotFound(err)
}
