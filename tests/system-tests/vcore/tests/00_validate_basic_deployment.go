package vcore_system_test

import (
	. "github.com/onsi/ginkgo/v2"

	"github.com/openshift-kni/eco-gotests/tests/system-tests/vcore/internal/vcorecommon"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/vcore/internal/vcoreparams"
)

var _ = Describe(
	"vCore Basic Deployment Suite",
	Ordered,
	ContinueOnFailure,
	Label(vcoreparams.Label), func() {
		vcorecommon.VerifyInitialDeploymentConfig()

		vcorecommon.VerifyCGroupDefault()

		vcorecommon.VerifyPostDeploymentConfig()

		vcorecommon.VerifyNMStateSuite()

		vcorecommon.VerifyMetaLBSuite()

		vcorecommon.VerifyLSOSuite()

		vcorecommon.VerifyODFSuite()

		vcorecommon.VerifyESKAndCLOSuite()

		vcorecommon.VerifyServiceMeshSuite()

		vcorecommon.VerifyHelmSuite()

		vcorecommon.VerifyRedisSuite()
	})
