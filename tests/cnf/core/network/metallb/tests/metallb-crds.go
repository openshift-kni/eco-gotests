package tests

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift-kni/eco-goinfra/pkg/metallb"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"

	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/define"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/frrconfig"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/ipaddr"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netinittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netparam"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/cmd"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/metallbenv"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/tsparams"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

var _ = Describe("MetalLb New CRDs", Ordered, Label("newcrds"), ContinueOnFailure, func() {
	var (
		ipv4metalLBIPList        []string
		addressPool              = []string{"3.3.3.1", "3.3.3.240"}
		ipAddressPool            *metallb.IPAddressPoolBuilder
		l3ClientPod              *pod.Builder
		sriovInterfacesUnderTest []string
		ipSecondaryInterface1    = "3.3.3.10/24"
		ipSecondaryInterface2    = "3.3.3.20/24"
		cnfWorkerNodeList        []*nodes.Builder
	)

	BeforeAll(func() {
		By("Retrieving master nodes")
		masterNodeList, err := nodes.List(APIClient,
			metav1.ListOptions{LabelSelector: labels.Set(NetConfig.ControlPlaneLabelMap).String()})
		Expect(err).ToNot(HaveOccurred(), "An unexpected error occurred while getting master nodes.")
		Expect(len(masterNodeList)).To(BeNumerically(">", 0),
			"Master node list is empty")
		firstMasterNode := masterNodeList[0]

		By("Retrieving worker nodes")
		cnfWorkerNodeList, err = nodes.List(APIClient,
			metav1.ListOptions{LabelSelector: labels.Set(NetConfig.WorkerLabelMap).String()})
		Expect(err).ToNot(HaveOccurred(), "Failed to discover worker nodes")
		Expect(err).ToNot(HaveOccurred(), "An unexpected error occurred while getting worker nodes.")
		Expect(len(cnfWorkerNodeList)).To(BeNumerically(">", 1),
			"Worker node list is less then 2")

		By("Selecting worker node for BFD tests")
		workerLabelMap, workerNodeList = setWorkerNodeListAndLabelForBfdTests(cnfWorkerNodeList, metalLbTestsLabel)

		By("Retrieving available IPv4 addresses for MetalLB testing")
		ipv4metalLBIPList, _, err = metallbenv.GetMetalLbIPByIPStack()
		Expect(err).ToNot(HaveOccurred(), tsparams.MlbAddressListError)
		if len(ipv4metalLBIPList) < 2 {
			Skip("There are not enough IPv4 addresses configured in the env variable")
		}

		By("Retrieving external IPv4 addresses of worker nodes")
		ipv4NodeAddrList, err := nodes.ListExternalIPv4Networks(
			APIClient, metav1.ListOptions{LabelSelector: labels.Set(workerLabelMap).String()})
		Expect(err).ToNot(HaveOccurred(), "Failed to collect external nodes ip addresses")

		By("Setup MetalLB CR")
		err = metallbenv.CreateNewMetalLbDaemonSetAndWaitUntilItsRunning(tsparams.DefaultTimeout, workerLabelMap)
		Expect(err).ToNot(HaveOccurred(), "Failed create MetalLB CR")

		By("Creating nginx test pod")
		setupNGNXPod(workerNodeList[0].Definition.Name)

		By("Creating an IPAddresspool and BGPAdvertisement")
		ipAddressPool = setupBgpAdvertisementAndIPAddressPool(addressPool)

		By("Generating ConfigMap configuration for the external FRR pod")
		masterConfigMap := createConfigMap(tsparams.LocalBGPASN, ipv4NodeAddrList, false, true)

		By("Creating External NAD")
		err = define.CreateExternalNad(APIClient, frrconfig.ExternalMacVlanNADName, tsparams.TestNamespaceName)
		Expect(err).ToNot(HaveOccurred(), "Failed to create the external network-attachment-definition")

		By("Creating FRR-L3client pod on a Master node")
		staticIPAnnotation := pod.StaticIPAnnotation(
			frrconfig.ExternalMacVlanNADName, []string{fmt.Sprintf("%s/%d", ipv4metalLBIPList[0], 24)})

		l3ClientPod = createFrrPod(
			firstMasterNode.Object.Name, masterConfigMap.Definition.Name, []string{}, staticIPAnnotation)

		By("Configuring BGP and BFD")
		frrk8sPods, err := pod.List(APIClient, NetConfig.MlbOperatorNamespace, metav1.ListOptions{
			LabelSelector: tsparams.FRRK8sDefaultLabel,
		})
		Expect(err).ToNot(HaveOccurred(), "Failed to list pods")
		bfdProfile := createBFDProfileAndVerifyIfItsReady(frrk8sPods)

		createBGPPeerAndVerifyIfItsReady(tsparams.BGPTestPeer, ipv4metalLBIPList[0], bfdProfile.Definition.Name,
			tsparams.LocalBGPASN, false, 0,
			frrk8sPods)

		By("Checking that BGP and BFD sessions are established and up")
		verifyMetalLbBFDAndBGPSessionsAreUPOnFrrPod(l3ClientPod, ipv4NodeAddrList)

		By("Configuring Local GW mode")
		setLocalGWMode(true)

		By("Adding IP to a secondary interface on the worker 0")
		sriovInterfacesUnderTest, err = NetConfig.GetSriovInterfaces(1)
		Expect(err).ToNot(HaveOccurred(), "Failed to retrieve SR-IOV interfaces for testing")

		addOrDeleteNodeSecIPAddViaFRRK8S("add", workerNodeList[0].Object.Name,
			ipSecondaryInterface1, sriovInterfacesUnderTest[0])

		By("Creating a L2Advertisement")

		_, err = metallb.NewL2AdvertisementBuilder(
			APIClient, "l2advertisement", NetConfig.MlbOperatorNamespace).
			WithIPAddressPools([]string{ipAddressPool.Definition.Name}).
			WithInterfaces([]string{sriovInterfacesUnderTest[0]}).
			Create()
		Expect(err).ToNot(HaveOccurred(), "An unexpected error occurred while creating L2Advertisement.")
	})

	AfterEach(func() {
		By("Removing IP to a secondary interface from the worker 0")
		addOrDeleteNodeSecIPAddViaFRRK8S("del", workerNodeList[0].Object.Name,
			ipSecondaryInterface1, sriovInterfacesUnderTest[0])

		By("Removing MetalLB CRs and cleaning the test ns")
		resetOperatorAndTestNS()
	})

	AfterAll(func() {
		if len(cnfWorkerNodeList) > 2 {
			removeNodeLabel(workerNodeList, metalLbTestsLabel)
		}

		By("Reverting GW mode to the Sharing")
		setLocalGWMode(false)
	})

	It("Concurrent Layer2 and Layer3 should work concurrently Layer 2 and Layer 3", reportxml.ID("50059"), func() {
		By("Creating MetalLB service")
		setupMetalLbService("service-1", netparam.IPV4Family, ipAddressPool, "Local")

		By(fmt.Sprintf("Creating macvlan NAD with the secondary interface %s", sriovInterfacesUnderTest[0]))
		createExternalNadWithMasterInterface("l2nad", sriovInterfacesUnderTest[0])

		By("Creating L2 client")
		staticIPAnnotation := pod.StaticIPAnnotation("l2nad", []string{ipSecondaryInterface2})

		l2ClientPod, err := pod.NewBuilder(APIClient, "l2client", tsparams.TestNamespaceName, NetConfig.CnfNetTestContainer).
			DefineOnNode(workerNodeList[1].Object.Name).
			WithSecondaryNetwork(staticIPAnnotation).
			CreateAndWaitUntilRunning(time.Minute)
		Expect(err).ToNot(HaveOccurred(), "Failed to create l2 client pod")

		By("Validating that l2 client can curl to LB address")
		httpTrafficValidation(l2ClientPod, ipaddr.RemovePrefix(ipSecondaryInterface2), addressPool[0])

		By("Validating that l3 client can curl to LB address")
		httpTrafficValidation(l3ClientPod, ipv4metalLBIPList[0], addressPool[0], tsparams.FRRSecondContainerName)
	})
})

