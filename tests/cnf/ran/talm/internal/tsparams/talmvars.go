package tsparams

import (
	"fmt"

	cguv1alpha1 "github.com/openshift-kni/cluster-group-upgrades-operator/pkg/api/clustergroupupgrades/v1alpha1"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/ranparam"
	"github.com/openshift-kni/k8sreporter"
	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	policiesv1 "open-cluster-management.io/governance-policy-propagator/api/v1"
	policiesv1beta1 "open-cluster-management.io/governance-policy-propagator/api/v1beta1"
	placementrulev1 "open-cluster-management.io/multicloud-operators-subscription/pkg/apis/apps/placementrule/v1"
)

var (
	// Labels represents the range of labels that can be used for test cases selection.
	Labels = append(ranparam.Labels, LabelSuite)

	// ReporterHubNamespacesToDump tells to the reporter which namespaces on the hub to collect pod logs from.
	ReporterHubNamespacesToDump = map[string]string{
		TestNamespace:                       "",
		ranparam.OpenshiftOperatorNamespace: "",
	}

	// ReporterSpokeNamespacesToDump tells the reporter which namespaces on the spokes to collect pod logs from.
	ReporterSpokeNamespacesToDump = map[string]string{
		TemporaryNamespace:  "",
		PreCacheSpokeNS:     "",
		TalmBackupNamespace: "",
	}

	// ReporterHubCRsToDump is the CRs the reporter should dump on the hub.
	ReporterHubCRsToDump = []k8sreporter.CRData{
		{Cr: &corev1.NamespaceList{}},
		{Cr: &corev1.PodList{}},
		{Cr: &cguv1alpha1.ClusterGroupUpgradeList{}, Namespace: ptr.To(TestNamespace)},
		{Cr: &cguv1alpha1.PreCachingConfigList{}, Namespace: ptr.To(TestNamespace)},
		{Cr: &policiesv1.PolicyList{}, Namespace: ptr.To(TestNamespace)},
		{Cr: &policiesv1.PlacementBindingList{}, Namespace: ptr.To(TestNamespace)},
		{Cr: &placementrulev1.PlacementRuleList{}, Namespace: ptr.To(TestNamespace)},
		{Cr: &policiesv1beta1.PolicySetList{}, Namespace: ptr.To(TestNamespace)},
	}

	// ReporterSpokeCRsToDump is the CRs the reporter should dump on the spokes.
	ReporterSpokeCRsToDump = []k8sreporter.CRData{
		{Cr: &corev1.NamespaceList{}},
		{Cr: &corev1.PodList{}},
		{Cr: &olmv1alpha1.CatalogSourceList{}, Namespace: ptr.To(TemporaryNamespace)},
	}

	// TalmNonExistentClusterMessage is the condition message for when a cluster is non-existent.
	TalmNonExistentClusterMessage = fmt.Sprintf(
		"Unable to select clusters: cluster %s is not a ManagedCluster", NonExistentClusterName)
	// TalmNonExistentPolicyMessage is the condition message for when a policy is non-existent.
	TalmNonExistentPolicyMessage = fmt.Sprintf("Missing managed policies: [%s]", NonExistentPolicyName)
)
