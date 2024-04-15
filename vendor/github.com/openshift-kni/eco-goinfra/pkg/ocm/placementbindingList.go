package ocm

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	policiesv1 "open-cluster-management.io/governance-policy-propagator/api/v1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ListPlacementBindingsInAllNamespaces returns a cluster-wide placementBinding inventory.
func ListPlacementBindingsInAllNamespaces(apiClient *clients.Settings,
	options ...runtimeclient.ListOptions) (
	[]*PlacementBindingBuilder, error) {
	logMessage := string("Listing all placementBindings in all namespaces")
	passedOptions := runtimeclient.ListOptions{}

	if len(options) > 1 {
		glog.V(100).Infof("'options' parameter must be empty or single-valued")

		return nil, fmt.Errorf("error: more than one ListOptions was passed")
	}

	if len(options) == 1 {
		passedOptions = options[0]
		logMessage += fmt.Sprintf(" with the options %v", passedOptions)
	}

	glog.V(100).Infof(logMessage)

	placementBindingList := &policiesv1.PlacementBindingList{}

	err := apiClient.Client.List(context.TODO(), placementBindingList, &passedOptions)

	if err != nil {
		glog.V(100).Infof("Failed to list all placementBindings in all namespaces due to %s", err.Error())

		return nil, err
	}

	var placementBindingObjects []*PlacementBindingBuilder

	for _, placementBinding := range placementBindingList.Items {
		copiedplacementBinding := placementBinding
		placementBinding := &PlacementBindingBuilder{
			apiClient:  apiClient,
			Object:     &copiedplacementBinding,
			Definition: &copiedplacementBinding,
		}

		placementBindingObjects = append(placementBindingObjects, placementBinding)
	}

	return placementBindingObjects, nil
}
