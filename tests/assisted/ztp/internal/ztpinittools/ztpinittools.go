package ztpinittools

import (
	"github.com/openshift-kni/eco-gotests/pkg/clients"
	"github.com/openshift-kni/eco-gotests/tests/assisted/ztp/internal/ztpconfig"
	"github.com/openshift-kni/eco-gotests/tests/internal/inittools"
)

var (
	// APIClient provides API access to cluster.
	APIClient *clients.Settings
	// ZTPConfig provides access to general configuration parameters.
	ZTPConfig *ztpconfig.ZTPConfig
)

func init() {
	APIClient = inittools.APIClient
	ZTPConfig = ztpconfig.NewZTPConfig()
}
