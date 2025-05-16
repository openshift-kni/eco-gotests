package cnfhelper

import (
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedinstall/cnf/internal/cnfinittools"
)

// GetAllTestClients is used to quickly obtain a list of all the test clients.
func GetAllTestClients() []*clients.Settings {
	return []*clients.Settings{
		cnfinittools.TargetHubAPIClient,
		cnfinittools.TargetSNOAPIClient,
	}
}
