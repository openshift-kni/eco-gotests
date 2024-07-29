package tests

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	typesGomega "github.com/onsi/gomega/types"
	"github.com/openshift-kni/eco-goinfra/pkg/configmap"
	"github.com/openshift-kni/eco-goinfra/pkg/daemonset"
	"github.com/openshift-kni/eco-goinfra/pkg/metallb"
	"github.com/openshift-kni/eco-goinfra/pkg/nad"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-goinfra/pkg/service"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/internal/coreparams"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/define"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/ipaddr"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netenv"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netinittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/cmd"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/frr"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/prometheus"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/tsparams"
	"gopkg.in/k8snetworkplumbingwg/multus-cni.v4/pkg/types"
	corev1 "k8s.io/api/core/v1"
)

// test cases variables that are accessible across entire package.
var (
	ipv4metalLbIPList []string
	ipv4NodeAddrList  []string
	ipv6metalLbIPList []string
	ipv6NodeAddrList  []string
	externalNad       *nad.Builder
	cnfWorkerNodeList []*nodes.Builder
	workerNodeList    []*nodes.Builder
	masterNodeList    []*nodes.Builder
	workerLabelMap    map[string]string
	metalLbTestsLabel = map[string]string{"metallb": "metallbtests"}
)

func removeNodeLabel(workerNodeList []*nodes.Builder, nodeSelector map[string]string) {
	updateNodeLabel(workerNodeList, nodeSelector, true)
}

func addNodeLabel(workerNodeList []*nodes.Builder, nodeSelector map[string]string) {
	updateNodeLabel(workerNodeList, nodeSelector, false)
}

func updateNodeLabel(workerNodeList []*nodes.Builder, nodeLabel map[string]string, removeLabel bool) {
	for _, worker := range workerNodeList {
		worker, err := nodes.Pull(APIClient, worker.Definition.Name)
		Expect(err).ToNot(HaveOccurred(), "Fail to pull latest worker %s object", worker.Definition.Name)

		if removeLabel {
			worker.RemoveLabel(netenv.MapFirstKeyValue(nodeLabel))
		} else {
			worker.WithNewLabel(netenv.MapFirstKeyValue(nodeLabel))
		}

		_, err = worker.Update()
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Fail to update node's labels %s", worker.Definition.Name))
	}
}

func setWorkerNodeListAndLabelForBfdTests(
	workerNodeList []*nodes.Builder, nodeSelector map[string]string) (map[string]string, []*nodes.Builder) {
	if len(workerNodeList) > 2 {
		By(fmt.Sprintf(
			"Worker node number is greater than 2. Limit worker nodes for bfd test using label %v", nodeSelector))
		addNodeLabel(workerNodeList[:2], nodeSelector)

		return metalLbTestsLabel, workerNodeList[:2]
	}

	return NetConfig.WorkerLabelMap, workerNodeList
}

func createConfigMap(
	bgpAsn int, nodeAddrList []string, enableMultiHop, enableBFD bool) *configmap.Builder {
	frrBFDConfig := frr.DefineBGPConfig(
		bgpAsn, tsparams.LocalBGPASN, removePrefixFromIPList(nodeAddrList), enableMultiHop, enableBFD)
	configMapData := frr.DefineBaseConfig(tsparams.DaemonsFile, frrBFDConfig, "")
	masterConfigMap, err := configmap.NewBuilder(APIClient, "frr-master-node-config", tsparams.TestNamespaceName).
		WithData(configMapData).Create()
	Expect(err).ToNot(HaveOccurred(), "Failed to create config map")

	return masterConfigMap
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
	peerIP, bfdProfileName string, remoteAsn uint32, eBgpMultiHop bool, frrk8sPods []*pod.Builder) {
	By("Creating BGP Peer")

	bgpPeer := metallb.NewBPGPeerBuilder(APIClient, "testpeer", NetConfig.MlbOperatorNamespace,
		peerIP, tsparams.LocalBGPASN, remoteAsn).WithPassword(tsparams.BGPPassword).WithEBGPMultiHop(eBgpMultiHop)

	if bfdProfileName != "" {
		bgpPeer.WithBFDProfile(bfdProfileName)
	}

	_, err := bgpPeer.Create()
	Expect(err).ToNot(HaveOccurred(), "Failed to create BGP peer")

	By("Verifying if BGP protocol configured")

	for _, frrk8sPod := range frrk8sPods {
		Eventually(frr.IsProtocolConfigured,
			time.Minute, tsparams.DefaultRetryInterval).WithArguments(frrk8sPod, "router bgp").
			Should(BeTrue(), "BGP is not configured on the Speakers")
	}
}

