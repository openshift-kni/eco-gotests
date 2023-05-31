package find

import (
	"fmt"

	"github.com/openshift-kni/eco-gotests/pkg/hive"
	. "github.com/openshift-kni/eco-gotests/tests/assisted/ztp/internal/ztpinittools"
	"github.com/openshift-kni/eco-gotests/tests/internal/cluster"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SpokeClusterName returns the spoke cluster name based on hub and spoke cluster apiclients.
func SpokeClusterName() (string, error) {
	spokeClusterVersion, err := cluster.GetOCPClusterVersion(SpokeConfig)
	if err != nil {
		return "", err
	}

	spokeClusterID := spokeClusterVersion.Object.Spec.ClusterID

	clusterDeployments, err := hive.ListClusterDeploymentsInAllNamespaces(HubAPIClient, &client.ListOptions{})
	if err != nil {
		return "", err
	}

	for _, clusterDeploymentBuilder := range clusterDeployments {
		if clusterDeploymentBuilder.Object.Spec.ClusterMetadata != nil &&
			clusterDeploymentBuilder.Object.Spec.ClusterMetadata.ClusterID == string(spokeClusterID) {
			return clusterDeploymentBuilder.Object.Namespace, nil
		}
	}

	return "", fmt.Errorf("could not find ClusterDeployment from provided API clients")
}
