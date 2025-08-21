package tests

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/configmap"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/metallb"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/nad"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/namespace"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/nodes"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/pod"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/reportxml"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/service"
	netcmd "github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/core/network/internal/cmd"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/core/network/internal/define"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/core/network/internal/frrconfig"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/core/network/internal/ipaddr"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/core/network/internal/netenv"
	. "github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/core/network/internal/netinittools"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/core/network/internal/netparam"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/core/network/metallb/internal/cmd"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/core/network/metallb/internal/frr"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/core/network/metallb/internal/metallbenv"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/core/network/metallb/internal/tsparams"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/internal/cluster"
	"k8s.io/apimachinery/pkg/labels"
)

var _ = Describe("BFD", Ordered, Label(tsparams.LabelBFDTestCases), ContinueOnFailure, func() {

	BeforeAll(func() {
		var err error
		By("Getting MetalLb load balancer ip addresses")
		ipv4metalLbIPList, ipv6metalLbIPList, err = metallbenv.GetMetalLbIPByIPStack()
		Expect(err).ToNot(HaveOccurred(), tsparams.MlbAddressListError)

		if len(ipv4metalLbIPList) < 2 {
			Skip("MetalLb BFD tests require 2 ip addresses. Please check ECO_CNF_CORE_NET_MLB_ADDR_LIST env var")
		}

		By("Getting external nodes ip addresses")
		cnfWorkerNodeList, err = nodes.List(APIClient,
			metav1.ListOptions{LabelSelector: labels.Set(NetConfig.WorkerLabelMap).String()})
		Expect(err).ToNot(HaveOccurred(), "Failed to discover worker nodes")

		By("Selecting worker node for BFD tests")
		workerLabelMap, workerNodeList = setWorkerNodeListAndLabelForBfdTests(cnfWorkerNodeList, metalLbTestsLabel)

		ipv4NodeAddrList, err = nodes.ListExternalIPv4Networks(
			APIClient, metav1.ListOptions{LabelSelector: labels.Set(workerLabelMap).String()})
		Expect(err).ToNot(HaveOccurred(), "Failed to collect external nodes ip addresses")

		By("Creating a new instance of MetalLB Speakers on workers")
		err = metallbenv.CreateNewMetalLbDaemonSetAndWaitUntilItsRunning(tsparams.DefaultTimeout, workerLabelMap)
		Expect(err).ToNot(HaveOccurred(), "Failed to recreate metalLb daemonset")

		err = metallbenv.IsEnvVarMetalLbIPinNodeExtNetRange(ipv4NodeAddrList, ipv4metalLbIPList, nil)
		Expect(err).ToNot(HaveOccurred(), "Failed to validate metalLb exported ip address")
		err = define.CreateExternalNad(APIClient, frrconfig.ExternalMacVlanNADName, tsparams.TestNamespaceName)
		Expect(err).ToNot(HaveOccurred(), "Failed to create a network-attachment-definition")
	})

	Context("single hop", Label("singlehop"), func() {
		BeforeEach(func() {
			By("Collect running metallb bgp speakers")
			frrk8sPods, err := pod.List(APIClient, NetConfig.MlbOperatorNamespace, metav1.ListOptions{
				LabelSelector: tsparams.FRRK8sDefaultLabel,
			})
			Expect(err).ToNot(HaveOccurred(), "Failed to list pods")
			bfdProfile := createBFDProfileAndVerifyIfItsReady(frrk8sPods)

			createBGPPeerAndVerifyIfItsReady(tsparams.BGPTestPeer,
				ipv4metalLbIPList[0], bfdProfile.Definition.Name, tsparams.RemoteBGPASN, false, 0,
				frrk8sPods)

			By("Creating MetalLb configMap")
			bfdConfigMap := createConfigMap(tsparams.RemoteBGPASN, ipv4NodeAddrList, false, true)

			By("Creating static ip annotation")
			staticIPAnnotation := pod.StaticIPAnnotation(
				frrconfig.ExternalMacVlanNADName, []string{fmt.Sprintf("%s/24", ipv4metalLbIPList[0])})

			By("Listing control-plane nodes")
			masterNodeList, err := nodes.List(APIClient,
				metav1.ListOptions{LabelSelector: labels.Set(NetConfig.ControlPlaneLabelMap).String()})
			Expect(err).ToNot(HaveOccurred(), "Failed to discover control-plane nodes")

			By("Creating FRR Pod with network and IP address")
			frrPod := createFrrPod(
				masterNodeList[0].Object.Name, bfdConfigMap.Object.Name, []string{}, staticIPAnnotation)

			By("Checking that BGP and BFD sessions are established and up")
			verifyMetalLbBFDAndBGPSessionsAreUPOnFrrPod(frrPod, ipv4NodeAddrList)

			By("Set Local GW mode")
			setLocalGWMode(false)
		})

		It("basic functionality should provide fast link failure detection", reportxml.ID("47188"), func() {
			scaleDownMetalLbSpeakers()
			testBFDFailOver()
			testBFDFailBack()
		})

		It("provides Prometheus BFD metrics", reportxml.ID("47187"), func() {
			mlbNs, err := namespace.Pull(APIClient, NetConfig.MlbOperatorNamespace)
			Expect(err).ToNot(HaveOccurred(),
				fmt.Sprintf("Failed to pull %s namespace", NetConfig.MlbOperatorNamespace))
			_, err = mlbNs.WithLabel(tsparams.PrometheusMonitoringLabel, "true").Update()
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to redefine %s namespace with the label %s",
				NetConfig.MlbOperatorNamespace, tsparams.PrometheusMonitoringLabel))

			frrk8sPods, err := pod.List(APIClient, NetConfig.MlbOperatorNamespace, metav1.ListOptions{
				LabelSelector: tsparams.FRRK8sDefaultLabel})
			Expect(err).ToNot(HaveOccurred(), "Failed to list MetalLB speaker pods")

			prometheusPods, err := pod.List(APIClient, NetConfig.PrometheusOperatorNamespace, metav1.ListOptions{
				LabelSelector: tsparams.PrometheusMonitoringPodLabel,
			})
			Expect(err).ToNot(HaveOccurred(), "Failed to list prometheus pods")

			verifyMetricPresentInPrometheus(frrk8sPods, prometheusPods[0], "frrk8s_bfd_")
		})

		AfterEach(func() {
			By("Removing label from Workers")
			removeNodeLabel(workerNodeList, tsparams.MetalLbSpeakerLabel)

			By("Resetting MetalLB speakerNodeSelector to default value")
			metalLbIo, err := metallb.Pull(APIClient, tsparams.MetalLbIo, NetConfig.MlbOperatorNamespace)
			Expect(err).ToNot(HaveOccurred(), "Failed to pull MetalLB object")
			_, err = metalLbIo.RemoveLabel("metal").
				WithSpeakerNodeSelector(workerLabelMap).Update(false)
			Expect(err).ToNot(HaveOccurred(), "Failed to reset MetalLB SpeakerNodeSelector to default value")

			By("Cleaning MetalLb operator namespace")
			metalLbNs, err := namespace.Pull(APIClient, NetConfig.MlbOperatorNamespace)
			Expect(err).ToNot(HaveOccurred(), "Failed to pull metalLb operator namespace")
			err = metalLbNs.CleanObjects(tsparams.DefaultTimeout, metallb.GetBGPPeerGVR(), metallb.GetBFDProfileGVR())
			Expect(err).ToNot(HaveOccurred(), "Failed to remove object's from operator namespace")

			By("Cleaning test namespace")
			err = namespace.NewBuilder(APIClient, tsparams.TestNamespaceName).
				CleanObjects(tsparams.DefaultTimeout, pod.GetGVR(), configmap.GetGVR())
			Expect(err).ToNot(HaveOccurred(), "Failed to clean objects from test namespace")

		})
	})

	Context("multihop", Label("multihop"), func() {
		speakerRoutesMap := make(map[string]string)

		BeforeEach(func() {
			By("Collecting information before test")
			frrk8sPods, err := pod.List(APIClient, NetConfig.MlbOperatorNamespace, metav1.ListOptions{
				LabelSelector: tsparams.FRRK8sDefaultLabel,
			})
			Expect(err).ToNot(HaveOccurred(), "Failed to list speaker pods")

			speakerRoutesMap, err = buildRoutesMap(frrk8sPods, ipv4metalLbIPList)
			Expect(err).ToNot(HaveOccurred(), "Failed to build speaker route map")

			By("Configuring Local GW mode")
			setLocalGWMode(true)
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
				metallb.GetIPAddressPoolGVR())
			Expect(err).ToNot(HaveOccurred(), "Failed to remove object's from operator namespace")

			By("Removing static routes from the speakers")
			frrk8sPods, err := pod.List(APIClient, NetConfig.MlbOperatorNamespace, metav1.ListOptions{
				LabelSelector: tsparams.FRRK8sDefaultLabel,
			})
			Expect(err).ToNot(HaveOccurred(), "Failed to list pods")
			for _, frrk8sPod := range frrk8sPods {
				out, err := netenv.SetStaticRoute(frrk8sPod, "del", "172.16.0.1",
					frrconfig.ContainerName, speakerRoutesMap)
				Expect(err).ToNot(HaveOccurred(), out)
			}

			By("Removing label from Workers")
			removeNodeLabel(workerNodeList, tsparams.MetalLbSpeakerLabel)

			By("Resetting MetalLB speakerNodeSelector to default value")
			metalLbIo, err := metallb.Pull(APIClient, tsparams.MetalLbIo, NetConfig.MlbOperatorNamespace)
			Expect(err).ToNot(HaveOccurred(), "Failed to pull metallb object")
			_, err = metalLbIo.RemoveLabel("metal").
				WithSpeakerNodeSelector(workerLabelMap).Update(false)
			Expect(err).ToNot(HaveOccurred(), "Failed to reset metallb SpeakerNodeSelector to default value")

			By("Cleaning test namespace")
			err = namespace.NewBuilder(APIClient, tsparams.TestNamespaceName).CleanObjects(
				tsparams.DefaultTimeout,
				pod.GetGVR(),
				service.GetGVR(),
				configmap.GetGVR(),
				nad.GetGVR())
			Expect(err).ToNot(HaveOccurred(), "Failed to clean test namespace")
		})

		DescribeTable("should provide fast link failure detection", reportxml.ID("47186"),
			func(bgpProtocol, ipStack string, externalTrafficPolicy corev1.ServiceExternalTrafficPolicyType) {
				err := define.CreateExternalNad(APIClient, frrconfig.ExternalMacVlanNADName, tsparams.TestNamespaceName)
				Expect(err).ToNot(HaveOccurred(), "Failed to create a network-attachment-definition")

				By("Verifying that speaker route map is not empty")
				Expect(speakerRoutesMap).ToNot(BeNil(), "Speaker route map is empty")

				By("Setting test iteration parameters")
				masterClientPodIP, subMast, mlbAddressList, nodeAddrList, addressPool, frrMasterIPs, err :=
					metallbenv.DefineIterationParams(
						ipv4metalLbIPList, ipv6metalLbIPList, ipv4NodeAddrList, ipv6NodeAddrList, ipStack)

				if err != nil {
					Skip(err.Error())
				}

				By("Collecting running MetalLB speakers")
				frrk8sPods, err := pod.List(APIClient, NetConfig.MlbOperatorNamespace, metav1.ListOptions{
					LabelSelector: tsparams.FRRK8sDefaultLabel,
				})
				Expect(err).ToNot(HaveOccurred(), "Failed to list metalLb speaker pods")
				bfdProfile := createBFDProfileAndVerifyIfItsReady(frrk8sPods)

				neighbourASN := uint32(tsparams.LocalBGPASN)
				var eBgpMultiHop bool
				if bgpProtocol == tsparams.EBGPProtocol {
					neighbourASN = tsparams.RemoteBGPASN
					eBgpMultiHop = true
				}
				createBGPPeerAndVerifyIfItsReady(tsparams.BGPTestPeer,
					masterClientPodIP, bfdProfile.Definition.Name, neighbourASN, eBgpMultiHop, 0, frrk8sPods)

				By("Creating an IPAddressPool and BGPAdvertisement for bfd tests")
				ipAddressPool := setupBgpAdvertisementAndIPAddressPool(addressPool)

				By("Creating a MetalLB service")
				setupMetalLbService("service-1", ipStack, ipAddressPool, externalTrafficPolicy)

				By("Creating nginx test pod on worker node")
				setupNGNXPod(workerNodeList[0].Definition.Name)

				By("Creating internal NAD")
				masterBridgePlugin, err := nad.NewMasterBridgePlugin("internalnad", "br0").
					WithIPAM(nad.IPAMStatic()).GetMasterPluginConfig()
				Expect(err).ToNot(HaveOccurred(), "Failed to create master bridge plugin setting")
				bridgeNad, err := nad.NewBuilder(APIClient, "internal", tsparams.TestNamespaceName).
					WithMasterPlugin(masterBridgePlugin).Create()
				Expect(err).ToNot(HaveOccurred(), "Failed to create internal NAD")

				By("Discovering Master nodes")
				masterNodes, err := nodes.List(APIClient,
					metav1.ListOptions{LabelSelector: labels.Set(NetConfig.ControlPlaneLabelMap).String()})
				Expect(err).ToNot(HaveOccurred(), "Failed to discover control-plane nodes")

				By("Creating FRR pod one on master node")
				createFrrPodOnMasterNodeAndWaitUntilRunning("frronmaster1",
					mlbAddressList[0], subMast, frrMasterIPs[0], bridgeNad.Definition.Name,
					masterNodes[0].Object.Name, addressPool[0], nodeAddrList[0])

				By("Creating FRR pod two on master node")
				createFrrPodOnMasterNodeAndWaitUntilRunning("frronmaster2",
					mlbAddressList[1], subMast, frrMasterIPs[1], bridgeNad.Definition.Name,
					masterNodes[0].Object.Name, addressPool[0], nodeAddrList[1])

				By("Creating client pod config map")
				masterConfigMap := createConfigMap(int(neighbourASN), nodeAddrList, eBgpMultiHop, true)

				By("Creating FRR pod in the test namespace")
				frrPod := createFrrPod(
					masterNodes[0].Object.Name,
					masterConfigMap.Object.Name,
					[]string{},
					pod.StaticIPAnnotation(bridgeNad.Definition.Name, []string{fmt.Sprintf("%s/%s", masterClientPodIP, subMast)}))

				// Add static routes from client towards Speaker via router internal IPs
				for index, workerAddress := range netcmd.RemovePrefixFromIPList(nodeAddrList) {
					buffer, err := cmd.SetRouteOnPod(frrPod, workerAddress, frrMasterIPs[index])
					Expect(err).ToNot(HaveOccurred(), buffer.String())
				}
				By("Adding static routes to the speakers")
				for _, frrk8sPod := range frrk8sPods {
					out, err := netenv.SetStaticRoute(frrk8sPod, "add", masterClientPodIP,
						frrconfig.ContainerName, speakerRoutesMap)
					Expect(err).ToNot(HaveOccurred(), out)
				}

				By("Checking that BGP and BFD sessions are established and up")
				verifyMetalLbBFDAndBGPSessionsAreUPOnFrrPod(frrPod, netcmd.RemovePrefixFromIPList(nodeAddrList))

				By("Running http check")
				httpOutput, err := cmd.Curl(frrPod, masterClientPodIP, addressPool[0], ipStack, tsparams.FRRSecondContainerName)
				Expect(err).ToNot(HaveOccurred(), httpOutput)

				scaleDownMetalLbSpeakers()
				testBFDFailOver()

				By("Running http check after fail-over")
				httpOutput, err = cmd.Curl(frrPod, masterClientPodIP, addressPool[0], ipStack, tsparams.FRRSecondContainerName)
				// If externalTrafficPolicy is Local, the server pod should be unreachable.
				switch externalTrafficPolicy {
				case corev1.ServiceExternalTrafficPolicyTypeLocal:
					Expect(err).To(HaveOccurred(), httpOutput)
				case corev1.ServiceExternalTrafficPolicyTypeCluster:
					Expect(err).ToNot(HaveOccurred(), httpOutput)
				}
				testBFDFailBack()
			},

			Entry("", tsparams.IBPGPProtocol, netparam.IPV4Family, corev1.ServiceExternalTrafficPolicyTypeCluster,
				reportxml.SetProperty("BGPPeer", tsparams.IBPGPProtocol),
				reportxml.SetProperty("IPStack", netparam.IPV4Family),
				reportxml.SetProperty("TrafficPolicy", "Cluster")),
			Entry("", tsparams.IBPGPProtocol, netparam.IPV4Family, corev1.ServiceExternalTrafficPolicyTypeLocal,
				reportxml.SetProperty("BGPPeer", tsparams.IBPGPProtocol),
				reportxml.SetProperty("IPStack", netparam.IPV4Family),
				reportxml.SetProperty("TrafficPolicy", "Local")),
			Entry("", tsparams.EBGPProtocol, netparam.IPV4Family, corev1.ServiceExternalTrafficPolicyTypeCluster,
				reportxml.SetProperty("BGPPeer", tsparams.EBGPProtocol),
				reportxml.SetProperty("IPStack", netparam.IPV4Family),
				reportxml.SetProperty("TrafficPolicy", "Custer")),
			Entry("", tsparams.EBGPProtocol, netparam.IPV4Family, corev1.ServiceExternalTrafficPolicyTypeLocal,
				reportxml.SetProperty("BGPPeer", tsparams.EBGPProtocol),
				reportxml.SetProperty("IPStack", netparam.IPV4Family),
				reportxml.SetProperty("TrafficPolicy", "Local")),
		)

	})

	AfterAll(func() {
		if len(cnfWorkerNodeList) > 2 {
			removeNodeLabel(workerNodeList, metalLbTestsLabel)
		}
		By("Cleaning Metallb namespace")
		metalLbNs, err := namespace.Pull(APIClient, NetConfig.MlbOperatorNamespace)
		Expect(err).ToNot(HaveOccurred(), "Failed to pull metalLb namespace")
		err = metalLbNs.CleanObjects(tsparams.DefaultTimeout, metallb.GetMetalLbIoGVR())
		Expect(err).ToNot(HaveOccurred(), "Failed to clean metalLb operator namespace")

		By("Cleaning test namespace")
		err = namespace.NewBuilder(APIClient, tsparams.TestNamespaceName).CleanObjects(
			tsparams.DefaultTimeout, pod.GetGVR(), nad.GetGVR())
		Expect(err).ToNot(HaveOccurred(), "Failed to clean test namespace")

		By("Reverting Local GW mode")
		setLocalGWMode(false)
	})
})

