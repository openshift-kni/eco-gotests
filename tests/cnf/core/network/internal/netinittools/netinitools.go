package netinittools

import (
	"github.com/openshift-kni/eco-gotests/pkg/clients"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netconfig"
	"github.com/openshift-kni/eco-gotests/tests/internal/inittools"
)

var (
	// APIClient provides API access to cluster.
	APIClient *clients.Settings
	// NetConfig provides access to general configuration parameters.
	NetConfig *netconfig.NetworkConfig
)

// init loads all variables automatically when this package is imported. Once package is imported a user has full
// access to all vars within init function. It is recommended to import this package using dot import.
func init() {
	NetConfig = netconfig.NewNetConfig()
	APIClient = inittools.APIClient
}
