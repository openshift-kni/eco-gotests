package siteconfig

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/msg"
	siteconfigv1alpha1 "github.com/openshift-kni/eco-goinfra/pkg/schemes/siteconfig/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/util/wait"
	goclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// CIBuilder provides struct for the ClusterInstance object.
type CIBuilder struct {
	// ClusterInstance definition. Used to create a clusterinstance object.
	Definition *siteconfigv1alpha1.ClusterInstance
	// Created clusterinstance object.
	Object *siteconfigv1alpha1.ClusterInstance
	// apiClient opens api connection to interact with the cluster.
	apiClient goclient.Client
	// Used in functions that define or mutate clusterinstance definition.
	// errorMsg is processed before the clusterinstance object is created.
	errorMsg string
}

// NewCIBuilder creates a new instance of CIBuilder.
func NewCIBuilder(apiClient *clients.Settings, name, nsname string) *CIBuilder {
	glog.V(100).Infof(
		"Initializing new ClusterInstance structure with the following params: name: %s, nsname: %s",
		name, nsname)

	if apiClient == nil {
		glog.V(100).Info("The apiClient for the clusterinstance is nil")

		return nil
	}

	err := apiClient.AttachScheme(siteconfigv1alpha1.AddToScheme)
	if err != nil {
		glog.V(100).Infof("Failed to add siteconfig v1alpha1 scheme to client schemes")

		return nil
	}

	builder := &CIBuilder{
		apiClient: apiClient.Client,
		Definition: &siteconfigv1alpha1.ClusterInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the clusterinstance is empty")

		builder.errorMsg = "clusterinstance 'name' cannot be empty"

		return builder
	}

	if nsname == "" {
		glog.V(100).Infof("The namespace of the clusterinstance is empty")

		builder.errorMsg = "clusterinstance 'nsname' cannot be empty"

		return builder
	}

	return builder
}

// PullClusterInstance retrieves an existing ClusterInstance from the cluster.
func PullClusterInstance(apiClient *clients.Settings, name, nsname string) (*CIBuilder, error) {
	glog.V(100).Infof(
		"Pulling existing clusterinstance with name %s from namespace %s", name, nsname)

	if apiClient == nil {
		glog.V(100).Infof("The apiClient is nil")

		return nil, fmt.Errorf("apiClient cannot be nil")
	}

	err := apiClient.AttachScheme(siteconfigv1alpha1.AddToScheme)
	if err != nil {
		glog.V(100).Infof(
			"Failed to add siteconfigv1alpha1 scheme to client schemes")

		return nil, fmt.Errorf("failed to add siteconfigv1alpha1 to client schemes")
	}

	builder := &CIBuilder{
		apiClient: apiClient.Client,
		Definition: &siteconfigv1alpha1.ClusterInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the clusterinstance is empty")

		return nil, fmt.Errorf("clusterinstance 'name' cannot be empty")
	}

	if nsname == "" {
		glog.V(100).Infof("The namespace of the clusterinstance is empty")

		return nil, fmt.Errorf("clusterinstance 'nsname' cannot be empty")
	}

	if !builder.Exists() {
		return nil, fmt.Errorf("clusterinstance object %s does not exist in namespace %s", name, nsname)
	}

	builder.Definition = builder.Object

	return builder, nil
}

// WithExtraManifests includes manifests via configmap name.
func (builder *CIBuilder) WithExtraManifests(extraManifestsName string) *CIBuilder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof("Adding extra manifest %s to ClusterInstance definition", extraManifestsName)

	if extraManifestsName == "" {
		glog.V(100).Infof("checking the clusterinstance extramanifest is empty")

		builder.errorMsg = "clusterinstance extramanifest cannot be empty"

		return builder
	}

	builder.Definition.Spec.ExtraManifestsRefs =
		append(builder.Definition.Spec.ExtraManifestsRefs, corev1.LocalObjectReference{
			Name: extraManifestsName,
		})

	return builder
}

// WithExtraLabels applies extraLabels to ClusterInstance definition.
func (builder *CIBuilder) WithExtraLabels(key string, labels map[string]string) *CIBuilder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof("Defining clusterinstance extraLabels to %s:%v", key, labels)

	if key == "" {
		glog.V(100).Infof("checking the key is empty")

		builder.errorMsg = "can not apply empty key"

		return builder
	}

	if len(labels) == 0 {
		glog.V(100).Infof("checking the labels are empty")

		builder.errorMsg = "labels can not be empty"

		return builder
	}

	for key := range labels {
		if key == "" {
			glog.V(100).Infof("The 'labels' key cannot be empty")

			builder.errorMsg = "can not apply a labels with an empty key"

			return builder
		}
	}

	if builder.Definition.Spec.ExtraLabels == nil {
		builder.Definition.Spec.ExtraLabels = map[string]map[string]string{}
	}

	builder.Definition.Spec.ExtraLabels[key] = labels

	return builder
}

