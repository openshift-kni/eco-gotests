package deployment

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-gotests/pkg/clients"
	v1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
)

// Builder provides struct for deployment object containing connection to the cluster and the deployment definitions.
type Builder struct {
	// Deployment definition. Used to create the deployment object.
	Definition *v1.Deployment
	// Created deployment object
	Object *v1.Deployment
	// Used in functions that define or mutate deployment definition. errorMsg is processed before the deployment
	// object is created.
	errorMsg  string
	apiClient *clients.Settings
}

// NewBuilder creates a new instance of Builder.
func NewBuilder(
	apiClient *clients.Settings, name, nsname string, labels map[string]string, containerSpec *coreV1.Container) *Builder {
	glog.V(100).Infof(
		"Initializing new deployment structure with the following params: "+
			"name: %s, namespace: %s, labels: %s, containerSpec %v",
		name, nsname, labels, containerSpec)

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

	builder.WithAdditionalContainerSpecs([]coreV1.Container{*containerSpec})

	if name == "" {
		glog.V(100).Infof("The name of the deployment is empty")

		builder.errorMsg = "deployment 'name' cannot be empty"
	}

	if nsname == "" {
		glog.V(100).Infof("The namespace of the deployment is empty")

		builder.errorMsg = "deployment 'namespace' cannot be empty"
	}

	if labels == nil {
		glog.V(100).Infof("There are no labels for the deployment")

		builder.errorMsg = "deployment 'labels' cannot be empty"
	}

	return &builder
}

// Pull loads an existing deployment into Builder struct.
func Pull(apiClient *clients.Settings, name, nsname string) (*Builder, error) {
	glog.V(100).Infof("Pulling existing deployment name: %s under namespace: %s", name, nsname)

	builder := Builder{
		apiClient: apiClient,
		Definition: &v1.Deployment{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
		},
	}

	if name == "" {
		builder.errorMsg = "deployment 'name' cannot be empty"
	}

	if nsname == "" {
		builder.errorMsg = "deployment 'namespace' cannot be empty"
	}

	if !builder.Exists() {
		return nil, fmt.Errorf("deployment oject %s doesn't exist in namespace %s", name, nsname)
	}

	builder.Definition = builder.Object

	return &builder, nil
}

// WithNodeSelector applies a nodeSelector to the deployment definition.
func (builder *Builder) WithNodeSelector(selector map[string]string) *Builder {
	glog.V(100).Infof("Applying nodeSelector %s to deployment %s in namespace %s",
		selector, builder.Definition.Name, builder.Definition.Namespace)

	if builder.errorMsg != "" {
		return builder
	}

	// Make sure NewBuilder was already called to set builder.Definition.
	if builder.Definition == nil {
		glog.V(100).Infof("The deployment is undefined")

		builder.errorMsg = "cannot add nodeSelector to undefined deployment"

		return builder
	}

	builder.Definition.Spec.Template.Spec.NodeSelector = selector

	return builder
}

