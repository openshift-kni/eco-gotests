package cgu

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/cluster-group-upgrades-operator/pkg/api/clustergroupupgrades/v1alpha1"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ListInAllNamespaces returns a cluster-wide cgu inventory.
func ListInAllNamespaces(apiClient *clients.Settings, options ...runtimeclient.ListOptions) ([]*CguBuilder, error) {
	logMessage := string("Listing CGUS in all namespaces ")

	cguList := &v1alpha1.ClusterGroupUpgradeList{}
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

	err := apiClient.Client.List(context.TODO(), cguList, &passedOptions)

	if err != nil {
		glog.V(100).Infof("Failed to list all CGUs in all namespaces due to %s", err.Error())

		return nil, err
	}

	var cguObjects []*CguBuilder

	for _, policy := range cguList.Items {
		copiedCgu := policy
		cguBuilder := &CguBuilder{
			apiClient:  apiClient.ClientCgu,
			Object:     &copiedCgu,
			Definition: &copiedCgu,
		}

		cguObjects = append(cguObjects, cguBuilder)
	}

	return cguObjects, nil
}
