package kmmparams

import "github.com/openshift-kni/eco-gotests/tests/hw-accel/internal/hwaccelparams"

var (
	// Labels represents the range of labels that can be used for test cases selection.
	Labels = []string{hwaccelparams.Label, Label}

	// KmmHubSelector represents MCM object generic selector.
	KmmHubSelector = map[string]string{"cluster.open-cluster-management.io/clusterset": "default"}
)
