package cnfparams

import (
	"time"

	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedinstall/internal/ibiparams"
)

var (
	// Labels represents the range of labels that can be used for test cases selection.
	Labels = []string{ibiparams.Label, Label}
	// WorkloadReadyTimeout represents the timeout for the workload to be ready.
	WorkloadReadyTimeout = 5 * time.Minute
)
