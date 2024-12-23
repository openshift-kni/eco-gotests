package tests

import (
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/deployment"
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

const (
	noAdvertise           = "65535:65282"
	customCommunity       = "500:500"
	ipaddressPoolName1    = "ipaddresspool1"
	ipaddressPoolName2    = "ipaddresspool2"
	bgpAdvertisementName1 = "bgpadvertisement1"
	bgpAdvertisementName2 = "bgpadvertisement2"
	bgpPeerName1          = "bgppeer1"
	bgpPeerName2          = "bgppeer2"
)

var _ = Describe("MetalLB NodeSelector", Ordered, Label(tsparams.LabelBGPTestCases), ContinueOnFailure, func() {
	var (
		frrK8WebHookServer = "frr-k8s-webhook-server"
		err                error
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

		By("Listing master nodes")
		masterNodeList, err = nodes.List(APIClient,
			metav1.ListOptions{LabelSelector: labels.Set(NetConfig.ControlPlaneLabelMap).String()})
		Expect(err).ToNot(HaveOccurred(), "Fail to list master nodes")
		Expect(len(masterNodeList)).To(BeNumerically(">", 0),
			"Failed to detect master nodes")
	})

	AfterAll(func() {
		if len(cnfWorkerNodeList) > 2 {
			By("Remove custom metallb test label from nodes")
			removeNodeLabel(workerNodeList, metalLbTestsLabel)
		}
	})

	Context("Single IPAddressPool", func() {

		var (
			nodeAddrList []string
			addressPool  []string
			err          error
		)

		BeforeAll(func() {
			By("Setting test iteration parameters")
			_, _, _, nodeAddrList, addressPool, _, err =
				metallbenv.DefineIterationParams(
					ipv4metalLbIPList, ipv6metalLbIPList, ipv4NodeAddrList, ipv6NodeAddrList, netparam.IPV4Family)
			Expect(err).ToNot(HaveOccurred(), "Fail to set iteration parameters")

			By("Creating a new instance of MetalLB Speakers on workers")
			err = metallbenv.CreateNewMetalLbDaemonSetAndWaitUntilItsRunning(tsparams.DefaultTimeout, workerLabelMap)
			Expect(err).ToNot(HaveOccurred(), "Failed to recreate metalLb daemonset")

			By("Waiting until the new frr-k8s-webhook-server deployment is in Ready state.")
			frrk8sWebhookDeployment, err := deployment.Pull(
				APIClient, frrK8WebHookServer, NetConfig.MlbOperatorNamespace)
			Expect(err).ToNot(HaveOccurred(), "Fail to pull frr-k8s-webhook-server")
			Expect(frrk8sWebhookDeployment.IsReady(30*time.Second)).To(BeTrue(),
				"frr-k8s-webhook-server deployment is not ready")

			By("Selecting worker node for BGP tests")
			workerLabelMap, workerNodeList = setWorkerNodeListAndLabelForBfdTests(cnfWorkerNodeList, metalLbTestsLabel)
			ipv4NodeAddrList, err = nodes.ListExternalIPv4Networks(
				APIClient, metav1.ListOptions{LabelSelector: labels.Set(workerLabelMap).String()})
			Expect(err).ToNot(HaveOccurred(), "Failed to collect external nodes ip addresses")
		})

		AfterEach(func() {
			By("Clean metallb operator and test namespaces")
			resetOperatorAndTestNS()
		})

		It("Advertise a single IPAddressPool with different attributes using the node selector option",
			reportxml.ID("53987"), func() {

				By("Create a single IPAddressPool")
				ipAddressPool := createIPAddressPool(ipaddressPoolName1, addressPool)

				frrPod0, frrPod1 := setupTestCase(ipAddressPool, ipAddressPool)

				By("Creating a BGPAdvertisement with nodeSelector for bgpPeer1")

				workerNodeLabel := []metav1.LabelSelector{
					{MatchLabels: map[string]string{tsparams.LabelHostName: workerNodeList[0].Definition.Name}},
				}

				By("Creating a BGPAdvertisement without the nodeSelector for both bgppeers")
				setupBgpAdvertisementWithNodeSelector(bgpAdvertisementName2, noAdvertise, 100,
					[]string{}, ipAddressPool, workerNodeLabel, false)

				setupBgpAdvertisementWithNodeSelector(bgpAdvertisementName1, customCommunity, 100,
					[]string{bgpPeerName1}, ipAddressPool, workerNodeLabel, true)

				verifyBGPConnectivityAndPrefixes(frrPod0, frrPod1, nodeAddrList, addressPool, addressPool)

				By("Validate BGP Custom Community exists with the node selector")
				output, err := frr.GetBGPCommunityStatus(frrPod0, customCommunity, strings.ToLower(netparam.IPV4Family))
				Expect(err).ToNot(HaveOccurred(), "Failed to collect bgp community status")
				Expect(len(output.Routes)).To(Equal(2))

				By("Validate BGP Custom Community does not exist without the node selector")
				_, err = frr.GetBGPCommunityStatus(frrPod1, customCommunity, strings.ToLower(netparam.IPV4Family))
				Expect(err).To(HaveOccurred(), "Failed to collect bgp community status")
			})
	})

	Context("Dual IPAddressPools", func() {

		var (
			nodeAddrList []string
			addressPool1 []string
			addressPool2 = []string{"4.4.4.1", "4.4.4.240"}
			err          error
		)

		BeforeAll(func() {
			By("Setting test iteration parameters")
			_, _, _, nodeAddrList, addressPool1, _, err =
				metallbenv.DefineIterationParams(
					ipv4metalLbIPList, ipv6metalLbIPList, ipv4NodeAddrList, ipv6NodeAddrList, netparam.IPV4Family)
			Expect(err).ToNot(HaveOccurred(), "Fail to set iteration parameters")

			By("Creating a new instance of MetalLB Speakers on workers")
			err = metallbenv.CreateNewMetalLbDaemonSetAndWaitUntilItsRunning(tsparams.DefaultTimeout, workerLabelMap)
			Expect(err).ToNot(HaveOccurred(), "Failed to recreate metalLb daemonset")

			By("Waiting until the new frr-k8s-webhook-server deployment is in Ready state.")
			frrk8sWebhookDeployment, err := deployment.Pull(
				APIClient, frrK8WebHookServer, NetConfig.MlbOperatorNamespace)
			Expect(err).ToNot(HaveOccurred(), "Fail to pull frr-k8s-webhook-server")
			Expect(frrk8sWebhookDeployment.IsReady(30*time.Second)).To(BeTrue(),
				"frr-k8s-webhook-server deployment is not ready")

			By("Selecting worker node for BGP tests")
			workerLabelMap, workerNodeList = setWorkerNodeListAndLabelForBfdTests(cnfWorkerNodeList, metalLbTestsLabel)
			ipv4NodeAddrList, err = nodes.ListExternalIPv4Networks(
				APIClient, metav1.ListOptions{LabelSelector: labels.Set(workerLabelMap).String()})
			Expect(err).ToNot(HaveOccurred(), "Failed to collect external nodes ip addresses")
		})

		AfterEach(func() {
			By("Clean metallb operator and test namespaces")
			resetOperatorAndTestNS()
		})

		It("Advertise separate IPAddressPools using the node selector",
			reportxml.ID("53986"), func() {
				By("Create two IPAddressPools")
				ipAddressPool1 := createIPAddressPool(ipaddressPoolName1, addressPool1)
				ipAddressPool2 := createIPAddressPool(ipaddressPoolName2, addressPool2)

				frrPod0, frrPod1 := setupTestCase(ipAddressPool1, ipAddressPool2)

				By("Creating a BGPAdvertisement with nodeSelector for bgpPeer1")
				workerNode0Label := []metav1.LabelSelector{
					{MatchLabels: map[string]string{tsparams.LabelHostName: workerNodeList[0].Definition.Name}},
				}

				workerNode1Label := []metav1.LabelSelector{
					{MatchLabels: map[string]string{tsparams.LabelHostName: workerNodeList[1].Definition.Name}},
				}

				By("Creating a BGPAdvertisement with the nodeSelector to bgppeer1")
				setupBgpAdvertisementWithNodeSelector(bgpAdvertisementName1, noAdvertise, 100,
					[]string{bgpPeerName1}, ipAddressPool1, workerNode0Label, true)

				setupBgpAdvertisementWithNodeSelector(bgpAdvertisementName2, customCommunity, 200,
					[]string{bgpPeerName2}, ipAddressPool2, workerNode1Label, true)

				verifyBGPConnectivityAndPrefixes(frrPod0, frrPod1, nodeAddrList, addressPool1, addressPool2)

				By("Validate Local Preference from Frr node0")
				err = frr.ValidateLocalPref(frrPod0, 100, strings.ToLower(netparam.IPV4Family))
				Expect(err).ToNot(HaveOccurred(), "Fail to validate local preference")

				By("Validate Local Preference from Frr node1")
				err = frr.ValidateLocalPref(frrPod1, 200, strings.ToLower(netparam.IPV4Family))
				Expect(err).ToNot(HaveOccurred(), "Fail to validate local preference")

				By("Validate BGP Community exists with the node selector")
				output, err := frr.GetBGPCommunityStatus(frrPod0, noAdvertise, strings.ToLower(netparam.IPV4Family))
				Expect(err).ToNot(HaveOccurred(), "Failed to collect bgp community status")
				Expect(len(output.Routes)).To(Equal(1))

				By("Validate BGP Community does not exist without the node selector")
				output, err = frr.GetBGPCommunityStatus(frrPod1, customCommunity, strings.ToLower(netparam.IPV4Family))
				Expect(err).ToNot(HaveOccurred(), "Failed to collect bgp community status")
				Expect(len(output.Routes)).To(Equal(1))
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

func createBGPPeer(name, peerIP string) error {
	_, err := metallb.NewBPGPeerBuilder(APIClient, name, NetConfig.MlbOperatorNamespace,
		peerIP, tsparams.LocalBGPASN, tsparams.LocalBGPASN).WithPassword(tsparams.BGPPassword).Create()

	return err
}

func setupTestCase(ipAddressPool1, ipAddressPool2 *metallb.IPAddressPoolBuilder) (*pod.Builder, *pod.Builder) {
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
		externalNad.Definition.Name, []string{fmt.Sprintf("%s/%s", ipv4metalLbIPList[0], netparam.IPSubnet24)})

	By("Creating static ip annotation for FRR master 1")

	staticIPAnnotation1 := pod.StaticIPAnnotation(
		externalNad.Definition.Name, []string{fmt.Sprintf("%s/%s", ipv4metalLbIPList[1], netparam.IPSubnet24)})

	By("Creating MetalLb configMap for FRR master 0")

	masterConfigMap0 := createConfigMap(tsparams.LocalBGPASN, ipv4NodeAddrList, false, false)

	By("Creating MetalLb configMap for FRR master 1")

	masterConfigMap1 := createConfigMap(tsparams.LocalBGPASN, ipv4NodeAddrList, false, false)

	By("Create FRR Pod on Master 0")

	frrPod0 := createFrrPod(
		masterNodeList[0].Object.Name, masterConfigMap0.Definition.Name, []string{}, staticIPAnnotation0, "frr-master0")

	By("Create FRR Pod on Master 1")

	frrPod1 := createFrrPod(
		masterNodeList[1].Object.Name, masterConfigMap1.Definition.Name, []string{}, staticIPAnnotation1, "frr-master1")

	By("Create two BGPPeers")

	err := createBGPPeer(bgpPeerName1, ipv4metalLbIPList[0])
	Expect(err).ToNot(HaveOccurred(), "Fail to create BGPPeer1")

	err = createBGPPeer(bgpPeerName2, ipv4metalLbIPList[1])
	Expect(err).ToNot(HaveOccurred(), "Fail to create BGPPeer2")

	return frrPod0, frrPod1
}

func verifyBGPConnectivityAndPrefixes(frrPod0, frrPod1 *pod.Builder, nodeAddrList, addressPool1,
	addressPool2 []string) {
	By("Checking that BGP session is established and up on Frr Master 0")
	verifyMetalLbBGPSessionsAreUPOnFrrPod(frrPod0, removePrefixFromIPList(ipv4NodeAddrList))

	By("Checking that BGP session is established and up on Frr Master 1")
	verifyMetalLbBGPSessionsAreUPOnFrrPod(frrPod1, removePrefixFromIPList(ipv4NodeAddrList))

	By("Validating BGP route prefix on Frr Master 0")
	validatePrefix(frrPod0, netparam.IPV4Family, removePrefixFromIPList(nodeAddrList), addressPool1, 32)

	By("Validating BGP route prefix on Frr Master 1")
	validatePrefix(frrPod1, netparam.IPV4Family, removePrefixFromIPList(nodeAddrList), addressPool2, 32)
}
