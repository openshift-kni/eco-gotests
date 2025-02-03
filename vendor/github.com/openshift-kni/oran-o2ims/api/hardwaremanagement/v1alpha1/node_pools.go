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

// LocationSpec is the geographical location of the requested node.
type LocationSpec struct {
	// Location
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Location",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	Location string `json:"location,omitempty"`
	// Site
	// +kubebuilder:validation:Required
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Site",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	Site string `json:"site"`
}

// NodePoolSpec describes a pool of nodes to allocate
type NodePoolSpec struct {
	// CloudID is the identifier of the O-Cloud that generated this request. The hardware
	// manager may want to use this to tag the nodes in its database, and to generate
	// statistics.
	//
	// +kubebuilder:validation:Required
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Cloud ID",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	CloudID string `json:"cloudID"`

	// LocationSpec is the geographical location of the requested node.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Location Spec",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	LocationSpec `json:",inline"`

	// HwMgrId is the identifier for the hardware manager plugin instance.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Hardware Manager ID",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	HwMgrId string `json:"hwMgrId,omitempty"`

	//+operator-sdk:csv:customresourcedefinitions:type=spec
	NodeGroup []NodeGroup `json:"nodeGroup"`

	//+operator-sdk:csv:customresourcedefinitions:type=spec
	Extensions map[string]string `json:"extensions,omitempty"`
}

type NodeGroup struct {
	NodePoolData NodePoolData `json:"nodePoolData"` // Explicitly include as a named field
	Size         int          `json:"size" yaml:"size"`
}

type Properties struct {
	NodeNames []string `json:"nodeNames,omitempty"`
}

// GenerationStatus represents the observed generation for an operator.
type GenerationStatus struct {
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// NodePoolStatus describes the observed state of a request to allocate and prepare
// a node that will eventually be part of a deployment manager.
type NodePoolStatus struct {
	// Properties represent the node properties in the pool
	//+operator-sdk:csv:customresourcedefinitions:type=status
	Properties Properties `json:"properties,omitempty"`

	// Conditions represent the latest available observations of an NodePool's state.
	// +optional
	// +kubebuilder:validation:Type=array
	// +kubebuilder:validation:Items=Type=object
	//+operator-sdk:csv:customresourcedefinitions:type=status
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`

	//+operator-sdk:csv:customresourcedefinitions:type=status
	HwMgrPlugin GenerationStatus `json:"hwMgrPlugin,omitempty"`

	//+operator-sdk:csv:customresourcedefinitions:type=status
	SelectedPools map[string]string `json:"selectedPools,omitempty"`
}

// NodePool is the schema for an allocation request of nodes
//
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=nodepools,shortName=orannp
// +kubebuilder:printcolumn:name="HwMgr Id",type="string",JSONPath=".spec.hwMgrId"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.conditions[-1:].reason"
// +operator-sdk:csv:customresourcedefinitions:displayName="Node Pool",resources={{Namespace, v1}}
type NodePool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodePoolSpec   `json:"spec,omitempty"`
	Status NodePoolStatus `json:"status,omitempty"`
}

// NodePoolList contains a list of node allocation requests.
//
// +kubebuilder:object:root=true
type NodePoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NodePool `json:"items"`
}

func init() {
	SchemeBuilder.Register(
		&NodePool{},
		&NodePoolList{},
	)
}
