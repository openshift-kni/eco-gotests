package tests

import (
	"fmt"
	"time"

	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/mco"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-goinfra/pkg/scc"
	"github.com/openshift-kni/eco-goinfra/pkg/sriov"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/dpdk/internal/tsparams"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netinittools"
	"github.com/openshift-kni/eco-gotests/tests/internal/cluster"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
)

const (
	srIovNetworkTwoResName = "dpdkpolicytwo"
	srIovPolicyOneResName  = "dpdkpolicyone"
	setSEBool              = "setsebool container_use_devices "
)

var (
	workerNodes *nodes.Builder
	mcp         *mco.MCPBuilder
)

var _ = Describe("rootless", Ordered, Label(tsparams.LabelSuite), ContinueOnFailure, func() {

	Context("server-tx, client-rx connectivity test on different nodes", Label("rootless"), func() {
		BeforeAll(func() {

			By("Discover worker nodes")
			workerNodes = nodes.NewBuilder(APIClient, NetConfig.WorkerLabelMap)
			err := workerNodes.Discover()
			Expect(err).ToNot(HaveOccurred(), "Fail to discover nodes")

			By(fmt.Sprintf("Pulling MCP based on label %s", NetConfig.CnfMcpLabel))
			mcp, err = mco.Pull(APIClient, NetConfig.CnfMcpLabel)
			Expect(err).ToNot(HaveOccurred(), "Fail to pull MCP ")

			By("Collecting SR-IOV interface for rootless dpdk tests")
			srIovInterfacesUnderTest, err := NetConfig.GetSriovInterfaces(1)
			Expect(err).ToNot(HaveOccurred(), "Failed to retrieve SR-IOV interfaces for testing")

			By("Creating first dpdk-policy")
			_, err = sriov.NewPolicyBuilder(
				APIClient,
				"dpdk-policy-one",
				NetConfig.SriovOperatorNamespace,
				srIovPolicyOneResName,
				5,
				[]string{fmt.Sprintf("%s#0-1", srIovInterfacesUnderTest[0])},
				NetConfig.WorkerLabelMap).WithMTU(1500).WithDevType("vfio-pci").WithVhostNet(true).Create()
			Expect(err).ToNot(HaveOccurred(), "Fail to create dpdk policy")

			By("Creating second dpdk-policy")
			_, err = sriov.NewPolicyBuilder(
				APIClient,
				"dpdk-policy-two",
				NetConfig.SriovOperatorNamespace,
				srIovNetworkTwoResName,
				5,
				[]string{fmt.Sprintf("%s#2-4", srIovInterfacesUnderTest[0])},
				NetConfig.WorkerLabelMap).WithMTU(1500).WithDevType("vfio-pci").WithVhostNet(false).Create()
			Expect(err).ToNot(HaveOccurred(), "Fail to create dpdk policy")

			By("Waiting until cluster is stable")
			err = mcp.WaitToBeStableFor(time.Minute, tsparams.MCOWaitTimeout)
			Expect(err).ToNot(HaveOccurred(), "Failed to wait until cluster is stable")

			By("Setting selinux flag container_use_devices to 1 on all compute nodes")
			err = cluster.ExecCmd(APIClient, NetConfig.WorkerLabel, setSEBool+"1")
			Expect(err).ToNot(HaveOccurred(), "Fail to enable selinux flag")
		})

		It("single VF, multiple tap devices, multiple mac-vlans", polarion.ID("63806"), func() {
			Skip("TODO")
		})
	})

	AfterAll(func() {
		By("Removing all pods from test namespace")
		runningNamespace, err := namespace.Pull(APIClient, tsparams.TestNamespaceName)
		Expect(err).ToNot(HaveOccurred(), "Failed to pull namespace")
		Expect(runningNamespace.CleanObjects(tsparams.WaitTimeout, pod.GetGVR())).ToNot(HaveOccurred())

		By("Removing all SR-IOV Policy")
		err = sriov.CleanAllNetworkNodePolicies(APIClient, NetConfig.SriovOperatorNamespace, metaV1.ListOptions{})
		Expect(err).ToNot(HaveOccurred(), "Fail to clean srIovPolicy")

		By("Removing all srIovNetworks")
		err = sriov.CleanAllNetworksByTargetNamespace(
			APIClient, NetConfig.SriovOperatorNamespace, tsparams.TestNamespaceName, metaV1.ListOptions{})
		Expect(err).ToNot(HaveOccurred(), "Fail to clean srIovNetworks")

		By("Removing SecurityContextConstraints")
		testScc, err := scc.Pull(APIClient, "scc-test-admin")
		if err == nil {
			err = testScc.Delete()
			Expect(err).ToNot(HaveOccurred(), "Fail to remove scc")
		}

		By("Waiting until cluster is stable")
		err = mcp.WaitToBeStableFor(time.Minute, tsparams.MCOWaitTimeout)
		Expect(err).ToNot(HaveOccurred(), "Fail to wait until cluster is stable")
	})
})
