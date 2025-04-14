package tests

import (
	"fmt"

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
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/metallbenv"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/tsparams"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

var _ = Describe("BGP", Ordered, Label(tsparams.LabelBGPTestCases), ContinueOnFailure, func() {

	BeforeAll(func() {
		var err error
		By("Getting MetalLb load balancer ip addresses")
		ipv4metalLbIPList, ipv6metalLbIPList, err = metallbenv.GetMetalLbIPByIPStack()
		Expect(err).ToNot(HaveOccurred(), tsparams.MlbAddressListError)

		By("Getting external nodes ip addresses")
		cnfWorkerNodeList, err = nodes.List(APIClient,
			metav1.ListOptions{LabelSelector: labels.Set(NetConfig.WorkerLabelMap).String()})
		Expect(err).ToNot(HaveOccurred(), "Failed to discover worker nodes")

		By("Selecting worker node for BGP tests")
		workerLabelMap, workerNodeList = setWorkerNodeListAndLabelForBfdTests(cnfWorkerNodeList, metalLbTestsLabel)
		ipv4NodeAddrList, err = nodes.ListExternalIPv4Networks(
			APIClient, metav1.ListOptions{LabelSelector: labels.Set(workerLabelMap).String()})
		Expect(err).ToNot(HaveOccurred(), "Failed to collect external nodes ip addresses")

		By("Creating a new instance of MetalLB Speakers on workers")
		err = metallbenv.CreateNewMetalLbDaemonSetAndWaitUntilItsRunning(tsparams.DefaultTimeout, workerLabelMap)
		Expect(err).ToNot(HaveOccurred(), "Failed to recreate metalLb daemonset")

		err = metallbenv.IsEnvVarMetalLbIPinNodeExtNetRange(ipv4NodeAddrList, ipv4metalLbIPList, nil)
		Expect(err).ToNot(HaveOccurred(), "Failed to validate metalLb exported ip address")

		By("Listing master nodes")
		masterNodeList, err = nodes.List(APIClient,
			metav1.ListOptions{LabelSelector: labels.Set(NetConfig.ControlPlaneLabelMap).String()})
		Expect(err).ToNot(HaveOccurred(), "Fail to list master nodes")
		Expect(len(masterNodeList)).To(BeNumerically(">", 0),
			"Failed to detect master nodes")
	})

	var frrk8sPods []*pod.Builder

	BeforeEach(func() {
		By("Creating External NAD")
		err := define.CreateExternalNad(APIClient, frrconfig.ExternalMacVlanNADName, tsparams.TestNamespaceName)
		Expect(err).ToNot(HaveOccurred(), "Failed to create a network-attachment-definition")

		By("Verifying that the frrk8sPod deployment is in Ready state and create a list of the pods on " +
			"worker nodes.")
		frrk8sPods = verifyAndCreateFRRk8sPodList()

		createBGPPeerAndVerifyIfItsReady(tsparams.BgpPeerName1, ipv4metalLbIPList[0], "",
			tsparams.LocalBGPASN, false, 0, frrk8sPods)
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

		By("Cleaning test namespace")
		err = namespace.NewBuilder(APIClient, tsparams.TestNamespaceName).CleanObjects(
			tsparams.DefaultTimeout,
			pod.GetGVR(),
			service.GetGVR(),
			configmap.GetGVR(),
			nad.GetGVR())
		Expect(err).ToNot(HaveOccurred(), "Failed to clean test namespace")
	})

	Context("functionality", func() {
		DescribeTable("Creating AddressPool with bgp-advertisement", reportxml.ID("47174"),
			func(ipStack string, prefixLen int) {

				if ipStack == netparam.IPV6Family {
					Skip("bgp test cases doesn't support ipv6 yet")
				}

				createBGPPeerAndVerifyIfItsReady(tsparams.BgpPeerName1, ipv4metalLbIPList[0], "",
					tsparams.LocalBGPASN, false, 0,
					frrk8sPods)

				By("Setting test iteration parameters")
				_, subMask, mlbAddressList, nodeAddrList, addressPool, _, err :=
					metallbenv.DefineIterationParams(
						ipv4metalLbIPList, ipv6metalLbIPList, ipv4NodeAddrList, ipv6NodeAddrList, ipStack)
				Expect(err).ToNot(HaveOccurred(), "Fail to set iteration parameters")

				By("Creating MetalLb configMap")
				masterConfigMap := createConfigMap(tsparams.LocalBGPASN, nodeAddrList, false, false)

				By("Creating static ip annotation")
				staticIPAnnotation := pod.StaticIPAnnotation(
					frrconfig.ExternalMacVlanNADName, []string{fmt.Sprintf("%s/%s", mlbAddressList[0], subMask)})

				By("Creating FRR Pod")
				frrPod := createFrrPod(
					masterNodeList[0].Object.Name, masterConfigMap.Definition.Name, []string{}, staticIPAnnotation)

				By("Creating an IPAddressPool and BGPAdvertisement")
				ipAddressPool := setupBgpAdvertisementAndIPAddressPool(tsparams.BGPAdvAndAddressPoolName, addressPool, prefixLen)

				By("Creating a MetalLB service")
				setupMetalLbService(
					tsparams.MetallbServiceName,
					ipStack,
					tsparams.LabelValue1,
					ipAddressPool,
					corev1.ServiceExternalTrafficPolicyTypeCluster)

				By("Creating nginx test pod on worker node")
				setupNGNXPod(workerNodeList[0].Definition.Name, tsparams.LabelValue1)

				By("Checking that BGP session is established and up")
				verifyMetalLbBGPSessionsAreUPOnFrrPod(frrPod, cmd.RemovePrefixFromIPList(nodeAddrList))

				By("Validating BGP route prefix")
				validatePrefix(frrPod, ipStack, prefixLen, removePrefixFromIPList(nodeAddrList), addressPool)
			},

			Entry("", netparam.IPV4Family, 32,
				reportxml.SetProperty("IPStack", netparam.IPV4Family),
				reportxml.SetProperty("PrefixLenght", netparam.IPSubnet32)),
			Entry("", netparam.IPV4Family, 28,
				reportxml.SetProperty("IPStack", netparam.IPV4Family),
				reportxml.SetProperty("PrefixLenght", netparam.IPSubnet28)),
			Entry("", netparam.IPV6Family, 128,
				reportxml.SetProperty("IPStack", netparam.IPV6Family),
				reportxml.SetProperty("PrefixLenght", netparam.IPSubnet128)),
			Entry("", netparam.IPV6Family, 64,
				reportxml.SetProperty("IPStack", netparam.IPV6Family),
				reportxml.SetProperty("PrefixLenght", netparam.IPSubnet64)),
		)

		It("provides Prometheus BGP metrics", reportxml.ID("47202"), func() {
			By("Creating static ip annotation")
			staticIPAnnotation := pod.StaticIPAnnotation(
				frrconfig.ExternalMacVlanNADName, []string{fmt.Sprintf("%s/%s", ipv4metalLbIPList[0],
					netparam.IPSubnet24)})

			By("Creating MetalLb configMap")
			masterConfigMap := createConfigMap(tsparams.LocalBGPASN, ipv4NodeAddrList, false, false)

			By("Creating FRR Pod")
			frrPod := createFrrPod(
				masterNodeList[0].Object.Name, masterConfigMap.Definition.Name, []string{}, staticIPAnnotation)

			createBGPPeerAndVerifyIfItsReady(tsparams.BgpPeerName1, ipv4metalLbIPList[0], "",
				tsparams.LocalBGPASN, false, 0, frrk8sPods)

			By("Checking that BGP session is established and up")
			verifyMetalLbBGPSessionsAreUPOnFrrPod(frrPod, cmd.RemovePrefixFromIPList(ipv4NodeAddrList))

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
				metallb.GetBGPPeerGVR(),
				metallb.GetBGPAdvertisementGVR(),
				metallb.GetIPAddressPoolGVR(),
				metallb.GetMetalLbIoGVR())
			Expect(err).ToNot(HaveOccurred(), "Failed to remove object's from operator namespace")

			By("Cleaning test namespace")
			err = namespace.NewBuilder(APIClient, tsparams.TestNamespaceName).CleanObjects(
				tsparams.DefaultTimeout,
				pod.GetGVR(),
				service.GetGVR(),
				configmap.GetGVR(),
				nad.GetGVR())
			Expect(err).ToNot(HaveOccurred(), "Failed to clean test namespace")
		})
	})
})
