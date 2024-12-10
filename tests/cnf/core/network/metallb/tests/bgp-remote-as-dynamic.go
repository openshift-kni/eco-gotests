package tests

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/metallb"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"github.com/openshift-kni/eco-goinfra/pkg/schemes/metallb/mlbtypesv1beta2"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netinittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/frr"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/metallbenv"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/tsparams"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

var _ = Describe("BGP remote-dynamicAS", Ordered, Label(tsparams.LabelDynamicRemoteASTestCases),
	ContinueOnFailure, func() {
		var (
			err           error
			dynamicASiBGP = "internal"
			dynamicASeBGP = "external"
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
			By("Remove custom metallb test label from nodes")
			removeNodeLabel(workerNodeList, metalLbTestsLabel)
		})

		Context("single hop", func() {
			var (
				hubIPv4ExternalAddresses     = []string{"172.16.0.10", "172.16.0.11"}
				externalAdvertisedIPv4Routes = []string{"192.168.100.0/24", "192.168.200.0/24"}
				externalAdvertisedIPv6Routes = []string{"2001:100::0/64", "2001:200::0/64"}
			)

			AfterEach(func() {
				By("Clean metallb operator and test namespaces")
				resetOperatorAndTestNS()
			})

			It("Verify the establishment of an eBGP adjacency using neighbor peer remote-as external",
				reportxml.ID("76821"), func() {
					By("Setup test cases with Frr Node AS 64500 and external Frr AS 64501")
					frrk8sPods, frrPod := setupBGPRemoteASTestCase(hubIPv4ExternalAddresses, externalAdvertisedIPv4Routes,
						externalAdvertisedIPv6Routes, dynamicASeBGP, tsparams.RemoteBGPASN)

					By("Checking that BGP session is established and up")
					verifyMetalLbBGPSessionsAreUPOnFrrPod(frrPod, removePrefixFromIPList(ipv4NodeAddrList))

					By("Validating external FRR AS number received on the FRR nodes")
					Eventually(func() error {
						return frr.ValidateBGPRemoteAS(frrk8sPods, ipv4metalLbIPList[0], tsparams.RemoteBGPASN)
					}, 60*time.Second, 5*time.Second).Should(Succeed(),
						fmt.Sprintf("The remoteASN does not match the expected AS: %d", tsparams.RemoteBGPASN))
				})

			It("Verify the establishment of an iBGP adjacency using neighbor peer remote-as internal",
				reportxml.ID("76822"), func() {
					By("Setup test cases with Frr Node AS 64500 and external Frr AS 64500")
					frrk8sPods, frrPod := setupBGPRemoteASTestCase(hubIPv4ExternalAddresses, externalAdvertisedIPv4Routes,
						externalAdvertisedIPv6Routes, dynamicASiBGP, tsparams.LocalBGPASN)

					By("Checking that BGP session is established and up")
					verifyMetalLbBGPSessionsAreUPOnFrrPod(frrPod, removePrefixFromIPList(ipv4NodeAddrList))

					By("Validating external FRR AS number received on the FRR nodes")
					Eventually(func() error {
						return frr.ValidateBGPRemoteAS(frrk8sPods, ipv4metalLbIPList[0], tsparams.LocalBGPASN)
					}, 60*time.Second, 5*time.Second).Should(Succeed(),
						fmt.Sprintf("The remoteASN does not match the expected AS: %d", tsparams.LocalBGPASN))
				})

			It("Verify the failure to establish a iBGP adjacency with a misconfigured external FRR pod",
				reportxml.ID("76825"), func() {
					By("Setup test cases with Frr Node AS 64500 and misconfigured iBGP external Frr AS 64501")
					frrk8sPods, frrPod := setupBGPRemoteASTestCase(hubIPv4ExternalAddresses, externalAdvertisedIPv4Routes,
						externalAdvertisedIPv6Routes, dynamicASiBGP, tsparams.RemoteBGPASN)

					By("Checking that BGP session is down")
					verifyMetalLbBGPSessionsAreDownOnFrrPod(frrPod, removePrefixFromIPList(ipv4NodeAddrList))

					By("Validating external FRR AS number received is incorrect and marked as 0 on the FRR nodes")
					Eventually(func() error {
						return frr.ValidateBGPRemoteAS(frrk8sPods, ipv4metalLbIPList[0], 0)
					}, 60*time.Second, 5*time.Second).Should(Succeed(),
						fmt.Sprintf("The remoteASN does not match the expected AS: %d", 0))
				})
		})
	})

func createBGPPeerWithDynamicASN(peerIP, dynamicASN string, eBgpMultiHop bool) {
	By("Creating BGP Peer")

	bgpPeer := metallb.NewBPGPeerBuilder(APIClient, "testpeer", NetConfig.MlbOperatorNamespace,
		peerIP, tsparams.LocalBGPASN, 0).WithDynamicASN(mlbtypesv1beta2.DynamicASNMode(dynamicASN)).
		WithPassword(tsparams.BGPPassword).WithEBGPMultiHop(eBgpMultiHop)

	_, err := bgpPeer.Create()
	Expect(err).ToNot(HaveOccurred(), "Failed to create BGP peer")
}

func deployFrrExternalPod(hubIPAddresses, externalAdvertisedIPv4Routes,
	externalAdvertisedIPv6Routes []string, localAS int) *pod.Builder {
	By("Creating External NAD")
	createExternalNad(tsparams.ExternalMacVlanNADName)

	By("Creating static ip annotation")

	staticIPAnnotation := pod.StaticIPAnnotation(
		externalNad.Definition.Name, []string{fmt.Sprintf("%s/%s", ipv4metalLbIPList[0], "24")})

	By("Creating MetalLb configMap")

	masterConfigMap := createConfigMapWithStaticRoutes(localAS, ipv4NodeAddrList, hubIPAddresses,
		externalAdvertisedIPv4Routes, externalAdvertisedIPv6Routes, false, false)

	By("Creating FRR Pod")

	frrPod := createFrrPod(
		masterNodeList[0].Object.Name, masterConfigMap.Definition.Name, []string{}, staticIPAnnotation)

	return frrPod
}

func setupBGPRemoteASTestCase(hubIPv4ExternalAddresses, externalAdvertisedIPv4Routes,
	externalAdvertisedIPv6Routes []string, dynamicAS string, externalFrrAS int) ([]*pod.Builder, *pod.Builder) {
	By("Creating a new instance of MetalLB Speakers on workers")

	err := metallbenv.CreateNewMetalLbDaemonSetAndWaitUntilItsRunning(tsparams.DefaultTimeout, workerLabelMap)
	Expect(err).ToNot(HaveOccurred(), "Failed to recreate metalLb daemonset")

	By("Collect connection information for the Frr Node pods")

	frrk8sPods, err := pod.List(APIClient, NetConfig.MlbOperatorNamespace, metav1.ListOptions{
		LabelSelector: tsparams.FRRK8sDefaultLabel,
	})
	Expect(err).ToNot(HaveOccurred(), "Failed to list frr pods")

	By("Collect connection information for the Frr external pod")

	frrPod := deployFrrExternalPod(hubIPv4ExternalAddresses, externalAdvertisedIPv4Routes,
		externalAdvertisedIPv6Routes, externalFrrAS)

	By("Creating a BGP Peer with dynamicASN")
	createBGPPeerWithDynamicASN(ipv4metalLbIPList[0], dynamicAS, false)

	return frrk8sPods, frrPod
}