func setupBgpAdvertisement(addressPool []string, prefixLen int32) *metallb.IPAddressPoolBuilder {
	ipAddressPool, err := metallb.NewIPAddressPoolBuilder(
		APIClient,
		"address-pool",
		NetConfig.MlbOperatorNamespace,
		[]string{fmt.Sprintf("%s-%s", addressPool[0], addressPool[1])}).Create()
	Expect(err).ToNot(HaveOccurred(), "Failed to create IPAddressPool")

	_, err = metallb.
		NewBGPAdvertisementBuilder(APIClient, "bgpadvertisement", NetConfig.MlbOperatorNamespace).
		WithIPAddressPools([]string{ipAddressPool.Definition.Name}).
		WithCommunities([]string{"65535:65282"}).
		WithLocalPref(100).
		WithAggregationLength4(prefixLen).Create()
	Expect(err).ToNot(HaveOccurred(), "Failed to create BGPAdvertisement")

	return ipAddressPool
}

func setupL2Advertisement(addressPool []string) *metallb.IPAddressPoolBuilder {
	ipAddressPool, err := metallb.NewIPAddressPoolBuilder(
		APIClient,
		"l2address-pool",
		NetConfig.MlbOperatorNamespace,
		[]string{fmt.Sprintf("%s-%s", addressPool[0], addressPool[1])}).Create()
	Expect(err).ToNot(HaveOccurred(), "Failed to create IPAddressPool")

	_, err = metallb.
		NewL2AdvertisementBuilder(APIClient, "l2advertisement", NetConfig.MlbOperatorNamespace).
		WithIPAddressPools([]string{ipAddressPool.Definition.Name}).Create()
	Expect(err).ToNot(HaveOccurred(), "Failed to create BGPAdvertisement")

	return ipAddressPool
}

func verifyMetalLbBGPSessionsAreUPOnFrrPod(frrPod *pod.Builder, peerAddrList []string) {
	for _, peerAddress := range removePrefixFromIPList(peerAddrList) {
		Eventually(frr.BGPNeighborshipHasState,
			time.Minute*3, tsparams.DefaultRetryInterval).
			WithArguments(frrPod, peerAddress, "Established").Should(
			BeTrue(), "Failed to receive BGP status UP")
	}
}

func createFrrPod(
	nodeName string,
	configmapName string,
	defaultCMD []string,
	secondaryNetConfig []*types.NetworkSelectionElement,
	podName ...string) *pod.Builder {
	name := tsparams.FRRContainerName

	if len(podName) > 0 {
		name = podName[0]
	}

	frrPod := pod.NewBuilder(APIClient, name, tsparams.TestNamespaceName, NetConfig.FrrImage).
		DefineOnNode(nodeName).
		WithTolerationToMaster().
		WithSecondaryNetwork(secondaryNetConfig).
		RedefineDefaultCMD(defaultCMD)

	By("Creating FRR container")

	if configmapName != "" {
		frrContainer := pod.NewContainerBuilder(
			tsparams.FRRSecondContainerName, NetConfig.CnfNetTestContainer, tsparams.SleepCMD).
			WithSecurityCapabilities([]string{"NET_ADMIN", "NET_RAW", "SYS_ADMIN"}, true)

		frrCtr, err := frrContainer.GetContainerCfg()
		Expect(err).ToNot(HaveOccurred(), "Failed to get container configuration")
		frrPod.WithAdditionalContainer(frrCtr).WithLocalVolume(configmapName, "/etc/frr")
	}

	By("Creating FRR pod in the test namespace")

	frrPod, err := frrPod.WithPrivilegedFlag().CreateAndWaitUntilRunning(time.Minute)
	Expect(err).ToNot(HaveOccurred(), "Failed to create FRR test pod")

	return frrPod
}

