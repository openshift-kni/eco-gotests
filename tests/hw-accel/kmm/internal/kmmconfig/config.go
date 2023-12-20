package kmmconfig

import (
	"log"

	"github.com/kelseyhightower/envconfig"
)

// ModulesConfig contains environment information related to kmm tests.
type ModulesConfig struct {
	PullSecret           string `envconfig:"ECO_HWACCEL_KMM_PULL_SECRET"`
	Registry             string `envconfig:"ECO_HWACCEL_KMM_REGISTRY"`
	SubscriptionName     string `envconfig:"ECO_HWACCEL_KMM_SUBSCRIPTION_NAME"`
	CatalogSourceName    string `envconfig:"ECO_HWACCEL_KMM_CATALOG_SOURCE_NAME"`
	UpgradeTargetVersion string `envconfig:"ECO_HWACCEL_KMM_UPGRADE_TARGET_VERSION"`
}

// NewModulesConfig returns instance of ModulesConfig type.
func NewModulesConfig() *ModulesConfig {
	log.Print("Creating new ModulesConfig")

	modulesConfig := new(ModulesConfig)

	err := envconfig.Process("eco_hwaccel_kmm_", modulesConfig)
	if err != nil {
		log.Printf("failed to instantiate ModulesConfig: %v", err)

		return nil
	}

	return modulesConfig
}