func createFrrPodOnMasterNodeAndWaitUntilRunning(
	name, metalLbAddr, subMask, internalFrrIP, bridgeNadName, masterNodeName, mlbPoolIP, nodeAddr string) {
	By("Creating static ip annotation for FRR pod two on master node")

	podMasterOneNetCfg := pod.StaticIPAnnotation(
		frrconfig.ExternalMacVlanNADName, []string{fmt.Sprintf("%s/%s", metalLbAddr, subMask)})
	podMasterOneNetCfg = append(podMasterOneNetCfg, pod.StaticIPAnnotation(
		bridgeNadName, []string{fmt.Sprintf("%s/%s", internalFrrIP, subMask)})...)

	By("Creating FRR pod on master node")
	createFrrPod(
		masterNodeName,
		"",
		cmd.DefineRouteAndSleep(mlbPoolIP, ipaddr.RemovePrefix(nodeAddr)),
		podMasterOneNetCfg,
		name,
	)
}

func scaleDownMetalLbSpeakers() {
	By("Changing the label selector for MetalLb speakers")

	metalLbIo, err := metallb.Pull(APIClient, tsparams.MetalLbIo, NetConfig.MlbOperatorNamespace)
	Expect(err).ToNot(HaveOccurred(), "Failed to pull metallb.io object")
	_, err = metalLbIo.WithSpeakerNodeSelector(tsparams.MetalLbSpeakerLabel).Update(false)
	Expect(err).ToNot(HaveOccurred(), "Failed to update metallb object with the new MetalLb label")

	By("Verifying that the MetalLb speakers are not running on nodes after label update")
	metalLbDaemonSetShouldMatchConditionAndBeInReadyState(
		BeZero(), "Failed to scale down metalLb speaker pods to zero")
}

