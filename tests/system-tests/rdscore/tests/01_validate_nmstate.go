package rds_core_system_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"

	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"

	"github.com/openshift-kni/eco-gotests/tests/system-tests/rdscore/internal/rdscorecommon"
	. "github.com/openshift-kni/eco-gotests/tests/system-tests/rdscore/internal/rdscoreinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/rdscore/internal/rdscoreparams"
)

var _ = Describe(
	"NMState validation",
	Ordered,
	ContinueOnFailure,
	Label(rdscoreparams.LabelValidateNMState), func() {
		It(fmt.Sprintf("Verifies %s namespace exists", RDSCoreConfig.NMStateOperatorNamespace),
			Label("nmstate-ns"), rdscorecommon.VerifyNMStateNamespaceExists)

		It("Verifies NMState instance exists",
			Label("nmstate-instance"), polarion.ID("67027"), rdscorecommon.VerifyNMStateInstanceExists)

		It("Verifies all NodeNetworkConfigurationPolicies are Available",
			Label("nmstate-nncp"), polarion.ID("71846"), rdscorecommon.VerifyAllNNCPsAreOK)
	})
