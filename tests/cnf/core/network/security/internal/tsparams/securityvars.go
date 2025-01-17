package tsparams

import (
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netparam"
	"github.com/openshift-kni/k8sreporter"
	mcfgv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
)

var (
	// Labels represents the range of labels that can be used for test cases selection.
	Labels = append(netparam.Labels, LabelSuite)

	// ReporterNamespacesToDump tells to the reporter from where to collect logs.
	ReporterNamespacesToDump = map[string]string{
		TestNamespaceName: TestNamespaceName,
	}

	// TestNamespaceName security-tests namespace where all test cases are performed.
	TestNamespaceName = "security-tests"
	// ReporterCRDsToDump tells to the reporter what CRs to dump.
	ReporterCRDsToDump = []k8sreporter.CRData{
		{Cr: &mcfgv1.MachineConfigList{}},
	}
)
