package rds_core_system_test

import (
	. "github.com/onsi/ginkgo/v2"

	"github.com/rh-ecosystem-edge/eco-gotests/tests/system-tests/rdscore/internal/rdscorecommon"
)

var _ = Describe(
	"RDS Core Top Level Suite",
	Ordered,
	ContinueOnFailure,
	Label("rds-core-workflow"), func() {
		rdscorecommon.VerifySRIOVSuite()

		rdscorecommon.VerifyNMStateSuite()

		rdscorecommon.VefityPersistentStorageSuite()

		rdscorecommon.VerifyHardRebootSuite()

		rdscorecommon.VerifyGracefulRebootSuite()
	})
