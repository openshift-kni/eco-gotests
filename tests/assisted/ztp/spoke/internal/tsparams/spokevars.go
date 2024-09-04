package tsparams

import (
	bmhv1alpha1 "github.com/metal3-io/baremetal-operator/apis/metal3.io/v1alpha1"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	hiveextV1Beta1 "github.com/openshift-kni/eco-goinfra/pkg/schemes/assisted/api/hiveextension/v1beta1"
	agentInstallV1Beta1 "github.com/openshift-kni/eco-goinfra/pkg/schemes/assisted/api/v1beta1"
	hivev1 "github.com/openshift-kni/eco-goinfra/pkg/schemes/assisted/hive/api/v1"
	"github.com/openshift-kni/eco-gotests/tests/assisted/ztp/internal/ztpparams"
	"github.com/openshift-kni/k8sreporter"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
		{Cr: &hivev1.ClusterDeploymentList{}},
		{Cr: &hiveextV1Beta1.AgentClusterInstallList{}},
		{Cr: &agentInstallV1Beta1.InfraEnvList{}},
		{Cr: &bmhv1alpha1.BareMetalHostList{}},
		{Cr: &agentInstallV1Beta1.AgentList{}},
	}
)

var schemesToAdd = []clients.SchemeAttacher{
	hivev1.AddToScheme,
	hiveextV1Beta1.AddToScheme,
	agentInstallV1Beta1.AddToScheme,
	bmhv1alpha1.AddToScheme,
}

// SetReporterSchemes add specified schemes to the reporter.
func SetReporterSchemes(crScheme *runtime.Scheme) error {
	for _, addScheme := range schemesToAdd {
		if err := addScheme(crScheme); err != nil {
			return err
		}
	}

	return nil
}
