package ztpinittools

import (
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-gotests/tests/assisted/ztp/internal/ztpconfig"
	"github.com/openshift-kni/eco-gotests/tests/internal/inittools"
)

var (
	// HubAPIClient provides API access to hub cluster.
	HubAPIClient *clients.Settings
	// SpokeConfig provides a SpokeConfig for accessing spoke cluster.
	SpokeConfig *ztpconfig.SpokeConfig
	// ZTPConfig provides access to general configuration parameters.
	ZTPConfig *ztpconfig.ZTPConfig
)

func init() {
	HubAPIClient = inittools.APIClient
	SpokeConfig = ztpconfig.NewSpokeConfig()
	ZTPConfig = ztpconfig.NewZTPConfig()
}
