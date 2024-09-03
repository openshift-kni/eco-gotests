package tsparams

import (
	sriovv1 "github.com/k8snetworkplumbingwg/sriov-network-operator/api/v1"
	cguv1alpha1 "github.com/openshift-kni/cluster-group-upgrades-operator/pkg/api/clustergroupupgrades/v1alpha1"
	"github.com/openshift-kni/eco-goinfra/pkg/schemes/argocd/argocdoperator"
	"github.com/openshift-kni/eco-goinfra/pkg/schemes/argocd/argocdtypes/v1alpha1"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/raninittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/ranparam"
	"github.com/openshift-kni/k8sreporter"
	imageregistryv1 "github.com/openshift/api/imageregistry/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/utils/ptr"
	policiesv1 "open-cluster-management.io/governance-policy-propagator/api/v1"
	placementrulev1 "open-cluster-management.io/multicloud-operators-subscription/pkg/apis/apps/placementrule/v1"
)

// ArgoCdGitDetails is the details for a single app in ArgoCD.
type ArgoCdGitDetails struct {
	Repo   string
	Branch string
	Path   string
}

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
		TestNamespace: "",
	}

	// ReporterHubCRsToDump is the CRs the reporter should dump on the hub.
	ReporterHubCRsToDump = []k8sreporter.CRData{
		{Cr: &corev1.NamespaceList{}},
		{Cr: &corev1.PodList{}},
		{Cr: &policiesv1.PolicyList{}},
		{Cr: &placementrulev1.PlacementRuleList{}, Namespace: ptr.To(TestNamespace)},
		{Cr: &cguv1alpha1.ClusterGroupUpgradeList{}},
		{Cr: &corev1.ConfigMapList{}, Namespace: ptr.To(TestNamespace)},
		{Cr: &corev1.SecretList{}, Namespace: ptr.To(TestNamespace)},
		{Cr: &argocdoperator.ArgoCDList{}},
		{Cr: &v1alpha1.ApplicationList{}},
	}

	// ReporterSpokeCRsToDump is the CRs the reporter should dump on the spokes.
	ReporterSpokeCRsToDump = []k8sreporter.CRData{
		{Cr: &corev1.NamespaceList{}},
		{Cr: &corev1.PodList{}},
		{Cr: &policiesv1.PolicyList{}},
		{Cr: &corev1.PersistentVolumeList{}},
		{Cr: &corev1.PersistentVolumeClaimList{}, Namespace: ptr.To(ImageRegistryNamespace)},
		{Cr: &storagev1.StorageClassList{}},
		{Cr: &corev1.ServiceAccountList{}, Namespace: ptr.To(CustomSourceTestNamespace)},
		{Cr: &sriovv1.SriovNetworkList{}, Namespace: ptr.To(RANConfig.SriovOperatorNamespace)},
		{Cr: &sriovv1.SriovNetworkList{}, Namespace: ptr.To(TestNamespace)},
		{Cr: &imageregistryv1.ConfigList{}},
	}

	// ArgoCdApps is the slice of the Argo CD app names defined in this package.
	ArgoCdApps = []string{
		ArgoCdClustersAppName,
		ArgoCdPoliciesAppName,
	}
	// ArgoCdAppDetails contains more details for each of the ArgoCdApps.
	ArgoCdAppDetails = map[string]ArgoCdGitDetails{}

	// ImageRegistryPolicies is a slice of all the policies the image registry test creates.
	ImageRegistryPolicies = []string{
		"image-registry-policy-sc",
		"image-registry-policy-pvc",
		"image-registry-policy-pv",
		"image-registry-policy-config",
	}
)
