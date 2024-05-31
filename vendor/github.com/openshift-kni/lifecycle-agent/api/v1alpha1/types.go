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
)

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=imagebasedupgrades,scope=Cluster,shortName=ibu
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Desired Stage",type="string",JSONPath=".spec.stage"
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.conditions[-1:].reason"
// +kubebuilder:printcolumn:name="Details",type="string",JSONPath=".status.conditions[-1:].message"
// +kubebuilder:validation:XValidation:message="can not change spec.seedImageRef while ibu is in progress", rule="!has(oldSelf.status) || oldSelf.status.conditions.exists(c, c.type=='Idle' && c.status=='True') || has(oldSelf.spec.seedImageRef) && has(self.spec.seedImageRef) && oldSelf.spec.seedImageRef==self.spec.seedImageRef || !has(self.spec.seedImageRef) && !has(oldSelf.spec.seedImageRef)"
// +kubebuilder:validation:XValidation:message="can not change spec.oadpContent while ibu is in progress", rule="!has(oldSelf.status) || oldSelf.status.conditions.exists(c, c.type=='Idle' && c.status=='True') || has(oldSelf.spec.oadpContent) && has(self.spec.oadpContent) && oldSelf.spec.oadpContent==self.spec.oadpContent || !has(self.spec.oadpContent) && !has(oldSelf.spec.oadpContent)"
// +kubebuilder:validation:XValidation:message="can not change spec.extraManifests while ibu is in progress", rule="!has(oldSelf.status) || oldSelf.status.conditions.exists(c, c.type=='Idle' && c.status=='True') || has(oldSelf.spec.extraManifests) && has(self.spec.extraManifests) && oldSelf.spec.extraManifests==self.spec.extraManifests || !has(self.spec.extraManifests) && !has(oldSelf.spec.extraManifests)"
// +kubebuilder:validation:XValidation:message="can not change spec.autoRollbackOnFailure while ibu is in progress", rule="!has(oldSelf.status) || oldSelf.status.conditions.exists(c, c.type=='Idle' && c.status=='True') || has(oldSelf.spec.autoRollbackOnFailure) && has(self.spec.autoRollbackOnFailure) && oldSelf.spec.autoRollbackOnFailure==self.spec.autoRollbackOnFailure || !has(self.spec.autoRollbackOnFailure) && !has(oldSelf.spec.autoRollbackOnFailure)"
// +operator-sdk:csv:customresourcedefinitions:displayName="Image-based Cluster Upgrade",resources={{Namespace, v1},{Deployment,apps/v1}}
// ImageBasedUpgrade is the Schema for the ImageBasedUpgrades API
type ImageBasedUpgrade struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ImageBasedUpgradeSpec   `json:"spec,omitempty"`
	Status ImageBasedUpgradeStatus `json:"status,omitempty"`
}

// ImageBasedUpgradeStage defines the type for the IBU stage field
type ImageBasedUpgradeStage string

// Stages defines the string values for valid stages
var Stages = struct {
	Idle     ImageBasedUpgradeStage
	Prep     ImageBasedUpgradeStage
	Upgrade  ImageBasedUpgradeStage
	Rollback ImageBasedUpgradeStage
}{
	Idle:     "Idle",
	Prep:     "Prep",
	Upgrade:  "Upgrade",
	Rollback: "Rollback",
}

// ImageBasedUpgradeSpec defines the desired state of ImageBasedUpgrade
type ImageBasedUpgradeSpec struct {
	//+kubebuilder:validation:Enum=Idle;Prep;Upgrade;Rollback
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Stage"
	Stage ImageBasedUpgradeStage `json:"stage,omitempty"`
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Seed Image Reference"
	SeedImageRef SeedImageRef `json:"seedImageRef,omitempty"`
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Additional Images"
	AdditionalImages ConfigMapRef `json:"additionalImages,omitempty"`
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="OADP Content"
	OADPContent []ConfigMapRef `json:"oadpContent,omitempty"`
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Extra Manifests"
	ExtraManifests []ConfigMapRef `json:"extraManifests,omitempty"`
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Auto Rollback On Failure"
	AutoRollbackOnFailure AutoRollbackOnFailure `json:"autoRollbackOnFailure,omitempty"`
}

// SeedImageRef defines the seed image and OCP version for the upgrade
type SeedImageRef struct {
	//+operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	Version string `json:"version,omitempty"`
	//+operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	Image string `json:"image,omitempty"`
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Pull Secret Reference"
	PullSecretRef *PullSecretRef `json:"pullSecretRef,omitempty"`
}

type AutoRollbackOnFailure struct {
	//+operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	DisabledForPostRebootConfig bool `json:"disabledForPostRebootConfig,omitempty"` // If true, disable auto-rollback for post-reboot config service-unit(s)
	//+operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	DisabledForUpgradeCompletion bool `json:"disabledForUpgradeCompletion,omitempty"` // If true, disable auto-rollback for Upgrade completion handler
	//+operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	DisabledInitMonitor bool `json:"disabledInitMonitor,omitempty"` // If true, disable LCA Init Monitor watchdog, which triggers auto-rollback if timeout occurs before upgrade completion
	// +kubebuilder:validation:Minimum=0
	//+operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number"}
	InitMonitorTimeoutSeconds int `json:"initMonitorTimeoutSeconds,omitempty"` // LCA Init Monitor watchdog timeout, in seconds. Value = 0 is treated as "use default" when writing config file in Prep stage
}

// ConfigMapRef defines a reference to a config map
type ConfigMapRef struct {
	// +kubebuilder:validation:Required
	// +required
	//+operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	Name string `json:"name"`

	// +kubebuilder:validation:Required
	// +required
	//+operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	Namespace string `json:"namespace"`
}

// PullSecretRef defines a reference to a secret with credentials for pulling container images
type PullSecretRef struct {
	// +kubebuilder:validation:Required
	// +required
	//+operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	Name string `json:"name"`
}

// ImageBasedUpgradeStatus defines the observed state of ImageBasedUpgrade
type ImageBasedUpgradeStatus struct {
	ObservedGeneration int64       `json:"observedGeneration,omitempty"`
	StartedAt          metav1.Time `json:"startedAt,omitempty"`
	CompletedAt        metav1.Time `json:"completedAt,omitempty"`
	// +operator-sdk:csv:customresourcedefinitions:type=status,displayName="Conditions",xDescriptors={"urn:alm:descriptor:io.kubernetes.conditions"}
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// +operator-sdk:csv:customresourcedefinitions:type=status,displayName="Valid Next Stage"
	ValidNextStages []ImageBasedUpgradeStage `json:"validNextStages,omitempty"`
}

// +kubebuilder:object:root=true

// ImageBasedUpgradeList contains a list of ImageBasedUpgrade
type ImageBasedUpgradeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ImageBasedUpgrade `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ImageBasedUpgrade{}, &ImageBasedUpgradeList{})
}
