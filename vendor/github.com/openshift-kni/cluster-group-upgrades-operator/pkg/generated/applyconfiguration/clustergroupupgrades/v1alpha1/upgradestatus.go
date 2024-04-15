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

import (
	v1alpha1 "github.com/openshift-kni/cluster-group-upgrades-operator/pkg/api/clustergroupupgrades/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// UpgradeStatusApplyConfiguration represents an declarative configuration of the UpgradeStatus type for use
// with apply.
type UpgradeStatusApplyConfiguration struct {
	StartedAt                       *v1.Time                                        `json:"startedAt,omitempty"`
	CompletedAt                     *v1.Time                                        `json:"completedAt,omitempty"`
	CurrentBatch                    *int                                            `json:"currentBatch,omitempty"`
	CurrentBatchStartedAt           *v1.Time                                        `json:"currentBatchStartedAt,omitempty"`
	CurrentBatchRemediationProgress map[string]*v1alpha1.ClusterRemediationProgress `json:"currentBatchRemediationProgress,omitempty"`
}

// UpgradeStatusApplyConfiguration constructs an declarative configuration of the UpgradeStatus type for use with
// apply.
func UpgradeStatus() *UpgradeStatusApplyConfiguration {
	return &UpgradeStatusApplyConfiguration{}
}

// WithStartedAt sets the StartedAt field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the StartedAt field is set to the value of the last call.
func (b *UpgradeStatusApplyConfiguration) WithStartedAt(value v1.Time) *UpgradeStatusApplyConfiguration {
	b.StartedAt = &value
	return b
}

// WithCompletedAt sets the CompletedAt field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the CompletedAt field is set to the value of the last call.
func (b *UpgradeStatusApplyConfiguration) WithCompletedAt(value v1.Time) *UpgradeStatusApplyConfiguration {
	b.CompletedAt = &value
	return b
}

// WithCurrentBatch sets the CurrentBatch field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the CurrentBatch field is set to the value of the last call.
func (b *UpgradeStatusApplyConfiguration) WithCurrentBatch(value int) *UpgradeStatusApplyConfiguration {
	b.CurrentBatch = &value
	return b
}

// WithCurrentBatchStartedAt sets the CurrentBatchStartedAt field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the CurrentBatchStartedAt field is set to the value of the last call.
func (b *UpgradeStatusApplyConfiguration) WithCurrentBatchStartedAt(value v1.Time) *UpgradeStatusApplyConfiguration {
	b.CurrentBatchStartedAt = &value
	return b
}

// WithCurrentBatchRemediationProgress puts the entries into the CurrentBatchRemediationProgress field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, the entries provided by each call will be put on the CurrentBatchRemediationProgress field,
// overwriting an existing map entries in CurrentBatchRemediationProgress field with the same key.
func (b *UpgradeStatusApplyConfiguration) WithCurrentBatchRemediationProgress(entries map[string]*v1alpha1.ClusterRemediationProgress) *UpgradeStatusApplyConfiguration {
	if b.CurrentBatchRemediationProgress == nil && len(entries) > 0 {
		b.CurrentBatchRemediationProgress = make(map[string]*v1alpha1.ClusterRemediationProgress, len(entries))
	}
	for k, v := range entries {
		b.CurrentBatchRemediationProgress[k] = v
	}
	return b
}
