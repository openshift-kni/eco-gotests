package tsparams

import (
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/mgmt/internal/mgmtparams"
	"github.com/openshift-kni/k8sreporter"
	lcav1alpha1 "github.com/openshift-kni/lifecycle-agent/api/v1alpha1"
	v1 "github.com/openshift/api/config/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	// LCANamespace is the namespace used by the lifecycle-agent.
	LCANamespace = "openshift-lifecycle-agent"

	// LCAWorkloadName is the name used for creating resources needed to backup workload app.
	LCAWorkloadName = "ibu-workload-app"

	// LCAOADPNamespace is the namespace used by the OADP operator.
	LCAOADPNamespace = "openshift-adp"

	// LCAKlusterletNamespace is the namespace that contains the klusterlet.
	LCAKlusterletNamespace = "open-cluster-management-agent"
)

var (
	// Labels represents the range of labels that can be used for test cases selection.
	Labels = append(mgmtparams.Labels, LabelSuite)

	// ReporterNamespacesToDump tells to the reporter from where to collect logs.
	ReporterNamespacesToDump = map[string]string{
		LCANamespace:           "lca",
		LCAWorkloadName:        "workload",
		LCAKlusterletNamespace: "klusterlet",
	}

	// ReporterCRDsToDump tells to the reporter what CRs to dump.
	ReporterCRDsToDump = []k8sreporter.CRData{
		{Cr: &corev1.PodList{}},
		{Cr: &batchv1.JobList{}},
		{Cr: &corev1.ConfigMapList{}},
		{Cr: &appsv1.DeploymentList{}},
		{Cr: &corev1.ServiceList{}},
		{Cr: &lcav1alpha1.ImageBasedUpgradeList{}},
		{Cr: &v1.ClusterOperatorList{}},
	}
)
