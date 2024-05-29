package raninittools

import (
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/bmc"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/ranconfig"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/ranparam"
	"github.com/openshift-kni/eco-gotests/tests/internal/inittools"
)

var (
	// HubAPIClient provides API access to the first spoke cluster.
	HubAPIClient *clients.Settings
	// Spoke1APIClient provides API access to cluster.
	Spoke1APIClient *clients.Settings
	// Spoke2APIClient provides API access to the second spoke cluster.
	Spoke2APIClient *clients.Settings
	// RANConfig provides access to configuration.
	RANConfig *ranconfig.RANConfig
	// BMCClient provides access to the BMC. Nil when BMC configs are not provided.
	BMCClient *bmc.BMC
)

func init() {
	Spoke1APIClient = inittools.APIClient
	RANConfig = ranconfig.NewRANConfig()

	if RANConfig.HubKubeconfig != "" {
		HubAPIClient = clients.New(RANConfig.HubKubeconfig)
	}

	if RANConfig.Spoke2Kubeconfig != "" {
		Spoke2APIClient = clients.New(RANConfig.Spoke2Kubeconfig)
	}

	// If all of the BMC configuration is valid, setup the BMC client.
	hosts := strings.Split(RANConfig.BmcHosts, ",")
	bmcTimeout, err := time.ParseDuration(RANConfig.BmcTimeout)

	if hosts[0] != "" &&
		RANConfig.BmcUsername != "" &&
		RANConfig.BmcPassword != "" && err == nil {
		if len(hosts) > 1 {
			glog.V(ranparam.LogLevel).Infof("More than one BMC host found, using the first host '%s'", hosts[0])
		}

		BMCClient = bmc.New(hosts[0]).
			WithRedfishUser(RANConfig.BmcUsername, RANConfig.BmcPassword).
			WithRedfishTimeout(bmcTimeout)
	}
}
