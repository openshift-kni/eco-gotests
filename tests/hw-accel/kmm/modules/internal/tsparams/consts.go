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
	// RelImgMustGather represents identifier for must-gather image in operator environment variables.
	RelImgMustGather = "RELATED_IMAGES_MUST_GATHER"
	// RelImgSign represents identifier for sign image in operator environment variables.
	RelImgSign = "RELATED_IMAGES_SIGN"
	// RelImgWorker represents identifier for worker image in operator environment variables.
	RelImgWorker = "RELATED_IMAGES_WORKER"
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
	ModuleBuildAndSignNamespace = "ocp-56252"
	// InTreeReplacementNamespace represents test case namespace name.
	InTreeReplacementNamespace = "ocp-62745"
	// MultipleModuleTestNamespace represents test case namespace name.
	MultipleModuleTestNamespace = "multiple-modules"
	// VersionModuleTestNamespace represents test case namespace name.
	VersionModuleTestNamespace = "modver"
	// PreflightTemplateImage represents image against which preflightvalidationocp will build against.
	PreflightTemplateImage = "quay.io/openshift-release-dev/ocp-release:4.10.15-%s"
	// PreflightName represents preflightvalidation ocp object name.
	PreflightName = "preflight"
	// ScannerTestNamespace represents test case namespace name.
	ScannerTestNamespace = "kmm-scanner"
)
