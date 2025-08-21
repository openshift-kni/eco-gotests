package ocm

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/clients"
	placementrulev1 "open-cluster-management.io/multicloud-operators-subscription/pkg/apis/apps/placementrule/v1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ListPlacementrulesInAllNamespaces returns a cluster-wide placementrule inventory.
func ListPlacementrulesInAllNamespaces(apiClient *clients.Settings,
	options ...runtimeclient.ListOptions) (
	[]*PlacementRuleBuilder, error) {
	logMessage := string("Listing all placementrules in all namespaces")
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

	placementRuleList := &placementrulev1.PlacementRuleList{}

	err := apiClient.Client.List(context.TODO(), placementRuleList, &passedOptions)

	if err != nil {
		glog.V(100).Infof("Failed to list all placementrules in all namespaces due to %s", err.Error())

		return nil, err
	}

	var placementRuleObjects []*PlacementRuleBuilder

	for _, placementRule := range placementRuleList.Items {
		copiedPlacementRule := placementRule
		placementRuleBuilder := &PlacementRuleBuilder{
			apiClient:  apiClient,
			Object:     &copiedPlacementRule,
			Definition: &copiedPlacementRule,
		}

		placementRuleObjects = append(placementRuleObjects, placementRuleBuilder)
	}

	return placementRuleObjects, nil
}
