package tests

import (
	"encoding/xml"
	"fmt"
	"net"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netinittools"

	multinetpolicyapiv1 "github.com/k8snetworkplumbingwg/multi-networkpolicy/pkg/apis/k8s.cni.cncf.io/v1beta1"
	"github.com/openshift-kni/eco-goinfra/pkg/nad"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/networkpolicy"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"github.com/openshift-kni/eco-goinfra/pkg/sriov"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netenv"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netparam"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/policy/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/internal/params"
	"gopkg.in/k8snetworkplumbingwg/multus-cni.v4/pkg/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

type podsData map[string]struct {
	IPv4      string
	IPv6      string
	Protocols []string
	Ports     []string
}

var _ = Describe("Multi-NetworkPolicy : Bond CNI", Ordered, Label("bondcnioversriov"), ContinueOnFailure, func() {

	var (
		sriovInterfacesUnderTest                         []string
		tNs1, tNs2                                       *namespace.Builder
		testPod1, testPod2, testPod3, testPod4, testPod5 *pod.Builder
		ports                                            = []string{"5001", "5002", "5003"}
		protocols                                        = []string{"tcp", "tcp", "udp"}
		allOpen                                          = map[string]string{"5001": "pass", "5002": "pass", "5003": "pass"}
		allClose                                         = map[string]string{"5001": "fail", "5002": "fail", "5003": "fail"}
		p5001Open                                        = map[string]string{"5001": "pass", "5002": "fail", "5003": "fail"}
		p5001p5002Open                                   = map[string]string{"5001": "pass", "5002": "pass", "5003": "fail"}
	)

	const (
		testNs1, testNs2             = "policy-ns1", "policy-ns2"
		nicPf1, nicPf2               = "pf1", "pf2"
		ns1, ns2                     = "ns1", "ns2"
		pod1, pod2, pod3, pod4, pod5 = "pod1", "pod2", "pod3", "pod4", "pod5"
	)

	testData := podsData{
		"pod1": {IPv4: "192.168.10.10/24", IPv6: "2001:0:0:1::10/64", Protocols: protocols, Ports: ports},
		"pod2": {IPv4: "192.168.10.11/24", IPv6: "2001:0:0:1::11/64", Protocols: protocols, Ports: ports},
		"pod3": {IPv4: "192.168.10.12/24", IPv6: "2001:0:0:1::12/64", Protocols: protocols, Ports: ports},
		"pod4": {IPv4: "192.168.20.11/24", IPv6: "2001:0:0:2::11/64", Protocols: protocols, Ports: ports},
		"pod5": {IPv4: "192.168.20.12/24", IPv6: "2001:0:0:2::12/64", Protocols: protocols, Ports: ports},
	}

	BeforeAll(func() {
		By("Verifying if Multi-NetPolicy tests can be executed on given cluster")
		err := netenv.DoesClusterHasEnoughNodes(APIClient, NetConfig, 1, 1)
		Expect(err).ToNot(HaveOccurred(),
			"Cluster doesn't support Multi-NetPolicy test cases as it doesn't have enough nodes")

		By("Listing Worker nodes")
		workerNodeList, err := nodes.List(
			APIClient, metav1.ListOptions{LabelSelector: labels.Set(NetConfig.WorkerLabelMap).String()})
		Expect(err).ToNot(HaveOccurred(), "Failed to list worker nodes")

		By("Fetching SR-IOV interfaces from ENV VAR")
		sriovInterfacesUnderTest, err = NetConfig.GetSriovInterfaces(2)
		Expect(err).ToNot(HaveOccurred(), "Failed to retrieve SR-IOV interfaces for testing")

		By("Enable MultiNetworkPolicy support")
		enableMultiNetworkPolicy(true)

		By("Deploy Test Resources: 2 Namespaces")
		tNs1, err = namespace.NewBuilder(APIClient, testNs1).WithMultipleLabels(params.PrivilegedNSLabels).
			WithLabel("ns", "ns1").Create()
		Expect(err).ToNot(HaveOccurred(), "Failed to create test namespace")
		tNs2, err = namespace.NewBuilder(APIClient, testNs2).WithMultipleLabels(params.PrivilegedNSLabels).
			WithLabel("ns", "ns2").Create()
		Expect(err).ToNot(HaveOccurred(), "Failed to create test namespace")

		By("Deploy Test Resources: 2 Sriov Policies")
		_, err = sriov.NewPolicyBuilder(APIClient,
			"nicpf1", NetConfig.SriovOperatorNamespace, nicPf1, 5,
			[]string{sriovInterfacesUnderTest[0]}, NetConfig.WorkerLabelMap).
			WithDevType("netdevice").
			Create()
		Expect(err).ToNot(HaveOccurred(), "Failed to create test policy")

		_, err = sriov.NewPolicyBuilder(APIClient,
			"nicpf2", NetConfig.SriovOperatorNamespace, nicPf2, 5,
			[]string{sriovInterfacesUnderTest[1]}, NetConfig.WorkerLabelMap).
			WithDevType("netdevice").
			Create()
		Expect(err).ToNot(HaveOccurred(), "Failed to create test policy")

		By("Deploy Test Resources: 4 Sriov Networks")
		defineAndCreateSriovNetwork(ns1+nicPf1, nicPf1, testNs1)
		defineAndCreateSriovNetwork(ns1+nicPf2, nicPf2, testNs1)
		defineAndCreateSriovNetwork(ns2+nicPf1, nicPf1, testNs2)
		defineAndCreateSriovNetwork(ns2+nicPf2, nicPf2, testNs2)

		err = netenv.WaitForSriovAndMCPStable(APIClient, tsparams.MCOWaitTimeout, 10*time.Second,
			NetConfig.CnfMcpLabel, NetConfig.SriovOperatorNamespace)
		Expect(err).ToNot(HaveOccurred(), "Sriov and MCP are not stable")

		By("Deploy Test Resources: 2 NADs for bond CNI")
		defineAndCreateBondNAD("bond", testNs1)
		defineAndCreateBondNAD("bond", testNs2)

		By("Deploy Test Resources: 5 Pods")
		testPod1 = defineAndCreatePodWithBondIf(pod1, testNs1, ns1+nicPf1, ns1+nicPf2,
			workerNodeList[0].Object.Name, testData)
		testPod2 = defineAndCreatePodWithBondIf(pod2, testNs1, ns1+nicPf1, ns1+nicPf2,
			workerNodeList[0].Object.Name, testData)
		testPod3 = defineAndCreatePodWithBondIf(pod3, testNs1, ns1+nicPf1, ns1+nicPf2,
			workerNodeList[0].Object.Name, testData)
		testPod4 = defineAndCreatePodWithBondIf(pod4, testNs2, ns2+nicPf1, ns2+nicPf2,
			workerNodeList[0].Object.Name, testData)
		testPod5 = defineAndCreatePodWithBondIf(pod5, testNs2, ns2+nicPf1, ns2+nicPf2,
			workerNodeList[0].Object.Name, testData)

		By("Check egress traffic from pod1 to other 4 pods. All ports should be open")

		verifyPaths(testPod1, testPod2, allOpen, allOpen, testData)
		verifyPaths(testPod1, testPod3, allOpen, allOpen, testData)
		verifyPaths(testPod1, testPod4, allOpen, allOpen, testData)
		verifyPaths(testPod1, testPod5, allOpen, allOpen, testData)

		By("Check ingress traffic to pod1 from other 4 pods. All ports should be open")

		verifyPaths(testPod2, testPod1, allOpen, allOpen, testData)
		verifyPaths(testPod3, testPod1, allOpen, allOpen, testData)
		verifyPaths(testPod4, testPod1, allOpen, allOpen, testData)
		verifyPaths(testPod5, testPod1, allOpen, allOpen, testData)
	})

	AfterEach(func() {
		err := tNs1.CleanObjects(1*time.Minute, networkpolicy.GetMultiNetworkGVR())
		Expect(err).ToNot(HaveOccurred(), "failed to clean Multi-NetworkPolicies in test namespace")
	})

	AfterAll(func() {
		By("Removing SRIOV configuration and wait for MCP stable")
		err := netenv.RemoveSriovConfigurationAndWaitForSriovAndMCPStable()
		Expect(err).ToNot(HaveOccurred(), "Failed to remove SRIOV configuration and MCP stable")

		By("Delete test namespace")
		err = tNs1.Delete()
		Expect(err).ToNot(HaveOccurred(), "Failed to delete test namespace")
		err = tNs2.Delete()
		Expect(err).ToNot(HaveOccurred(), "Failed to delete test namespace")
	})

	It("Egress - block all", reportxml.ID("77169"), func() {

		By("Create Multi Network Policy")
		_, err := networkpolicy.NewMultiNetworkPolicyBuilder(APIClient, "egress-deny", testNs1).
			WithNetwork(fmt.Sprintf("%s/bond,%s/bond", testNs1, testNs2)).
			WithPodSelector(metav1.LabelSelector{MatchLabels: map[string]string{"app": "pod1"}}).
			WithPolicyType(multinetpolicyapiv1.PolicyTypeEgress).
			Create()
		Expect(err).ToNot(HaveOccurred(), "Failed to create Multi Network Policy")

		By("Check egress traffic from pod1 to other 4 pods. All ports should be filtered")

		verifyPaths(testPod1, testPod2, allClose, allClose, testData)
		verifyPaths(testPod1, testPod3, allClose, allClose, testData)
		verifyPaths(testPod1, testPod4, allClose, allClose, testData)
		verifyPaths(testPod1, testPod5, allClose, allClose, testData)

		By("Check ingress traffic to pod1 from other 4 pods. All ports should be open")

		verifyPaths(testPod2, testPod1, allOpen, allOpen, testData)
		verifyPaths(testPod3, testPod1, allOpen, allOpen, testData)
		verifyPaths(testPod4, testPod1, allOpen, allOpen, testData)
		verifyPaths(testPod5, testPod1, allOpen, allOpen, testData)
	})

	It("Egress - allow all", reportxml.ID("77201"), func() {

		By("Create Multi Network Policy")
		testEgressRule, err := networkpolicy.NewEgressRuleBuilder().GetEgressRuleCfg()
		Expect(err).ToNot(HaveOccurred(), "egress rule configuration not generated")

		_, err = networkpolicy.NewMultiNetworkPolicyBuilder(APIClient, "egress-allow", testNs1).
			WithNetwork(fmt.Sprintf("%s/bond,%s/bond", testNs1, testNs2)).
			WithPodSelector(metav1.LabelSelector{MatchLabels: map[string]string{"app": "pod1"}}).
			WithPolicyType(multinetpolicyapiv1.PolicyTypeEgress).
			WithEgressRule(*testEgressRule).
			Create()
		Expect(err).ToNot(HaveOccurred(), "Failed to create Multi Network Policy")

		By("Check egress traffic from pod1 to other 4 pods. All ports should be open")

		verifyPaths(testPod1, testPod2, allOpen, allOpen, testData)
		verifyPaths(testPod1, testPod3, allOpen, allOpen, testData)
		verifyPaths(testPod1, testPod4, allOpen, allOpen, testData)
		verifyPaths(testPod1, testPod5, allOpen, allOpen, testData)

		By("Check ingress traffic to pod1 from other 4 pods. All ports should be open")

		verifyPaths(testPod2, testPod1, allOpen, allOpen, testData)
		verifyPaths(testPod3, testPod1, allOpen, allOpen, testData)
		verifyPaths(testPod4, testPod1, allOpen, allOpen, testData)
		verifyPaths(testPod5, testPod1, allOpen, allOpen, testData)
	})

	It("Egress - podSelector - NonExistent Label", reportxml.ID("77199"), func() {

		By("Create Multi Network Policy")
		testEgressRule, err := networkpolicy.NewEgressRuleBuilder().
			WithPeerPodSelector(metav1.LabelSelector{MatchLabels: map[string]string{"app": "none"}}).
			GetEgressRuleCfg()
		Expect(err).ToNot(HaveOccurred(), "egress rule configuration not generated")

		_, err = networkpolicy.NewMultiNetworkPolicyBuilder(APIClient, "egress-podsel-nonexist", testNs1).
			WithNetwork(fmt.Sprintf("%s/bond,%s/bond", testNs1, testNs2)).
			WithPodSelector(metav1.LabelSelector{MatchLabels: map[string]string{"app": "pod1"}}).
			WithPolicyType(multinetpolicyapiv1.PolicyTypeEgress).
			WithEgressRule(*testEgressRule).
			Create()
		Expect(err).NotTo(HaveOccurred(), "failed to create multi network policy")

		By("Check egress traffic from pod1 to other 4 pods. All ports should be filtered")

		verifyPaths(testPod1, testPod2, allClose, allClose, testData)
		verifyPaths(testPod1, testPod3, allClose, allClose, testData)
		verifyPaths(testPod1, testPod4, allClose, allClose, testData)
		verifyPaths(testPod1, testPod5, allClose, allClose, testData)

		By("Check ingress traffic to pod1 from other 4 pods. All ports should be open")

		verifyPaths(testPod2, testPod1, allOpen, allOpen, testData)
		verifyPaths(testPod3, testPod1, allOpen, allOpen, testData)
		verifyPaths(testPod4, testPod1, allOpen, allOpen, testData)
		verifyPaths(testPod5, testPod1, allOpen, allOpen, testData)
	})

	It("Egress - namespaceSelector - NonExistent Label", reportxml.ID("77197"), func() {

		By("Create Multi Network Policy")
		testEgressRule, err := networkpolicy.NewEgressRuleBuilder().
			WithPeerNamespaceSelector(metav1.LabelSelector{MatchLabels: map[string]string{"ns": "none"}}).
			GetEgressRuleCfg()
		Expect(err).ToNot(HaveOccurred(), "egress rule configuration not generated")

		_, err = networkpolicy.NewMultiNetworkPolicyBuilder(APIClient, "egress-nssel-nonexist", testNs1).
			WithNetwork(fmt.Sprintf("%s/bond,%s/bond", testNs1, testNs2)).
			WithPodSelector(metav1.LabelSelector{MatchLabels: map[string]string{"app": "pod1"}}).
			WithPolicyType(multinetpolicyapiv1.PolicyTypeEgress).
			WithEgressRule(*testEgressRule).
			Create()
		Expect(err).NotTo(HaveOccurred(), "failed to create multi network policy")

		By("Check egress traffic from pod1 to other 4 pods. All ports should be filtered")

		verifyPaths(testPod1, testPod2, allClose, allClose, testData)
		verifyPaths(testPod1, testPod3, allClose, allClose, testData)
		verifyPaths(testPod1, testPod4, allClose, allClose, testData)
		verifyPaths(testPod1, testPod5, allClose, allClose, testData)

		By("Check ingress traffic to pod1 from other 4 pods. All ports should be open")

		verifyPaths(testPod2, testPod1, allOpen, allOpen, testData)
		verifyPaths(testPod3, testPod1, allOpen, allOpen, testData)
		verifyPaths(testPod4, testPod1, allOpen, allOpen, testData)
		verifyPaths(testPod5, testPod1, allOpen, allOpen, testData)
	})

	It("Egress - Pod and/or Namespace Selector", reportxml.ID("77204"), func() {

		By("Create Multi Network Policy")
		testEgressRule, err := networkpolicy.NewEgressRuleBuilder().
			WithPeerPodAndNamespaceSelector(metav1.LabelSelector{MatchLabels: map[string]string{"app": "pod4"}},
				metav1.LabelSelector{MatchLabels: map[string]string{"ns": "ns2"}}).
			WithPeerPodSelector(metav1.LabelSelector{MatchLabels: map[string]string{"app": "pod2"}}).
			GetEgressRuleCfg()
		Expect(err).ToNot(HaveOccurred(), "egress rule configuration not generated")

		_, err = networkpolicy.NewMultiNetworkPolicyBuilder(APIClient, "egress-pod-ns-selector", testNs1).
			WithNetwork(fmt.Sprintf("%s/bond,%s/bond", testNs1, testNs2)).
			WithPodSelector(metav1.LabelSelector{MatchLabels: map[string]string{"app": "pod1"}}).
			WithPolicyType(multinetpolicyapiv1.PolicyTypeEgress).
			WithEgressRule(*testEgressRule).
			Create()
		Expect(err).NotTo(HaveOccurred(), "failed to create multi network policy")

		By("Check egress traffic from pod1 to other 4 pods. Only pod2 and pod4 should be accessible on all ports")

		verifyPaths(testPod1, testPod2, allOpen, allOpen, testData)
		verifyPaths(testPod1, testPod3, allClose, allClose, testData)
		verifyPaths(testPod1, testPod4, allOpen, allOpen, testData)
		verifyPaths(testPod1, testPod5, allClose, allClose, testData)

		By("Check ingress traffic to pod1 from other 4 pods. All ports should be open")

		verifyPaths(testPod2, testPod1, allOpen, allOpen, testData)
		verifyPaths(testPod3, testPod1, allOpen, allOpen, testData)
		verifyPaths(testPod4, testPod1, allOpen, allOpen, testData)
		verifyPaths(testPod5, testPod1, allOpen, allOpen, testData)
	})

	It("Egress - IPBlock IPv4 and IPv6 and Ports", reportxml.ID("77202"), func() {

		By("Create Multi Network Policy")
		testEgressRule, err := networkpolicy.NewEgressRuleBuilder().
			WithPortAndProtocol(5001, "TCP").
			WithCIDR("192.168.10.0/24", []string{"192.168.10.12/32"}).
			WithCIDR("2001:0:0:2::/64", []string{"2001:0:0:2::12/128"}).
			GetEgressRuleCfg()
		Expect(err).ToNot(HaveOccurred(), "egress rule configuration not generated")

		_, err = networkpolicy.NewMultiNetworkPolicyBuilder(APIClient, "egress-ipv4v6-port", testNs1).
			WithNetwork(fmt.Sprintf("%s/bond,%s/bond", testNs1, testNs2)).
			WithPodSelector(metav1.LabelSelector{MatchLabels: map[string]string{"app": "pod1"}}).
			WithPolicyType(multinetpolicyapiv1.PolicyTypeEgress).
			WithEgressRule(*testEgressRule).
			Create()
		Expect(err).NotTo(HaveOccurred(), "failed to create multi network policy")

		By("Check egress traffic from pod1 to other 4 pods. " +
			"Pod2 tcp port 5001 should be accessible over IPv4." +
			"Pod4 tcp port 5001 should be accessible over IPv6")

		verifyPaths(testPod1, testPod2, p5001Open, allClose, testData)
		verifyPaths(testPod1, testPod3, allClose, allClose, testData)
		verifyPaths(testPod1, testPod4, allClose, p5001Open, testData)
		verifyPaths(testPod1, testPod5, allClose, allClose, testData)

		By("Check ingress traffic to pod1 from other 4 pods. All ports should be open")

		verifyPaths(testPod2, testPod1, allOpen, allOpen, testData)
		verifyPaths(testPod3, testPod1, allOpen, allOpen, testData)
		verifyPaths(testPod4, testPod1, allOpen, allOpen, testData)
		verifyPaths(testPod5, testPod1, allOpen, allOpen, testData)
	})

	It("Ingress - block all", reportxml.ID("77237"), func() {

		By("Create Multi Network Policy")
		_, err := networkpolicy.NewMultiNetworkPolicyBuilder(APIClient, "ingress-deny", testNs1).
			WithNetwork(fmt.Sprintf("%s/bond,%s/bond", testNs1, testNs2)).
			WithPodSelector(metav1.LabelSelector{MatchLabels: map[string]string{"app": "pod1"}}).
			WithPolicyType(multinetpolicyapiv1.PolicyTypeIngress).
			Create()
		Expect(err).ToNot(HaveOccurred(), "Failed to create Multi Network Policy")

		By("Check egress traffic from pod1 to other 4 pods. All ports should be open")

		verifyPaths(testPod1, testPod2, allOpen, allOpen, testData)
		verifyPaths(testPod1, testPod3, allOpen, allOpen, testData)
		verifyPaths(testPod1, testPod4, allOpen, allOpen, testData)
		verifyPaths(testPod1, testPod5, allOpen, allOpen, testData)

		By("Check ingress traffic to pod1 from other 4 pods. All ports should be filtered")

		verifyPaths(testPod2, testPod1, allClose, allClose, testData)
		verifyPaths(testPod3, testPod1, allClose, allClose, testData)
		verifyPaths(testPod4, testPod1, allClose, allClose, testData)
		verifyPaths(testPod5, testPod1, allClose, allClose, testData)
	})

	It("Ingress - allow all", reportxml.ID("77236"), func() {

		By("Create Multi Network Policy")
		testIngressRule, err := networkpolicy.NewIngressRuleBuilder().GetIngressRuleCfg()
		Expect(err).ToNot(HaveOccurred(), "ingress rule configuration not generated")

		_, err = networkpolicy.NewMultiNetworkPolicyBuilder(APIClient, "ingress-allow", testNs1).
			WithNetwork(fmt.Sprintf("%s/bond,%s/bond", testNs1, testNs2)).
			WithPodSelector(metav1.LabelSelector{MatchLabels: map[string]string{"app": "pod1"}}).
			WithPolicyType(multinetpolicyapiv1.PolicyTypeIngress).
			WithIngressRule(*testIngressRule).
			Create()
		Expect(err).ToNot(HaveOccurred(), "Failed to create Multi Network Policy")

		By("Check egress traffic from pod1 to other 4 pods. All ports should be open")

		verifyPaths(testPod1, testPod2, allOpen, allOpen, testData)
		verifyPaths(testPod1, testPod3, allOpen, allOpen, testData)
		verifyPaths(testPod1, testPod4, allOpen, allOpen, testData)
		verifyPaths(testPod1, testPod5, allOpen, allOpen, testData)

		By("Check ingress traffic to pod1 from other 4 pods. All ports should be open")

		verifyPaths(testPod2, testPod1, allOpen, allOpen, testData)
		verifyPaths(testPod3, testPod1, allOpen, allOpen, testData)
		verifyPaths(testPod4, testPod1, allOpen, allOpen, testData)
		verifyPaths(testPod5, testPod1, allOpen, allOpen, testData)
	})

	It("Ingress - podSelector - NonExistent Label", reportxml.ID("77233"), func() {

		By("Create Multi Network Policy")
		testIngressRule, err := networkpolicy.NewIngressRuleBuilder().
			WithPeerPodSelector(metav1.LabelSelector{MatchLabels: map[string]string{"app": "none"}}).
			GetIngressRuleCfg()
		Expect(err).ToNot(HaveOccurred(), "ingress rule configuration not generated")

		_, err = networkpolicy.NewMultiNetworkPolicyBuilder(APIClient, "ingress-podsel-nonexist", testNs1).
			WithNetwork(fmt.Sprintf("%s/bond,%s/bond", testNs1, testNs2)).
			WithPodSelector(metav1.LabelSelector{MatchLabels: map[string]string{"app": "pod1"}}).
			WithPolicyType(multinetpolicyapiv1.PolicyTypeIngress).
			WithIngressRule(*testIngressRule).
			Create()
		Expect(err).NotTo(HaveOccurred(), "failed to create multi network policy")

		By("Check egress traffic from pod1 to other 4 pods. All ports should be open")

		verifyPaths(testPod1, testPod2, allOpen, allOpen, testData)
		verifyPaths(testPod1, testPod3, allOpen, allOpen, testData)
		verifyPaths(testPod1, testPod4, allOpen, allOpen, testData)
		verifyPaths(testPod1, testPod5, allOpen, allOpen, testData)

		By("Check ingress traffic to pod1 from other 4 pods. All ports should be filtered")

		verifyPaths(testPod2, testPod1, allClose, allClose, testData)
		verifyPaths(testPod3, testPod1, allClose, allClose, testData)
		verifyPaths(testPod4, testPod1, allClose, allClose, testData)
		verifyPaths(testPod5, testPod1, allClose, allClose, testData)
	})

	It("Ingress - namespaceSelector - NonExistent Label", reportxml.ID("77235"), func() {

		By("Create Multi Network Policy")
		testIngressRule, err := networkpolicy.NewIngressRuleBuilder().
			WithPeerNamespaceSelector(metav1.LabelSelector{MatchLabels: map[string]string{"ns": "none"}}).
			GetIngressRuleCfg()
		Expect(err).ToNot(HaveOccurred(), "ingress rule configuration not generated")

		_, err = networkpolicy.NewMultiNetworkPolicyBuilder(APIClient, "ingress-nssel-nonexist", testNs1).
			WithNetwork(fmt.Sprintf("%s/bond,%s/bond", testNs1, testNs2)).
			WithPodSelector(metav1.LabelSelector{MatchLabels: map[string]string{"app": "pod1"}}).
			WithPolicyType(multinetpolicyapiv1.PolicyTypeIngress).
			WithIngressRule(*testIngressRule).
			Create()
		Expect(err).NotTo(HaveOccurred(), "failed to create multi network policy")

		By("Check egress traffic from pod1 to other 4 pods. All ports should be open")

		verifyPaths(testPod1, testPod2, allOpen, allOpen, testData)
		verifyPaths(testPod1, testPod3, allOpen, allOpen, testData)
		verifyPaths(testPod1, testPod4, allOpen, allOpen, testData)
		verifyPaths(testPod1, testPod5, allOpen, allOpen, testData)

		By("Check ingress traffic to pod1 from other 4 pods. All ports should be filtered")

		verifyPaths(testPod2, testPod1, allClose, allClose, testData)
		verifyPaths(testPod3, testPod1, allClose, allClose, testData)
		verifyPaths(testPod4, testPod1, allClose, allClose, testData)
		verifyPaths(testPod5, testPod1, allClose, allClose, testData)
	})

	It("Ingress - Pod and/or Namespace Selector", reportxml.ID("77242"), func() {

		By("Create Multi Network Policy")
		testIngressRule, err := networkpolicy.NewIngressRuleBuilder().
			WithPeerPodAndNamespaceSelector(metav1.LabelSelector{MatchLabels: map[string]string{"app": "pod4"}},
				metav1.LabelSelector{MatchLabels: map[string]string{"ns": "ns2"}}).
			WithPeerPodSelector(metav1.LabelSelector{MatchLabels: map[string]string{"app": "pod2"}}).
			GetIngressRuleCfg()
		Expect(err).ToNot(HaveOccurred(), "ingress rule configuration not generated")

		_, err = networkpolicy.NewMultiNetworkPolicyBuilder(APIClient, "ingress-pod-ns-selector", testNs1).
			WithNetwork(fmt.Sprintf("%s/bond,%s/bond", testNs1, testNs2)).
			WithPodSelector(metav1.LabelSelector{MatchLabels: map[string]string{"app": "pod1"}}).
			WithPolicyType(multinetpolicyapiv1.PolicyTypeIngress).
			WithIngressRule(*testIngressRule).
			Create()
		Expect(err).NotTo(HaveOccurred(), "failed to create multi network policy")

		By("Check egress traffic from pod1 to other 4 pods. All ports should be open")

		verifyPaths(testPod1, testPod2, allOpen, allOpen, testData)
		verifyPaths(testPod1, testPod3, allOpen, allOpen, testData)
		verifyPaths(testPod1, testPod4, allOpen, allOpen, testData)
		verifyPaths(testPod1, testPod5, allOpen, allOpen, testData)

		By("Check ingress traffic to pod1 from other 4 pods. Only pod2 and pod4 can access pod1 on all ports")

		verifyPaths(testPod2, testPod1, allOpen, allOpen, testData)
		verifyPaths(testPod3, testPod1, allClose, allClose, testData)
		verifyPaths(testPod4, testPod1, allOpen, allOpen, testData)
		verifyPaths(testPod5, testPod1, allClose, allClose, testData)
	})

	It("Ingress - IPBlock IPv4 and IPv6 and Ports", reportxml.ID("77238"), func() {

		By("Create Multi Network Policy")
		testIngressRule, err := networkpolicy.NewIngressRuleBuilder().
			WithPortAndProtocol(5001, "TCP").
			WithCIDR("192.168.10.0/24", []string{"192.168.10.12/32"}).
			WithCIDR("2001:0:0:2::/64", []string{"2001:0:0:2::12/128"}).
			GetIngressRuleCfg()
		Expect(err).ToNot(HaveOccurred(), "ingress rule configuration not generated")

		_, err = networkpolicy.NewMultiNetworkPolicyBuilder(APIClient, "ingress-ipv4v6-port", testNs1).
			WithNetwork(fmt.Sprintf("%s/bond,%s/bond", testNs1, testNs2)).
			WithPodSelector(metav1.LabelSelector{MatchLabels: map[string]string{"app": "pod1"}}).
			WithPolicyType(multinetpolicyapiv1.PolicyTypeIngress).
			WithIngressRule(*testIngressRule).
			Create()
		Expect(err).NotTo(HaveOccurred(), "failed to create multi network policy")

		By("Check egress traffic from pod1 to other 4 pods. All ports should be open")

		verifyPaths(testPod1, testPod2, allOpen, allOpen, testData)
		verifyPaths(testPod1, testPod3, allOpen, allOpen, testData)
		verifyPaths(testPod1, testPod4, allOpen, allOpen, testData)
		verifyPaths(testPod1, testPod5, allOpen, allOpen, testData)

		By("Check ingress traffic to pod1 from other 4 pods. " +
			"Pod2 can access tcp port 5001 of pod1 over IPv4." +
			"Pod4 can access tcp port 5001 of pod4 over IPv6")

		verifyPaths(testPod2, testPod1, p5001Open, allClose, testData)
		verifyPaths(testPod3, testPod1, allClose, allClose, testData)
		verifyPaths(testPod4, testPod1, allClose, p5001Open, testData)
		verifyPaths(testPod5, testPod1, allClose, allClose, testData)
	})

	It("Ingress & Egress - Peer and Ports", reportxml.ID("77469"), func() {

		By("Create Multi Network Policy")
		testEgressRule, err := networkpolicy.NewEgressRuleBuilder().
			WithPeerPodSelector(metav1.LabelSelector{MatchLabels: map[string]string{"app": "pod2"}}).
			WithCIDR("2001:0:0:2::/64", []string{"2001:0:0:2::11/128"}).
			GetEgressRuleCfg()
		Expect(err).ToNot(HaveOccurred(), "egress rule configuration not generated")

		testIngressRule, err := networkpolicy.NewIngressRuleBuilder().
			WithCIDR("192.168.10.0/24", []string{"192.168.10.12/32"}).
			WithPeerPodAndNamespaceSelector(metav1.LabelSelector{MatchLabels: map[string]string{"app": "pod4"}},
				metav1.LabelSelector{MatchLabels: map[string]string{"ns": "ns2"}}).
			WithProtocol("TCP").
			GetIngressRuleCfg()
		Expect(err).ToNot(HaveOccurred(), "ingress rule configuration not generated")

		_, err = networkpolicy.NewMultiNetworkPolicyBuilder(APIClient, "ingress-egress", "policy-ns1").
			WithNetwork("policy-ns1/bond,policy-ns2/bond").
			WithPodSelector(metav1.LabelSelector{MatchLabels: map[string]string{"app": "pod1"}}).
			WithPolicyType(multinetpolicyapiv1.PolicyTypeIngress).
			WithPolicyType(multinetpolicyapiv1.PolicyTypeEgress).
			WithIngressRule(*testIngressRule).
			WithEgressRule(*testEgressRule).
			Create()
		Expect(err).NotTo(HaveOccurred(), "failed to create multi network policy")

		By("Check egress traffic from pod1 to other 4 pods. Only Pod5 ports should be accessible over IPv6")

		verifyPaths(testPod1, testPod2, allClose, allClose, testData)
		verifyPaths(testPod1, testPod3, allClose, allClose, testData)
		verifyPaths(testPod1, testPod4, allClose, allClose, testData)
		verifyPaths(testPod1, testPod5, allClose, allOpen, testData)

		By("Check ingress traffic to pod1 from other 4 pods. " +
			"Pod2 can access tcp ports 5001 & 5002 of pod1 over IPv4. " +
			"Pod4 can access tcp ports 5001 & 5002 of pod1 over both IPv4 & IPv6")

		verifyPaths(testPod2, testPod1, p5001p5002Open, allClose, testData)
		verifyPaths(testPod3, testPod1, allClose, allClose, testData)
		verifyPaths(testPod4, testPod1, p5001p5002Open, p5001p5002Open, testData)
		verifyPaths(testPod5, testPod1, allClose, allClose, testData)
	})
})

func defineAndCreateSriovNetwork(netName, resName, targetNs string) {
	_, err := sriov.NewNetworkBuilder(
		APIClient, netName, NetConfig.SriovOperatorNamespace, targetNs, resName).
		WithLogLevel(netparam.LogLevelDebug).
		Create()
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to create Sriov Network %s", netName))
}

func defineAndCreateBondNAD(nadName, nsName string) {
	config, err := nad.NewMasterBondPlugin("bond", "active-backup").
		WithFailOverMac(1).
		WithLinksInContainer(true).
		WithMiimon(100).
		WithLinks([]nad.Link{{Name: "net1"}, {Name: "net2"}}).
		WithCapabilities(&nad.Capability{IPs: true}).
		WithIPAM(&nad.IPAM{
			Type:   "static",
			Routes: []nad.Routes{{Dst: "192.168.0.0/16"}, {Dst: "2001::0/62"}},
		}).GetMasterPluginConfig()
	Expect(err).ToNot(HaveOccurred(), "Failed to get master bond plugin config")

	_, err = nad.NewBuilder(APIClient, nadName, nsName).WithMasterPlugin(config).Create()
	Expect(err).ToNot(HaveOccurred(), "Failed to create net-attach-def")
}

func defineAndCreatePodWithBondIf(
	podName, nsName, net1Name, net2Name, nodeName string, testData podsData) *pod.Builder {
	var rootUser int64

	securityContext := corev1.SecurityContext{
		RunAsUser: &rootUser,
		Capabilities: &corev1.Capabilities{
			Add: []corev1.Capability{"IPC_LOCK", "SYS_RESOURCE", "NET_RAW", "NET_ADMIN"},
		},
	}

	netAnnotation := []*types.NetworkSelectionElement{
		{
			Name:             net1Name,
			InterfaceRequest: "net1",
		},
		{
			Name:             net2Name,
			InterfaceRequest: "net2",
		},
		{
			Name:             "bond",
			InterfaceRequest: "bond1",
			IPRequest:        []string{testData[podName].IPv4, testData[podName].IPv6},
		},
	}

	tPodBuilder := pod.NewBuilder(APIClient, podName, nsName, NetConfig.CnfNetTestContainer).
		WithNodeSelector(map[string]string{"kubernetes.io/hostname": nodeName}).
		WithSecondaryNetwork(netAnnotation).
		WithPrivilegedFlag().
		WithLabel("app", podName)

	for index := range len(testData[podName].Protocols) {
		containerBuilder, err := pod.NewContainerBuilder(testData[podName].Protocols[index]+testData[podName].Ports[index],
			NetConfig.CnfNetTestContainer,
			[]string{"/bin/bash", "-c", fmt.Sprintf("testcmd -listen -interface bond1 -protocol %s -port %s",
				testData[podName].Protocols[index], testData[podName].Ports[index])}).
			WithSecurityContext(&securityContext).
			GetContainerCfg()
		Expect(err).ToNot(HaveOccurred(), "Failed to get container config")

		if index == 0 {
			tPodBuilder.RedefineDefaultContainer(*containerBuilder)
		} else {
			tPodBuilder.WithAdditionalContainer(containerBuilder)
		}
	}

	tPod, err := tPodBuilder.CreateAndWaitUntilRunning(1 * time.Minute)
	Expect(err).ToNot(HaveOccurred(), "Failed to create test pod")

	return tPod
}

func verifyPaths(
	sPod, dPod *pod.Builder,
	ipv4ExpectedResult, ipv6ExpectedResult map[string]string,
	testData podsData,
) {
	By("Deriving applicable paths between given source and destination pods")
	runNmapAndValidateResults(sPod, testData[sPod.Object.Name].IPv4,
		testData[dPod.Object.Name].Protocols, testData[dPod.Object.Name].Ports,
		strings.Split(testData[dPod.Object.Name].IPv4, "/")[0], ipv4ExpectedResult)
	runNmapAndValidateResults(sPod, testData[sPod.Object.Name].IPv6,
		testData[dPod.Object.Name].Protocols, testData[dPod.Object.Name].Ports,
		strings.Split(testData[dPod.Object.Name].IPv6, "/")[0], ipv6ExpectedResult)
}

func runNmapAndValidateResults(
	sPod *pod.Builder,
	sourceIP string,
	protocols []string,
	ports []string,
	targetIP string,
	expectedResult map[string]string) {
	// NmapXML defines the structure nmap command output in xml.
	type NmapXML struct {
		XMLName xml.Name `xml:"nmaprun"`
		Text    string   `xml:",chardata"`
		Host    struct {
			Text   string `xml:",chardata"`
			Status struct {
				Text  string `xml:",chardata"`
				State string `xml:"state,attr"`
			} `xml:"status"`
			Address []struct {
				Text     string `xml:",chardata"`
				Addr     string `xml:"addr,attr"`
				Addrtype string `xml:"addrtype,attr"`
			} `xml:"address"`
			Ports struct {
				Text string `xml:",chardata"`
				Port []struct {
					Text     string `xml:",chardata"`
					Protocol string `xml:"protocol,attr"`
					Portid   string `xml:"portid,attr"`
					State    struct {
						Text  string `xml:",chardata"`
						State string `xml:"state,attr"`
					} `xml:"state"`
				} `xml:"port"`
			} `xml:"ports"`
		} `xml:"host"`
	}

	By("Running nmap command in source pod")

	var nmapOutput NmapXML

	nmapCmd := fmt.Sprintf("nmap -v -oX - -sT -sU -p T:5001,T:5002,U:5003 %s", targetIP)

	if net.ParseIP(targetIP).To4() == nil {
		nmapCmd += " -6"
	}

	output, err := sPod.ExecCommand([]string{"/bin/bash", "-c", nmapCmd})
	Expect(err).NotTo(HaveOccurred(), "Failed to execute nmap command in source pod")

	err = xml.Unmarshal(output.Bytes(), &nmapOutput)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to unmarshal nmap output: %s", output.String()))

	By("Verifying nmap output is matching with expected results")
	Expect(len(nmapOutput.Host.Ports.Port)).To(Equal(len(ports)),
		fmt.Sprintf("number of ports in nmap output as expected. Nmap XML output: %v", nmapOutput.Host.Ports.Port))

	for index := range len(nmapOutput.Host.Ports.Port) {
		if expectedResult[nmapOutput.Host.Ports.Port[index].Portid] == "pass" {
			By(fmt.Sprintf("Path %s/%s =====> %s:%s:%s Expected to Pass\n",
				sPod.Object.Name, sourceIP, targetIP, protocols[index], ports[index]))
			Expect(nmapOutput.Host.Ports.Port[index].State.State).To(Equal("open"),
				fmt.Sprintf("Port is not open as expected. Output: %v", nmapOutput.Host.Ports.Port[index]))
		} else {
			By(fmt.Sprintf("Path %s/%s =====> %s:%s:%s Expected to Fail\n",
				sPod.Object.Name, sourceIP, targetIP, protocols[index], ports[index]))
			Expect(nmapOutput.Host.Ports.Port[index].State.State).To(SatisfyAny(Equal("open|filtered"), Equal("filtered")),
				fmt.Sprintf("Port is not filtered as expected. Output: %v", nmapOutput.Host.Ports.Port[index]))
		}
	}
}
