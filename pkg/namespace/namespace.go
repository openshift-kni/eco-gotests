package namespace

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-gotests/pkg/clients"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/pointer"
	"k8s.io/utils/strings/slices"
)

// Builder provides struct for namespace object containing connection to the cluster and the namespace definitions.
type Builder struct {
	// Namespace definition. Used to create namespace object.
	Definition *v1.Namespace
	// Created namespace object
	Object *v1.Namespace
	// Used in functions that define or mutate namespace definition. errorMsg is processed before the namespace
	// object is created
	errorMsg  string
	apiClient *clients.Settings
}

// NewBuilder creates new instance of Builder.
func NewBuilder(apiClient *clients.Settings, name string) *Builder {
	glog.V(100).Infof(
		"Initializing new namespace structure with the following param: %s", name)

	builder := Builder{
		apiClient: apiClient,
		Definition: &v1.Namespace{
			ObjectMeta: metaV1.ObjectMeta{
				Name: name,
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the namespace is empty")

		builder.errorMsg = "namespace 'name' cannot be empty"
	}

	return &builder
}

// WithLabel redefines namespace definition with the given label.
func (builder *Builder) WithLabel(key string, value string) *Builder {
	glog.V(100).Infof("Labeling the namespace %s with %s=%s", builder.Definition.Name, key, value)

	if builder.errorMsg != "" {
		return builder
	}

	if key == "" {
		glog.V(100).Infof("The key can't be empty")

		builder.errorMsg = "'key' cannot be empty"

		return builder
	}

	// Make sure NewBuilder was already called to set builder.Definition.
	if builder.Definition == nil {
		builder.errorMsg = "can not redefine undefined namespace"

		return builder
	}

	if builder.Definition.Labels == nil {
		builder.Definition.Labels = map[string]string{}
	}

	builder.Definition.Labels[key] = value

	return builder
}

// WithMultipleLabels redefines namespace definition with the given labels.
func (builder *Builder) WithMultipleLabels(labels map[string]string) *Builder {
	for k, v := range labels {
		builder.WithLabel(k, v)
	}

	return builder
}

// Create makes a namespace in the cluster and stores the created object in struct.
func (builder *Builder) Create() (*Builder, error) {
	glog.V(100).Infof("Creating namespace %s", builder.Definition.Name)

	if builder.errorMsg != "" {
		return nil, fmt.Errorf(builder.errorMsg)
	}

	var err error
	if !builder.Exists() {
		builder.Object, err = builder.apiClient.Namespaces().Create(
			context.TODO(), builder.Definition, metaV1.CreateOptions{})
	}

	return builder, err
}

// Update renovates the existing namespace object with the namespace definition in builder.
func (builder *Builder) Update() (*Builder, error) {
	glog.V(100).Infof("Updating the namespace %s with the namespace definition in the builder", builder.Definition.Name)

	if builder.errorMsg != "" {
		return nil, fmt.Errorf(builder.errorMsg)
	}

	var err error
	builder.Object, err = builder.apiClient.Namespaces().Update(
		context.TODO(), builder.Definition, metaV1.UpdateOptions{})

	return builder, err
}

// Delete removes a namespace.
func (builder *Builder) Delete() error {
	glog.V(100).Infof("Deleting namespace %s", builder.Definition.Name)

	if !builder.Exists() {
		return nil
	}

	err := builder.apiClient.Namespaces().Delete(context.TODO(), builder.Object.Name, metaV1.DeleteOptions{})

	if err != nil {
		return err
	}

	builder.Object = nil

	return err
}

// DeleteAndWait deletes a namespace and waits until it's removed from the cluster.
func (builder *Builder) DeleteAndWait(timeout time.Duration) error {
	glog.V(100).Infof("Deleting namespace %s and waiting for the removal to complete", builder.Definition.Name)

	if err := builder.Delete(); err != nil {
		return err
	}

	return wait.PollImmediate(time.Second, timeout, func() (bool, error) {
		_, err := builder.apiClient.Namespaces().Get(context.Background(), builder.Definition.Name, metaV1.GetOptions{})
		if k8serrors.IsNotFound(err) {

			return true, nil
		}

		return false, nil
	})
}

// Exists checks whether the given namespace exists.
func (builder *Builder) Exists() bool {
	glog.V(100).Infof("Checking if namespace %s exists", builder.Definition.Name)

	var err error
	builder.Object, err = builder.apiClient.Namespaces().Get(
		context.Background(), builder.Definition.Name, metaV1.GetOptions{})

	return err == nil || !k8serrors.IsNotFound(err)
}

// Pull loads existing namespace in to Builder struct.
func Pull(apiClient *clients.Settings, nsname string) (*Builder, error) {
	glog.V(100).Infof("Pulling existing namespace: %s from cluster", nsname)

	builder := Builder{
		apiClient: apiClient,
		Definition: &v1.Namespace{
			ObjectMeta: metaV1.ObjectMeta{
				Name: nsname,
			},
		},
	}

	if nsname == "" {
		builder.errorMsg = "'namespace' cannot be empty"
	}

	if !builder.Exists() {
		return nil, fmt.Errorf("namespace oject %s doesn't exist", nsname)
	}

	builder.Definition = builder.Object

	return &builder, nil
}

// CleanObjects removes given objects from the namespace.
func (builder *Builder) CleanObjects(cleanTimeout time.Duration, objects ...schema.GroupVersionResource) error {
	glog.V(100).Infof("Clean namespace: %s", builder.Definition.Name)

	if len(objects) == 0 {
		return fmt.Errorf("failed to remove empty list of object from namespace %s",
			builder.Definition.Name)
	}

	if !builder.Exists() {
		return fmt.Errorf("failed to remove resources from non-existent namespace %s",
			builder.Definition.Name)
	}

	for _, resource := range objects {
		glog.V(100).Infof("Clean all resources: %s in namespace: %s",
			resource.Resource, builder.Definition.Name)

		err := builder.apiClient.Resource(resource).Namespace(builder.Definition.Name).DeleteCollection(
			context.Background(), metaV1.DeleteOptions{
				GracePeriodSeconds: pointer.Int64(0),
			}, metaV1.ListOptions{})

		if err != nil {
			glog.V(100).Infof("Failed to remove resources: %s in namespace: %s",
				resource.Resource, builder.Definition.Name)

			return err
		}

		err = wait.PollImmediate(3*time.Second, cleanTimeout, func() (bool, error) {
			objList, err := builder.apiClient.Resource(resource).Namespace(builder.Definition.Name).List(
				context.Background(), metaV1.ListOptions{})

			if err != nil || len(objList.Items) > 1 {
				// avoid timeout due to default automatically created openshift
				// configmaps: kube-root-ca.crt openshift-service-ca.crt
				if resource.Resource == "configmaps" {
					return builder.hasOnlyDefaultConfigMaps(objList, err)
				}

				return false, err
			}

			return true, err
		})

		if err != nil {
			glog.V(100).Infof("Failed to remove resources: %s in namespace: %s",
				resource.Resource, builder.Definition.Name)

			return err
		}
	}

	return nil
}

// hasOnlyDefaultConfigMaps returns true if only default configMaps are present in a namespace.
func (builder *Builder) hasOnlyDefaultConfigMaps(objList *unstructured.UnstructuredList, err error) (bool, error) {
	if err != nil {
		return false, err
	}

	if len(objList.Items) != 2 {
		return false, err
	}

	var existingConfigMaps []string
	for _, configMap := range objList.Items {
		existingConfigMaps = append(existingConfigMaps, configMap.GetName())
	}

	// return false if existing configmaps are NOT default pre-deployed openshift configmaps
	if !(slices.Contains(existingConfigMaps, "kube-root-ca.crt") &&
		slices.Contains(existingConfigMaps, "openshift-service-ca.crt")) {
		return false, err
	}

	return true, nil
}
