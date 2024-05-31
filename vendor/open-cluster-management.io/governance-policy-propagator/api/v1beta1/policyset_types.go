// Copyright (c) 2021 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// A custom type is required since there is no way to have a kubebuilder marker
// apply to the items of a slice.

// +kubebuilder:validation:MinLength=1
type NonEmptyString string

// PolicySetSpec describes a group of policies that are related and
// can be placed on the same managed clusters.
type PolicySetSpec struct {
	// Description of this PolicySet.
	Description string `json:"description,omitempty"`
	// Policies that are grouped together within the PolicySet.
	// +kubebuilder:validation:Required
	Policies []NonEmptyString `json:"policies"`
}

// PolicySetStatus defines the observed state of PolicySet
type PolicySetStatus struct {
	Placement     []PolicySetStatusPlacement `json:"placement,omitempty"`
	Compliant     string                     `json:"compliant,omitempty"`
	StatusMessage string                     `json:"statusMessage,omitempty"`
}

// PolicySetStatusPlacement defines a placement object for the status
type PolicySetStatusPlacement struct {
	PlacementBinding string `json:"placementBinding,omitempty"`
	Placement        string `json:"placement,omitempty"`
	PlacementRule    string `json:"placementRule,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=policysets,scope=Namespaced
// +kubebuilder:resource:path=policysets,shortName=plcset
// +kubebuilder:printcolumn:name="Compliance state",type="string",JSONPath=".status.compliant"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// PolicySet is the Schema for the policysets API
type PolicySet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// +kubebuilder:validation:Required
	Spec   PolicySetSpec   `json:"spec"`
	Status PolicySetStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PolicySetList contains a list of PolicySet
type PolicySetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PolicySet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PolicySet{}, &PolicySetList{})
}
