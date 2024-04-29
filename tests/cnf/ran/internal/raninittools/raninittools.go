package raninittools

import (
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/ranconfig"
	"github.com/openshift-kni/eco-gotests/tests/internal/inittools"
)

var (
	// APIClient provides API access to cluster.
	APIClient *clients.Settings
	// Spoke1APIClient provides API access to the first spoke cluster.
	Spoke1APIClient *clients.Settings
	// Spoke2APIClient provides API access to the second spoke cluster.
	Spoke2APIClient *clients.Settings
	// RANConfig provides access to configuration.
	RANConfig *ranconfig.RANConfig
)

func init() {
	APIClient = inittools.APIClient
	RANConfig = ranconfig.NewRANConfig()

	if RANConfig.Spoke1Kubeconfig != "" {
		Spoke1APIClient = clients.New(RANConfig.Spoke1Kubeconfig)
	}

	if RANConfig.Spoke2Kubeconfig != "" {
		Spoke2APIClient = clients.New(RANConfig.Spoke2Kubeconfig)
	}
}
