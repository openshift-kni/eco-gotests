package tsparams

import (
	"time"

	"github.com/golang/glog"
)

const (
	// LabelSuite is the label for all the tests in this suite.
	LabelSuite = "deploymenttypes"
	// LabelDeploymentTypeTestCases is the label for deployment type checking.
	LabelDeploymentTypeTestCases = "deployment-types"
	/*
		// MultiClusterHubOperator is the name of the multi cluster hub operator.
		MultiClusterHubOperator = "multiclusterhub-operator"
		// AcmPolicyGeneratorName is the name of the ACM policy generator container.
		AcmPolicyGeneratorName = "acm-policy-generator"
		// TalmHubPodName is the name of the TALM pod on the hub cluster.
		TalmHubPodName = "cluster-group-upgrades-controller-manager"
	*/
	// ImageRegistryName is the name of the image registry config.
	ImageRegistryName = "cluster"
	// ImageRegistryNamespace is the namespace for the image registry and where its PVC is.
	ImageRegistryNamespace = "openshift-image-registry"
	/*
		// NetworkDiagnosticsNamespace is the namespace for network diagnostics.
		NetworkDiagnosticsNamespace = "openshift-network-diagnostics"
		// ConsoleNamespace is the namespace for the openshift console.
		ConsoleNamespace = "openshift-console"
	*/
	// ArgoCdPoliciesAppName is the name of the policies app in Argo CD.
	ArgoCdPoliciesAppName = "policies"
	// ArgoCdClustersAppName is the name of the clusters app in Argo CD.
	ArgoCdClustersAppName = "clusters"

	// ArgoCdChangeInterval is the interval to use for polling for changes to Argo CD.
	// ArgoCdChangeInterval = 10 * time.Second
	// ArgoCdChangeTimeout is the time to use for polling for changes to Argo CD.
	ArgoCdChangeTimeout = 10 * time.Minute

	/*
		// ZtpTestPathAcmCrs is the git path for the ACM CRs test.
		ZtpTestPathAcmCrs = "ztp-test/acm-crs"
		// ZtpTestPathClustersApp is the git path for the clusters app test.
		ZtpTestPathClustersApp = "ztp-test/klusterlet-addon"
		// ZtpTestPathRemoveNmState is the git path for the remove nm state test.
		ZtpTestPathRemoveNmState = "ztp-test/remove-nmstate"
		// ZtpTestPathTemplatingAutoIndent is the git path for the templating auto indent test.
		ZtpTestPathTemplatingAutoIndent = "ztp-test/hub-templating-autoindent"
		// ZtpTestPathTemplatingValid is the git path for the templating valid test.
		ZtpTestPathTemplatingValid = "ztp-test/hub-templating-valid"
		// ZtpTestPathTemplatingValid416 is the git path for the templating valid test starting from TALM 4.16.
		ZtpTestPathTemplatingValid416 = "ztp-test/hub-templating-valid-4.16"
		// ZtpTestPathNodeDeleteAddAnnotation is the git path for the node deletion add annotation test.
		ZtpTestPathNodeDeleteAddAnnotation = "ztp-test/node-delete/add-annotation"
		// ZtpTestPathNodeDeleteAddSuppression is the git path for the node deletion add suppression test.
		ZtpTestPathNodeDeleteAddSuppression = "ztp-test/node-delete/add-suppression"
		// ZtpTestPathCustomInterval is the git path for the policies app custom interval test.
		ZtpTestPathCustomInterval = "ztp-test/custom-interval"
		// ZtpTestPathInvalidInterval is the git path for the policies app invalid interval test.
		ZtpTestPathInvalidInterval = "ztp-test/invalid-interval"
		// ZtpTestPathImageRegistry is the git path for the policies app image registry test.
		ZtpTestPathImageRegistry = "ztp-test/image-registry"
		// ZtpTestPathCustomSourceNewCr is the git path for the policies app custome source new cr test.
		ZtpTestPathCustomSourceNewCr = "ztp-test/custom-source-crs/new-cr"
		// ZtpTestPathCustomSourceReplaceExisting is the path for the policies app custom source replace existing test.
		ZtpTestPathCustomSourceReplaceExisting = "ztp-test/custom-source-crs/replace-existing"
		// ZtpTestPathCustomSourceNoCrFile is the git path for the policies app custome source no cr file test.
		ZtpTestPathCustomSourceNoCrFile = "ztp-test/custom-source-crs/no-cr-file"
		// ZtpTestPathCustomSourceSearchPath is the git path for the policies app custome source search path test.
		ZtpTestPathCustomSourceSearchPath = "ztp-test/custom-source-crs/search-path"
		// ZtpTestPathDetachAIMNO is the git path for the siteconfig operator detach AI MNO cluster instance test.
		ZtpTestPathDetachAIMNO = "ztp-test/siteconfig-operator/detach-ai-mno"
		// ZtpTestPathDetachAISNO is the git path for the siteconfig operator detach AI SNO cluster instance test.
		ZtpTestPathDetachAISNO = "ztp-test/siteconfig-operator/detach-ai-sno"
		// ZtpTestPathNoClusterTemplateCm is the git path for the siteconfig operator non-existent cluster template cm test.
		ZtpTestPathNoClusterTemplateCm = "ztp-test/siteconfig-operator/non-existent-cluster-template-cm"
		// ZtpTestPathNoExtraManifestsCm is the git path for the siteconfig operator non-existent extra manifests cm test.
		ZtpTestPathNoExtraManifestsCm = "ztp-test/siteconfig-operator/non-existent-extra-manifests-cm"
		// ZtpTestPathInvalidTemplateRef is the git path for the siteconfig operator invalid template reference test.
		ZtpTestPathInvalidTemplateRef = "ztp-test/siteconfig-operator/invalid-template-ref"
		// ZtpTestPathValidTemplateRef is the git path for the siteconfig operator valid template reference test.
		ZtpTestPathValidTemplateRef = "ztp-test/siteconfig-operator/valid-template-ref"
		// ZtpTestPathUniqueClusterName is the git path for the siteconfig operator unique cluster name test.
		ZtpTestPathUniqueClusterName = "ztp-test/siteconfig-operator/unique-cluster-name"
		// ZtpTestPathDuplicateClusterName is the git path for the siteconfig operator duplicate cluster name test.
		ZtpTestPathDuplicateClusterName = "ztp-test/siteconfig-operator/duplicate-cluster-name"
		// ZtpTestPathNewClusterLabel is the git path for the siteconfig operator new cluster label test.
		ZtpTestPathNewClusterLabel = "ztp-test/siteconfig-operator/new-cluster-label"
		// ZtpKustomizationPath is the path to the kustomization file in the ztp test.
		ZtpKustomizationPath = "/kustomization.yaml"
	*/
	// TestNamespace is the namespace used for deployment types tests.
	TestNamespace = "deployment-test"
	/*
		// ClusterInstanceValidatedType is one of the type of ClusterInstance condition types.
		ClusterInstanceValidatedType = "ClusterInstanceValidated"
		// ProvisionedType is one of the type of ClusterInstance condition types.
		ProvisionedType = "Provisioned"
		// RenderedTemplatesValidatedType is one of the type of ClusterInstance condition types.
		RenderedTemplatesValidatedType = "RenderedTemplatesValidated"
		// ClusterInstanceFailReason is the reason for a failure of all ClusterInstance condition types.
		ClusterInstanceFailReason = "Failed"
		// ClusterInstanceInProgressReason is the reason for in-progress cluster provisioning.
		ClusterInstanceInProgressReason = "InProgress"
		// NonExistentExtraManifestConfigMapFailMessage is the message for ClusterInstanceValidated condition type.
		NonExistentExtraManifestConfigMapFailMessage = "Validation failed: failed to retrieve ExtraManifest"
		// NonExistentClusterTemplateConfigMapFailMessage is the message for ClusterInstanceValidated condition type.
		NonExistentClusterTemplateConfigMapFailMessage = "Validation failed: failed to validate cluster-level TemplateRef"
		// InvalidTemplateRefFailMessage is the message for invalid template reference condition.
		InvalidTemplateRefFailMessage = "Validation failed: failed to validate node-level TemplateRef"
		// ValidTemplateRefSuccessMessage is the message for valid template reference condition.
		ValidTemplateRefSuccessMessage = "Provisioning cluster"
		// RenderedManifestsFailMessage is the message for rendered manifests dry-run validation failure.
		RenderedManifestsFailMessage = "Rendered manifests failed dry-run validation"
		// SiteconfigOperatorDefaultReconcileTime is the default time for siteconfig controller to reconcile.
		SiteconfigOperatorDefaultReconcileTime = 5 * time.Minute
	*/
	// LogLevel is the verbosity of glog statements in this test suite.
	LogLevel glog.Level = 90
)
