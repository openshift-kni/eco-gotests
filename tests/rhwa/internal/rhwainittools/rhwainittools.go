package rhwainittools

import (
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-gotests/tests/internal/inittools"
	"github.com/openshift-kni/eco-gotests/tests/rhwa/internal/rhwaconfig"
)

var (
	// APIClient provides API access to cluster.
	APIClient *clients.Settings
	// RHWAConfig provides access to general configuration parameters.
	RHWAConfig *rhwaconfig.RHWAConfig
)

// init loads all variables automatically when this package is imported. Once package is imported a user has full
// access to all vars within init function. It is recommended to import this package using dot import.
func init() {
	RHWAConfig = rhwaconfig.NewRHWAConfig()
	APIClient = inittools.APIClient
}
