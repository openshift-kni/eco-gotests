package inittools

import (
	"flag"

	"github.com/openshift-kni/eco-gotests/pkg/clients"
	"github.com/openshift-kni/eco-gotests/tests/internal/config"
)

var (
	// APIClient provides access to cluster.
	APIClient *clients.Settings
	// GeneralConfig provides access to general configuration parameters.
	GeneralConfig *config.General
)

// init loads all variables automatically when this package is imported. Once package is imported a user has full
// access to all vars within init function. It is recommended to import this package using dot import.
func init() {
	if APIClient := clients.New(""); APIClient == nil {
		panic("can not load ApiClient. Please check your KUBECONFIG env var")
	}

	GeneralConfig = config.NewConfig()

	if GeneralConfig == nil {
		panic("error to load general config")
	}

	// Init glong lib and dump logs to stdout
	_ = flag.Lookup("logtostderr").Value.Set("true")
	_ = flag.Lookup("v").Value.Set(GeneralConfig.VerboseLevel)
	flag.Parse()
}
