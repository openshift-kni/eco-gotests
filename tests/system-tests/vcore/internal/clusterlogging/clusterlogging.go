package clusterlogging

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/msg"
	clov1 "github.com/openshift/cluster-logging-operator/api/logging/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	goclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Builder provides struct for clusterLogging object.
type Builder struct {
	// ClusterLogging definition. Used to create clusterLogging object with minimum set of required elements.
	Definition *clov1.ClusterLogging
	// Created clusterLogging object on the cluster.
	Object *clov1.ClusterLogging
	// api client to interact with the cluster.
	apiClient goclient.Client
	// errorMsg is processed before clusterLogging object is created.
	errorMsg string
}

// NewBuilder method creates new instance of builder.
func NewBuilder(
	apiClient *clients.Settings, name, nsname string) *Builder {
	glog.V(100).Infof("Initializing new clusterLogging structure with the following params: name: %s, namespace: %s",
		name, nsname)

	builder := &Builder{
		apiClient: apiClient.Client,
		Definition: &clov1.ClusterLogging{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the clusterLogging is empty")

		builder.errorMsg = "the clusterLogging 'name' cannot be empty"
	}

	if nsname == "" {
		glog.V(100).Infof("The namespace of the clusterLogging is empty")

		builder.errorMsg = "the clusterLogging 'nsname' cannot be empty"
	}

	return builder
}

// Pull retrieves an existing clusterLogging object from the cluster.
func Pull(apiClient *clients.Settings, name, nsname string) (*Builder, error) {
	glog.V(100).Infof(
		"Pulling clusterLogging object name:%s in namespace: %s", name, nsname)

	if apiClient == nil {
		glog.V(100).Infof("The apiClient is empty")

		return nil, fmt.Errorf("clusterLogging 'apiClient' cannot be empty")
	}

	builder := Builder{
		apiClient: apiClient.Client,
		Definition: &clov1.ClusterLogging{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the clusterLogging is empty")

		return nil, fmt.Errorf("clusterLogging 'name' cannot be empty")
	}

	if nsname == "" {
		glog.V(100).Infof("The namespace of the clusterLogging is empty")

		return nil, fmt.Errorf("clusterLogging 'nsname' cannot be empty")
	}

	if !builder.Exists() {
		return nil, fmt.Errorf("clusterLogging object %s does not exist in namespace %s", name, nsname)
	}

	builder.Definition = builder.Object

	return &builder, nil
}

// Get returns clusterLogging object if found.
func (builder *Builder) Get() (*clov1.ClusterLogging, error) {
	if valid, err := builder.validate(); !valid {
		return nil, err
	}

	glog.V(100).Infof("Getting clusterLogging %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	clusterLogging := &clov1.ClusterLogging{}
	err := builder.apiClient.Get(context.TODO(), goclient.ObjectKey{
		Name:      builder.Definition.Name,
		Namespace: builder.Definition.Namespace,
	}, clusterLogging)

	if err != nil {
		return nil, err
	}

	return clusterLogging, err
}

// Create makes a clusterLogging in the cluster and stores the created object in struct.
func (builder *Builder) Create() (*Builder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Creating the clusterLogging %s in namespace %s",
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

// Delete removes clusterLogging from a cluster.
func (builder *Builder) Delete() error {
	if valid, err := builder.validate(); !valid {
		return err
	}

	glog.V(100).Infof("Deleting the clusterLogging %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	if !builder.Exists() {
		return fmt.Errorf("clusterLogging cannot be deleted because it does not exist")
	}

	err := builder.apiClient.Delete(context.TODO(), builder.Definition)

	if err != nil {
		return fmt.Errorf("can not delete clusterLogging: %w", err)
	}

	builder.Object = nil

	return nil
}

// Exists checks whether the given clusterLogging exists.
func (builder *Builder) Exists() bool {
	if valid, _ := builder.validate(); !valid {
		return false
	}

	glog.V(100).Infof("Checking if clusterLogging %s exists in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	builder.Object, err = builder.Get()

	return err == nil || !k8serrors.IsNotFound(err)
}

// Update renovates the existing clusterLogging object with clusterLogging definition in builder.
func (builder *Builder) Update(force bool) (*Builder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Info("Updating clusterLogging %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	err := builder.apiClient.Update(context.TODO(), builder.Definition)

	if err != nil {
		if force {
			glog.V(100).Infof(
				msg.FailToUpdateNotification("clusterLogging", builder.Definition.Name, builder.Definition.Namespace))

			err := builder.Delete()

			if err != nil {
				glog.V(100).Infof(
					msg.FailToUpdateError("clusterLogging", builder.Definition.Name, builder.Definition.Namespace))

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

// WithCollection sets the clusterLogging operator's collection configuration.
func (builder *Builder) WithCollection(
	collection clov1.CollectionSpec) *Builder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof(
		"Setting clusterLogging %s in namespace %s with the collection config: %v",
		builder.Definition.Name, builder.Definition.Namespace, collection)

	builder.Definition.Spec.Collection = &collection

	return builder
}

// WithManagementState sets the clusterLogging operator's managementState configuration.
func (builder *Builder) WithManagementState(
	managementState clov1.ManagementState) *Builder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof(
		"Setting clusterLogging %s in namespace %s with the managementState config: %v",
		builder.Definition.Name, builder.Definition.Namespace, managementState)

	builder.Definition.Spec.ManagementState = managementState

	return builder
}

// WithLogStore sets the clusterLogging operator's logStore configuration.
func (builder *Builder) WithLogStore(
	logStore clov1.LogStoreSpec) *Builder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof(
		"Setting clusterLogging %s in namespace %s with the logStore config: %v",
		builder.Definition.Name, builder.Definition.Namespace, logStore)

	builder.Definition.Spec.LogStore = &logStore

	return builder
}

// WithVisualization sets the clusterLogging operator's visualization configuration.
func (builder *Builder) WithVisualization(
	visualization clov1.VisualizationSpec) *Builder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof(
		"Setting clusterLogging %s in namespace %s with the visualization config: %v",
		builder.Definition.Name, builder.Definition.Namespace, visualization)

	builder.Definition.Spec.Visualization = &visualization

	return builder
}

<<<<<<< HEAD:vendor/github.com/openshift-kni/eco-goinfra/pkg/clusterlogging/clusterlogging.go
// IsReady checks for the duration of timeout if the clusterLogging instance state is Ready.
func (builder *Builder) IsReady(timeout time.Duration) bool {
	if valid, _ := builder.validate(); !valid {
		return false
	}

	err := wait.PollUntilContextTimeout(
		context.TODO(), time.Second, timeout, true, func(ctx context.Context) (bool, error) {
			if !builder.Exists() {
				return false, nil
			}

			for _, condition := range builder.Definition.Status.Conditions {
				if condition.Type == clov1.ConditionReady {
					if condition.Status == corev1.ConditionTrue {
						return true, nil
					}
				}
			}

			return false, nil
		})

	return err == nil
}

=======
>>>>>>> 0a60de03 (vCore: logging refactoring, switch to the loki):tests/system-tests/vcore/internal/clusterlogging/clusterlogging.go
// validate will check that the builder and builder definition are properly initialized before
// accessing any member fields.
func (builder *Builder) validate() (bool, error) {
	resourceCRD := "ClusterLogging"

	if builder == nil {
		glog.V(100).Infof("The %s builder is uninitialized", resourceCRD)

		return false, fmt.Errorf("error: received nil %s builder", resourceCRD)
	}

	if builder.Definition == nil {
		glog.V(100).Infof("The %s is undefined", resourceCRD)

		builder.errorMsg = msg.UndefinedCrdObjectErrString(resourceCRD)
	}

	if builder.apiClient == nil {
		glog.V(100).Infof("The %s builder apiclient is nil", resourceCRD)

		builder.errorMsg = fmt.Sprintf("%s builder cannot have nil apiClient", resourceCRD)
	}

	if builder.errorMsg != "" {
		glog.V(100).Infof("The %s builder has error message: %s", resourceCRD, builder.errorMsg)

		return false, fmt.Errorf(builder.errorMsg)
	}

	return true, nil
}
