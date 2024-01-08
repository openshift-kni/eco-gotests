package tests

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/nad"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-goinfra/pkg/sriov"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/cmd"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netenv"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netinittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netparam"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/sriov/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	srIovPolicyNodeOneName        = "sriov-policy-node-one"
	srIovPolicySamePFResName      = "sriovpolicysamepf"
	srIovNetworkAllMultiSamePF    = "sriovnet-allmulti-same"
	srIovNetworkDefaultSamePF     = "sriovnet-default-same"
	multicastServerName           = "mc-source-server"
	clientDefaultName             = "client-default"
	clientAllmultiEnabledName     = "client-allmulti-enabled"
	multicastServerIPv6           = "2001:100::20/64"
	clientAllmultiEnabledIPv6     = "2001:100::1/64"
	clientAllmultiDisabledIPv6    = "2001:100::2/64"
	multicastServerIPv6Mac        = "60:00:00:00:10:10"
	clientAllmultiEnabledIPv6Mac  = "60:00:00:00:00:11"
	clientAllmultiDisabledIPv6Mac = "60:00:00:00:00:12"
	multicastIPv6GroupIP          = "ff05:5::5:"
)

var (
	workerNodes          []*nodes.Builder
	multicastPingIPv6CMD = []string{"bash", "-c", "sleep 5; ping -I net1 ff05:5::05"}
	tcpDumpCMD           = []string{"bash", "-c", "tcpdump -i net1 -c 10"}
	addIPv6MCGroupMacCMD = []string{"bash", "-c", "ip maddr add 33:33:0:0:0:5 dev net1"}
)

