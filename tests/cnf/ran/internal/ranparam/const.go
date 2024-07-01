package ranparam

import "github.com/golang/glog"

const (
	// Label represents the label for the ran test cases.
	Label = "ran"
	// AcmOperatorName operator name of ACM.
	AcmOperatorName = "advanced-cluster-management"
	// AcmOperatorNamespace ACM's namespace.
	AcmOperatorNamespace = "rhacm"
	// TalmOperatorHubNamespace TALM namespace.
	TalmOperatorHubNamespace = "topology-aware-lifecycle-manager"
	// TalmContainerName is the name of the container in the talm pod.
	TalmContainerName = "manager"
	// OpenshiftOperatorNamespace is the namespace where operators are.
	OpenshiftOperatorNamespace = "openshift-operators"
	// OpenshiftGitops name.
	OpenshiftGitops = "openshift-gitops"
	// OpenshiftGitopsRepoServer ocp git repo server.
	OpenshiftGitopsRepoServer = "openshift-gitops-repo-server"
	// PtpContainerName is the name of the container in the PTP daemon pod.
	PtpContainerName = "linuxptp-daemon-container"
	// PtpDaemonsetLabelSelector is the label selector to find the PTP daemon pod.
	PtpDaemonsetLabelSelector = "app=linuxptp-daemon"
	// LogLevel is the verbosity for ran/internal packages.
	LogLevel glog.Level = 80
)
