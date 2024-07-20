package vcore_system_test

import (
	. "github.com/onsi/ginkgo/v2"

	"github.com/openshift-kni/eco-gotests/tests/system-tests/vcore/internal/vcorecommon"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/vcore/internal/vcoreparams"
)

var _ = Describe(
	"vCore Operators Deployment Suite",
	Ordered,
	ContinueOnFailure,
	Label(vcoreparams.Label), func() {
		vcorecommon.VerifyNMStateSuite()

		vcorecommon.VerifyServiceMeshSuite()

		vcorecommon.VerifyHelmSuite()

		vcorecommon.VerifyRedisSuite()

		vcorecommon.VerifyNTOSuite()

		vcorecommon.VerifySRIOVSuite()

		vcorecommon.VerifyKedaSuite()

		vcorecommon.VerifyNROPSuite()
	})
