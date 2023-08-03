package spoke_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/assisted"
	"github.com/openshift-kni/eco-gotests/tests/assisted/internal/url"
	"github.com/openshift-kni/eco-gotests/tests/assisted/ztp/internal/find"
	"github.com/openshift-kni/eco-gotests/tests/assisted/ztp/internal/meets"
	. "github.com/openshift-kni/eco-gotests/tests/assisted/ztp/internal/ztpinittools"
	"github.com/openshift-kni/eco-gotests/tests/assisted/ztp/spoke/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
)

var _ = Describe(
	"ACIDebugInfoLogsURL",
	Ordered,
	ContinueOnFailure,
	Label(tsparams.LabelDebugInfoLogsURLVerificationTestCases), func() {
		var (
			spokeClusterName, spokeClusterNameSpace string
			agentClusterInstall                     *assisted.AgentClusterInstallBuilder
			err                                     error
			debugInfoLogsURLACI                     string
		)

		BeforeAll(func() {
			By("Check that spoke API client is ready")
			reqMet, msg := meets.SpokeAPIClientReadyRequirement()
			if !reqMet {
				Skip(msg)
			}

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

			By("Get the debug info logs URL from the AgentClusterInstall")
			debugInfoLogsURLACI = agentClusterInstall.Object.Status.DebugInfo.LogsURL
		})
		It("Assert agent cluster install debug logs url is set and content retrievable",
			polarion.ID("45450"), func() {

				By("Check that the DebugInfo logs url in AgentClusterInstall is set")
				Expect(debugInfoLogsURLACI).Should(ContainSubstring("https://"),
					"error verifying that debug info logs url in agent cluster install is set")

				By("Check that the DebugInfo logs in AgentClusterInstall can be retrieved")
				_, statusCode, err := url.Fetch(debugInfoLogsURLACI, "GET", true)
				Expect(err).ToNot(HaveOccurred(), "error retrieving log content: %s", err)

				By("Check that the DebugInfo logs AgentClusterInstall return 200 when downloaded")
				Expect(statusCode).To(Equal(200), "error verifying http return code is 200")
			})
	})
