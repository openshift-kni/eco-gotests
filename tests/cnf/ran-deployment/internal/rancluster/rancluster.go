package rancluster

import (
	"fmt"

	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/hive"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"

	"github.com/openshift-kni/eco-gotests/tests/internal/cluster"
)

// AreClustersPresent checks all of the provided clusters and returns false if any are nil.
func AreClustersPresent(clusters []*clients.Settings) bool {
	for _, cluster := range clusters {
		if cluster == nil {
			return false
		}
	}

	return true
}

// GetSpokeClusterName gets the spoke cluster name as string.
func GetSpokeClusterName(hubAPIClient, spokeAPIClient *clients.Settings) (string, error) {
	spokeClusterVersion, err := cluster.GetOCPClusterVersion(spokeAPIClient)
	if err != nil {
		return "", err
	}

	spokeClusterID := spokeClusterVersion.Object.Spec.ClusterID

	clusterDeployments, err := hive.ListClusterDeploymentsInAllNamespaces(hubAPIClient)
	if err != nil {
		return "", err
	}

	for _, clusterDeploymentBuilder := range clusterDeployments {
		if clusterDeploymentBuilder.Object.Spec.ClusterMetadata != nil &&
			clusterDeploymentBuilder.Object.Spec.ClusterMetadata.ClusterID == string(spokeClusterID) {
			return clusterDeploymentBuilder.Object.Spec.ClusterName, nil
		}
	}

	return "", fmt.Errorf("could not find ClusterDeployment from provided API clients")
}

// GetNodeNames gets node names in cluster.
func GetNodeNames(spokeAPIClient *clients.Settings) ([]string, error) {
	nodeList, err := nodes.List(
		spokeAPIClient,
	)

	if err != nil {
		return nil, err
	}

	nodeNames := []string{}
	for _, node := range nodeList {
		nodeNames = append(nodeNames, node.Definition.Name)
	}

	return nodeNames, nil
}
