package telcoconfig

import (
	"github.com/golang/glog"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/lca/imagebasedupgrade/internal/ibuconfig"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/lca/imagebasedupgrade/telco/internal/telcoparams"
)

// TelcoConfig type contains telco configuration.
type TelcoConfig struct {
	*ibuconfig.IBUConfig
}

// NewTelcoConfig returns instance of TelcoConfig type.
func NewTelcoConfig() *TelcoConfig {
	glog.V(telcoparams.TelcoLogLevel).Info("Creating new TelcoConfig struct")

	var telcoConfig TelcoConfig
	telcoConfig.IBUConfig = ibuconfig.NewIBUConfig()

	return &telcoConfig
}
