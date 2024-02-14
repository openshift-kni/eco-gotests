package ztpconfig

import (
	"fmt"
	"strings"

	"github.com/golang/glog"
	"github.com/kelseyhightower/envconfig"
	"github.com/openshift-kni/eco-goinfra/pkg/assisted"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/configmap"
	"github.com/openshift-kni/eco-goinfra/pkg/hive"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-goinfra/pkg/secret"
	"github.com/openshift-kni/eco-gotests/tests/assisted/internal/assistedconfig"
	"github.com/openshift-kni/eco-gotests/tests/assisted/ztp/internal/find"
	"github.com/openshift-kni/eco-gotests/tests/assisted/ztp/internal/ztpparams"
	"github.com/openshift-kni/eco-gotests/tests/internal/cluster"
	. "github.com/openshift-kni/eco-gotests/tests/internal/inittools"
)

// ZTPConfig type contains ztp configuration.
type ZTPConfig struct {
	*assistedconfig.AssistedConfig
	*HubConfig
	*SpokeConfig
}

// HubConfig contains environment information related to the hub cluster.
type HubConfig struct {
	HubOCPVersion              string
	HubOCPXYVersion            string
	HubAgentServiceConfig      *assisted.AgentServiceConfigBuilder
	hubAssistedServicePod      *pod.Builder
	hubAssistedImageServicePod *pod.Builder
	HubPullSecret              *secret.Builder
	HubInstallConfig           *configmap.Builder
}

// SpokeConfig contains environment information related to the spoke cluster.
type SpokeConfig struct {
	SpokeAPIClient           *clients.Settings
	SpokeOCPVersion          string
	SpokeOCPXYVersion        string
	SpokeClusterName         string
	SpokeKubeConfig          string `envconfig:"ECO_ASSISTED_ZTP_SPOKE_KUBECONFIG"`
	SpokeClusterImageSet     string `envconfig:"ECO_ASSISTED_ZTP_SPOKE_CLUSTERIMAGESET"`
	SpokeClusterDeployment   *hive.ClusterDeploymentBuilder
	SpokeAgentClusterInstall *assisted.AgentClusterInstallBuilder
	SpokeInfraEnv            *assisted.InfraEnvBuilder
	SpokeInstallConfig       *configmap.Builder
}

// NewZTPConfig returns instance of ZTPConfig type.
func NewZTPConfig() *ZTPConfig {
	glog.V(ztpparams.ZTPLogLevel).Info("Creating new ZTPConfig struct")

	var ztpconfig ZTPConfig
	ztpconfig.AssistedConfig = assistedconfig.NewAssistedConfig()
	ztpconfig.newHubConfig()
	ztpconfig.newSpokeConfig()

	return &ztpconfig
}

// newHubConfig creates a new HubConfig member for a ZTPConfig.
func (ztpconfig *ZTPConfig) newHubConfig() {
	glog.V(ztpparams.ZTPLogLevel).Info("Creating new HubConfig struct")

	ztpconfig.HubConfig = new(HubConfig)

	ztpconfig.HubConfig.HubOCPVersion, _ = find.ClusterVersion(APIClient)

	splitVersion := strings.Split(ztpconfig.HubConfig.HubOCPVersion, ".")
	if len(splitVersion) >= 2 {
		ztpconfig.HubConfig.HubOCPXYVersion = fmt.Sprintf("%s.%s", splitVersion[0], splitVersion[1])
	}

	ztpconfig.HubConfig.HubAgentServiceConfig, _ = assisted.PullAgentServiceConfig(APIClient)
	if ztpconfig.HubConfig.HubAgentServiceConfig != nil {
		_ = ztpconfig.HubAssistedServicePod()
		_ = ztpconfig.HubAssistedImageServicePod()
	}

	ztpconfig.HubConfig.HubPullSecret, _ = cluster.GetOCPPullSecret(APIClient)
	ztpconfig.HubConfig.HubInstallConfig, _ = configmap.Pull(APIClient, "cluster-config-v1", "kube-system")
}

