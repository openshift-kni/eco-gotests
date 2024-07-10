package rdsmanagementconfig

const (
	// Label is used to select system tests for management cluster deployment.
	Label = "rdsmanagement"
	// LabelValidateDeployment is used to select all basic management cluster deployment tests.
	LabelValidateDeployment = "rds-management-deployment"
	// LabelValidateOdf is used to select ODF deployment and configuration tests.
	LabelValidateOdf = "rds-management-odf"
	// LabelValidateIdm is used to select IDM deployment and configuration tests.
	LabelValidateIdm = "rds-management-idm"
	// LabelValidateKafka is used to select Kafka deployment and configuration tests.
	LabelValidateKafka = "rds-management-kafka"
	// LabelValidateSatellite is used to select Satellite deployment and configuration tests.
	LabelValidateSatellite = "rds-management-satellite"
	// LabelValidateQuay is used to select Quay deployment and configuration tests.
	LabelValidateQuay = "rds-management-quay"
	// LabelValidateOpenshiftVirtualization is used to select OpenShift Virtualization deployment and 
	// configuration tests.
	LabelValidateOpenshiftVirtualization = "rds-management-openshift-virtualization"

	// RdsManagementLogLevel configures logging level for management related tests.
	RdsManagementLogLevel = 90

	
	// AppsNadName is the name of the NetworkAttachmentDefinition used by the applications.
	AppsNadName = "apps-nad"
    // SatelliteNadName is the name of the NetworkAttachmentDefinition used by the Satellite.
	SatelliteNadName = "sat-nad"

	// NMStateOperatorName is the name of the NMState operator.
	NMStateOperatorName = "kubernetes-nmstate-operator"
	// NMStateInstanceName is the name of the NMState instance.
	NMStateInstanceName = "nmstate"
	
	// PerformanceAddonOperatorName is the name of the Performance Addon operator.
	PerformanceAddonOperatorName = "performance-addon-operator"

	// OpenshiftVirtualizationOperatorName is the name of the OpenShift Virtualization operator.
	OpenshiftVirtualizationOperatorName = "kubevirt-hyperconverged"

	// LocalStorageNS is the namespace of the Local Storage operator.
	LocalStorageNS = "openshift-local-storage"
	// LocalStorageOperatorName is the name of the Local Storage operator.
	LocalStorageOperatorName = "local-storage-operator"

	// OdfNS is the namespace of ODF.
	OdfNS = "openshift-storage"
	// OdfOperatorName is the name of the ODF operator.
	OdfOperatorName = "odf-operator"

	// QuayOperatorName is the name of the Quay operator.
	QuayOperatorName = "quay-operator"
	
	// MetalLBOperatorName is the name of the MetalLB operator.
	MetalLBOperatorName = "metallb-operator"
	// MetalLBInstanceName is a metallb operator namespace.
	MetalLBInstanceName = "metallb"
	// MetalLBDaemonSetName default metalLb speaker daemonset names.
	MetalLBDaemonSetName = "speaker"
	// MetalLBSubscriptionName is a metallb operator subscription name.
	MetalLBSubscriptionName = "metallb-operator-subscription"
	// MetalLBOperatorDeploymentName is a metallb operator deployment name.
	MetalLBOperatorDeploymentName = "metallb-operator-controller-manager"

	// AcmOperatorName is the name of the ACM operator.
	AcmOperatorName = "advanced-cluster-management"
	// AcmInstanceKind is the ACM instance kind.
	AcmInstanceKind = "MultiClusterHub"
	// AcmInstanceName is the name of the ACM instance name.
	AcmInstanceName = "multiclusterhub"

	// KafkaOperatorName is the name of the Kafka operator.
	KafkaOperatorName = "amq-streams"

	// MonitoringNS is the namespace of monitoring.
	MonitoringNS = "openshift-monitoring"
	// KafkaAdapterNamespace is the namespace of the kafka adapter.
	KafkaAdapterNamespace = "prom-kafka-adapter"

	// ElasticsearchNS is the namespace of elasticsearch.
	ElasticsearchNS = "openshift-operators-redhat"
	// ElasticsearchOperatorName is the name of the elasticsearch operator.
	ElasticsearchOperatorName = "elasticsearch-operator"

	// LoggingNS is the namespace of OpenShift Logging.
	LoggingNS = "openshift-logging"
	// LoggingOperatorName is the name of the OpenShift Logging operator.
	LoggingOperatorName = "cluster-logging"
	
	// AnsibleOperatorName is the name of the Ansible Automation Platform operator.
	AnsibleOperatorName = "ansible-automation-platform-operator"
	// AnsibleInstanceKind is the Ansible Automation Platform instance kind.
	AnsibleInstanceKind = "AutomationController"
	// AnsibleInstanceName is the name of the Ansible Automation Platform instance name.
	AnsibleInstanceName = "npss"

	// CertManagerNS is the namespace of Cert Manager.
	CertManagerNS = "openshift-cert-manager-operator"
	// CertManagerOperatorName is the name of the Cert Manager operator.
	CertManagerOperatorName = "openshift-cert-manager-operator"
	// CertManagerSubscriptionName is the Cert Manager operator subscription name.
	CertManagerSubscriptionName = "openshift-cert-manager-operator"

	// AMQ7OperatorName is the name of the AMQ7 operator.
	AMQ7OperatorName = "amq7-interconnect-operator"
	// AMQ7SubscriptionName is the AMQ7 operator subscription name.
	AMQ7SubscriptionName = "amq7-interconnect-operator"

	// PrometheusOperatorName is the name of the Prometheus operator.
	PrometheusOperatorName = "prometheus"
	// PrometheusSubscriptionName is the Prometheus operator subscription name.
	PrometheusSubscriptionName = "prometheus"
	// PrometheusSubscriptionSource is the Prometheus operator subscription source.
	PrometheusSubscriptionSource = "community-operators"

	// ECKOperatorName is the name of the ECK operator.
	ECKOperatorName = "elasticsearch-eck-operator-certified"
	// ECKSubscriptionName is the ECK operator subscription name.
	ECKSubscriptionName = "elasticsearch-eck-operator-certified"
	// ECKSubscriptionSource is the ECK operator subscription source.
	ECKSubscriptionSource = "certified-operators"

	// StfOperatorName is the namespace of STF.
	StfOperatorName = "service-telemetry-operator"
	// STFSubscriptionName is the ECK operator subscription name.
	StfSubscriptionName = "service-telemetry-operator"

	// ConditionTypeReadyString constant to fix linter warning.
	ConditionTypeReadyString = "Ready"

	// ConstantTrueString constant to fix linter warning.
	ConstantTrueString = "True"
)
