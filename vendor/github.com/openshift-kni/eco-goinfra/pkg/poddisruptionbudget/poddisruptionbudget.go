package poddisruptionbudget

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/msg"
	policyv1 "k8s.io/api/policy/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	policyv1typed "k8s.io/client-go/kubernetes/typed/policy/v1"
)

// Builder provides a struct for the PodDisruptionBudget object and definition.
type Builder struct {
	// PodDisruptionBudget definition
	Definition *policyv1.PodDisruptionBudget

	// Created PodDisruptionBudget object
	Object *policyv1.PodDisruptionBudget

	// Used in functions that define or mutate PodDisruptionBudget definition.
	// errorMsg is processed before the PodDisruptionBudget
	errorMsg  string
	apiClient policyv1typed.PolicyV1Interface
}

// NewBuilder creates a new PodDisruptionBudget builder.
func NewBuilder(apiClient *clients.Settings, name, nsname string) *Builder {
	glog.V(100).Infof("Initializing new PodDisruptionBudget structure with the following params: "+
		"name=%s, namespace=%s", name, nsname)

	if apiClient == nil {
		glog.V(100).Info("API client is nil")

		return nil
	}

	builder := &Builder{
		apiClient: apiClient.PolicyV1Interface,
		Definition: &policyv1.PodDisruptionBudget{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
		},
	}

	if name == "" {
		glog.V(100).Info("PodDisruptionBudget name is empty")

		builder.errorMsg = "PodDisruptionBudget 'name' cannot be empty"

		return builder
	}

	if nsname == "" {
		glog.V(100).Info("PodDisruptionBudget namespace is empty")

		builder.errorMsg = "PodDisruptionBudget 'namespace' cannot be empty"

		return builder
	}

	return builder
}

// Pull retrieves the PodDisruptionBudget from the cluster.
func Pull(apiClient *clients.Settings, name, nsname string) (*Builder, error) {
	if apiClient == nil {
		glog.V(100).Info("apiClient is nil")

		return nil, fmt.Errorf("apiClient is nil")
	}

	glog.V(100).Infof("Pulling PodDisruptionBudget with the following params: name=%s, namespace=%s",
		name, nsname)

	builder := &Builder{
		apiClient: apiClient.PolicyV1Interface,
		Definition: &policyv1.PodDisruptionBudget{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
		},
	}

	if name == "" {
		glog.V(100).Info("PodDisruptionBudget name is empty")

		return nil, fmt.Errorf("PodDisruptionBudget 'name' cannot be empty")
	}

	if nsname == "" {
		glog.V(100).Info("PodDisruptionBudget namespace is empty")

		return nil, fmt.Errorf("PodDisruptionBudget 'namespace' cannot be empty")
	}

	if !builder.Exists() {
		return nil, fmt.Errorf("PodDisruptionBudget object %s does not exist in namespace %s", name, nsname)
	}

	builder.Object = builder.Definition

	return builder, nil
}

// Create creates the PodDisruptionBudget in the cluster.
func (builder *Builder) Create() (*Builder, error) {
	if valid, err := builder.validate(); !valid {
		return nil, err
	}

	glog.V(100).Infof("Creating pod disruption budget %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	if !builder.Exists() {
		builder.Object, err = builder.apiClient.PodDisruptionBudgets(
			builder.Definition.Namespace).Create(context.TODO(),
			builder.Definition, metav1.CreateOptions{})
	}

	return builder, err
}

// Delete deletes the PodDisruptionBudget from the cluster.
func (builder *Builder) Delete() error {
	if valid, err := builder.validate(); !valid {
		return err
	}

	glog.V(100).Infof("Deleting pod disruption budget %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	if !builder.Exists() {
		glog.V(100).Infof("Pod disruption budget %s cannot be deleted because it does not exist",
			builder.Definition.Name)

		builder.Object = nil

		return nil
	}

	err := builder.apiClient.PodDisruptionBudgets(builder.Definition.Namespace).Delete(context.TODO(),
		builder.Definition.Name, metav1.DeleteOptions{})

	if err != nil {
		return err
	}

	builder.Object = nil

	return nil
}

// Exists checks if the PodDisruptionBudget exists in the cluster.
func (builder *Builder) Exists() bool {
	glog.V(100).Info("Checking if the PodDisruptionBudget exists in the cluster")

	var err error
	builder.Object, err = builder.apiClient.PodDisruptionBudgets(
		builder.Definition.Namespace).Get(context.TODO(),
		builder.Definition.Name,
		metav1.GetOptions{})

	return err == nil || !k8serrors.IsNotFound(err)
}

// WithPDBSpec sets the PodDisruptionBudgetSpec for the PodDisruptionBudget.
func (builder *Builder) WithPDBSpec(spec policyv1.PodDisruptionBudgetSpec) *Builder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof("Setting PodDisruptionBudgetSpec for PodDisruptionBudget %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	builder.Definition.Spec = spec

	return builder
}

// Update updates the PodDisruptionBudget in the cluster.
func (builder *Builder) Update(force bool) (*Builder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Updating pod disruption budget %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	if !builder.Exists() {
		return builder, fmt.Errorf("pod disruption budget %s does not exist in namespace %s",
			builder.Definition.Name, builder.Definition.Namespace)
	}

	_, err := builder.apiClient.PodDisruptionBudgets(
		builder.Definition.Namespace).Update(context.TODO(),
		builder.Definition, metav1.UpdateOptions{})

	if err != nil {
		if force {
			glog.V(100).Infof("Force updating pod disruption budget %s in namespace %s",
				builder.Definition.Name, builder.Definition.Namespace)

			err := builder.Delete()

			if err != nil {
				glog.V(100).Infof(msg.FailToUpdateError("pod disruption budget",
					builder.Definition.Name, builder.Definition.Namespace))

				return nil, err
			}

			return builder.Create()
		}

		return nil, err
	}

	builder.Object = builder.Definition

	return builder, err
}

// GetGVR returns the GroupVersionResource for the Pod Disruption Budget.
func GetGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "policy",
		Version:  "v1",
		Resource: "poddisruptionbudgets",
	}
}

func (builder *Builder) validate() (bool, error) {
	resourceCRD := "PodDisruptionBudget"

	if builder == nil {
		glog.V(100).Infof("The %s builder is uninitialized", resourceCRD)

		return false, fmt.Errorf("error: received nil %s builder", resourceCRD)
	}

	if builder.Definition == nil {
		glog.V(100).Infof("The %s is undefined", resourceCRD)

		return false, fmt.Errorf(msg.UndefinedCrdObjectErrString(resourceCRD))
	}

	if builder.apiClient == nil {
		glog.V(100).Infof("The %s API client is uninitialized", resourceCRD)

		return false, fmt.Errorf("%s builder cannot have nil apiClient", resourceCRD)
	}

	if builder.errorMsg != "" {
		glog.V(100).Infof("The %s builder has error message: %s",
			resourceCRD, builder.errorMsg)

		return false, fmt.Errorf(builder.errorMsg)
	}

	return true, nil
}
