package ranparam

import "github.com/golang/glog"

const (
	// Label represents the label for the ran test cases.
	Label string = "ran"
	// AcmOperatorName operator name of ACM.
	AcmOperatorName string = "advanced-cluster-management"
	// AcmOperatorNamespace ACM's namespace.
	AcmOperatorNamespace string = "rhacm"
	// OpenshiftGitops name.
	OpenshiftGitops string = "openshift-gitops"
	// OpenshiftGitopsRepoServer ocp git repo server.
	OpenshiftGitopsRepoServer string = "openshift-gitops-repo-server"
	// LogLevel is the verbosity for ran/internal packages.
	LogLevel glog.Level = 80
)
