package spoke_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/assisted"
	"github.com/openshift-kni/eco-goinfra/pkg/configmap"
	"github.com/openshift-kni/eco-gotests/tests/assisted/ztp/internal/find"
	"github.com/openshift-kni/eco-gotests/tests/assisted/ztp/internal/installconfig"
	. "github.com/openshift-kni/eco-gotests/tests/assisted/ztp/internal/ztpinittools"
	"github.com/openshift-kni/eco-gotests/tests/assisted/ztp/spoke/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
)

const (
	installConfigConfMap   = "cluster-config-v1"
	installConfigConfMapNS = "kube-system"
)

var _ = Describe(
	"Openshift Spoke cluster deployed with FIPS mode enabled",
	Ordered,
	ContinueOnFailure,
	Label(tsparams.LabelFipsVerificationTestCases), func() {
		var fipsEnabledOnSpoke bool
		When("on MCE 2.0 and above", func() {
			BeforeAll(func() {
				By("Getting configmap from hub")
				fipsConfMap, err := configmap.Pull(HubAPIClient, installConfigConfMap, installConfigConfMapNS)
				Expect(err).ToNot(HaveOccurred(), "error extracting configmap "+installConfigConfMap)
				Expect(fipsConfMap.Object.Data["install-config"]).ToNot(BeEmpty(),
					"error pulling install-config from HUB cluster")
				installConfigData, err := installconfig.NewInstallConfigFromString(fipsConfMap.Object.Data["install-config"])
				Expect(err).NotTo(HaveOccurred(), "error reading in install-config as yaml")
				if !installConfigData.FIPS {
					Skip("Hub cluster is not FIPS enabled")
				}
				By("Get the spoke cluster name")
				spokeClusterName, err := find.SpokeClusterName()
				spokeClusterNameSpace := spokeClusterName
				Expect(err).ToNot(HaveOccurred(),
					"error getting the spoke cluster name")

				By("Pull the AgentClusterInstall from the HUB")
				agentClusterInstall, err := assisted.PullAgentClusterInstall(
					HubAPIClient, spokeClusterName, spokeClusterNameSpace)
				Expect(err).ToNot(HaveOccurred(),
					"error pulling agentclusterinstall %s in namespace %s", spokeClusterName, spokeClusterNameSpace)
				By("Checking agentclusterinstall has fips:true annotation")
				fipsEnabledOnSpoke = false
				if override,
					ok := agentClusterInstall.Object.Annotations["agent-install.openshift.io/install-config-overrides"]; ok {
					agentClusterInstallOverrideConfig, err := installconfig.NewInstallConfigFromString(override)
					Expect(err).ToNot(HaveOccurred(), "error getting installconfig from spoke override")

					if agentClusterInstallOverrideConfig.FIPS {
						fipsEnabledOnSpoke = true
					}
				}
				if !fipsEnabledOnSpoke {
					Skip("spoke should be installed with fips enabled")
				}
			})

			It("Assert Spoke cluster was deployed with FIPS", polarion.ID("65865"), func() {
				By("Getting configmap")
				fipsConfMap, err := configmap.Pull(SpokeConfig.APIClient, installConfigConfMap, installConfigConfMapNS)
				Expect(err).ToNot(HaveOccurred(), "error extracting configmap "+installConfigConfMap)
				Expect(fipsConfMap.Object.Data["install-config"]).ToNot(BeEmpty(),
					"error pulling install-config from spoke cluster")
				installConfigData, err := installconfig.NewInstallConfigFromString(fipsConfMap.Object.Data["install-config"])
				Expect(err).NotTo(HaveOccurred(), "error reading in install-config as yaml")
				Expect(installConfigData.FIPS).To(BeTrue(),
					"spoke Cluster is not installed with fips")
			})
		})
	})
