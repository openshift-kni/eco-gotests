package ocloudparams

import (
	"time"
)

const (
	// Label represents O-Cloud system tests label that can be used for test cases selection.
	Label = "ocloud"

	// DefaultTimeout is the timeout used for test resources creation.
	DefaultTimeout = 900 * time.Second

	// OCloudLogLevel configures logging level for O-Cloud related tests.
	OCloudLogLevel = 90

	// AcmNamespace is the namespace for ACM.
	AcmNamespace = "rhacm"

	// AcmSubscriptionName is the name of the ACM operator subscription.
	AcmSubscriptionName = "acm-operator-subscription"

	// AcmInstanceName is the name of the ACM multicluster hub instance.
	AcmInstanceName = "multiclusterhub"

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

	// PtpCPURequest is cpu request for the PTP container.
	PtpCPURequest = "50m"

	// PtpMemoryRequest is cpu request for the PTP container.
	PtpMemoryRequest = "100Mi"

	// PtpCPULimit is cpu limit for the PTP container.
	PtpCPULimit = "1m"

	// PtpMemoryLimit is cpu limit for the PTP container.
	PtpMemoryLimit = "1Mi"

	// SriovNamespace is the namespace for the SR-IOV operator.
	SriovNamespace = "openshift-sriov-network-operator"

	// LifecycleAgentNamespace is the namespace for the Lifecycle Agent operator.
	LifecycleAgentNamespace = "openshift-lifecycle-agent"

	//nolint:lll
	// PodmanTagOperatorUpgrade command to create a tag for the redhat-operators upgrade.
	PodmanTagOperatorUpgrade = "podman tag registry.hub01.oran.telcoqe.eng.rdu2.dc.redhat.com:5000/olm/redhat-operators:v4.18-new registry.hub01.oran.telcoqe.eng.rdu2.dc.redhat.com:5000/olm/redhat-operators:v4.18-day2"
	//nolint:lll
	// PodmanTagSriovUpgrade command to create a tag for the SR-IOV FEC operator upgrade.
	PodmanTagSriovUpgrade = "podman tag registry.hub01.oran.telcoqe.eng.rdu2.dc.redhat.com:5000/olm/far-edge-sriov-fec:v4.18-new registry.hub01.oran.telcoqe.eng.rdu2.dc.redhat.com:5000/olm/far-edge-sriov-fec:v4.18-day2"
	//nolint:lll
	// PodmanPushOperatorUpgrade command to push the redhat-operators upgrade version.
	PodmanPushOperatorUpgrade = "podman push registry.hub01.oran.telcoqe.eng.rdu2.dc.redhat.com:5000/olm/redhat-operators:v4.18-day2"
	//nolint:lll
	// PodmanPushSriovUpgrade command to push the SR-IOV FEC operator upgrade version.
	PodmanPushSriovUpgrade = "podman push registry.hub01.oran.telcoqe.eng.rdu2.dc.redhat.com:5000/olm/far-edge-sriov-fec:v4.18-day2"
	//nolint:lll
	// PodmanTagOperatorDowngrade command to create a tag for the redhat-operators downgrade.
	PodmanTagOperatorDowngrade = "podman tag registry.hub01.oran.telcoqe.eng.rdu2.dc.redhat.com:5000/olm/redhat-operators:v4.18-old registry.hub01.oran.telcoqe.eng.rdu2.dc.redhat.com:5000/olm/redhat-operators:v4.18-day2"
	//nolint:lll
	// PodmanTagSriovDowngrade command to create a tag for the SR-IOV FEC operator downgrade.
	PodmanTagSriovDowngrade = "podman tag registry.hub01.oran.telcoqe.eng.rdu2.dc.redhat.com:5000/olm/far-edge-sriov-fec:v4.18 registry.hub01.oran.telcoqe.eng.rdu2.dc.redhat.com:5000/olm/far-edge-sriov-fec:v4.18-day2"
	//nolint:lll
	// PodmanPushOperatorDowngrade command to push the redhat-operators downgrade version.
	PodmanPushOperatorDowngrade = "podman push registry.hub01.oran.telcoqe.eng.rdu2.dc.redhat.com:5000/olm/redhat-operators:v4.18-day2"
	//nolint:lll
	// PodmanPushSriovDowngrade command to push the SR-IOV FEC operator downgrade version.
	PodmanPushSriovDowngrade = "podman push registry.hub01.oran.telcoqe.eng.rdu2.dc.redhat.com:5000/olm/far-edge-sriov-fec:v4.18-day2"
	//nolint:lll
	// SnoKubeconfigCreate command to get the SNO kubeconfig file.
	SnoKubeconfigCreate = "oc -n %s get secret %s-admin-kubeconfig -o json | jq -r .data.kubeconfig | base64 -d > tmp/%s/auth/kubeconfig"
	//nolint:lll
	// CreateImageBasedInstallationConfig command to create the image based installation configuration template.
	CreateImageBasedInstallationConfig = "openshift-install image-based create image-config-template --dir tmp/ibi-iso-workdir"
	// CreateIsoImage command to create the ISO image.
	CreateIsoImage = "openshift-install image-based create image --dir tmp/ibi-iso-workdir"
	//nolint:lll
	// CheckIbiCompleted command to check that the image based installation has finished.
	CheckIbiCompleted = "journalctl -u install-rhcos-and-restore-seed.service | grep 'Finished SNO Image-based Installation.'"
	// SpokeSSHUser ssh user of the spoke cluster.
	SpokeSSHUser = "core"
	// SpokeSSHPasskeyPath path to the ssh key of the spoke cluster.
	SpokeSSHPasskeyPath = "/home/kni/.ssh/id_rsa"
	// SeedGeneratorName name of the seedgenerator CR.
	SeedGeneratorName = "seedimage"
	// RegistryCertPath path to the registry certificate.
	RegistryCertPath = "/opt/registry/certs/registry.crt"
	// IbiConfigTemplate template for the image based installation configuration.
	IbiConfigTemplate = "/home/kni/eco-gotests/tests/system-tests/o-cloud/internal/ocloudconfigfiles/ibi-config.yaml.tmpl"
	// IbiConfigTemplateYaml path to the YAML file with the image based installation configuration.
	IbiConfigTemplateYaml = "tmp/ibi-iso-workdir/image-based-installation-config.yaml"
	// IbiBasedImageSourcePath path to the base image.
	IbiBasedImageSourcePath = "tmp/ibi-iso-workdir/rhcos-ibi.iso"
)
