package helper

import (
	"fmt"
	"strings"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/olm"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/raninittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/talm/internal/tsparams"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
)

// CreateTalmTestNamespace creates the test namespace on the hub and the spokes, if present.
func CreateTalmTestNamespace() error {
	_, err := namespace.NewBuilder(raninittools.APIClient, tsparams.TestNamespace).Create()
	if err != nil {
		return err
	}

	if raninittools.Spoke1APIClient != nil {
		_, err := namespace.NewBuilder(raninittools.Spoke1APIClient, tsparams.TestNamespace).Create()
		if err != nil {
			return err
		}
	}

	// spoke 2 may be optional depending on what tests are running
	if raninittools.Spoke2APIClient != nil {
		_, err := namespace.NewBuilder(raninittools.Spoke2APIClient, tsparams.TestNamespace).Create()
		if err != nil {
			return err
		}
	}

	return nil
}

// InitializeVariables initializes hub and spoke name and version variables in tsparams.
func InitializeVariables() error {
	var err error

	tsparams.TalmVersion, err = getOperatorVersionFromCsv(
		raninittools.APIClient, tsparams.OperatorHubTalmNamespace, tsparams.OpenshiftOperatorNamespace)
	if err != nil {
		return err
	}

	glog.V(tsparams.LogLevel).Infof("hub cluster has TALM version '%s'", tsparams.TalmVersion)

	if raninittools.RANConfig.Spoke1Kubeconfig != "" {
		tsparams.Spoke1Name, err = getClusterName(raninittools.RANConfig.Spoke1Kubeconfig)
		if err != nil {
			return err
		}
	}

	if raninittools.RANConfig.Spoke2Kubeconfig != "" {
		tsparams.Spoke2Name, err = getClusterName(raninittools.RANConfig.Spoke2Kubeconfig)
		if err != nil {
			return err
		}
	}

	return nil
}

// getClusterName extracts the cluster name from provided kubeconfig, assuming there's one cluster in the kubeconfig.
func getClusterName(kubeconfigPath string) (string, error) {
	rawConfig, _ := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
		&clientcmd.ConfigOverrides{
			CurrentContext: "",
		}).RawConfig()

	for _, cluster := range rawConfig.Clusters {
		// Get a cluster name by parsing it from the server hostname. Expects the url to start with
		// `https://api.cluster-name.` so splitting by `.` gives the cluster name.
		splits := strings.Split(cluster.Server, ".")
		clusterName := splits[1]

		glog.V(tsparams.LogLevel).Infof("cluster name %s found for kubeconfig at %s", clusterName, kubeconfigPath)

		return clusterName, nil
	}

	return "", fmt.Errorf("could not get cluster name for kubeconfig at %s", kubeconfigPath)
}

// getOperatorVersionFromCsv returns operator version from csv, or an empty string if no CSV for the provided operator
// is found.
func getOperatorVersionFromCsv(client *clients.Settings, operatorName, operatorNamespace string) (string, error) {
	csv, err := olm.ListClusterServiceVersion(client, operatorNamespace, metav1.ListOptions{})
	if err != nil {
		return "", err
	}

	for _, csv := range csv {
		if strings.Contains(csv.Object.Name, operatorName) {
			return csv.Object.Spec.Version.String(), nil
		}
	}

	return "", fmt.Errorf("could not find version for operator %s in namespace %s", operatorName, operatorNamespace)
}
