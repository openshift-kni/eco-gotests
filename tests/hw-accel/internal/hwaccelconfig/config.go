package hwaccelconfig

import (
	"log"

	"github.com/openshift-kni/eco-gotests/tests/internal/config"
)

// HwAccelConfig contains Hardware Accelerators configuration.
type HwAccelConfig struct {
	*config.GeneralConfig
}

// NewHwAccelConfig returns instance of HwAccelConfig.
func NewHwAccelConfig() *HwAccelConfig {
	log.Print("Creating new HwAccelConfig struct")

	var hwaccelConfig HwAccelConfig

	hwaccelConfig.GeneralConfig = config.NewConfig()

	return &hwaccelConfig
}
