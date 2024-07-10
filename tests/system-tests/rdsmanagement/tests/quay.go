package management_system_test

import (
	. "github.com/onsi/ginkgo/v2"

	"github.com/openshift-kni/eco-gotests/tests/system-tests/management/internal/managementcommon"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/management/internal/managementparams"
)

var _ = Describe(
	"Quay Suite",
	Ordered,
	ContinueOnFailure,
	Label(managementparams.Label), func() {
		// Add tests here
	})
