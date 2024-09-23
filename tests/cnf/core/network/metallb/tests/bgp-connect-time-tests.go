package tests

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/configmap"
	"github.com/openshift-kni/eco-goinfra/pkg/metallb"
	"github.com/openshift-kni/eco-goinfra/pkg/nad"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"github.com/openshift-kni/eco-goinfra/pkg/service"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netinittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/frr"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/metallbenv"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/tsparams"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

var _ = Describe("FRR", Ordered, Label(tsparams.LabelBGPTestCases), ContinueOnFailure, func() {
	var frrk8sPods []*pod.Builder

	BeforeAll(func() {
		var (
			err error
		)

		By("Getting MetalLb load balancer ip addresses")
		ipv4metalLbIPList, ipv6metalLbIPList, err = metallbenv.GetMetalLbIPByIPStack()
		Expect(err).ToNot(HaveOccurred(), "An error occurred while "+
			"determining the IP addresses from ECO_CNF_CORE_NET_MLB_ADDR_LIST environment variable.")

		By("Getting external nodes ip addresses")
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

	BeforeEach(func() {
		By("Creating a new instance of MetalLB Speakers on workers")
		err := metallbenv.CreateNewMetalLbDaemonSetAndWaitUntilItsRunning(tsparams.DefaultTimeout, workerLabelMap)
		Expect(err).ToNot(HaveOccurred(), "Failed to recreate metalLb daemonset")

		By("Collecting information before test")
		frrk8sPods, err = pod.List(APIClient, NetConfig.MlbOperatorNamespace, metav1.ListOptions{
			LabelSelector: tsparams.FRRK8sDefaultLabel,
		})
		Expect(err).ToNot(HaveOccurred(), "Failed to list frr pods")
	})

	AfterEach(func() {
		By("Cleaning MetalLb operator namespace")
		metalLbNs, err := namespace.Pull(APIClient, NetConfig.MlbOperatorNamespace)
		Expect(err).ToNot(HaveOccurred(), "Failed to pull metalLb operator namespace")
		err = metalLbNs.CleanObjects(
			tsparams.DefaultTimeout,
			metallb.GetBGPPeerGVR(),
			metallb.GetBFDProfileGVR(),
			metallb.GetBGPPeerGVR(),
			metallb.GetBGPAdvertisementGVR(),
			metallb.GetIPAddressPoolGVR(),
			metallb.GetMetalLbIoGVR())
		Expect(err).ToNot(HaveOccurred(), "Failed to remove object's from operator namespace")

		By("Cleaning test namespace")
		err = namespace.NewBuilder(APIClient, tsparams.TestNamespaceName).CleanObjects(
			tsparams.DefaultTimeout,
			pod.GetGVR(),
			service.GetServiceGVR(),
			configmap.GetGVR(),
			nad.GetGVR())
		Expect(err).ToNot(HaveOccurred(), "Failed to clean test namespace")
	})

	It("Verify configuration of a FRR node router peer with the connectTime less than the default of 120 seconds",
		reportxml.ID("74414"), func() {

			By("Creating BGP Peers with 10 second retry connect timer")
			createBGPPeerAndVerifyIfItsReady(ipv4metalLbIPList[0], "", 64500, false,
				10, frrk8sPods)

			By("Validate BGP Peers with 10 second retry connect timer")
			Eventually(func() int {
				// Get the connect time configuration
				connectTimeValue, err := frr.FetchBGPConnectTimeValue(frrk8sPods, ipv4metalLbIPList[0])
				Expect(err).ToNot(HaveOccurred(), "Failed to verify BGP connect time")

				// Return the integer value of ConnectRetryTimer for assertion
				return connectTimeValue
			}, 60*time.Second, 5*time.Second).Should(Equal(10),
				"Failed to fetch BGP connect time")
		})

	It("Verify the retry timers reconnects to a neighbor with a timer connect less then 10s after a BGP tcp reset",
		reportxml.ID("74416"), func() {

			By("Create an external FRR Pod")
			frrPod := createAndDeployFRRPod()

			By("Creating BGP Peers with 10 second retry connect timer")
			createBGPPeerAndVerifyIfItsReady(ipv4metalLbIPList[0], "", 64500, false,
				10, frrk8sPods)

			By("Validate BGP Peers with 10 second retry connect timer")
			Eventually(func() int {
				// Get the connect time configuration
				connectTimeValue, err := frr.FetchBGPConnectTimeValue(frrk8sPods, ipv4metalLbIPList[0])
				Expect(err).ToNot(HaveOccurred(), "Failed to verify BGP connect time")

				// Return the integer value of ConnectRetryTimer for assertion
				return connectTimeValue
			}, 60*time.Second, 5*time.Second).Should(Equal(10),
				"Failed to fetch BGP connect time")

			By("Reset the BGP session ")
			err := frr.ResetBGPConnection(frrPod)
			Expect(err).ToNot(HaveOccurred(), "Failed to reset BGP connection")

			By("Verify that BGP session is re-established and up in less then 10 seconds")
			verifyMaxReConnectTime(frrPod, removePrefixFromIPList(ipv4NodeAddrList), time.Second*10)
		})

	It("Update the timer to less then the default on an existing BGP connection",
		reportxml.ID("74417"), func() {

			By("Creating BGP Peers")
			createBGPPeerAndVerifyIfItsReady(ipv4metalLbIPList[0], "",
				64500, false, 0, frrk8sPods)

			By("Validate BGP Peers with the default retry connect timer")
			Eventually(func() int {
				// Get the connect time configuration
				connectTimeValue, err := frr.FetchBGPConnectTimeValue(frrk8sPods, ipv4metalLbIPList[0])
				Expect(err).ToNot(HaveOccurred(), "Failed to verify BGP connect time")

				// Return the integer value of ConnectRetryTimer for assertion
				return connectTimeValue
			}, 60*time.Second, 5*time.Second).Should(Equal(120),
				"Failed to fetch BGP connect time")

			By("Update the BGP Peers connect timer to 10 seconds")
			bgpPeer, err := metallb.PullBGPPeer(APIClient, "testpeer", NetConfig.MlbOperatorNamespace)
			Expect(err).ToNot(HaveOccurred(), "Failed to find bgp peer")

			_, err = bgpPeer.WithConnectTime(metav1.Duration{Duration: 10 * time.Second}).Update(true)
			Expect(err).ToNot(HaveOccurred(), "Failed to update the bgp peer with a 10s connect timer")

			By("Validate BGP Peers with the default retry connect timer")
			Eventually(func() int {
				// Get the connect time configuration
				connectTimeValue, err := frr.FetchBGPConnectTimeValue(frrk8sPods, ipv4metalLbIPList[0])
				Expect(err).ToNot(HaveOccurred(), "Failed to verify BGP connect time")

				// Return the integer value of ConnectRetryTimer for assertion
				return connectTimeValue
			}, 60*time.Second, 5*time.Second).Should(Equal(10),
				"Failed to fetch BGP connect time")
		})
})

func createAndDeployFRRPod() *pod.Builder {
	By("Creating External NAD")
	createExternalNad()

	By("Creating static ip annotation")

	staticIPAnnotation := pod.StaticIPAnnotation(
		externalNad.Definition.Name, []string{fmt.Sprintf("%s/%s", ipv4metalLbIPList[0], "24")})

	By("Creating MetalLb configMap")

	masterConfigMap := createConfigMap(tsparams.LocalBGPASN, ipv4NodeAddrList, false, false)

	By("Creating FRR Pod")

	frrPod := createFrrPod(
		masterNodeList[0].Object.Name, masterConfigMap.Definition.Name, []string{}, staticIPAnnotation)

	return frrPod
}

func verifyMaxReConnectTime(frrPod *pod.Builder, peerAddrList []string, maxConnectTime time.Duration) {
	for _, peerAddress := range removePrefixFromIPList(peerAddrList) {
		Eventually(frr.BGPNeighborshipHasState,
			maxConnectTime, time.Second).
			WithArguments(frrPod, peerAddress, "Established").Should(
			BeTrue(), "Failed to receive BGP status UP")
	}
}
