package mgmtinittools

import (
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-gotests/tests/internal/inittools"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedinstall/mgmt/internal/mgmtconfig"
)

var (
	// APIClient provides API access to the cluster.
	APIClient *clients.Settings
	// MGMTConfig provides access to general configuration parameters.
	MGMTConfig *mgmtconfig.MGMTConfig
)

func init() {
	MGMTConfig = mgmtconfig.NewMGMTConfig()
	APIClient = inittools.APIClient
}
