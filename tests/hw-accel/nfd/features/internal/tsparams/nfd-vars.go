package tsparams

import (
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/internal/hwaccelparams"

	"github.com/openshift-kni/k8sreporter"
	nfdv1 "github.com/openshift/cluster-nfd-operator/api/v1"
)

var (
	// ReporterNamespacesToDump tells to the reporter from where to collect logs.
	ReporterNamespacesToDump = map[string]string{
		"openshift-nfd":     "nfd",
		"openshift-nfd-hub": "nfd-hub",
		"":                  "other",
	}

	// ReporterCRDsToDump tells to the reporter what CRs to dump.
	ReporterCRDsToDump = []k8sreporter.CRData{
		{Cr: &nfdv1.NodeFeatureDiscoveryList{}},
	}

	// Labels represents the range of labels that can be used for test cases selection.
	Labels = []string{hwaccelparams.Label, "nfd"}

	// FeatureLabel contains all labels that suppose to be in worker node.
	FeatureLabel = []string{"ADX", "AESNI", "AVX", "AVX2", "AVX512BW", "AVX512CD", "AVX512DQ",
		"AVX512F", "AVX512VL", "AVX512VNNI", "FMA3",
		"HYPERVISOR", "IBPB", "MPX", "STIBP", "VMX"}
	// Namespace represents operator namespace value.
	Namespace = "openshift-nfd"
	// Name represents operand name.
	Name = "nfd-instance-test"
)
