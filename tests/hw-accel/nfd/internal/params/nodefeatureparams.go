package params

import "k8s.io/apimachinery/pkg/runtime/schema"

// PossibleGVRs holds the GroupVersionResource configurations that are supported.
var PossibleGVRs = []schema.GroupVersionResource{
	{
		Group:    "nfd.k8s-sigs.io",
		Version:  "v1alpha1",
		Resource: "nodefeatures",
	},
	{
		Group:    "nfd.openshift.io",
		Version:  "v1alpha1",
		Resource: "nodefeatures",
	},
}
