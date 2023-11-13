package spoke_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-gotests/tests/assisted/internal/url"
	. "github.com/openshift-kni/eco-gotests/tests/assisted/ztp/internal/ztpinittools"
	"github.com/openshift-kni/eco-gotests/tests/assisted/ztp/spoke/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
	"github.com/openshift/assisted-service/models"
)

var _ = Describe(
	"ACIDebugInfo",
	Ordered,
	ContinueOnFailure,
	Label(tsparams.LabelDebugInfoVerificationTestCases), func() {

		DescribeTable("aci debug info url checks", func(urlToCheck func() string) {
			Expect(urlToCheck()).Should(ContainSubstring("https://"),
				"error verifying that the debug info url in agent cluster install is set")
			_, statusCode, err := url.Fetch(urlToCheck(), "GET", true)
			Expect(err).ToNot(HaveOccurred(), "error retrieving content from url: %s", err)
			Expect(statusCode).To(Equal(200), "error verifying http return code is 200")
		},
			Entry("Logs URL", func() string { return ZTPConfig.SpokeAgentClusterInstall.Object.Status.DebugInfo.LogsURL },
				polarion.ID("42811")),
			Entry("Events URL", func() string { return ZTPConfig.SpokeAgentClusterInstall.Object.Status.DebugInfo.EventsURL },
				polarion.ID("42812")),
		)

		It("Assert agent cluster install state and stateInfo params are valid",
			polarion.ID("42813"), func() {

				By("Check that the DebugInfo state in AgentClusterInstall shows cluster is installed")
				Expect(ZTPConfig.SpokeAgentClusterInstall.Object.Status.DebugInfo.State).
					Should(Equal(models.ClusterStatusAddingHosts),
						"error verifying that DebugInfo state shows that the cluster is installed")

				By("Check that the DebugInfo stateInfo in AgentClusterInstall shows cluster is installed")
				Expect(ZTPConfig.SpokeAgentClusterInstall.Object.Status.DebugInfo.StateInfo).Should(Equal("Cluster is installed"),
					"error verifying that DebugInfo stateInfo shows that the cluster is installed")
			})
	})
