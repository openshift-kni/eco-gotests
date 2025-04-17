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
	// ImageRegistryName is the name of the image registry config.
	ImageRegistryName = "cluster"
	// ImageRegistryNamespace is the namespace for the image registry and where its PVC is.
	ImageRegistryNamespace = "openshift-image-registry"
	// ArgoCdPoliciesAppName is the name of the policies app in Argo CD.
	ArgoCdPoliciesAppName = "policies"
	// ArgoCdClustersAppName is the name of the clusters app in Argo CD.
	ArgoCdClustersAppName = "clusters"

	// ArgoCdChangeTimeout is the time to use for polling for changes to Argo CD.
	ArgoCdChangeTimeout = 10 * time.Minute

	// TestNamespace is the namespace used for deployment types tests.
	TestNamespace = "deployment-test"
	// LogLevel is the verbosity of glog statements in this test suite.
	LogLevel glog.Level = 90
)
