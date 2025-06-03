package tsparams

import (
	"time"

	"github.com/golang/glog"
)

const (
	// LabelSuite is the label for all the tests in this suite.
	LabelSuite string = "deploymenttypes"
	// LabelDeploymentTypeTestCases is the label for deployment type checking.
	LabelDeploymentTypeTestCases string = "deployment-types"
	// ImageRegistryName is the name of the image registry config.
	ImageRegistryName string = "cluster"
	// ImageRegistryNamespace is the namespace for the image registry and where its PVC is.
	ImageRegistryNamespace string = "openshift-image-registry"
	// ArgoCdPoliciesAppName is the name of the policies app in Argo CD.
	ArgoCdPoliciesAppName string = "policies"
	// ArgoCdClustersAppName is the name of the clusters app in Argo CD.
	ArgoCdClustersAppName string = "clusters"

	// ArgoCdChangeTimeout is the time to use for polling for changes to Argo CD.
	ArgoCdChangeTimeout = 10 * time.Minute

	// TestNamespace is the namespace used for deployment types tests.
	TestNamespace = "deployment-test"
	// LogLevel is the verbosity of glog statements in this test suite.
	LogLevel glog.Level = 90
)
