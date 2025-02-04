/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// HardwareManagerAdaptorID defines the type for the Hardware Manager Adaptor
type HardwareManagerAdaptorID string

// SupportedAdaptors defines the string values for valid stages
var SupportedAdaptors = struct {
	Loopback HardwareManagerAdaptorID
	Dell     HardwareManagerAdaptorID
}{
	Loopback: "loopback",
	Dell:     "dell-hwmgr",
}

// ConditionType is a string representing the condition's type
type ConditionType string

// ConditionTypes define the different types of conditions that will be set
var ConditionTypes = struct {
	Validation ConditionType
}{
	Validation: "Validation",
}

// ConditionReason is a string representing the condition's reason
type ConditionReason string

// ConditionReasons define the different reasons that conditions will be set for
var ConditionReasons = struct {
	Completed  ConditionReason
	Failed     ConditionReason
	InProgress ConditionReason
}{
	Completed:  "Completed",
	Failed:     "Failed",
	InProgress: "InProgress",
}

// OAuthGrantType is a string representing the OAuth2 grant type
type OAuthGrantType string

// OAuthGrantTypes define the different reasons that conditions will be set for
var OAuthGrantTypes = struct {
	ClientCredentials OAuthGrantType
	Password          OAuthGrantType
}{
	ClientCredentials: "client_credentials",
	Password:          "password",
}

// LoopbackData defines configuration data for loopback adaptor instance
type LoopbackData struct {
	// A test string
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	AddtionalInfo string `json:"additionalInfo,omitempty"`
}

// DellData defines configuration data for dell-hwmgr adaptor instance
type DellData struct {
	// +kubebuilder:validation:Required
	// +required
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	AuthSecret string `json:"authSecret"`

	// +kubebuilder:validation:Required
	// +required
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	ApiUrl string `json:"apiUrl"`

	// CaBundleName references a config map that contains a set of custom CA certificates to be used when communicating
	// with a hardware manager that has its TLS certificate signed by a non-public CA certificate.
	// +optional
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Custom CA Certificates",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	CaBundleName *string `json:"caBundleName,omitempty"`

	// Tenant allows the specification of the hardware manager tenant to use for this instance.
	// +optional
	Tenant *string `json:"tenant,omitempty"`

	// insecureSkipTLSVerify indicates that the plugin should not confirm the validity of the TLS certificate of the hardware manager.
	// This is insecure and is not recommended.
	// +optional
	InsecureSkipTLSVerify bool `json:"insecureSkipTLSVerify,omitempty"`
}

// HardwareManagerSpec defines the desired state of HardwareManager
type HardwareManagerSpec struct {
	// Important: Run "make" to regenerate code after modifying this file

	// The adaptor ID
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=loopback;dell-hwmgr
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	AdaptorID HardwareManagerAdaptorID `json:"adaptorId"`

	// Config data for an instance of the loopback adaptor
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	LoopbackData *LoopbackData `json:"loopbackData,omitempty"`

	// Config data for an instance of the dell-hwmgr adaptor
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	DellData *DellData `json:"dellData,omitempty"`
}

type ResourcePoolList []string
type PerSiteResourcePoolList map[string]ResourcePoolList

// HardwareManagerStatus defines the observed state of HardwareManager
type HardwareManagerStatus struct {
	// +operator-sdk:csv:customresourcedefinitions:type=status
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions describe the state of the UpdateService resource.
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// ResourcePools provides a per-site list of resource pools
	// +operator-sdk:csv:customresourcedefinitions:type=status
	ResourcePools PerSiteResourcePoolList `json:"resourcePools,omitempty"`
}

// +operator-sdk:csv:customresourcedefinitions:resources={{Service,v1,policy-engine-service}}
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=hardwaremanagers,scope=Namespaced
// +kubebuilder:resource:shortName=hwmgr;hwmgrs
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="The age of the HardwareManager resource."
// +kubebuilder:printcolumn:name="Adaptor ID",type="string",JSONPath=".status.adaptorId",description="The adaptor ID.",priority=1
// +kubebuilder:printcolumn:name="Reason",type="string",JSONPath=".status.conditions[-1:].reason"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[-1:].status"
// +kubebuilder:printcolumn:name="Details",type="string",JSONPath=".status.conditions[-1:].message"

// HardwareManager is the Schema for the hardwaremanagers API
type HardwareManager struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HardwareManagerSpec   `json:"spec,omitempty"`
	Status HardwareManagerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// HardwareManagerList contains a list of HardwareManager
type HardwareManagerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HardwareManager `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HardwareManager{}, &HardwareManagerList{})
}
