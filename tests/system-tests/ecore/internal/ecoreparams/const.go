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

	// SRIOVOperatorNS namespace where SR-IOV operator is installed.
	SRIOVOperatorNS = "openshift-sriov-network-operator"

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

	// LabelEcoreValidateSriov is used to select test for SR-IOV validation.
	LabelEcoreValidateSriov = "ecore_validate_sriov"

	// LabelEcoreValidateODFStorage is used to select tests for ODF storage validation.
	LabelEcoreValidateODFStorage = "ecore_validate_odf_storage"

	// LabelEcoreValidateNmstate is used to select tests for NMState validation.
	LabelEcoreValidateNmstate = "ecore_validate_nmstate"

	// NMStateInstanceName is a name of the NMState instance.
	NMStateInstanceName = "nmstate"

	// NMStateNS namespace for OpensShift-NMState operator.
	NMStateNS = "openshift-nmstate"

	// LabelEcoreValidateReboots is used to select tests that reboot cluster.
	LabelEcoreValidateReboots = "ecore_validate_reboots"

	// ConditionTypeReadyString constant to fix linter warning.
	ConditionTypeReadyString = "Ready"

	// ConstantTrueString constant to fix linter warning.
	ConstantTrueString = "True"
)
