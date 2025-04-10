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
	"github.com/openshift-kni/eco-goinfra/pkg/nmstate"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/define"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/frrconfig"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netinittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netnmstate"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netparam"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/frr"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/metallbenv"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/tsparams"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

var _ = Describe("BGP remote-dynamicAS", Ordered, Label(tsparams.LabelDynamicRemoteASTestCases),
	ContinueOnFailure, func() {
		var (
			err                          error
			frrk8sPods                   []*pod.Builder
			srIovInterfacesUnderTest     []string
			nodeNetConfigPolicyName      = "eth-int-worker0"
			bfdProfile                   *metallb.BFDBuilder
			ipAddressPool                *metallb.IPAddressPoolBuilder
			ethernetIPAddresses          = []string{"10.100.100.254/24", "2001:100::1/64"}
			ipAddressPoolRange           = []string{"3.3.3.1", "3.3.3.240"}
			externalAdvertisedIPv4Routes = []string{"192.168.100.0/24", "192.168.200.0/24"}
			externalAdvertisedIPv6Routes = []string{"2001:100::0/64", "2001:200::0/64"}
		)

		BeforeAll(func() {
			By("Getting MetalLb load balancer ip addresses")
			ipv4metalLbIPList, ipv6metalLbIPList, err = metallbenv.GetMetalLbIPByIPStack()
			Expect(err).ToNot(HaveOccurred(), tsparams.MlbAddressListError)

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
			Expect(err).ToNot(HaveOccurred(), "Failed to create metalLb daemonset")

			err = metallbenv.IsEnvVarMetalLbIPinNodeExtNetRange(ipv4NodeAddrList, ipv4metalLbIPList, nil)
			Expect(err).ToNot(HaveOccurred(), "Failed to validate metalLb exported ip address")

			By("Collecting interface information to enable a node ethernet interface")
			srIovInterfacesUnderTest, err = NetConfig.GetSriovInterfaces(1)
			Expect(err).ToNot(HaveOccurred(), "Failed to retrieve SR-IOV interfaces for testing")

			By(fmt.Sprintf(
				"Enable ethernet interface %s on %s", srIovInterfacesUnderTest[0],
				workerNodeList[0].Definition.Name))

			enableEthernetInterface(nodeNetConfigPolicyName, workerNodeList[0].Definition.Name,
				srIovInterfacesUnderTest[0])

			By("Collecting the frrk8sPod Pod list for worker nodes from the cnfWorkerNodeList")
			frrk8sPods = verifyAndCreateFRRk8sPodList()

			By("Creating a bfd profile")
			bfdProfile, err = metallb.NewBFDBuilder(APIClient, "bfdprofile", NetConfig.MlbOperatorNamespace).
				WithRcvInterval(300).WithTransmitInterval(300).WithEchoInterval(100).
				WithEchoMode(true).WithPassiveMode(false).WithMinimumTTL(5).
				WithMultiplier(3).Create()
			Expect(err).ToNot(HaveOccurred(), "Failed to create BFD profile")
			Expect(bfdProfile.Exists()).To(BeTrue(), "BFD profile doesn't exist")

			By("Creating host-device NAD for external FRR pod")
			_, err = define.HostDeviceNad(APIClient, frrconfig.ExternalMacVlanNADName, srIovInterfacesUnderTest[0],
				tsparams.TestNamespaceName, nad.IPAMStatic())
			Expect(err).ToNot(HaveOccurred(), "Failed to create a network-attachment-definition")

			By("Create a single IPAddressPool")
			ipAddressPool = createIPAddressPool("ipaddresspool", ipAddressPoolRange)

			By("Creating a MetalLB service")
			setupMetalLbService(tsparams.MetallbServiceName, netparam.IPV4Family, ipAddressPool, "Cluster")

			By(fmt.Sprintf(
				"Creating a BGPAdvertisement with the nodeSelector to %s", workerNodeList[0].Definition.Name))
			worker0NodeLabel := []metav1.LabelSelector{
				{MatchLabels: map[string]string{netparam.LabelHostName: workerNodeList[0].Definition.Name}},
			}

			setupBgpAdvertisement("bgpadvertisment", tsparams.CustomCommunity, ipAddressPool.Definition.Name,
				200, []string{tsparams.BgpPeerName1}, worker0NodeLabel)
		})

		AfterAll(func() {
			By("Remove custom metallb test label from nodes")
			removeNodeLabel(workerNodeList, metalLbTestsLabel)

			By("Removing secondary interface on worker node 0")
			ethIntWorker0Policy := nmstate.NewPolicyBuilder(APIClient, nodeNetConfigPolicyName, NetConfig.WorkerLabelMap).
				WithAbsentInterface(srIovInterfacesUnderTest[0])
			err = netnmstate.UpdatePolicyAndWaitUntilItsAvailable(netparam.DefaultTimeout, ethIntWorker0Policy)
			Expect(err).ToNot(HaveOccurred(), "Failed to update NMState network policy")

			By("Removing NMState policies")
			err = nmstate.CleanAllNMStatePolicies(APIClient)
			Expect(err).ToNot(HaveOccurred(), "Failed to remove all NMState policies")

			By("Clean metallb operator and test namespaces")
			resetOperatorAndTestNS()
		})

		AfterEach(func() {
			By("Clean metallb operator and test namespaces")
			resetTestNSBetweenTestCases()
		})

		It("Verify iBGP peering established with BGP Unnumbered",
			reportxml.ID("80393"), func() {
				By("Verifying that the frrk8sPod deployment is in Ready state and create a list of the pods on " +
					"worker nodes.")
				fmt.Println("frrk8sPods[0]", frrk8sPods[0].Definition.Name)
				By("Verify link local IP address on worker frrk8s pod")
				linkLocalAddress := getLinkLocalAddress(frrk8sPods[0], srIovInterfacesUnderTest[0], tsparams.FRRContainerName)

				By("Create a frr config-map")
				frrConfigMap := createConfigMapWithUnnumbered(tsparams.LocalBGPASN, tsparams.LocalBGPASN,
					srIovInterfacesUnderTest[0], linkLocalAddress, externalAdvertisedIPv4Routes, externalAdvertisedIPv6Routes,
					false, false)

				By("Creating static ip annotation for master FRR pod")
				frrStaticIPAnnotation := pod.StaticIPAnnotationWithInterfaceAndNamespace(
					tsparams.ExternalMacVlanNADName, tsparams.TestNamespaceName, srIovInterfacesUnderTest[0],
					ethernetIPAddresses)

				frrPod := createFrrPod(
					workerNodeList[1].Object.Name, frrConfigMap.Definition.Name, []string{}, frrStaticIPAnnotation)

				By("Creating BGP Peers")
				createBGPPeerUnnumberedAndVerifyIfItsReady(tsparams.BgpPeerName1, tsparams.BgpPeerDynamicASiBGP,
					srIovInterfacesUnderTest[0], bfdProfile.Definition.Name, tsparams.LocalBGPASN, 0, false,
					0, frrk8sPods)

				By("Checking that BGP session is established and up")
				verifyMetalLbBGPSessionsAreUPOnFrrPod(frrPod, []string{srIovInterfacesUnderTest[0]})
			})

		It("Verify eBGP peering established with IPv4 prefixes advertised and received by the FRR worker node",
			reportxml.ID("80394"), func() {
				By("Verifying that the frrk8sPod deployment is in Ready state and create a list of the pods on " +
					"worker nodes.")
				By("Creating nginx test pod on worker node 0")
				setupNGNXPod(workerNodeList[0].Definition.Name)

				By("Creating static ip annotation for master FRR pod")
				frrStaticIPAnnotation := pod.StaticIPAnnotationWithInterfaceAndNamespace(
					tsparams.ExternalMacVlanNADName, tsparams.TestNamespaceName, srIovInterfacesUnderTest[0],
					ethernetIPAddresses)

				By("Verify link local IP address on worker frrk8s pod")
				linkLocalAddress := getLinkLocalAddress(frrk8sPods[0], srIovInterfacesUnderTest[0], tsparams.FRRContainerName)
				fmt.Println("linkLocalAddress: ", linkLocalAddress)

				By("Create a frr config-map")
				frrConfigMap := createConfigMapWithUnnumbered(tsparams.RemoteBGPASN, tsparams.LocalBGPASN,
					srIovInterfacesUnderTest[0], linkLocalAddress, externalAdvertisedIPv4Routes, externalAdvertisedIPv6Routes,
					false, true)

				By("Creating BGP Peers")
				createBGPPeerUnnumberedAndVerifyIfItsReady(tsparams.BgpPeerName1, tsparams.BgpPeerDynamicASeBGP,
					srIovInterfacesUnderTest[0], bfdProfile.Definition.Name, tsparams.LocalBGPASN, 0, false,
					0, frrk8sPods)

				frrPod := createFrrPod(
					workerNodeList[1].Object.Name, frrConfigMap.Definition.Name, []string{}, frrStaticIPAnnotation)

				By("Verify link local IP address on worker frrk8s pod")
				linkLocalAddress = getLinkLocalAddress(frrk8sPods[0], srIovInterfacesUnderTest[0], tsparams.FRRContainerName)
				fmt.Println("linkLocalAddress: ", linkLocalAddress)

				By("Checking that BGP session is established and up")
				verifyMetalLbBGPSessionsAreUPOnFrrPod(frrPod, []string{srIovInterfacesUnderTest[0]})

				By("Validating BGP route prefix")
				validatePrefix(frrPod, netparam.IPV4Family, netparam.IPSubnetInt32, []string{linkLocalAddress},
					ipAddressPoolRange)
			})

		It("Verify iBGP peering established with configured peerASN and unnumbered",
			reportxml.ID("80395"), func() {
				By("Verifying that the frrk8sPod deployment is in Ready state and create a list of the pods on " +
					"worker nodes.")
				By("Creating nginx test pod on worker node 0")
				setupNGNXPod(workerNodeList[0].Definition.Name)

				By("Creating static ip annotation for master FRR pod")
				frrStaticIPAnnotation := pod.StaticIPAnnotationWithInterfaceAndNamespace(
					tsparams.ExternalMacVlanNADName, tsparams.TestNamespaceName, srIovInterfacesUnderTest[0],
					ethernetIPAddresses)

				By("Verify link local IP address on worker frrk8s pod")
				linkLocalAddress := getLinkLocalAddress(frrk8sPods[0], srIovInterfacesUnderTest[0], tsparams.FRRContainerName)
				fmt.Println("linkLocalAddress: ", linkLocalAddress)

				By("Create a frr config-map")
				frrConfigMap := createConfigMapWithUnnumbered(tsparams.LocalBGPASN, tsparams.LocalBGPASN,
					srIovInterfacesUnderTest[0], linkLocalAddress, externalAdvertisedIPv4Routes, externalAdvertisedIPv6Routes,
					false, true)

				By("Creating BGP Peers")
				createBGPPeerUnnumberedAndVerifyIfItsReady(tsparams.BgpPeerName1, "",
					srIovInterfacesUnderTest[0], bfdProfile.Definition.Name, tsparams.LocalBGPASN, tsparams.LocalBGPASN, false,
					0, frrk8sPods)

				frrPod := createFrrPod(
					workerNodeList[1].Object.Name, frrConfigMap.Definition.Name, []string{}, frrStaticIPAnnotation)

				By("Verify link local IP address on worker frrk8s pod")
				linkLocalAddress = getLinkLocalAddress(frrk8sPods[0], srIovInterfacesUnderTest[0], tsparams.FRRContainerName)

				By("Checking that BGP session is established and up")
				verifyMetalLbBGPSessionsAreUPOnFrrPod(frrPod, []string{srIovInterfacesUnderTest[0]})

				By("Validating BGP route prefix")
				validatePrefix(frrPod, netparam.IPV4Family, netparam.IPSubnetInt32, []string{linkLocalAddress},
					ipAddressPoolRange)
			})
	})

