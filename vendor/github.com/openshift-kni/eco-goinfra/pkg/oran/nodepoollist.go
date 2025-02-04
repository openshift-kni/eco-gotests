package oran

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	hardwaremanagementv1alpha1 "github.com/openshift-kni/oran-o2ims/api/hardwaremanagement/v1alpha1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ListNodePools returns a list of NodePools in all namespaces, using the provided options.
func ListNodePools(apiClient *clients.Settings, options ...runtimeclient.ListOptions) ([]*NodePoolBuilder, error) {
	if apiClient == nil {
		glog.V(100).Info("NodePools 'apiClient' parameter cannot be nil")

		return nil, fmt.Errorf("failed to list nodePools, 'apiClient' parameter is nil")
	}

	err := apiClient.AttachScheme(hardwaremanagementv1alpha1.AddToScheme)
	if err != nil {
		glog.V(100).Info("Failed to add hardwaremanagement v1alpha1 scheme to client schemes")

		return nil, err
	}

	logMessage := "Listing NodePools in all namespaces"
	passedOptions := runtimeclient.ListOptions{}

	if len(options) > 1 {
		glog.V(100).Info("NodePools 'options' parameter must be empty or single-valued")

		return nil, fmt.Errorf("error: more than one ListOptions was passed")
	}

	if len(options) == 1 {
		passedOptions = options[0]
		logMessage += fmt.Sprintf(" with the options %v", passedOptions)
	}

	glog.V(100).Info(logMessage)

	nodePoolList := new(hardwaremanagementv1alpha1.NodePoolList)
	err = apiClient.Client.List(context.TODO(), nodePoolList, &passedOptions)

	if err != nil {
		glog.V(100).Infof("Failed to list NodePools in all namespaces due to %v", err)

		return nil, err
	}

	var nodePoolObjects []*NodePoolBuilder

	for _, nodePool := range nodePoolList.Items {
		copiedNodePool := nodePool
		nodePoolBuilder := &NodePoolBuilder{
			apiClient:  apiClient.Client,
			Object:     &copiedNodePool,
			Definition: &copiedNodePool,
		}

		nodePoolObjects = append(nodePoolObjects, nodePoolBuilder)
	}

	return nodePoolObjects, nil
}
