package accelconfig

import (
	"log"

	"github.com/golang/glog"
	"github.com/kelseyhightower/envconfig"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-gotests/tests/internal/config"
)

// AccelConfig contains environment information related to ocp upgrade tests.
type AccelConfig struct {
	PullSecret           string `envconfig:"ECO_ACCEL_PULL_SECRET"`
	Registry             string `envconfig:"ECO_ACCEL_REGISTRY"`
	UpgradeTargetVersion string `envconfig:"ECO_ACCEL_UPGRADE_TARGET_IMAGE"`
	SpokeKubeConfig      string `envconfig:"ECO_ACCEL_SPOKE_KUBECONFIG"`
	HubClusterName       string `envconfig:"ECO_ACCEL_HUB_CLUSTER_NAME"`
	HubMinorVersion      string `envconfig:"ECO_ACCEL_HUB_MINOR_VERSION"`
	SpokeAPIClient       *clients.Settings
	*config.GeneralConfig
}

// NewAccelConfig returns instance of AccelConfig type.
func NewAccelConfig() *AccelConfig {
	log.Print("Creating new AccelConfig")

	var accelConfig AccelConfig
	accelConfig.GeneralConfig = config.NewConfig()

	err := envconfig.Process("eco_accel_", &accelConfig)
	if err != nil {
		log.Printf("failed to instantiate AccelConfig: %v", err)

		return nil
	}

	if accelConfig.SpokeKubeConfig != "" {
		glog.V(90).Infof("Creating spoke api client from %s", accelConfig.SpokeKubeConfig)

		if accelConfig.SpokeAPIClient = clients.New(
			accelConfig.SpokeKubeConfig); accelConfig.SpokeAPIClient == nil {
			glog.V(90).Infof("failed to load provided spoke kubeconfig")
		}
	}

	return &accelConfig
}
