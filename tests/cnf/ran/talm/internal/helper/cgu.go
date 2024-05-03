package helper

import (
	"github.com/openshift-kni/eco-goinfra/pkg/cgu"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/raninittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/talm/internal/tsparams"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SetupCguWithNamespace creates the policy with a namespace and its components for a cguBuilder then creates the
// cguBuilder.
func SetupCguWithNamespace(cguBuilder *cgu.CguBuilder, suffix string) (*cgu.CguBuilder, error) {
	// The client doesn't matter since we only want the definition. Kind and APIVersion are necessary for TALM.
	tempNs := namespace.NewBuilder(raninittools.APIClient, tsparams.TemporaryNamespace+suffix)
	tempNs.Definition.Kind = "Namespace"
	tempNs.Definition.APIVersion = corev1.SchemeGroupVersion.Version

	_, err := CreatePolicy(raninittools.APIClient, tempNs.Definition, suffix)
	if err != nil {
		return nil, err
	}

	err = CreatePolicyComponents(
		raninittools.APIClient, suffix, cguBuilder.Definition.Spec.Clusters, metav1.LabelSelector{})
	if err != nil {
		return nil, err
	}

	return cguBuilder.Create()
}
