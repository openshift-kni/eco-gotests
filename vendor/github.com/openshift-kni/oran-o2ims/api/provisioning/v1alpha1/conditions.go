package v1alpha1

// ConditionType is a string representing the condition's type
type ConditionType string

// The following constants define the different types of conditions that will be set for ClusterTemplate
var CTconditionTypes = struct {
	Validated ConditionType
}{
	Validated: "ClusterTemplateValidated",
}

// The following constants define the different types of conditions that will be set for ProvisioningRequest
var PRconditionTypes = struct {
	Validated                 ConditionType
	HardwareTemplateRendered  ConditionType
	HardwareProvisioned       ConditionType
	HardwareNodeConfigApplied ConditionType
	HardwareConfigured        ConditionType
	ClusterInstanceRendered   ConditionType
	ClusterResourcesCreated   ConditionType
	ClusterInstanceProcessed  ConditionType
	ClusterProvisioned        ConditionType
	ConfigurationApplied      ConditionType
	UpgradeCompleted          ConditionType
}{
	Validated:                 "ProvisioningRequestValidated",
	HardwareTemplateRendered:  "HardwareTemplateRendered",
	HardwareProvisioned:       "HardwareProvisioned",
	HardwareNodeConfigApplied: "HardwareNodeConfigApplied",
	HardwareConfigured:        "HardwareConfigured",
	ClusterInstanceRendered:   "ClusterInstanceRendered",
	ClusterResourcesCreated:   "ClusterResourcesCreated",
	ClusterInstanceProcessed:  "ClusterInstanceProcessed",
	ClusterProvisioned:        "ClusterProvisioned",
	ConfigurationApplied:      "ConfigurationApplied",
	UpgradeCompleted:          "UpgradeCompleted",
}

// ConditionReason is a string representing the condition's reason
type ConditionReason string

// The following constants define the different reasons that conditions will be set for ClusterTemplate
var CTconditionReasons = struct {
	Completed ConditionReason
	Failed    ConditionReason
}{
	Completed: "Completed",
	Failed:    "Failed",
}

// The following constants define the different reasons that conditions will be set for ProvisioningRequest
var CRconditionReasons = struct {
	NotApplied      ConditionReason
	ClusterNotReady ConditionReason
	Completed       ConditionReason
	Failed          ConditionReason
	InProgress      ConditionReason
	Missing         ConditionReason
	OutOfDate       ConditionReason
	TimedOut        ConditionReason
	Unknown         ConditionReason
}{
	NotApplied:      "NotApplied",
	ClusterNotReady: "ClusterNotReady",
	Completed:       "Completed",
	Failed:          "Failed",
	InProgress:      "InProgress",
	Missing:         "Missing",
	OutOfDate:       "OutOfDate",
	TimedOut:        "TimedOut",
	Unknown:         "Unknown",
}
