package kmmconfig

import (
	"log"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/internal/kmmparams"

	"github.com/kelseyhightower/envconfig"
)

// ModulesConfig contains environment information related to kmm tests.
type ModulesConfig struct {
	PullSecret           string `envconfig:"ECO_HWACCEL_KMM_PULL_SECRET"`
	Registry             string `envconfig:"ECO_HWACCEL_KMM_REGISTRY"`
	DevicePluginImage    string `envconfig:"ECO_HWACCEL_KMM_DEVICE_PLUGIN_IMAGE"`
	SubscriptionName     string `envconfig:"ECO_HWACCEL_KMM_SUBSCRIPTION_NAME"`
	CatalogSourceName    string `envconfig:"ECO_HWACCEL_KMM_CATALOG_SOURCE_NAME"`
	UpgradeTargetVersion string `envconfig:"ECO_HWACCEL_KMM_UPGRADE_TARGET_VERSION"`
	SpokeKubeConfig      string `envconfig:"ECO_HWACCEL_KMM_SPOKE_KUBECONFIG"`
	SpokeClusterName     string `envconfig:"ECO_HWACCEL_KMM_SPOKE_CLUSTER_NAME"`
	SpokeAPIClient       *clients.Settings
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

	if modulesConfig.SpokeKubeConfig != "" {
		glog.V(kmmparams.KmmLogLevel).Infof("Creating spoke api client from %s", modulesConfig.SpokeKubeConfig)

		if modulesConfig.SpokeAPIClient = clients.New(
			modulesConfig.SpokeKubeConfig); modulesConfig.SpokeAPIClient == nil {
			glog.V(kmmparams.KmmLogLevel).Infof("failed to load provided spoke kubeconfig: %v", err)
		}
	}

	return modulesConfig
}
