package raninittools

import (
	"github.com/openshift-kni/eco-goinfra/pkg/bmc"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran-deployment/internal/ranconfig"
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
	// BMCClient provides access to the BMC. Nil when BMC configs are not provided.
	BMCClient *bmc.BMC
)

func init() {
	RANConfig = ranconfig.NewRANConfig()
	HubAPIClient = RANConfig.HubAPIClient
	Spoke1APIClient = RANConfig.Spoke1APIClient
	Spoke2APIClient = RANConfig.Spoke2APIClient
}
