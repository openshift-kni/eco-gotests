package kmmparams

import "github.com/rh-ecosystem-edge/eco-gotests/tests/hw-accel/internal/hwaccelparams"

var (
	// Labels represents the range of labels that can be used for test cases selection.
	Labels = []string{hwaccelparams.Label, Label}

	// LocalImageRegistry represents the local registry used in KMM tests.
	LocalImageRegistry = "image-registry.openshift-image-registry.svc:5000"

	// KmmHubSelector represents MCM object generic selector.
	KmmHubSelector = map[string]string{"cluster.open-cluster-management.io/clusterset": "default"}
)

// KmmTestHelperLabelName represents label set on the helper resources.
var KmmTestHelperLabelName = "kmm-test-helper"
