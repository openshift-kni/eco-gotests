package accelinittools

import (
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-gotests/tests/accel/internal/accelconfig"
	"github.com/openshift-kni/eco-gotests/tests/internal/inittools"
)

var (
	// HubAPIClient provides API access to hub cluster.
	HubAPIClient *clients.Settings
	// SpokeAPIClient provides API access to spoke cluster.
	SpokeAPIClient *clients.Settings
	// AccelConfig provides access to configuration parameters.
	AccelConfig *accelconfig.AccelConfig
)

func init() {
	HubAPIClient = inittools.APIClient
	AccelConfig = accelconfig.NewAccelConfig()
	SpokeAPIClient = AccelConfig.SpokeAPIClient
}
