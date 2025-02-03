/*
Copyright (c) 2024 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in
compliance with the License. You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed under the License is
distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
implied. See the License for the specific language governing permissions and limitations under the
License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NodePoolData provides the necessary information for populating a node pool
type NodePoolData struct {
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// +kubebuilder:validation:Enum=master;worker
	Role string `json:"role"`
	// +kubebuilder:validation:MinLength=1
	HwProfile string `json:"hwProfile"`
	// ResourcePoolId is the identifier for the Resource Pool in the hardware manager instance.
	// +optional
	ResourcePoolId string `json:"resourcePoolId,omitempty"`
	// +optional
	ResourceSelector string `json:"resourceSelector,omitempty"`
}

// HardwareTemplateSpec defines the desired state of HardwareTemplate
type HardwareTemplateSpec struct {

	// HwMgrId is the identifier for the hardware manager plugin adaptor.
	// +kubebuilder:validation:MinLength=1
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Hardware Manager ID",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	HwMgrId string `json:"hwMgrId"`

	// BootInterfaceLabel is the label of the boot interface.
	// +kubebuilder:validation:MinLength=1
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Boot Interface Label",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	BootInterfaceLabel string `json:"bootInterfaceLabel"`

	// HardwareProvisioningTimeout defines the timeout duration string for the hardware provisioning.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Hardware Provisioning Timeout",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	HardwareProvisioningTimeout string `json:"hardwareProvisioningTimeout,omitempty"`

	// NodePoolData defines a collection of NodePoolData items
	// +kubebuilder:validation:MinItems=1
	//+operator-sdk:csv:customresourcedefinitions:type=spec
	NodePoolData []NodePoolData `json:"nodePoolData"`

	// Extensions holds additional custom key-value pairs that can be used to extend the node pool's configuration.
	//+operator-sdk:csv:customresourcedefinitions:type=spec
	Extensions map[string]string `json:"extensions,omitempty"`
}

// HardwareTemplateStatus defines the observed state of HardwareTemplate
type HardwareTemplateStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of HardwareTemplate
	// Important: Run "make" to regenerate code after modifying this file

	//+operator-sdk:csv:customresourcedefinitions:type=status
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=hardwaretemplates,shortName=oranhwtmpl
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
//+kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.conditions[-1:].reason"
//+kubebuilder:printcolumn:name="Details",type="string",JSONPath=".status.conditions[-1:].message"

// HardwareTemplate is the Schema for the hardwaretemplates API
// +kubebuilder:validation:XValidation:message="Spec changes are not allowed for a HardwareTemplate that has passed the validation", rule="!has(oldSelf.status) || oldSelf.status.conditions.exists(c, c.type=='Validation' && c.status=='False') || oldSelf.spec == self.spec"
// +operator-sdk:csv:customresourcedefinitions:displayName="ORAN O2IMS Hardware Template",resources={{ConfigMap, v1}}
type HardwareTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HardwareTemplateSpec   `json:"spec,omitempty"`
	Status HardwareTemplateStatus `json:"status,omitempty"`
}

// HardwareTemplateList contains a list of HardwareTemplate.
//
// +kubebuilder:object:root=true
type HardwareTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HardwareTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(
		&HardwareTemplate{},
		&HardwareTemplateList{},
	)
}
