package ztpconfig

import (
	"fmt"
	"log"

	"github.com/kelseyhightower/envconfig"
	"github.com/openshift-kni/eco-gotests/pkg/clients"
	"github.com/openshift-kni/eco-gotests/tests/assisted/internal/assistedconfig"
)

// ZTPConfig type contains ztp configuration.
type ZTPConfig struct {
	*assistedconfig.AssistedConfig
}

// NewZTPConfig returns instance of ZTPConfig type.
func NewZTPConfig() *ZTPConfig {
	log.Print("Creating new ZTPConfig struct")

	var ztpconfig ZTPConfig
	ztpconfig.AssistedConfig = assistedconfig.NewAssistedConfig()

	return &ztpconfig
}

// SpokeConfig contains environment information related to the spoke cluster.
type SpokeConfig struct {
	APIClient       *clients.Settings
	KubeConfig      string `envconfig:"ECO_ASSISTED_ZTP_SPOKE_KUBECONFIG"`
	ClusterImageSet string `envconfig:"ECO_ASSISTED_ZTP_SPOKE_CLUSTERIMAGESET"`
}

// GetAPIClient implements the cluster.APIClientGetter interface by returning it's APIClient member.
func (config *SpokeConfig) GetAPIClient() (*clients.Settings, error) {
	if config.APIClient == nil {
		return nil, fmt.Errorf("APIClient was nil")
	}

	return config.APIClient, nil
}

// NewSpokeConfig returns instance of SpokeConfig type.
func NewSpokeConfig() *SpokeConfig {
	log.Print("Creating new SpokeConfig struct")

	spokeConfig := new(SpokeConfig)

	err := envconfig.Process("eco_assisted_ztp_spoke_", spokeConfig)
	if err != nil {
		log.Printf("falied to instantiate SpokeConfig: %v", err)

		return nil
	}

	if spokeConfig.KubeConfig != "" {
		log.Printf("Creating spoke api client from %s\n", spokeConfig.KubeConfig)

		if spokeConfig.APIClient = clients.New(
			spokeConfig.KubeConfig); spokeConfig.APIClient == nil {
			log.Printf("falied to load provided spoke kubeconfig: %v", err)
		}
	} else {
		spokeConfig.APIClient = nil
	}

	return spokeConfig
}
