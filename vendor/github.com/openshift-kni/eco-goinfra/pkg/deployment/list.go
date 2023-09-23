package deployment

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// List returns deployment inventory in the given namespace.
func List(apiClient *clients.Settings, nsname string, options metaV1.ListOptions) ([]*Builder, error) {
	glog.V(100).Infof("Listing deployments in the namespace %s with the options %v", nsname, options)

	if nsname == "" {
		glog.V(100).Infof("deployment 'nsname' parameter can not be empty")

		return nil, fmt.Errorf("failed to list deployments, 'nsname' parameter is empty")
	}

	deploymentList, err := apiClient.Deployments(nsname).List(context.Background(), options)

	if err != nil {
		glog.V(100).Infof("Failed to list deployments in the namespace %s due to %s", nsname, err.Error())

		return nil, err
	}

	var deploymentObjects []*Builder

	for _, runningDeployment := range deploymentList.Items {
		copiedDeployment := runningDeployment
		deploymentBuilder := &Builder{
			apiClient:  apiClient,
			Object:     &copiedDeployment,
			Definition: &copiedDeployment,
		}

		deploymentObjects = append(deploymentObjects, deploymentBuilder)
	}

	return deploymentObjects, nil
}
