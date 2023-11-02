package tests

import (
	"fmt"
	"net"
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
	"github.com/openshift-kni/eco-goinfra/pkg/service"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netinittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netparam"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/frr"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/metallbenv"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

var _ = Describe("BGP", Ordered, Label(tsparams.LabelBGPTestCases), ContinueOnFailure, func() {

	BeforeAll(func() {
		var err error
		By("Getting MetalLb load balancer ip addresses")
		ipv4metalLbIPList, ipv6metalLbIPList, err = metallbenv.GetMetalLbIPByIPStack()
		Expect(err).ToNot(HaveOccurred(), "An unexpected error occurred while "+
			"determining the IP addresses from the ECO_CNF_CORE_NET_MLB_ADDR_LIST environment variable.")

		By("Getting external nodes ip addresses")
		cnfWorkerNodeList, err = nodes.List(APIClient,
			metaV1.ListOptions{LabelSelector: labels.Set(NetConfig.WorkerLabelMap).String()})
		Expect(err).ToNot(HaveOccurred(), "Failed to discover worker nodes")

		By("Selecting worker node for BGP tests")
		workerLabelMap, workerNodeList = setWorkerNodeListAndLabelForBfdTests(cnfWorkerNodeList, metalLbTestsLabel)
		ipv4NodeAddrList, err = nodes.ListExternalIPv4Networks(
			APIClient, metaV1.ListOptions{LabelSelector: labels.Set(workerLabelMap).String()})
		Expect(err).ToNot(HaveOccurred(), "Failed to collect external nodes ip addresses")

		By("Creating a new instance of MetalLB Speakers on workers")
		err = metallbenv.CreateNewMetalLbDaemonSetAndWaitUntilItsRunning(tsparams.DefaultTimeout, workerLabelMap)
		Expect(err).ToNot(HaveOccurred(), "Failed to recreate metalLb daemonset")

		err = metallbenv.IsEnvVarMetalLbIPinNodeExtNetRange(ipv4NodeAddrList, ipv4metalLbIPList, nil)
		Expect(err).ToNot(HaveOccurred(), "Failed to validate metalLb exported ip address")

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
			service.GetServiceGVR(),
			configmap.GetGVR(),
			nad.GetGVR())
		Expect(err).ToNot(HaveOccurred(), "Failed to clean test namespace")
	})

	Context("functionality", func() {
		DescribeTable("Creating AddressPool with bgp-advertisement", polarion.ID("47174"),
			func(ipStack string, prefixLen int) {

				if ipStack == netparam.IPV6Family {
					Skip("bgp test cases doesn't support ipv6 yet")
				}

				By("Creating External NAD")
				createExternalNad()

				By("Listing metalLb speakers pod")
				speakerPods, err := pod.List(APIClient, NetConfig.MlbOperatorNamespace, metaV1.ListOptions{
					LabelSelector: tsparams.MetalLbDefaultSpeakerLabel,
				})
				Expect(err).ToNot(HaveOccurred(), "Fail to list speaker pods")
				createBGPPeerAndVerifyIfItsReady(
					ipv4metalLbIPList[0], "", tsparams.LocalBGPASN, false, speakerPods)

				By("Setting test iteration parameters")
				_, subMask, mlbAddressList, nodeAddrList, addressPool, _, err :=
					metallbenv.DefineIterationParams(
						ipv4metalLbIPList, ipv6metalLbIPList, ipv4NodeAddrList, ipv6NodeAddrList, ipStack)
				Expect(err).ToNot(HaveOccurred(), "Fail to set iteration parameters")

				By("Creating MetalLb configMap")
				masterConfigMap := createConfigMap(
					tsparams.LocalBGPASN, tsparams.LocalBGPASN, nodeAddrList, false, false)

				By("Creating external FRR container")
				masterNodeList, err := nodes.List(APIClient,
					metaV1.ListOptions{LabelSelector: labels.Set(NetConfig.ControlPlaneLabelMap).String()})
				Expect(err).ToNot(HaveOccurred(), "Fail to list master nodes")
				Expect(len(masterNodeList)).To(BeNumerically(">", 0),
					"Failed to detect master nodes")

				By("Creating static ip annotation")
				staticIPAnnotation := pod.StaticIPAnnotation(
					externalNad.Definition.Name, []string{fmt.Sprintf("%s/%s", mlbAddressList[0], subMask)})

				By("Creating FRR Pod")
				frrPod := createFrrPod(
					masterNodeList[0].Object.Name, masterConfigMap.Definition.Name, []string{}, staticIPAnnotation)

				By("Creating an IPAddressPool and BGPAdvertisement")
				ipAddressPool := setupBgpAdvertisement(addressPool, int32(prefixLen))

				By("Creating a MetalLB service")
				setupMetalLbService(ipStack, ipAddressPool, "Cluster")

				By("Creating nginx test pod on worker node")
				setupNGNXPod()

				By("Checking that BGP session is established and up")
				verifyMetalLbBGPSessionsAreUPOnFrrPod(frrPod, removePrefixFromIPList(nodeAddrList))

				By("Validating BGP route prefix")
				validatePrefix(frrPod, ipStack, removePrefixFromIPList(nodeAddrList), addressPool, prefixLen)
			},

			Entry("", netparam.IPV4Family, 32,
				polarion.SetProperty("IPStack", netparam.IPV4Family),
				polarion.SetProperty("PrefixLenght", netparam.IPSubnet32)),
			Entry("", netparam.IPV4Family, 28,
				polarion.SetProperty("IPStack", netparam.IPV4Family),
				polarion.SetProperty("PrefixLenght", netparam.IPSubnet28)),
			Entry("", netparam.IPV6Family, 128,
				polarion.SetProperty("IPStack", netparam.IPV6Family),
				polarion.SetProperty("PrefixLenght", netparam.IPSubnet128)),
			Entry("", netparam.IPV6Family, 64,
				polarion.SetProperty("IPStack", netparam.IPV6Family),
				polarion.SetProperty("PrefixLenght", netparam.IPSubnet64)),
		)

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
				service.GetServiceGVR(),
				configmap.GetGVR(),
				nad.GetGVR())
			Expect(err).ToNot(HaveOccurred(), "Failed to clean test namespace")
		})
	})
})

