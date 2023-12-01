package tests

import (
	"fmt"
	"strings"
	"time"

	multinetpolicyapiv1 "github.com/k8snetworkplumbingwg/multi-networkpolicy/pkg/apis/k8s.cni.cncf.io/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/daemonset"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/networkpolicy"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-goinfra/pkg/sriov"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netenv"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netinittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/policy/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/internal/cluster"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

var (
	serverPodIP            = "192.168.0.1/24"
	firstClientPodIP       = "192.168.0.2/24"
	secondClientPodIP      = "192.168.0.3/24"
	labelServerPod         = "pod1"
	multiNetworkPolicyName = "verificationpolicy"
	port5001               = 5001
	port5003               = 5003
)

var _ = Describe("SRIOV", Ordered, Label("multinetworkpolicy"), ContinueOnFailure, func() {

	var (
		workerNodeList                             []*nodes.Builder
		serverPod, firstClientPod, secondClientPod *pod.Builder
		srIovNet                                   *sriov.NetworkBuilder
	)

	BeforeAll(func() {
		By("Enabling MultiNetworkPolicy feature")
		enableMultiNetworkPolicy(true)

		By("Listing worker nodes")
		var err error
		workerNodeList, err = nodes.List(
			APIClient, metaV1.ListOptions{LabelSelector: labels.Set(NetConfig.WorkerLabelMap).String()})
		Expect(err).ToNot(HaveOccurred(), "Failed to list worker nodes")
		Expect(len(workerNodeList)).To(BeNumerically(">", 1),
			"Failed cluster doesn't have enough nodes")

		By("Collecting SR-IOV interface for multinetwork policy tests")
		srIovInterfacesUnderTest, err := NetConfig.GetSriovInterfaces(1)
		Expect(err).ToNot(HaveOccurred(), "Failed to retrieve SR-IOV interfaces for testing")

		By("Configuring SR-IOV policy")
		sriovPolicy, err := sriov.NewPolicyBuilder(
			APIClient,
			"policysriov",
			NetConfig.SriovOperatorNamespace,
			"sriovpolicy",
			6,
			srIovInterfacesUnderTest,
			NetConfig.WorkerLabelMap).Create()
		Expect(err).ToNot(HaveOccurred(), "Failed to create SR-IOV policy")

		By("Creating sr-iov network with ipam static config")
		srIovNet, err = sriov.NewNetworkBuilder(
			APIClient,
			"sriovnetpolicy",
			NetConfig.SriovOperatorNamespace,
			tsparams.TestNamespaceName,
			sriovPolicy.Definition.Spec.ResourceName).
			WithStaticIpam().Create()
		Expect(err).ToNot(HaveOccurred(), "Failed to configure sr-iov network")

		By("Waiting until cluster MCP and SR-IOV are stable")
		err = netenv.WaitForSriovAndMCPStable(
			APIClient, tsparams.MCOWaitTimeout, time.Minute, NetConfig.CnfMcpLabel, NetConfig.SriovOperatorNamespace)
		Expect(err).ToNot(HaveOccurred(), "Failed cluster is not stable")
	})

	BeforeEach(func() {
		By("Creating first client pod")
		firstClientPod = createClientPod(
			"client1", srIovNet.Definition.Name, workerNodeList[1].Definition.Name, firstClientPodIP, "pod2")

		By("Creating second client pod")
		secondClientPod = createClientPod(
			"client2", srIovNet.Definition.Name, workerNodeList[1].Definition.Name, secondClientPodIP, "pod3")

		By("Creating server pod")
		serverPod = createServerPod(srIovNet.Definition.Name, workerNodeList[0].Definition.Name)
	})

	It("Ingress Default rule without PolicyType deny all", polarion.ID("53901"), func() {
		_, err := networkpolicy.NewMultiNetworkPolicyBuilder(
			APIClient, multiNetworkPolicyName, tsparams.TestNamespaceName).
			WithNetwork(srIovNet.Definition.Name).
			WithEmptyIngress().
			WithPodSelector(metaV1.LabelSelector{MatchLabels: map[string]string{"pod": labelServerPod}}).
			Create()
		Expect(err).ToNot(HaveOccurred(), "Failed to create multiNetworkPolicy")

		By("Traffic verification")
		// All traffic should be blocked to the serverPod
		err = runTCPTraffic(firstClientPod, removePrefixFromIP(serverPodIP), port5001)
		Expect(err).Should(HaveOccurred(), fmt.Sprintf("Unexpectedly pod %s can reach %s with port %d",
			firstClientPod.Definition.Name, serverPod.Definition.Name, port5001))

		err = runTCPTraffic(secondClientPod, removePrefixFromIP(serverPodIP), port5003)
		Expect(err).Should(HaveOccurred(), fmt.Sprintf("Unexpectedly pod %s can reach %s with port %d",
			secondClientPod.Definition.Name, serverPod.Definition.Name, port5003))

		// Traffic between firstClientPod and secondClientPod should not be affected (not blocked)
		err = runTCPTraffic(secondClientPod, removePrefixFromIP(firstClientPodIP), port5001)
		Expect(err).ShouldNot(HaveOccurred(), fmt.Sprintf("Pod %s can NOT reach %s with port %d",
			secondClientPod.Definition.Name, firstClientPod.Definition.Name, port5001))
	})

	// 53899
	// The test fails due to OCPBUGS-974
	It("Ingress Default rule without PolicyType allow all", polarion.ID("53899"), func() {
		By("Apply MultiNetworkPolicy with ingress rule allow all without PolicyType field")

		_, err := networkpolicy.NewMultiNetworkPolicyBuilder(
			APIClient, multiNetworkPolicyName, tsparams.TestNamespaceName).
			WithNetwork(srIovNet.Definition.Name).
			WithIngressRule(multinetpolicyapiv1.MultiNetworkPolicyIngressRule{}).
			WithPodSelector(metaV1.LabelSelector{MatchLabels: map[string]string{}}).Create()
		Expect(err).ToNot(HaveOccurred(), "Failed to create multiNetworkPolicy")

		// All traffic is accepted
		By("Traffic verification")
		err = runTCPTraffic(firstClientPod, removePrefixFromIP(serverPodIP), port5001)
		Expect(err).ShouldNot(HaveOccurred(), fmt.Sprintf("Unexpectedly pod %s can NOT reach %s with port %d",
			firstClientPod.Definition.Name, serverPod.Definition.Name, port5001))

		err = runTCPTraffic(secondClientPod, removePrefixFromIP(serverPodIP), port5001)
		Expect(err).ShouldNot(HaveOccurred(), fmt.Sprintf("Unexpectedly pod %s can NOT reach %s with port %d",
			secondClientPod.Definition.Name, serverPod.Definition.Name, port5001))

		// Traffic between firstClientPod and secondClientPod should not be affected (not blocked)
		err = runTCPTraffic(firstClientPod, removePrefixFromIP(secondClientPodIP), port5001)
		Expect(err).ShouldNot(HaveOccurred(), fmt.Sprintf("pod %s can NOT reach %s with port %d",
			secondClientPod.Definition.Name, firstClientPod.Definition.Name, port5001))
	})

	AfterEach(func() {
		testNameSpace, err := namespace.Pull(APIClient, tsparams.TestNamespaceName)
		Expect(err).ToNot(HaveOccurred(), "Failed to pull namespace")
		err = testNameSpace.CleanObjects(
			10*time.Minute,
			networkpolicy.GetMultiNetworkGVR())
		Expect(err).ToNot(HaveOccurred(), "Failed to remove multiNetworkPolicy object from namespace")

		// All traffic is accepted
		err = runTCPTraffic(firstClientPod, removePrefixFromIP(serverPodIP), port5001)
		Expect(err).ShouldNot(HaveOccurred(), fmt.Sprintf("pod %s can NOT reach %s with port %d",
			firstClientPod.Definition.Name, serverPod.Definition.Name, port5001))

		err = testNameSpace.CleanObjects(
			10*time.Minute,
			pod.GetGVR())
		Expect(err).ToNot(HaveOccurred(), "Failed to remove pod objects from namespace")
	})

	AfterAll(func() {
		By("Removing all SR-IOV Policies")
		err := sriov.CleanAllNetworkNodePolicies(APIClient, NetConfig.SriovOperatorNamespace, metaV1.ListOptions{})
		Expect(err).ToNot(HaveOccurred(), "Fail to clean srIovPolicy")

		By("Removing all srIovNetworks")
		err = sriov.CleanAllNetworksByTargetNamespace(
			APIClient, NetConfig.SriovOperatorNamespace, tsparams.TestNamespaceName, metaV1.ListOptions{})
		Expect(err).ToNot(HaveOccurred(), "Fail to clean sriov networks")

		By("Waiting until cluster MCP and SR-IOV are stable")
		err = netenv.WaitForSriovAndMCPStable(
			APIClient, tsparams.MCOWaitTimeout, time.Minute, NetConfig.CnfMcpLabel, NetConfig.SriovOperatorNamespace)
		Expect(err).ToNot(HaveOccurred(), "Fail to wait until cluster is stable")
	})
})

// runTCPTraffic sends TCP traffic from clientPod to serverIP.
func runTCPTraffic(clientPod *pod.Builder, serverIP string, port int) error {
	buffer, err := clientPod.ExecCommand(
		[]string{"testcmd", fmt.Sprintf("-port=%d", port), "-interface=net1",
			fmt.Sprintf("-server=%s", serverIP), "-protocol=tcp", "-mtu=1200"},
	)

	if err != nil {
		return fmt.Errorf("%w: %s", err, buffer.String())
	}

	return nil
}

func enableMultiNetworkPolicy(status bool) {
	By(fmt.Sprintf("Configuring MultiNetworkPolicy mode %v", status))

	clusterNetwork, err := cluster.GetOCPNetworkOperatorConfig(APIClient)
	Expect(err).ToNot(HaveOccurred(), "Failed to collect network.operator object")

	clusterNetwork, err = clusterNetwork.SetMultiNetworkPolicy(status, 20*time.Minute)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to set MultiNetworkPolicy mode %v", status))

	network, err := clusterNetwork.Get()
	Expect(err).ToNot(HaveOccurred(), "Failed to collect network.operator object")
	Expect(network.Spec.UseMultiNetworkPolicy).To(BeEquivalentTo(&status),
		"Failed network.operator UseMultiNetworkPolicy flag is not in expected state")

	Eventually(func() error {
		multusDs, err := daemonset.Pull(APIClient, tsparams.MultiNetworkPolicyDSName, NetConfig.MultusNamesapce)

		if err != nil {
			return err
		}

		if multusDs.IsReady(10 * time.Second) {
			return nil
		}

		return fmt.Errorf("DS is not ready")
	}, tsparams.WaitTimeout, tsparams.RetryInterval).ShouldNot(HaveOccurred(),
		"Failed MultiNetworkPolicy daemonSet is not ready")
}

func createClientPod(podName, srIovNetwork, nodeName, ipaddress, label string) *pod.Builder {
	podCmd := []string{"bash", "-c",
		fmt.Sprintf("testcmd --listen -interface=net1 --protocol=tcp --mtu=1500 -port=%d",
			port5001)}

	staticAnnotation := pod.StaticIPAnnotation(srIovNetwork, []string{ipaddress})
	clientPod, err := pod.NewBuilder(APIClient, podName, tsparams.TestNamespaceName, NetConfig.CnfNetTestContainer).
		WithSecondaryNetwork(staticAnnotation).
		DefineOnNode(nodeName).WithLabel("pod", label).
		RedefineDefaultCMD(podCmd).
		WithPrivilegedFlag().
		CreateAndWaitUntilRunning(tsparams.WaitTimeout)

	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to define pod annotation for clientPod with IPAddress: %s", ipaddress))

	return clientPod
}

func createServerPod(sriovNetworkName, nodeName string) *pod.Builder {
	baseContainer, err := pod.NewContainerBuilder(
		"container1", NetConfig.CnfNetTestContainer, setTestCmdTCPServer(port5003)).GetContainerCfg()
	Expect(err).ToNot(HaveOccurred(), "Failed to define base container")

	additionalContainer, err := pod.NewContainerBuilder(
		"container2", NetConfig.CnfNetTestContainer, setTestCmdTCPServer(port5001)).GetContainerCfg()
	Expect(err).ToNot(HaveOccurred(), "Failed to define additional container")

	initCommand := []string{"bash", "-c",
		fmt.Sprintf("ping %s -c 3 -w 90 && ping %s -c 3 -w 90",
			removePrefixFromIP(firstClientPodIP), removePrefixFromIP(secondClientPodIP))}

	InitContainer, err := pod.NewContainerBuilder(
		"init", NetConfig.CnfNetTestContainer, initCommand).GetContainerCfg()
	Expect(err).ToNot(HaveOccurred(), "Failed to define init container")

	serverPod, err := pod.NewBuilder(
		APIClient, "server", tsparams.TestNamespaceName, NetConfig.CnfNetTestContainer).
		RedefineDefaultContainer(*baseContainer).
		WithAdditionalContainer(additionalContainer).
		WithAdditionalInitContainer(InitContainer).
		DefineOnNode(nodeName).
		WithPrivilegedFlag().WithLabel("pod", labelServerPod).
		WithSecondaryNetwork(pod.StaticIPAnnotation(sriovNetworkName, []string{serverPodIP})).
		CreateAndWaitUntilRunning(tsparams.WaitTimeout)
	Expect(err).ToNot(HaveOccurred(), "Failed to create server pod")

	return serverPod
}

func setTestCmdTCPServer(port int) []string {
	return []string{"bash", "-c",
		fmt.Sprintf("testcmd --listen -interface=net1 --protocol=tcp --mtu=1500 -port=%d", port)}
}

func removePrefixFromIP(ipAddr string) string {
	return strings.Split(ipAddr, "/")[0]
}
