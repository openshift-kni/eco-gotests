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

	// KernelConfig contains all kernel config labels that suppose to be in worker node.
	KernelConfig = []string{"NO_HZ"}

	// Topology label.
	Topology = []string{"cpu.topology"}

	// NUMA labels.
	NUMA = []string{"memory-numa"}

	// Namespace represents operator namespace value.
	Namespace = "openshift-nfd"

	// Name represents operand name.
	Name = "nfd-instance-test"

	// DefaultBlackList list of default value which nfd ignore when detecting features.
	DefaultBlackList = []string{
		"BMI1",
		"BMI2",
		"CLMUL",
		"CMOV",
		"CX16",
		"ERMS",
		"F16C",
		"HTT",
		"LZCNT",
		"MMX",
		"MMXEXT",
		"NX",
		"POPCNT",
		"RDRAND",
		"RDSEED",
		"RDTSCP",
		"SGX",
		"SGXLC",
		"SSE",
		"SSE2",
		"SSE3",
		"SSE4",
		"SSE42",
		"SSSE3",
		"TDX_GUEST",
	}
	// NfdPods list of NFD operator pods.
	NfdPods = []string{
		"nfd-controller-manager",
		"nfd-master",
		"nfd-topology",
		"nfd-worker",
	}
)