func validatePrefix(
	masterNodeFRRPod *pod.Builder, ipProtoVersion string, workerNodesAddresses, addressPool []string, prefixLength int) {
	Eventually(
		frr.GetBGPStatus, time.Minute, tsparams.DefaultRetryInterval).
		WithArguments(masterNodeFRRPod, strings.ToLower(ipProtoVersion), "test").ShouldNot(BeNil())

	bgpStatus, err := frr.GetBGPStatus(masterNodeFRRPod, strings.ToLower(ipProtoVersion), "test")
	Expect(err).ToNot(HaveOccurred(), "Failed to verify bgp status")
	_, subnet, err := net.ParseCIDR(fmt.Sprintf("%s/%d", addressPool[0], prefixLength))
	Expect(err).ToNot(HaveOccurred(), "Failed to parse CIDR")
	Expect(bgpStatus.Routes).To(HaveKey(subnet.String()), "Failed to verify subnet in bgp status output")

	var nextHopAddresses []string

	for _, nextHop := range bgpStatus.Routes[subnet.String()] {
		Expect(nextHop.PrefixLen).To(BeNumerically("==", prefixLength),
			"Failed prefix length is not in expected value")

		for _, nHop := range nextHop.Nexthops {
			nextHopAddresses = append(nextHopAddresses, nHop.IP)
		}
	}

	Expect(workerNodesAddresses).To(ContainElements(nextHopAddresses),
		"Failed next hop address in not in node addresses list")

	_, err = frr.GetBGPCommunityStatus(masterNodeFRRPod, strings.ToLower(ipProtoVersion))
	Expect(err).ToNot(HaveOccurred(), "Failed to collect bgp community status")
}
