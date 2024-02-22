package rdscoreparams

const (
	// Label is used to select tests for RDS Core setup.
	Label = "rdscore"

	// RDSCoreLogLevel configures logging level for RDS Core related tests.
	RDSCoreLogLevel = 90

	// NMStateInstanceName is a name of the NMState instance.
	NMStateInstanceName = "nmstate"

	// MachineConfidDaemonPodSelector is a a label selector for all machine-config-daemon pods.
	MachineConfidDaemonPodSelector = "k8s-app=machine-config-daemon"

	// LabelValidatePerformanceProfile is a test selector for performance profile validation.
	LabelValidatePerformanceProfile = "rds-core-performance-profile"

	// MachineConfigDaemonContainerName is a name of container within machine-config-daemon pod.
	MachineConfigDaemonContainerName = "machine-config-daemon"

	// LabelValidateNMState a label to select tests for NMState validation.
	LabelValidateNMState = "rds-core-validate-nmstate"

	// ConditionTypeReadyString constant to fix linter warning.
	ConditionTypeReadyString = "Ready"

	// ConstantTrueString constant to fix linter warning.
	ConstantTrueString = "True"
)
