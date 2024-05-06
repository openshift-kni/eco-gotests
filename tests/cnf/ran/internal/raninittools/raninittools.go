package raninittools

import (
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/ranconfig"
	"github.com/openshift-kni/eco-gotests/tests/internal/inittools"
)

var (
	// HubAPIClient provides API access to the first spoke cluster.
	HubAPIClient *clients.Settings
	// Spoke1APIClient provides API access to cluster.
	Spoke1APIClient *clients.Settings
	// Spoke2APIClient provides API access to the second spoke cluster.
	Spoke2APIClient *clients.Settings
	// RANConfig provides access to configuration.
	RANConfig *ranconfig.RANConfig
)

func init() {
	Spoke1APIClient = inittools.APIClient
	RANConfig = ranconfig.NewRANConfig()

	if RANConfig.HubKubeconfig != "" {
		HubAPIClient = clients.New(RANConfig.HubKubeconfig)
	}

	if RANConfig.Spoke2Kubeconfig != "" {
		Spoke2APIClient = clients.New(RANConfig.Spoke2Kubeconfig)
	}
}
