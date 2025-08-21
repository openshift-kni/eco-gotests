package cgu

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/clients"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ListInAllNamespaces returns a cluster-wide cgu inventory.
func ListInAllNamespaces(apiClient *clients.Settings, options ...metav1.ListOptions) ([]*CguBuilder, error) {
	logMessage := "Listing CGUS in all namespaces"
	passedOptions := metav1.ListOptions{}

	if apiClient == nil {
		glog.V(100).Infof("CGUs 'apiClient' parameter can not be empty")

		return nil, fmt.Errorf("failed to list cgu objects, 'apiClient' parameter is empty")
	}

	if len(options) > 1 {
		glog.V(100).Infof("'options' parameter must be empty or single-valued")

		return nil, fmt.Errorf("error: more than one ListOptions was passed")
	}

	if len(options) == 1 {
		passedOptions = options[0]
		logMessage += fmt.Sprintf(" with the options %v", passedOptions)
	}

	glog.V(100).Infof(logMessage)

	cguList, err := apiClient.ClientCgu.RanV1alpha1().
		ClusterGroupUpgrades("").List(context.TODO(), passedOptions)

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
