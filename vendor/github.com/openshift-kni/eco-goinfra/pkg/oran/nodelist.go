package oran

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	hardwaremanagementv1alpha1 "github.com/openshift-kni/oran-o2ims/api/hardwaremanagement/v1alpha1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ListNodes returns a list of Nodes in all namespaces, using the provided options.
func ListNodes(apiClient *clients.Settings, options ...runtimeclient.ListOptions) ([]*NodeBuilder, error) {
	if apiClient == nil {
		glog.V(100).Info("Nodes 'apiClient' parameter cannot be nil")

		return nil, fmt.Errorf("failed to list nodes, 'apiClient' parameter is nil")
	}

	err := apiClient.AttachScheme(hardwaremanagementv1alpha1.AddToScheme)
	if err != nil {
		glog.V(100).Info("Failed to add hardwaremanagement v1alpha1 scheme to client schemes")

		return nil, err
	}

	logMessage := "Listing Nodes in all namespaces"
	passedOptions := runtimeclient.ListOptions{}

	if len(options) > 1 {
		glog.V(100).Info("Nodes 'options' parameter must be empty or single-valued")

		return nil, fmt.Errorf("error: more than one ListOptions was passed")
	}

	if len(options) == 1 {
		passedOptions = options[0]
		logMessage += fmt.Sprintf(" with the options %v", passedOptions)
	}

	glog.V(100).Info(logMessage)

	nodeList := new(hardwaremanagementv1alpha1.NodeList)
	err = apiClient.Client.List(context.TODO(), nodeList, &passedOptions)

	if err != nil {
		glog.V(100).Infof("Failed to list Nodes in all namespaces due to %v", err)

		return nil, err
	}

	var nodeObjects []*NodeBuilder

	for _, node := range nodeList.Items {
		copiedNode := node
		nodeBuilder := &NodeBuilder{
			apiClient:  apiClient.Client,
			Object:     &copiedNode,
			Definition: &copiedNode,
		}

		nodeObjects = append(nodeObjects, nodeBuilder)
	}

	return nodeObjects, nil
}
