package tsparams

const (
	// LabelSuite represents kmm label that can be used for test cases selection.
	LabelSuite = "module"
	// BuildArgName represents kmod key passed to kmm-ci example.
	BuildArgName = "MY_MODULE"
	// ModuleNodeLabelTemplate represents template of the label set on a node for a Module.
	ModuleNodeLabelTemplate = "kmm.node.kubernetes.io/%s.%s.ready"
	// DevicePluginNodeLabelTemplate represents template label set by KMM on a node for a Device Plugin.
	DevicePluginNodeLabelTemplate = "kmm.node.kubernetes.io/%s.%s.device-plugin-ready"
	// DevicePluginImageTemplate represents test image location in remote registry.
	// Will be moved in quay.io/organization/ocp-edge-qe once repository is set up.
	DevicePluginImageTemplate = "quay.io/cvultur/device-plugin:latest-%s"
	// UseDtkModuleTestNamespace represents test case namespace name.
	UseDtkModuleTestNamespace = "ocp-54283-use-dtk"
	// UseLocalMultiStageTestNamespace represents test case namespace name.
	UseLocalMultiStageTestNamespace = "ocp-53651-multi-stage"
	// WebhookModuleTestNamespace represents test case namespace name.
	WebhookModuleTestNamespace = "webhook"
	// SimpleKmodModuleTestNamespace represents test case namespace name.
	SimpleKmodModuleTestNamespace = "simple-kmod"
	// DevicePluginTestNamespace represents test case namespace name.
	DevicePluginTestNamespace = "ocp-53678-devplug"
	// RealtimeKernelNamespace represents test case namespace name.
	RealtimeKernelNamespace = "ocp-53656-rtkernel"
	// FirmwareTestNamespace represents test case namespace name.
	FirmwareTestNamespace = "simple-kmod-firmware"
	// ModuleBuildAndSignNamespace represents test case namespace name.
	ModuleBuildAndSignNamespace = "ocp-56252-module-signing"
)
