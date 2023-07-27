package tests

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift-kni/eco-goinfra/pkg/deployment"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"

	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"

	. "github.com/openshift-kni/eco-gotests/tests/rhwa/internal/rhwainittools"
	"github.com/openshift-kni/eco-gotests/tests/rhwa/internal/rhwaparams"
	"github.com/openshift-kni/eco-gotests/tests/rhwa/nmo-operator/internal/nmoparams"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe(
	"NMO tests",
	Ordered,
	ContinueOnFailure,
	Label(nmoparams.Label), func() {
		BeforeAll(func() {
			By("Get NMO deployment object")
			nmoDeployment, err := deployment.Pull(
				APIClient, nmoparams.OperatorDeploymentName, rhwaparams.RhwaOperatorNs)
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to get NMO deployment %s", err))

			By("Verify NMO deployment is Ready")
			Expect(nmoDeployment.IsReady(rhwaparams.DefaultTimeout)).To(BeTrue(), "NMO deployment is not Ready")
		})
		It("Verify Node Maintenance Operator pod is running", polarion.ID("46315"), func() {
			_, err := pod.WaitForAllPodsInNamespaceRunning(
				APIClient,
				rhwaparams.RhwaOperatorNs,
				v1.ListOptions{},
				rhwaparams.DefaultTimeout,
			)
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Pod is not ready %s", err))
		})
	})
