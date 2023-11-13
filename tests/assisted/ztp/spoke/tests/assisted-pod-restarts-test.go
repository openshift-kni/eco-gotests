package spoke_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
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
				if ZTPConfig.SpokeAgentClusterInstall.Object.Status.DebugInfo.State != models.ClusterStatusAddingHosts {
					Skip("spoke cluster has not been installed")
				}

			})

			DescribeTable("no restarts for assisted pods",
				func(podName string, getPodName func() *pod.Builder) {

					By("Get the " + podName + " pod")
					podBuilder := getPodName()

					By("Assure the " + podName + " pod didn't restart")
					Expect(podBuilder.Object.Status.ContainerStatuses[0].RestartCount).To(Equal(int32(0)),
						"failed asserting 0 restarts for "+podName+" pod")
				},
				Entry("Assert the assisted-service pod wasn't restarted after creation",
					"assisted-service", ZTPConfig.HubAssistedServicePod, polarion.ID("56581")),
				Entry("Assert the assisted-image-service pod wasn't restarted after creation",
					"assisted-image-service", ZTPConfig.HubAssistedImageServicePod, polarion.ID("56582")),
			)

		})
	})
