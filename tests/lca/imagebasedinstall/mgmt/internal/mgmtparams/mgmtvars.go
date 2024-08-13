package mgmtparams

import (
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedinstall/internal/ibiparams"
)

var (
	// Labels represents the range of labels that can be used for test cases selection.
	Labels = append(ibiparams.Labels, Label)
)
