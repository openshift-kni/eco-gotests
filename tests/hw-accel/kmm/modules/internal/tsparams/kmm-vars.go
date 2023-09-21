package tsparams

import (
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/internal/kmmparams"
	"github.com/openshift-kni/k8sreporter"
	moduleV1Beta1 "github.com/rh-ecosystem-edge/kernel-module-management/api/v1beta1"
	v1 "k8s.io/api/core/v1"
)

var (
	// Labels represents the range of labels that can be used for test cases selection.
	Labels = append(kmmparams.Labels, LabelSuite)

	// LocalImageRegistry represents the local registry used in KMM tests.
	LocalImageRegistry = "image-registry.openshift-image-registry.svc:5000"

	// ReporterNamespacesToDump tells to the reporter from where to collect logs.
	ReporterNamespacesToDump = map[string]string{
		KmmOperatorNamespace:          "kmm",
		UseDtkModuleTestNamespace:     "module",
		SimpleKmodModuleTestNamespace: "module",
		DevicePluginTestNamespace:     "module",
		RealtimeKernelNamespace:       "module",
		FirmwareTestNamespace:         "module",
		ModuleBuildAndSignNamespace:   "module",
	}

	// ReporterCRDsToDump tells to the reporter what CRs to dump.
	ReporterCRDsToDump = []k8sreporter.CRData{
		{Cr: &moduleV1Beta1.ModuleList{}},
	}

	// DTKImage represents Driver Toolkit image in internal image registry.
	DTKImage       = "image-registry.openshift-image-registry.svc:5000/openshift/driver-toolkit"
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

	// KmmTestHelperLabelName represents label set on the helper resources.
	KmmTestHelperLabelName = "kmm-test-helper"
)
