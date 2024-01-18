package tests

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/nad"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
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
	srIovPolicyNode1Name          = "sriov-policy-node-1"
	srIovPolicyNode2Name          = "sriov-policy-node-2"
	srIovPolicyNode0PFResName     = "sriovpolicynode0pf"
	srIovPolicyNode1PFResName     = "sriovpolicynode1pf"
	srIovNetworkAllMultiNode1PF   = "sriovnet-allmulti-node1"
	srIovNetworkDefaultNode1PF    = "sriovnet-default-node1"
	srIovNetworkDefaultNode2      = "sriovnet-default-node2"
	multicastServerName           = "mc-source-server"
	clientDefaultName             = "client-default"
	clientAllmultiEnabledName     = "client-allmulti-enabled"
	multicastServerIPv6           = "2001:100::20/64"
	multicastServerIPv4           = "192.168.100.20/24"
	clientAllmultiEnabledIPv6     = "2001:100::1/64"
	clientAllmultiEnabledIPv4     = "192.168.100.1/24"
	clientAllmultiDisabledIPv6    = "2001:100::2/64"
	clientAllmultiDisabledIPv4    = "192.168.100.2/24"
	multicastServerIPv6Mac        = "60:00:00:00:10:10"
	multicastServerIPv4Mac        = "20:04:0f:f1:88:20"
	clientAllmultiEnabledIPv6Mac  = "60:00:00:00:00:11"
	clientAllmultiEnabledIPv4Mac  = "20:04:0f:f1:88:11"
	clientAllmultiDisabledIPv6Mac = "60:00:00:00:00:12"
	clientAllmultiDisabledIPv4Mac = "20:04:0f:f1:88:12"
	multicastIPv6GroupIP          = "ff05:5::5:"
	multicastIPv4GroupIP          = "239.100.100.250:"
)

