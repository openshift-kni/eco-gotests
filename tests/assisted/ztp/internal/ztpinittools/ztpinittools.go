package ztpinittools

import (
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-gotests/tests/assisted/ztp/internal/ztpconfig"
	"github.com/openshift-kni/eco-gotests/tests/internal/inittools"
)

var (
	// HubAPIClient provides API access to hub cluster.
	HubAPIClient *clients.Settings
	// SpokeAPIClient provides API access to spoke cluster.
	SpokeAPIClient *clients.Settings
	// ZTPConfig provides access to general configuration parameters.
	ZTPConfig *ztpconfig.ZTPConfig
)

func init() {
	HubAPIClient = inittools.APIClient
	ZTPConfig = ztpconfig.NewZTPConfig()
	SpokeAPIClient = ZTPConfig.SpokeAPIClient
}
