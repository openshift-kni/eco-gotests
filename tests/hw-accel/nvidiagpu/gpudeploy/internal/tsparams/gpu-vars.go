package tsparams

import (
	nvidiagpuv1 "github.com/NVIDIA/gpu-operator/api/v1"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/internal/hwaccelparams"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/nvidiagpu/internal/gpuparams"
	"github.com/openshift-kni/k8sreporter"
)

var (
	// Labels represents the range of labels that can be used for test cases selection.
	Labels = append(gpuparams.Labels, LabelSuite)

	// ReporterNamespacesToDump tells to the reporter from where to collect logs.
	ReporterNamespacesToDump = map[string]string{
		hwaccelparams.NFDNamespace: "nfd-operator",
		"nvidia-gpu-operator":      "gpu-operator",
		GPUTestNamespace:           "test-gpu-burn",
	}

	// ReporterCRDsToDump tells to the reporter what CRs to dump.
	ReporterCRDsToDump = []k8sreporter.CRData{
		{Cr: &nvidiagpuv1.ClusterPolicyList{}},
	}
)
