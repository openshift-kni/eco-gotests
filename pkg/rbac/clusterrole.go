package rbac

import (
	"context"
	"fmt"

	"github.com/openshift-kni/eco-gotests/pkg/clients"
	v1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

/* ClusterRoleBuilder provides struct for clusterrole object
   which contains connection to cluster and clusterrole definition.
*/
type ClusterRoleBuilder struct {
	// clusterrole definition. Used to create clusterrole object.
	Definition *v1.ClusterRole
	// Created clusterrole object
	Object *v1.ClusterRole
	// Used in functions that defines or mutates clusterrole definition. errorMsg is processed before clusterrole
	// object is created.
	errorMsg  string
	apiClient *clients.Settings
}

// NewClusterRoleBuilder creates new instance of ClusterRoleBuilder.
func NewClusterRoleBuilder(apiClient *clients.Settings, name string, rule v1.PolicyRule) *ClusterRoleBuilder {
	builder := ClusterRoleBuilder{
		apiClient: apiClient,
		Definition: &v1.ClusterRole{
			ObjectMeta: metaV1.ObjectMeta{
				Name: name,
			},
			Rules: []v1.PolicyRule{rule},
		},
	}

	if name == "" {
		builder.errorMsg = "clusterrole 'name' cannot be empty"
	}

	builder.WithRules([]v1.PolicyRule{rule})

	return &builder
}

// WithRules appends additional rules to clusterrole definition.
func (builder *ClusterRoleBuilder) WithRules(rules []v1.PolicyRule) *ClusterRoleBuilder {
	// Make sure NewClusterRoleBuilder was already called to set builder.Definition.
	if builder.Definition == nil {
		builder.errorMsg = "can not redefine undefined clusterrole"
	}

	if len(rules) == 0 {
		builder.errorMsg = "cannot accept nil or empty slice as rules"
	}

	if builder.errorMsg != "" {
		return builder
	}

	for _, rule := range rules {
		if len(rule.APIGroups) == 0 {
			builder.errorMsg = "clusterrole rule must contain at least one APIGroup entry"
		}

		if len(rule.Verbs) == 0 {
			builder.errorMsg = "clusterrole rule must contain at least one Verb entry"
		}

		if len(rule.Resources) == 0 {
			builder.errorMsg = "clusterrole rule must contain at least one Resource entry"
		}

		if builder.errorMsg != "" {
			return builder
		}
	}

	if builder.Definition.Rules == nil {
		builder.Definition.Rules = rules

		return builder
	}

	builder.Definition.Rules = append(builder.Definition.Rules, rules...)

	return builder
}

// Create generates clusterrole on cluster and stores created object in struct.
func (builder *ClusterRoleBuilder) Create() (*ClusterRoleBuilder, error) {
	if builder.errorMsg != "" {
		return nil, fmt.Errorf(builder.errorMsg)
	}

	var err error
	if !builder.Exists() {
		builder.Object, err = builder.apiClient.ClusterRoles().Create(
			context.TODO(), builder.Definition, metaV1.CreateOptions{})
	}

	return builder, err
}

// Delete removes clusterrole from cluster.
func (builder *ClusterRoleBuilder) Delete() error {
	if !builder.Exists() {
		return nil
	}

	err := builder.apiClient.ClusterRoles().Delete(
		context.TODO(), builder.Object.Name, metaV1.DeleteOptions{})

	if err != nil {
		return err
	}

	builder.Object = nil

	return err
}

// Update modifies clusterrole object on cluster.
func (builder *ClusterRoleBuilder) Update() (*ClusterRoleBuilder, error) {
	if builder.errorMsg != "" {
		return nil, fmt.Errorf(builder.errorMsg)
	}

	var err error
	builder.Object, err = builder.apiClient.ClusterRoles().Update(
		context.TODO(), builder.Definition, metaV1.UpdateOptions{})

	return builder, err
}

// Exists checks if clusterrole exists on cluster.
func (builder *ClusterRoleBuilder) Exists() bool {
	var err error
	builder.Object, err = builder.apiClient.ClusterRoles().Get(
		context.Background(), builder.Definition.Name, metaV1.GetOptions{})

	return err == nil || !k8serrors.IsNotFound(err)
}
