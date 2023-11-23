package clusteroperator

import (
	"context"
	"time"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

// List returns clusterOperators inventory.
func List(apiClient *clients.Settings) ([]*Builder, error) {
	glog.V(100).Info("Listing all clusterOperators")

	coList, err := apiClient.ClusterOperators().List(context.Background(), metaV1.ListOptions{})

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
func WaitForAllClusteroperatorsAvailable(apiClient *clients.Settings, timeout time.Duration) (bool, error) {
	glog.V(100).Info("Waiting for all clusterOperators to be in available state")

	coList, err := List(apiClient)
	if err != nil {
		glog.V(100).Infof("Failed to list all clusterOperators due to %s", err.Error())

		return false, err
	}

	err = wait.PollImmediate(fiveScds, timeout, func() (bool, error) {
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
func WaitForAllClusteroperatorsStopProgressing(apiClient *clients.Settings, timeout time.Duration) (bool, error) {
	glog.V(100).Infof("Waiting for all clusteroperators to stop progressing")

	coList, err := List(apiClient)
	if err != nil {
		glog.V(100).Infof("Failed to list all clusterOperators due to %s", err.Error())

		return false, err
	}

	err = wait.PollImmediate(fiveScds, timeout, func() (bool, error) {
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
