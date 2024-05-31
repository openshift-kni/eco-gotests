package spoke_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-gotests/pkg/assisted"
	"github.com/openshift-kni/eco-gotests/pkg/configmap"
	"github.com/openshift-kni/eco-gotests/tests/assisted/ztp/internal/find"
	"github.com/openshift-kni/eco-gotests/tests/assisted/ztp/internal/meets"
	. "github.com/openshift-kni/eco-gotests/tests/assisted/ztp/internal/ztpinittools"
	"github.com/openshift-kni/eco-gotests/tests/assisted/ztp/spoke/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/internal/reportxml"
	hiveextV1Beta1 "github.com/openshift/assisted-service/api/hiveextension/v1beta1"
	"gopkg.in/yaml.v2"
)

var _ = Describe(
	"PlatformTypeVerification",
	Ordered,
	ContinueOnFailure,
	Label(tsparams.LabelPlatformVerificationTestCases), func() {

		var (
			agentClusterInstall  *assisted.AgentClusterInstallBuilder
			platformType         string
			spokeClusterPlatform map[interface{}]interface{}
		)

		BeforeAll(func() {

			By("Check that spoke API client is ready")
			reqMet, msg := meets.SpokeAPIClientReadyRequirement()
			if !reqMet {
				Skip(msg)
			}

			By("Get spoke cluster name")
			spokeCluster, err := find.SpokeClusterName()
			Expect(err).NotTo(HaveOccurred(), "error getting spoke cluster name from APIClients")

			By("Get spoke cluster platformType from agentclusterinstall")
			agentClusterInstall, err = assisted.PullAgentClusterInstall(HubAPIClient, spokeCluster, spokeCluster)
			Expect(err).NotTo(HaveOccurred(), "error pulling agentclusterinstall from hub cluster")
			platformType = string(agentClusterInstall.Object.Status.PlatformType)

			By("Get spoke cluster-config configmap")
			clusterConfigMap, err := configmap.Pull(SpokeConfig.APIClient, "cluster-config-v1", "kube-system")
			Expect(err).NotTo(HaveOccurred(), "error pulling cluster config configmap from spoke cluster")
			Expect(clusterConfigMap.Object.Data["install-config"]).ToNot(
				BeEmpty(), "error pulling install-config from spoke cluster",
			)

			By("Read install-config data from configmap")
			installConfigData := make(map[interface{}]interface{})
			err = yaml.Unmarshal([]byte(clusterConfigMap.Object.Data["install-config"]), &installConfigData)
			Expect(err).NotTo(HaveOccurred(), "error reading in install-config as yaml")

			var ok bool
			spokeClusterPlatform, ok = installConfigData["platform"].(map[interface{}]interface{})
			Expect(ok).To(BeTrue(), "type assertion was wrong for install-config platform")
		})

		DescribeTable("none platform checks", func(masterCount int) {
			if platformType != string(hiveextV1Beta1.NonePlatformType) {
				Skip(fmt.Sprintf("Platform type was not %s", string(hiveextV1Beta1.NonePlatformType)))
			}
			if masterCount != agentClusterInstall.Object.Spec.ProvisionRequirements.ControlPlaneAgents {
				Skip("Did not match controlplane agent count")
			}
			Expect(spokeClusterPlatform["none"]).NotTo(BeNil(), "spoke does not contain a none platform key")
		},
			Entry("SNO install", 1, reportxml.ID("56200")),
			Entry("MNO install", 3, reportxml.ID("56202")),
		)

		It("installs on BareMetal platform", reportxml.ID("56203"), func() {
			if platformType != string(hiveextV1Beta1.BareMetalPlatformType) {
				Skip(fmt.Sprintf("Platform type was not %s", string(hiveextV1Beta1.BareMetalPlatformType)))
			}
			Expect(spokeClusterPlatform["baremetal"]).NotTo(BeNil(), "spoke does not contain a baremetal platform key")
		})

		It("installs on VSphere platform", reportxml.ID("56201"), func() {
			if platformType != string(hiveextV1Beta1.VSpherePlatformType) {
				Skip(fmt.Sprintf("Platform type was not %s", string(hiveextV1Beta1.VSpherePlatformType)))
			}
			Expect(spokeClusterPlatform["vsphere"]).NotTo(BeNil(), "spoke does not contain a vsphere platform key")
		})
	},
)
