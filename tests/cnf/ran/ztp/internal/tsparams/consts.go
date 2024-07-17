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

	// MultiClusterHubOperator is the name of the multi cluster hub operator.
	MultiClusterHubOperator = "multiclusterhub-operator"
	// AcmPolicyGeneratorName is the name of the ACM policy generator container.
	AcmPolicyGeneratorName = "acm-policy-generator"
	// TalmHubPodName is the name of the TALM pod on the hub cluster.
	TalmHubPodName = "cluster-group-upgrades-controller-manager"

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

	// LogLevel is the verbosity of glog statements in this test suite.
	LogLevel glog.Level = 90
)
