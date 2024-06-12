package vcoreparams

const (
	// Label is used to select system tests for vCore deployment.
	Label = "vcore"
	// LabelVCoreDeployment is used to select all basic vCore deployment tests.
	LabelVCoreDeployment = "vcoredeployment"
	// LabelVCoreOperators is used to select all vCore initial-deployment deployment and configuration tests
	// excluding odf part.
	LabelVCoreOperators = "vcoreoperators"
	// LabelVCoreOdf is used to select odf configuration tests.
	LabelVCoreOdf = "vcoreodf"
	// LabelUserCases is used to select all vCore user-cases tests.
	LabelUserCases = "vcoreusercases"
	// LabelVCoreRequirements is used to select all vCore requirements tests.
	LabelVCoreRequirements = "vcorerequirements"
	// VCoreLogLevel configures logging level for vCore related tests.
	VCoreLogLevel = 90
	// LabelVCoreDebug is used to select vCore tests under debug.
	LabelVCoreDebug = "vcoredebug"

	// MasterNodeRole master node role.
	MasterNodeRole = "master"

	// WorkerNodeRole master node role.
	WorkerNodeRole = "worker"

	// HugePagesSize hugepages size.
	HugePagesSize = "2M"

	// ExpectedHugePagesResource hugepages resource size.
	ExpectedHugePagesResource = "64Gi"

	// TopologyConfig performanceprofile topology config.
	TopologyConfig = "single-numa-node"

	// OpenshiftMachineAPINamespace openshift machine-api namespace.
	OpenshiftMachineAPINamespace = "openshift-machine-api"

	// MonitoringNetworkPolicyName monitoring networkpolicy name.
	MonitoringNetworkPolicyName = "allow-from-openshift-monitoring-ingress"

	// AllowAllNetworkPolicyName networkpolicy name.
	AllowAllNetworkPolicyName = "allow-all-ingress"

	// SctpModuleName sctp module name.
	SctpModuleName = "load-sctp-module"

	// TemplateFilesFolder path to the template files folder.
	TemplateFilesFolder = "./internal/config-files/"

	// ConfigurationFolderName path to the folder dedicated to the saving all initial-deployment configuration.
	ConfigurationFolderName = "vcore-configfiles"

	// RegistryRepository local registry repository to mirror images to.
	RegistryRepository = "openshift"

	// OperatorsNamespace is a operator's deployment namespace.
	OperatorsNamespace = "openshift-marketplace"

	// SccName scc name.
	SccName = "vcore-control-plane-worker-scc"

	// SystemReservedCPU systemreserved cpu value.
	SystemReservedCPU = "500m"

	// SystemReservedMemory systemreserved memory value.
	SystemReservedMemory = "27Gi"

	// NMStateInstanceName is a name of the NMState instance.
	NMStateInstanceName = "nmstate"

	// NMStateOperatorName is a name of the NMState operator.
	NMStateOperatorName = "kubernetes-nmstate-operator"

	// NMStateDeploymentName is the NMState operator deployment name.
	NMStateDeploymentName = "nmstate-operator"

	// MetalLBOperatorNamespace is a metallb operator namespace.
	MetalLBOperatorNamespace = "metallb-system"

	// MetalLBOperatorName is a metallb operator name.
	MetalLBOperatorName = "metallb-operator"

	// MetalLBDaemonSetName default metalLb speaker daemonset names.
	MetalLBDaemonSetName = "speaker"

	// MetalLBSubscriptionName is a metallb operator subscription name.
	MetalLBSubscriptionName = "metallb-operator-subscription"

	// MetalLBOperatorDeploymentName is a metallb operator deployment name.
	MetalLBOperatorDeploymentName = "metallb-operator-controller-manager"

	// MetalLBInstanceName is a metallb operator namespace.
	MetalLBInstanceName = "metallb"

	// LSONamespace is a local storage operator namespace.
	LSONamespace = "openshift-local-storage"

	// LSOName is a local storage operator instance name pattern.
	LSOName = "local-storage-operator"

	// ODFNamespace is an odf namespace.
	ODFNamespace = "openshift-storage"

	// StorageClassName is a storage class name.
	StorageClassName = "ocs-storagecluster-cephfs"

	// ESKOperatorName is an elasticsearch operator name.
	ESKOperatorName = "elasticsearch-operator"

	// ESKNamespace is an elasticsearch operator namespace.
	ESKNamespace = "openshift-operators-redhat"

	// ESKInstanceName is an elasticsearch instance name.
	ESKInstanceName = "instance"

	// CLOName is a clusterlogging operator name.
	CLOName = "cluster-logging"

	// CLONamespace is a clusterlogging operator namespace.
	CLONamespace = "openshift-logging"

	// CLODeploymentName is a clusterlogging operator deployment name.
	CLODeploymentName = "cluster-logging-operator"

	// CLOInstanceName is a clusterlogging instance name.
	CLOInstanceName = "instance"

	// DTPONamespace is a distributed tracing platform operator namespace.
	DTPONamespace = "openshift-distributed-tracing"

	// DTPOperatorSubscriptionName is a distributed tracing platform operator subscription name.
	DTPOperatorSubscriptionName = "jaeger-product"

	// DTPOperatorDeploymentName is a distributed tracing platform operator deployment name.
	DTPOperatorDeploymentName = "jaeger-operator"

	// KialiNamespace is a kiali operator namespace.
	KialiNamespace = "openshift-operators"

	// KialiOperatorSubscriptionName is a kiali operator subscription name.
	KialiOperatorSubscriptionName = "kiali-ossm"

	// KialiOperatorDeploymentName is a kiali operator deployment name.
	KialiOperatorDeploymentName = "kiali-operator"

	// SMONamespace is a service mesh operator namespace.
	SMONamespace = "openshift-operators"

	// SMOSubscriptionName is a service mesh operator subscription name.
	SMOSubscriptionName = "servicemeshoperator"

	// SMODeploymentName is a service mesh operator deployment name.
	SMODeploymentName = "istio-operator"

	// IstioNamespace is an istio operator namespace.
	IstioNamespace = "istio-system"

	// NTONamespace is a node tuning operator namespace.
	NTONamespace = "openshift-cluster-node-tuning-operator"

	// NTODeploymentName is a node tuning operator deployment name.
	NTODeploymentName = "cluster-node-tuning-operator"

	// SRIOVNamespace is a SR-IOV operator namespace.
	SRIOVNamespace = "openshift-sriov-network-operator"

	// SRIOVSubscriptionName is a SR-IOV operator subscription name.
	SRIOVSubscriptionName = "sriov-network-operator-subscription"

	// SRIOVDeploymentName is a SR-IOV operator deployment name.
	SRIOVDeploymentName = "sriov-network-operator"

	// SRIOVInjectorDaemonsetName is a SR-IOV operator injector daemonset name.
	SRIOVInjectorDaemonsetName = "network-resources-injector"

	// SRIOVWebhookDaemonsetName is a SR-IOV operator webhook daemonset name.
	SRIOVWebhookDaemonsetName = "operator-webhook"

	// KedaNamespace is a keda operator namespace.
	KedaNamespace = "openshift-keda"

	// KedaWatchNamespace is a keda watch namespace name.
	KedaWatchNamespace = "test-appspace"

	// KedaSubscriptionName is a keda operator subscription name.
	KedaSubscriptionName = "openshift-custom-metrics-autoscaler-operator"

	// KedaDeploymentName is a keda operator deployment name.
	KedaDeploymentName = "custom-metrics-autoscaler-operator"

	// KedaControllerName is a kedaController name.
	KedaControllerName = "keda"
)
