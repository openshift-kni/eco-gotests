package ranparam

const (
	// HubKubeEnvKey is the hub's kubeconfig env var.
	HubKubeEnvKey string = "KUBECONFIG"
	//  SpokeKubeEnvKey is the spoke's kubeconfig env var.
	SpokeKubeEnvKey string = "KUBECONFIG_SPOKE1"
	// AcmOperatorName operator name of ACM.
	AcmOperatorName string = "advanced-cluster-management"
	// AcmOperatorNamespace ACM's namespace.
	AcmOperatorNamespace string = "rhacm"
	// OpenshiftGitops name.
	OpenshiftGitops string = "openshift-gitops"
	// OpenshiftGitopsRepoServer ocp git repo server.
	OpenshiftGitopsRepoServer string = "openshift-gitops-repo-server"
)
