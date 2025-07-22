package tests

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/configmap"
	"github.com/openshift-kni/eco-goinfra/pkg/metallb"
	"github.com/openshift-kni/eco-goinfra/pkg/metallb/mlbtypesv1beta2"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netinittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netparam"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/frr"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/metallbenv"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/tsparams"
	"gopkg.in/k8snetworkplumbingwg/multus-cni.v4/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

var _ = Describe("BGP remote-dynamicAS", Ordered, Label(tsparams.LabelDynamicRemoteASTestCases),
	ContinueOnFailure, func() {
		var (
			err                          error
			dynamicASiBGP                = "internal"
			dynamicASeBGP                = "external"
			frrExternalMasterIPAddress   = "172.16.0.1"
			hubIPv4ExternalAddresses     = []string{"172.16.0.10", "172.16.0.11"}
			externalAdvertisedIPv4Routes = []string{"192.168.100.0/24", "192.168.200.0/24"}
			externalAdvertisedIPv6Routes = []string{"2001:100::0/64", "2001:200::0/64"}
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
				externalAdvertisedIPv4Routes = []string{"192.168.100.0/24", "192.168.200.0/24"}
			)

			AfterEach(func() {
				By("Clean metallb operator and test namespaces")
				resetOperatorAndTestNS()
			})

			It("Verify the establishment of an eBGP adjacency using neighbor peer remote-as external",
				reportxml.ID("76821"), func() {
					By("Setup test cases with Frr Node AS 64500 and external Frr AS 64501")
					speakerPods, frrPod := setupBGPRemoteASTestCase(hubIPv4ExternalAddresses, externalAdvertisedIPv4Routes,
						externalAdvertisedIPv6Routes, dynamicASeBGP, tsparams.RemoteBGPASN)

					By("Checking that BGP session is established and up")
					verifyMetalLbBGPSessionsAreUPOnFrrPod(frrPod, removePrefixFromIPList(ipv4NodeAddrList))

					By("Validating external FRR AS number received on the FRR nodes")
					Eventually(func() error {
						return frr.ValidateBGPRemoteAS(speakerPods, ipv4metalLbIPList[0], tsparams.RemoteBGPASN)
					}, 60*time.Second, 5*time.Second).Should(Succeed(),
						fmt.Sprintf("The remoteASN does not match the expected AS: %d", tsparams.RemoteBGPASN))
				})

			It("Verify the establishment of an iBGP adjacency using neighbor peer remote-as internal",
				reportxml.ID("76822"), func() {
					By("Setup test cases with Frr Node AS 64500 and external Frr AS 64500")
					speakerPods, frrPod := setupBGPRemoteASTestCase(hubIPv4ExternalAddresses, externalAdvertisedIPv4Routes,
						externalAdvertisedIPv6Routes, dynamicASiBGP, tsparams.LocalBGPASN)

					By("Checking that BGP session is established and up")
					verifyMetalLbBGPSessionsAreUPOnFrrPod(frrPod, removePrefixFromIPList(ipv4NodeAddrList))

					By("Validating external FRR AS number received on the FRR nodes")
					Eventually(func() error {
						return frr.ValidateBGPRemoteAS(speakerPods, ipv4metalLbIPList[0], tsparams.LocalBGPASN)
					}, 60*time.Second, 5*time.Second).Should(Succeed(),
						fmt.Sprintf("The remoteASN does not match the expected AS: %d", tsparams.LocalBGPASN))
				})

			It("Verify the failure to establish a iBGP adjacency with a misconfigured external FRR pod",
				reportxml.ID("76825"), func() {
					By("Setup test cases with Frr Node AS 64500 and misconfigured iBGP external Frr AS 64501")
					speakerPods, frrPod := setupBGPRemoteASTestCase(hubIPv4ExternalAddresses, externalAdvertisedIPv4Routes,
						externalAdvertisedIPv6Routes, dynamicASiBGP, tsparams.RemoteBGPASN)

					By("Checking that BGP session is down")
					verifyMetalLbBGPSessionsAreDownOnFrrPod(frrPod, removePrefixFromIPList(ipv4NodeAddrList))

					By("Validating external FRR AS number received is incorrect and marked as 0 on the FRR nodes")
					Eventually(func() error {
						return frr.ValidateBGPRemoteAS(speakerPods, ipv4metalLbIPList[0], 0)
					}, 60*time.Second, 5*time.Second).Should(Succeed(),
						fmt.Sprintf("The remoteASN does not match the expected AS: %d", 0))
				})
		})

		Context("multi hop", func() {
			var (
				frrNodeSecIntIPv4Addresses = []string{"10.100.100.254", "10.100.100.253"}
				hubPodWorkerNames          = []string{"hub-pod-worker-0", "hub-pod-worker-1"}
			)

			AfterEach(func() {
				By("Removing static routes from the speakers")
				speakerPods, err := pod.List(APIClient, NetConfig.MlbOperatorNamespace, metav1.ListOptions{
					LabelSelector: tsparams.MetalLbDefaultSpeakerLabel,
				})
				Expect(err).ToNot(HaveOccurred(), "Failed to list pods")

				speakerRoutesMap := buildRoutesMapWithSpecificRoutes(speakerPods, []string{ipv4metalLbIPList[0],
					ipv4metalLbIPList[1], frrNodeSecIntIPv4Addresses[0], frrNodeSecIntIPv4Addresses[1]})

				for _, speakerPod := range speakerPods {
					out, err := frr.SetStaticRoute(speakerPod, "del", frrExternalMasterIPAddress, speakerRoutesMap)
					Expect(err).ToNot(HaveOccurred(), out)
				}

				By("Clean metallb operator and test namespaces")
				resetOperatorAndTestNS()
			})

			It("Verify the establishment of a multi-hop iBGP adjacency using neighbor peer remote-as external",
				reportxml.ID("76823"), func() {
					frrPod, speakerPods := setupBGPRemoteASMultiHopTest(ipv4metalLbIPList, hubIPv4ExternalAddresses,
						externalAdvertisedIPv4Routes, externalAdvertisedIPv6Routes, hubPodWorkerNames,
						frrExternalMasterIPAddress, tsparams.LocalBGPASN, false)

					By("Creating a BGP Peer with dynamicASN")
					createBGPPeerWithDynamicASN(frrExternalMasterIPAddress, dynamicASiBGP, false)

					By("Checking that BGP session is established and up")
					verifyMetalLbBGPSessionsAreUPOnFrrPod(frrPod, removePrefixFromIPList(ipv4NodeAddrList))

					By("Validating external FRR AS number received on the FRR nodes")
					Eventually(func() error {
						return frr.ValidateBGPRemoteAS(speakerPods, frrExternalMasterIPAddress, tsparams.LocalBGPASN)
					}, 60*time.Second, 5*time.Second).Should(Succeed(),
						fmt.Sprintf("The remoteASN does not match the expected AS: %d", tsparams.LocalBGPASN))
				})

			It("Verify the establishment of a multi-hop eBGP adjacency using neighbor peer remote-as external",
				reportxml.ID("76824"), func() {
					frrPod, speakerPods := setupBGPRemoteASMultiHopTest(ipv4metalLbIPList, hubIPv4ExternalAddresses,
						externalAdvertisedIPv4Routes, externalAdvertisedIPv6Routes, hubPodWorkerNames,
						frrExternalMasterIPAddress, tsparams.RemoteBGPASN, true)

					By("Creating a BGP Peer with dynamicASN")
					createBGPPeerWithDynamicASN(frrExternalMasterIPAddress, dynamicASeBGP, true)

					By("Checking that BGP session is established and up")
					verifyMetalLbBGPSessionsAreUPOnFrrPod(frrPod, removePrefixFromIPList(ipv4NodeAddrList))

					By("Validating external FRR AS number received on the FRR nodes")
					Eventually(func() error {
						return frr.ValidateBGPRemoteAS(speakerPods, frrExternalMasterIPAddress, tsparams.RemoteBGPASN)
					}, 60*time.Second, 5*time.Second).Should(Succeed(),
						fmt.Sprintf("The remoteASN does not match the expected AS: %d", tsparams.RemoteBGPASN))
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

	speakerPods, err := pod.List(APIClient, NetConfig.MlbOperatorNamespace, metav1.ListOptions{
		LabelSelector: tsparams.MetalLbDefaultSpeakerLabel,
	})
	Expect(err).ToNot(HaveOccurred(), "Failed to list frr pods")

	By("Collect connection information for the Frr external pod")

	frrPod := deployFrrExternalPod(hubIPv4ExternalAddresses, externalAdvertisedIPv4Routes,
		externalAdvertisedIPv6Routes, externalFrrAS)

	By("Creating a BGP Peer with dynamicASN")
	createBGPPeerWithDynamicASN(ipv4metalLbIPList[0], dynamicAS, false)

	return speakerPods, frrPod
}

func setupBGPRemoteASMultiHopTest(ipv4metalLbIPList, hubIPv4ExternalAddresses, externalAdvertisedIPv4Routes,
	externalAdvertisedIPv6Routes, hubPodWorkerName []string, frrExternalMasterIPAddress string, asNumber int,
	eBGP bool) (*pod.Builder, []*pod.Builder) {
	By("Creating a new instance of MetalLB Speakers on workers")

	err := metallbenv.CreateNewMetalLbDaemonSetAndWaitUntilItsRunning(tsparams.DefaultTimeout, workerLabelMap)
	Expect(err).ToNot(HaveOccurred(), "Failed to recreate metalLb daemonset")

	By("Collecting information before test")

	speakerPods, err := pod.List(APIClient, NetConfig.MlbOperatorNamespace, metav1.ListOptions{
		LabelSelector: tsparams.MetalLbDefaultSpeakerLabel,
	})
	Expect(err).ToNot(HaveOccurred(), "Failed to list speaker pods")
	By("Setting test parameters")

	masterClientPodIP, _, _, nodeAddrList, _, _, err :=
		metallbenv.DefineIterationParams(
			ipv4metalLbIPList, ipv6metalLbIPList, ipv4NodeAddrList, ipv6NodeAddrList, netparam.IPV4Family)
	Expect(err).ToNot(HaveOccurred(), "Fail to set iteration parameters")

	By("Creating External NAD for master FRR pod")
	createExternalNad(tsparams.ExternalMacVlanNADName)

	By("Creating External NAD for hub FRR pods")
	createExternalNad(tsparams.HubMacVlanNADName)

	By("Creating static ip annotation for hub0")

	hub0BRstaticIPAnnotation := createStaticIPAnnotations(tsparams.ExternalMacVlanNADName,
		tsparams.HubMacVlanNADName,
		[]string{fmt.Sprintf("%s/24", ipv4metalLbIPList[0])},
		[]string{fmt.Sprintf("%s/24", hubIPv4ExternalAddresses[0])})

	By("Creating static ip annotation for hub1")

	hub1BRstaticIPAnnotation := createStaticIPAnnotations(tsparams.ExternalMacVlanNADName,
		tsparams.HubMacVlanNADName,
		[]string{fmt.Sprintf("%s/24", ipv4metalLbIPList[1])},
		[]string{fmt.Sprintf("%s/24", hubIPv4ExternalAddresses[1])})

	By("Creating MetalLb Hub pod configMap")

	hubConfigMap := createHubConfigMap("hub-node-config")

	By("Creating FRR Hub pod on worker node 0")

	_ = createFrrHubPod(hubPodWorkerName[0],
		workerNodeList[0].Object.Name, hubConfigMap.Definition.Name, []string{}, hub0BRstaticIPAnnotation)

	By("Creating FRR Hub pod on worker node 1")

	_ = createFrrHubPod(hubPodWorkerName[1],
		workerNodeList[1].Object.Name, hubConfigMap.Definition.Name, []string{}, hub1BRstaticIPAnnotation)

	By("Creating configmap and MetalLb Master pod")

	frrPod := createMasterFrrPod(asNumber, frrExternalMasterIPAddress, nodeAddrList, hubIPv4ExternalAddresses,
		externalAdvertisedIPv4Routes, externalAdvertisedIPv6Routes, eBGP)

	By("Adding static routes to the speakers")

	speakerRoutesMap := buildRoutesMapWithSpecificRoutes(speakerPods, ipv4metalLbIPList)

	for _, speakerPod := range speakerPods {
		out, err := frr.SetStaticRoute(speakerPod, "add", masterClientPodIP, speakerRoutesMap)
		Expect(err).ToNot(HaveOccurred(), out)
	}

	return frrPod, speakerPods
}

func buildRoutesMapWithSpecificRoutes(podList []*pod.Builder, nextHopList []string) map[string]string {
	Expect(len(podList)).ToNot(BeZero(), "Pod list is empty")
	Expect(len(nextHopList)).ToNot(BeZero(), "Nexthop IP addresses list is empty")
	Expect(len(nextHopList)).To(BeNumerically(">=", len(podList)),
		fmt.Sprintf("Number of speaker IP addresses[%d] is less than the number of pods[%d]",
			len(nextHopList), len(podList)))

	routesMap := make(map[string]string)

	for _, frrPod := range podList {
		if frrPod.Definition.Spec.NodeName == workerNodeList[0].Definition.Name {
			routesMap[frrPod.Definition.Spec.NodeName] = nextHopList[1]
		} else {
			routesMap[frrPod.Definition.Spec.NodeName] = nextHopList[0]
		}
	}

	return routesMap
}

func createConfigMapWithStaticRoutes(
	bgpAsn int, nodeAddrList, hubIPAddresses, externalAdvertisedIPv4Routes, externalAdvertisedIPv6Routes []string,
	enableMultiHop, enableBFD bool) *configmap.Builder {
	frrBFDConfig := frr.DefineBGPConfigWithStaticRouteAndNetwork(
		bgpAsn, tsparams.LocalBGPASN, hubIPAddresses, externalAdvertisedIPv4Routes,
		externalAdvertisedIPv6Routes, removePrefixFromIPList(nodeAddrList), enableMultiHop, enableBFD)
	configMapData := frr.DefineBaseConfig(tsparams.DaemonsFile, frrBFDConfig, "")
	masterConfigMap, err := configmap.NewBuilder(APIClient, "frr-master-node-config", tsparams.TestNamespaceName).
		WithData(configMapData).Create()
	Expect(err).ToNot(HaveOccurred(), "Failed to create config map")

	return masterConfigMap
}

func createStaticIPAnnotations(internalNADName, externalNADName string, internalIPAddresses,
	externalIPAddresses []string) []*types.NetworkSelectionElement {
	ipAnnotation := pod.StaticIPAnnotation(internalNADName, internalIPAddresses)
	ipAnnotation = append(ipAnnotation,
		pod.StaticIPAnnotation(externalNADName, externalIPAddresses)...)

	return ipAnnotation
}

func createMasterFrrPod(localAS int, frrExternalMasterIPAddress string, ipv4NodeAddrList,
	hubIPAddresses, externalAdvertisedIPv4Routes,
	externalAdvertisedIPv6Routes []string, ebgpMultiHop bool) *pod.Builder {
	masterConfigMap := createConfigMapWithStaticRoutes(localAS, ipv4NodeAddrList, hubIPAddresses,
		externalAdvertisedIPv4Routes, externalAdvertisedIPv6Routes, ebgpMultiHop, false)

	By("Creating static ip annotation for master FRR pod")

	masterStaticIPAnnotation := pod.StaticIPAnnotation(
		tsparams.HubMacVlanNADName, []string{fmt.Sprintf("%s/24", frrExternalMasterIPAddress)})

	By("Creating FRR Master Pod")

	frrPod := createFrrPod(
		masterNodeList[0].Object.Name, masterConfigMap.Definition.Name, []string{}, masterStaticIPAnnotation)

	return frrPod
}
