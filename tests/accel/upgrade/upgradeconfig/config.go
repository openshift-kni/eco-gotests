package upgradeconfig

import (
	"log"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"

	"github.com/kelseyhightower/envconfig"
)

// ModulesConfig contains environment information related to ocp upgrade tests.
type ModulesConfig struct {
	PullSecret           string `envconfig:"ECO_ACCEL_UPGRADE_PULL_SECRET"`
	Registry             string `envconfig:"ECO_ACCEL_UPGRADE_REGISTRY"`
	UpgradeTargetVersion string `envconfig:"ECO_ACCEL_UPGRADE_UPGRADE_TARGET_IMAGE"`
	HubKubeConfig        string `envconfig:"ECO_ACCEL_UPGRADE_HUB_KUBECONFIG"`
	HubClusterName       string `envconfig:"ECO_ACCEL_UPGRADE_HUB_CLUSTER_NAME"`
	HubAPIClient         *clients.Settings
}

// NewModulesConfig returns instance of ModulesConfig type.
func NewModulesConfig() *ModulesConfig {
	log.Print("Creating new ModulesConfig")

	modulesConfig := new(ModulesConfig)

	err := envconfig.Process("eco_accel_upgrade_", modulesConfig)
	if err != nil {
		log.Printf("failed to instantiate ModulesConfig: %v", err)

		return nil
	}

	if modulesConfig.HubKubeConfig != "" {
		glog.V(90).Infof("Creating spoke api client from %s", modulesConfig.HubKubeConfig)

		if modulesConfig.HubAPIClient = clients.New(
			modulesConfig.HubKubeConfig); modulesConfig.HubAPIClient == nil {
			glog.V(90).Infof("failed to load provided hub kubeconfig: %v", err)
		}
	}

	return modulesConfig
}
