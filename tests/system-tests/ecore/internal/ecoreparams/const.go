package ecoreparams

const (
	// Label is used to select system tests for SPK operator.
	Label = "ecore"

	// MachineConfidDaemonPodSelector is a a label selector for all machine-config-daemon pods.
	MachineConfidDaemonPodSelector = "k8s-app=machine-config-daemon"

	// MachineConfigOperatorNS namespace for Machine Config Operator.
	MachineConfigOperatorNS = "openshift-machine-config-operator"

	// MachineConfigDaemonContainerName container name within machine-config-daemon pod.
	MachineConfigDaemonContainerName = "machine-config-daemon"

	// LabelEcoreValidateNAD is used to select all tests for network-attachment-definition validation.
	LabelEcoreValidateNAD = "ecore_validate_nad"

	// ECoreLogLevel configures logging level for ECore related tests.
	ECoreLogLevel = 90

	// LabelEcoreValidateMCP is used to select tests for custom MCP validation.
	LabelEcoreValidateMCP = "ecore_validate_mcp"

	// LabelEcoreValidateSysReservedMemory is used to select test for system reserved memory.
	LabelEcoreValidateSysReservedMemory = "ecore_validate_sys_reserved_memory"

	// LabelEcoreValidatePerformanceProfile is used to select tests for PerformanceProfile validation.
	LabelEcoreValidatePerformanceProfile = "ecore_validate_performance_profile"

	// LabelEcoreValidatePolicies is used to select test for policies validation.
	LabelEcoreValidatePolicies = "ecore_validate_policies"
)
