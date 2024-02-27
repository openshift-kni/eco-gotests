package rds_core_system_test

import (
	. "github.com/onsi/ginkgo/v2"

	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"

	"github.com/openshift-kni/eco-gotests/tests/system-tests/rdscore/internal/rdscorecommon"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/rdscore/internal/rdscoreparams"
)

var _ = Describe(
	"SR-IOV verification",
	Ordered,
	ContinueOnFailure,
	Label(rdscoreparams.LabelValidateSRIOV), func() {
		It("Verifices SR-IOV workloads on the same node",
			Label("sriov-same-node"), polarion.ID("71949"), MustPassRepeatedly(3),
			rdscorecommon.VerifySRIOVWorkloadsOnSameNode)

		It("Verifices SR-IOV workloads on different nodes",
			Label("sriov-different-node"), polarion.ID("71950"), MustPassRepeatedly(3),
			rdscorecommon.VerifySRIOVWorkloadsOnDifferentNodes)
	})
