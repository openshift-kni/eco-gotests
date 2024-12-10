package tests

import (
	"encoding/xml"
	"fmt"
	"net"
	"slices"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netinittools"

	multinetpolicyapiv1 "github.com/k8snetworkplumbingwg/multi-networkpolicy/pkg/apis/k8s.cni.cncf.io/v1beta1"
	"github.com/openshift-kni/eco-goinfra/pkg/nad"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/networkpolicy"
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
)

type podsData map[string]struct {
	IPv4         string
	IPv6         string
	ProtocolPort []string
}

var _ = Describe("Multi-NetworkPolicy : Bond CNI", Ordered, Label("bondcnioversriov"), ContinueOnFailure, func() {

	var (
		sriovInterfacesUnderTest                         []string
		tNs1, tNs2                                       *namespace.Builder
		testPod1, testPod2, testPod3, testPod4, testPod5 *pod.Builder
	)

	const (
		testNs1, testNs2             = "policy-ns1", "policy-ns2"
		nicPf1, nicPf2               = "pf1", "pf2"
		ns1, ns2                     = "ns1", "ns2"
		pod1, pod2, pod3, pod4, pod5 = "pod1", "pod2", "pod3", "pod4", "pod5"
	)

	testData := podsData{
		"pod1": {IPv4: "192.168.10.10/24", IPv6: "2001:0:0:1::10/64",
			ProtocolPort: []string{"tcp:5001", "tcp:5002", "udp:5003"}},
		"pod2": {IPv4: "192.168.10.11/24", IPv6: "2001:0:0:1::11/64",
			ProtocolPort: []string{"tcp:5001", "tcp:5002", "udp:5003"}},
		"pod3": {IPv4: "192.168.10.12/24", IPv6: "2001:0:0:1::12/64",
			ProtocolPort: []string{"tcp:5001", "tcp:5002", "udp:5003"}},
		"pod4": {IPv4: "192.168.20.11/24", IPv6: "2001:0:0:2::11/64",
			ProtocolPort: []string{"tcp:5001", "tcp:5002", "udp:5003"}},
		"pod5": {IPv4: "192.168.20.12/24", IPv6: "2001:0:0:2::12/64",
			ProtocolPort: []string{"tcp:5001", "tcp:5002", "udp:5003"}},
	}

	BeforeAll(func() {
		By("Verifying if Multi-NetPolicy tests can be executed on given cluster")
		err := netenv.DoesClusterHasEnoughNodes(APIClient, NetConfig, 1, 1)
		Expect(err).ToNot(HaveOccurred(),
			"Cluster doesn't support Multi-NetPolicy test cases as it doesn't have enough nodes")

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
		testPod1 = defineAndCreatePodWithBondIf(pod1, testNs1, ns1+nicPf1, ns1+nicPf2, testData)
		testPod2 = defineAndCreatePodWithBondIf(pod2, testNs1, ns1+nicPf1, ns1+nicPf2, testData)
		testPod3 = defineAndCreatePodWithBondIf(pod3, testNs1, ns1+nicPf1, ns1+nicPf2, testData)
		testPod4 = defineAndCreatePodWithBondIf(pod4, testNs2, ns2+nicPf1, ns2+nicPf2, testData)
		testPod5 = defineAndCreatePodWithBondIf(pod5, testNs2, ns2+nicPf1, ns2+nicPf2, testData)

		// Expected Result 1's and 0's represent Pass and Fail respectively when converted to Binary.
		// Since we have 3 ports per destination pod, Octal notation is intuitive for readability.
		// Example: expected Result (Octal) 0o70 --> (Binary) 0b111000 represents as below for the destination pod.
		//
		//   Pass     Pass     Pass     Fail     Fail     Fail
		//     1       1        1        0         0       0
		// tcp:5001 tcp:5002 udp:5003 tcp:5001 tcp:5002 udp:5003
		// --IPv4-- --IPv4-- --IPv4-- --IPv6-- --IPv6-- --IPv6--
		// -------------------Destination Pod-------------------
		//
		// Appropriate octal number is passed for each test.
		By("Check egress traffic from pod1 to other 4 pods. All ports should be accessible")
		defineNmapValidPaths(
			[]*pod.Builder{testPod1}, []*pod.Builder{testPod2, testPod3, testPod4, testPod5},
			[]int64{0o77, 0o77, 0o77, 0o77}, testData)

		By("Check ingress traffic to pod1 from other 4 pods. All ports should be accessible")
		defineNmapValidPaths(
			[]*pod.Builder{testPod2, testPod3, testPod4, testPod5}, []*pod.Builder{testPod1},
			[]int64{0o77, 0o77, 0o77, 0o77}, testData)
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

		By("Check egress traffic from pod1 to other 4 pods. All ports should be blocked")
		defineNmapValidPaths(
			[]*pod.Builder{testPod1}, []*pod.Builder{testPod2, testPod3, testPod4, testPod5},
			[]int64{0o00, 0o00, 0o00, 0o00}, testData)

		By("Check ingress traffic to pod1 from other 4 pods. All ports should be accessible")
		defineNmapValidPaths(
			[]*pod.Builder{testPod2, testPod3, testPod4, testPod5}, []*pod.Builder{testPod1},
			[]int64{0o77, 0o77, 0o77, 0o77}, testData)
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

		By("Check egress traffic from pod1 to other 4 pods. All ports should be accessible")
		defineNmapValidPaths(
			[]*pod.Builder{testPod1}, []*pod.Builder{testPod2, testPod3, testPod4, testPod5},
			[]int64{0o77, 0o77, 0o77, 0o77}, testData)

		By("Check ingress traffic to pod1 from other 4 pods. All ports should be accessible")
		defineNmapValidPaths(
			[]*pod.Builder{testPod2, testPod3, testPod4, testPod5}, []*pod.Builder{testPod1},
			[]int64{0o77, 0o77, 0o77, 0o77}, testData)
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

		By("Check egress traffic from pod1 to other 4 pods. All ports should be blocked")
		defineNmapValidPaths(
			[]*pod.Builder{testPod1}, []*pod.Builder{testPod2, testPod3, testPod4, testPod5},
			[]int64{0o00, 0o00, 0o00, 0o00}, testData)

		By("Check ingress traffic to pod1 from other 4 pods. All ports should be accessible")
		defineNmapValidPaths(
			[]*pod.Builder{testPod2, testPod3, testPod4, testPod5}, []*pod.Builder{testPod1},
			[]int64{0o77, 0o77, 0o77, 0o77}, testData)
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

		By("Check egress traffic from pod1 to other 4 pods. All ports should be blocked")
		defineNmapValidPaths(
			[]*pod.Builder{testPod1}, []*pod.Builder{testPod2, testPod3, testPod4, testPod5},
			[]int64{0o00, 0o00, 0o00, 0o00}, testData)

		By("Check ingress traffic to pod1 from other 4 pods. All ports should be accessible")
		defineNmapValidPaths(
			[]*pod.Builder{testPod2, testPod3, testPod4, testPod5}, []*pod.Builder{testPod1},
			[]int64{0o77, 0o77, 0o77, 0o77}, testData)
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
		defineNmapValidPaths(
			[]*pod.Builder{testPod1}, []*pod.Builder{testPod2, testPod3, testPod4, testPod5},
			[]int64{0o77, 0o00, 0o77, 0o00}, testData)

		By("Check ingress traffic to pod1 from other 4 pods. All ports should be accessible")
		defineNmapValidPaths(
			[]*pod.Builder{testPod2, testPod3, testPod4, testPod5}, []*pod.Builder{testPod1},
			[]int64{0o77, 0o77, 0o77, 0o77}, testData)
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
		defineNmapValidPaths(
			[]*pod.Builder{testPod1}, []*pod.Builder{testPod2, testPod3, testPod4, testPod5},
			[]int64{0o40, 0o00, 0o04, 0o00}, testData)

		By("Check ingress traffic to pod1 from other 4 pods. All ports should be accessible")
		defineNmapValidPaths(
			[]*pod.Builder{testPod2, testPod3, testPod4, testPod5}, []*pod.Builder{testPod1},
			[]int64{0o77, 0o77, 0o77, 0o77}, testData)
	})

	It("Ingress - block all", reportxml.ID("77237"), func() {

		By("Create Multi Network Policy")
		_, err := networkpolicy.NewMultiNetworkPolicyBuilder(APIClient, "ingress-deny", testNs1).
			WithNetwork(fmt.Sprintf("%s/bond,%s/bond", testNs1, testNs2)).
			WithPodSelector(metav1.LabelSelector{MatchLabels: map[string]string{"app": "pod1"}}).
			WithPolicyType(multinetpolicyapiv1.PolicyTypeIngress).
			Create()
		Expect(err).ToNot(HaveOccurred(), "Failed to create Multi Network Policy")

		By("Check egress traffic from pod1 to other 4 pods. All ports should be accessible")
		defineNmapValidPaths(
			[]*pod.Builder{testPod1}, []*pod.Builder{testPod2, testPod3, testPod4, testPod5},
			[]int64{0o77, 0o77, 0o77, 0o77}, testData)

		By("Check ingress traffic to pod1 from other 4 pods. All ports should be blocked")
		defineNmapValidPaths(
			[]*pod.Builder{testPod2, testPod3, testPod4, testPod5}, []*pod.Builder{testPod1},
			[]int64{0o00, 0o00, 0o00, 0o00}, testData)
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

		By("Check egress traffic from pod1 to other 4 pods. All ports should be accessible")
		defineNmapValidPaths(
			[]*pod.Builder{testPod1}, []*pod.Builder{testPod2, testPod3, testPod4, testPod5},
			[]int64{0o77, 0o77, 0o77, 0o77}, testData)

		By("Check ingress traffic to pod1 from other 4 pods. All ports should be accessible")
		defineNmapValidPaths(
			[]*pod.Builder{testPod2, testPod3, testPod4, testPod5}, []*pod.Builder{testPod1},
			[]int64{0o77, 0o77, 0o77, 0o77}, testData)
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

		By("Check egress traffic from pod1 to other 4 pods. All ports should be accessible")
		defineNmapValidPaths(
			[]*pod.Builder{testPod1}, []*pod.Builder{testPod2, testPod3, testPod4, testPod5},
			[]int64{0o77, 0o77, 0o77, 0o77}, testData)

		By("Check ingress traffic to pod1 from other 4 pods. All ports should be blocked")
		defineNmapValidPaths(
			[]*pod.Builder{testPod2, testPod3, testPod4, testPod5}, []*pod.Builder{testPod1},
			[]int64{0o00, 0o00, 0o00, 0o00}, testData)
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

		By("Check egress traffic from pod1 to other 4 pods. All ports should be accessible")
		defineNmapValidPaths(
			[]*pod.Builder{testPod1}, []*pod.Builder{testPod2, testPod3, testPod4, testPod5},
			[]int64{0o77, 0o77, 0o77, 0o77}, testData)

		By("Check ingress traffic to pod1 from other 4 pods. All ports should be blocked")
		defineNmapValidPaths(
			[]*pod.Builder{testPod2, testPod3, testPod4, testPod5}, []*pod.Builder{testPod1},
			[]int64{0o00, 0o00, 0o00, 0o00}, testData)
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

		By("Check egress traffic from pod1 to other 4 pods. All ports should be accessible")
		defineNmapValidPaths(
			[]*pod.Builder{testPod1}, []*pod.Builder{testPod2, testPod3, testPod4, testPod5},
			[]int64{0o77, 0o77, 0o77, 0o77}, testData)

		By("Check ingress traffic to pod1 from other 4 pods. Only pod2 and pod4 can access pod1 on all ports")
		defineNmapValidPaths(
			[]*pod.Builder{testPod2, testPod3, testPod4, testPod5}, []*pod.Builder{testPod1},
			[]int64{0o77, 0o00, 0o77, 0o00}, testData)
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

		By("Check egress traffic from pod1 to other 4 pods. All ports should be accessible")
		defineNmapValidPaths(
			[]*pod.Builder{testPod1}, []*pod.Builder{testPod2, testPod3, testPod4, testPod5},
			[]int64{0o77, 0o77, 0o77, 0o77}, testData)

		By("Check ingress traffic to pod1 from other 4 pods. " +
			"Pod2 can access tcp port 5001 of pod1 over IPv4." +
			"Pod4 can access tcp port 5001 of pod4 over IPv6")
		defineNmapValidPaths(
			[]*pod.Builder{testPod2, testPod3, testPod4, testPod5}, []*pod.Builder{testPod1},
			[]int64{0o40, 0o00, 0o04, 0o00}, testData)
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
		defineNmapValidPaths(
			[]*pod.Builder{testPod1}, []*pod.Builder{testPod2, testPod3, testPod4, testPod5},
			[]int64{0o00, 0o00, 0o00, 0o07}, testData)

		By("Check ingress traffic to pod1 from other 4 pods. " +
			"Pod2 can access tcp ports 5001 & 5002 of pod1 over IPv4. " +
			"Pod4 can access tcp ports 5001 & 5002 of pod1 over both IPv4 & IPv6")
		defineNmapValidPaths(
			[]*pod.Builder{testPod2, testPod3, testPod4, testPod5}, []*pod.Builder{testPod1},
			[]int64{0o60, 0o00, 0o66, 0o00}, testData)
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

func defineAndCreatePodWithBondIf(podName, nsName, net1Name, net2Name string, testData podsData) *pod.Builder {
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
			IPRequest:        append(append([]string{}, testData[podName].IPv4), testData[podName].IPv6),
		},
	}

	tPodBuilder := pod.NewBuilder(APIClient, podName, nsName, NetConfig.CnfNetTestContainer).
		WithNodeSelector(map[string]string{"kubernetes.io/hostname": "worker-0"}).
		WithSecondaryNetwork(netAnnotation).
		WithPrivilegedFlag().
		WithLabel("app", podName)

	for index, protocolPort := range testData[podName].ProtocolPort {
		containerBuilder, err := pod.NewContainerBuilder(strings.ReplaceAll(protocolPort, ":", ""),
			NetConfig.CnfNetTestContainer,
			[]string{"/bin/bash", "-c", fmt.Sprintf("testcmd -listen -interface bond1 -protocol %s -port %s",
				strings.Split(protocolPort, ":")[0], strings.Split(protocolPort, ":")[1])}).
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

func defineNmapValidPaths(sPods, dPods []*pod.Builder, expectedResult []int64, testData podsData) {
	By("Deriving applicable paths between given source and destination pods")

	resultIndex := 0

	for _, sPod := range sPods {
		for _, dPod := range dPods {
			var ipv4host, ipv6host string

			// Convert expectedResult to Binary format string and do zero padding if the length of binary string is not
			// equal to number of available paths. i.e., twice (ipv4 and ipv6) the no. of ports a pod listens to.
			resultBinary := strconv.FormatInt(expectedResult[resultIndex], 2)
			if len(resultBinary) < 2*len(testData[dPod.Object.Name].ProtocolPort) {
				resultBinary = strings.Join([]string{strings.Repeat("0",
					2*len(testData[dPod.Object.Name].ProtocolPort)-len(resultBinary)), resultBinary}, "")
			}

			resultIndex++

			nmapPorts, nmapProtocols := defineNmapPortsProtocols(testData[dPod.Object.Name].ProtocolPort)

			if ipv4host != testData[dPod.Object.Name].IPv4 {
				ipv4host = testData[dPod.Object.Name].IPv4
				runNmapAndValidateResults(sPod, testData[sPod.Object.Name].IPv4, nmapProtocols, nmapPorts,
					strings.Split(ipv4host, "/")[0], resultBinary[0:len(nmapPorts)])
			}

			if ipv6host != testData[dPod.Object.Name].IPv6 {
				ipv6host = testData[dPod.Object.Name].IPv6
				runNmapAndValidateResults(sPod, testData[sPod.Object.Name].IPv6, nmapProtocols, nmapPorts,
					strings.Split(ipv6host, "/")[0], resultBinary[len(nmapPorts):])
			}
		}
	}
}

func defineNmapPortsProtocols(ports []string) ([]string, []string) {
	var nmapPorts, nmapProtocols []string

	for _, port := range ports {
		nmapPort := strings.ToUpper(string(strings.Split(port, ":")[0][0])) + ":" + strings.Split(port, ":")[1]
		nmapProtocol := "-s" + strings.ToUpper(string(strings.Split(port, ":")[0][0]))

		if !slices.Contains(nmapPorts, nmapPort) {
			nmapPorts = append(nmapPorts, nmapPort)
		}

		if !slices.Contains(nmapProtocols, nmapProtocol) {
			nmapProtocols = append(nmapProtocols, nmapProtocol)
		}
	}

	return nmapPorts, nmapProtocols
}

func runNmapAndValidateResults(
	sPod *pod.Builder,
	sourceIP string,
	protocols,
	ports []string,
	targetIP string,
	expectedResult string) {
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

	nmapCmd := fmt.Sprintf("nmap -v -oX - %s -p %s %s", strings.Join(protocols, " "), strings.Join(ports, ","), targetIP)

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
		if string(expectedResult[index]) == "1" {
			// If the expectedResult is 1 then the path/port is expected to be open.
			By(fmt.Sprintf("Path %s/%s =====> %s:%s Expected to Pass\n",
				sPod.Object.Name, sourceIP, targetIP, ports[index]))
			Expect(nmapOutput.Host.Ports.Port[index].State.State).To(Equal("open"),
				fmt.Sprintf("Port is not open as expected. Output: %v", nmapOutput.Host.Ports.Port[index]))
		} else {
			// If the expectedResult is 0 then the path/port is expected to be filtered.
			By(fmt.Sprintf("Path %s/%s =====> %s:%s Expected to Fail\n",
				sPod.Object.Name, sourceIP, targetIP, ports[index]))
			Expect(nmapOutput.Host.Ports.Port[index].State.State).To(SatisfyAny(Equal("open|filtered"), Equal("filtered")),
				fmt.Sprintf("Port is not filtered as expected. Output: %v", nmapOutput.Host.Ports.Port[index]))
		}
	}
}
