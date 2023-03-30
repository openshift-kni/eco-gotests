package ztpconfig

import (
	"log"

	"github.com/kelseyhightower/envconfig"
	"github.com/openshift-kni/eco-gotests/tests/assisted/internal/assistedconfig"
)

// ZTPConfig type contains ztp configuration.
type ZTPConfig struct {
	*assistedconfig.AssistedConfig
	HubConfig   hubConfig
	SpokeConfig spokeConfig
}

// hubConfig contains environment information related to the hub cluster.
type hubConfig struct {
	OCPVersion      float32 `envconfig:"ECO_ASSISTED_ZTP_HUB_OCP_VERSION"`
	OperatorVersion float32 `envconfig:"ECO_ASSISTED_ZTP_HUB_ACM_VERSION"`
	IPStackv4       bool    `envconfig:"ECO_ASSISTED_ZTP_HUB_BAREMETAL_NET_IPV4"`
	IPStackv6       bool    `envconfig:"ECO_ASSISTED_ZTP_HUB_BAREMETAL_NET_IPV6"`
	Disconnected    bool    `envconfig:"ECO_ASSISTED_ZTP_HUB_DISCONNECTED_INSTALL"`
}

// spokeConfig contains environment information related to the spoke cluster.
type spokeConfig struct {
	OCPVersion float32 `envconfig:"ECO_ASSISTED_ZTP_SPOKE_OCP_VERSION"`
	PullSecret string  `envconfig:"ECO_ASSISTED_ZTP_SPOKE_PULL_SECRET"`
}

// NewZTPConfig returns instance of ZTPConfig type.
func NewZTPConfig() *ZTPConfig {
	log.Print("Creating new ZTPConfig struct")

	var ztpconfig ZTPConfig
	ztpconfig.AssistedConfig = assistedconfig.NewAssistedConfig()

	err := envconfig.Process("eco_assisted_ztp_hub_", &ztpconfig.HubConfig)
	if err != nil {
		log.Printf("falied to instantiate HubConfig: %v", err)

		return nil
	}

	err = envconfig.Process("eco_assisted_ztp_spoke_", &ztpconfig.SpokeConfig)
	if err != nil {
		log.Printf("falied to instantiate SpokeConfig: %v", err)

		return nil
	}

	return &ztpconfig
}
