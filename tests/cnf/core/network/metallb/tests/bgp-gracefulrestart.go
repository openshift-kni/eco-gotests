package tests

import (
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift-kni/eco-goinfra/pkg/metallb"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"

	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/ipaddr"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netinittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netparam"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/cmd"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/frr"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/metallbenv"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/tsparams"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

var _ = Describe("BGP Graceful Restart", Ordered, Label(tsparams.LabelGRTestCases), ContinueOnFailure, func() {
	var (
		ipAddressPool     *metallb.IPAddressPoolBuilder
		frrPod            *pod.Builder
		bgpPeer           *metallb.BGPPeerBuilder
		ipv4MetalLbIPList []string
		err               error
		workerLabelMap    map[string]string
		addressPool       = []string{"4.4.4.1", "4.4.4.240"}
		masterNodeList    []*nodes.Builder
	)

	BeforeAll(func() {
		By("Getting MetalLb load balancer ip addresses")
		ipv4MetalLbIPList, _, err = metallbenv.GetMetalLbIPByIPStack()
		Expect(err).ToNot(HaveOccurred(), tsparams.MlbAddressListError)

		if len(ipv4MetalLbIPList) < 2 {
			Skip("MetalLb GR tests require 2 ip addresses. Please check ECO_CNF_CORE_NET_MLB_ADDR_LIST env var")
		}

		By("Getting external nodes ip addresses")
		cnfWorkerNodeList, err := nodes.List(APIClient,
			metav1.ListOptions{LabelSelector: labels.Set(NetConfig.WorkerLabelMap).String()})
		Expect(err).ToNot(HaveOccurred(), "Failed to discover worker nodes")

		By("Selecting worker node for BGP GR tests")
		workerLabelMap, workerNodeList = setWorkerNodeListAndLabelForBfdTests(cnfWorkerNodeList, metalLbTestsLabel)

		ipv4NodeAddrList, err = nodes.ListExternalIPv4Networks(
			APIClient, metav1.ListOptions{LabelSelector: labels.Set(workerLabelMap).String()})
		Expect(err).ToNot(HaveOccurred(), "Failed to collect external nodes ip addresses")

		err = metallbenv.IsEnvVarMetalLbIPinNodeExtNetRange(ipv4NodeAddrList, ipv4MetalLbIPList, nil)
		Expect(err).ToNot(HaveOccurred(), "Failed to validate metalLb exported ip address")
		createExternalNad(tsparams.ExternalMacVlanNADName)

		By("Listing master nodes")
		masterNodeList, err = nodes.List(APIClient,
			metav1.ListOptions{LabelSelector: labels.Set(NetConfig.ControlPlaneLabelMap).String()})
		Expect(err).ToNot(HaveOccurred(), "Fail to list master nodes")
	})

	AfterEach(func() {
		By("Cleaning MetalLb operator namespace")
		resetOperatorAndTestNS()
	})

	Context("without bfd", func() {
		BeforeEach(func() {
			By("Creating a new instance of MetalLB Speakers on workers")
			err = metallbenv.CreateNewMetalLbDaemonSetAndWaitUntilItsRunning(tsparams.DefaultTimeout, workerLabelMap)
			Expect(err).ToNot(HaveOccurred(), "Failed to recreate metalLb daemonset")

			By("Creating External NAD")
			createExternalNad(tsparams.ExternalMacVlanNADName)

			By("Creating static ip annotation")
			staticIPAnnotation := pod.StaticIPAnnotation(
				externalNad.Definition.Name, []string{fmt.Sprintf("%s/%s", ipv4MetalLbIPList[0], "24")})

			By("Creating MetalLb configMap")
			masterConfigMap := createConfigMap(tsparams.LocalBGPASN, ipv4NodeAddrList, false, false)

			By("Creating FRR Pod")
			frrPod = createFrrPod(
				masterNodeList[0].Object.Name, masterConfigMap.Definition.Name, []string{}, staticIPAnnotation)

			By("Configuring BGP peers without GracefulRestart")
			bgpPeer, err = metallb.NewBPGPeerBuilder(APIClient, "testpeer", NetConfig.MlbOperatorNamespace,
				ipv4MetalLbIPList[0], tsparams.LocalBGPASN, tsparams.LocalBGPASN).WithPassword(tsparams.BGPPassword).Create()
			Expect(err).ToNot(HaveOccurred(), "Fail to create bgpPeer")

			By("Checking that BGP sessions are established and up")
			verifyMetalLbBGPSessionsAreUPOnFrrPod(frrPod, ipv4NodeAddrList)

			By("Creating static ip annotation")
			pod.StaticIPAnnotation(
				externalNad.Definition.Name, []string{fmt.Sprintf("%s/%s", ipv4MetalLbIPList[0], netparam.IPSubnet24)})

			By("Creating an IPAddressPool and BGPAdvertisement")
			ipAddressPool = setupBgpAdvertisement(addressPool, 32)

			By("Creating nginx test pod on worker node")
			setupNGNXPod(workerNodeList[0].Definition.Name)
		})

		It("Verify basic functionality with externalTrafficPolicy set to Local", reportxml.ID("77074"), func() {
			gracefulRestartTest("Local", addressPool[0], ipv4MetalLbIPList[0], frrPod, bgpPeer, ipAddressPool, workerLabelMap)
		})

		It("Verify basic functionality with externalTrafficPolicy set to Cluster", reportxml.ID("77077"), func() {
			gracefulRestartTest("Cluster", addressPool[0], ipv4MetalLbIPList[0], frrPod, bgpPeer, ipAddressPool, workerLabelMap)

			By("Verifying that the routes have been removed after the gracefulRestart timeout")
			waitForGracefulRestartTimeout(frrPod, ipv4NodeAddrList[0])
			verifyRoutesAmount(frrPod, 0)
		})

	})
})

func verifyRoutesAmount(masterNodeFRRPod *pod.Builder, amountRoutes int) {
	Eventually(func() int {
		bgpStatus, err := frr.GetBGPStatus(masterNodeFRRPod, strings.ToLower(netparam.IPV4Family))
		Expect(err).ToNot(HaveOccurred(), "Failed to get BGP status from FRR pod")

		return len(bgpStatus.Routes)
	}, 1*time.Minute, tsparams.DefaultRetryInterval).Should(Equal(amountRoutes),
		fmt.Sprintf("Expected %d routes, but the actual count did not match", amountRoutes))
}

func gracefulRestartTest(
	extTrafficPolicy corev1.ServiceExternalTrafficPolicyType,
	destIPaddress,
	ipv4metalLbIP string,
	frrPod *pod.Builder,
	bgpPeer *metallb.BGPPeerBuilder,
	ipAddressPool *metallb.IPAddressPoolBuilder,
	workerLabelMap map[string]string) {
	By("Creating a MetalLB service")
	setupMetalLbService(netparam.IPV4Family, ipAddressPool, extTrafficPolicy)

	By("Checking that routes have been added to the BGP table")
	verifyRoutesAmount(frrPod, 1)

	By("Running http check")
	validateCurlToNginxPod(frrPod, ipv4metalLbIP, destIPaddress)

	By("Delete metalb CR to  remove all frr-k8s pods and simulate GR scenario")
	removeMetalLbCR()

	By("Checking that BGP sessions are down")
	verifyMetalLbBGPSessionsAreDown(frrPod, ipv4NodeAddrList)

	By("Checking that routes have been remove from the BGP table")
	verifyRoutesAmount(frrPod, 0)

	By("Running http check")
	validateCurlFailsToNginxPod(frrPod, ipv4metalLbIP, destIPaddress)

	By("Creating a new instance of MetalLB Speakers on workers")

	err := metallbenv.CreateNewMetalLbDaemonSetAndWaitUntilItsRunning(tsparams.DefaultTimeout, workerLabelMap)
	Expect(err).ToNot(HaveOccurred(), "Failed to recreate metalLb daemonset")

	By("Configuring BGP peers with GR enabled")

	_, err = bgpPeer.WithGracefulRestart(true).Update(false)
	Expect(err).ToNot(HaveOccurred(), "Failed to configure GracefulRestart")

	By("Checking that BGP sessions are established and up")
	verifyMetalLbBGPSessionsAreUPOnFrrPod(frrPod, ipv4NodeAddrList)

	By("Checking that BGP GracefulRestart is enabled on the neighbors")
	verifyGREnabledOnNeighbors(frrPod, ipv4NodeAddrList)

	By("Checking that routes exists in the bgp table")
	verifyRoutesAmount(frrPod, 1)

	By("Running http check")
	validateCurlToNginxPod(frrPod, ipv4metalLbIP, destIPaddress)

	By("Delete metalb CR to remove all frr-k8s pods and simulate GR scenario")
	removeMetalLbCR()

	By("Checking that BGP sessions are down")
	verifyMetalLbBGPSessionsAreDown(frrPod, ipv4NodeAddrList)

	By("Checking that routes exists in the bgp table")
	verifyRoutesAmount(frrPod, 1)

	By("Running http check")

	_, err = cmd.Curl(frrPod, ipv4metalLbIP, destIPaddress, netparam.IPV4Family, tsparams.FRRSecondContainerName)
	Expect(err).ToNot(HaveOccurred(), "Unexpectedly succeeded in curling the Nginx pod")
}

func verifyMetalLbBGPSessionsAreDown(frrPod *pod.Builder, peerAddrList []string) {
	for _, peerAddress := range removePrefixFromIPList(peerAddrList) {
		Eventually(frr.BGPNeighborshipHasState,
			30*time.Second, tsparams.DefaultRetryInterval).
			WithArguments(frrPod, peerAddress, "Established").ShouldNot(
			BeTrue(), "Failed to receive BGP status Down")
	}
}

func verifyGREnabledOnNeighbors(frrPod *pod.Builder, peerAddrList []string) {
	for _, peerAddress := range removePrefixFromIPList(peerAddrList) {
		grStatus, err := frr.GetGracefulRestartStatus(frrPod, peerAddress)

		Expect(err).ToNot(HaveOccurred(), "Failed to get GracefulRestart status")
		Expect(grStatus.RemoteGrMode).To(Or(Equal("Helper"), Equal("Restart")),
			fmt.Sprintf("GracefulRestart is not enabled on the peer %s", peerAddress))
	}
}

func waitForGracefulRestartTimeout(frrPod *pod.Builder, peerAddress string) {
	grStatus, err := frr.GetGracefulRestartStatus(frrPod, ipaddr.RemovePrefix(peerAddress))
	Expect(err).ToNot(HaveOccurred(), "Failed to get GracefulRestart status")
	time.Sleep(time.Duration(grStatus.Timers.RestartTimerRemaining) * time.Second)
}

func validateCurlToNginxPod(frrPod *pod.Builder, srcIP, destIP string) {
	Eventually(func() error {
		_, err := cmd.Curl(
			frrPod, srcIP, destIP, netparam.IPV4Family, tsparams.FRRSecondContainerName)

		return err
	}, 40*time.Second, tsparams.DefaultRetryInterval).Should(BeNil(), "Failed to curl nginx pod")
}

func validateCurlFailsToNginxPod(frrPod *pod.Builder, srcIP, destIP string) {
	Consistently(func() error {
		_, err := cmd.Curl(
			frrPod, srcIP, destIP, netparam.IPV4Family, tsparams.FRRSecondContainerName)

		return err
	}, 30*time.Second, tsparams.DefaultRetryInterval).ShouldNot(BeNil(), "Expected curl to fail, but it succeeded")
}

func removeMetalLbCR() {
	metalLbNs, err := namespace.Pull(APIClient, NetConfig.MlbOperatorNamespace)
	Expect(err).ToNot(HaveOccurred(), "Failed to pull metalLb operator namespace")
	err = metalLbNs.CleanObjects(tsparams.DefaultTimeout, metallb.GetMetalLbIoGVR())
	Expect(err).ToNot(HaveOccurred(), "Failed to remove MetalLB object from operator namespace")
}
