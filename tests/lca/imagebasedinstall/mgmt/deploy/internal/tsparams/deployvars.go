package tsparams

import (
	bmhv1alpha1 "github.com/metal3-io/baremetal-operator/apis/metal3.io/v1alpha1"
	hivev1 "github.com/openshift-kni/eco-goinfra/pkg/schemes/hive/api/v1"
	ibiv1alpha1 "github.com/openshift-kni/eco-goinfra/pkg/schemes/imagebasedinstall/api/hiveextensions/v1alpha1"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedinstall/mgmt/internal/mgmtparams"
	"github.com/openshift-kni/k8sreporter"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

var (
	// Labels represents the range of labels that can be used for test cases selection.
	Labels = append(mgmtparams.Labels, LabelSuite)

	// ReporterNamespacesToDump tells to the reporter from where to collect logs.
	ReporterNamespacesToDump = map[string]string{
		mgmtparams.IBIONamespace: "ibio",
	}

	// ReporterCRDsToDump tells to the reporter what CRs to dump.
	ReporterCRDsToDump = []k8sreporter.CRData{
		{Cr: &corev1.PodList{}},
		{Cr: &corev1.SecretList{}},
		{Cr: &corev1.ConfigMapList{}},
		{Cr: &appsv1.DeploymentList{}},
		{Cr: &corev1.ServiceList{}},
		{Cr: &ibiv1alpha1.ImageClusterInstallList{}},
		{Cr: &hivev1.ClusterImageSetList{}},
		{Cr: &hivev1.ClusterDeploymentList{}},
		{Cr: &bmhv1alpha1.BareMetalHostList{}},
	}
)