func testBFDFailOver() {
	By("Adding metalLb label to compute nodes")
	addNodeLabel(workerNodeList, tsparams.MetalLbSpeakerLabel)

	By("Pulling metalLb speaker daemonset")
	metalLbDaemonSetShouldMatchConditionAndBeInReadyState(
		Not(BeZero()), "Failed to run metalLb speakers on top of nodes with test label")

	frrPod, err := pod.Pull(APIClient, tsparams.FRRContainerName, tsparams.TestNamespaceName)
	Expect(err).ToNot(HaveOccurred(), "Failed to pull frr test pod")

	By("Checking that BGP and BFD sessions are established and up")
	verifyMetalLbBFDAndBGPSessionsAreUPOnFrrPod(frrPod, ipv4NodeAddrList)
	verifyMetalLbBFDAndBGPSessionsAreUPOnFrrPod(frrPod, ipv4NodeAddrList)

	By("Removing Speaker pod from one of the compute nodes")

	firstWorkerNode, err := nodes.Pull(APIClient, workerNodeList[0].Object.Name)
	Expect(err).ToNot(HaveOccurred(), "Failed to pull worker node object")
	_, err = firstWorkerNode.RemoveLabel(netenv.MapFirstKeyValue(tsparams.MetalLbSpeakerLabel)).Update()
	Expect(err).ToNot(HaveOccurred(), "Failed to remove metalLb label from worker node")

	By("Verifying that cluster has reduced the number of speakers by 1")
	metalLbDaemonSetShouldMatchConditionAndBeInReadyState(
		BeEquivalentTo(len(workerNodeList)-1), "The number of running speaker pods is not expected")
	By("Verifying that FRR pod still has BFD and BGP session UP with one of the MetalLb speakers")

	secondWorkerNode, err := nodes.Pull(APIClient, workerNodeList[1].Object.Name)
	Expect(err).ToNot(HaveOccurred(), "Failed to pull compute node object")
	secondWorkerIP, err := secondWorkerNode.ExternalIPv4Network()
	Expect(err).ToNot(HaveOccurred(), "Failed to collect external node ip")

	// Sleep until BFD timeout
	time.Sleep(1200 * time.Millisecond)

	bpgUp, err := frr.BGPNeighborshipHasState(frrPod, ipaddr.RemovePrefix(secondWorkerIP), "Established")
	Expect(err).ToNot(HaveOccurred(), "Failed to collect bgp state from FRR router")
	Expect(bpgUp).Should(BeTrue(), "BGP is not in expected established state")
	Expect(netenv.BFDHasStatus(frrPod, ipaddr.RemovePrefix(secondWorkerIP), "up")).Should(BeNil(),
		"BFD is not in expected up state")

	By("Verifying that FRR pod lost BFD and BGP session with one of the MetalLb speakers")

	firstWorkerNodeIP, err := firstWorkerNode.ExternalIPv4Network()
	Expect(err).ToNot(HaveOccurred(), "Failed to collect external node ip")
	bpgUp, err = frr.BGPNeighborshipHasState(frrPod, ipaddr.RemovePrefix(firstWorkerNodeIP), "Established")
	Expect(err).ToNot(HaveOccurred(), "Failed to collect BGP state")
	Expect(bpgUp).Should(BeFalse(), "BGP is not in expected down state")
	Expect(netenv.BFDHasStatus(frrPod, ipaddr.RemovePrefix(firstWorkerNodeIP), "down")).
		ShouldNot(HaveOccurred(), "BFD is not in expected down state")
}

