package samsunginittools

import (
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-gotests/tests/internal/inittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/samsung-vcore/internal/samsungconfig"
)

var (
	// APIClient provides API access to cluster.
	APIClient *clients.Settings
	// SamsungConfig provides access to SPK system tests configuration parameters.
	SamsungConfig *samsungconfig.SamsungConfig
)

// init loads all variables automatically when this package is imported. Once package is imported a user has full
// access to all vars within init function. It is recommended to import this package using dot import.
func init() {
	SamsungConfig = samsungconfig.NewSamsungConfig()
	APIClient = inittools.APIClient
}
