package tests

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift-kni/eco-goinfra/pkg/deployment"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"

	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"

	. "github.com/openshift-kni/eco-gotests/tests/rhwa/internal/rhwainittools"
	"github.com/openshift-kni/eco-gotests/tests/rhwa/internal/rhwaparams"
	"github.com/openshift-kni/eco-gotests/tests/rhwa/mdr-operator/internal/mdrparams"
)

var _ = Describe(
	"MDR tests",
	Ordered,
	ContinueOnFailure,
	Label(mdrparams.Label), func() {
		BeforeAll(func() {
			By("Get MDR deployment object")
			mdrDeployment, err := deployment.Pull(
				APIClient, mdrparams.OperatorDeploymentName, rhwaparams.RhwaOperatorNs)
			Expect(err).ToNot(HaveOccurred(), "Failed to get MDR deployment")

			By("Verify MDR deployment is Ready")
			Expect(mdrDeployment.IsReady(rhwaparams.DefaultTimeout)).To(BeTrue(), "MDR deployment is not Ready")
		})
		It("Verify Machine Deletion Remediation Operator pod is running", polarion.ID("65767"), func() {
			_, err := pod.WaitForAllPodsInNamespaceRunning(
				APIClient,
				rhwaparams.RhwaOperatorNs,
				rhwaparams.DefaultTimeout,
			)
			Expect(err).ToNot(HaveOccurred(), "Pod is not ready")
		})
	})
