package rbac

import (
	"context"
	"fmt"

	"github.com/openshift-kni/eco-gotests/pkg/clients"
	v1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RoleBuilder provides a struct for role object containing connection to the cluster and the role definitions.
type RoleBuilder struct {
	// Role definition. Used to create a role object
	Definition *v1.Role
	// Created role object
	Object *v1.Role

	// Used in functions that define or mutate role definition. errorMsg is processed
	// before the role object is created
	errorMsg  string
	apiClient *clients.Settings
}

// NewRoleBuilder create a new instance of RoleBuilder.
func NewRoleBuilder(apiClient *clients.Settings, name, nsname string, rule v1.PolicyRule) *RoleBuilder {
	builder := RoleBuilder{
		apiClient: apiClient,
		Definition: &v1.Role{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
		},
	}

	if name == "" {
		builder.errorMsg = "Role 'name' cannot be empty"
	}

	if nsname == "" {
		builder.errorMsg = "Role 'nsname' cannot be empty"
	}

	builder.WithRules([]v1.PolicyRule{rule})

	return &builder
}

// WithRules adds the specified PolicyRule to the Role.
func (builder *RoleBuilder) WithRules(rules []v1.PolicyRule) *RoleBuilder {
	if builder.Definition == nil {
		builder.errorMsg = "cannot redefine undefined role"
	}

	if len(rules) == 0 {
		builder.errorMsg = "cannot create role with empty rule"
	}

	if builder.errorMsg != "" {
		return builder
	}

	for _, rule := range rules {
		if len(rule.Verbs) == 0 {
			builder.errorMsg = "role must contain at least one Verb"
		}

		if len(rule.Resources) == 0 {
			builder.errorMsg = "role must contain at least one Resource"
		}

		if len(rule.APIGroups) == 0 {
			builder.errorMsg = "role must contain at least one APIGroup"
		}

		if builder.errorMsg != "" {
			return builder
		}
	}

	if builder.Definition.Rules == nil {
		builder.Definition.Rules = rules
	} else {
		builder.Definition.Rules = append(builder.Definition.Rules, rules...)
	}

	return builder
}

// Create makes a Role in the cluster and stores the created object in struct.
func (builder *RoleBuilder) Create() (*RoleBuilder, error) {
	if builder.errorMsg != "" {
		return nil, fmt.Errorf(builder.errorMsg)
	}

	var err error
	if !builder.Exists() {
		builder.Object, err = builder.apiClient.Roles(builder.Definition.Namespace).Create(
			context.TODO(), builder.Definition, metaV1.CreateOptions{})
	}

	return builder, err
}

// Delete removes a Role.
func (builder *RoleBuilder) Delete() error {
	if !builder.Exists() {
		return nil
	}

	err := builder.apiClient.Roles(builder.Definition.Namespace).Delete(
		context.TODO(), builder.Object.Name, metaV1.DeleteOptions{})

	if err != nil {
		return err
	}

	builder.Object = nil

	return err
}

// Update modifies the existing Role object with role definition in builder.
func (builder *RoleBuilder) Update() (*RoleBuilder, error) {
	if builder.errorMsg != "" {
		return nil, fmt.Errorf(builder.errorMsg)
	}

	var err error
	builder.Object, err = builder.apiClient.Roles(builder.Definition.Namespace).Update(
		context.TODO(), builder.Definition, metaV1.UpdateOptions{})

	return builder, err
}

// Exists checks whether the given Role exists.
func (builder *RoleBuilder) Exists() bool {
	var err error
	builder.Object, err = builder.apiClient.Roles(builder.Definition.Namespace).Get(
		context.Background(), builder.Definition.Name, metaV1.GetOptions{})

	return err == nil || !k8serrors.IsNotFound(err)
}