// WithReplicas sets the desired number of replicas in the deployment definition.
func (builder *Builder) WithReplicas(replicas int32) *Builder {
	glog.V(100).Infof("Setting %d replicas in deployment %s in namespace %s",
		replicas, builder.Definition.Name, builder.Definition.Namespace)

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
	glog.V(100).Infof("Appending a list of container specs %v to deployment %s in namespace %s",
		specs, builder.Definition.Name, builder.Definition.Namespace)

	if builder.errorMsg != "" {
		return builder
	}

	// Make sure NewBuilder was already called to set builder.Definition.
	if builder.Definition == nil {
		glog.V(100).Infof("The deployment is undefined")

		builder.errorMsg = "cannot add container specs to undefined deployment"

		return builder
	}

	if specs == nil {
		glog.V(100).Infof("The container specs are empty")

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

// Create generates a deployment in cluster and stores the created object in struct.
func (builder *Builder) Create() (*Builder, error) {
	glog.V(100).Infof("Creating deployment %s in namespace %s", builder.Definition.Name, builder.Definition.Namespace)

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

// Update renovates the existing deployment object with the deployment definition in builder.
func (builder *Builder) Update() (*Builder, error) {
	glog.V(100).Infof("Updating deployment %s in namespace %s", builder.Definition.Name, builder.Definition.Namespace)

	if builder.errorMsg != "" {
		return nil, fmt.Errorf(builder.errorMsg)
	}

	var err error
	builder.Object, err = builder.apiClient.Deployments(builder.Definition.Namespace).Update(
		context.TODO(), builder.Definition, metaV1.UpdateOptions{})

	return builder, err
}

// Delete removes a deployment.
func (builder *Builder) Delete() error {
	glog.V(100).Infof("Deleting deployment %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

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

// CreateAndWaitUntilReady creates a deployment in the cluster and waits until the deployment is available.
func (builder *Builder) CreateAndWaitUntilReady(timeout time.Duration) (*Builder, error) {
	glog.V(100).Infof("Creating deployment %s in namespace %s and waiting for the defined period until it's ready",
		builder.Definition.Name, builder.Definition.Namespace)

	if _, err := builder.Create(); err != nil {
		return nil, fmt.Errorf(err.Error())
	}

	if builder.IsReady(timeout) {
		return builder, nil
	}

	return nil, fmt.Errorf("deployment %s in namespace %s is not ready",
		builder.Definition.Name, builder.Definition.Namespace,
	)
}

// IsReady periodically checks if deployment is in ready status.
func (builder *Builder) IsReady(timeout time.Duration) bool {
	glog.V(100).Infof("Running periodic check until deployment %s in namespace %s is ready",
		builder.Definition.Name, builder.Definition.Namespace)

	if !builder.Exists() {
		return false
	}

	err := wait.PollImmediate(time.Second, timeout, func() (bool, error) {

		var err error
		builder.Object, err = builder.apiClient.Deployments(builder.Definition.Namespace).Get(
			context.Background(), builder.Definition.Name, metaV1.GetOptions{})

		if err != nil {
			return false, err
		}

		if builder.Object.Status.ReadyReplicas > 0 && builder.Object.Status.Replicas == builder.Object.Status.ReadyReplicas {
			return true, nil
		}

		return false, nil
	})

	return err == nil
}

// DeleteAndWait deletes a deployment and waits until it is removed from the cluster.
func (builder *Builder) DeleteAndWait(timeout time.Duration) error {
	glog.V(100).Infof("Deleting deployment %s in namespace %s and waiting for the defined period until it's removed",
		builder.Definition.Name, builder.Definition.Namespace)

	if err := builder.Delete(); err != nil {
		return err
	}

	// Polls the deployment every second until it's removed.
	return wait.PollImmediate(time.Second, timeout, func() (bool, error) {
		_, err := builder.apiClient.Deployments(builder.Definition.Namespace).Get(
			context.Background(), builder.Definition.Name, metaV1.GetOptions{})
		if k8serrors.IsNotFound(err) {

			return true, nil
		}

		return false, nil
	})
}

// Exists checks whether the given deployment exists.
func (builder *Builder) Exists() bool {
	glog.V(100).Infof("Checking if deployment %s exists in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	builder.Object, err = builder.apiClient.Deployments(builder.Definition.Namespace).Get(
		context.Background(), builder.Definition.Name, metaV1.GetOptions{})

	return err == nil || !k8serrors.IsNotFound(err)
}

// GetGVR returns deployment's GroupVersionResource which could be used for Clean function.
func GetGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
}

// List returns deployment inventory in the given namespace.
func List(apiClient *clients.Settings, nsname string, options metaV1.ListOptions) ([]*Builder, error) {
	glog.V(100).Infof("Listing deployments in the namespace %s with the options %v", nsname, options)

	if nsname == "" {
		glog.V(100).Infof("deployment 'nsname' parameter can not be empty")

		return nil, fmt.Errorf("failed to list deployments, 'nsname' parameter is empty")
	}

	deploymentList, err := apiClient.Deployments(nsname).List(context.Background(), options)

	if err != nil {
		glog.V(100).Infof("Failed to list deployments in the namespace %s due to %s", nsname, err.Error())

		return nil, err
	}

	var deploymentObjects []*Builder

	for _, runningDeployment := range deploymentList.Items {
		copiedDeployment := runningDeployment
		deploymentBuilder := &Builder{
			apiClient:  apiClient,
			Object:     &copiedDeployment,
			Definition: &copiedDeployment,
		}

		deploymentObjects = append(deploymentObjects, deploymentBuilder)
	}

	return deploymentObjects, nil
}

// WaitUntilCondition waits for the duration of the defined timeout or until the
// deployment gets to a specific condition.
func (builder *Builder) WaitUntilCondition(condition v1.DeploymentConditionType, timeout time.Duration) error {
	glog.V(100).Infof("Waiting for the defined period until deployment %s in namespace %s has condition %v",
		builder.Definition.Name, builder.Definition.Namespace, condition)

	if !builder.Exists() {
		return fmt.Errorf("cannot wait for deployment condition because it does not exist")
	}

	if builder.errorMsg != "" {
		return fmt.Errorf(builder.errorMsg)
	}

	return wait.PollImmediate(time.Second, timeout, func() (bool, error) {
		updateDeployment, err := builder.apiClient.Deployments(builder.Definition.Namespace).Get(
			context.Background(), builder.Definition.Name, metaV1.GetOptions{})
		if err != nil {
			return false, nil
		}

		for _, cond := range updateDeployment.Status.Conditions {
			if cond.Type == condition && cond.Status == coreV1.ConditionTrue {
				return true, nil
			}
		}

		return false, nil

	})
}
