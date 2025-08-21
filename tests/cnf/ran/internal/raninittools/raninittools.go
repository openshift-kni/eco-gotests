package raninittools

import (
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/clients"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/ran/internal/ranconfig"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/internal/inittools"
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
