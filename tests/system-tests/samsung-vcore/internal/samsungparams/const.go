package samsungparams

const (
	// Label is used to select system tests for Samsung vCore deployment.
	Label = "samsungvcore"
	// LabelSamsungVCoreDeployment is used to select all basic Samsung deployment tests.
	LabelSamsungVCoreDeployment = "samsungvcoredeployment"
	// LabelSamsungVCoreOperators is used to select all Samsung initial-deployment deployment and configuration tests
	// excluding odf part.
	LabelSamsungVCoreOperators = "samsungvcoreoperators"
	// LabelSamsungVCoreOdf is used to select odf Samsung operator deployment and configuration tests.
	LabelSamsungVCoreOdf = "samsungvcoreodf"
	// LabelSamsungUserCases is used to select all Samsung user-cases tests.
	LabelSamsungUserCases = "samsungvcoreusercases"
	// LabelSamsungVCoreRequirements is used to select all Samsung requirements tests.
	LabelSamsungVCoreRequirements = "samsungvcorerequirements"
	// SamsungLogLevel configures logging level for Samsung related tests.
	SamsungLogLevel = 90

	// MasterNodeRole master node role.
	MasterNodeRole = "master"

	// WorkerNodeRole master node role.
	WorkerNodeRole = "worker"

	// ExpectedSamsungPPNodesCnt expected samsung-vcore-pp nodes count.
	ExpectedSamsungPPNodesCnt = 2

	// ExpectedSamsungCNFNodesCnt expected samsung-vcore-cnf nodes count.
	ExpectedSamsungCNFNodesCnt = 2

	// ExpectedOdfNodesCnt expected odf nodes count.
	ExpectedOdfNodesCnt = 3

	// SamsungPpMcpName samsung-pp workers mcp name.
	SamsungPpMcpName = "samsung-pp"

	// SamsungCnfMcpName samsung-cnf workers mcp name.
	SamsungCnfMcpName = "samsung-cnf"

	// SamsungOdfMcpName odf workers mcp name.
	SamsungOdfMcpName = "odf"

	// OpenshiftMachineAPINamespace openshift machine-api namespace.
	OpenshiftMachineAPINamespace = "openshift-machine-api"

	// MasterChronyConfigName master nodes chrony configuration file.
	MasterChronyConfigName = "98-master-etc-chrony-conf"

	// WorkerChronyConfigName worker nodes chrony configuration file.
	WorkerChronyConfigName = "98-worker-etc-chrony-conf"

	// MonitoringNetworkPolicyName monitoring networkpolicy name.
	MonitoringNetworkPolicyName = "allow-from-openshift-monitoring-ingress"

	// AllowAllNetworkPolicyName networkpolicy name.
	AllowAllNetworkPolicyName = "allow-all-ingress"

	// SctpModuleName sctp module name.
	SctpModuleName = "load-sctp-module"

	// TemplateFilesFolder path to the template files folder.
	TemplateFilesFolder = "./internal/config-files/"

	// ConfigurationFolderName path to the folder dedicated to the saving all initial-deployment configuration.
	ConfigurationFolderName = "samsung-configfiles"

	// CombinedPullSecretFile combine secret auth file name.
	CombinedPullSecretFile = "combined-secret.json"

	// PathToThePrivateKey path to the private ssh key.
	PathToThePrivateKey = ".ssh/id_rsa"

	// RegistryRepository local registry repository to mirror images to.
	RegistryRepository = "openshift"

	// SccName scc name.
	SccName = "samsung-cnf-scc"

	// SystemReservedCPU systemreserved cpu value.
	SystemReservedCPU = "500m"

	// SystemReservedMemory systemreserved memory value.
	SystemReservedMemory = "27Gi"
)
