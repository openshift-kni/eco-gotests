package tsparams

import (
	"time"

	"github.com/golang/glog"
)

const (
	// LabelSuite represents upgrade label that can be used for test cases selection.
	LabelSuite = "ibbf"

	// LabelIBBFe2e represents e2e label that can be used for test cases selection.
	LabelIBBFe2e = "ibbfEndtoEnd"

	// ArgoCdClustersAppName is the name of the clusters app in Argo CD.
	ArgoCdClustersAppName = "clusters"

	// ArgoCdPoliciesAppName is the name of the clusters app in Argo CD.
	ArgoCdPoliciesAppName = "policies"

	// IBBFTestPath is the name of the path to IBBF test siteconfig in git.
	IBBFTestPath = "ibbf_test"

	// RHACMNamespace is the name of the ACM namespace.
	RHACMNamespace = "rhacm"

	// TestCMName is the name of the  configmap used for preservation testing.
	TestCMName = "testConfigMap"

	// ArgoCdChangeInterval is the interval to use for polling for changes to Argo CD.
	ArgoCdChangeInterval = 10 * time.Second

	// ArgoCdChangeTimeout is the time to use for polling for changes to Argo CD.
	ArgoCdChangeTimeout = 10 * time.Minute

	// ZtpKustomizationPath is the path to the kustomization file in the ztp test.
	ZtpKustomizationPath = "/kustomization.yaml"

	// LogLevel is the verbosity of glog statements in this test suite.
	LogLevel glog.Level = 90
)
