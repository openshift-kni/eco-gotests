package spoke_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/assisted"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-gotests/tests/assisted/ztp/internal/find"
	. "github.com/openshift-kni/eco-gotests/tests/assisted/ztp/internal/ztpinittools"
	"github.com/openshift-kni/eco-gotests/tests/assisted/ztp/spoke/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
	"github.com/openshift/assisted-service/models"
)

var _ = Describe(
	"AssistedPodsNoRestarts",
	Ordered,
	ContinueOnFailure,
	Label(tsparams.LabelAssistedPodsNoRestartsTestCases), func() {
		When("on MCE 2.0 and above", func() {

			BeforeAll(func() {
				By("Get spoke cluster name")
				spokeCluster, err := find.SpokeClusterName()
				Expect(err).NotTo(HaveOccurred(), "error getting spoke cluster name from APIClients")

				By("Check that the DebugInfo state in AgentClusterInstall shows cluster is installed")
				agentClusterInstall, err := assisted.PullAgentClusterInstall(HubAPIClient, spokeCluster, spokeCluster)
				Expect(err).NotTo(HaveOccurred(), "error pulling spoke's agentclusterinstall from hub cluster")
				if agentClusterInstall.Object.Status.DebugInfo.State != models.ClusterStatusAddingHosts {
					Skip("spoke cluster has not been installed")
				}

			})

			DescribeTable("no restarts for assisted pods",
				func(podName string, getPodName func() (*pod.Builder, error)) {

					By("Get the " + podName + " pod")
					podBuilder, err := getPodName()
					Expect(err).ShouldNot(HaveOccurred(), "failed to search for "+podName+" pod")

					By("Assure the " + podName + " pod didn't restart")
					Expect(podBuilder.Object.Status.ContainerStatuses[0].RestartCount).To(Equal(int32(0)),
						"failed asserting 0 restarts for "+podName+" pod")
				},
				Entry("Assert the assisted-service pod wasn't restarted after creation",
					"assisted-service", find.AssistedServicePod, polarion.ID("56581")),
				Entry("Assert the assisted-image-service pod wasn't restarted after creation",
					"assisted-image-service", find.AssistedImageServicePod, polarion.ID("56582")),
			)

		})
	})
