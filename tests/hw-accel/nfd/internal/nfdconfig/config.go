package nfdconfig

import (
	"log"

	"github.com/kelseyhightower/envconfig"
)

// NfdConfig contains environment information related to nfd tests.
type NfdConfig struct {
	SubscriptionName     string `envconfig:"ECO_HWACCEL_NFD_SUBSCRIPTION_NAME"`
	Image                string `envconfig:"ECO_HWACCEL_NFD_CR_IMAGE"`
	CatalogSource        string `envconfig:"ECO_HWACCEL_NFD_CATALOG_SOURCE"`
	CustomCatalogSource  string `envconfig:"ECO_HWACCEL_NFD_CUSTOM_NFD_CATALOG_SOURCE"`
	AwsTest              bool   `envconfig:"ECO_HWACCEL_NFD_AWS_TESTS"`
	UpgradeTargetVersion string `envconfig:"ECO_HWACCEL_NFD_UPGRADE_TARGET_VERSION"`
	CPUFlagsHelperImage  string `envconfig:"ECO_HWACCEL_NFD_CPU_FLAGS_HELPER_IMAGE"`
}

// NewNfdConfig returns instance of NfdConfig type.
func NewNfdConfig() *NfdConfig {
	log.Print("Creating new NfdConfig")

	nfdConfig := new(NfdConfig)

	err := envconfig.Process("eco_hwaccel_nfd_", nfdConfig)
	if err != nil {
		log.Printf("failed to instantiate NfdConfig: %v", err)

		return nil
	}

	return nfdConfig
}
