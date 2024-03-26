package kmmparams

import (
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/internal/hwaccelparams"
	v1 "k8s.io/api/core/v1"
)

var (
	// Labels represents the range of labels that can be used for test cases selection.
	Labels = []string{hwaccelparams.Label, Label}

	// LocalImageRegistry represents the local registry used in KMM tests.
	LocalImageRegistry = "image-registry.openshift-image-registry.svc:5000"

	// KmmHubSelector represents MCM object generic selector.
	KmmHubSelector = map[string]string{"cluster.open-cluster-management.io/clusterset": "default"}

	// KmmTestHelperLabelName represents label set on the helper resources.
	KmmTestHelperLabelName = "kmm-test-helper"

	// DTKImage represents Driver Toolkit image in internal image registry.
	DTKImage = "image-registry.openshift-image-registry.svc:5000/openshift/driver-toolkit"

	trueVar        = true
	capabilityAll  = []v1.Capability{"ALL"}
	defaultGroupID = int64(3000)
	defaultUserID  = int64(2000)

	// PrivilegedSC represents a privileged security context definition.
	PrivilegedSC = &v1.SecurityContext{
		Privileged:     &trueVar,
		RunAsGroup:     &defaultGroupID,
		RunAsUser:      &defaultUserID,
		SeccompProfile: &v1.SeccompProfile{Type: "RuntimeDefault"},
		Capabilities: &v1.Capabilities{
			Add: capabilityAll,
		},
	}

	// ReasonBuildList represents expected events to be found for a successful build job.
	ReasonBuildList = []string{ReasonBuildCreated, ReasonBuildStarted, ReasonBuildCompleted, ReasonBuildSucceeded}
	// ReasonSignList represents expected events to be found for a successful sign job.
	ReasonSignList = []string{ReasonSignCreated, ReasonBuildStarted, ReasonBuildCompleted, ReasonBuildSucceeded}
)
