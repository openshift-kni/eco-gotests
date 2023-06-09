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
)

var _ conversion.Convertible = &NUMAResourcesScheduler{}

// ConvertTo converts this NUMAResourcesScheduler to the Hub version (v1).
func (src *NUMAResourcesScheduler) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*nropv1.NUMAResourcesScheduler)
	// +kubebuilder:docs-gen:collapse=rote conversion
	return src.ConvertToV1Rote(dst)
}

// ConvertFrom converts from the Hub version (v1) to this version.
func (dst *NUMAResourcesScheduler) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*nropv1.NUMAResourcesScheduler)
	// +kubebuilder:docs-gen:collapse=rote conversion
	return dst.ConvertFromV1Rote(src)
}

func (src *NUMAResourcesScheduler) ConvertToV1Rote(dst *nropv1.NUMAResourcesScheduler) error {
	// ObjectMeta
	dst.ObjectMeta = src.ObjectMeta
	// Spec
	dst.Spec.SchedulerImage = src.Spec.SchedulerImage
	dst.Spec.SchedulerName = src.Spec.SchedulerName
	dst.Spec.LogLevel = src.Spec.LogLevel
	if src.Spec.CacheResyncPeriod != nil {
		dst.Spec.CacheResyncPeriod = src.Spec.CacheResyncPeriod.DeepCopy()
	}
	// Status
	dst.Status.Deployment = nropv1.NamespacedName{
		Namespace: src.Status.Deployment.Namespace,
		Name:      src.Status.Deployment.Name,
	}
	dst.Status.SchedulerName = src.Status.SchedulerName
	if src.Status.Conditions != nil {
		dst.Status.Conditions = make([]metav1.Condition, len(src.Status.Conditions))
		copy(dst.Status.Conditions, src.Status.Conditions)
	}
	return nil
}

func (dst *NUMAResourcesScheduler) ConvertFromV1Rote(src *nropv1.NUMAResourcesScheduler) error {
	// ObjectMeta
	dst.ObjectMeta = src.ObjectMeta
	// Spec
	dst.Spec.SchedulerImage = src.Spec.SchedulerImage
	dst.Spec.SchedulerName = src.Spec.SchedulerName
	dst.Spec.LogLevel = src.Spec.LogLevel
	if src.Spec.CacheResyncPeriod != nil {
		dst.Spec.CacheResyncPeriod = src.Spec.CacheResyncPeriod.DeepCopy()
	}
	// Status
	dst.Status.Deployment = NamespacedName{
		Namespace: src.Status.Deployment.Namespace,
		Name:      src.Status.Deployment.Name,
	}
	dst.Status.SchedulerName = src.Status.SchedulerName
	if src.Status.Conditions != nil {
		dst.Status.Conditions = make([]metav1.Condition, len(src.Status.Conditions))
		copy(dst.Status.Conditions, src.Status.Conditions)
	}
	return nil

}
