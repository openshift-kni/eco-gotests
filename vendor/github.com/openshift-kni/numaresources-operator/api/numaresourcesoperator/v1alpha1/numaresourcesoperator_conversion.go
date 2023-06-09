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
	"sigs.k8s.io/controller-runtime/pkg/conversion"

	nropv1 "github.com/openshift-kni/numaresources-operator/api/numaresourcesoperator/v1"
	mcov1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
)

var _ conversion.Convertible = &NUMAResourcesOperator{}

// ConvertTo converts this NUMAResourcesOperator to the Hub version (v1).
func (src *NUMAResourcesOperator) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*nropv1.NUMAResourcesOperator)
	if err := src.ConvertToV1NodeGroups(dst); err != nil {
		return err
	}
	// +kubebuilder:docs-gen:collapse=rote conversion
	return src.ConvertToV1Rote(dst)
}

// ConvertFrom converts from the Hub version (v1) to this version.
func (dst *NUMAResourcesOperator) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nropv1.NUMAResourcesOperator)
	// +kubebuilder:docs-gen:collapse=rote conversion
	return dst.ConvertFromV1Rote(src)
}

func (src *NUMAResourcesOperator) ConvertToV1NodeGroups(dst *nropv1.NUMAResourcesOperator) error {
	if src.Spec.NodeGroups != nil {
		dst.Spec.NodeGroups = make([]nropv1.NodeGroup, 0, len(src.Spec.NodeGroups))
		for idx := range src.Spec.NodeGroups {
			srcNG := &src.Spec.NodeGroups[idx] // shortcut
			srcConf := srcNG.NormalizeConfig() // handle DisablePodFingerprint
			dst.Spec.NodeGroups = append(dst.Spec.NodeGroups, nropv1.NodeGroup{
				MachineConfigPoolSelector: srcNG.MachineConfigPoolSelector.DeepCopy(),
				Config:                    convertNodeGroupConfigV1Alpha1ToV1(srcConf),
			})
		}
	}
	return nil
}

func (src *NUMAResourcesOperator) ConvertToV1Rote(dst *nropv1.NUMAResourcesOperator) error {
	// ObjectMeta
	dst.ObjectMeta = src.ObjectMeta
	// Spec
	dst.Spec.ExporterImage = src.Spec.ExporterImage
	dst.Spec.LogLevel = src.Spec.LogLevel
	if src.Spec.PodExcludes != nil {
		dst.Spec.PodExcludes = make([]nropv1.NamespacedName, 0, len(src.Spec.PodExcludes))
		for idx := range src.Spec.PodExcludes {
			dst.Spec.PodExcludes = append(dst.Spec.PodExcludes, nropv1.NamespacedName{
				Namespace: src.Spec.PodExcludes[idx].Namespace,
				Name:      src.Spec.PodExcludes[idx].Name,
			})
		}
	}
	// Status
	if src.Status.DaemonSets != nil {
		dst.Status.DaemonSets = make([]nropv1.NamespacedName, 0, len(src.Status.DaemonSets))
		for idx := range src.Status.DaemonSets {
			dst.Status.DaemonSets = append(dst.Status.DaemonSets, nropv1.NamespacedName{
				Namespace: src.Status.DaemonSets[idx].Namespace,
				Name:      src.Status.DaemonSets[idx].Name,
			})
		}
	}
	if src.Status.MachineConfigPools != nil {
		dst.Status.MachineConfigPools = make([]nropv1.MachineConfigPool, 0, len(src.Status.MachineConfigPools))
		for idx := range src.Status.MachineConfigPools {
			dst.Status.MachineConfigPools = append(dst.Status.MachineConfigPools, convertMachineConfigPoolV1Alpha1ToV1(src.Status.MachineConfigPools[idx]))
		}
	}
	if src.Status.Conditions != nil {
		dst.Status.Conditions = make([]metav1.Condition, len(src.Status.Conditions))
		copy(dst.Status.Conditions, src.Status.Conditions)
	}

	return nil
}

func (dst *NUMAResourcesOperator) ConvertFromV1Rote(src *nropv1.NUMAResourcesOperator) error {
	// ObjectMeta
	dst.ObjectMeta = src.ObjectMeta
	// Spec
	if src.Spec.NodeGroups != nil {
		dst.Spec.NodeGroups = make([]NodeGroup, 0, len(src.Spec.NodeGroups))
		for idx := range src.Spec.NodeGroups {
			srcNG := &src.Spec.NodeGroups[idx] // shortcut
			srcConf := *srcNG.Config           // shortcut, no normalization needed
			dst.Spec.NodeGroups = append(dst.Spec.NodeGroups, NodeGroup{
				MachineConfigPoolSelector: srcNG.MachineConfigPoolSelector.DeepCopy(),
				Config:                    convertNodeGroupConfigV1ToV1Alpha1(srcConf),
			})
		}
	}
	dst.Spec.ExporterImage = src.Spec.ExporterImage
	dst.Spec.LogLevel = src.Spec.LogLevel
	if src.Spec.PodExcludes != nil {
		dst.Spec.PodExcludes = make([]NamespacedName, len(src.Spec.PodExcludes))
		for idx := range src.Spec.PodExcludes {
			dst.Spec.PodExcludes[idx].Namespace = src.Spec.PodExcludes[idx].Namespace
			dst.Spec.PodExcludes[idx].Name = src.Spec.PodExcludes[idx].Name
		}
	}
	// Status
	if src.Status.DaemonSets != nil {
		dst.Status.DaemonSets = make([]NamespacedName, len(src.Status.DaemonSets))
		for idx := range src.Spec.PodExcludes {
			dst.Spec.PodExcludes[idx].Namespace = src.Spec.PodExcludes[idx].Namespace
			dst.Spec.PodExcludes[idx].Name = src.Spec.PodExcludes[idx].Name
		}
	}
	if src.Status.MachineConfigPools != nil {
		dst.Status.MachineConfigPools = make([]MachineConfigPool, 0, len(src.Status.MachineConfigPools))
		for idx := range src.Status.MachineConfigPools {
			dst.Status.MachineConfigPools = append(dst.Status.MachineConfigPools, convertMachineConfigPoolV1ToV1Alpha1(src.Status.MachineConfigPools[idx]))
		}
	}
	if src.Status.Conditions != nil {
		dst.Status.Conditions = make([]metav1.Condition, len(src.Status.Conditions))
		copy(dst.Status.Conditions, src.Status.Conditions)
	}

	return nil
}