func setupMetalLbService(
	ipStack string,
	ipAddressPool *metallb.IPAddressPoolBuilder,
	extTrafficPolicy corev1.ServiceExternalTrafficPolicyType) {
	servicePort, err := service.DefineServicePort(80, 80, "TCP")
	Expect(err).ToNot(HaveOccurred(), "Failed to define service port")

	_, err = service.NewBuilder(APIClient, "service-mlb", tsparams.TestNamespaceName,
		map[string]string{"app": "nginx1"}, *servicePort).
		WithExternalTrafficPolicy(extTrafficPolicy).
		WithIPFamily([]corev1.IPFamily{corev1.IPFamily(ipStack)}, corev1.IPFamilyPolicySingleStack).
		WithAnnotation(map[string]string{"metallb.universe.tf/address-pool": ipAddressPool.Definition.Name}).
		Create()
	Expect(err).ToNot(HaveOccurred(), "Failed to create MetalLB Service")
}

func setupNGNXPod(nodeName string) {
	_, err := pod.NewBuilder(
		APIClient, "mlbnginxtpod"+nodeName, tsparams.TestNamespaceName, NetConfig.CnfNetTestContainer).
		DefineOnNode(nodeName).
		WithLabel("app", "nginx1").
		RedefineDefaultCMD(cmd.DefineNGNXAndSleep()).
		WithPrivilegedFlag().CreateAndWaitUntilRunning(tsparams.DefaultTimeout)
	Expect(err).ToNot(HaveOccurred(), "Failed to create nginx test pod")
}

func removePrefixFromIPList(ipAddressList []string) []string {
	var ipAddressListWithoutPrefix []string
	for _, ipaddress := range ipAddressList {
		ipAddressListWithoutPrefix = append(ipAddressListWithoutPrefix, ipaddr.RemovePrefix(ipaddress))
	}

	return ipAddressListWithoutPrefix
}

func verifyMetricPresentInPrometheus(
	frrk8sPods []*pod.Builder, prometheusPod *pod.Builder, metricPrefix string, expectedMetrics ...[]string) {
	By("Verifying if metrics are present in Prometheus database")

	for _, frrk8sPod := range frrk8sPods {
		var (
			metricsFromSpeaker []string
			err                error
		)

		Eventually(func() error {
			metricsFromSpeaker, err = frr.GetMetricsByPrefix(frrk8sPod, metricPrefix)

			return err
		}, time.Minute, tsparams.DefaultRetryInterval).ShouldNot(HaveOccurred(),
			"Failed to collect metrics from speaker pods")

		if len(expectedMetrics) > 0 {
			By("Verifying if metrics match expected list of metrics")
			Expect(expectedMetrics[0]).To(ContainElements(metricsFromSpeaker))
		}

		Eventually(
			prometheus.PodMetricsPresentInDB, 5*time.Minute, tsparams.DefaultRetryInterval).WithArguments(
			prometheusPod, frrk8sPod.Definition.Name, metricsFromSpeaker).Should(
			BeTrue(), "Failed to match metric in prometheus")
	}
}

func metalLbDaemonSetShouldMatchConditionAndBeInReadyState(
	expectedCondition typesGomega.GomegaMatcher, errorMessage string) {
	metalLbDs, err := daemonset.Pull(APIClient, tsparams.MetalLbDsName, NetConfig.MlbOperatorNamespace)
	Expect(err).ToNot(HaveOccurred(), "Failed to pull metalLb speaker daemonSet")

	Eventually(func() int32 {
		if metalLbDs.Exists() {
			return metalLbDs.Object.Status.NumberAvailable
		}

		return 0
	}, tsparams.DefaultTimeout, tsparams.DefaultRetryInterval).Should(expectedCondition, errorMessage)
	Expect(metalLbDs.IsReady(120*time.Second)).To(BeTrue(), "MetalLb daemonSet is not Ready")
}
