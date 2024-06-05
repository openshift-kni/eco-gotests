package tsparams

import (
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/mgmt/internal/mgmtparams"
	"github.com/openshift-kni/k8sreporter"
	lcav1 "github.com/openshift-kni/lifecycle-agent/api/imagebasedupgrade/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	// LCANamespace is the namespace used by the lifecycle-agent.
	LCANamespace = "openshift-lifecycle-agent"
)

var (
	// Labels represents the range of labels that can be used for test cases selection.
	Labels = append(mgmtparams.Labels, LabelSuite)

	// ReporterNamespacesToDump tells to the reporter from where to collect logs.
	ReporterNamespacesToDump = map[string]string{
		LCANamespace: "lca",
	}

	// ReporterCRDsToDump tells to the reporter what CRs to dump.
	ReporterCRDsToDump = []k8sreporter.CRData{
		{Cr: &corev1.PodList{}},
		{Cr: &lcav1.ImageBasedUpgradeList{}},
	}
)
