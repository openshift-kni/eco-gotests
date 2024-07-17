package tsparams

import (
	"time"

	"github.com/golang/glog"
)

const (
	// LabelSuite is the label for all the tests in this suite.
	LabelSuite = "ran-ztp"
	// LabelArgoCdAcmCrsTestCases is the label for the ACM CRs test cases.
	LabelArgoCdAcmCrsTestCases = "ztp-argocd-acm-crs"
	// LabelArgoCdClustersAppTestCases is the label for the Argo CD clusters app test cases.
	LabelArgoCdClustersAppTestCases = "ztp-argocd-clusters"
	// LabelArgoCdHubTemplatingTestCases is the label for a particular set of test cases.
	LabelArgoCdHubTemplatingTestCases = "ztp-argocd-hub-templating"
	// LabelArgoCdNodeDeletionTestCases is the label for a particular set of test cases.
	LabelArgoCdNodeDeletionTestCases = "ztp-argocd-node-delete"
	// LabelArgoCdPoliciesAppTestCases is the label for a particular set of test cases.
	LabelArgoCdPoliciesAppTestCases = "ztp-argocd-policies"
	// LabelMachineConfigTestCases is the label for a particular set of test cases.
	LabelMachineConfigTestCases = "ztp-machine-config"

	// MultiClusterHubOperator is the name of the multi cluster hub operator.
	MultiClusterHubOperator = "multiclusterhub-operator"
	// AcmPolicyGeneratorName is the name of the ACM policy generator container.
	AcmPolicyGeneratorName = "acm-policy-generator"
	// TalmHubPodName is the name of the TALM pod on the hub cluster.
	TalmHubPodName = "cluster-group-upgrades-controller-manager"
	// ImageRegistryName is the name of the image registry config.
	ImageRegistryName = "cluster"
	// ImageRegistryNamespace is the namespace for the image registry and where its PVC is.
	ImageRegistryNamespace = "openshift-image-registry"

	// ArgoCdPoliciesAppName is the name of the policies app in Argo CD.
	ArgoCdPoliciesAppName = "policies"
	// ArgoCdClustersAppName is the name of the clusters app in Argo CD.
	ArgoCdClustersAppName = "clusters"

	// ArgoCdChangeInterval is the interval to use for polling for changes to Argo CD.
	ArgoCdChangeInterval = 10 * time.Second
	// ArgoCdChangeTimeout is the time to use for polling for changes to Argo CD.
	ArgoCdChangeTimeout = 10 * time.Minute

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
	// ZtpKustomizationPath is the path to the kustomization file in the ztp test.
	ZtpKustomizationPath = "/kustomization.yaml"

	// TestNamespace is the namespace used for ZTP tests.
	TestNamespace = "ztp-test"
	// AcmCrsPolicyName is the name of the policy for ACM CRs.
	AcmCrsPolicyName = "acm-crs-policy"
	// HubTemplatingPolicyName is the name used for the hub templating policy.
	HubTemplatingPolicyName = "hub-templating-policy-sriov-config"
	// HubTemplatingCguName is the name used for the hub templating CGU.
	HubTemplatingCguName = "hub-templating"
	// HubTemplatingCguNamespace is the namespace used by the hub templating CGU. It should be different than the
	// policy namespace.
	HubTemplatingCguNamespace = "default"
	// NodeDeletionCrAnnotation is the annotation applied in the node deletion tests.
	NodeDeletionCrAnnotation = "bmac.agent-install.openshift.io/remove-agent-and-node-on-delete"
	// ZtpGeneratedAnnotation is the annotation applied to ztp generated resources.
	ZtpGeneratedAnnotation = "ran.openshift.io/ztp-gitops-generated"
	// CustomIntervalDefaultPolicyName is the name of the default policy created in the custom interval test.
	CustomIntervalDefaultPolicyName = "custom-interval-policy-default"
	// CustomIntervalOverridePolicyName is the name of the override policy created in the custom interval test.
	CustomIntervalOverridePolicyName = "custom-interval-policy-override"
	// CustomSourceCrPolicyName is the name of the policy for the custom source CR.
	CustomSourceCrPolicyName = "custom-source-cr-policy-config"
	// CustomSourceCrName is the name of the custom source CR itself.
	CustomSourceCrName = "custom-source-cr"
	// CustomSourceTestNamespace is the test namespace for the custom source test.
	CustomSourceTestNamespace = "default"
	// CustomSourceStorageClass is the storage class used in the custom source test.
	CustomSourceStorageClass = "example-storage-class"
	// ImageRegistrySC is the storage class created by the policies app image registry tests.
	ImageRegistrySC = "image-registry-sc"
	// ImageRegistryPV is the persistent volume created by the policies app image registry tests.
	ImageRegistryPV = "image-registry-pv-filesystem"
	// ImageRegistryPVC is the persistent volume claim created by the policies app image registry tests.
	ImageRegistryPVC = "image-registry-pvc"
	// ImageRegistryPath is the path to where the image registry PV will be.
	ImageRegistryPath = "/var/imageregistry"

	// LogLevel is the verbosity of glog statements in this test suite.
	LogLevel glog.Level = 90
)
