package tsparams

const (
	// LabelSuite represents kmm label that can be used for test cases selection.
	LabelSuite = "module"
	// LabelSanity represents kmm label for short-running tests used for test case selection.
	LabelSanity = "kmm-sanity"
	// LabelLongRun represent kmm label for long-running tests used for test case selection.
	LabelLongRun = "kmm-longrun"
	// KmmOperatorNamespace represents the namespace where KMM is installed.
	KmmOperatorNamespace = "openshift-kmm"
	// DeploymentName represents the name of the KMM operator deployment.
	DeploymentName = "kmm-operator-controller"
	// BuildArgName represents kmod key passed to kmm-ci example.
	BuildArgName = "MY_MODULE"
	// ModuleNodeLabelTemplate represents template of the label set on a node for a Module.
	ModuleNodeLabelTemplate = "kmm.node.kubernetes.io/%s.%s.ready"
	// DevicePluginNodeLabelTemplate represents template label set by KMM on a node for a Device Plugin.
	DevicePluginNodeLabelTemplate = "kmm.node.kubernetes.io/%s.%s.device-plugin-ready"
	// UseDtkModuleTestNamespace represents test case namespace name.
	UseDtkModuleTestNamespace = "54283-use-dtk"
	// UseLocalMultiStageTestNamespace represents test case namespace name.
	UseLocalMultiStageTestNamespace = "53651-multi-stage"
	// WebhookModuleTestNamespace represents test case namespace name.
	WebhookModuleTestNamespace = "webhook"
	// SimpleKmodModuleTestNamespace represents test case namespace name.
	SimpleKmodModuleTestNamespace = "simple-kmod"
	// DevicePluginTestNamespace represents test case namespace name.
	DevicePluginTestNamespace = "53678-devplug"
	// RealtimeKernelNamespace represents test case namespace name.
	RealtimeKernelNamespace = "53656-rtkernel"
	// FirmwareTestNamespace represents test case namespace name.
	FirmwareTestNamespace = "simple-kmod-firmware"
	// ModuleBuildAndSignNamespace represents test case namespace name.
	ModuleBuildAndSignNamespace = "56252"
	// InTreeReplacementNamespace represents test case namespace name.
	InTreeReplacementNamespace = "62745"
	// MultipleModuleTestNamespace represents test case namespace name.
	MultipleModuleTestNamespace = "multiple-modules"
)
