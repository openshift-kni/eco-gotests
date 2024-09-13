/*
Copyright 2024.

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

// ExtraAnnotationSearch Looks up a specific manifest Annotation for this cluster
func (c *ClusterInstanceSpec) ExtraAnnotationSearch(kind string) (map[string]string, bool) {
	annotations, ok := c.ExtraAnnotations[kind]
	return annotations, ok
}

// ExtraAnnotationSearch Looks up a specific manifest annotation for this node, with fallback to cluster
func (node *NodeSpec) ExtraAnnotationSearch(kind string, cluster *ClusterInstanceSpec) (map[string]string, bool) {
	annotations, ok := node.ExtraAnnotations[kind]
	if ok {
		return annotations, ok
	}
	return cluster.ExtraAnnotationSearch(kind)
}

// ExtraLabelSearch Looks up a specific manifest label for this cluster
func (c *ClusterInstanceSpec) ExtraLabelSearch(kind string) (map[string]string, bool) {
	labels, ok := c.ExtraLabels[kind]
	return labels, ok
}

// ExtraLabelSearch Looks up a specific manifest label for this node, with fallback to cluster
func (node *NodeSpec) ExtraLabelSearch(kind string, cluster *ClusterInstanceSpec) (map[string]string, bool) {
	labels, ok := node.ExtraLabels[kind]
	if ok {
		return labels, ok
	}
	return cluster.ExtraLabelSearch(kind)
}
