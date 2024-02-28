package rds_core_system_test

import (
	. "github.com/onsi/ginkgo/v2"

	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"

	"github.com/openshift-kni/eco-gotests/tests/system-tests/rdscore/internal/rdscorecommon"
)

var _ = Describe(
	"Persistent storage validation",
	Ordered,
	ContinueOnFailure,
	Label("rds-core-persistent-storage"), func() {

		It("Verifies CephFS",
			Label("odf-cephfs-pvc"), polarion.ID("71850"), MustPassRepeatedly(3), rdscorecommon.VerifyCephFSPVC)

		It("Verifies CephRBD",
			Label("odf-cephrbd-pvc"), polarion.ID("71989"), MustPassRepeatedly(3), rdscorecommon.VerifyCephRBDPVC)

	})
