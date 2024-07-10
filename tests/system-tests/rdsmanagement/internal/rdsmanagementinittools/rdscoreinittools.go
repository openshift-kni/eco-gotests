package rdsmanagementinittools

import (
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-gotests/tests/internal/inittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/rdsmanagement/internal/rdsmanagementconfig"
)

var (
	// APIClient provides API access to cluster.
	APIClient *clients.Settings
	// RDSManagementConfig provides access to RDS Management tests configuration parameters.
	RDSManagementConfig *rdsmanagementconfig.ManagementConfig
)

// init loads all variables automatically when this package is imported. Once package is imported a user has full
// access to all vars within init function. It is recommended to import this package using dot import.
func init() {
	RDSManagementConfig = rdsmanagementconfig.NewManagementConfig()
	APIClient = inittools.APIClient
}
