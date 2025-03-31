package ocloudparams

const (
	// Label represents O-Cloud system tests label that can be used for test cases selection.
	Label = "ocloud"

	// OCloudLogLevel configures logging level for O-Cloud related tests.
	OCloudLogLevel = 90

	// AcmNamespace is the namespace for ACM.
	AcmNamespace = "rhacm"
	// AcmSubscriptionName is the name of the ACM operator subscription.
	AcmSubscriptionName = "acm-operator-subscription"

	// OpenshiftGitOpsNamespace is the namespace for the GitOps operator.
	OpenshiftGitOpsNamespace = "openshift-operators"
	// OpenshiftGitOpsSubscriptionName is the name of the GitOps operator subscription.
	OpenshiftGitOpsSubscriptionName = "openshift-gitops-operator-subscription"

	// OCloudO2ImsNamespace is the namespace for the O-Cloud manager operator.
	OCloudO2ImsNamespace = "oran-o2ims"
	// OCloudO2ImsSubscriptionName is the name of the O-Cloud manager operator subscription.
	OCloudO2ImsSubscriptionName = "oran-o2ims-operator-subscription"

	// OCloudHardwareManagerPluginNamespace is the namespace for the O-Cloud hardware manager plugin operator.
	OCloudHardwareManagerPluginNamespace = "oran-hwmgr-plugin"
	//nolint:lll
	// OCloudHardwareManagerPluginSubscriptionName is the name of the O-Cloud hardware manager plugin operator subscription.
	OCloudHardwareManagerPluginSubscriptionName = "oran-hwmgr-plugin-operator-subscription"

	// PtpNamespace is the namespace for the PTP operator.
	PtpNamespace = "openshift-ptp"
	// PtpOperatorSubscriptionName is the name of the PTP operator subscription.
	PtpOperatorSubscriptionName = "ptp-operator-subscription"
	// PtpDeploymentName is the name of the PTP deployment.
	PtpDeploymentName = "ptp-operator"
	// PtpContainerName is the name of the PTP container.
	PtpContainerName = "ptp-operator"

	// SriovNamespace is the namespace for the SR-IOV operator.
	SriovNamespace = "openshift-sriov-network-operator"

	// LifecycleAgentNamespace is the namespace for the Lifecycle Agent operator.
	LifecycleAgentNamespace = "openshift-lifecycle-agent"
)
