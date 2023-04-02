package tests

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-gotests/pkg/nad"
	"github.com/openshift-kni/eco-gotests/pkg/namespace"
	"github.com/openshift-kni/eco-gotests/pkg/nodes"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/internal/coreparams"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/define"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netinittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/metallbenv"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
)

// test cases variables that are accessible across entire file.
var (
	ipv4metalLbIPList []string
	ipv4NodeAddrList  []string
)

var _ = Describe("BFD", Ordered, Label(tsparams.LabelBFDTestCases), ContinueOnFailure, func() {

	BeforeAll(func() {

		By("Verify MetalLb DaemonSet is not running on workers")
		err := metallbenv.CreateNewMetalLbDaemonSetAndWaitUntilItsRunning(180 * time.Second)
		Expect(err).ToNot(HaveOccurred(), "Failed to recreate metalLb daemonset")

		By("Getting MetalLb load balancer ip addresses")
		ipv4metalLbIPList, _, err = metallbenv.GetMetalLbIPByIPStack()
		Expect(err).ToNot(HaveOccurred(), "An unexpected error occurred while "+
			"determining the IP addresses from the ECO_CNF_CORE_NET_MLB_ADDR_LIST environment variable.")

		if len(ipv4metalLbIPList) < 2 {
			Skip("MetalLb BFD tests require 2 ip addresses. Please check ECO_CNF_CORE_NET_MLB_ADDR_LIST env var")
		}

		By("Getting external nodes ip addresses")
		workerNodeList := nodes.NewBuilder(APIClient, NetConfig.WorkerLabelMap)
		Expect(workerNodeList.Discover()).ToNot(HaveOccurred(), "Failed to discover worker nodes")
		ipv4NodeAddrList, err = workerNodeList.ExternalIPv4Networks()
		Expect(err).ToNot(HaveOccurred(), "Failed to collect external nodes ip addresses")

		err = metallbenv.IsEnvVarMetalLbIPinNodeExtNetRange(ipv4NodeAddrList, ipv4metalLbIPList, nil)
		Expect(err).ToNot(HaveOccurred(), "Failed to validate metalLb exported ip address")

		By("Creating external BR-EX NetworkAttachmentDefinition")
		macVlanPlugin, err := define.MasterNadPlugin(coreparams.OvnExternalBridge, "bridge", nad.IPAMStatic())
		Expect(err).ToNot(HaveOccurred(), "Failed to define master nad plugin")
		externalNad := nad.NewBuilder(APIClient, tsparams.ExternalMacVlanNADName, tsparams.TestNamespaceName)
		_, err = externalNad.WithMasterPlugin(macVlanPlugin).Create()
		Expect(err).ToNot(HaveOccurred(), "Failed to create external NetworkAttachmentDefinition")
		Expect(externalNad.Exists()).To(BeTrue(), "Failed to detect external NetworkAttachmentDefinition")
	})

	BeforeEach(func() {
	})

	Context("multi hops", Label("mutihop"), func() {

		It("should provide fast link failure detection", polarion.ID("47186"), func() {

		})

	})

	Context("single hop", Label("singlehop"), func() {

		It("provides Prometheus BFD metrics", polarion.ID("47187"), func() {

		})

		It("basic functionality should provide fast link failure detection", polarion.ID("47188"), func() {

		})

	})

	AfterEach(func() {

	})

	AfterAll(func() {
		By("Clean test namespace")
		err := namespace.NewBuilder(APIClient, tsparams.TestNamespaceName).Clean(180)
		Expect(err).ToNot(HaveOccurred(), "error to clean test namespace")

	})
})