func enableEthernetInterface(policyName, nodeName, interfaceName string) {
	ethernetInterface := nmstate.NewPolicyBuilder(APIClient, policyName, map[string]string{
		"kubernetes.io/hostname": nodeName,
	})

	ethernetInterface.WithEthernetInterface(interfaceName, "10.100.100.1", "2001:100::1")

	_, err := ethernetInterface.Create()
	Expect(err).ToNot(HaveOccurred(),
		"fail to add an IP address to interface: %s", interfaceName)
}

func createConfigMapWithUnnumbered(localAS, remoteAS int, interfaceName, peerLinkLocalAddress string,
	externalAdvertisedIPv4Routes,
	externalAdvertisedIPv6Routes []string, multiHop, bfd bool) *configmap.Builder {
	frrBFDConfig := frr.DefineBGPConfigWithUnnumbered(localAS, remoteAS, interfaceName, peerLinkLocalAddress,
		externalAdvertisedIPv4Routes, externalAdvertisedIPv6Routes, multiHop, bfd)
	configMapData := frrconfig.DefineBaseConfig(frrconfig.DaemonsFile, frrBFDConfig, "")
	frrConfigMap, err := configmap.NewBuilder(APIClient, "frr-master-node-config", tsparams.TestNamespaceName).
		WithData(configMapData).Create()
	Expect(err).ToNot(HaveOccurred(), "Failed to create config map")

	return frrConfigMap
}

func getLinkLocalAddress(frrPod *pod.Builder, interfaceName, containerName string) string {
	var linkLocal string

	Eventually(func() string {
		interfaceStatus, err := frr.GetInterfaceStatus(frrPod, interfaceName, containerName)
		Expect(err).ToNot(HaveOccurred(), "Failed to verify interface details")

		for _, addr := range interfaceStatus.IPAddresses {
			if strings.HasPrefix(addr.Address, "fe80::") {
				linkLocal = strings.Split(addr.Address, "/")[0]

				return linkLocal
			}
		}

		return ""
	}, 30*time.Second, tsparams.DefaultRetryInterval).Should(HavePrefix("fe80::"),
		"Expected a link-local address starting with fe80::")

	return linkLocal
}
