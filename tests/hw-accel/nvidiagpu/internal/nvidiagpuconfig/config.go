package nvidiagpuconfig

import (
	"log"

	"github.com/kelseyhightower/envconfig"
)

// NvidiaGPUConfig contains environment information related to nvidiagpu tests.
type NvidiaGPUConfig struct {
	GPUBurnImage string `envconfig:"ECO_HWACCEL_NVIDIAGPU_GPUBURN_IMAGE"`
}

// NewNvidiaGPUConfig returns instance of NvidiaGPUConfig type.
func NewNvidiaGPUConfig() *NvidiaGPUConfig {
	log.Print("Creating new NvidiaGPUConfig")

	nvidiaGPUConfig := new(NvidiaGPUConfig)

	err := envconfig.Process("eco_hwaccel_nvidiagpu_", nvidiaGPUConfig)
	if err != nil {
		log.Printf("failed to instantiate nvidiaGPUConfig: %v", err)

		return nil
	}

	return nvidiaGPUConfig
}
