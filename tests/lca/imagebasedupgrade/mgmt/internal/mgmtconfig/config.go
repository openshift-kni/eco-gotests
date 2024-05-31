package mgmtconfig

import (
	"github.com/golang/glog"
	"github.com/kelseyhightower/envconfig"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/internal/ibuconfig"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/internal/seedimage"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/mgmt/internal/mgmtparams"
)

// MGMTConfig type contains mgmt configuration.
//
//nolint:lll
type MGMTConfig struct {
	*ibuconfig.IBUConfig
	SeedImage            string `envconfig:"ECO_LCA_IBU_MGMT_SEED_IMAGE" default:"quay.io/ocp-edge-qe/ib-seedimage-public:ci"`
	SeedClusterInfo      *seedimage.SeedImageContent
	IBUWorkloadImage     string `envconfig:"ECO_LCA_IBU_MGMT_WORKLOAD_IMAGE" default:"registry.redhat.io/openshift4/ose-hello-openshift-rhel8@sha256:10dca31348f07e1bfb56ee93c324525cceefe27cb7076b23e42ac181e4d1863e"`
	IdlePostUpgrade      bool   `envconfig:"ECO_LCA_IBU_MGMT_IDLE_POST_UPGRADE" default:"false"`
	RollbackAfterUpgrade bool   `envconfig:"ECO_LCA_IBU_MGMT_ROLLBACK_AFTER_UPGRADE" default:"false"`
	ExtraManifests       bool   `envconfig:"ECO_LCA_IBU_MGMT_EXTRA_MANIFESTS" default:"true"`
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
