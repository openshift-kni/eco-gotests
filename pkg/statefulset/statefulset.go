package statefulset

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

// Builder provides struct for statefulset object containing connection to the cluster and the statefulset definitions.
type Builder struct {
	// StatefulSet definition. Used to create the statefulset object.
	Definition *v1.StatefulSet
	// Created statefulset object
	Object *v1.StatefulSet
	// Used in functions that define or mutate statefulset definition. errorMsg is processed before the statefulset
	// object is created.
	errorMsg  string
	apiClient *clients.Settings
}

// NewBuilder creates a new instance of Builder.
func NewBuilder(
	apiClient *clients.Settings,
	name string,
	nsname string,
	labels map[string]string,
	containerSpec *coreV1.Container) *Builder {
	glog.V(100).Infof(
		"Initializing new statefulset structure with the following params: "+
			"name: %s, namespace: %s, labels: %s, containerSpec %v",
		name, nsname, labels, containerSpec)

	builder := Builder{
		apiClient: apiClient,
		Definition: &v1.StatefulSet{
			Spec: v1.StatefulSetSpec{
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
		glog.V(100).Infof("The name of the statefulset is empty")

		builder.errorMsg = "statefulset 'name' cannot be empty"
	}

	if nsname == "" {
		glog.V(100).Infof("The namespace of the statefulset is empty")

		builder.errorMsg = "statefulset 'namespace' cannot be empty"
	}

	if labels == nil {
		glog.V(100).Infof("There are no labels for the statefulset")

		builder.errorMsg = "statefulset 'labels' cannot be empty"
	}

	return &builder
}

// WithAdditionalContainerSpecs appends a list of container specs to the statefulset definition.
func (builder *Builder) WithAdditionalContainerSpecs(specs []coreV1.Container) *Builder {
	glog.V(100).Infof("Appending a list of container specs %v to statefulset %s in namespace %s",
		specs, builder.Definition.Name, builder.Definition.Namespace)

	if builder.errorMsg != "" {
		return builder
	}

	// Make sure NewBuilder was already called to set builder.Definition.
	if builder.Definition == nil {
		glog.V(100).Infof("The statefulset is undefined")

		builder.errorMsg = "cannot add container specs to undefined statefulset"
	}

	if specs == nil {
		glog.V(100).Infof("The container specs are empty")

		builder.errorMsg = "cannot accept nil or empty list as container specs"
	}

	if builder.errorMsg != "" {
		return builder
	}

	if builder.Definition.Spec.Template.Spec.Containers == nil {
		builder.Definition.Spec.Template.Spec.Containers = specs

		return builder
	}

	builder.Definition.Spec.Template.Spec.Containers = append(builder.Definition.Spec.Template.Spec.Containers, specs...)

	return builder
}

// Create generates a statefulset in cluster and stores the created object in struct.
func (builder *Builder) Create() (*Builder, error) {
	glog.V(100).Infof("Creating statefulset %s in namespace %s", builder.Definition.Name, builder.Definition.Namespace)

	if builder.errorMsg != "" {
		return nil, fmt.Errorf(builder.errorMsg)
	}

	var err error
	if !builder.Exists() {
		builder.Object, err = builder.apiClient.StatefulSets(builder.Definition.Namespace).Create(
			context.TODO(), builder.Definition, metaV1.CreateOptions{})
	}

	return builder, err
}

// Exists checks whether the given statefulset exists.
func (builder *Builder) Exists() bool {
	glog.V(100).Infof("Checking if statefulset %s exists in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	builder.Object, err = builder.apiClient.StatefulSets(builder.Definition.Namespace).Get(
		context.Background(), builder.Definition.Name, metaV1.GetOptions{})

	return err == nil || !k8serrors.IsNotFound(err)
}

// IsReady periodically checks if statefulset is in ready status.
func (builder *Builder) IsReady(timeout time.Duration) bool {
	glog.V(100).Infof("Running periodic check until statefulset %s in namespace %s is ready",
		builder.Definition.Name, builder.Definition.Namespace)

	if !builder.Exists() {
		return false
	}

	err := wait.PollImmediate(time.Second, timeout, func() (bool, error) {

		var err error
		builder.Object, err = builder.apiClient.StatefulSets(builder.Definition.Namespace).Get(
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

// GetGVR returns pod's GroupVersionResource which could be used for Clean function.
func GetGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"}
}

// List returns statefulset inventory in the given namespace.
func List(apiClient *clients.Settings, nsname string, options metaV1.ListOptions) ([]*Builder, error) {
	glog.V(100).Infof("Listing statefulsets in the namespace %s with the options %v", nsname, options)

	if nsname == "" {
		glog.V(100).Infof("statefulset 'nsname' parameter can not be empty")

		return nil, fmt.Errorf("failed to list statefulsets, 'nsname' parameter is empty")
	}

	statefulsetList, err := apiClient.StatefulSets(nsname).List(context.Background(), options)

	if err != nil {
		glog.V(100).Infof("Failed to list statefulsets in the namespace %s due to %s", nsname, err.Error())

		return nil, err
	}

	var statefulsetObjects []*Builder

	for _, runningStatefulSet := range statefulsetList.Items {
		copiedStatefulSet := runningStatefulSet
		statefulsetBuilder := &Builder{
			apiClient:  apiClient,
			Object:     &copiedStatefulSet,
			Definition: &copiedStatefulSet,
		}

		statefulsetObjects = append(statefulsetObjects, statefulsetBuilder)
	}

	return statefulsetObjects, nil
}