var _ = Describe("allmulti", Ordered, Label(tsparams.LabelSuite), ContinueOnFailure, func() {

	BeforeAll(func() {
		By("Discover worker nodes")
		var err error

		workerNodes, err = nodes.List(APIClient,
			metaV1.ListOptions{LabelSelector: labels.Set(NetConfig.WorkerLabelMap).String()})
		Expect(err).ToNot(HaveOccurred(), "Fail to discover nodes")

		By("Collecting SR-IOV interfaces for allmulti testing")
		srIovInterfacesUnderTest, err := NetConfig.GetSriovInterfaces(1)
		Expect(err).ToNot(HaveOccurred(), "Failed to retrieve SR-IOV interfaces for testing")

		By(fmt.Sprintf("Define and create sriov network policy on %s", workerNodes[0].Definition.Name))
		nodeSelectorWorker0 := map[string]string{
			"kubernetes.io/hostname": workerNodes[0].Definition.Name,
		}

		_, err = sriov.NewPolicyBuilder(
			APIClient,
			srIovPolicyNodeOneName,
			NetConfig.SriovOperatorNamespace,
			srIovPolicySamePFResName,
			10,
			[]string{fmt.Sprintf("%s#0-9", srIovInterfacesUnderTest[0])},
			nodeSelectorWorker0).WithMTU(9000).WithVhostNet(true).Create()
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to create an sriov policy on %s",
			workerNodes[0].Definition.Name))

		By("Define and create sriov network with allmuti enabled")
		defineAndCreateSrIovNetwork(srIovNetworkAllMultiSamePF, srIovPolicySamePFResName, true)

		By("Define and create sriov network with allmuti disabled")
		defineAndCreateSrIovNetwork(srIovNetworkDefaultSamePF, srIovPolicySamePFResName, false)

		By("Waiting until cluster MCP and SR-IOV are stable")
		err = netenv.WaitForSriovAndMCPStable(
			APIClient, tsparams.MCOWaitTimeout, time.Minute, NetConfig.CnfMcpLabel, NetConfig.SriovOperatorNamespace)
		Expect(err).ToNot(HaveOccurred(), "fail cluster is not stable")
	})

	It("Validate a pod can receive non-member multicast IPv6 traffic over a secondary SRIOV interface"+
		" when allmulti mode is enabled from a multicast source in the same PF", polarion.ID("67813"), func() {

		By("Define and run a multicast server")
		serverNetAnnotation := pod.StaticIPAnnotationWithMacAddress(srIovNetworkDefaultSamePF, []string{multicastServerIPv6},
			multicastServerIPv6Mac)
		multicastSourceServerPod, err := pod.NewBuilder(APIClient, multicastServerName,
			tsparams.TestNamespaceName, NetConfig.CnfNetTestContainer).DefineOnNode(workerNodes[0].Definition.Name).
			WithPrivilegedFlag().RedefineDefaultCMD(multicastPingIPv6CMD).WithSecondaryNetwork(serverNetAnnotation).
			CreateAndWaitUntilRunning(netparam.DefaultTimeout)
		Expect(err).ToNot(HaveOccurred(), "fail to define and run multicast source server")

		By("Define and run a client pod with allmulti disabled")
		defaultClientNetAnnotation := pod.StaticIPAnnotationWithMacAddress(srIovNetworkDefaultSamePF,
			[]string{clientAllmultiDisabledIPv6}, clientAllmultiDisabledIPv6Mac)

		defaultClientPod, err := pod.NewBuilder(APIClient, clientDefaultName, tsparams.TestNamespaceName,
			NetConfig.CnfNetTestContainer).DefineOnNode(workerNodes[0].Definition.Name).WithPrivilegedFlag().
			WithSecondaryNetwork(defaultClientNetAnnotation).CreateAndWaitUntilRunning(netparam.DefaultTimeout)
		Expect(err).ToNot(HaveOccurred(), "fail to define and run default client")

		By("Define and run a client pod with allmulti enabled")
		allMultiClientNetAnnotation := pod.StaticIPAnnotationWithMacAddress(srIovNetworkAllMultiSamePF,
			[]string{clientAllmultiEnabledIPv6}, clientAllmultiEnabledIPv6Mac)

		clientPodAllMultiEnabled, err := pod.NewBuilder(APIClient, clientAllmultiEnabledName, tsparams.TestNamespaceName,
			NetConfig.CnfNetTestContainer).DefineOnNode(workerNodes[0].Definition.Name).WithPrivilegedFlag().
			WithSecondaryNetwork(allMultiClientNetAnnotation).CreateAndWaitUntilRunning(netparam.DefaultTimeout)
		Expect(err).ToNot(HaveOccurred(), "fail to define and run allmulti enabled client")

		By("Verify connectivity between the clients and multicast source")
		err = cmd.ICMPConnectivityCheck(multicastSourceServerPod, []string{clientAllmultiEnabledIPv6,
			clientAllmultiDisabledIPv6})
		Expect(err).ToNot(HaveOccurred(), "Fail to ping between the multicast source and the clients")

		By("Verify multicast group is not accessible from container without allmulti enabled")

		Consistently(func() string {
			output, err := defaultClientPod.ExecCommand(tcpDumpCMD)
			if err != nil {
				glog.V(100).Info(err)
			}

			return output.String()

		}, 5*time.Second, 1*time.Second).ShouldNot(MatchRegexp(multicastIPv6GroupIP))

		By("Verify multicast group is accessible from container with allmulti enabled")

		Eventually(func() string {
			output, err := clientPodAllMultiEnabled.ExecCommand(tcpDumpCMD)
			if err != nil {
				glog.V(100).Info(err)
			}

			return output.String()

		}, 5*time.Second, 1*time.Second).Should(MatchRegexp(multicastIPv6GroupIP))

		By("Add client without allmulti enabled to the multicast group")
		_, err = defaultClientPod.ExecCommand(addIPv6MCGroupMacCMD)
		Expect(err).ToNot(HaveOccurred(), "fail to add the multicast group mac address")

		By("Verify the client receives traffic from the multicast group after being added to the group")
		Eventually(func() string {
			output, err := defaultClientPod.ExecCommand(tcpDumpCMD)
			if err != nil {
				glog.V(100).Info(err)
			}

			return output.String()

		}, 5*time.Second, 1*time.Second).Should(MatchRegexp(multicastIPv6GroupIP))

	})

	AfterAll(func() {
		By("Removing all SR-IOV Policy")
		err := sriov.CleanAllNetworkNodePolicies(APIClient, NetConfig.SriovOperatorNamespace, metaV1.ListOptions{})
		Expect(err).ToNot(HaveOccurred(), "Fail to clean srIovPolicy")

		By("Removing all srIovNetworks")
		err = sriov.CleanAllNetworksByTargetNamespace(
			APIClient, NetConfig.SriovOperatorNamespace, tsparams.TestNamespaceName, metaV1.ListOptions{})
		Expect(err).ToNot(HaveOccurred(), "Fail to clean sriov networks")
	})
})

func defineAndCreateSrIovNetwork(srIovNetwork, resName string, allMulti bool) {
	srIovNetworkObject := sriov.NewNetworkBuilder(
		APIClient, srIovNetwork, NetConfig.SriovOperatorNamespace, tsparams.TestNamespaceName, resName).
		WithStaticIpam().WithIPAddressSupport().WithMacAddressSupport()

	if allMulti {
		srIovNetworkObject.WithTrustFlag(true).WithMetaPluginAllMultiFlag(true)
	}

	srIovNetworkObject, err := srIovNetworkObject.Create()
	Expect(err).ToNot(HaveOccurred(), "Fail to create sriov network")

	Eventually(func() bool {
		_, err := nad.Pull(APIClient, srIovNetworkObject.Object.Name, tsparams.TestNamespaceName)

		return err == nil
	}, tsparams.WaitTimeout, tsparams.RetryInterval).Should(BeTrue(), "Fail to pull "+
		"NetworkAttachmentDefinition")
}
