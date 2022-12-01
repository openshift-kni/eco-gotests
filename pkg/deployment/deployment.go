package deployment

import (
	"context"
	"fmt"
	"time"

	"github.com/openshift-kni/eco-gotests/pkg/clients"
	v1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

// Builder provides struct for deployment object which contains connection to cluster and deployment definition.
type Builder struct {
	// deployment definition. Used to create deployment object.
	Definition *v1.Deployment
	// Created deployment object
	Object *v1.Deployment
	// Used in functions that defines or mutates deployment definition. errorMsg is processed before deployment
	// object is created.
	errorMsg  string
	apiClient *clients.Settings
}

// NewBuilder creates new instance of Builder.
func NewBuilder(
	name, nsname string, labels map[string]string, containerSpec coreV1.Container, apiClient *clients.Settings) *Builder {
	builder := Builder{
		apiClient: apiClient,
		Definition: &v1.Deployment{
			Spec: v1.DeploymentSpec{
				Selector: &metaV1.LabelSelector{
					MatchLabels: labels,
				},
				Template: coreV1.PodTemplateSpec{
					ObjectMeta: metaV1.ObjectMeta{
						Labels: labels,
					},
				},
			},
			ObjectMeta: metaV1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
		},
	}

	builder.WithAdditionalContainerSpecs([]coreV1.Container{containerSpec})

	if name == "" {
		builder.errorMsg = "deployment 'name' cannot be empty"
	}

	if nsname == "" {
		builder.errorMsg = "deployment 'namespace' cannot be empty"
	}

	if labels == nil {
		builder.errorMsg = "deployment 'labels' cannot be empty"
	}

	return &builder
}

// WithNodeSelector applies a nodeSelector to the deployment definition.
func (builder *Builder) WithNodeSelector(selector map[string]string) *Builder {
	if builder.errorMsg != "" {
		return builder
	}

	// Make sure NewBuilder was already called to set builder.Definition.
	if builder.Definition == nil {
		builder.errorMsg = "cannot add nodeSelector to undefined deployment"

		return builder
	}

	builder.Definition.Spec.Template.Spec.NodeSelector = selector

	return builder
}

// WithReplicas sets the desired number of replicas in the deployment definition.
func (builder *Builder) WithReplicas(replicas int32) *Builder {
	if builder.errorMsg != "" {
		return builder
	}

	// Make sure NewBuilder was already called to set builder.Definition.
	if builder.Definition == nil {
		builder.errorMsg = "cannot add replicas to undefined deployment"

		return builder
	}

	builder.Definition.Spec.Replicas = &replicas

	return builder
}

// WithAdditionalContainerSpecs appends a list of container specs to the deployment definition.
func (builder *Builder) WithAdditionalContainerSpecs(specs []coreV1.Container) *Builder {
	if builder.errorMsg != "" {
		return builder
	}

	// Make sure NewBuilder was already called to set builder.Definition.
	if builder.Definition == nil {
		builder.errorMsg = "cannot add container specs to undefined deployment"

		return builder
	}

	if specs == nil {
		builder.errorMsg = "cannot accept nil or empty list as container specs"

		return builder
	}

	if builder.Definition.Spec.Template.Spec.Containers == nil {
		builder.Definition.Spec.Template.Spec.Containers = specs

		return builder
	}

	builder.Definition.Spec.Template.Spec.Containers = append(builder.Definition.Spec.Template.Spec.Containers, specs...)

	return builder
}

// Create creates deployment on cluster and stores created object in struct.
func (builder *Builder) Create() (*Builder, error) {
	if builder.errorMsg != "" {
		return nil, fmt.Errorf(builder.errorMsg)
	}

	var err error
	if !builder.Exists() {
		builder.Object, err = builder.apiClient.Deployments(builder.Definition.Namespace).Create(
			context.TODO(), builder.Definition, metaV1.CreateOptions{})
	}

	return builder, err
}

// Update updates existing deployment object with deployment definition in builder.
func (builder *Builder) Update() (*Builder, error) {
	if builder.errorMsg != "" {
		return nil, fmt.Errorf(builder.errorMsg)
	}

	var err error
	builder.Object, err = builder.apiClient.Deployments(builder.Definition.Namespace).Update(
		context.TODO(), builder.Definition, metaV1.UpdateOptions{})

	return builder, err
}

// Delete deletes deployment.
func (builder *Builder) Delete() error {
	if !builder.Exists() {
		return nil
	}

	err := builder.apiClient.Deployments(builder.Definition.Namespace).Delete(
		context.TODO(), builder.Object.Name, metaV1.DeleteOptions{})

	if err != nil {
		return err
	}

	builder.Object = nil

	return err
}

// CreateAndWaitUntilReady creates a deployment on the cluster and waits until deployment is available.
func (builder *Builder) CreateAndWaitUntilReady(timeout time.Duration) (*Builder, error) {
	_, err := builder.Create()
	if err != nil {
		return nil, fmt.Errorf(err.Error())
	}

	// Polls every one second to determine if deployment is available.
	err = wait.PollImmediate(time.Second, timeout, func() (bool, error) {
		builder.Object, err = builder.apiClient.Deployments(builder.Definition.Namespace).Get(
			context.Background(), builder.Definition.Name, metaV1.GetOptions{})

		if err != nil {
			return false, nil
		}

		for _, condition := range builder.Object.Status.Conditions {
			if condition.Type == "Available" {
				return condition.Status == "True", nil
			}
		}

		return false, err

	})

	if err == nil {
		return builder, nil
	}

	return nil, err
}

// DeleteAndWait deletes a deployment and waits until it is removed from the cluster.
func (builder *Builder) DeleteAndWait(timeout time.Duration) error {
	if err := builder.Delete(); err != nil {
		return err
	}

	// Polls the deployment every 1 second until it no longer exists.
	return wait.PollImmediate(time.Second, timeout, func() (bool, error) {
		_, err := builder.apiClient.Deployments(builder.Definition.Namespace).Get(
			context.Background(), builder.Definition.Name, metaV1.GetOptions{})
		if k8serrors.IsNotFound(err) {

			return true, nil
		}

		return false, nil
	})
}

// Exists tells whether the given deployment exists.
func (builder *Builder) Exists() bool {
	var err error
	builder.Object, err = builder.apiClient.Deployments(builder.Definition.Namespace).Get(
		context.Background(), builder.Definition.Name, metaV1.GetOptions{})

	return err == nil || !k8serrors.IsNotFound(err)
}
