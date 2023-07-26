package tests

import (
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netenv"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netinittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/sriov/internal/sriovenv"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/sriov/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	workerNodeList *nodes.Builder
)

var _ = Describe("NUMAExcludeTopology", Ordered, Label(tsparams.LabelNUMASriovExcludeTopologyTestCases),
	ContinueOnFailure, func() {

		BeforeAll(func() {
			By("Verify cluster supports SRIOV test cases")
			err := netenv.DoesClusterHasEnoughNodes(APIClient, NetConfig, 3, 2)
			Expect(err).ToNot(HaveOccurred(), "Cluster does not support SRIOV test cases")

			By("Validating SR-IOV interfaces")
			workerNodeList = nodes.NewBuilder(APIClient, NetConfig.WorkerLabelMap)
			Expect(workerNodeList.Discover()).ToNot(HaveOccurred(), "Failed to discover worker nodes")
			Expect(sriovenv.ValidateSriovInterfaces(workerNodeList, 2)).ToNot(HaveOccurred(),
				"Failed to get required SR-IOV interfaces")
		})

		AfterAll(func() {
			By("Remove all SR-IOV networks")
			// Waiting for eco-infra PR
			By("Remove all SR-IOV policies")
			// Waiting for eco-infra PR
		})

		Context("NUMA ExcludeTopology", func() {
			It("Validate the creation of a pod with excludeTopology set to False and an SRIOV interface in a different "+
				"NUMA node than the pod", polarion.ID("63492"), func() {
				Skip("TODO")
			})
		})
	})
