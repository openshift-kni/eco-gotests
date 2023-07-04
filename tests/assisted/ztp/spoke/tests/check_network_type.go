package spoke_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/assisted"
	"github.com/openshift-kni/eco-goinfra/pkg/network"
	"github.com/openshift-kni/eco-gotests/tests/assisted/ztp/internal/find"
	"github.com/openshift-kni/eco-gotests/tests/assisted/ztp/internal/meets"
	. "github.com/openshift-kni/eco-gotests/tests/assisted/ztp/internal/ztpinittools"
	"github.com/openshift-kni/eco-gotests/tests/assisted/ztp/spoke/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
)

var _ = Describe(
	"NetworkType",
	Ordered,
	ContinueOnFailure,
	Label(tsparams.LabelNetworkTypeVerificationTestCases), func() {
		var (
			spokeClusterName, spokeClusterNameSpace string
			agentClusterInstall                     *assisted.AgentClusterInstallBuilder
			err                                     error
			networkTypeACI                          string
			agentClusterInstallConditionMessage     string
		)

		When("on MCE 2.0 and above", func() {
			BeforeAll(func() {
				By("Get the spoke cluster name")
				spokeClusterName, err = find.SpokeClusterName()
				spokeClusterNameSpace = spokeClusterName
				Expect(err).ToNot(HaveOccurred(),
					"error getting the spoke cluster name")

				By("Pull the AgentClusterInstall from the HUB")
				agentClusterInstall, err = assisted.PullAgentClusterInstall(
					HubAPIClient, spokeClusterName, spokeClusterNameSpace)
				Expect(err).ToNot(HaveOccurred(),
					"error pulling agentclusterinstall %s in namespace %s", spokeClusterName, spokeClusterNameSpace)

				By("Get the networktype from the AgentClusterInstall")
				networkTypeACI = agentClusterInstall.Object.Spec.Networking.NetworkType
				Expect(networkTypeACI).To(Or(Equal("OVNKubernetes"), Equal("OpenShiftSDN"), Equal("")))

				By("Get the AgentClusterInstall conditions with type set to 'Completed'")
				agentClusterInstallCondition, err := agentClusterInstall.GetCondition("Completed")
				Expect(err).ToNot(HaveOccurred(),
					"error getting spoke cluster installation conditions")

				By("Initialize the agentClusterInstallConditionMessage")
				agentClusterInstallConditionMessage = agentClusterInstallCondition.Message

			})
			It("Assert IPv4 spoke cluster with OVNKubernetes set as NetworkType gets deployed",
				polarion.ID("44899"), func() {

					expectedNetworkType := "OVNKubernetes"

					By("Check that spoke cluster is IPV4 Single Stack")
					reqMet, msg := meets.SpokeSingleStackIPv4Requirement()
					if !reqMet {
						Skip(msg)
					}

					By("Check that the networktype in AgentClusterInstall is set")
					if networkTypeACI != expectedNetworkType {
						Skip("the network type in ACI is not set to OVNKubernetes")
					}

					By("Check that the deployment of the spoke has completed")
					Expect(agentClusterInstallConditionMessage).Should(ContainSubstring("The installation has completed"))

				})
			It("Assert the NetworkType in the IPV4 spoke matches ACI and is set to OVNKubernetes",
				polarion.ID("44900"), func() {

					expectedNetworkType := "OVNKubernetes"

					By("Check that spoke cluster is IPV4 Single Stack")
					reqMet, msg := meets.SpokeSingleStackIPv4Requirement()
					if !reqMet {
						Skip(msg)
					}

					By("Check that the networktype in AgentClusterInstall is set")
					if networkTypeACI != expectedNetworkType {
						Skip("the network type in ACI is not set to OVNKubernetes")
					}

					By("Get the network config from the spoke")
					spokeClusterNetwork, err := network.PullConfig(SpokeConfig.APIClient)
					Expect(err).ToNot(HaveOccurred(),
						"error pulling network configuration from the spoke")

					By("Assure the networktype in AgentClusterInstall matches the networktype in the spoke")
					Expect(expectedNetworkType).To(Equal(spokeClusterNetwork.Object.Spec.NetworkType))
				})

		})
	})
