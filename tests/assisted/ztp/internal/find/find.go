package find

import (
	"fmt"
	"strings"

	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/hive"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	. "github.com/openshift-kni/eco-gotests/tests/assisted/ztp/internal/ztpinittools"
	"github.com/openshift-kni/eco-gotests/tests/internal/cluster"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// HubClusterVersionXY returns the Major.Minor part of the HUB cluster's OCP version.
func HubClusterVersionXY() (string, error) {
	clusterVersion, err := cluster.GetOCPClusterVersion(HubAPIClient)
	if err != nil {
		return "", err
	}

	splittedVersion := strings.Split(clusterVersion.Object.Status.Desired.Version, ".")

	return fmt.Sprintf("%s.%s", splittedVersion[0], splittedVersion[1]), nil
}

// SpokeClusterVersionXY returns the Major.Minor part of the Spoke cluster's OCP version.
func SpokeClusterVersionXY() (string, error) {
	clusterVersion, err := cluster.GetOCPClusterVersion(SpokeConfig)
	if err != nil {
		return "", err
	}

	splittedVersion := strings.Split(clusterVersion.Object.Status.Desired.Version, ".")

	return fmt.Sprintf("%s.%s", splittedVersion[0], splittedVersion[1]), nil
}

// HubClusterVersion returns the OCP version of the HUB cluster.
func HubClusterVersion() (string, error) {
	clusterVersion, err := cluster.GetOCPClusterVersion(HubAPIClient)
	if err != nil {
		return "", err
	}

	return clusterVersion.Object.Status.Desired.Version, nil
}

// SpokeClusterVersion returns the OCP version of the Spoke cluster.
func SpokeClusterVersion() (string, error) {
	clusterVersion, err := cluster.GetOCPClusterVersion(SpokeConfig)
	if err != nil {
		return "", err
	}

	return clusterVersion.Object.Status.Desired.Version, nil
}

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
			return clusterDeploymentBuilder.Object.Spec.ClusterName, nil
		}
	}

	return "", fmt.Errorf("could not find ClusterDeployment from provided API clients")
}

// AssistedServicePod returns pod running assisted-service.
func AssistedServicePod() (*pod.Builder, error) {
	return getPodBuilder(HubAPIClient, "app=assisted-service")
}

// AssistedImageServicePod returns pod running assisted-image-service.
func AssistedImageServicePod() (*pod.Builder, error) {
	return getPodBuilder(HubAPIClient, "app=assisted-image-service")
}

// InfrastructureOperatorPod returns pod running infrastructure-operator.
func InfrastructureOperatorPod() (*pod.Builder, error) {
	return getPodBuilder(HubAPIClient, "control-plane=infrastructure-operator")
}

// getPodBuilder returns a podBuilder of a pod based on provided label.
func getPodBuilder(apiClient *clients.Settings, label string) (*pod.Builder, error) {
	if apiClient == nil {
		return nil, fmt.Errorf("apiClient is nil")
	}

	podList, err := pod.ListInAllNamespaces(apiClient, metav1.ListOptions{LabelSelector: label})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods on cluster: %w", err)
	}

	if len(podList) == 0 {
		return nil, fmt.Errorf("pod with label '%s' not currently running", label)
	}

	if len(podList) > 1 {
		return nil, fmt.Errorf("got unexpected pods when checking for pods with label '%s'", label)
	}

	return podList[0], nil
}
