package mgmtconfig

import (
	"github.com/golang/glog"
	"github.com/kelseyhightower/envconfig"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/internal/ibuconfig"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/mgmt/internal/mgmtparams"
)

// MGMTConfig type contains mgmt configuration.
type MGMTConfig struct {
	*ibuconfig.IBUConfig
	SeedImage        string `envconfig:"ECO_LCA_IBU_MGMT_SEED_IMAGE" default:"quay.io/ocp-edge-qe/ib-seedimage-public:ci"`
	SeedImageVersion string `envconfig:"ECO_LCA_IBU_MGMT_SEED_IMAGE_VERSION" default:""`
	IdlePostUpgrade  bool   `envconfig:"ECO_LCA_IBU_MGMT_IDLE_POST_UPGRADE" default:"false"`
}

// NewMGMTConfig returns instance of MGMTConfig type.
func NewMGMTConfig() *MGMTConfig {
	glog.V(mgmtparams.MGMTLogLevel).Info("Creating new MGMTConfig struct")

	var mgmtConfig MGMTConfig
	mgmtConfig.IBUConfig = ibuconfig.NewIBUConfig()

	err := envconfig.Process("eco_lca_ibu_mgmt_", &mgmtConfig)
	if err != nil {
		return nil
	}

	return &mgmtConfig
}
