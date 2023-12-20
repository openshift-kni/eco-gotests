package tsparams

import "github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/internal/kmmparams"

var (
	// Labels represents the range of labels that can be used for test cases selection.
	Labels = append(kmmparams.Labels, LabelSuite)
)
