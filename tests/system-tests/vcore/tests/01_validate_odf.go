package vcore_system_test

import (
	. "github.com/onsi/ginkgo/v2"

	"github.com/openshift-kni/eco-gotests/tests/system-tests/vcore/internal/vcorecommon"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/vcore/internal/vcoreparams"
)

var _ = Describe(
	"vCore Internal ODF Test Suite",
	Ordered,
	ContinueOnFailure,
	Label(vcoreparams.Label), func() {
		vcorecommon.VerifyLSOSuite()

		vcorecommon.VerifyODFSuite()

		vcorecommon.VerifyLokiSuite()

	})
