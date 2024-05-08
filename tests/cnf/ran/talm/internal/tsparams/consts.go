package tsparams

import (
	"time"

	"github.com/golang/glog"
)

const (
	// LabelSuite is the label for all the tests in this suite.
	LabelSuite = "talm"
	// LabelBackupTestCases is the label for a particular test file.
	LabelBackupTestCases = "backup"
	// LabelBatchingTestCases is the label for a particular test file.
	LabelBatchingTestCases = "batching"
	// LabelBlockingCRTestCases is the label for a particular test file.
	LabelBlockingCRTestCases = "blockingcr"
	// LabelCanaryTestCases is the label for a particular test file.
	LabelCanaryTestCases = "canary"
	// LabelMissingSpokeTestCases is the label for a set of batching test cases.
	LabelMissingSpokeTestCases = "missingspoke"
	// LabelMissingPolicyTestCases is the label for a set of batching test cases.
	LabelMissingPolicyTestCases = "missingpolicy"
	// LabelCatalogSourceTestCases is the label for a set of batching test cases.
	LabelCatalogSourceTestCases = "catalogsource"
	// LabelTempNamespaceTestCases is the label for a set of batching test cases.
	LabelTempNamespaceTestCases = "tempnamespace"

	// TestNamespace is the testing namespace created on the hub.
	TestNamespace = "talm-test"
	// TemporaryNamespace is a temporary namespace for testing created on the spokes.
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
	// PreCachingConfigName is the name for the test talm pre caching config.
	PreCachingConfigName = "talm-precachingconfig"
	// NonExistentPolicyName is the name for non-existent policies.
	NonExistentPolicyName = "non-existent-policy"
	// NonExistentClusterName is the name for non-existent clusters.
	NonExistentClusterName = "non-existent-cluster"

	// TalmPodLabelSelector is the label selector to find talm pods.
	TalmPodLabelSelector = "pod-template-hash"
	// TalmContainerName is the name of the container in the talm pod.
	TalmContainerName = "manager"

	// OperatorHubTalmNamespace talm namespace.
	OperatorHubTalmNamespace = "topology-aware-lifecycle-manager"
	// OpenshiftOperatorNamespace is the namespace where operators are.
	OpenshiftOperatorNamespace = "openshift-operators"
	// OpenshiftLoggingNamespace is the namespace where logging pods are.
	OpenshiftLoggingNamespace = "openshift-logging"

	// BackupPath is the path the temporary filesystem is mounted to.
	BackupPath = "/var/recovery"
	// RANTestPath is where the temporary filesystem file is.
	RANTestPath = "/var/ran-test-talm-recovery"
	// FSSize is the size of the temporary filesystem.
	FSSize = "100M"

	// ClustersSelectedType is the type for a CGU condition.
	ClustersSelectedType = "ClustersSelected"
	// ValidatedType is the type for a CGU condition.
	ValidatedType = "Validated"
	// ReadyType is the type for a CGU condition.
	ReadyType = "Ready"
	// SucceededType is the type for a CGU condition.
	SucceededType = "Succeeded"
	// ProgressingType is the type for a CGU condition.
	ProgressingType = "Progressing"
	// CompletedReason is the reason for a CGU condition.
	CompletedReason = "Completed"
	// TimedOutReason is the reason for a CGU condition.
	TimedOutReason = "TimedOut"
	// UpgradeCompletedReason is the reason for a CGU condition.
	UpgradeCompletedReason = "UpgradeCompleted"
	// TalmTimeoutMessage is the message for a CGU condition.
	TalmTimeoutMessage = "Policy remediation took too long"
	// TalmCanaryTimeoutMessage is the message for a CGU condition.
	TalmCanaryTimeoutMessage = "Policy remediation took too long on canary clusters"
	// TalmBlockedMessage is the message for a CGU condition.
	TalmBlockedMessage = "Blocking CRs that are not completed: [%s]"
	// TalmMissingCRMessage is the message for a CGU condition.
	TalmMissingCRMessage = "Missing blocking CRs: [%s]"

	// PreCacheContainerName is the name of the pre cache container.
	PreCacheContainerName = "pre-cache-container"
	// PreCachePodLabel is the label for the pre cache pod.
	PreCachePodLabel = "job-name=pre-cache"
	// PreCacheSpokeNS is the namespace from the pre cache test.
	PreCacheSpokeNS = "openshift-talo-pre-cache"
	// PreCacheOverrideName is the name of the config map for excluding images from precaching.
	PreCacheOverrideName = "cluster-group-upgrade-overrides"

	// MasterNodeSelector when used in a label selector finds all master nodes.
	MasterNodeSelector = "node-role.kubernetes.io/master="

	// TalmDefaultReconcileTime is the default time for talm to reconcile.
	TalmDefaultReconcileTime = 5 * time.Minute
	// TalmSystemStablizationTime is the default time to wait for talm to settle.
	TalmSystemStablizationTime = 15 * time.Second

	// LogLevel is the verbosity of glog statements in this test suite.
	LogLevel glog.Level = 90
)