func convertMachineConfigPoolV1Alpha1ToV1(src MachineConfigPool) nropv1.MachineConfigPool {
	dst := nropv1.MachineConfigPool{}
	dst.Name = src.Name
	if src.Conditions != nil {
		dst.Conditions = make([]mcov1.MachineConfigPoolCondition, len(src.Conditions))
		copy(dst.Conditions, src.Conditions)
	}
	if src.Config != nil {
		dst.Config = convertNodeGroupConfigV1Alpha1ToV1(*src.Config)
	}
	return dst
}

func convertMachineConfigPoolV1ToV1Alpha1(src nropv1.MachineConfigPool) MachineConfigPool {
	dst := MachineConfigPool{}
	dst.Name = src.Name
	if src.Conditions != nil {
		dst.Conditions = make([]mcov1.MachineConfigPoolCondition, len(src.Conditions))
		copy(dst.Conditions, src.Conditions)
	}
	if src.Config != nil {
		dst.Config = convertNodeGroupConfigV1ToV1Alpha1(*src.Config)
	}
	return dst
}

func convertNodeGroupConfigV1Alpha1ToV1(src NodeGroupConfig) *nropv1.NodeGroupConfig {
	dst := nropv1.NodeGroupConfig{}
	if src.PodsFingerprinting != nil {
		switch *src.PodsFingerprinting {
		case PodsFingerprintingEnabled:
			dst.PodsFingerprinting = ptrToPodsFingerPrintEnabledV1(nropv1.PodsFingerprintingEnabled)
		case PodsFingerprintingDisabled:
			dst.PodsFingerprinting = ptrToPodsFingerPrintEnabledV1(nropv1.PodsFingerprintingDisabled)
		}
	}
	if src.InfoRefreshMode != nil {
		switch *src.InfoRefreshMode {
		case InfoRefreshPeriodic:
			dst.InfoRefreshMode = ptrToInfoRefreshModeV1(nropv1.InfoRefreshPeriodic)
		case InfoRefreshEvents:
			dst.InfoRefreshMode = ptrToInfoRefreshModeV1(nropv1.InfoRefreshEvents)
		case InfoRefreshPeriodicAndEvents:
			dst.InfoRefreshMode = ptrToInfoRefreshModeV1(nropv1.InfoRefreshPeriodicAndEvents)
		}
	}
	if src.InfoRefreshPeriod != nil {
		dst.InfoRefreshPeriod = src.InfoRefreshPeriod.DeepCopy()
	}
	return &dst
}

func convertNodeGroupConfigV1ToV1Alpha1(src nropv1.NodeGroupConfig) *NodeGroupConfig {
	dst := NodeGroupConfig{}
	if src.PodsFingerprinting != nil {
		switch *src.PodsFingerprinting {
		case nropv1.PodsFingerprintingEnabledExclusiveResources:
			fallthrough
		case nropv1.PodsFingerprintingEnabled:
			dst.PodsFingerprinting = ptrToPodsFingerPrintEnabledV1Alpha1(PodsFingerprintingEnabled)
		case nropv1.PodsFingerprintingDisabled:
			dst.PodsFingerprinting = ptrToPodsFingerPrintEnabledV1Alpha1(PodsFingerprintingDisabled)
		}
	}
	if src.InfoRefreshMode != nil {
		switch *src.InfoRefreshMode {
		case nropv1.InfoRefreshPeriodic:
			dst.InfoRefreshMode = ptrToInfoRefreshModeV1Alpha1(InfoRefreshPeriodic)
		case nropv1.InfoRefreshEvents:
			dst.InfoRefreshMode = ptrToInfoRefreshModeV1Alpha1(InfoRefreshEvents)
		case nropv1.InfoRefreshPeriodicAndEvents:
			dst.InfoRefreshMode = ptrToInfoRefreshModeV1Alpha1(InfoRefreshPeriodicAndEvents)
		}
	}
	if src.InfoRefreshPeriod != nil {
		dst.InfoRefreshPeriod = src.InfoRefreshPeriod.DeepCopy()
	}
	return &dst
}

func ptrToPodsFingerPrintEnabledV1(v nropv1.PodsFingerprintingMode) *nropv1.PodsFingerprintingMode {
	return &v
}

func ptrToPodsFingerPrintEnabledV1Alpha1(v PodsFingerprintingMode) *PodsFingerprintingMode {
	return &v
}

func ptrToInfoRefreshModeV1(v nropv1.InfoRefreshMode) *nropv1.InfoRefreshMode {
	return &v
}

func ptrToInfoRefreshModeV1Alpha1(v InfoRefreshMode) *InfoRefreshMode {
	return &v
}
