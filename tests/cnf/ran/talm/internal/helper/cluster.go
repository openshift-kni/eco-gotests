package helper

import (
	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/clusterversion"
	"github.com/openshift-kni/eco-goinfra/pkg/ocm"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/raninittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/talm/internal/tsparams"
	v1 "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetClusterVersionDefinition returns a unstructured ClusterVersion definition based on the apiClient.
// Use "Image" to include only DesiredUpdate.Image retrieved from the provided apiClient
// Use "Version" to include only DesiredUpdate.Version retrieved from the provided apiClient
// Use "Both" to include both DesiredUpdate.Image and DesiredUpdate.Image retrieved from the provided apiClient.
func GetClusterVersionDefinition(config string, apiClient *clients.Settings) (*v1.ClusterVersion, error) {
	clusterVersion := &v1.ClusterVersion{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterVersion",
			APIVersion: "config.openshift.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "version",
		},
	}

	var desiredUpdate v1.Update

	currentVersion, err := clusterversion.Pull(apiClient)
	if err != nil {
		return nil, err
	}

	// channel and upstream specs are required when desiredUpdate.version is used
	if config != "Image" {
		version, err := getClusterVersion(apiClient)
		if err != nil {
			return nil, err
		}

		desiredUpdate.Version = version
		clusterVersion.Spec.Upstream = v1.URL(RANConfig.OcpUpgradeUpstreamURL)
		clusterVersion.Spec.Channel = currentVersion.Definition.Spec.Channel
	}

	// no other specs are needed when desiredUpdate.image is used, but usually force upgrade is used in combination
	// when upgrade path is unavailable in upgrade graph.
	if config != "Version" {
		desiredUpdate.Image = currentVersion.Object.Status.Desired.Image
		desiredUpdate.Force = true
	}

	// Add composed desiredUpdate to cluster version spec.
	clusterVersion.Spec.DesiredUpdate = &desiredUpdate

	return clusterVersion, nil
}

func getClusterVersion(client *clients.Settings) (string, error) {
	clusterVersion, err := clusterversion.Pull(client)
	if err != nil {
		return "", err
	}

	histories := clusterVersion.Object.Status.History
	for i := len(histories) - 1; i >= 0; i-- {
		history := histories[i]
		if history.State == "Completed" {
			return history.Version, nil
		}
	}

	glog.V(tsparams.LogLevel).Info("No completed version found in clusterversion. Returning desired version")

	return clusterVersion.Object.Status.Desired.Version, nil
}

// DeleteClusterLabel deletes a label from a specified cluster.
func DeleteClusterLabel(clusterName string, labelToBeDeleted string) error {
	managedCluster, err := ocm.PullManagedCluster(HubAPIClient, clusterName)
	if err != nil {
		return err
	}

	delete(managedCluster.Object.Labels, labelToBeDeleted)

	managedCluster.Definition = managedCluster.Object
	_, err = managedCluster.Update()

	return err
}

// DoesClusterLabelExist looks for a label on a managed cluster and returns true if it exists.
func DoesClusterLabelExist(clusterName string, expectedLabel string) (bool, error) {
	managedCluster, err := ocm.PullManagedCluster(HubAPIClient, clusterName)
	if err != nil {
		return false, err
	}

	for label := range managedCluster.Object.Labels {
		if label == expectedLabel {
			return true, nil
		}
	}

	return false, nil
}
