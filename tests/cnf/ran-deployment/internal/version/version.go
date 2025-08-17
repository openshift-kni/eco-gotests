package version

import (
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/clusterversion"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran-deployment/internal/ranparam"
	configv1 "github.com/openshift/api/config/v1"
)

// GetOCPVersion uses the cluster version on a given cluster to find the latest OCP version, returning the desired
// version if the latest version could not be found.
func GetOCPVersion(client *clients.Settings) (string, error) {
	clusterVersion, err := clusterversion.Pull(client)
	if err != nil {
		return "", err
	}

	// Workaround for an issue in eco-goinfra where builder.Object is nil even when Pull returns a nil error.
	if clusterVersion.Object == nil {
		return "", fmt.Errorf("failed to get ClusterVersion object")
	}

	histories := clusterVersion.Object.Status.History
	for i := len(histories) - 1; i >= 0; i-- {
		if histories[i].State == configv1.CompletedUpdate {
			return histories[i].Version, nil
		}
	}

	glog.V(ranparam.LogLevel).Info("No completed cluster version found in history, returning desired version")

	return clusterVersion.Object.Status.Desired.Version, nil
}
