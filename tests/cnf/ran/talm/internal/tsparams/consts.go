package tsparams

import (
	"github.com/golang/glog"
)

const (
	// LabelSuite is the label for all the tests in this suite.
	LabelSuite = "talm"
	// LabelBackupTestCases is the label for a particular test case.
	LabelBackupTestCases = "backup"

	// TestNamespace is the tests namespace.
	TestNamespace = "talm-test"
	// TemporaryNamespace is a temporary namespace for testing.
	TemporaryNamespace = "talm-namespace-temp"
	// CguName is the name for the test talm cgu.
	CguName = "talm-cgu"
	// PolicyName is the name for the test talm policy.
	PolicyName = "talm-policy"
	// PolicySetName is the name for the test talm policy set.
	PolicySetName = "talm-policyset"
	// PlacementRuleName is the name for the test talm placement rule.
	PlacementRuleName = "talm-placementrule"
	// PlacementBindingName is the name for the test talm placement binding.
	PlacementBindingName = "talm-placementbinding"
	// CatalogSourceName is the name for the test talm catalog source.
	CatalogSourceName = "talm-catsrc"

	// TalmPodLabelSelector is the label selector to find talm pods.
	TalmPodLabelSelector = "pod-template-hash"
	// TalmContainerName is the name of the container in the talm pod.
	TalmContainerName = "manager"
	// OperatorHubTalmNamespace talm namespace.
	OperatorHubTalmNamespace = "topology-aware-lifecycle-manager"
	// OpenshiftOperatorNamespace is the namespace where operators are.
	OpenshiftOperatorNamespace = "openshift-operators"

	// BackupPath is the path the temporary filesystem is mounted to.
	BackupPath = "/var/recovery"
	// RANTestPath is where the temporary filesystem file is.
	RANTestPath = "/var/ran-test-talm-recovery"
	// FSSize is the size of the temporary filesystem.
	FSSize = "100M"

	// LogLevel is the verbosity of glog statements in this test suite.
	LogLevel glog.Level = 90
)
