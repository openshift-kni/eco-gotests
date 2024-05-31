package netparam

import "github.com/openshift-kni/eco-gotests/tests/cnf/core/internal/coreparams"

var (
	// Labels represents the range of labels that can be used for test cases selection.
	Labels = append(coreparams.Labels, Label)
)
