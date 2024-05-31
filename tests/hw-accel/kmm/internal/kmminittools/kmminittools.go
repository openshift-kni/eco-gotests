package kmminittools

import "github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/internal/kmmconfig"

var (
	// ModulesConfig provides access to general configuration parameters.
	ModulesConfig *kmmconfig.ModulesConfig
)

func init() {
	ModulesConfig = kmmconfig.NewModulesConfig()
}
