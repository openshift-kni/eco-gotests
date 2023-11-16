// Copyright (c) 2021 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"

	policyv1 "open-cluster-management.io/governance-policy-propagator/api/v1"
)

// PolicyAutomationSpec defines the desired state of PolicyAutomation
type PolicyAutomationSpec struct {
	// PolicyRef is the name of the policy that this automation resource
	// is bound to.
	// +kubebuilder:validation:Required
	PolicyRef string `json:"policyRef"`
	// Mode decides how automation is going to be triggered
	Mode PolicyAutomationMode `json:"mode"`
	// EventHook decides when automation is going to be triggered
	// +kubebuilder:validation:Enum={noncompliant}
	// +kubebuilder:validation:Required
	EventHook string `json:"eventHook,omitempty"`
	// RescanAfter is reserved for future use.
	RescanAfter string `json:"rescanAfter,omitempty"`
	// DelayAfterRunSeconds sets the minimum number of seconds before
	// an automation can run again due to a new violation on the same
	// managed cluster. This only applies to the EveryEvent Mode.  The
	// default value is 0.
	// +kubebuilder:validation:Minimum=0
	DelayAfterRunSeconds uint `json:"delayAfterRunSeconds,omitempty"`
	// +kubebuilder:validation:Required
	Automation AutomationDef `json:"automationDef"`
}

// +kubebuilder:validation:Enum={once,everyEvent,disabled}
// +kubebuilder:validation:Required
type PolicyAutomationMode string

const (
	Once       PolicyAutomationMode = "once"
	EveryEvent PolicyAutomationMode = "everyEvent"
	Disabled   PolicyAutomationMode = "disabled"
)

const DefaultPolicyViolationsLimit = 1000

// AutomationDef defines the automation to invoke
type AutomationDef struct {
	// Type of the automation to invoke
	Type string `json:"type,omitempty"`
	// Name of the Ansible Template to run in Tower as a job
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// ExtraVars is passed to the Ansible job at execution time and is a known Ansible entity.
	// +kubebuilder:pruning:PreserveUnknownFields
	ExtraVars *runtime.RawExtension `json:"extra_vars,omitempty"`
	// TowerSecret is the name of the secret that contains the Ansible Automation Platform
	// credential.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	TowerSecret string `json:"secret"`
	// JobTTL sets the time to live for the Kubernetes AnsibleJob object after the Ansible job run has finished.
	JobTTL *int `json:"jobTtl,omitempty"`
	// +kubebuilder:validation:Minimum=0
	// The maximum number of violating cluster contexts that will be provided to the Ansible job as extra variables.
	// When policyViolationsLimit is set to 0, it means no limit.
	// The default value is 1000.
	PolicyViolationsLimit *uint `json:"policyViolationsLimit,omitempty"`
}

// ViolationContext defines the non-compliant replicated policy information
// that is sent to the AnsibleJob through extra_vars.
type ViolationContext struct {
	TargetClusters   []string                          `json:"targetClusters" ansibleJob:"target_clusters"`
	PolicyName       string                            `json:"policyName" ansibleJob:"policy_name"`
	PolicyNamespace  string                            `json:"policyNamespace" ansibleJob:"policy_namespace"`
	HubCluster       string                            `json:"hubCluster" ansibleJob:"hub_cluster"`
	PolicySets       []string                          `json:"policySets" ansibleJob:"policy_sets"`
	PolicyViolations map[string]ReplicatedPolicyStatus `json:"policyViolations" ansibleJob:"policy_violations"`
}

// PolicyAutomationStatus defines the observed state of PolicyAutomation
type PolicyAutomationStatus struct {
	// Cluster name as the key of ClustersWithEvent
	ClustersWithEvent map[string]ClusterEvent `json:"clustersWithEvent,omitempty"`
}

//+kubebuilder:object:root=true

// PolicyAutomation is the Schema for the policyautomations API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=policyautomations,scope=Namespaced
// +kubebuilder:resource:path=policyautomations,shortName=plca
type PolicyAutomation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// +kubebuilder:validation:Required
	Spec   PolicyAutomationSpec   `json:"spec"`
	Status PolicyAutomationStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PolicyAutomationList contains a list of PolicyAutomation
type PolicyAutomationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PolicyAutomation `json:"items"`
}

// PolicyAutomation events on each target cluster
type ClusterEvent struct {
	// Policy automation start time for everyEvent mode
	AutomationStartTime string `json:"automationStartTime"`
	// The last policy compliance transition event time
	EventTime string `json:"eventTime"`
}

func init() {
	SchemeBuilder.Register(&PolicyAutomation{}, &PolicyAutomationList{})
}

// ReplicatedDetailsPerTemplate defines the replicated policy compliance details and history
type ReplicatedDetailsPerTemplate struct {
	ComplianceState policyv1.ComplianceState      `json:"compliant"`
	History         []ReplicatedComplianceHistory `json:"history"`
}

// ReplicatedComplianceHistory defines the replicated policy compliance details history
type ReplicatedComplianceHistory struct {
	LastTimestamp metav1.Time `json:"lastTimestamp,omitempty" protobuf:"bytes,7,opt,name=lastTimestamp"`
	Message       string      `json:"message,omitempty" protobuf:"bytes,4,opt,name=message"`
}

// ReplicatedPolicyStatus defines the replicated policy status
type ReplicatedPolicyStatus struct {
	ComplianceState  policyv1.ComplianceState       `json:"compliant"`         // used by replicated policy
	ViolationMessage string                         `json:"violation_message"` // used by replicated policy
	Details          []ReplicatedDetailsPerTemplate `json:"details"`           // used by replicated policy
}
