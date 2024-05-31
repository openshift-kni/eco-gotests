/*
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2023 Red Hat, Inc.
 */

package v1

func (nodeGroup NodeGroup) NormalizeConfig() NodeGroupConfig {
	conf := DefaultNodeGroupConfig()
	if nodeGroup.Config != nil {
		conf = conf.Merge(*nodeGroup.Config)
	}
	return conf
}
func (current NodeGroupConfig) Merge(updated NodeGroupConfig) NodeGroupConfig {
	conf := NodeGroupConfig{}
	current.DeepCopyInto(&conf)

	if updated.PodsFingerprinting != nil {
		conf.PodsFingerprinting = updated.PodsFingerprinting
	}
	if updated.InfoRefreshPeriod != nil {
		conf.InfoRefreshPeriod = updated.InfoRefreshPeriod
	}
	if updated.InfoRefreshMode != nil {
		conf.InfoRefreshMode = updated.InfoRefreshMode
	}
	return conf
}
