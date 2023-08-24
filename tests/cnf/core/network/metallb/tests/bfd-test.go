package tests

import (
	"fmt"
	"strings"
	"time"

	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netparam"
	"github.com/openshift-kni/eco-gotests/tests/internal/cluster"
	coreV1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/openshift-kni/eco-goinfra/pkg/configmap"
	"github.com/openshift-kni/eco-goinfra/pkg/daemonset"
	"github.com/openshift-kni/eco-goinfra/pkg/metallb"
	"github.com/openshift-kni/eco-goinfra/pkg/nad"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-goinfra/pkg/service"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/internal/coreparams"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/define"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netinittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/cmd"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/frr"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/metallbenv"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/prometheus"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
)

// test cases variables that are accessible across entire file.
var (
	ipv4metalLbIPList []string
	ipv4NodeAddrList  []string
	ipv6metalLbIPList []string
	ipv6NodeAddrList  []string
	externalNad       *nad.Builder
	workerNodeList    *nodes.Builder
)

var _ = Describe("BFD", Ordered, Label(tsparams.LabelBFDTestCases), ContinueOnFailure, func() {

	BeforeAll(func() {
		By("Creating a new instance of MetalLB Speakers on workers")
		err := metallbenv.CreateNewMetalLbDaemonSetAndWaitUntilItsRunning(tsparams.DefaultTimeout)
		Expect(err).ToNot(HaveOccurred(), "Failed to recreate metalLb daemonset")

		By("Getting MetalLb load balancer ip addresses")
		ipv4metalLbIPList, ipv6metalLbIPList, err = metallbenv.GetMetalLbIPByIPStack()
		Expect(err).ToNot(HaveOccurred(), "An unexpected error occurred while "+
			"determining the IP addresses from the ECO_CNF_CORE_NET_MLB_ADDR_LIST environment variable.")

		if len(ipv4metalLbIPList) < 2 {
			Skip("MetalLb BFD tests require 2 ip addresses. Please check ECO_CNF_CORE_NET_MLB_ADDR_LIST env var")
		}

		By("Getting external nodes ip addresses")
		workerNodeList = nodes.NewBuilder(APIClient, NetConfig.WorkerLabelMap)
		Expect(workerNodeList.Discover()).ToNot(HaveOccurred(), "Failed to discover worker nodes")
		ipv4NodeAddrList, err = workerNodeList.ExternalIPv4Networks()
		Expect(err).ToNot(HaveOccurred(), "Failed to collect external nodes ip addresses")

		err = metallbenv.IsEnvVarMetalLbIPinNodeExtNetRange(ipv4NodeAddrList, ipv4metalLbIPList, nil)
		Expect(err).ToNot(HaveOccurred(), "Failed to validate metalLb exported ip address")
		createExternalNad()
	})

	Context("single hop", Label("singlehop"), func() {
		BeforeEach(func() {
			By("Collect running metallb bgp speakers")
			speakerPods, err := pod.List(APIClient, NetConfig.MlbOperatorNamespace, v1.ListOptions{
				LabelSelector: tsparams.MetalLbDefaultSpeakerLabel,
			})
			Expect(err).ToNot(HaveOccurred(), "Failed to list pods")
			bfdProfile := createBFDProfileAndVerifyIfItsReady(speakerPods)

			createBGPPeerAndVerifyIfItsReady(
				"testpeer",
				ipv4metalLbIPList[0],
				bfdProfile.Definition.Name,
				tsparams.LocalBGPASN,
				tsparams.RemoteBGPASN,
				false, speakerPods)

			By("Creating MetalLb configMap")
			frrBFDConfig := frr.DefineBFDConfig(
				tsparams.RemoteBGPASN, tsparams.LocalBGPASN, removePrefixFromIPList(ipv4NodeAddrList), false)
			configMapData := frr.DefineBaseConfig(tsparams.DaemonsFile, frrBFDConfig, "")

			bfdConfigMap, err := configmap.NewBuilder(
				APIClient, tsparams.FRRDefaultConfigMapName, tsparams.TestNamespaceName).WithData(configMapData).Create()
			Expect(err).ToNot(HaveOccurred(), "Failed to create configmap")

			By("Creating FRR Pod With network and IP")
			masterNodeList := nodes.NewBuilder(APIClient, map[string]string{"node-role.kubernetes.io/master": ""})
			err = masterNodeList.Discover()
			Expect(err).ToNot(HaveOccurred(), "Failed to discover control-plane nodes")

			frrPod := pod.NewBuilder(
				APIClient, tsparams.FRRContainerName, tsparams.TestNamespaceName, NetConfig.FrrImage).
				DefineOnNode(masterNodeList.Objects[0].Object.Name).WithTolerationToMaster().WithPrivilegedFlag()

			By("Creating FRR container")
			frrContainer := pod.NewContainerBuilder(tsparams.FRRSecondContainerName, NetConfig.FrrImage, tsparams.SleepCMD).
				WithSecurityCapabilities([]string{"NET_ADMIN", "NET_RAW", "SYS_ADMIN"}, true)

			frrCtr, err := frrContainer.GetContainerCfg()
			Expect(err).ToNot(HaveOccurred(), "Failed to get container configuration")

			By("Creating static ip annotation")
			staticIPAnnotation := pod.StaticIPAnnotation(
				externalNad.Definition.Name, []string{fmt.Sprintf("%s/24", ipv4metalLbIPList[0])})

			By("Creating FRR pod in the test namespace")
			frrPod.WithAdditionalContainer(frrCtr).WithSecondaryNetwork(staticIPAnnotation)
			frrPod.WithLocalVolume(bfdConfigMap.Definition.Name, "/etc/frr").RedefineDefaultCMD([]string{})
			_, err = frrPod.CreateAndWaitUntilRunning(time.Minute)
			Expect(err).ToNot(HaveOccurred(), "Failed to create FRR test pod")

			By("Checking that BGP and BFD sessions are established and up")
			verifyMetalLbBFDAndBGPSessionsAreUPOnFrrPod(frrPod, ipv4NodeAddrList)

			By("Set Local GW mode")
			setLocalGWMode(false)
		})

		It("basic functionality should provide fast link failure detection", polarion.ID("47188"), func() {
			scaleDownMetalLbSpeakers()
			testBFDFailOver()
			testBFDFailBack()
		})

		It("provides Prometheus BFD metrics", polarion.ID("47187"), func() {
			mlbNs, err := namespace.Pull(APIClient, NetConfig.MlbOperatorNamespace)
			Expect(err).ToNot(HaveOccurred(),
				fmt.Sprintf("Failed to pull %s namespace", NetConfig.MlbOperatorNamespace))
			_, err = mlbNs.WithLabel(tsparams.PrometheusMonitoringLabel, "true").Update()
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to redefine %s namespace with the label %s",
				NetConfig.MlbOperatorNamespace, tsparams.PrometheusMonitoringLabel))

			speakerPods, err := pod.List(APIClient, NetConfig.MlbOperatorNamespace, v1.ListOptions{
				LabelSelector: tsparams.MetalLbDefaultSpeakerLabel})
			Expect(err).ToNot(HaveOccurred(), "Failed to list MetalLB speaker pods")

			prometheusPods, err := pod.List(APIClient, NetConfig.PrometheusOperatorNamespace, v1.ListOptions{
				LabelSelector: tsparams.PrometheusMonitoringPodLabel,
			})
			Expect(err).ToNot(HaveOccurred(), "Failed to list prometheus pods")

			for _, speakerPod := range speakerPods {
				var metricsFromSpeaker []string
				Eventually(func() error {
					metricsFromSpeaker, err = frr.GetMetricsByPrefix(speakerPod, "metallb_bfd_")

					return err
				}, time.Minute, tsparams.DefaultRetryInterval).ShouldNot(HaveOccurred(),
					"Failed to collect metrics from speaker pods")
				Eventually(
					prometheus.PodMetricsPresentInDB, time.Minute, tsparams.DefaultRetryInterval).WithArguments(
					prometheusPods[0], speakerPod.Definition.Name, metricsFromSpeaker).Should(
					BeTrue(), "Failed to match metric in prometheus")
			}
		})

		AfterEach(func() {
			By("Removing label from Workers")
			Expect(workerNodeList.Discover()).ToNot(HaveOccurred(), "Failed to discover worker nodes")
			for _, worker := range workerNodeList.Objects {
				_, err := worker.RemoveLabel(mapFirstKeyValue(tsparams.MetalLbSpeakerLabel)).Update()
				Expect(err).ToNot(HaveOccurred(), "Failed to remove label from worker")
			}

			By("Resetting MetalLB speakerNodeSelector to default value")
			metalLbIo, err := metallb.Pull(APIClient, tsparams.MetalLbIo, NetConfig.MlbOperatorNamespace)
			Expect(err).ToNot(HaveOccurred(), "Failed to pull MetalLB object")
			_, err = metalLbIo.RemoveLabel("metal").
				WithSpeakerNodeSelector(NetConfig.WorkerLabelMap).Update(false)
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
			speakerPodList, err := pod.List(APIClient, NetConfig.MlbOperatorNamespace, v1.ListOptions{
				LabelSelector: tsparams.MetalLbDefaultSpeakerLabel,
			})
			Expect(err).ToNot(HaveOccurred(), "Failed to list speaker pods")

			speakerRoutesMap, err = buildRoutesMap(speakerPodList, ipv4metalLbIPList)
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
				metallb.GetBFDProfileGVR(),
				metallb.GetBGPPeerGVR(),
				metallb.GetBGPAdvertisementGVR(),
				metallb.GetIPAddressPoolGVR())
			Expect(err).ToNot(HaveOccurred(), "Failed to remove object's from operator namespace")

			By("Removing static routes from the speakers")
			speakerPods, err := pod.List(APIClient, NetConfig.MlbOperatorNamespace, v1.ListOptions{
				LabelSelector: tsparams.MetalLbDefaultSpeakerLabel,
			})
			Expect(err).ToNot(HaveOccurred(), "Failed to list pods")
			for _, speakerPod := range speakerPods {
				out, err := frr.SetStaticRoute(speakerPod, "del", "172.16.0.1", speakerRoutesMap)
				Expect(err).ToNot(HaveOccurred(), out)
			}

			By("Removing label from Workers")
			for _, worker := range workerNodeList.Objects {
				_, err := worker.RemoveLabel(mapFirstKeyValue(tsparams.MetalLbSpeakerLabel)).Update()
				Expect(err).ToNot(HaveOccurred(), "Failed to remove label from worker")
			}

			By("Resetting MetalLB speakerNodeSelector to default value")
			metalLbIo, err := metallb.Pull(APIClient, tsparams.MetalLbIo, NetConfig.MlbOperatorNamespace)
			Expect(err).ToNot(HaveOccurred(), "Failed to pull metallb object")
			_, err = metalLbIo.RemoveLabel("metal").
				WithSpeakerNodeSelector(NetConfig.WorkerLabelMap).Update(false)
			Expect(err).ToNot(HaveOccurred(), "Failed to reset metallb SpeakerNodeSelector to default value")

			By("Cleaning test namespace")
			err = namespace.NewBuilder(APIClient, tsparams.TestNamespaceName).CleanObjects(
				tsparams.DefaultTimeout,
				pod.GetGVR(),
				service.GetServiceGVR(),
				configmap.GetGVR(),
				nad.GetGVR())
			Expect(err).ToNot(HaveOccurred(), "Failed to clean test namespace")
		})

		DescribeTable("should provide fast link failure detection", polarion.ID("47186"),
			func(bgpProtocol, ipStack string, externalTrafficPolicy coreV1.ServiceExternalTrafficPolicyType) {
				createExternalNad()

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
				speakerPods, err := pod.List(APIClient, NetConfig.MlbOperatorNamespace, v1.ListOptions{
					LabelSelector: tsparams.MetalLbDefaultSpeakerLabel,
				})
				Expect(err).ToNot(HaveOccurred(), "Failed to list metalLb speaker pods")
				bfdProfile := createBFDProfileAndVerifyIfItsReady(speakerPods)

				var ebgpMultiHop bool
				neighbourASN := uint32(tsparams.LocalBGPASN)
				if bgpProtocol == tsparams.EBGPProtocol {
					neighbourASN = tsparams.RemoteBGPASN
					ebgpMultiHop = true
				}
				createBGPPeerAndVerifyIfItsReady(
					"testpeer",
					masterClientPodIP,
					bfdProfile.Definition.Name,
					tsparams.LocalBGPASN,
					neighbourASN,
					ebgpMultiHop,
					speakerPods)

				By("Creating an IPAddressPool and BGPAdvertisement")
				ipAddressPool, err := metallb.NewIPAddressPoolBuilder(
					APIClient, "address-pool", NetConfig.MlbOperatorNamespace,
					[]string{fmt.Sprintf("%s-%s", addressPool[0], addressPool[1])}).Create()
				Expect(err).ToNot(HaveOccurred(), "Failed to create IPAddressPool")

				bgpAdvertisement := metallb.NewBGPAdvertisementBuilder(
					APIClient, "bgpadvertisement", NetConfig.MlbOperatorNamespace)
				bgpAdvertisement.WithIPAddressPools([]string{ipAddressPool.Definition.Name}).
					WithCommunities([]string{"65535:65282"}).WithLocalPref(100)
				switch ipStack {
				case netparam.IPV4Family:
					bgpAdvertisement.WithAggregationLength4(32)
				case netparam.IPV6Family:
					bgpAdvertisement.WithAggregationLength6(128)
				}

				_, err = bgpAdvertisement.Create()
				Expect(err).ToNot(HaveOccurred(), "Failed to create BGPAdvertisement")

				By("Creating a MetalLB service")
				servicePort, err := service.DefineServicePort(80, 80, "TCP")
				Expect(err).ToNot(HaveOccurred(), "Failed to define service port")
				mlbService := service.NewBuilder(APIClient, "service-mlb", tsparams.TestNamespaceName,
					map[string]string{"app": "nginx1"}, *servicePort).
					WithExternalTrafficPolicy(externalTrafficPolicy).
					WithIPFamily([]coreV1.IPFamily{coreV1.IPFamily(ipStack)}, coreV1.IPFamilyPolicySingleStack).
					WithAnnotation(map[string]string{"metallb.universe.tf/address-pool": ipAddressPool.Definition.Name})
				_, err = mlbService.Create()
				Expect(err).ToNot(HaveOccurred(), "Failed to create MetalLB Service")

				By("Creating nginx test pod on worker node")
				mlbClientPod, err := pod.NewBuilder(
					APIClient, "mlbnginxtpod", tsparams.TestNamespaceName, NetConfig.CnfNetTestContainer).
					DefineOnNode(workerNodeList.Objects[0].Definition.Name).WithLabel("app", "nginx1").
					RedefineDefaultCMD(cmd.DefineNGNXAndSleep()).WithPrivilegedFlag().Create()
				Expect(err).ToNot(HaveOccurred(), "Failed to create nginx test pod")
				err = mlbClientPod.WaitUntilRunning(tsparams.DefaultTimeout)
				Expect(err).ToNot(HaveOccurred(), "Failed timeout nginx test pod is not in running state")

				By("Creating internal NAD")
				masterBridgePlugin, err := nad.NewMasterBridgePlugin("internalnad", "br0").
					WithIPAM(nad.IPAMStatic()).GetMasterPluginConfig()
				Expect(err).ToNot(HaveOccurred(), "Failed to create master bridge plugin setting")
				bridgeNad, err := nad.NewBuilder(APIClient, "internal", tsparams.TestNamespaceName).
					WithMasterPlugin(masterBridgePlugin).Create()
				Expect(err).ToNot(HaveOccurred(), "Failed to create internal NAD")

				By("Discovering Master nodes")
				masterNodes := nodes.NewBuilder(APIClient, NetConfig.ControlPlaneLabelMap)
				Expect(masterNodes.Discover()).ToNot(HaveOccurred(), "Failed to discover master nodes")

				By("Creating FRR pod one on master node")
				createFrrPodOnMasterNodeAndWaitUntilRunning("frronmaster1",
					mlbAddressList[0], subMast, frrMasterIPs[0], bridgeNad.Definition.Name,
					masterNodes.Objects[0].Object.Name, addressPool[0], nodeAddrList[0])

				By("Creating FRR pod two on master node")
				createFrrPodOnMasterNodeAndWaitUntilRunning("frronmaster2",
					mlbAddressList[1], subMast, frrMasterIPs[1], bridgeNad.Definition.Name,
					masterNodes.Objects[0].Object.Name, addressPool[0], nodeAddrList[1])

				By("Creating client pod config map")
				frrBFDConfig := frr.DefineBFDConfig(
					int(neighbourASN), 64500, removePrefixFromIPList(nodeAddrList), ebgpMultiHop)
				configMapData := frr.DefineBaseConfig(tsparams.DaemonsFile, frrBFDConfig, "")
				masterConfigMap, err := configmap.NewBuilder(
					APIClient, "frr-master-node-config", tsparams.TestNamespaceName).
					WithData(configMapData).Create()
				Expect(err).ToNot(HaveOccurred(), "Failed to create config map")

				By("Creating FRR container")
				frrContainer := pod.NewContainerBuilder(
					tsparams.FRRSecondContainerName, NetConfig.CnfNetTestContainer, tsparams.SleepCMD).
					WithSecurityCapabilities([]string{"NET_ADMIN", "NET_RAW", "SYS_ADMIN"}, true)

				frrCtr, err := frrContainer.GetContainerCfg()
				Expect(err).ToNot(HaveOccurred(), "Failed to get container configuration")

				By("Creating FRR pod in the test namespace")
				frrPod := pod.NewBuilder(
					APIClient, tsparams.FRRContainerName, tsparams.TestNamespaceName, NetConfig.FrrImage).
					DefineOnNode(masterNodes.Objects[0].Object.Name).WithTolerationToMaster().WithPrivilegedFlag()

				frrPod.WithAdditionalContainer(frrCtr).WithSecondaryNetwork(
					pod.StaticIPAnnotation(
						bridgeNad.Definition.Name, []string{fmt.Sprintf("%s/%s", masterClientPodIP, subMast)}))
				frrPod.WithLocalVolume(masterConfigMap.Definition.Name, "/etc/frr").RedefineDefaultCMD([]string{})
				_, err = frrPod.CreateAndWaitUntilRunning(time.Minute)
				Expect(err).ToNot(HaveOccurred(), "Failed to create FRR test pod")

				// Add static routes from client towards Speaker via router internal IPs
				for index, workerAddress := range removePrefixFromIPList(nodeAddrList) {
					buffer, err := cmd.SetRouteOnPod(frrPod, workerAddress, frrMasterIPs[index])
					Expect(err).ToNot(HaveOccurred(), buffer.String())
				}
				By("Adding static routes to the speakers")
				for _, speakerPod := range speakerPods {
					out, err := frr.SetStaticRoute(speakerPod, "add", masterClientPodIP, speakerRoutesMap)
					Expect(err).ToNot(HaveOccurred(), out)
				}

				By("Checking that BGP and BFD sessions are established and up")
				verifyMetalLbBFDAndBGPSessionsAreUPOnFrrPod(frrPod, removePrefixFromIPList(nodeAddrList))

				By("Running http check")
				httpOutput, err := cmd.Curl(frrPod, masterClientPodIP, addressPool[0], ipStack, tsparams.FRRSecondContainerName)
				Expect(err).ToNot(HaveOccurred(), httpOutput)

				scaleDownMetalLbSpeakers()
				testBFDFailOver()

				By("Running http check after fail-over")
				httpOutput, err = cmd.Curl(frrPod, masterClientPodIP, addressPool[0], ipStack, tsparams.FRRSecondContainerName)
				// If externalTrafficPolicy is Local, the server pod should be unreachable.
				switch externalTrafficPolicy {
				case coreV1.ServiceExternalTrafficPolicyTypeLocal:
					Expect(err).To(HaveOccurred(), httpOutput)
				case coreV1.ServiceExternalTrafficPolicyTypeCluster:
					Expect(err).ToNot(HaveOccurred(), httpOutput)
				}
				testBFDFailBack()
			},

			Entry("", tsparams.IBPGPProtocol, netparam.IPV4Family, coreV1.ServiceExternalTrafficPolicyTypeCluster,
				polarion.SetProperty("BGPPeer", tsparams.IBPGPProtocol),
				polarion.SetProperty("IPStack", netparam.IPV4Family),
				polarion.SetProperty("TrafficPolicy", "Cluster")),
			Entry("", tsparams.IBPGPProtocol, netparam.IPV4Family, coreV1.ServiceExternalTrafficPolicyTypeLocal,
				polarion.SetProperty("BGPPeer", tsparams.IBPGPProtocol),
				polarion.SetProperty("IPStack", netparam.IPV4Family),
				polarion.SetProperty("TrafficPolicy", "Local")),
			Entry("", tsparams.EBGPProtocol, netparam.IPV4Family, coreV1.ServiceExternalTrafficPolicyTypeCluster,
				polarion.SetProperty("BGPPeer", tsparams.EBGPProtocol),
				polarion.SetProperty("IPStack", netparam.IPV4Family),
				polarion.SetProperty("TrafficPolicy", "Custer")),
			Entry("", tsparams.EBGPProtocol, netparam.IPV4Family, coreV1.ServiceExternalTrafficPolicyTypeLocal,
				polarion.SetProperty("BGPPeer", tsparams.EBGPProtocol),
				polarion.SetProperty("IPStack", netparam.IPV4Family),
				polarion.SetProperty("TrafficPolicy", "Local")),
		)

	})

	AfterAll(func() {
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

func createFrrPodOnMasterNodeAndWaitUntilRunning(name,
	metalLbAddr, subMask, internalFrrIP, bridgeNadName, masterNodeName, mlbPoolIP, nodeAddr string) {
	By("Creating static ip annotation for FRR pod two on master node")

	podMasterOneNetCfg := pod.StaticIPAnnotation(
		tsparams.ExternalMacVlanNADName, []string{fmt.Sprintf("%s/%s", metalLbAddr, subMask)})
	podMasterOneNetCfg = append(podMasterOneNetCfg, pod.StaticIPAnnotation(
		bridgeNadName, []string{fmt.Sprintf("%s/%s", internalFrrIP, subMask)})...)

	By("Creating FRR pod on master node")

	frrPodMasterOne, err := pod.NewBuilder(APIClient, name, tsparams.TestNamespaceName, NetConfig.FrrImage).
		DefineOnNode(masterNodeName).
		RedefineDefaultCMD(cmd.DefineRouteAndSleep(mlbPoolIP, removePrefixFromIP(nodeAddr))).
		WithPrivilegedFlag().WithTolerationToMaster().WithSecondaryNetwork(podMasterOneNetCfg).Create()
	Expect(err).ToNot(HaveOccurred(), "Failed to create FRR pod on master node")
	Expect(frrPodMasterOne.WaitUntilRunning(3*time.Minute)).ToNot(
		HaveOccurred(), "Failed timeout frr pod on master node is not in running state")
}

func scaleDownMetalLbSpeakers() {
	By("Changing the label selector for MetalLb speakers")

	metalLbIo, err := metallb.Pull(APIClient, tsparams.MetalLbIo, NetConfig.MlbOperatorNamespace)
	Expect(err).ToNot(HaveOccurred(), "Failed to pull metallb.io object")
	_, err = metalLbIo.WithSpeakerNodeSelector(tsparams.MetalLbSpeakerLabel).Update(false)
	Expect(err).ToNot(HaveOccurred(), "Failed to update metallb object with the new MetalLb label")

	By("Verifying that the MetalLb speakers are not running on nodes after label update")

	metalLbDs, err := daemonset.Pull(APIClient, tsparams.MetalLbDsName, NetConfig.MlbOperatorNamespace)
	Expect(err).ToNot(HaveOccurred(), "Failed to pull metalLb speaker daemonSet")
	metalLbDaemonSetShouldMatchConditionAndBeInReadyState(
		metalLbDs, BeZero(), "Failed to scale down metalLb speaker pods to zero")
}

func testBFDFailOver() {
	By("Adding metalLb label to compute nodes")

	for _, worker := range workerNodeList.Objects {
		_, err := worker.WithNewLabel(mapFirstKeyValue(tsparams.MetalLbSpeakerLabel)).Update()
		Expect(err).ToNot(HaveOccurred(), "Failed to append new metalLb label to nodes objects")
	}

	By("Pulling metalLb speaker daemonset")

	metalLbDs, err := daemonset.Pull(APIClient, tsparams.MetalLbDsName, NetConfig.MlbOperatorNamespace)
	Expect(err).ToNot(HaveOccurred(), "Failed to pull metalLb speaker daemonSet")
	metalLbDaemonSetShouldMatchConditionAndBeInReadyState(
		metalLbDs, Not(BeZero()), "Failed to run metalLb speakers on top of nodes with test label")

	frrPod, err := pod.Pull(APIClient, tsparams.FRRContainerName, tsparams.TestNamespaceName)
	Expect(err).ToNot(HaveOccurred(), "Failed to pull frr test pod")

	By("Checking that BGP and BFD sessions are established and up")
	verifyMetalLbBFDAndBGPSessionsAreUPOnFrrPod(frrPod, ipv4NodeAddrList)
	verifyMetalLbBFDAndBGPSessionsAreUPOnFrrPod(frrPod, ipv4NodeAddrList)

	By("Removing Speaker pod from one of the compute nodes")

	firstWorkerNode, err := nodes.PullNode(APIClient, workerNodeList.Objects[0].Object.Name)
	Expect(err).ToNot(HaveOccurred(), "Failed to pull worker node object")
	_, err = firstWorkerNode.RemoveLabel(mapFirstKeyValue(tsparams.MetalLbSpeakerLabel)).Update()
	Expect(err).ToNot(HaveOccurred(), "Failed to remove metalLb label from worker node")

	By("Verifying that cluster has reduced the number of speakers by 1")
	metalLbDaemonSetShouldMatchConditionAndBeInReadyState(
		metalLbDs, BeEquivalentTo(len(workerNodeList.Objects)-1), "The number of running speaker pods is not expected")
	By("Verifying that FRR pod still has BFD and BGP session UP with one of the MetalLb speakers")

	secondWorkerNode, err := nodes.PullNode(APIClient, workerNodeList.Objects[1].Object.Name)
	Expect(err).ToNot(HaveOccurred(), "Failed to pull compute node object")
	secondWorkerIP, err := secondWorkerNode.ExternalIPv4Network()
	Expect(err).ToNot(HaveOccurred(), "Failed to collect external node ip")

	// Sleep until BFD timeout
	time.Sleep(1200 * time.Millisecond)

	bpgUp, err := frr.BGPNeighborshipHasState(frrPod, removePrefixFromIP(secondWorkerIP), "Established")
	Expect(err).ToNot(HaveOccurred(), "Failed to collect bgp state from FRR router")
	Expect(bpgUp).Should(BeTrue(), "BGP is not in expected established state")
	Expect(frr.BFDHasStatus(frrPod, removePrefixFromIP(secondWorkerIP), "up")).Should(BeNil(),
		"BFD is not in expected up state")

	By("Verifying that FRR pod lost BFD and BGP session with one of the MetalLb speakers")

	firstWorkerNodeIP, err := firstWorkerNode.ExternalIPv4Network()
	Expect(err).ToNot(HaveOccurred(), "Failed to collect external node ip")
	bpgUp, err = frr.BGPNeighborshipHasState(frrPod, removePrefixFromIP(firstWorkerNodeIP), "Established")
	Expect(err).ToNot(HaveOccurred(), "Failed to collect BGP state")
	Expect(bpgUp).Should(BeFalse(), "BGP is not in expected down state")
	Expect(frr.BFDHasStatus(frrPod, removePrefixFromIP(firstWorkerNodeIP), "down")).
		ShouldNot(HaveOccurred(), "BFD is not in expected down state")
}

func testBFDFailBack() {
	By("Bringing Speaker pod back by labeling node")

	firstWorkerNode, err := nodes.PullNode(APIClient, workerNodeList.Objects[0].Object.Name)
	Expect(err).ToNot(HaveOccurred(), "Failed to pull worker node object")
	_, err = firstWorkerNode.WithNewLabel(mapFirstKeyValue(tsparams.MetalLbSpeakerLabel)).Update()
	Expect(err).ToNot(HaveOccurred(), "Failed to append metalLb label to worker node")

	By("Check if speakers daemonSet is UP and running")

	frrPod, err := pod.Pull(APIClient, tsparams.FRRContainerName, tsparams.TestNamespaceName)
	Expect(err).ToNot(HaveOccurred(), "Failed to pull frr test pod")
	metalLbDs, err := daemonset.Pull(APIClient, tsparams.MetalLbDsName, NetConfig.MlbOperatorNamespace)
	Expect(err).ToNot(HaveOccurred(), "Failed to pull metalLb speaker daemonSet")
	metalLbDaemonSetShouldMatchConditionAndBeInReadyState(
		metalLbDs, BeEquivalentTo(len(workerNodeList.Objects)), "The number of running speak pods is not expected")
	verifyMetalLbBFDAndBGPSessionsAreUPOnFrrPod(frrPod, ipv4NodeAddrList)
}

func createExternalNad() {
	By("Creating external BR-EX NetworkAttachmentDefinition")

	macVlanPlugin, err := define.MasterNadPlugin(coreparams.OvnExternalBridge, "bridge", nad.IPAMStatic())
	Expect(err).ToNot(HaveOccurred(), "Failed to define master nad plugin")
	externalNad, err = nad.NewBuilder(APIClient, tsparams.ExternalMacVlanNADName, tsparams.TestNamespaceName).
		WithMasterPlugin(macVlanPlugin).Create()
	Expect(err).ToNot(HaveOccurred(), "Failed to create external NetworkAttachmentDefinition")
	Expect(externalNad.Exists()).To(BeTrue(), "Failed to detect external NetworkAttachmentDefinition")
}

func createBGPPeerAndVerifyIfItsReady(
	name, peerIP, bfdProfileName string, asn, remouteAsn uint32, ebgpMultiHop bool, speakerPods []*pod.Builder) {
	By("Creating BGP Peer")

	_, err := metallb.NewBPGPeerBuilder(APIClient, name, NetConfig.MlbOperatorNamespace,
		peerIP, asn, remouteAsn).WithBFDProfile(bfdProfileName).
		WithPassword(tsparams.BGPPassword).WithEBGPMultiHop(ebgpMultiHop).Create()
	Expect(err).ToNot(HaveOccurred(), "Failed tp create BGP peer")

	By("Verifying if BGP protocol configured")

	for _, speakerPod := range speakerPods {
		Eventually(frr.IsProtocolConfigured,
			time.Minute, tsparams.DefaultRetryInterval).WithArguments(speakerPod, "router bgp").
			Should(BeTrue(), "BGP is not configured on the Speakers")
	}
}

func createBFDProfileAndVerifyIfItsReady(mlbSpeakerPods []*pod.Builder) *metallb.BFDBuilder {
	By("Creating BFD profile")

	bfdProfile, err := metallb.NewBFDBuilder(APIClient, "bfdprofile", NetConfig.MlbOperatorNamespace).
		WithRcvInterval(300).WithTransmitInterval(300).WithEchoInterval(100).
		WithEchoMode(true).WithPassiveMode(false).WithMinimumTTL(5).
		WithMultiplier(3).Create()
	Expect(err).ToNot(HaveOccurred(), "Failed to create BFD profile")
	Expect(bfdProfile.Exists()).To(BeTrue(), "BFD profile doesn't exist")

	for _, speakerPod := range mlbSpeakerPods {
		Eventually(frr.IsProtocolConfigured,
			time.Minute, tsparams.DefaultRetryInterval).WithArguments(speakerPod, "bfd").
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
	for _, peerAddress := range removePrefixFromIPList(peerAddrList) {
		Eventually(frr.BGPNeighborshipHasState,
			time.Minute*3, tsparams.DefaultRetryInterval).
			WithArguments(frrPod, peerAddress, "Established").Should(
			BeTrue(), "Failed to receive BGP status UP")
		Eventually(frr.BFDHasStatus,
			time.Minute, tsparams.DefaultRetryInterval).
			WithArguments(frrPod, peerAddress, "up").
			ShouldNot(HaveOccurred(), "Failed to receive BFD status UP")
	}
}

func metalLbDaemonSetShouldMatchConditionAndBeInReadyState(
	metalLbDs *daemonset.Builder, expectedCondition types.GomegaMatcher, errorMessage string) {
	Eventually(func() int32 {
		if metalLbDs.Exists() {
			return metalLbDs.Object.Status.NumberAvailable
		}

		return 0
	}, tsparams.DefaultTimeout, tsparams.DefaultRetryInterval).Should(expectedCondition, errorMessage)
	Expect(metalLbDs.IsReady(120*time.Second)).To(BeTrue(), "MetalLb daemonSet is not Ready")
}

func removePrefixFromIPList(ipAddrs []string) []string {
	var ipAddrsWithoutPrefix []string
	for _, ipaddr := range ipAddrs {
		ipAddrsWithoutPrefix = append(ipAddrsWithoutPrefix, removePrefixFromIP(ipaddr))
	}

	return ipAddrsWithoutPrefix
}

func removePrefixFromIP(ipAddr string) string {
	return strings.Split(ipAddr, "/")[0]
}

func mapFirstKeyValue(inputMap map[string]string) (string, string) {
	for key, value := range inputMap {
		return key, value
	}

	return "", ""
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

	for num, speakerPod := range podList {
		routesMap[speakerPod.Definition.Spec.NodeName] = nextHopList[num]
	}

	return routesMap, nil
}
