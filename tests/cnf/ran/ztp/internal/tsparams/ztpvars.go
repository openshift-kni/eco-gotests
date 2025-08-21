package tsparams

import (
	"github.com/openshift-kni/k8sreporter"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/ran/internal/ranparam"
	corev1 "k8s.io/api/core/v1"
)

// ArgoCdGitDetails is the details for a single app in ArgoCD.
type ArgoCdGitDetails struct {
	Repo   string
	Branch string
	Path   string
}

var (
	// Labels represents the range of labels that can be used for test cases selection.
	Labels = append(ranparam.Labels, LabelSuite)

	// ReporterNamespacesToDump tells to the reporter from where to collect logs.
	ReporterNamespacesToDump = map[string]string{
		TestNamespace: "",
	}
	// ReporterCRDsToDump tells to the reporter what CRs to dump.
	ReporterCRDsToDump = []k8sreporter.CRData{
		{Cr: &corev1.PodList{}},
	}

	// ArgoCdApps is the slice of the Argo CD app names defined in this package.
	ArgoCdApps = []string{
		ArgoCdClustersAppName,
		ArgoCdPoliciesAppName,
	}
	// ArgoCdAppDetails contains more details for each of the ArgoCdApps.
	ArgoCdAppDetails = map[string]ArgoCdGitDetails{}
)
