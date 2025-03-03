package tests

import (
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/metallb"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netinittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netparam"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/frr"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/metallbenv"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/tsparams"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

var _ = Describe("MetalLB NodeSelector", Ordered, Label(tsparams.LabelBGPTestCases), ContinueOnFailure, func() {
	var (
		nodeAddrList     []string
		addressPool      []string
		frrk8sPods       []*pod.Builder
		worker0NodeLabel []metav1.LabelSelector
		worker1NodeLabel []metav1.LabelSelector
		err              error
	)

	const (
		ipaddressPoolName1    = "ipaddresspool1"
		ipaddressPoolName2    = "ipaddresspool2"
		bgpAdvertisementName1 = "bgpadvertisement1"
		bgpAdvertisementName2 = "bgpadvertisement2"
	)

	BeforeAll(func() {

		By("Getting MetalLb load balancer ip addresses")
		ipv4metalLbIPList, ipv6metalLbIPList, err = metallbenv.GetMetalLbIPByIPStack()
		Expect(err).ToNot(HaveOccurred(), tsparams.MlbAddressListError)

		By("List CNF worker nodes in cluster")
		cnfWorkerNodeList, err = nodes.List(APIClient,
			metav1.ListOptions{LabelSelector: labels.Set(NetConfig.WorkerLabelMap).String()})
		Expect(err).ToNot(HaveOccurred(), "Failed to discover worker nodes")

		By("Selecting worker node for BGP tests")
		workerLabelMap, workerNodeList = setWorkerNodeListAndLabelForBfdTests(cnfWorkerNodeList, metalLbTestsLabel)
		ipv4NodeAddrList, err = nodes.ListExternalIPv4Networks(
			APIClient, metav1.ListOptions{LabelSelector: labels.Set(workerLabelMap).String()})
		Expect(err).ToNot(HaveOccurred(), "Failed to collect external nodes ip addresses")

		err = metallbenv.IsEnvVarMetalLbIPinNodeExtNetRange(ipv4NodeAddrList, ipv4metalLbIPList, nil)
		Expect(err).ToNot(HaveOccurred(), "Failed to validate metalLb exported ip address")

		By("Collecting information before test")
		frrk8sPods, err = pod.List(APIClient, NetConfig.MlbOperatorNamespace, metav1.ListOptions{
			LabelSelector: tsparams.LabelFRRNode,
		})
		Expect(err).ToNot(HaveOccurred(), "Failed to list frrk8s pods")

		By("Listing master nodes")
		masterNodeList, err = nodes.List(APIClient,
			metav1.ListOptions{LabelSelector: labels.Set(NetConfig.ControlPlaneLabelMap).String()})
		Expect(err).ToNot(HaveOccurred(), "Fail to list master nodes")
		Expect(len(masterNodeList)).To(BeNumerically(">", 0),
			"Failed to detect master nodes")

		By("Setting test iteration parameters")
		_, _, _, nodeAddrList, addressPool, _, err =
			metallbenv.DefineIterationParams(
				ipv4metalLbIPList, ipv6metalLbIPList, ipv4NodeAddrList, ipv6NodeAddrList, netparam.IPV4Family)
		Expect(err).ToNot(HaveOccurred(), "Fail to set iteration parameters")

		worker0NodeLabel = []metav1.LabelSelector{
			{MatchLabels: map[string]string{netparam.LabelHostName: workerNodeList[0].Definition.Name}},
		}

		worker1NodeLabel = []metav1.LabelSelector{
			{MatchLabels: map[string]string{netparam.LabelHostName: workerNodeList[1].Definition.Name}},
		}
	})

	AfterAll(func() {
		if len(cnfWorkerNodeList) > 2 {
			By("Remove custom metallb test label from nodes")
			removeNodeLabel(workerNodeList, metalLbTestsLabel)
		}
	})

	BeforeEach(func() {
		By("Creating a new instance of MetalLB Speakers on workers")
		err = metallbenv.CreateNewMetalLbDaemonSetAndWaitUntilItsRunning(tsparams.DefaultTimeout, workerLabelMap)
		Expect(err).ToNot(HaveOccurred(), "Failed to recreate metalLb daemonset")
	})

	AfterEach(func() {
		By("Clean metallb operator and test namespaces")
		resetOperatorAndTestNS()
	})

	Context("Single IPAddressPool", func() {

		It("Advertise a single IPAddressPool with different attributes using the node selector option",
			reportxml.ID("53987"), func() {

				By("Create a single IPAddressPool")
				ipAddressPool := createIPAddressPool(ipaddressPoolName1, addressPool)

				By("Setup test case with services, test pods and bgppeers")
				frrPod0, frrPod1 := setupTestCase(ipAddressPool, ipAddressPool, frrk8sPods)

				By(fmt.Sprintf("Creating a BGPAdvertisement with nodeSelector for bgpPeer1 and LocalPref set to "+
					"200 and community %s", tsparams.NoAdvertiseCommunity))

				setupBgpAdvertisement(bgpAdvertisementName1, tsparams.NoAdvertiseCommunity, ipaddressPoolName1, 200,
					[]string{tsparams.BgpPeerName1}, worker0NodeLabel)

				By(fmt.Sprintf("Creating a BGPAdvertisement with nodeSelector for bgpPeer2 and LocalPref set "+
					"to 100 and community to %s", tsparams.CustomCommunity))

				setupBgpAdvertisement(bgpAdvertisementName2, tsparams.CustomCommunity, ipaddressPoolName1,
					100, []string{tsparams.BgpPeerName2}, worker1NodeLabel)

				verifyBGPConnectivityAndPrefixes(frrPod0, frrPod1, nodeAddrList, addressPool, addressPool)

				By(fmt.Sprintf("Validate BGP Custom Community %s exists with the node selector",
					tsparams.CustomCommunity))
				bgpStatus, err := frr.GetBGPCommunityStatus(frrPod0, tsparams.NoAdvertiseCommunity,
					strings.ToLower(netparam.IPV4Family))
				Expect(err).ToNot(HaveOccurred(), "Failed to collect bgp community status")
				Expect(len(bgpStatus.Routes)).To(Equal(2))

				By(fmt.Sprintf("Validate BGP Custom Community %s exists with the node selector",
					tsparams.CustomCommunity))
				bgpStatus, err = frr.GetBGPCommunityStatus(frrPod1, tsparams.CustomCommunity,
					strings.ToLower(netparam.IPV4Family))
				Expect(err).ToNot(HaveOccurred(), "Failed to collect bgp community status")
				Expect(len(bgpStatus.Routes)).To(Equal(2))
			})
	})

	Context("Dual IPAddressPools", func() {

		var addressPool2 = []string{"4.4.4.1", "4.4.4.240"}

		It("Advertise separate IPAddressPools using the node selector",
			reportxml.ID("53986"), func() {
				By("Create two IPAddressPools")
				ipAddressPool1 := createIPAddressPool(ipaddressPoolName1, addressPool)
				ipAddressPool2 := createIPAddressPool(ipaddressPoolName2, addressPool2)

				By("Setup test case with services, test pods and bgppeers")
				frrPod0, frrPod1 := setupTestCase(ipAddressPool1, ipAddressPool2, frrk8sPods)

				By("Creating a BGPAdvertisement with the nodeSelector to bgppeer1")
				setupBgpAdvertisement(bgpAdvertisementName1, tsparams.NoAdvertiseCommunity, ipaddressPoolName1,
					100, []string{tsparams.BgpPeerName1}, worker0NodeLabel)

				By("Creating a BGPAdvertisement with the nodeSelector to bgppeer2")
				setupBgpAdvertisement(bgpAdvertisementName2, tsparams.CustomCommunity, ipaddressPoolName2,
					200, []string{tsparams.BgpPeerName2}, worker1NodeLabel)

				verifyBGPConnectivityAndPrefixes(frrPod0, frrPod1, nodeAddrList, addressPool, addressPool2)

				By("Validate Local Preference from Frr node0")
				err = frr.ValidateLocalPref(frrPod0, 100, strings.ToLower(netparam.IPV4Family))
				Expect(err).ToNot(HaveOccurred(), "Fail to validate local preference")

				By("Validate Local Preference from Frr node1")
				err = frr.ValidateLocalPref(frrPod1, 200, strings.ToLower(netparam.IPV4Family))
				Expect(err).ToNot(HaveOccurred(), "Fail to validate local preference")

				By(fmt.Sprintf("Validate BGP Community %s exists on received route prefix",
					tsparams.NoAdvertiseCommunity))
				bgpStatus, err := frr.GetBGPCommunityStatus(frrPod0, tsparams.NoAdvertiseCommunity,
					strings.ToLower(netparam.IPV4Family))
				Expect(err).ToNot(HaveOccurred(), "Failed to collect bgp community status")
				Expect(len(bgpStatus.Routes)).To(Equal(1))

				By(fmt.Sprintf("Validate BGP Community %s exists on received route prefix",
					tsparams.CustomCommunity))
				bgpStatus, err = frr.GetBGPCommunityStatus(frrPod1, tsparams.CustomCommunity,
					strings.ToLower(netparam.IPV4Family))
				Expect(err).ToNot(HaveOccurred(), "Failed to collect bgp community status")
				Expect(len(bgpStatus.Routes)).To(Equal(1))
			})
	})
})

func createIPAddressPool(name string, ipPrefix []string) *metallb.IPAddressPoolBuilder {
	ipAddressPool, err := metallb.NewIPAddressPoolBuilder(
		APIClient,
		name,
		NetConfig.MlbOperatorNamespace,
		[]string{fmt.Sprintf("%s-%s", ipPrefix[0], ipPrefix[1])}).Create()
	Expect(err).ToNot(HaveOccurred(), "Failed to create IPAddressPool")

	return ipAddressPool
}

func setupTestCase(ipAddressPool1, ipAddressPool2 *metallb.IPAddressPoolBuilder,
	frrk8sPods []*pod.Builder) (*pod.Builder, *pod.Builder) {
	By("Creating two MetalLB service")

	setupMetalLbService("service-1", netparam.IPV4Family, ipAddressPool1, "Cluster")
	setupMetalLbService("service-2", netparam.IPV4Family, ipAddressPool2, "Cluster")

	By("Creating nginx test pod on worker node 0")
	setupNGNXPod(workerNodeList[0].Definition.Name)

	By("Creating nginx test pod on worker node 1")
	setupNGNXPod(workerNodeList[1].Definition.Name)

	By("Creating External NAD for master FRR pods")
	createExternalNad(tsparams.ExternalMacVlanNADName)

	By("Creating static ip annotation for FRR master 0")

	staticIPAnnotation0 := pod.StaticIPAnnotation(
		tsparams.ExternalMacVlanNADName, []string{fmt.Sprintf("%s/%s", ipv4metalLbIPList[0],
			netparam.IPSubnet24)})

	By("Creating static ip annotation for FRR master 1")

	staticIPAnnotation1 := pod.StaticIPAnnotation(
		tsparams.ExternalMacVlanNADName, []string{fmt.Sprintf("%s/%s", ipv4metalLbIPList[1],
			netparam.IPSubnet24)})

	By("Creating MetalLb configMap for FRR master pods")

	masterConfigMap := createConfigMap(tsparams.LocalBGPASN, ipv4NodeAddrList, false, false)

	By("Create FRR Pod on Master 0")

	frrPod0 := createFrrPod(
		masterNodeList[0].Object.Name, masterConfigMap.Definition.Name, []string{}, staticIPAnnotation0, "frr-master0")

	By("Create FRR Pod on Master 1")

	frrPod1 := createFrrPod(
		masterNodeList[1].Object.Name, masterConfigMap.Definition.Name, []string{}, staticIPAnnotation1, "frr-master1")

	By("Create two BGPPeers")
	createBGPPeerAndVerifyIfItsReady(tsparams.BgpPeerName1, ipv4metalLbIPList[0], "", tsparams.LocalBGPASN,
		false, 0, frrk8sPods)

	createBGPPeerAndVerifyIfItsReady(tsparams.BgpPeerName2, ipv4metalLbIPList[1], "", tsparams.LocalBGPASN,
		false, 0, frrk8sPods)

	return frrPod0, frrPod1
}

func verifyBGPConnectivityAndPrefixes(frrPod0, frrPod1 *pod.Builder, nodeAddrList, addressPool1,
	addressPool2 []string) {
	By("Checking that BGP session is established and up on Frr Master 0")
	verifyMetalLbBGPSessionsAreUPOnFrrPod(frrPod0, removePrefixFromIPList(ipv4NodeAddrList))

	By("Checking that BGP session is established and up on Frr Master 1")
	verifyMetalLbBGPSessionsAreUPOnFrrPod(frrPod1, removePrefixFromIPList(ipv4NodeAddrList))

	By("Validating BGP route prefix on Frr Master 0")
	validatePrefix(frrPod0, netparam.IPV4Family, netparam.IPSubnetInt32,
		removePrefixFromIPList(nodeAddrList), addressPool1)

	By("Validating BGP route prefix on Frr Master 1")
	validatePrefix(frrPod1, netparam.IPV4Family, netparam.IPSubnetInt32,
		removePrefixFromIPList(nodeAddrList), addressPool2)
}
