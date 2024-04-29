package tsparams

import (
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/ranparam"
	"github.com/openshift-kni/k8sreporter"
	v1 "k8s.io/api/core/v1"
)

var (
	// Labels represents the range of labels that can be used for test cases selection.
	Labels = append(ranparam.Labels, LabelSuite)

	// ReporterNamespacesToDump tells to the reporter from where to collect logs.
	ReporterNamespacesToDump = map[string]string{
		TestNamespace: "",
	}

	// ReporterCRDsToDump tells to the reporter what CRs to dump.
	ReporterCRDsToDump = []k8sreporter.CRData{
		{Cr: &v1.PodList{}},
	}

	// Spoke1Name is the name of the first spoke cluster.
	Spoke1Name string
	// Spoke2Name is the name of the second spoke cluster.
	Spoke2Name string
	// TalmVersion talm version.
	TalmVersion string
)
