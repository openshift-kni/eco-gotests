package clusteroperator

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

// List returns clusterOperators inventory.
func List(apiClient *clients.Settings, options ...metav1.ListOptions) ([]*Builder, error) {
	logMessage := "Listing all clusterOperators"
	passedOptions := metav1.ListOptions{}

	if len(options) > 1 {
		glog.V(100).Infof("'options' parameter must be empty or single-valued")

		return nil, fmt.Errorf("error: more than one ListOptions was passed")
	}

	if len(options) == 1 {
		passedOptions = options[0]
		logMessage += fmt.Sprintf(" with the options %v", passedOptions)
	}

	glog.V(100).Infof(logMessage)

	coList, err := apiClient.ClusterOperators().List(context.Background(), passedOptions)

	if err != nil {
		glog.V(100).Infof("Failed to list clusterOperators due to %s", err.Error())

		return nil, err
	}

	var coObjects []*Builder

	for _, clusterOperator := range coList.Items {
		copiedCo := clusterOperator
		coBuilder := &Builder{
			apiClient:  apiClient,
			Object:     &copiedCo,
			Definition: &copiedCo,
		}

		coObjects = append(coObjects, coBuilder)
	}

	return coObjects, nil
}

// WaitForAllClusteroperatorsAvailable waits until all clusterOperators are in available state.
func WaitForAllClusteroperatorsAvailable(
	apiClient *clients.Settings, timeout time.Duration, options ...metav1.ListOptions) (bool, error) {
	glog.V(100).Info("Waiting for all clusterOperators to be in available state")

	err := wait.PollUntilContextTimeout(context.TODO(), fiveScds, timeout, true, func(ctx context.Context) (bool, error) {
		coList, err := List(apiClient, options...)

		if err != nil {
			glog.V(100).Infof("Failed to list all clusterOperators due to %s", err.Error())

			return false, err
		}

		for _, clusteroperator := range coList {
			if !clusteroperator.IsAvailable() {
				glog.V(100).Infof("The %s clusterOperator is not available",
					clusteroperator.Object.Name)

				return false, nil
			}
		}

		return true, nil
	})

	if err == nil {
		glog.V(100).Infof("All clusterOperators were found available before timeout: %v",
			timeout)

		return true, nil
	}

	// Here err is "timed out waiting for the condition"
	glog.V(100).Infof("Not all clusterOperators were found available before timeout: %v",
		timeout)

	return false, err
}

// WaitForAllClusteroperatorsStopProgressing waits until all clusterOperators stopped progressing.
func WaitForAllClusteroperatorsStopProgressing(
	apiClient *clients.Settings, timeout time.Duration, options ...metav1.ListOptions) (bool, error) {
	glog.V(100).Infof("Waiting for all clusteroperators to stop progressing")

	coList, err := List(apiClient, options...)
	if err != nil {
		glog.V(100).Infof("Failed to list all clusterOperators due to %s", err.Error())

		return false, err
	}

	err = wait.PollUntilContextTimeout(context.TODO(), fiveScds, timeout, true, func(ctx context.Context) (bool, error) {
		for _, clusteroperator := range coList {
			if clusteroperator.IsProgressing() {
				glog.V(100).Infof("The %s clusterOperator is still progressing",
					clusteroperator.Object.Name)

				return false, nil
			}
		}

		return true, nil
	})

	if err == nil {
		glog.V(100).Infof("All clusterOperators stopped progressing before timeout: %v",
			timeout)

		return true, nil
	}

	// Here err is "timed out waiting for the condition"
	glog.V(100).Infof("Not all clusterOperators stopped progressing before timeout: %v",
		timeout)

	return false, err
}
