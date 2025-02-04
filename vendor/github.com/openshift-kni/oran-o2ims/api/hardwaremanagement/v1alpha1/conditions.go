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

type ConditionType string

// The following constants define the different types of conditions that will be set
const (
	Provisioned ConditionType = "Provisioned"
	Configured  ConditionType = "Configured"
	Validation  ConditionType = "Validation"
	Unknown     ConditionType = "Unknown" // Indicates the condition has not been evaluated
)

// ConditionReason describes the reasons for a condition's status.
type ConditionReason string

const (
	InProgress     ConditionReason = "InProgress"
	Completed      ConditionReason = "Completed"
	Unprovisioned  ConditionReason = "Unprovisioned"
	Failed         ConditionReason = "Failed"
	NotInitialized ConditionReason = "NotInitialized"
	TimedOut       ConditionReason = "TimedOut"
	ConfigUpdate   ConditionReason = "ConfigurationUpdateRequested"
	ConfigApplied  ConditionReason = "ConfigurationApplied"
)

// ConditionMessage provides detailed messages associated with condition status updates.
type ConditionMessage string

const (
	AwaitConfig   ConditionMessage = "Spec updated; awaiting configuration application by the hardware plugin"
	ConfigSuccess ConditionMessage = "Configuration has been applied successfully"
)
