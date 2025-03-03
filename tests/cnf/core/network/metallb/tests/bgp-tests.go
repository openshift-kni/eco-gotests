package tests

import (
	"fmt"
	"strings"
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
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/cmd"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/define"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/frrconfig"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netinittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netparam"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/frr"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/metallbenv"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/tsparams"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

var _ = Describe("BGP", Ordered, Label(tsparams.LabelBGPTestCases), ContinueOnFailure, func() {

	var err error
	BeforeAll(func() {
		By("Getting IP addresses from ENV VAR to be used for external FRR")
		ipv4metalLbIPList, ipv6metalLbIPList, err = metallbenv.GetMetalLbIPByIPStack()
		Expect(err).ToNot(HaveOccurred(), tsparams.MlbAddressListError)
		Expect(len(ipv4metalLbIPList)).To(BeNumerically(">=", 2), "Expected to find atleast two IPv4 addresses in ENV VAR")
		Expect(len(ipv6metalLbIPList)).To(BeNumerically(">=", 2), "Expected to find atleast two IPv6 addresses in ENV VAR")

		By("Verify if the IPs passed in ENV VAR are in the same subnet as node external ip addresses")
		err = metallbenv.IsEnvVarMetalLbIPinNodeExtNetRange(ipv4NodeAddrList, ipv4metalLbIPList, nil)
		Expect(err).ToNot(HaveOccurred(), "Failed to validate metalLb exported ip address")

		By("Selecting Worker nodes for MetalLB BGP tests")
		cnfWorkerNodeList, err = nodes.List(APIClient,
			metav1.ListOptions{LabelSelector: labels.Set(NetConfig.WorkerLabelMap).String()})
		Expect(err).ToNot(HaveOccurred(), "Failed to discover worker nodes")

		workerLabelMap, workerNodeList = setWorkerNodeListAndLabelForBfdTests(cnfWorkerNodeList, metalLbTestsLabel)
		ipv4NodeAddrList, err = nodes.ListExternalIPv4Networks(
			APIClient, metav1.ListOptions{LabelSelector: labels.Set(workerLabelMap).String()})
		Expect(err).ToNot(HaveOccurred(), "Failed to collect worker nodes external ip addresses")

		By("Verify if the IPs passed in ENV VAR are in the same subnet as node external ip addresses")
		err = metallbenv.IsEnvVarMetalLbIPinNodeExtNetRange(ipv4NodeAddrList, ipv4metalLbIPList, nil)
		Expect(err).ToNot(HaveOccurred(), "Failed to validate metalLb exported ip address")

		By("Listing master nodes")
		masterNodeList, err = nodes.List(APIClient,
			metav1.ListOptions{LabelSelector: labels.Set(NetConfig.ControlPlaneLabelMap).String()})
		Expect(err).ToNot(HaveOccurred(), "Fail to list master nodes")
		Expect(len(masterNodeList)).To(BeNumerically(">", 0),
			"master nodes list is empty")

		By("Creating a new instance of MetalLB Speakers on workers")
		err = metallbenv.CreateNewMetalLbDaemonSetAndWaitUntilItsRunning(tsparams.DefaultTimeout, workerLabelMap)
		Expect(err).ToNot(HaveOccurred(), "Failed to recreate metalLb speakers daemonset")
	})

	AfterAll(func() {
		if len(cnfWorkerNodeList) > 2 {
			By("Remove custom metallb test label from nodes")
			removeNodeLabel(workerNodeList, metalLbTestsLabel)
		}

		By("Cleaning MetalLb operator namespace")
		metalLbNs, err := namespace.Pull(APIClient, NetConfig.MlbOperatorNamespace)
		Expect(err).ToNot(HaveOccurred(), "Failed to pull metalLb operator namespace")
		err = metalLbNs.CleanObjects(
			tsparams.DefaultTimeout,
			metallb.GetBGPPeerGVR(),
			metallb.GetBFDProfileGVR(),
			metallb.GetBGPAdvertisementGVR(),
			metallb.GetIPAddressPoolGVR(),
			metallb.GetMetalLbIoGVR())
		Expect(err).ToNot(HaveOccurred(), "Failed to remove object's from operator namespace")
	})

	Context("Functionality", func() {
		DescribeTable("Creating AddressPool with bgp-advertisement", reportxml.ID("47174"),
			func(ipStack string, prefixLen int) {

				if ipStack == netparam.IPV6Family {
					Skip("bgp test cases doesn't support ipv6 yet")
				}

				_, extFrrPod, _ := setupDefaultIPv4TestEnv(ipStack, int32(prefixLen), false)

				By("Validating BGP route prefix")
				validatePrefix(extFrrPod, ipStack, cmd.RemovePrefixFromIPList(ipv4NodeAddrList), tsparams.LBipv4Range, prefixLen)
			},

			Entry("", netparam.IPV4Family, 32,
				reportxml.SetProperty("IPStack", netparam.IPV4Family),
				reportxml.SetProperty("PrefixLength", netparam.IPSubnet32)),
			Entry("", netparam.IPV4Family, 28,
				reportxml.SetProperty("IPStack", netparam.IPV4Family),
				reportxml.SetProperty("PrefixLength", netparam.IPSubnet28)),
			Entry("", netparam.IPV6Family, 128,
				reportxml.SetProperty("IPStack", netparam.IPV6Family),
				reportxml.SetProperty("PrefixLength", netparam.IPSubnet128)),
			Entry("", netparam.IPV6Family, 64,
				reportxml.SetProperty("IPStack", netparam.IPV6Family),
				reportxml.SetProperty("PrefixLength", netparam.IPSubnet64)),
		)

		DescribeTable("Verify external FRR BGP Peer cannot propagate routes to Speaker", reportxml.ID("47203"),
			func(ipStack string) {

				if ipStack == netparam.IPV6Family {
					Skip("bgp test cases doesn't support ipv6 yet")
				}

				frrk8sPods, extFrrPod, _ := setupDefaultIPv4TestEnv(netparam.IPV4Family, int32(32), true)

				By("Verify external FRR is advertising prefixes")
				advRoutes, err := frr.GetBGPAdvertisedRoutes(extFrrPod, cmd.RemovePrefixFromIPList(ipv4NodeAddrList))
				Expect(err).ToNot(HaveOccurred(), "Failed to get BGP Advertised routes")
				Expect(len(advRoutes)).To(BeNumerically(">", 0), "BGP Advertised routes should not be empty")

				By("Verify MetalLB FRR pod is not receiving routes from External FRR Pod")
				recRoutes, err := frr.VerifyBGPReceivedRoutesOnFrrNodes(frrk8sPods)
				Expect(err).ToNot(HaveOccurred(), "Failed to verify BGP routes")
				Expect(recRoutes).ShouldNot(SatisfyAny(
					ContainSubstring(tsparams.ExtFrrConnectedPool[0]), ContainSubstring(tsparams.ExtFrrConnectedPool[1])),
					"Received routes validation failed")
			},

			Entry("", netparam.IPV4Family,
				reportxml.SetProperty("IPStack", netparam.IPV4Family)),
			Entry("", netparam.IPV6Family,
				reportxml.SetProperty("IPStack", netparam.IPV6Family)),
		)

		It("provides Prometheus BGP metrics", reportxml.ID("47202"), func() {

			frrk8sPods, extFrrPod, _ := setupDefaultIPv4TestEnv(netparam.IPV4Family, 32, false)

			By("Validating BGP route prefix")
			validatePrefix(extFrrPod, netparam.IPV4Family, cmd.RemovePrefixFromIPList(ipv4NodeAddrList),
				tsparams.LBipv4Range, 32)

			By("Label namespace")
			testNs, err := namespace.Pull(APIClient, NetConfig.MlbOperatorNamespace)
			Expect(err).ToNot(HaveOccurred())
			_, err = testNs.WithLabel(tsparams.PrometheusMonitoringLabel, "true").Update()
			Expect(err).ToNot(HaveOccurred())

			By("Listing prometheus pods")
			prometheusPods, err := pod.List(APIClient, NetConfig.PrometheusOperatorNamespace, metav1.ListOptions{
				LabelSelector: tsparams.PrometheusMonitoringPodLabel,
			})
			Expect(err).ToNot(HaveOccurred(), "Failed to list prometheus pods")

			verifyMetricPresentInPrometheus(
				frrk8sPods, prometheusPods[0], "frrk8s_bgp_", tsparams.MetalLbBgpMetrics)

		})

		AfterEach(func() {
			By("Cleaning MetalLb operator namespace")
			metalLbNs, err := namespace.Pull(APIClient, NetConfig.MlbOperatorNamespace)
			Expect(err).ToNot(HaveOccurred(), "Failed to pull metalLb operator namespace")
			err = metalLbNs.CleanObjects(
				tsparams.DefaultTimeout,
				metallb.GetBGPPeerGVR(),
				metallb.GetBFDProfileGVR(),
				metallb.GetBGPAdvertisementGVR(),
				metallb.GetIPAddressPoolGVR())
			Expect(err).ToNot(HaveOccurred(), "Failed to remove object's from operator namespace")

			By("Cleaning test namespace")
			err = namespace.NewBuilder(APIClient, tsparams.TestNamespaceName).CleanObjects(tsparams.DefaultTimeout,
				pod.GetGVR(),
				service.GetServiceGVR(),
				configmap.GetGVR(),
				nad.GetGVR())
			Expect(err).ToNot(HaveOccurred(), "Failed to clean test namespace")
		})
	})

	Context("Updates", func() {
		DescribeTable("Verify bgp-advertisement updates", reportxml.ID("47178"),
			func(ipStack string, prefixLen int) {

				if ipStack == netparam.IPV6Family {
					Skip("bgp test cases doesn't support ipv6 yet")
				}

				_, extFrrPod, bgpAdv := setupDefaultIPv4TestEnv(ipStack, int32(prefixLen), false)

				By("Validating BGP route prefix")
				validatePrefix(extFrrPod, ipStack, cmd.RemovePrefixFromIPList(ipv4NodeAddrList), tsparams.LBipv4Range, prefixLen)

				By("Validate BGP Community on External FRR Pod")
				bgpStatus, err := frr.GetBGPCommunityStatusWithCommArg(extFrrPod, ipStack, tsparams.CommunityNoAdv)
				Expect(err).ToNot(HaveOccurred(), "Failed to collect bgp community status")
				Expect(len(bgpStatus.Routes)).To(BeNumerically(">", 0),
					"Failed to fetch BGP routes with required Community")

				By("Validate BGP Local Preference on External FRR Pod")
				bgpStatus, err = frr.GetBGPStatusWithCmd(extFrrPod, strings.ToLower(ipStack))
				Expect(err).ToNot(HaveOccurred(), "Failed to collect bgp command output")
				for _, frrRoute := range bgpStatus.Routes {
					Expect(frrRoute[0].LocalPref).To(Equal(uint32(100)))
				}

				By("Update BGP Advertisements")
				_, err = bgpAdv.
					WithLocalPref(200).
					WithAggregationLength4(28).
					WithCommunities([]string{"300:300"}).
					Update(false)
				Expect(err).ToNot(HaveOccurred(), "Failed to update BGPAdvertisement")

				By("Validating BGP route prefix")
				Eventually(validatePrefixWithReturn(
					extFrrPod, ipStack, tsparams.LBipv4Range, 28),
					time.Minute, tsparams.DefaultRetryInterval).ShouldNot(HaveOccurred(),
					"Failed to validate BGP route prefix")
				validatePrefix(extFrrPod, ipStack, cmd.RemovePrefixFromIPList(ipv4NodeAddrList), tsparams.LBipv4Range, 28)

				By("Validate BGP Community on External FRR Pod")
				bgpStatus, err = frr.GetBGPCommunityStatusWithCommArg(extFrrPod, ipStack, tsparams.CustomCommunity)
				Expect(err).ToNot(HaveOccurred(), "Failed to collect bgp community status")
				Expect(len(bgpStatus.Routes)).To(BeNumerically(">", 0),
					"Failed to fetch BGP routes with required Community")

				By("Validate BGP Local Preference on External FRR Pod")
				bgpStatus, err = frr.GetBGPStatusWithCmd(extFrrPod, strings.ToLower(ipStack))
				Expect(err).ToNot(HaveOccurred(), "Failed to collect bgp command output")
				for _, frrRoute := range bgpStatus.Routes {
					Expect(frrRoute[0].LocalPref).To(Equal(uint32(200)))
				}
			},
			Entry("", netparam.IPV4Family, 32,
				reportxml.SetProperty("IPStack", netparam.IPV4Family),
				reportxml.SetProperty("PrefixLength", netparam.IPSubnet32)),
			Entry("", netparam.IPV6Family, 128,
				reportxml.SetProperty("IPStack", netparam.IPV6Family),
				reportxml.SetProperty("PrefixLength", netparam.IPSubnet128)),
		)

		It("BGP Timer update", reportxml.ID("47180"), func() {
			_, extFrrPod, _ := setupDefaultIPv4TestEnv(netparam.IPV4Family, int32(32), false)

			By("Verify BGP Timers of neighbors in external FRR Pod")
			verifyBGPTimer(extFrrPod, ipv4NodeAddrList, 180000, 60000)

			By("Update BGP Timers")
			bgpPeer, err := metallb.PullBGPPeer(APIClient, "testpeer", NetConfig.MlbOperatorNamespace)
			Expect(err).NotTo(HaveOccurred(), "Failed to fetch BGP peer")

			_, err = bgpPeer.WithHoldTime(metav1.Duration{Duration: 30000 * time.Millisecond}).
				WithKeepalive(metav1.Duration{Duration: 10000 * time.Millisecond}).
				Update(false)
			Expect(err).NotTo(HaveOccurred(), "Failed to update BGP peer")

			By("Verify BGP Timers of neighbors in external FRR Pod are updated")
			err = frr.ResetBGPConnection(extFrrPod)
			Expect(err).NotTo(HaveOccurred(), "Failed to reset BGP connection")

			verifyBGPTimer(extFrrPod, ipv4NodeAddrList, 30000, 10000)

		})

		AfterEach(func() {
			By("Cleaning MetalLb operator namespace")
			metalLbNs, err := namespace.Pull(APIClient, NetConfig.MlbOperatorNamespace)
			Expect(err).ToNot(HaveOccurred(), "Failed to pull metalLb operator namespace")
			err = metalLbNs.CleanObjects(
				tsparams.DefaultTimeout,
				metallb.GetBGPPeerGVR(),
				metallb.GetBFDProfileGVR(),
				metallb.GetBGPAdvertisementGVR(),
				metallb.GetIPAddressPoolGVR())
			Expect(err).ToNot(HaveOccurred(), "Failed to remove object's from operator namespace")

			By("Cleaning test namespace")
			err = namespace.NewBuilder(APIClient, tsparams.TestNamespaceName).CleanObjects(tsparams.DefaultTimeout,
				pod.GetGVR(),
				service.GetServiceGVR(),
				configmap.GetGVR(),
				nad.GetGVR())
			Expect(err).ToNot(HaveOccurred(), "Failed to clean test namespace")
		})

	})

	Context("Log level feature", func() {
		It("Verify frrk8s pod default Info logs", reportxml.ID("49810"), func() {
			frrk8spods, _, _ := setupDefaultIPv4TestEnv(netparam.IPV4Family, int32(32), false)

			validateFrrk8sPodLogLevel(frrk8spods, "informational")

		})

		It("Verify frrk8s debug logs", reportxml.ID("49812"), func() {
			By("Creating a new instance of MetalLB with Log level set to debug")
			err = metallbenv.CreateNewMetalLbDaemonSetAndWaitUntilItsRunning(tsparams.DefaultTimeout, workerLabelMap, "debug")
			Expect(err).ToNot(HaveOccurred(), "Failed to recreate metalLb speakers daemonset")

			frrk8spods, _, _ := setupDefaultIPv4TestEnv(netparam.IPV4Family, int32(32), false)

			validateFrrk8sPodLogLevel(frrk8spods, "debug")
		})

		AfterEach(func() {
			By("Cleaning MetalLb operator namespace")
			metalLbNs, err := namespace.Pull(APIClient, NetConfig.MlbOperatorNamespace)
			Expect(err).ToNot(HaveOccurred(), "Failed to pull metalLb operator namespace")
			err = metalLbNs.CleanObjects(
				tsparams.DefaultTimeout,
				metallb.GetBGPPeerGVR(),
				metallb.GetBFDProfileGVR(),
				metallb.GetBGPAdvertisementGVR(),
				metallb.GetIPAddressPoolGVR())
			Expect(err).ToNot(HaveOccurred(), "Failed to remove object's from operator namespace")

			By("Cleaning test namespace")
			err = namespace.NewBuilder(APIClient, tsparams.TestNamespaceName).CleanObjects(tsparams.DefaultTimeout,
				pod.GetGVR(),
				service.GetServiceGVR(),
				configmap.GetGVR(),
				nad.GetGVR())
			Expect(err).ToNot(HaveOccurred(), "Failed to clean test namespace")
		})
	})

})

func setupNginxPodAndSvc(ipStack, etpType string, workerNames []string, ipAddrPool *metallb.IPAddressPoolBuilder) {
	By("Creating a LB service with label app=nginx1")
	setupMetalLbService(ipStack, ipAddrPool, corev1.ServiceExternalTrafficPolicyType(etpType))

	By("Creating nginx test pod on worker node")

	for _, workerName := range workerNames {
		setupNGNXPod(workerName)
	}
}

func setupDefaultIPv4TestEnv(ipStack string, aggPrefixLen int32, advertise bool) ([]*pod.Builder, *pod.Builder,
	*metallb.BGPAdvertisementBuilder) {
	By("Listing MetalLB frrk8s pods")

	frrk8sPods, err := pod.List(APIClient, NetConfig.MlbOperatorNamespace, metav1.ListOptions{
		LabelSelector: tsparams.FRRK8sDefaultLabel})
	Expect(err).ToNot(HaveOccurred(), "Fail to list frrk8s pods")
	Expect(len(frrk8sPods)).To(BeNumerically(">", 0), "frrk8s pods list is empty")

	By("Creating BGPPeer with external FRR Pod")
	createBGPPeerAndVerifyIfItsReady(
		ipv4metalLbIPList[0], "", tsparams.LocalBGPASN, false, 0,
		frrk8sPods)

	By("Creating an IPAddressPool")

	ipAddressPool, err := metallb.NewIPAddressPoolBuilder(
		APIClient,
		"address-pool",
		NetConfig.MlbOperatorNamespace,
		[]string{fmt.Sprintf("%s-%s", tsparams.LBipv4Range[0], tsparams.LBipv4Range[1])}).Create()
	Expect(err).ToNot(HaveOccurred(), "Failed to create IPAddressPool")

	By("Creating a BGPAdvertisement")

	bgpAdv, err := metallb.
		NewBGPAdvertisementBuilder(APIClient, "bgpadvertisement", NetConfig.MlbOperatorNamespace).
		WithIPAddressPools([]string{ipAddressPool.Definition.Name}).
		WithAggregationLength4(aggPrefixLen).
		WithCommunities([]string{"65535:65282"}).
		WithLocalPref(100).
		Create()
	Expect(err).ToNot(HaveOccurred(), "Failed to create BGPAdvertisement")

	By("Deploy nginx on single worker with LB service")
	setupNginxPodAndSvc(ipStack, "Cluster", []string{workerNodeList[0].Object.Name}, ipAddressPool)

	By("Deploy external FRR Pod with ConfigMap and macvlan NAD")

	extFrrPod := setupExtFrrPodAndDeps(advertise)

	By("Checking that BGP session is established on external FRR Pod")
	verifyMetalLbBGPSessionsAreUPOnFrrPod(extFrrPod, cmd.RemovePrefixFromIPList(ipv4NodeAddrList))

	return frrk8sPods, extFrrPod, bgpAdv
}

func setupExtFrrPodAndDeps(advertise bool) *pod.Builder {
	By("Creating configMap with selected worker nodes as BGP Peers for external FRR Pod")

	var masterConfigMap *configmap.Builder
	if advertise {
		masterConfigMap = createConfigMapWithNetwork(tsparams.LocalBGPASN, ipv4NodeAddrList,
			tsparams.ExtFrrConnectedPool, []string{}, false, false)
	} else {
		masterConfigMap = createConfigMap(tsparams.LocalBGPASN, ipv4NodeAddrList, false, false)
	}

	By("Creating macvlan NAD for external FRR Pod")

	err := define.CreateExternalNad(APIClient, frrconfig.ExternalMacVlanNADName, tsparams.TestNamespaceName)
	Expect(err).ToNot(HaveOccurred(), "Failed to create a macvlan NAD")

	By("Creating external FRR Pod with configMap mount and external NAD")

	frrPod := createFrrPod(masterNodeList[0].Object.Name, masterConfigMap.Object.Name, []string{},
		pod.StaticIPAnnotation(frrconfig.ExternalMacVlanNADName, []string{fmt.Sprintf("%s/%s", ipv4metalLbIPList[0], "24")}))

	return frrPod
}

func createConfigMapWithNetwork(
	bgpAsn int, nodeAddrList, externalAdvertisedIPv4Routes, externalAdvertisedIPv6Routes []string,
	enableMultiHop, enableBFD bool) *configmap.Builder {
	frrBFDConfig := frr.DefineBGPConfigWithNetwork(
		bgpAsn, tsparams.LocalBGPASN, externalAdvertisedIPv4Routes,
		externalAdvertisedIPv6Routes, cmd.RemovePrefixFromIPList(nodeAddrList), enableMultiHop, enableBFD)
	configMapData := frrconfig.DefineBaseConfig(frrconfig.DaemonsFile, frrBFDConfig, "")
	masterConfigMap, err := configmap.NewBuilder(APIClient, "frr-master-node-config", tsparams.TestNamespaceName).
		WithData(configMapData).Create()
	Expect(err).ToNot(HaveOccurred(), "Failed to create config map")

	return masterConfigMap
}

func verifyBGPTimer(frrPod *pod.Builder, peerAddrList []string, hTimer, aTimer int) {
	for _, peerAddress := range cmd.RemovePrefixFromIPList(peerAddrList) {
		Eventually(frr.GetBGPNeighborTimer,
			time.Minute*3, tsparams.DefaultRetryInterval).
			WithArguments(frrPod, peerAddress, hTimer, aTimer).Should(
			BeTrue(), "Failed to verify BGP Timer on peer")
	}
}

func validateFrrk8sPodLogLevel(frrk8spods []*pod.Builder, logLevel string) {
	for _, frrk8sPod := range frrk8spods {
		res, err := frrk8sPod.ExecCommand([]string{"vtysh", "-c", "show logging"}, "frr")
		Expect(err).ToNot(HaveOccurred(), "Failed to get frr log")
		Expect(res.String()).To(ContainSubstring(logLevel), "Logs does not match the log level")
	}
}
