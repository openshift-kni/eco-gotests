//go:build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AutoRollbackOnFailure) DeepCopyInto(out *AutoRollbackOnFailure) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AutoRollbackOnFailure.
func (in *AutoRollbackOnFailure) DeepCopy() *AutoRollbackOnFailure {
	if in == nil {
		return nil
	}
	out := new(AutoRollbackOnFailure)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ConfigMapRef) DeepCopyInto(out *ConfigMapRef) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ConfigMapRef.
func (in *ConfigMapRef) DeepCopy() *ConfigMapRef {
	if in == nil {
		return nil
	}
	out := new(ConfigMapRef)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ImageBasedUpgrade) DeepCopyInto(out *ImageBasedUpgrade) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ImageBasedUpgrade.
func (in *ImageBasedUpgrade) DeepCopy() *ImageBasedUpgrade {
	if in == nil {
		return nil
	}
	out := new(ImageBasedUpgrade)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ImageBasedUpgrade) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ImageBasedUpgradeList) DeepCopyInto(out *ImageBasedUpgradeList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ImageBasedUpgrade, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ImageBasedUpgradeList.
func (in *ImageBasedUpgradeList) DeepCopy() *ImageBasedUpgradeList {
	if in == nil {
		return nil
	}
	out := new(ImageBasedUpgradeList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ImageBasedUpgradeList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ImageBasedUpgradeSpec) DeepCopyInto(out *ImageBasedUpgradeSpec) {
	*out = *in
	in.SeedImageRef.DeepCopyInto(&out.SeedImageRef)
	if in.OADPContent != nil {
		in, out := &in.OADPContent, &out.OADPContent
		*out = make([]ConfigMapRef, len(*in))
		copy(*out, *in)
	}
	if in.ExtraManifests != nil {
		in, out := &in.ExtraManifests, &out.ExtraManifests
		*out = make([]ConfigMapRef, len(*in))
		copy(*out, *in)
	}
	out.AutoRollbackOnFailure = in.AutoRollbackOnFailure
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ImageBasedUpgradeSpec.
func (in *ImageBasedUpgradeSpec) DeepCopy() *ImageBasedUpgradeSpec {
	if in == nil {
		return nil
	}
	out := new(ImageBasedUpgradeSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ImageBasedUpgradeStatus) DeepCopyInto(out *ImageBasedUpgradeStatus) {
	*out = *in
	in.StartedAt.DeepCopyInto(&out.StartedAt)
	in.CompletedAt.DeepCopyInto(&out.CompletedAt)
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]v1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.ValidNextStages != nil {
		in, out := &in.ValidNextStages, &out.ValidNextStages
		*out = make([]ImageBasedUpgradeStage, len(*in))
		copy(*out, *in)
	}
	in.RollbackAvailabilityExpiration.DeepCopyInto(&out.RollbackAvailabilityExpiration)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ImageBasedUpgradeStatus.
func (in *ImageBasedUpgradeStatus) DeepCopy() *ImageBasedUpgradeStatus {
	if in == nil {
		return nil
	}
	out := new(ImageBasedUpgradeStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PullSecretRef) DeepCopyInto(out *PullSecretRef) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PullSecretRef.
func (in *PullSecretRef) DeepCopy() *PullSecretRef {
	if in == nil {
		return nil
	}
	out := new(PullSecretRef)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SeedImageRef) DeepCopyInto(out *SeedImageRef) {
	*out = *in
	if in.PullSecretRef != nil {
		in, out := &in.PullSecretRef, &out.PullSecretRef
		*out = new(PullSecretRef)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SeedImageRef.
func (in *SeedImageRef) DeepCopy() *SeedImageRef {
	if in == nil {
		return nil
	}
	out := new(SeedImageRef)
	in.DeepCopyInto(out)
	return out
}
