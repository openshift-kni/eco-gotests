package ranparam

import (
	"github.com/golang/glog"
)

const (
	// Label represents the label for the ran test cases.
	Label = "ran-deployment"

	// AcmOperatorNamespace ACM's namespace.
	AcmOperatorNamespace = "rhacm"
	// MceOperatorNamespace is the namespace for the MCE operator.
	MceOperatorNamespace = "multicluster-engine"
	// TalmOperatorHubNamespace TALM namespace.
	TalmOperatorHubNamespace = "topology-aware-lifecycle-manager"
	// TalmContainerName is the name of the container in the talm pod.
	TalmContainerName = "manager"
	// OpenshiftOperatorNamespace is the namespace where operators are.
	OpenshiftOperatorNamespace = "openshift-operators"
	// OpenshiftGitOpsNamespace is the namespace for the GitOps operator.
	OpenshiftGitOpsNamespace = "openshift-gitops"
	// OpenshiftGitopsRepoServer ocp git repo server.
	OpenshiftGitopsRepoServer = "openshift-gitops-repo-server"

	// LogLevel is the verbosity for ran/internal packages.
	LogLevel glog.Level = 80
)

// HubOperatorName represets the possible operator names that may have associated versions on the hub cluster.
type HubOperatorName string

const (
	// ACM is the name of the advanced cluster management operator.
	ACM HubOperatorName = "advanced-cluster-management"
	// TALM is the name of the topology aware lifecycle manager operator.
	TALM HubOperatorName = "topology-aware-lifecycle-manager"
	// GitOps is the name of the GitOps operator.
	GitOps HubOperatorName = "openshift-gitops-operator"
	// MCE is the name of the multicluster engine operator.
	MCE HubOperatorName = "multicluster-engine"
)
