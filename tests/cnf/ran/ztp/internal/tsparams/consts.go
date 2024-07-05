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

	// TestNamespace is the namespace used for ZTP tests.
	TestNamespace = "ztp-test"

	// MultiClusterHubOperator is the name of the multi cluster hub operator.
	MultiClusterHubOperator = "multiclusterhub-operator"
	// AcmPolicyGeneratorName is the name of the ACM policy generator container.
	AcmPolicyGeneratorName = "acm-policy-generator"

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
	// ZtpKustomizationPath is the path to the kustomization file in the ztp test.
	ZtpKustomizationPath = "/kustomization.yaml"

	// AcmCrsPolicyName is the name of the policy for ACM CRs.
	AcmCrsPolicyName = "acm-crs-policy"

	// LogLevel is the verbosity of glog statements in this test suite.
	LogLevel glog.Level = 90
)
