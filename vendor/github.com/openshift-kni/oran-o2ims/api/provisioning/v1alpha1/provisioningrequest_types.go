/*
Copyright 2023.

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
	"k8s.io/apimachinery/pkg/runtime"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ProvisioningRequestSpec defines the desired state of ProvisioningRequest
type ProvisioningRequestSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Name specifies a human-readable name for this provisioning request, intended for identification and descriptive purposes.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Name",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	Name string `json:"name,omitempty"`

	// Description specifies a brief description of this provisioning request, providing additional context or details.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Description",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	Description string `json:"description,omitempty"`

	// TemplateName defines the base name of the referenced ClusterTemplate.
	// The full name of the ClusterTemplate is constructed as <TemplateName.TemplateVersion>.
	// +kubebuilder:validation:MinLength=1
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Template Name",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	TemplateName string `json:"templateName"`

	// TemplateVersion defines the version of the referenced ClusterTemplate.
	// The full name of the ClusterTemplate is constructed as <TemplateName.TemplateVersion>.
	// +kubebuilder:validation:MinLength=1
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Template Version",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	TemplateVersion string `json:"templateVersion"`

	// TemplateParameters provides the input data that conforms to the OpenAPI v3 schema defined in the referenced ClusterTemplate's spec.templateParameterSchema.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Template Parameters",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	TemplateParameters runtime.RawExtension `json:"templateParameters"`

	// Extensions holds additional custom key-value pairs that can be used to extend the cluster's configuration.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Extensions",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	Extensions runtime.RawExtension `json:"extensions,omitempty"`
}

// NodePoolRef references a node pool.
type NodePoolRef struct {
	// Contains the name of the created NodePool.
	Name string `json:"name,omitempty"`
	// Contains the namespace of the created NodePool.
	Namespace string `json:"namespace,omitempty"`
	// Represents the timestamp of the first status check for hardware provisioning
	HardwareProvisioningCheckStart *metav1.Time `json:"hardwareProvisioningCheckStart,omitempty"`
	// Represents the timestamp of the first status check for hardware configuring
	HardwareConfiguringCheckStart *metav1.Time `json:"hardwareConfiguringCheckStart,omitempty"`
}

type ClusterDetails struct {
	// Contains the name of the created ClusterInstance.
	Name string `json:"name,omitempty"`

	// Says if ZTP has complete or not.
	ZtpStatus string `json:"ztpStatus,omitempty"`

	// A timestamp indicating the cluster provisoning has started
	ClusterProvisionStartedAt *metav1.Time `json:"clusterProvisionStartedAt,omitempty"`

	// Holds the first timestamp when the configuration was found NonCompliant for the cluster.
	NonCompliantAt *metav1.Time `json:"nonCompliantAt,omitempty"`
}

type Extensions struct {
	// ClusterDetails references to the ClusterInstance.
	ClusterDetails *ClusterDetails `json:"clusterDetails,omitempty"`

	// NodePoolRef references to the NodePool.
	NodePoolRef *NodePoolRef `json:"nodePoolRef,omitempty"`

	// Holds policies that are matched with the ManagedCluster created by the ProvisioningRequest.
	Policies []PolicyDetails `json:"policies,omitempty"`
}

// PolicyDetails holds information about an ACM policy.
type PolicyDetails struct {
	// The compliance of the ManagedCluster created through a ProvisioningRequest with the current
	// policy.
	Compliant string `json:"compliant,omitempty"`
	// The policy's name.
	PolicyName string `json:"policyName,omitempty"`
	// The policy's namespace.
	PolicyNamespace string `json:"policyNamespace,omitempty"`
	// The policy's remediation action.
	RemediationAction string `json:"remediationAction,omitempty"`
}

// ProvisioningPhase defines the various phases of the provisioning process.
type ProvisioningPhase string

const (
	// StateProgressing means the provisioning process is currently in progress.
	// It could be in progress during hardware provisioning, cluster installation, or cluster configuration.
	StateProgressing ProvisioningPhase = "progressing"

	// StateFulfilled means the provisioning process has been successfully completed for all stages.
	StateFulfilled ProvisioningPhase = "fulfilled"

	// StateFailed means the provisioning process has failed at any stage, including resource validation
	// and preparation prior to provisioning, hardware provisioning, cluster installation, or cluster configuration.
	StateFailed ProvisioningPhase = "failed"

	// StateDeleting indicates that the provisioning resources are in the process of being deleted.
	// This state is set when the deletion process for the ProvisioningRequest and its resources
	// has started, ensuring that all dependent resources are removed before finalizing the
	// ProvisioningRequest deletion.
	StateDeleting ProvisioningPhase = "deleting"
)

// ProvisionedResources contains the resources that were provisioned as part of the provisioning process.
type ProvisionedResources struct {
	// The identifier of the provisioned oCloud Node Cluster.
	OCloudNodeClusterId string `json:"oCloudNodeClusterId,omitempty"`
}

type ProvisioningStatus struct {
	// The current state of the provisioning process.
	// +kubebuilder:validation:Enum=progressing;fulfilled;failed;deleting
	ProvisioningPhase ProvisioningPhase `json:"provisioningPhase,omitempty"`

	// The details about the current state of the provisioning process.
	ProvisioningDetails string `json:"provisioningDetails,omitempty"`

	// The resources that have been successfully provisioned as part of the provisioning process.
	ProvisionedResources *ProvisionedResources `json:"provisionedResources,omitempty"`

	// The timestamp of the last update to the provisioning status.
	UpdateTime metav1.Time `json:"updateTime,omitempty"`
}

// ProvisioningRequestStatus defines the observed state of ProvisioningRequest
type ProvisioningRequestStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	//+operator-sdk:csv:customresourcedefinitions:type=status
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Extensions contain extra details about the resources and the configuration used for/by
	// the ProvisioningRequest.
	Extensions Extensions `json:"extensions,omitempty"`

	ProvisioningStatus ProvisioningStatus `json:"provisioningStatus,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster,shortName=oranpr
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
//+kubebuilder:printcolumn:name="ProvisionPhase",type="string",JSONPath=".status.provisioningStatus.provisioningPhase"
//+kubebuilder:printcolumn:name="ProvisionDetails",type="string",JSONPath=".status.provisioningStatus.provisioningDetails"

// ProvisioningRequest is the Schema for the provisioningrequests API
// +operator-sdk:csv:customresourcedefinitions:displayName="ORAN O2IMS Provisioning Request",resources={{Namespace, v1},{ClusterInstance, siteconfig.open-cluster-management.io/v1alpha1}}
type ProvisioningRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProvisioningRequestSpec   `json:"spec,omitempty"`
	Status ProvisioningRequestStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ProvisioningRequestList contains a list of ProvisioningRequest
type ProvisioningRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProvisioningRequest `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ProvisioningRequest{}, &ProvisioningRequestList{})
}