var (
	workerNodes          []*nodes.Builder
	multicastPingIPv6CMD = []string{"bash", "-c", "sleep 5; ping -I net1 ff05:5::05"}
	multicastPingIPv4CMD = []string{"bash", "-c", "sleep 5; ping -I net1 239.100.100.250"}
	tcpDumpCMD           = []string{"bash", "-c", "tcpdump -i net1 -c 10"}
	addIPv6MCGroupMacCMD = []string{"bash", "-c", "ip maddr add 33:33:0:0:0:5 dev net1"}
	addIPv4MCGroupMacCMD = []string{"bash", "-c", "ip maddr add 01:00:5e:64:64:fa dev net1"}
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
			srIovPolicyNode1Name,
			NetConfig.SriovOperatorNamespace,
			srIovPolicyNode0PFResName,
			6,
			[]string{fmt.Sprintf("%s#0-5", srIovInterfacesUnderTest[0])},
			nodeSelectorWorker0).WithMTU(9000).WithVhostNet(true).Create()
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to create an sriov policy on %s",
			workerNodes[0].Definition.Name))

		By(fmt.Sprintf("Define and create sriov network policy on %s", workerNodes[1].Definition.Name))
		nodeSelectorWorker1 := map[string]string{
			"kubernetes.io/hostname": workerNodes[1].Definition.Name,
		}

		_, err = sriov.NewPolicyBuilder(
			APIClient,
			srIovPolicyNode2Name,
			NetConfig.SriovOperatorNamespace,
			srIovPolicyNode1PFResName,
			6,
			[]string{fmt.Sprintf("%s#0-5", srIovInterfacesUnderTest[0])},
			nodeSelectorWorker1).WithMTU(9000).WithVhostNet(true).Create()
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to create an sriov policy on %s",
			workerNodes[1].Definition.Name))

		By("Define and create sriov network with allmuti enabled")
		defineAndCreateSrIovNetwork(srIovNetworkAllMultiNode1PF, srIovPolicyNode0PFResName, true)

		By("Define and create sriov network with allmuti disabled")
		defineAndCreateSrIovNetwork(srIovNetworkDefaultNode1PF, srIovPolicyNode0PFResName, false)

		By("Define and create sriov network with allmuti disabled on a different node")
		defineAndCreateSrIovNetwork(srIovNetworkDefaultNode2, srIovPolicyNode1PFResName, false)

		By("Waiting until cluster MCP and SR-IOV are stable")
		err = netenv.WaitForSriovAndMCPStable(
			APIClient, tsparams.MCOWaitTimeout, time.Minute, NetConfig.CnfMcpLabel, NetConfig.SriovOperatorNamespace)
		Expect(err).ToNot(HaveOccurred(), "fail cluster is not stable")
	})

	It("Validate a pod can receive non-member multicast IPv6 traffic over a secondary SRIOV interface"+
		" when allmulti mode is enabled from a multicast source in the same PF", polarion.ID("67813"), func() {
		multicastIPv6Address := []string{multicastServerIPv6}
		multicastServer := createMulticastServer(multicastServerName, srIovNetworkDefaultNode1PF, multicastServerIPv6Mac,
			multicastIPv6Address, multicastPingIPv6CMD, workerNodes[0].Definition.Name)

		defaultClient := createDefaultClient(clientDefaultName, srIovNetworkDefaultNode1PF,
			clientAllmultiDisabledIPv6Mac, workerNodes[0].Definition.Name, []string{clientAllmultiDisabledIPv6})

		allMultiEnabledClient := createAllMultiClient(clientAllmultiEnabledName, srIovNetworkAllMultiNode1PF,
			clientAllmultiEnabledIPv6Mac, workerNodes[0].Definition.Name, []string{clientAllmultiEnabledIPv6})

		runAllMultiTestCases(multicastServer, defaultClient, allMultiEnabledClient, multicastServerIPv6,
			clientAllmultiDisabledIPv6, multicastIPv6GroupIP, addIPv6MCGroupMacCMD)
	})

	It("Validate a pod can receive non-member multicast IPv4 traffic over a secondary SRIOV interface"+
		" when allmulti mode is enabled from a multicast source is on a different node",
		polarion.ID("67813"), func() {
			multicastServer := createMulticastServer(multicastServerName, srIovNetworkDefaultNode2,
				multicastServerIPv4Mac, []string{multicastServerIPv4}, multicastPingIPv4CMD,
				workerNodes[1].Definition.Name)

			defaultClient := createDefaultClient(clientDefaultName, srIovNetworkDefaultNode1PF,
				clientAllmultiDisabledIPv4Mac, workerNodes[0].Definition.Name, []string{clientAllmultiDisabledIPv4})

			allMultiEnabledClient := createAllMultiClient(clientAllmultiEnabledName, srIovNetworkAllMultiNode1PF,
				clientAllmultiEnabledIPv4Mac, workerNodes[0].Definition.Name, []string{clientAllmultiEnabledIPv4})

			runAllMultiTestCases(multicastServer, defaultClient, allMultiEnabledClient, clientAllmultiEnabledIPv4,
				clientAllmultiDisabledIPv4, multicastIPv4GroupIP, addIPv4MCGroupMacCMD)
		})

	AfterEach(func() {
		By("Removing all pods from test namespace")
		runningNamespace, err := namespace.Pull(APIClient, tsparams.TestNamespaceName)
		Expect(err).ToNot(HaveOccurred(), "Failed to pull namespace")
		Expect(runningNamespace.CleanObjects(
			tsparams.WaitTimeout, pod.GetGVR())).ToNot(HaveOccurred())

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

func createMulticastServer(
	name string,
	sriovNetwork string,
	macAddress string,
	ipAddress []string,
	multicastCmd []string,
	nodeName string) *pod.Builder {
	By("Define and run a multicast server")

	sriovNetworkMC := pod.StaticIPAnnotationWithMacAddress(sriovNetwork, ipAddress, macAddress)
	multicastSourceClient, err := pod.NewBuilder(APIClient, name, tsparams.TestNamespaceName,
		NetConfig.CnfNetTestContainer).DefineOnNode(nodeName).
		WithPrivilegedFlag().RedefineDefaultCMD(multicastCmd).WithSecondaryNetwork(sriovNetworkMC).
		CreateAndWaitUntilRunning(netparam.DefaultTimeout)
	Expect(err).ToNot(HaveOccurred(), "Failed to define and run multicast source server")

	return multicastSourceClient
}

func createDefaultClient(
	name string,
	sriovNetwork string,
	macAddress string,
	nodeName string,
	ipAddress []string) *pod.Builder {
	By("Define and run a client pod with allmulti disabled")

	sriovNetworkDefault := pod.StaticIPAnnotationWithMacAddress(sriovNetwork, ipAddress, macAddress)

	clientDefault, err := pod.NewBuilder(APIClient, name, tsparams.TestNamespaceName,
		NetConfig.CnfNetTestContainer).DefineOnNode(nodeName).WithPrivilegedFlag().
		WithSecondaryNetwork(sriovNetworkDefault).CreateAndWaitUntilRunning(netparam.DefaultTimeout)
	Expect(err).ToNot(HaveOccurred(), "Failed to define and run default client")

	return clientDefault
}

func createAllMultiClient(
	name string,
	sriovNetwork string,
	macAddress string,
	nodeName string,
	ipAddress []string) *pod.Builder {
	By("Define and run a client pod with allmulti enabled")

	sriovNetworkEnabled := pod.StaticIPAnnotationWithMacAddress(sriovNetwork, ipAddress, macAddress)

	clientAllMultiEnabled, err := pod.NewBuilder(APIClient, name, tsparams.TestNamespaceName,
		NetConfig.CnfNetTestContainer).DefineOnNode(nodeName).WithPrivilegedFlag().
		WithSecondaryNetwork(sriovNetworkEnabled).CreateAndWaitUntilRunning(netparam.DefaultTimeout)
	Expect(err).ToNot(HaveOccurred(), "Failed to define and run default client")

	return clientAllMultiEnabled
}

func runAllMultiTestCases(
	multicastSourcePod *pod.Builder,
	defaultClientPod *pod.Builder,
	allMultiEnabledPod *pod.Builder,
	allMulticastEnabledIP string,
	defaultClientIP string,
	multicastGroupIP string,
	addIPMCGroupMacCMD []string) {
	By("Verify connectivity between the clients and multicast source")

	clientIPAddresses := []string{allMulticastEnabledIP, defaultClientIP}
	err := cmd.ICMPConnectivityCheck(multicastSourcePod, clientIPAddresses)
	Expect(err).ToNot(HaveOccurred(), "Fail to ping between the multicast source and the clients")

	By("Verify multicast group is not accessible from container without allmulti enabled")
	Consistently(func() string {
		output, err := defaultClientPod.ExecCommand(tcpDumpCMD)
		if err != nil {
			glog.V(100).Info(err)
		}

		return output.String()

	}, 5*time.Second, 1*time.Second).ShouldNot(MatchRegexp(multicastGroupIP))

	By("Verify multicast group is accessible from container with allmulti enabled")

	Eventually(func() string {
		output, err := allMultiEnabledPod.ExecCommand(tcpDumpCMD)
		if err != nil {
			glog.V(100).Info(err)
		}

		return output.String()

	}, 5*time.Second, 1*time.Second).Should(MatchRegexp(multicastGroupIP))

	By("Add client without allmulti enabled to the multicast group")

	_, err = defaultClientPod.ExecCommand(addIPMCGroupMacCMD)
	Expect(err).ToNot(HaveOccurred(), "fail to add the multicast group mac address")

	By("Verify the client receives traffic from the multicast group after being added to the group")
	Eventually(func() string {
		output, err := defaultClientPod.ExecCommand(tcpDumpCMD)
		if err != nil {
			glog.V(100).Info(err)
		}

		return output.String()

	}, 5*time.Second, 1*time.Second).Should(MatchRegexp(multicastGroupIP))
}