// WaitForCondition waits until the ClusterInstance
// has a condition that matches the expected, checking only the Type, Status, Reason, and Message fields.
// For the message field, it matches if the message contains the expected.
// Zero fields in the expected condition are ignored.
func (builder *CIBuilder) WaitForCondition(expected metav1.Condition, timeout time.Duration) (*CIBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	if !builder.Exists() {
		glog.V(100).Infof("The clusterinstance does not exist on the cluster")

		return builder, fmt.Errorf(
			"clusterinstance object %s does not exist in namespace %s", builder.Definition.Name, builder.Definition.Namespace)
	}

	err := wait.PollUntilContextTimeout(
		context.TODO(), 3*time.Second, timeout, true, func(ctx context.Context) (bool, error) {
			var err error
			builder.Object, err = builder.Get()

			if err != nil {
				glog.V(100).Info("failed to get clusterinstance %s/%s: %w",
					builder.Definition.Namespace, builder.Definition.Name, err)

				return false, nil
			}

			builder.Definition = builder.Object

			for _, condition := range builder.Object.Status.Conditions {
				if expected.Type != "" && condition.Type != expected.Type {
					continue
				}

				if expected.Status != "" && condition.Status != expected.Status {
					continue
				}

				if expected.Reason != "" && condition.Reason != expected.Reason {
					continue
				}

				if expected.Message != "" && !strings.Contains(condition.Message, expected.Message) {
					continue
				}

				return true, nil
			}

			return false, nil
		})

	return builder, err
}

// Get fetches the defined ClusterInstance from the cluster.
func (builder *CIBuilder) Get() (*siteconfigv1alpha1.ClusterInstance, error) {
	if valid, err := builder.validate(); !valid {
		return nil, err
	}

	glog.V(100).Infof("Getting clusterinstance %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	ClusterInstance := &siteconfigv1alpha1.ClusterInstance{}
	err := builder.apiClient.Get(context.TODO(), goclient.ObjectKey{
		Name:      builder.Definition.Name,
		Namespace: builder.Definition.Namespace,
	}, ClusterInstance)

	if err != nil {
		return nil, err
	}

	return ClusterInstance, err
}

// Create generates an ClusterInstance on the cluster.
func (builder *CIBuilder) Create() (*CIBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Creating the clusterinstance %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	if !builder.Exists() {
		err = builder.apiClient.Create(context.TODO(), builder.Definition)
		if err == nil {
			builder.Object = builder.Definition
		}
	}

	return builder, err
}

// Update modifies an existing ClusterInstance on the cluster.
func (builder *CIBuilder) Update(force bool) (*CIBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Updating clusterinstance %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	if !builder.Exists() {
		glog.V(100).Infof("clusterinstance %s does not exist in namespace %s",
			builder.Definition.Name, builder.Definition.Namespace)

		return builder, fmt.Errorf("cannot update non-existent clusterinstance")
	}

	err := builder.apiClient.Update(context.TODO(), builder.Definition)

	if err != nil {
		if force {
			glog.V(100).Infof(
				msg.FailToUpdateNotification("clusterinstance", builder.Definition.Name, builder.Definition.Namespace))

			err := builder.Delete()

			if err != nil {
				glog.V(100).Infof(
					msg.FailToUpdateError("clusterinstance", builder.Definition.Name, builder.Definition.Namespace))

				return nil, err
			}

			return builder.Create()
		}
	}

	if err == nil {
		builder.Object = builder.Definition
	}

	return builder, err
}

// Delete removes an ClusterInstance from the cluster.
func (builder *CIBuilder) Delete() error {
	if valid, err := builder.validate(); !valid {
		return err
	}

	glog.V(100).Infof("Deleting the clusterinstance %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	if !builder.Exists() {
		glog.V(100).Infof("clusterinstance %s cannot be deleted because it does not exist in namespace %s",
			builder.Definition.Name, builder.Definition.Namespace)

		builder.Object = nil

		return nil
	}

	err := builder.apiClient.Delete(context.TODO(), builder.Definition)

	if err != nil {
		return fmt.Errorf("cannot delete clusterinstance: %w", err)
	}

	builder.Object = nil

	return nil
}

// Exists checks if the defined ClusterInstance has already been created.
func (builder *CIBuilder) Exists() bool {
	if valid, _ := builder.validate(); !valid {
		return false
	}

	glog.V(100).Infof("Checking if clusterinstance %s exists in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	builder.Object, err = builder.Get()

	return err == nil || !k8serrors.IsNotFound(err)
}

// validate will check that the builder and builder definition are properly initialized before
// accessing any member fields.
func (builder *CIBuilder) validate() (bool, error) {
	resourceCRD := "ClusterInstance"

	if builder == nil {
		glog.V(100).Infof("The %s builder is uninitialized", resourceCRD)

		return false, fmt.Errorf("error: received nil %s builder", resourceCRD)
	}

	if builder.Definition == nil {
		glog.V(100).Infof("The %s is undefined", resourceCRD)

		return false, fmt.Errorf(msg.UndefinedCrdObjectErrString(resourceCRD))
	}

	if builder.apiClient == nil {
		glog.V(100).Infof("The %s builder apiclient is nil", resourceCRD)

		return false, fmt.Errorf("%s builder cannot have nil apiClient", resourceCRD)
	}

	if builder.errorMsg != "" {
		glog.V(100).Infof("The %s builder has error message: %s", resourceCRD, builder.errorMsg)

		return false, fmt.Errorf(builder.errorMsg)
	}

	return true, nil
}
