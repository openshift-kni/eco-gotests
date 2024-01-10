package tsparams

import (
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/internal/kmmparams"
	"github.com/openshift-kni/k8sreporter"
	mcmV1Beta1 "github.com/rh-ecosystem-edge/kernel-module-management/api-hub/v1beta1"
)

var (
	// Labels represents the range of labels that can be used for test cases selection.
	Labels = append(kmmparams.Labels, LabelSuite)

	// ReporterNamespacesToDump configures reporter namespaces to dump.
	ReporterNamespacesToDump = map[string]string{
		KmmHubOperatorNamespace: "kmm-hub",
	}

	// ReporterCRDsToDump configures the CRs to dump.
	ReporterCRDsToDump = []k8sreporter.CRData{
		{Cr: &mcmV1Beta1.ManagedClusterModuleList{}},
	}
)
