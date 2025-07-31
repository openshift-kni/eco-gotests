package tsparams

const (
	// AMDGPULogLevel - Log Level for AMD GPU Tests.
	AMDGPULogLevel = 90
	// AMDGPUNamespace - Namespace for the AMD GPU Operator.
	AMDGPUNamespace = "openshift-amd-gpu"
	// AMDNFDLabelKey - The key of the label added by NFD.
	AMDNFDLabelKey = "feature.node.kubernetes.io/amd-gpu"
	// AMDNFDLabelValue - The value of the label added by NFD.
	AMDNFDLabelValue = "true"
	// DeviceConfigName - The name of the DeviceConfig CR.
	DeviceConfigName = "amd-gpu-device-config"
	// LabelSuite represents 'AMD GPU Basic' label that can be used for test cases selection.
	LabelSuite = "amd-gpu-basic"
	// MaxAttempts - Max retry attempts.
	MaxAttempts = 10
)
