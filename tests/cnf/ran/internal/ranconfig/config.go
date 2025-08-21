package ranconfig

import (
	"github.com/golang/glog"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/internal/cnfconfig"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/ran/internal/ranparam"
)

// RANConfig contains configuration for the RAN directory.
type RANConfig struct {
	*cnfconfig.CNFConfig
}

// NewRANConfig returns an instance of RANConfig.
func NewRANConfig() *RANConfig {
	glog.V(ranparam.LogLevel).Infof("Creating new RANConfig struct")

	var ranConfig RANConfig
	ranConfig.CNFConfig = cnfconfig.NewCNFConfig()

	return &ranConfig
}