// newSpokeConfig creates a new SpokeConfig member for a ZTPConfig.
func (ztpconfig *ZTPConfig) newSpokeConfig() {
	glog.V(ztpparams.ZTPLogLevel).Info("Creating new SpokeConfig struct")

	ztpconfig.SpokeConfig = new(SpokeConfig)

	err := envconfig.Process("eco_assisted_ztp_spoke_", ztpconfig.SpokeConfig)
	if err != nil {
		glog.V(ztpparams.ZTPLogLevel).Infof("failed to instantiate SpokeConfig: %v", err)
	}

	if ztpconfig.SpokeConfig.SpokeKubeConfig != "" {
		glog.V(ztpparams.ZTPLogLevel).Infof("Creating spoke api client from %s", ztpconfig.SpokeConfig.SpokeKubeConfig)

		if ztpconfig.SpokeConfig.SpokeAPIClient = clients.New(
			ztpconfig.SpokeConfig.SpokeKubeConfig); ztpconfig.SpokeConfig.SpokeAPIClient == nil {
			glog.V(ztpparams.ZTPLogLevel).Infof("failed to load provided spoke kubeconfig: %v", err)
		}

		ztpconfig.SpokeConfig.SpokeClusterName, _ =
			find.SpokeClusterName(APIClient, ztpconfig.SpokeConfig.SpokeAPIClient)
		ztpconfig.SpokeConfig.SpokeOCPVersion, _ = find.ClusterVersion(ztpconfig.SpokeConfig.SpokeAPIClient)

		splitVersion := strings.Split(ztpconfig.SpokeConfig.SpokeOCPVersion, ".")
		if len(splitVersion) >= 2 {
			ztpconfig.SpokeConfig.SpokeOCPXYVersion = fmt.Sprintf("%s.%s", splitVersion[0], splitVersion[1])
		}

		ztpconfig.SpokeConfig.SpokeClusterDeployment, _ = hive.PullClusterDeployment(APIClient,
			ztpconfig.SpokeConfig.SpokeClusterName, ztpconfig.SpokeConfig.SpokeClusterName)

		ztpconfig.SpokeConfig.SpokeAgentClusterInstall, _ = assisted.PullAgentClusterInstall(APIClient,
			ztpconfig.SpokeConfig.SpokeClusterName, ztpconfig.SpokeConfig.SpokeClusterName)

		ztpconfig.SpokeConfig.SpokeInfraEnv, _ = assisted.PullInfraEnvInstall(APIClient,
			ztpconfig.SpokeConfig.SpokeClusterName, ztpconfig.SpokeConfig.SpokeClusterName)

		ztpconfig.SpokeConfig.SpokeInstallConfig, _ =
			configmap.Pull(ztpconfig.SpokeConfig.SpokeAPIClient, "cluster-config-v1", "kube-system")
	} else {
		ztpconfig.SpokeConfig.SpokeAPIClient = nil
	}

	if ztpconfig.SpokeConfig.SpokeClusterImageSet == "" {
		ztpconfig.SpokeConfig.SpokeClusterImageSet = ztpconfig.HubOCPXYVersion
	}
}

// HubAssistedServicePod retrieves the assisted service pod from the hub
// and populates hubAssistedServicePod.
func (ztpconfig *ZTPConfig) HubAssistedServicePod() *pod.Builder {
	if ztpconfig.hubAssistedServicePod == nil || !ztpconfig.hubAssistedServicePod.Exists() {
		ztpconfig.hubAssistedServicePod, _ = find.AssistedServicePod(APIClient)
	}

	return ztpconfig.hubAssistedServicePod
}

// HubAssistedImageServicePod retrieves the assisted image service pod from the hub
// and populates hubAssistedImageServicePod.
func (ztpconfig *ZTPConfig) HubAssistedImageServicePod() *pod.Builder {
	if ztpconfig.hubAssistedImageServicePod == nil || !ztpconfig.hubAssistedImageServicePod.Exists() {
		ztpconfig.hubAssistedImageServicePod, _ = find.AssistedImageServicePod(APIClient)
	}

	return ztpconfig.hubAssistedImageServicePod
}
