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
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/ipaddr"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netenv"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netinittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/policy/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/internal/cluster"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

var (
	tcpProtocol            = "tcp"
	sctpProtocol           = "sctp"
	serverPodIP            = "192.168.0.1/24"
	serverPodIPv6          = "2001:1db8:85a3::1/64"
	firstClientPodIP       = "192.168.0.2/24"
	firstClientPodIPv6     = "2001:1db8:85a3::2/64"
	secondClientPodIP      = "192.168.0.3/24"
	secondClientPodIPv6    = "2001:1db8:85a3::3/64"
	labelServerPod         = "pod1"
	labelFirstClientPod    = "pod2"
	labelSecondClientPod   = "pod3"
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
	Context("ipv4", func() {
		BeforeEach(func() {
			By("Creating first client pod")
			firstClientPod = createClientPod(
				"client1",
				srIovNet.Definition.Name,
				workerNodeList[1].Definition.Name,
				tcpProtocol,
				firstClientPodIP,
				labelFirstClientPod)

			By("Creating second client pod")
			secondClientPod = createClientPod(
				"client2",
				srIovNet.Definition.Name,
				workerNodeList[1].Definition.Name,
				tcpProtocol,
				secondClientPodIP,
				labelSecondClientPod)

			By("Creating server pod")
			serverPod = createServerPod(
				srIovNet.Definition.Name,
				workerNodeList[0].Definition.Name,
				serverPodIP,
				firstClientPodIP,
				secondClientPodIP,
				tcpProtocol)
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
			err = runTraffic(firstClientPod, removePrefixFromIP(serverPodIP), tcpProtocol, port5001)
			Expect(err).Should(HaveOccurred(), fmt.Sprintf("Unexpectedly pod %s can reach %s with port %d",
				firstClientPod.Definition.Name, serverPod.Definition.Name, port5001))

			err = runTraffic(secondClientPod, removePrefixFromIP(serverPodIP), tcpProtocol, port5003)
			Expect(err).Should(HaveOccurred(), fmt.Sprintf("Unexpectedly pod %s can reach %s with port %d",
				secondClientPod.Definition.Name, serverPod.Definition.Name, port5003))

			// Traffic between firstClientPod and secondClientPod should not be affected (not blocked)
			err = runTraffic(secondClientPod, removePrefixFromIP(firstClientPodIP), tcpProtocol, port5001)
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
			err = runTraffic(firstClientPod, removePrefixFromIP(serverPodIP), tcpProtocol, port5001)
			Expect(err).ShouldNot(HaveOccurred(), fmt.Sprintf("Unexpectedly pod %s can NOT reach %s with port %d",
				firstClientPod.Definition.Name, serverPod.Definition.Name, port5001))

			err = runTraffic(secondClientPod, removePrefixFromIP(serverPodIP), tcpProtocol, port5001)
			Expect(err).ShouldNot(HaveOccurred(), fmt.Sprintf("Unexpectedly pod %s can NOT reach %s with port %d",
				secondClientPod.Definition.Name, serverPod.Definition.Name, port5001))

			// Traffic between firstClientPod and secondClientPod should not be affected (not blocked)
			err = runTraffic(firstClientPod, removePrefixFromIP(secondClientPodIP), tcpProtocol, port5001)
			Expect(err).ShouldNot(HaveOccurred(), fmt.Sprintf("pod %s can NOT reach %s with port %d",
				secondClientPod.Definition.Name, firstClientPod.Definition.Name, port5001))
		})

		// 53900
		It("Egress TCP endPort allow specific pod", polarion.ID("53900"), func() {
			By("Apply MultiNetworkPolicy with egress rule allow ports in range 5000-5002")

			// Update egress rule with port range and delete port 5001 when the bug OCPBUGS-975 is fixed
			//Port:     &policyPort5000,
			//EndPort:  &policyPort5002,
			egressRule, err := networkpolicy.NewEgressRuleBuilder().WithPortAndProtocol(uint16(port5001), "TCP").
				WithPeerPodSelector(metaV1.LabelSelector{MatchLabels: map[string]string{"pod": labelServerPod}}).
				GetEgressRuleCfg()
			Expect(err).ToNot(HaveOccurred(), "Failed to build egress rule")

			_, err = networkpolicy.NewMultiNetworkPolicyBuilder(
				APIClient, multiNetworkPolicyName, tsparams.TestNamespaceName).
				WithNetwork(srIovNet.Definition.Name).
				WithPodSelector(metaV1.LabelSelector{MatchLabels: map[string]string{"pod": labelFirstClientPod}}).
				WithPolicyType(multinetpolicyapiv1.PolicyTypeEgress).
				WithEgressRule(*egressRule).Create()

			Expect(err).ToNot(HaveOccurred(), "Failed to create multiNetworkPolicy")

			By("Traffic verification")
			// Traffic from firstClientPod to serverPod with port range 5000-5002 should pass.
			err = runTraffic(firstClientPod, ipaddr.RemovePrefix(serverPodIP), tcpProtocol, port5001)
			Expect(err).ShouldNot(HaveOccurred(), fmt.Sprintf("pod %s can NOT reach %s with port %d",
				firstClientPod.Definition.Name, serverPod.Definition.Name, port5001))

			// Port 5003 is out of the accepted port range. Traffic should be dropped.
			err = runTraffic(firstClientPod, ipaddr.RemovePrefix(serverPodIP), tcpProtocol, port5003)
			Expect(err).Should(HaveOccurred(), fmt.Sprintf("unexpectedly pod %s can reach %s with port %d",
				firstClientPod.Definition.Name, serverPod.Definition.Name, port5003))

			// Traffic between firstClientPod and secondClientPod is not allowed
			err = runTraffic(firstClientPod, ipaddr.RemovePrefix(secondClientPodIP), tcpProtocol, port5001)
			Expect(err).Should(HaveOccurred(), fmt.Sprintf("unexpectedly pod %s can reach %s with port %d",
				firstClientPod.Definition.Name, secondClientPod.Definition.Name, port5001))

			// Traffic between secondClientPod and serverPod is not affected by rule.
			err = runTraffic(secondClientPod, ipaddr.RemovePrefix(serverPodIP), tcpProtocol, port5001)
			Expect(err).ShouldNot(HaveOccurred(), fmt.Sprintf("pod %s can NOT reach %s with port %d",
				secondClientPod.Definition.Name, serverPod.Definition.Name, port5001))

			err = runTraffic(secondClientPod, ipaddr.RemovePrefix(serverPodIP), tcpProtocol, port5003)
			Expect(err).ShouldNot(HaveOccurred(), fmt.Sprintf("pod %s can NOT reach %s with port %d",
				secondClientPod.Definition.Name, serverPod.Definition.Name, port5003))
		})

		// 53898
		It("Ingress and Egress allow IPv4 address", polarion.ID("53898"), func() {
			By("Apply MultiNetworkPolicy with ingress and egress rules allow specific IPv4 addresses")
			egressRule, err := networkpolicy.NewEgressRuleBuilder().WithPeerPodSelectorAndCIDR(
				metaV1.LabelSelector{MatchLabels: map[string]string{"pod": labelSecondClientPod}},
				ipaddr.RemovePrefix(secondClientPodIP)+"/"+"32").
				GetEgressRuleCfg()
			Expect(err).ToNot(HaveOccurred(), "Failed to build egress rule")

			ingressRule, err := networkpolicy.NewIngressRuleBuilder().
				WithCIDR(ipaddr.RemovePrefix(firstClientPodIP) + "/" + "32").
				GetIngressRuleCfg()
			Expect(err).ToNot(HaveOccurred(), "Failed to build ingress rule")

			multiNetPolicy := networkpolicy.NewMultiNetworkPolicyBuilder(
				APIClient, multiNetworkPolicyName, tsparams.TestNamespaceName).
				WithNetwork(srIovNet.Definition.Name).
				WithPodSelector(metaV1.LabelSelector{MatchLabels: map[string]string{"pod": labelServerPod}}).
				WithPolicyType(multinetpolicyapiv1.PolicyTypeEgress).
				WithPolicyType(multinetpolicyapiv1.PolicyTypeIngress).
				WithIngressRule(*ingressRule).WithEgressRule(*egressRule)

			_, err = multiNetPolicy.Create()
			Expect(err).ToNot(HaveOccurred(), "Failed to create multiNetworkPolicy")

			By("Traffic verification")
			// Traffic from firstClientPod to serverPod with source IP netpolicyparameters.Pod2IPAddress should pass
			err = runTraffic(firstClientPod, ipaddr.RemovePrefix(serverPodIP), tcpProtocol, port5001)
			Expect(err).ShouldNot(HaveOccurred(), fmt.Sprintf("pod %s can NOT reach %s with port %d",
				firstClientPod.Definition.Name, serverPod.Definition.Name, port5001))

			// Traffic from serverPod to secondClientPod with destination IP netpolicyparameters.Pod3IPAddress should pass
			err = runTraffic(serverPod, ipaddr.RemovePrefix(secondClientPodIP), tcpProtocol, port5001)
			Expect(err).ShouldNot(HaveOccurred(), fmt.Sprintf("pod %s can NOT reach %s with port %d",
				serverPod.Definition.Name, secondClientPod.Definition.Name, port5001))

			// All other traffic should be dropped
			err = runTraffic(secondClientPod, ipaddr.RemovePrefix(serverPodIP), tcpProtocol, port5001)
			Expect(err).Should(HaveOccurred(), fmt.Sprintf("unexpectedly pod %s can reach %s with port %d",
				secondClientPod.Definition.Name, serverPod.Definition.Name, port5001))

			err = runTraffic(serverPod, ipaddr.RemovePrefix(firstClientPodIP), tcpProtocol, port5001)
			Expect(err).Should(HaveOccurred(), fmt.Sprintf("unexpectedly pod %s can reach %s with port %d",
				serverPod.Definition.Name, firstClientPod.Definition.Name, port5001))
		})

		// 55990
		It("Disable multi-network policy", polarion.ID("55990"), func() {
			By("Apply MultiNetworkPolicy with ingress rule deny all")
			_, err := networkpolicy.NewMultiNetworkPolicyBuilder(
				APIClient, multiNetworkPolicyName, tsparams.TestNamespaceName).
				WithNetwork(srIovNet.Definition.Name).WithPolicyType(multinetpolicyapiv1.PolicyTypeIngress).
				WithEmptyIngress().
				WithPodSelector(metaV1.LabelSelector{MatchLabels: map[string]string{"pod": labelServerPod}}).
				Create()
			Expect(err).ToNot(HaveOccurred(), "Failed to create multiNetworkPolicy")

			By("Traffic verification")
			// All traffic should be blocked to the serverPod
			err = runTraffic(firstClientPod, ipaddr.RemovePrefix(serverPodIP), tcpProtocol, port5001)
			Expect(err).Should(HaveOccurred(), fmt.Sprintf("Unexpectedly pod %s can reach %s with port %d",
				firstClientPod.Definition.Name, serverPod.Definition.Name, port5001))

			err = runTraffic(secondClientPod, ipaddr.RemovePrefix(serverPodIP), tcpProtocol, port5003)
			Expect(err).Should(HaveOccurred(), fmt.Sprintf("Unexpectedly pod %s can reach %s with port %d",
				secondClientPod.Definition.Name, serverPod.Definition.Name, port5003))

			// Traffic between firstClientPod and secondClientPod should not be affected (not blocked)
			err = runTraffic(secondClientPod, ipaddr.RemovePrefix(firstClientPodIP), tcpProtocol, port5001)
			Expect(err).ShouldNot(HaveOccurred(), fmt.Sprintf("Pod %s can NOT reach %s with port %d",
				secondClientPod.Definition.Name, firstClientPod.Definition.Name, port5001))

			By("Disable MultiNetworkPolicy feature")
			enableMultiNetworkPolicy(false)

			By("Traffic verification with MultiNetworkPolicy disabled")
			// All traffic is accepted and there is no any policy because feature is off
			err = runTraffic(firstClientPod, ipaddr.RemovePrefix(serverPodIP), tcpProtocol, port5001)
			Expect(err).ShouldNot(HaveOccurred(), fmt.Sprintf("pod %s can NOT reach %s with port %d",
				firstClientPod.Definition.Name, serverPod.Definition.Name, port5001))

			err = runTraffic(secondClientPod, ipaddr.RemovePrefix(serverPodIP), tcpProtocol, port5003)
			Expect(err).ShouldNot(HaveOccurred(), fmt.Sprintf("pod %s can NOT reach %s with port %d",
				secondClientPod.Definition.Name, serverPod.Definition.Name, port5003))

			By("Applying MultiNetworkPolicy should fail")
			_, err = networkpolicy.NewMultiNetworkPolicyBuilder(
				APIClient, multiNetworkPolicyName, tsparams.TestNamespaceName).
				WithNetwork(srIovNet.Definition.Name).WithPolicyType(multinetpolicyapiv1.PolicyTypeIngress).Create()
			Expect(err).To(HaveOccurred(), "Failed. Policy created with disabled multinetworkpolicy")

			By("Enable MultiNetworkPolicy feature")
			enableMultiNetworkPolicy(true)
		})
	})

	Context("ipv6 SCTP", func() {
		BeforeEach(func() {
			By("Creating first client pod")
			firstClientPod = createClientPod(
				"client1",
				srIovNet.Definition.Name,
				workerNodeList[1].Definition.Name,
				sctpProtocol,
				firstClientPodIPv6,
				labelFirstClientPod)

			By("Creating second client pod")
			secondClientPod = createClientPod(
				"client2",
				srIovNet.Definition.Name,
				workerNodeList[1].Definition.Name,
				sctpProtocol,
				secondClientPodIPv6,
				labelSecondClientPod)

			By("Creating server pod")
			serverPod = createServerPod(
				srIovNet.Definition.Name,
				workerNodeList[0].Definition.Name,
				serverPodIPv6,
				firstClientPodIPv6,
				secondClientPodIPv6,
				sctpProtocol)

			Consistently(func() bool {
				return firstClientPod.WaitUntilRunning(10*time.Second) == nil &&
					secondClientPod.WaitUntilRunning(10*time.Second) == nil &&
					serverPod.WaitUntilRunning(10*time.Second) == nil
			}, 1*time.Minute, 3*time.Second).Should(BeTrue(), "Failed not all pods are in running state")

			// Connectivity works without policy
			By("Testing connectivity without multiNetworkPolicy applied")
			err := runTraffic(firstClientPod, removePrefixFromIP(serverPodIPv6), sctpProtocol, port5001)
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("pod %s can NOT reach %s with port %d",
				firstClientPod.Definition.Name, serverPod.Definition.Name, port5001))
			err = runTraffic(secondClientPod, removePrefixFromIP(serverPodIPv6), sctpProtocol, port5003)
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("pod %s can NOT reach %s with port %d",
				secondClientPod.Definition.Name, serverPod.Definition.Name, port5003))
			err = runTraffic(secondClientPod, removePrefixFromIP(firstClientPodIPv6), sctpProtocol, port5001)
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("pod %s can NOT reach %s with port %d",
				secondClientPod.Definition.Name, firstClientPod.Definition.Name, port5001))

		})

		It("Ingress/Egress Allow access only to a specific port/protocol", polarion.ID("70040"), func() {
			ingressRule, err := networkpolicy.NewIngressRuleBuilder().WithPortAndProtocol(uint16(port5001), "SCTP").
				WithPeerPodSelector(metaV1.LabelSelector{MatchLabels: map[string]string{"pod": labelFirstClientPod}}).
				GetIngressRuleCfg()
			Expect(err).ToNot(HaveOccurred(), "Failed to build ingress rule")

			egressRule, err := networkpolicy.NewEgressRuleBuilder().WithPortAndProtocol(uint16(port5001), "SCTP").
				WithPeerPodSelectorAndCIDR(metaV1.LabelSelector{MatchLabels: map[string]string{"pod": labelSecondClientPod}},
					ipaddr.RemovePrefix(secondClientPodIPv6)+"/"+"128").GetEgressRuleCfg()
			Expect(err).ToNot(HaveOccurred(), "Failed to build egress rule")
			_, err = networkpolicy.NewMultiNetworkPolicyBuilder(
				APIClient, multiNetworkPolicyName, tsparams.TestNamespaceName).
				WithNetwork(srIovNet.Definition.Name).
				WithPodSelector(metaV1.LabelSelector{MatchLabels: map[string]string{"pod": labelServerPod}}).
				WithPolicyType(multinetpolicyapiv1.PolicyTypeIngress).WithPolicyType(multinetpolicyapiv1.PolicyTypeEgress).
				WithIngressRule(*ingressRule).WithEgressRule(*egressRule).Create()
			Expect(err).ToNot(HaveOccurred(), "Failed to create multiNetworkPolicy")

			By("Testing connectivity with multiNetworkPolicy applied")
			// Allowed port works as expected.
			err = runTraffic(firstClientPod, removePrefixFromIP(serverPodIPv6), sctpProtocol, port5001)
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("pod %s can NOT reach %s with port %d",
				firstClientPod.Definition.Name, serverPod.Definition.Name, port5001))
			// Same client but port is not allowed.
			err = runTraffic(firstClientPod, removePrefixFromIP(serverPodIPv6), sctpProtocol, port5003)
			Expect(err).To(HaveOccurred(), fmt.Sprintf("pod %s CAN reach %s with port %d",
				firstClientPod.Definition.Name, serverPod.Definition.Name, port5003))
			// Traffic from different pod denied.
			err = runTraffic(secondClientPod, removePrefixFromIP(serverPodIPv6), sctpProtocol, port5001)
			Expect(err).To(HaveOccurred(), fmt.Sprintf("pod %s CAN reach %s with port %d",
				secondClientPod.Definition.Name, serverPod.Definition.Name, port5001))
			// Egress policy works as expected.
			err = runTraffic(serverPod, removePrefixFromIP(firstClientPodIPv6), sctpProtocol, port5001)
			Expect(err).To(HaveOccurred(), fmt.Sprintf("pod %s CAN reach %s with port %d",
				serverPod.Definition.Name, firstClientPod.Definition.Name, port5001))
			err = runTraffic(serverPod, removePrefixFromIP(secondClientPodIPv6), sctpProtocol, port5001)
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("pod %s can NOT reach %s with port %d",
				serverPod.Definition.Name, secondClientPod.Definition.Name, port5001))
		})

		It("Ingress/Egress Allow access only to a specific subnet", polarion.ID("70041"), func() {
			ingressRule, err := networkpolicy.NewIngressRuleBuilder().
				WithCIDR(ipaddr.RemovePrefix(secondClientPodIPv6) + "/" + "128").
				GetIngressRuleCfg()
			Expect(err).ToNot(HaveOccurred(), "Failed to build ingress rule")

			policy, err := networkpolicy.NewMultiNetworkPolicyBuilder(
				APIClient, multiNetworkPolicyName, tsparams.TestNamespaceName).
				WithNetwork(srIovNet.Definition.Name).
				WithPodSelector(metaV1.LabelSelector{MatchLabels: map[string]string{"pod": labelServerPod}}).
				WithPolicyType(multinetpolicyapiv1.PolicyTypeIngress).
				WithIngressRule(*ingressRule).Create()
			Expect(err).ToNot(HaveOccurred(), "Failed to create multiNetworkPolicy")

			By("Testing connectivity with multiNetworkPolicy applied")
			err = runTraffic(secondClientPod, removePrefixFromIP(serverPodIPv6), sctpProtocol, port5001)
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("pod %s can NOT reach %s with port %d",
				secondClientPod.Definition.Name, serverPod.Definition.Name, port5001))
			err = runTraffic(secondClientPod, removePrefixFromIP(serverPodIPv6), sctpProtocol, port5003)
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("pod %s can NOT reach %s with port %d",
				secondClientPod.Definition.Name, serverPod.Definition.Name, port5003))

			err = runTraffic(firstClientPod, removePrefixFromIP(serverPodIPv6), sctpProtocol, port5001)
			Expect(err).To(HaveOccurred(), fmt.Sprintf("pod %s CAN reach %s with port %d",
				firstClientPod.Definition.Name, serverPod.Definition.Name, port5001))
			err = runTraffic(firstClientPod, removePrefixFromIP(serverPodIPv6), sctpProtocol, port5003)
			Expect(err).To(HaveOccurred(), fmt.Sprintf("pod %s CAN reach %s with port %d",
				firstClientPod.Definition.Name, serverPod.Definition.Name, port5003))

			err = policy.Delete()
			Expect(err).ToNot(HaveOccurred(), "Failed to delete multinetworkpolicy object")

			egressRule, err := networkpolicy.NewEgressRuleBuilder().
				WithPeerPodSelectorAndCIDR(metaV1.LabelSelector{MatchLabels: map[string]string{"pod": labelSecondClientPod}},
					ipaddr.RemovePrefix(secondClientPodIPv6)+"/"+"128").GetEgressRuleCfg()
			Expect(err).ToNot(HaveOccurred(), "Failed to build egressRule")

			_, err = networkpolicy.NewMultiNetworkPolicyBuilder(
				APIClient, multiNetworkPolicyName, tsparams.TestNamespaceName).
				WithNetwork(srIovNet.Definition.Name).
				WithPodSelector(metaV1.LabelSelector{MatchLabels: map[string]string{"pod": labelServerPod}}).
				WithPolicyType(multinetpolicyapiv1.PolicyTypeEgress).
				WithEgressRule(*egressRule).Create()
			Expect(err).ToNot(HaveOccurred(), "Failed to create multinetworkpolicy object")

			// Egress policy works as expected.
			err = runTraffic(serverPod, removePrefixFromIP(firstClientPodIPv6), sctpProtocol, port5001)
			Expect(err).To(HaveOccurred(), fmt.Sprintf("pod %s CAN reach %s with port %d",
				serverPod.Definition.Name, firstClientPod.Definition.Name, port5001))
			err = runTraffic(serverPod, removePrefixFromIP(secondClientPodIPv6), sctpProtocol, port5001)
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("pod %s can NOT reach %s with port %d",
				serverPod.Definition.Name, secondClientPod.Definition.Name, port5001))
		})
	})

	AfterEach(func() {
		testNameSpace, err := namespace.Pull(APIClient, tsparams.TestNamespaceName)
		Expect(err).ToNot(HaveOccurred(), "Failed to pull namespace")
		err = testNameSpace.CleanObjects(
			10*time.Minute,
			networkpolicy.GetMultiNetworkGVR())
		Expect(err).ToNot(HaveOccurred(), "Failed to remove multiNetworkPolicy object from namespace")

		protocol := tcpProtocol
		serverIP := serverPodIP
		// Pull the latest version of firstClientPod in order to get an updated network Annotations from the cluster.
		Expect(firstClientPod.Exists()).To(BeTrue(), "Client pod doesn't exist")
		if strings.Contains(firstClientPod.Object.Annotations["k8s.v1.cni.cncf.io/network-status"],
			removePrefixFromIP(firstClientPodIPv6)) {
			serverIP = serverPodIPv6
		}

		if strings.Contains(strings.Join(firstClientPod.Definition.Spec.Containers[0].Command, " "),
			fmt.Sprintf("protocol=%s", sctpProtocol)) {
			protocol = sctpProtocol
		}

		// All traffic is accepted
		err = runTraffic(firstClientPod, removePrefixFromIP(serverIP), protocol, port5001)
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

// runTraffic sends traffic from clientPod to serverIP.
func runTraffic(clientPod *pod.Builder, serverIP, protocol string, port int) error {
	buffer, err := clientPod.ExecCommand(
		[]string{"testcmd", fmt.Sprintf("-port=%d", port), "-interface=net1",
			fmt.Sprintf("-server=%s", serverIP), fmt.Sprintf("-protocol=%s", protocol), "-mtu=1200"},
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

	if status {
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
}

func defineClientCMD(protocol, serverIP string) []string {
	cmd := fmt.Sprintf("testcmd --listen --protocol=%s -port=%d -interface=net1 ", protocol, port5001)
	if protocol == sctpProtocol {
		cmd += fmt.Sprintf("-server=%s", serverIP)
	}

	return []string{"/bin/bash", "-c", cmd}
}

func createClientPod(podName, srIovNetwork, nodeName, protocol, ipaddress, label string) *pod.Builder {
	staticAnnotation := pod.StaticIPAnnotation(srIovNetwork, []string{ipaddress})
	clientPod, err := pod.NewBuilder(APIClient, podName, tsparams.TestNamespaceName, NetConfig.CnfNetTestContainer).
		WithSecondaryNetwork(staticAnnotation).
		DefineOnNode(nodeName).WithLabel("pod", label).
		RedefineDefaultCMD(defineClientCMD(protocol, removePrefixFromIP(ipaddress))).
		WithPrivilegedFlag().
		CreateAndWaitUntilRunning(tsparams.WaitTimeout)

	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to define pod annotation for clientPod with IPAddress: %s", ipaddress))

	return clientPod
}

func createServerPod(
	srIovNetworkName, nodeName, serverPodIP, firstClientPodIP, secondClientPodIP, protocol string) *pod.Builder {
	By("Creating server pod")

	initCommand := []string{"bash", "-c",
		fmt.Sprintf("ping %s -c 3 -w 90 && ping %s -c 3 -w 90",
			removePrefixFromIP(firstClientPodIP), removePrefixFromIP(secondClientPodIP))}

	InitContainer, err := pod.NewContainerBuilder(
		"init", NetConfig.CnfNetTestContainer, initCommand).GetContainerCfg()
	Expect(err).ToNot(HaveOccurred(), "Failed to define init container")

	serverPodContainer, err := pod.NewContainerBuilder(
		"testcmd", NetConfig.CnfNetTestContainer,
		setTestCmdServer(protocol, removePrefixFromIP(serverPodIP), port5003)).
		GetContainerCfg()
	Expect(err).ToNot(HaveOccurred(), "Failed to define server pod container")

	serverPod, err := pod.NewBuilder(
		APIClient, "server", tsparams.TestNamespaceName, NetConfig.CnfNetTestContainer).
		RedefineDefaultCMD(setTestCmdServer(protocol, removePrefixFromIP(serverPodIP), port5001)).
		WithAdditionalContainer(serverPodContainer).
		WithAdditionalInitContainer(InitContainer).
		DefineOnNode(nodeName).
		WithPrivilegedFlag().WithLabel("pod", labelServerPod).
		WithSecondaryNetwork(pod.StaticIPAnnotation(srIovNetworkName, []string{serverPodIP})).
		CreateAndWaitUntilRunning(tsparams.WaitTimeout)
	Expect(err).ToNot(HaveOccurred(), "Failed to create server pod")

	return serverPod
}

func setTestCmdServer(protocol, serverIP string, port int) []string {
	cmd := fmt.Sprintf("testcmd --listen -interface=net1 --protocol=%s --mtu=1500 -port=%d", protocol, port)
	if protocol == sctpProtocol {
		cmd += fmt.Sprintf(" -server=%s", serverIP)
	}

	return []string{"bash", "-c", cmd}
}

func removePrefixFromIP(ipAddr string) string {
	return strings.Split(ipAddr, "/")[0]
}
