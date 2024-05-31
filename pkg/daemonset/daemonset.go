package daemonset

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
	"k8s.io/apimachinery/pkg/util/wait"
)

// Builder provides struct for daemonset object containing connection to the cluster and the daemonset definitions.
type Builder struct {
	// Daemonset definition. Used to create a daemonset object.
	Definition *v1.DaemonSet
	// Created daemonset object.
	Object *v1.DaemonSet
	// Used in functions that define or mutate daemonset definition. errorMsg is processed before the daemonset
	// object is created.
	errorMsg  string
	apiClient *clients.Settings
}

var retryInterval = time.Second * 3

// NewBuilder creates a new instance of Builder.
func NewBuilder(
	apiClient *clients.Settings, name, nsname string, labels map[string]string, containerSpec coreV1.Container) *Builder {
	glog.V(100).Infof(
		"Initializing new daemonset structure with the following params: "+
			"name: %s, namespace: %s, labels: %s, containerSpec %v",
		name, nsname, labels, containerSpec)

	builder := Builder{
		apiClient: apiClient,
		Definition: &v1.DaemonSet{
			Spec: v1.DaemonSetSpec{
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
		glog.V(100).Infof("The name of the daemonset is empty")

		builder.errorMsg = "daemonset 'name' cannot be empty"
	}

	if nsname == "" {
		glog.V(100).Infof("The namespace of the daemonset is empty")

		builder.errorMsg = "daemonset 'namespace' cannot be empty"
	}

	if len(labels) == 0 {
		glog.V(100).Infof("There are no labels for the daemonset")

		builder.errorMsg = "daemonset 'labels' cannot be empty"
	}

	return &builder
}

// Pull loads an existing daemonSet into the Builder struct.
func Pull(apiClient *clients.Settings, name, nsname string) (*Builder, error) {
	glog.V(100).Infof("Pulling existing daemonset name:%s under namespace:%s", name, nsname)

	builder := Builder{
		apiClient: apiClient,
		Definition: &v1.DaemonSet{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
		},
	}

	if name == "" {
		builder.errorMsg = "daemonset 'name' cannot be empty"
	}

	if nsname == "" {
		builder.errorMsg = "daemonset 'namespace' cannot be empty"
	}

	if !builder.Exists() {
		return nil, fmt.Errorf("daemonset object %s doesn't exist in namespace %s", name, nsname)
	}

	builder.Definition = builder.Object

	return &builder, nil
}

// WithNodeSelector applies nodeSelector to the daemonset definition.
func (builder *Builder) WithNodeSelector(selector map[string]string) *Builder {
	glog.V(100).Infof("Applying nodeSelector %s to daemonset %s in namespace %s",
		selector, builder.Definition.Name, builder.Definition.Namespace)

	if builder.errorMsg != "" {
		return builder
	}

	// Make sure NewBuilder was already called to set builder.Definition.
	if builder.Definition == nil {
		glog.V(100).Infof("The daemonset is undefined")

		builder.errorMsg = "cannot add nodeSelector to undefined daemonset"

		return builder
	}

	builder.Definition.Spec.Template.Spec.NodeSelector = selector

	return builder
}

// WithAdditionalContainerSpecs appends a list of container specs to the daemonset definition.
func (builder *Builder) WithAdditionalContainerSpecs(specs []coreV1.Container) *Builder {
	glog.V(100).Infof("Appending a list of container specs %v to daemonset %s in namespace %s",
		specs, builder.Definition.Name, builder.Definition.Namespace)

	if builder.errorMsg != "" {
		return builder
	}

	// Make sure NewBuilder was already called to set builder.Definition.
	if builder.Definition == nil {
		glog.V(100).Infof("The daemonset is undefined")

		builder.errorMsg = "cannot add container specs to undefined daemonset"

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

// Create builds daemonset in the cluster and stores the created object in struct.
func (builder *Builder) Create() (*Builder, error) {
	glog.V(100).Infof("Creating daemonset %s in namespace %s", builder.Definition.Name, builder.Definition.Namespace)

	if builder.errorMsg != "" {
		return nil, fmt.Errorf(builder.errorMsg)
	}

	var err error
	if !builder.Exists() {
		builder.Object, err = builder.apiClient.DaemonSets(builder.Definition.Namespace).Create(
			context.TODO(), builder.Definition, metaV1.CreateOptions{})
	}

	return builder, err
}

// Update renovates the existing daemonset object with daemonset definition in builder.
func (builder *Builder) Update() (*Builder, error) {
	glog.V(100).Infof("Updating daemonset %s in namespace %s", builder.Definition.Name, builder.Definition.Namespace)

	if builder.errorMsg != "" {
		return nil, fmt.Errorf(builder.errorMsg)
	}

	var err error
	builder.Object, err = builder.apiClient.DaemonSets(builder.Definition.Namespace).Update(
		context.TODO(), builder.Definition, metaV1.UpdateOptions{})

	return builder, err
}

// Delete removes the daemonset.
func (builder *Builder) Delete() error {
	glog.V(100).Infof("Deleting daemonset %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	if !builder.Exists() {
		return nil
	}

	err := builder.apiClient.DaemonSets(builder.Definition.Namespace).Delete(
		context.TODO(), builder.Object.Name, metaV1.DeleteOptions{})

	if err != nil {
		return err
	}

	builder.Object = nil

	return err
}

// CreateAndWaitUntilReady creates a daemonset in the cluster and waits until the daemonset is available.
func (builder *Builder) CreateAndWaitUntilReady(timeout time.Duration) (*Builder, error) {
	glog.V(100).Infof("Creating daemonset %s in namespace %s and waiting for the defined period until it's ready",
		builder.Definition.Name, builder.Definition.Namespace)

	_, err := builder.Create()
	if err != nil {
		return nil, fmt.Errorf(err.Error())
	}

	// Polls every retryInterval to determine if daemonset is available.
	err = wait.PollImmediate(retryInterval, timeout, func() (bool, error) {
		builder.Object, err = builder.apiClient.DaemonSets(builder.Definition.Namespace).Get(
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

// DeleteAndWait deletes a daemonset and waits until it is removed from the cluster.
func (builder *Builder) DeleteAndWait(timeout time.Duration) error {
	glog.V(100).Infof("Deleting daemonset %s in namespace %s and waiting for the defined period until it's removed",
		builder.Definition.Name, builder.Definition.Namespace)

	if err := builder.Delete(); err != nil {
		return err
	}

	// Polls the daemonset every retryInterval until it's removed.
	return wait.PollImmediate(retryInterval, timeout, func() (bool, error) {
		_, err := builder.apiClient.DaemonSets(builder.Definition.Namespace).Get(
			context.Background(), builder.Definition.Name, metaV1.GetOptions{})
		if k8serrors.IsNotFound(err) {

			return true, nil
		}

		return false, nil
	})
}

// Exists checks whether the given daemonset exists.
func (builder *Builder) Exists() bool {
	glog.V(100).Infof("Checking if daemonset %s exists in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	builder.Object, err = builder.apiClient.DaemonSets(builder.Definition.Namespace).Get(
		context.Background(), builder.Definition.Name, metaV1.GetOptions{})

	return err == nil || !k8serrors.IsNotFound(err)
}

// IsReady waits for the daemonset to reach expected number of pods in Ready state.
func (builder *Builder) IsReady(timeout time.Duration) bool {
	glog.V(100).Infof("Running periodic check until daemonset %s in namespace %s is ready or "+
		"timeout %s exceeded", builder.Definition.Name, builder.Definition.Namespace, timeout.String())

	// Polls every retryInterval to determine if daemonset is available.
	err := wait.PollImmediate(retryInterval, timeout, func() (bool, error) {
		if !builder.Exists() {
			return false, fmt.Errorf("daemonset %s is not present on cluster", builder.Object.Name)
		}

		var err error
		builder.Object, err = builder.apiClient.DaemonSets(builder.Definition.Namespace).Get(
			context.Background(), builder.Definition.Name, metaV1.GetOptions{})

		if err != nil {
			return false, nil
		}

		if builder.Object.Status.NumberReady == builder.Object.Status.DesiredNumberScheduled {
			return true, nil
		}

		if builder.Object.Status.NumberReady == builder.Object.Status.UpdatedNumberScheduled {
			return true, nil
		}

		return false, err

	})

	return err == nil
}
