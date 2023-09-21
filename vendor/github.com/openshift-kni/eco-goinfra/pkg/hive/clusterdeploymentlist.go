package hive

import (
	"context"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	hiveV1 "github.com/openshift/hive/apis/hive/v1"
	goclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ListClusterDeploymentsInAllNamespaces returns a cluster-wide clusterdeployment inventory.
func ListClusterDeploymentsInAllNamespaces(
	apiClient *clients.Settings,
	options goclient.ListOption) ([]*ClusterDeploymentBuilder, error) {
	glog.V(100).Infof("Listing all clusterdeployments with the options %v", options)

	clusterDeployments := new(hiveV1.ClusterDeploymentList)
	err := apiClient.List(context.TODO(), clusterDeployments, options)

	if err != nil {
		glog.V(100).Infof("Failed to list all clusterDeployments due to %s", err.Error())

		return nil, err
	}

	var clusterDeploymentObjects []*ClusterDeploymentBuilder

	for _, clusterDeployment := range clusterDeployments.Items {
		copiedClusterDeployment := clusterDeployment
		clusterDeploymentBuilder := &ClusterDeploymentBuilder{
			apiClient:  apiClient,
			Object:     &copiedClusterDeployment,
			Definition: &copiedClusterDeployment,
		}

		clusterDeploymentObjects = append(clusterDeploymentObjects, clusterDeploymentBuilder)
	}

	return clusterDeploymentObjects, nil
}
