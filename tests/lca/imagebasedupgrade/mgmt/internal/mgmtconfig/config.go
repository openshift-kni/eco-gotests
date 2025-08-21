package mgmtconfig

import (
	"github.com/golang/glog"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/lca/imagebasedupgrade/internal/ibuconfig"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/lca/imagebasedupgrade/mgmt/internal/mgmtparams"
)

// MGMTConfig type contains mgmt configuration.
type MGMTConfig struct {
	*ibuconfig.IBUConfig
}

// NewMGMTConfig returns instance of MGMTConfig type.
func NewMGMTConfig() *MGMTConfig {
	glog.V(mgmtparams.MGMTLogLevel).Info("Creating new MGMTConfig struct")

	var mgmtConfig MGMTConfig
	mgmtConfig.IBUConfig = ibuconfig.NewIBUConfig()

	return &mgmtConfig
}
