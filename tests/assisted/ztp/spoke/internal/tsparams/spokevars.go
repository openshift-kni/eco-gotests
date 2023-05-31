package tsparams

import (
	bmhv1alpha1 "github.com/metal3-io/baremetal-operator/apis/metal3.io/v1alpha1"
	"github.com/openshift-kni/eco-gotests/tests/assisted/ztp/internal/ztpparams"
	"github.com/openshift-kni/k8sreporter"
	hiveextV1Beta1 "github.com/openshift/assisted-service/api/hiveextension/v1beta1"
	agentInstallV1Beta1 "github.com/openshift/assisted-service/api/v1beta1"
	v1 "github.com/openshift/hive/apis/hive/v1"
	corev1 "k8s.io/api/core/v1"
)

var (
	// Labels represents the range of labels that can be used for test cases selection.
	Labels = append(ztpparams.Labels, LabelSuite)

	// ReporterNamespacesToDump tells to the reporter from where to collect logs.
	ReporterNamespacesToDump = map[string]string{
		"multicluster-engine": "mce",
	}

	// ReporterCRDsToDump tells to the reporter what CRs to dump.
	ReporterCRDsToDump = []k8sreporter.CRData{
		{Cr: &corev1.SecretList{}},
		{Cr: &v1.ClusterDeploymentList{}},
		{Cr: &hiveextV1Beta1.AgentClusterInstallList{}},
		{Cr: &agentInstallV1Beta1.InfraEnvList{}},
		{Cr: &bmhv1alpha1.BareMetalHostList{}},
		{Cr: &agentInstallV1Beta1.AgentList{}},
	}
)
