package tsparams

import (
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/internal/kmmparams"
	"github.com/openshift-kni/k8sreporter"
	moduleV1Beta1 "github.com/rh-ecosystem-edge/kernel-module-management/api/v1beta1"
)

var (
	// Labels represents the range of labels that can be used for test cases selection.
	Labels = append(kmmparams.Labels, LabelSuite)

	// LocalImageRegistry represents the local registry used in KMM tests.
	LocalImageRegistry = "image-registry.openshift-image-registry.svc:5000"

	// ReporterNamespacesToDump tells to the reporter from where to collect logs.
	ReporterNamespacesToDump = map[string]string{
		"openshift-kmm":     "kmm",
		"openshift-kmm-hub": "kmm-hub",
		ModuleTestNamespace: "other",
	}

	// ReporterCRDsToDump tells to the reporter what CRs to dump.
	ReporterCRDsToDump = []k8sreporter.CRData{
		{Cr: &moduleV1Beta1.ModuleList{}},
	}
)
