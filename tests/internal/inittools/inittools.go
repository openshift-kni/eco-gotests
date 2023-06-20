package inittools

import (
	"flag"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-gotests/tests/internal/config"
)

var (
	// APIClient provides access to cluster.
	APIClient *clients.Settings
	// GeneralConfig provides access to general configuration parameters.
	GeneralConfig *config.GeneralConfig
)

// init loads all variables automatically when this package is imported. Once package is imported a user has full
// access to all vars within init function. It is recommended to import this package using dot import.
func init() {
	if GeneralConfig = config.NewConfig(); GeneralConfig == nil {
		glog.Fatalf("error to load general config")
	}

	_ = flag.Lookup("logtostderr").Value.Set("true")
	_ = flag.Lookup("v").Value.Set(GeneralConfig.VerboseLevel)

	if APIClient = clients.New(""); APIClient == nil {
		if GeneralConfig.DryRun {
			return
		}

		glog.Fatalf("can not load ApiClient. Please check your KUBECONFIG env var")
	}
}
