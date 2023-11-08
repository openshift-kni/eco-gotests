package ecoreinittools

import (
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-gotests/tests/internal/inittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/ecore/internal/ecoreconfig"
)

var (
	// APIClient provides API access to cluster.
	APIClient *clients.Settings
	// ECoreConfig provides access to ENAD system tests configuration parameters.
	ECoreConfig *ecoreconfig.ECoreConfig
)

// init loads all variables automatically when this package is imported. Once package is imported a user has full
// access to all vars within init function. It is recommended to import this package using dot import.
func init() {
	ECoreConfig = ecoreconfig.NewECoreConfig()
	APIClient = inittools.APIClient
}
