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
// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1alpha1

// PreCachingConfigCRApplyConfiguration represents an declarative configuration of the PreCachingConfigCR type for use
// with apply.
type PreCachingConfigCRApplyConfiguration struct {
	Name      *string `json:"name,omitempty"`
	Namespace *string `json:"namespace,omitempty"`
}

// PreCachingConfigCRApplyConfiguration constructs an declarative configuration of the PreCachingConfigCR type for use with
// apply.
func PreCachingConfigCR() *PreCachingConfigCRApplyConfiguration {
	return &PreCachingConfigCRApplyConfiguration{}
}

// WithName sets the Name field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Name field is set to the value of the last call.
func (b *PreCachingConfigCRApplyConfiguration) WithName(value string) *PreCachingConfigCRApplyConfiguration {
	b.Name = &value
	return b
}

// WithNamespace sets the Namespace field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Namespace field is set to the value of the last call.
func (b *PreCachingConfigCRApplyConfiguration) WithNamespace(value string) *PreCachingConfigCRApplyConfiguration {
	b.Namespace = &value
	return b
}
