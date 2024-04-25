package cnfinittools

import (
	"os"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/cnf/internal/cnfparams"
)

var (
	// TargetHubAPIClient is the api client to the hub cluster.
	TargetHubAPIClient *clients.Settings
	// TargetSNOAPIClient is the api client to the target SNO cluster.
	TargetSNOAPIClient *clients.Settings
)

// init loads all variables automatically when this package is imported. Once package is imported a user has full
// access to all vars within init function. It is recommended to import this package using dot import.
func init() {
	TargetHubAPIClient = DefineAPIClient(cnfparams.TargetHubKubeEnvKey)
	TargetSNOAPIClient = DefineAPIClient(cnfparams.TargetSNOKubeEnvKey)
}

// DefineAPIClient creates new api client instance connected to given cluster.
func DefineAPIClient(kubeconfigEnvVar string) *clients.Settings {
	kubeFilePath, present := os.LookupEnv(kubeconfigEnvVar)

	glog.V(cnfparams.CNFLogLevel).Infof("checking api client access")

	if !present {
		glog.V(cnfparams.CNFLogLevel).Infof("can not load api client. Please check %s env var", kubeconfigEnvVar)

		return nil
	}

	glog.V(cnfparams.CNFLogLevel).Infof("checking whether client is set or not")

	client := clients.New(kubeFilePath)
	if client == nil {
		glog.V(cnfparams.CNFLogLevel).Infof("client is not set please check %s env variable", kubeconfigEnvVar)

		return nil
	}

	return client
}