func testBFDFailBack() {
	By("Bringing Speaker pod back by labeling node")

	firstWorkerNode, err := nodes.Pull(APIClient, workerNodeList[0].Object.Name)
	Expect(err).ToNot(HaveOccurred(), "Failed to pull worker node object")
	_, err = firstWorkerNode.WithNewLabel(netenv.MapFirstKeyValue(tsparams.MetalLbSpeakerLabel)).Update()
	Expect(err).ToNot(HaveOccurred(), "Failed to append metalLb label to worker node")

	By("Check if speakers daemonSet is UP and running")

	frrPod, err := pod.Pull(APIClient, tsparams.FRRContainerName, tsparams.TestNamespaceName)
	Expect(err).ToNot(HaveOccurred(), "Failed to pull frr test pod")

	metalLbDaemonSetShouldMatchConditionAndBeInReadyState(
		BeEquivalentTo(len(workerNodeList)), "The number of running speaker pods is not expected")
	verifyMetalLbBFDAndBGPSessionsAreUPOnFrrPod(frrPod, ipv4NodeAddrList)
}

func createBFDProfileAndVerifyIfItsReady(frrk8sPods []*pod.Builder) *metallb.BFDBuilder {
	By("Creating BFD profile")

	bfdProfile, err := metallb.NewBFDBuilder(APIClient, "bfdprofile", NetConfig.MlbOperatorNamespace).
		WithRcvInterval(300).WithTransmitInterval(300).WithEchoInterval(100).
		WithEchoMode(true).WithPassiveMode(false).WithMinimumTTL(5).
		WithMultiplier(3).Create()
	Expect(err).ToNot(HaveOccurred(), "Failed to create BFD profile")
	Expect(bfdProfile.Exists()).To(BeTrue(), "BFD profile doesn't exist")

	for _, frrk8sPod := range frrk8sPods {
		Eventually(frr.IsProtocolConfigured,
			time.Minute, tsparams.DefaultRetryInterval).WithArguments(frrk8sPod, "bfd").
			Should(BeTrue(), "BFD is not configured on the Speakers")
	}

	return bfdProfile
}

