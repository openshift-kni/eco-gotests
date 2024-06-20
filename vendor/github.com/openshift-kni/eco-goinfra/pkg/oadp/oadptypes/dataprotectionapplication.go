/*
Copyright 2021.

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

package oadptypes

import (
	velero "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// DPAGroupVersion is group version used to register these objects.
	DPAGroupVersion = schema.GroupVersion{Group: "oadp.openshift.io", Version: "v1alpha1"}

	// DPASchemeBuilder is used to add go types to the GroupVersionKind scheme.
	DPASchemeBuilder = &scheme.Builder{GroupVersion: DPAGroupVersion}

	// DPAAddToScheme adds the types in this group-version to the given scheme.
	DPAAddToScheme = DPASchemeBuilder.AddToScheme
)

// ConditionReconciled represents Reconciled condition.
const ConditionReconciled = "Reconciled"

// ReconciledReasonComplete represents Reconciled condition complete reason.
const ReconciledReasonComplete = "Complete"

// ReconciledReasonError represents Reconciled condition error reason.
const ReconciledReasonError = "Error"

// ReconcileCompleteMessage represents Reconciled condition complete message.
const ReconcileCompleteMessage = "Reconcile complete"

// DefaultPlugin contains string name of plugin.
type DefaultPlugin string

// DefaultPluginAWS represents aws plugin name.
const DefaultPluginAWS DefaultPlugin = "aws"

// DefaultPluginGCP represents gcp plugin name.
const DefaultPluginGCP DefaultPlugin = "gcp"

// DefaultPluginMicrosoftAzure represents azure plugin name.
const DefaultPluginMicrosoftAzure DefaultPlugin = "azure"

// DefaultPluginCSI represents csi plugin name.
const DefaultPluginCSI DefaultPlugin = "csi"

// DefaultPluginVSM represents vsm plugin name.
const DefaultPluginVSM DefaultPlugin = "vsm"

// DefaultPluginOpenShift represents openshift plugin name.
const DefaultPluginOpenShift DefaultPlugin = "openshift"

// DefaultPluginKubeVirt represents kubevirt plugin name.
const DefaultPluginKubeVirt DefaultPlugin = "kubevirt"

// CustomPlugin sets name and image of custom plugin.
type CustomPlugin struct {
	Name  string `json:"name"`
	Image string `json:"image"`
}

// UnsupportedImageKey field does not have enum validation for development flexibility.
type UnsupportedImageKey string

// VeleroConfig contains velero configuration.
type VeleroConfig struct {
	// featureFlags defines the list of features to enable for Velero instance
	// +optional
	FeatureFlags   []string        `json:"featureFlags,omitempty"`
	DefaultPlugins []DefaultPlugin `json:"defaultPlugins,omitempty"`
	// customPlugins defines the custom plugin to be installed with Velero
	// +optional
	CustomPlugins []CustomPlugin `json:"customPlugins,omitempty"`
	// restoreResourceVersionPriority represents a configmap that will be created if
	// defined for use in conjunction with EnableAPIGroupVersions feature flag
	// Defining this field automatically add EnableAPIGroupVersions to the velero server feature flag
	// +optional
	RestoreResourcesVersionPriority string `json:"restoreResourcesVersionPriority,omitempty"`
	// If you need to install Velero without a default backup storage location noDefaultBackupLocation
	// flag is required for confirmation
	// +optional
	NoDefaultBackupLocation bool `json:"noDefaultBackupLocation,omitempty"`
	// Pod specific configuration
	PodConfig *PodConfig `json:"podConfig,omitempty"`
	// Velero server’s log level (use debug for the most logging, leave unset for velero default)
	// +optional
	// +kubebuilder:validation:Enum=trace;debug;info;warning;error;fatal;panic
	LogLevel string `json:"logLevel,omitempty"`
	// How often to check status on async backup/restore operations after backup processing. Default value is 2m.
	// +optional
	ItemOperationSyncFrequency string `json:"itemOperationSyncFrequency,omitempty"`
	// How long to wait on asynchronous BackupItemActions and RestoreItemActions
	// to complete before timing out. Default value is 1h.
	// +optional
	DefaultItemOperationTimeout string `json:"defaultItemOperationTimeout,omitempty"`
	// Use pod volume file system backup by default for volumes
	// +optional
	DefaultVolumesToFSBackup *bool `json:"defaultVolumesToFSBackup,omitempty"`
	// Specify whether CSI snapshot data should be moved to backup storage by default
	// +optional
	DefaultSnapshotMoveData *bool `json:"defaultSnapshotMoveData,omitempty"`
	// Disable informer cache for Get calls on restore. With this enabled, it will speed up restore in cases where
	// there are backup resources which already exist in the cluster, but for very large clusters this will increase
	// velero memory usage. Default is false.
	// +optional
	DisableInformerCache *bool `json:"disableInformerCache,omitempty"`
	// resourceTimeout defines how long to wait for several Velero resources before timeout occurs,
	// such as Velero CRD availability, volumeSnapshot deletion, and repo availability.
	// Default is 10m
	// +optional
	ResourceTimeout string `json:"resourceTimeout,omitempty"`
}

// PodConfig defines the pod configuration options.
type PodConfig struct {
	// labels to add to pods
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
	// nodeSelector defines the nodeSelector to be supplied to podSpec
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// tolerations defines the list of tolerations to be applied to daemonset
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
	// resourceAllocations defines the CPU and Memory resource allocations for the Pod
	// +optional
	// +nullable
	ResourceAllocations corev1.ResourceRequirements `json:"resourceAllocations,omitempty"`
	// env defines the list of environment variables to be supplied to podSpec
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`
}

// NodeAgentCommonFields defines node agent fields.
type NodeAgentCommonFields struct {
	// enable defines a boolean pointer whether we want the daemonset to
	// exist or not
	// +optional
	Enable *bool `json:"enable,omitempty"`
	// supplementalGroups defines the linux groups to be applied to the NodeAgent Pod
	// +optional
	SupplementalGroups []int64 `json:"supplementalGroups,omitempty"`
	// timeout defines the NodeAgent timeout, default value is 1h
	// +optional
	Timeout string `json:"timeout,omitempty"`
	// Pod specific configuration
	PodConfig *PodConfig `json:"podConfig,omitempty"`
}

// NodeAgentConfig is the configuration for node server.
type NodeAgentConfig struct {
	// Embedding NodeAgentCommonFields
	// +optional
	NodeAgentCommonFields `json:",inline"`

	// The type of uploader to transfer the data of pod volumes, the supported values are 'restic' or 'kopia'
	// +kubebuilder:validation:Enum=restic;kopia
	// +kubebuilder:validation:Required
	UploaderType string `json:"uploaderType"`
}

// ResticConfig is the configuration for restic server.
type ResticConfig struct {
	// Embedding NodeAgentCommonFields
	// +optional
	NodeAgentCommonFields `json:",inline"`
}

// ApplicationConfig defines the configuration for the Data Protection Application.
type ApplicationConfig struct {
	Velero *VeleroConfig `json:"velero,omitempty"`
	// (deprecation warning) ResticConfig is the configuration for restic DaemonSet.
	// restic is for backwards compatibility and is replaced by the nodeAgent
	// restic will be removed with the OADP 1.4
	// +kubebuilder:deprecatedversion:warning=1.3
	// +optional
	Restic *ResticConfig `json:"restic,omitempty"`

	// NodeAgent is needed to allow selection between kopia or restic
	// +optional
	NodeAgent *NodeAgentConfig `json:"nodeAgent,omitempty"`
}

// CloudStorageLocation defines BackupStorageLocation using bucket referenced by CloudStorage CR.
type CloudStorageLocation struct {
	CloudStorageRef corev1.LocalObjectReference `json:"cloudStorageRef"`
	// config is for provider-specific configuration fields.
	// +optional
	Config map[string]string `json:"config,omitempty"`

	// credential contains the credential information intended to be used with this location
	// +optional
	Credential *corev1.SecretKeySelector `json:"credential,omitempty"`

	// default indicates this location is the default backup storage location.
	// +optional
	Default bool `json:"default,omitempty"`

	// backupSyncPeriod defines how frequently to sync backup API objects from object storage. A value of 0 disables sync.
	// +optional
	// +nullable
	BackupSyncPeriod *metav1.Duration `json:"backupSyncPeriod,omitempty"`

	// Prefix and CACert are copied from velero/pkg/apis/v1/backupstoragelocation_types.go under ObjectStorageLocation

	// Prefix is the path inside a bucket to use for Velero storage. Optional.
	// +optional
	Prefix string `json:"prefix,omitempty"`

	// CACert defines a CA bundle to use when verifying TLS connections to the provider.
	// +optional
	CACert []byte `json:"caCert,omitempty"`
}

// BackupLocation defines the configuration for the DPA backup storage.
type BackupLocation struct {
	// +optional
	Name string `json:"name,omitempty"`
	// +optional
	Velero *velero.BackupStorageLocationSpec `json:"velero,omitempty"`
	// +optional
	CloudStorage *CloudStorageLocation `json:"bucket,omitempty"`
}

// SnapshotLocation defines the configuration for the DPA snapshot store.
type SnapshotLocation struct {
	Velero *velero.VolumeSnapshotLocationSpec `json:"velero"`
}

// NonAdmin sets nonadmin features.
type NonAdmin struct {
	// Enables non admin feature, by default is disabled
	// +optional
	Enable *bool `json:"enable,omitempty"`
}

// DataMover defines the various config for DPA data mover.
type DataMover struct {
	// enable flag is used to specify whether you want to deploy the volume snapshot mover controller
	// +optional
	Enable bool `json:"enable,omitempty"`
	// User supplied Restic Secret name
	// +optional
	CredentialName string `json:"credentialName,omitempty"`
	// User supplied timeout to be used for VolumeSnapshotBackup and VolumeSnapshotRestore
	// to complete, default value is 10m
	// +optional
	Timeout string `json:"timeout,omitempty"`
	// the number of batched volumeSnapshotBackups that can be inProgress at once, default value is 10
	// +optional
	MaxConcurrentBackupVolumes string `json:"maxConcurrentBackupVolumes,omitempty"`
	// the number of batched volumeSnapshotRestores that can be inProgress at once, default value is 10
	// +optional
	MaxConcurrentRestoreVolumes string `json:"maxConcurrentRestoreVolumes,omitempty"`
	// defines how often (in days) to prune the datamover snapshots from the repository
	// +optional
	PruneInterval string `json:"pruneInterval,omitempty"`
	// defines configurations for data mover volume options for a storageClass
	// +optional
	VolumeOptionsForStorageClasses map[string]DataMoverVolumeOptions `json:"volumeOptionsForStorageClasses,omitempty"`
	// defines the parameters that can be specified for retention of datamover snapshots
	// +optional
	SnapshotRetainPolicy *RetainPolicy `json:"snapshotRetainPolicy,omitempty"`
	// schedule is a cronspec (https://en.wikipedia.org/wiki/Cron#Overview) that
	// can be used to schedule datamover(volsync) synchronization to occur at regular, time-based
	// intervals. For example, in order to enforce datamover SnapshotRetainPolicy at a regular interval you need to
	// specify this Schedule trigger as a cron expression, by default the trigger is a manual trigger. For more details
	// on Volsync triggers, refer: https://volsync.readthedocs.io/en/stable/usage/triggers.html
	//+kubebuilder:validation:Pattern=`^(\d+|\*)(/\d+)?(\s+(\d+|\*)(/\d+)?){4}$`
	//+optional
	Schedule string `json:"schedule,omitempty"`
}

// RetainPolicy defines the fields for retention of datamover snapshots.
type RetainPolicy struct {
	// Hourly defines the number of snapshots to be kept hourly
	//+optional
	Hourly string `json:"hourly,omitempty"`
	// Daily defines the number of snapshots to be kept daily
	//+optional
	Daily string `json:"daily,omitempty"`
	// Weekly defines the number of snapshots to be kept weekly
	//+optional
	Weekly string `json:"weekly,omitempty"`
	// Monthly defines the number of snapshots to be kept monthly
	//+optional
	Monthly string `json:"monthly,omitempty"`
	// Yearly defines the number of snapshots to be kept yearly
	//+optional
	Yearly string `json:"yearly,omitempty"`
	// Within defines the number of snapshots to be kept Within the given time period
	//+optional
	Within string `json:"within,omitempty"`
}

// DataMoverVolumeOptions define data volume options.
type DataMoverVolumeOptions struct {
	SourceVolumeOptions      *VolumeOptions `json:"sourceVolumeOptions,omitempty"`
	DestinationVolumeOptions *VolumeOptions `json:"destinationVolumeOptions,omitempty"`
}

// VolumeOptions defines configurations for VolSync options.
type VolumeOptions struct {
	// storageClassName can be used to override the StorageClass of the source
	// or destination PVC
	//+optional
	StorageClassName string `json:"storageClassName,omitempty"`
	// accessMode can be used to override the accessMode of the source or
	// destination PVC
	//+optional
	AccessMode corev1.PersistentVolumeAccessMode `json:"accessMode,omitempty"`
	// cacheStorageClassName is the storageClass that should be used when provisioning
	// the data mover cache volume
	//+optional
	CacheStorageClassName string `json:"cacheStorageClassName,omitempty"`
	// cacheCapacity determines the size of the restic metadata cache volume
	//+optional
	CacheCapacity string `json:"cacheCapacity,omitempty"`
	// cacheAccessMode is the access mode to be used to provision the cache volume
	//+optional
	CacheAccessMode string `json:"cacheAccessMode,omitempty"`
}

// Features defines the configuration for the DPA to enable the tech preview features.
type Features struct {
	// (do not use warning) dataMover is for backwards compatibility and
	// will be removed in the future. Use Velero Built-in Data Mover instead
	// +optional
	DataMover *DataMover `json:"dataMover,omitempty"`
}

// DataProtectionApplicationSpec defines the desired state of Velero.
type DataProtectionApplicationSpec struct {
	// backupLocations defines the list of desired configuration to use for BackupStorageLocations
	// +optional
	BackupLocations []BackupLocation `json:"backupLocations"`
	// snapshotLocations defines the list of desired configuration to use for VolumeSnapshotLocations
	// +optional
	SnapshotLocations []SnapshotLocation `json:"snapshotLocations"`
	// unsupportedOverrides can be used to override images used in deployments.
	// Available keys are:
	//   - veleroImageFqin
	//   - awsPluginImageFqin
	//   - openshiftPluginImageFqin
	//   - azurePluginImageFqin
	//   - gcpPluginImageFqin
	//   - csiPluginImageFqin
	//   - resticRestoreImageFqin
	//   - kubevirtPluginImageFqin
	//   - nonAdminControllerImageFqin
	//   - operator-type
	//   - tech-preview-ack
	// +optional
	UnsupportedOverrides map[UnsupportedImageKey]string `json:"unsupportedOverrides,omitempty"`
	// add annotations to pods deployed by operator
	// +optional
	PodAnnotations map[string]string `json:"podAnnotations,omitempty"`
	// podDnsPolicy defines how a pod's DNS will be configured.
	// https://kubernetes.io/docs/concepts/services-networking/dns-pod-service/#pod-s-dns-policy
	// +optional
	PodDNSPolicy corev1.DNSPolicy `json:"podDnsPolicy,omitempty"`
	// podDnsConfig defines the DNS parameters of a pod in addition to
	// those generated from DNSPolicy.
	// https://kubernetes.io/docs/concepts/services-networking/dns-pod-service/#pod-dns-config
	// +optional
	PodDNSConfig corev1.PodDNSConfig `json:"podDnsConfig,omitempty"`
	// backupImages is used to specify whether you want to deploy a registry for enabling backup and restore of images
	// +optional
	BackupImages *bool `json:"backupImages,omitempty"`
	// configuration is used to configure the data protection application's server config
	Configuration *ApplicationConfig `json:"configuration"`
	// features defines the configuration for the DPA to enable the OADP tech preview features
	// +optional
	Features *Features `json:"features"`
	// nonAdmin defines the configuration for the DPA to enable backup and restore operations for non-admin users
	// +optional
	NonAdmin *NonAdmin `json:"nonAdmin,omitempty"`
}

// DataProtectionApplicationStatus defines the observed state of DataProtectionApplication.
type DataProtectionApplicationStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=dataprotectionapplications,shortName=dpa

// DataProtectionApplication is the Schema for the dpa API.
type DataProtectionApplication struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DataProtectionApplicationSpec   `json:"spec,omitempty"`
	Status DataProtectionApplicationStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DataProtectionApplicationList contains a list of Velero.
type DataProtectionApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DataProtectionApplication `json:"items"`
}

// BackupImages behavior when nil to true.
//
//nolint:stylecheck
func (dpa *DataProtectionApplication) BackupImages() bool {
	return dpa.Spec.BackupImages == nil || *dpa.Spec.BackupImages
}

// GetDisableInformerCache defaults DisableInformerCache behavior when nil to false.
func (dpa *DataProtectionApplication) GetDisableInformerCache() bool {
	if dpa.Spec.Configuration.Velero.DisableInformerCache == nil {
		return false
	}

	return *dpa.Spec.Configuration.Velero.DisableInformerCache
}

// HasFeatureFlag returns current state of velero featureFlag.
func (veleroConfig *VeleroConfig) HasFeatureFlag(flag string) bool {
	for _, featureFlag := range veleroConfig.FeatureFlags {
		if featureFlag == flag {
			return true
		}
	}

	return false
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DataProtectionApplication) DeepCopyInto(out *DataProtectionApplication) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DataProtectionApplication.
func (in *DataProtectionApplication) DeepCopy() *DataProtectionApplication {
	if in == nil {
		return nil
	}

	out := new(DataProtectionApplication)
	in.DeepCopyInto(out)

	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
//
//nolint:ireturn
func (in *DataProtectionApplication) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}

	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DataProtectionApplicationSpec) DeepCopyInto(out *DataProtectionApplicationSpec) {
	*out = *in
	if in.BackupLocations != nil {
		in, out := &in.BackupLocations, &out.BackupLocations
		*out = make([]BackupLocation, len(*in))

		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}

	if in.SnapshotLocations != nil {
		in, out := &in.SnapshotLocations, &out.SnapshotLocations
		*out = make([]SnapshotLocation, len(*in))

		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}

	if in.UnsupportedOverrides != nil {
		in, out := &in.UnsupportedOverrides, &out.UnsupportedOverrides
		*out = make(map[UnsupportedImageKey]string, len(*in))

		for key, val := range *in {
			(*out)[key] = val
		}
	}

	if in.PodAnnotations != nil {
		in, out := &in.PodAnnotations, &out.PodAnnotations
		*out = make(map[string]string, len(*in))

		for key, val := range *in {
			(*out)[key] = val
		}
	}

	in.PodDNSConfig.DeepCopyInto(&out.PodDNSConfig)

	if in.BackupImages != nil {
		in, out := &in.BackupImages, &out.BackupImages
		*out = new(bool)
		**out = **in
	}

	if in.Configuration != nil {
		in, out := &in.Configuration, &out.Configuration
		*out = new(ApplicationConfig)
		(*in).DeepCopyInto(*out)
	}

	if in.Features != nil {
		in, out := &in.Features, &out.Features
		*out = new(Features)
		(*in).DeepCopyInto(*out)
	}

	if in.NonAdmin != nil {
		in, out := &in.NonAdmin, &out.NonAdmin
		*out = new(NonAdmin)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DataProtectionApplicationSpec.
func (in *DataProtectionApplicationSpec) DeepCopy() *DataProtectionApplicationSpec {
	if in == nil {
		return nil
	}

	out := new(DataProtectionApplicationSpec)
	in.DeepCopyInto(out)

	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DataProtectionApplicationStatus) DeepCopyInto(out *DataProtectionApplicationStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]metav1.Condition, len(*in))

		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DataProtectionApplicationStatus.
func (in *DataProtectionApplicationStatus) DeepCopy() *DataProtectionApplicationStatus {
	if in == nil {
		return nil
	}

	out := new(DataProtectionApplicationStatus)
	in.DeepCopyInto(out)

	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BackupLocation) DeepCopyInto(out *BackupLocation) {
	*out = *in
	if in.Velero != nil {
		in, out := &in.Velero, &out.Velero
		*out = new(velero.BackupStorageLocationSpec)
		(*in).DeepCopyInto(*out)
	}

	if in.CloudStorage != nil {
		in, out := &in.CloudStorage, &out.CloudStorage
		*out = new(CloudStorageLocation)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BackupLocation.
func (in *BackupLocation) DeepCopy() *BackupLocation {
	if in == nil {
		return nil
	}

	out := new(BackupLocation)
	in.DeepCopyInto(out)

	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SnapshotLocation) DeepCopyInto(out *SnapshotLocation) {
	*out = *in
	if in.Velero != nil {
		in, out := &in.Velero, &out.Velero
		*out = new(velero.VolumeSnapshotLocationSpec)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SnapshotLocation.
func (in *SnapshotLocation) DeepCopy() *SnapshotLocation {
	if in == nil {
		return nil
	}

	out := new(SnapshotLocation)
	in.DeepCopyInto(out)

	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApplicationConfig) DeepCopyInto(out *ApplicationConfig) {
	*out = *in
	if in.Velero != nil {
		in, out := &in.Velero, &out.Velero
		*out = new(VeleroConfig)
		(*in).DeepCopyInto(*out)
	}

	if in.Restic != nil {
		in, out := &in.Restic, &out.Restic
		*out = new(ResticConfig)
		(*in).DeepCopyInto(*out)
	}

	if in.NodeAgent != nil {
		in, out := &in.NodeAgent, &out.NodeAgent
		*out = new(NodeAgentConfig)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApplicationConfig.
func (in *ApplicationConfig) DeepCopy() *ApplicationConfig {
	if in == nil {
		return nil
	}

	out := new(ApplicationConfig)
	in.DeepCopyInto(out)

	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Features) DeepCopyInto(out *Features) {
	*out = *in
	if in.DataMover != nil {
		in, out := &in.DataMover, &out.DataMover
		*out = new(DataMover)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Features.
func (in *Features) DeepCopy() *Features {
	if in == nil {
		return nil
	}

	out := new(Features)
	in.DeepCopyInto(out)

	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NonAdmin) DeepCopyInto(out *NonAdmin) {
	*out = *in
	if in.Enable != nil {
		in, out := &in.Enable, &out.Enable
		*out = new(bool)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NonAdmin.
func (in *NonAdmin) DeepCopy() *NonAdmin {
	if in == nil {
		return nil
	}

	out := new(NonAdmin)
	in.DeepCopyInto(out)

	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CloudStorageLocation) DeepCopyInto(out *CloudStorageLocation) {
	*out = *in
	out.CloudStorageRef = in.CloudStorageRef

	if in.Config != nil {
		in, out := &in.Config, &out.Config
		*out = make(map[string]string, len(*in))

		for key, val := range *in {
			(*out)[key] = val
		}
	}

	if in.Credential != nil {
		in, out := &in.Credential, &out.Credential
		*out = new(corev1.SecretKeySelector)
		(*in).DeepCopyInto(*out)
	}

	if in.BackupSyncPeriod != nil {
		in, out := &in.BackupSyncPeriod, &out.BackupSyncPeriod
		*out = new(metav1.Duration)
		**out = **in
	}

	if in.CACert != nil {
		in, out := &in.CACert, &out.CACert
		*out = make([]byte, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CloudStorageLocation.
func (in *CloudStorageLocation) DeepCopy() *CloudStorageLocation {
	if in == nil {
		return nil
	}

	out := new(CloudStorageLocation)
	in.DeepCopyInto(out)

	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VeleroConfig) DeepCopyInto(out *VeleroConfig) {
	*out = *in
	if in.FeatureFlags != nil {
		in, out := &in.FeatureFlags, &out.FeatureFlags
		*out = make([]string, len(*in))
		copy(*out, *in)
	}

	if in.DefaultPlugins != nil {
		in, out := &in.DefaultPlugins, &out.DefaultPlugins
		*out = make([]DefaultPlugin, len(*in))
		copy(*out, *in)
	}

	if in.CustomPlugins != nil {
		in, out := &in.CustomPlugins, &out.CustomPlugins
		*out = make([]CustomPlugin, len(*in))
		copy(*out, *in)
	}

	if in.PodConfig != nil {
		in, out := &in.PodConfig, &out.PodConfig
		*out = new(PodConfig)
		(*in).DeepCopyInto(*out)
	}

	if in.DefaultVolumesToFSBackup != nil {
		in, out := &in.DefaultVolumesToFSBackup, &out.DefaultVolumesToFSBackup
		*out = new(bool)
		**out = **in
	}

	if in.DefaultSnapshotMoveData != nil {
		in, out := &in.DefaultSnapshotMoveData, &out.DefaultSnapshotMoveData
		*out = new(bool)
		**out = **in
	}

	if in.DisableInformerCache != nil {
		in, out := &in.DisableInformerCache, &out.DisableInformerCache
		*out = new(bool)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VeleroConfig.
//
//nolint:stylecheck
func (in *VeleroConfig) DeepCopy() *VeleroConfig {
	if in == nil {
		return nil
	}

	out := new(VeleroConfig)
	in.DeepCopyInto(out)

	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ResticConfig) DeepCopyInto(out *ResticConfig) {
	*out = *in
	in.NodeAgentCommonFields.DeepCopyInto(&out.NodeAgentCommonFields)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ResticConfig.
func (in *ResticConfig) DeepCopy() *ResticConfig {
	if in == nil {
		return nil
	}

	out := new(ResticConfig)
	in.DeepCopyInto(out)

	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NodeAgentConfig) DeepCopyInto(out *NodeAgentConfig) {
	*out = *in
	in.NodeAgentCommonFields.DeepCopyInto(&out.NodeAgentCommonFields)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NodeAgentConfig.
func (in *NodeAgentConfig) DeepCopy() *NodeAgentConfig {
	if in == nil {
		return nil
	}

	out := new(NodeAgentConfig)
	in.DeepCopyInto(out)

	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DataMover) DeepCopyInto(out *DataMover) {
	*out = *in
	if in.VolumeOptionsForStorageClasses != nil {
		in, out := &in.VolumeOptionsForStorageClasses, &out.VolumeOptionsForStorageClasses
		*out = make(map[string]DataMoverVolumeOptions, len(*in))

		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}

	if in.SnapshotRetainPolicy != nil {
		in, out := &in.SnapshotRetainPolicy, &out.SnapshotRetainPolicy
		*out = new(RetainPolicy)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DataMover.
func (in *DataMover) DeepCopy() *DataMover {
	if in == nil {
		return nil
	}

	out := new(DataMover)
	in.DeepCopyInto(out)

	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DataMoverVolumeOptions) DeepCopyInto(out *DataMoverVolumeOptions) {
	*out = *in
	if in.SourceVolumeOptions != nil {
		in, out := &in.SourceVolumeOptions, &out.SourceVolumeOptions
		*out = new(VolumeOptions)
		**out = **in
	}

	if in.DestinationVolumeOptions != nil {
		in, out := &in.DestinationVolumeOptions, &out.DestinationVolumeOptions
		*out = new(VolumeOptions)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DataMoverVolumeOptions.
func (in *DataMoverVolumeOptions) DeepCopy() *DataMoverVolumeOptions {
	if in == nil {
		return nil
	}

	out := new(DataMoverVolumeOptions)
	in.DeepCopyInto(out)

	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PodConfig) DeepCopyInto(out *PodConfig) {
	*out = *in
	if in.Labels != nil {
		in, out := &in.Labels, &out.Labels
		*out = make(map[string]string, len(*in))

		for key, val := range *in {
			(*out)[key] = val
		}
	}

	if in.NodeSelector != nil {
		in, out := &in.NodeSelector, &out.NodeSelector
		*out = make(map[string]string, len(*in))

		for key, val := range *in {
			(*out)[key] = val
		}
	}

	if in.Tolerations != nil {
		in, out := &in.Tolerations, &out.Tolerations
		*out = make([]corev1.Toleration, len(*in))

		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}

	in.ResourceAllocations.DeepCopyInto(&out.ResourceAllocations)

	if in.Env != nil {
		in, out := &in.Env, &out.Env
		*out = make([]corev1.EnvVar, len(*in))

		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PodConfig.
func (in *PodConfig) DeepCopy() *PodConfig {
	if in == nil {
		return nil
	}

	out := new(PodConfig)
	in.DeepCopyInto(out)

	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NodeAgentCommonFields) DeepCopyInto(out *NodeAgentCommonFields) {
	*out = *in
	if in.Enable != nil {
		in, out := &in.Enable, &out.Enable
		*out = new(bool)
		**out = **in
	}

	if in.SupplementalGroups != nil {
		in, out := &in.SupplementalGroups, &out.SupplementalGroups
		*out = make([]int64, len(*in))
		copy(*out, *in)
	}

	if in.PodConfig != nil {
		in, out := &in.PodConfig, &out.PodConfig
		*out = new(PodConfig)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NodeAgentCommonFields.
func (in *NodeAgentCommonFields) DeepCopy() *NodeAgentCommonFields {
	if in == nil {
		return nil
	}

	out := new(NodeAgentCommonFields)
	in.DeepCopyInto(out)

	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DataProtectionApplicationList) DeepCopyInto(out *DataProtectionApplicationList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)

	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]DataProtectionApplication, len(*in))

		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DataProtectionApplicationList.
func (in *DataProtectionApplicationList) DeepCopy() *DataProtectionApplicationList {
	if in == nil {
		return nil
	}

	out := new(DataProtectionApplicationList)
	in.DeepCopyInto(out)

	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
//
//nolint:ireturn
func (in *DataProtectionApplicationList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}

	return nil
}
