package upgradeinittools

import (
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-gotests/tests/internal/config"
	"github.com/openshift-kni/eco-gotests/tests/internal/inittools"
)

var (
	// HubAPIClient provides API access to hub cluster.
	HubAPIClient *clients.Settings
	// GeneralConfig provides access to general configuration parameters.
	GeneralConfig *config.GeneralConfig
)

func init() {
	HubAPIClient = inittools.APIClient
	GeneralConfig = inittools.GeneralConfig
}
