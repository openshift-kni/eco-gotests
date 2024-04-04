package tsparams

import (
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/ranparam"
)

var (
	// Labels represents the range of labels that can be used for test cases selection.
	Labels = append(ranparam.Labels, LabelSuite)
)
