package tsparams

import (
	"time"

	"github.com/golang/glog"
)

const (
	// LabelSuite is the label for all the tests in this suite.
	LabelSuite = "ran-ztp"

	// TestNamespace is the namespace used for ZTP tests.
	TestNamespace = "ztp-test"

	// ArgoCdPoliciesAppName is the name of the policies app in Argo CD.
	ArgoCdPoliciesAppName = "policies"
	// ArgoCdClustersAppName is the name of the clusters app in Argo CD.
	ArgoCdClustersAppName = "clusters"

	// ArgoCdChangeInterval is the interval to use for polling for changes to Argo CD.
	ArgoCdChangeInterval = 10 * time.Second
	// ArgoCdChangeTimeout is the time to use for polling for changes to Argo CD.
	ArgoCdChangeTimeout = 10 * time.Minute

	// ZtpKustomizationPath is the path to the kustomization file in the ztp test.
	ZtpKustomizationPath = "/kustomization.yaml"

	// LogLevel is the verbosity of glog statements in this test suite.
	LogLevel glog.Level = 90
)