func addOrDeleteNodeSecIPAddViaFRRK8S(action string,
	workerNodeName string,
	ipaddress string,
	secInterface string) {
	fieldSelector := fmt.Sprintf("spec.nodeName=%s", workerNodeName)

	frrk8sPods, err := pod.List(APIClient, NetConfig.MlbOperatorNamespace, metav1.ListOptions{
		LabelSelector: tsparams.FRRK8sDefaultLabel, FieldSelector: fieldSelector},
	)
	Expect(err).ToNot(HaveOccurred(), "Failed to list frrk8s pods")

	buffer, err := frrk8sPods[0].ExecCommand([]string{"ip", "add", action,
		ipaddress, "dev", secInterface}, "frr")

	if err != nil && strings.Contains(buffer.String(), "already assigned") {
		glog.V(90).Infof("Warning: Address %s is already assigned to %s", ipaddress, secInterface)

		return
	}

	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to %s sec IP address: %s", action, buffer.String()))
}

func httpTrafficValidation(testPod *pod.Builder, srcIPAddress, dstIPAddress string, secContainerName ...string) {
	Eventually(func() error {
		_, err := cmd.Curl(
			testPod, srcIPAddress, dstIPAddress, netparam.IPV4Family, secContainerName...)

		return err
	}, 15*time.Second, 5*time.Second).ShouldNot(HaveOccurred(),
		fmt.Sprintf("Client %s can not curl LB IP address %s",
			testPod.Definition.Name, dstIPAddress))
}