func setLocalGWMode(status bool) {
	By(fmt.Sprintf("Configuring GW mode %v", status))

	clusterNetwork, err := cluster.GetOCPNetworkOperatorConfig(APIClient)
	Expect(err).ToNot(HaveOccurred(), "Failed to collect network.operator object")

	clusterNetwork, err = clusterNetwork.SetLocalGWMode(status, 20*time.Minute)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to set local GW mode %v", status))

	network, err := clusterNetwork.Get()
	Expect(err).ToNot(HaveOccurred(), "Failed to collect network.operator object")
	Expect(network.Spec.DefaultNetwork.OVNKubernetesConfig.GatewayConfig.RoutingViaHost).To(BeEquivalentTo(status),
		"Failed network.operator object is not in expected state")
}

func verifyMetalLbBFDAndBGPSessionsAreUPOnFrrPod(frrPod *pod.Builder, peerAddrList []string) {
	for _, peerAddress := range netcmd.RemovePrefixFromIPList(peerAddrList) {
		Eventually(frr.BGPNeighborshipHasState,
			time.Minute*4, tsparams.DefaultRetryInterval).
			WithArguments(frrPod, peerAddress, "Established").Should(
			BeTrue(), "Failed to receive BGP status UP")
		Eventually(netenv.BFDHasStatus,
			time.Minute, tsparams.DefaultRetryInterval).
			WithArguments(frrPod, peerAddress, "up").
			ShouldNot(HaveOccurred(), "Failed to receive BFD status UP")
	}
}

func buildRoutesMap(podList []*pod.Builder, nextHopList []string) (map[string]string, error) {
	if len(podList) == 0 {
		return nil, fmt.Errorf("pod list is empty")
	}

	if len(nextHopList) == 0 {
		return nil, fmt.Errorf("nexthop IP addresses list is empty")
	}

	if len(nextHopList) < len(podList) {
		return nil, fmt.Errorf("number of speaker IP addresses[%d] is less then number of pods[%d]",
			len(nextHopList), len(podList))
	}

	routesMap := make(map[string]string)

	for num, pod := range podList {
		routesMap[pod.Definition.Spec.NodeName] = nextHopList[num]
	}

	return routesMap, nil
}
