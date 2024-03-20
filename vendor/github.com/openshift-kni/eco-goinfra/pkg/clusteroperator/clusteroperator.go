package clusteroperator

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"fmt"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/msg"
	v1 "github.com/openshift/api/config/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	fiveScds time.Duration = 5 * time.Second
	isTrue                 = "True"
)

// Builder provides struct for clusterOperator object.
type Builder struct {
	// ClusterOperator definition. Used to create a clusterOperator object.
	Definition *v1.ClusterOperator
	// Created clusterOperator object.
	Object *v1.ClusterOperator
	// apiClient opens api connection to the cluster.
	apiClient *clients.Settings
	// Used in functions that define or mutate clusterOperator definition. errorMsg is processed before the
	// ClusterOperator object is created.
	errorMsg string
}

// Exists checks whether the given clusterOperator exists.
func (builder *Builder) Exists() bool {
	if valid, _ := builder.validate(); !valid {
		return false
	}

	glog.V(100).Infof("Checking if clusterOperator %s exists", builder.Definition.Name)

	_, err := builder.apiClient.ClusterOperators().Get(
		context.Background(),
		builder.Definition.Name,
		metav1.GetOptions{})

	return err == nil || !k8serrors.IsNotFound(err)
}

// IsAvailable check if the clusterOperator is available.
func (builder *Builder) IsAvailable() bool {
	glog.V(100).Infof("Verify the availability of %s clusterOperator", builder.Definition.Name)

	if !builder.Exists() {
		return false
	}

	for _, condition := range builder.Object.Status.Conditions {
		if condition.Type == "Available" {
			return condition.Status == isTrue
		}
	}

	return false
}

// IsDegraded checks if the clusterOperator is degraded.
func (builder *Builder) IsDegraded() bool {
	glog.V(100).Infof("Check if %s clusterOperator is degraded", builder.Definition.Name)

	if !builder.Exists() {
		return false
	}

	for _, condition := range builder.Object.Status.Conditions {
		if condition.Type == "Degraded" {
			return condition.Status == isTrue
		}
	}

	return false
}

// IsProgressing checks if the clusterOperator is progressing.
func (builder *Builder) IsProgressing() bool {
	glog.V(100).Infof("Check if %s clusterOperator is progressing", builder.Definition.Name)

	if !builder.Exists() {
		return false
	}

	for _, condition := range builder.Object.Status.Conditions {
		if condition.Type == "Progressing" {
			return condition.Status == isTrue
		}
	}

	return false
}

// WaitUntilAvailable waits for timeout duration or until clusterOperator is Available.
func (builder *Builder) WaitUntilAvailable(timeout time.Duration) error {
	return builder.WaitUntilConditionTrue("Available", timeout)
}

// WaitUntilProgressing waits for timeout duration or until clusterOperator is Progressing.
func (builder *Builder) WaitUntilProgressing(timeout time.Duration) error {
	return builder.WaitUntilConditionTrue("Progressing", timeout)
}

// WaitUntilConditionTrue waits for timeout duration or until clusterOperator gets to a specific status.
func (builder *Builder) WaitUntilConditionTrue(
	conditionType v1.ClusterStatusConditionType, timeout time.Duration) error {
	if valid, err := builder.validate(); !valid {
		return err
	}

	if !builder.Exists() {
		return fmt.Errorf("%s clusterOperator not found", builder.Definition.Name)
	}

	return wait.PollUntilContextTimeout(
		context.TODO(), time.Second, timeout, true, func(ctx context.Context) (bool, error) {
			var err error
			builder.Object, err = builder.apiClient.ClusterOperators().Get(
				context.Background(),
				builder.Definition.Name,
				metav1.GetOptions{})

			if err != nil {
				return false, nil
			}

			for _, condition := range builder.Object.Status.Conditions {
				if condition.Type == conditionType {
					return condition.Status == isTrue, nil
				}
			}

			return false, err
		})
}

// validate will check that the builder and builder definition are properly initialized before
// accessing any member fields.
func (builder *Builder) validate() (bool, error) {
	resourceCRD := "ClusterOperator"

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
