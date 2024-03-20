package raninittools

import (
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/ranconfig"
	"github.com/openshift-kni/eco-gotests/tests/internal/inittools"
)

var (
	// APIClient provides API access to cluster.
	APIClient *clients.Settings
	// RANConfig provides access to configuration.
	RANConfig *ranconfig.RANConfig
)

func init() {
	APIClient = inittools.APIClient
	RANConfig = ranconfig.NewRANConfig()
}
