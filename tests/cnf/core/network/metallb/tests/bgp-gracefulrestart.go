package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/openshift-kni/eco-goinfra/pkg/configmap"
	"github.com/openshift-kni/eco-goinfra/pkg/mco"
	"github.com/openshift-kni/eco-goinfra/pkg/nad"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netenv"
	"gopkg.in/k8snetworkplumbingwg/multus-cni.v4/pkg/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift-kni/eco-goinfra/pkg/metallb"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ignition "github.com/coreos/ignition/v2/config/v3_4/types"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/ipaddr"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netinittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netparam"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/cmd"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/frr"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/metallbenv"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/tsparams"
	ocpoperatorv1 "github.com/openshift/api/operator/v1"
	apimachinerytype "k8s.io/apimachinery/pkg/types"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

var _ = Describe("BGP Graceful Restart", Ordered, Label(tsparams.LabelGRTestCases), ContinueOnFailure, func() {
	var (
		ipAddressPool      *metallb.IPAddressPoolBuilder
		frrPod             *pod.Builder
		bgpPeer            *metallb.BGPPeerBuilder
		ipv4MetalLbIPList  []string
		err                error
		workerLabelMap     map[string]string
		addressPool        = []string{"4.4.4.1", "4.4.4.240"}
		masterNodeList     []*nodes.Builder
		staticIPAnnotation []*types.NetworkSelectionElement
	)

	BeforeAll(func() {
		By("Getting MetalLb load balancer ip addresses")
		ipv4MetalLbIPList, _, err = metallbenv.GetMetalLbIPByIPStack()
		Expect(err).ToNot(HaveOccurred(), tsparams.MlbAddressListError)

		if len(ipv4MetalLbIPList) < 2 {
			Skip("MetalLb GR tests require 2 ip addresses. Please check ECO_CNF_CORE_NET_MLB_ADDR_LIST env var")
		}

		By("Getting external nodes ip addresses")
		cnfWorkerNodeList, err := nodes.List(APIClient,
			metav1.ListOptions{LabelSelector: labels.Set(NetConfig.WorkerLabelMap).String()})
		Expect(err).ToNot(HaveOccurred(), "Failed to discover worker nodes")

		By("Selecting worker node for BGP GR tests")
		workerLabelMap, workerNodeList = setWorkerNodeListAndLabelForBfdTests(cnfWorkerNodeList, metalLbTestsLabel)

		ipv4NodeAddrList, err = nodes.ListExternalIPv4Networks(
			APIClient, metav1.ListOptions{LabelSelector: labels.Set(workerLabelMap).String()})
		Expect(err).ToNot(HaveOccurred(), "Failed to collect external nodes ip addresses")

		err = metallbenv.IsEnvVarMetalLbIPinNodeExtNetRange(ipv4NodeAddrList, ipv4MetalLbIPList, nil)
		Expect(err).ToNot(HaveOccurred(), "Failed to validate metalLb exported ip address")

		By("Listing master nodes")
		masterNodeList, err = nodes.List(APIClient,
			metav1.ListOptions{LabelSelector: labels.Set(NetConfig.ControlPlaneLabelMap).String()})
		Expect(err).ToNot(HaveOccurred(), "Fail to list master nodes")
	})

	BeforeEach(func() {
		By("Creating a new instance of MetalLB Speakers on workers")
		err = metallbenv.CreateNewMetalLbDaemonSetAndWaitUntilItsRunning(tsparams.DefaultTimeout, workerLabelMap)
		Expect(err).ToNot(HaveOccurred(), "Failed to recreate metalLb daemonset")

		By("Creating External NAD")
		createExternalNad(tsparams.ExternalMacVlanNADName)

		By("Creating static ip annotation")
		staticIPAnnotation = pod.StaticIPAnnotation(
			externalNad.Definition.Name, []string{fmt.Sprintf("%s/%s", ipv4MetalLbIPList[0], "24")})

		By("Creating an IPAddressPool and BGPAdvertisement")
		ipAddressPool = setupBgpAdvertisement(addressPool, 32)

		By("Creating nginx test pod on worker node")
		setupNGNXPod(workerNodeList[0].Definition.Name)
	})

	AfterAll(func() {
		if len(cnfWorkerNodeList) > 2 {
			By("Remove custom metallb test label from nodes")
			removeNodeLabel(workerNodeList, metalLbTestsLabel)
		}
	})

	AfterEach(func() {
		By("Cleaning MetalLb operator namespace")
		resetOperatorAndTestNS()
	})

	Context("without bfd", func() {
		BeforeEach(func() {
			By("Creating MetalLb configMap")
			masterConfigMap := createConfigMap(tsparams.LocalBGPASN, ipv4NodeAddrList, false, false)

			By("Creating FRR Pod")
			frrPod = createFrrPod(
				masterNodeList[0].Object.Name, masterConfigMap.Definition.Name, []string{}, staticIPAnnotation)

			By("Configuring BGP peers without GracefulRestart")
			bgpPeer, err = metallb.NewBPGPeerBuilder(APIClient, "testpeer", NetConfig.MlbOperatorNamespace,
				ipv4MetalLbIPList[0], tsparams.LocalBGPASN, tsparams.LocalBGPASN).WithPassword(tsparams.BGPPassword).Create()
			Expect(err).ToNot(HaveOccurred(), "Fail to create bgpPeer")

			By("Checking that BGP sessions are established and up")
			verifyMetalLbBGPSessionsAreUPOnFrrPod(frrPod, ipv4NodeAddrList)
		})

		It("Verify basic functionality with externalTrafficPolicy set to Local", reportxml.ID("77074"), func() {
			gracefulRestartTest(
				"Local", addressPool[0], ipv4MetalLbIPList[0], ipv4NodeAddrList, frrPod, bgpPeer, ipAddressPool, workerLabelMap)
		})

		It("Verify basic functionality with externalTrafficPolicy set to Cluster", reportxml.ID("77077"), func() {
			gracefulRestartTest(
				"Cluster", addressPool[0], ipv4MetalLbIPList[0], ipv4NodeAddrList, frrPod, bgpPeer, ipAddressPool, workerLabelMap)

			By("Verifying that the routes have been removed after the gracefulRestart timeout")
			waitForGracefulRestartTimeout(frrPod, ipv4NodeAddrList[0])
			verifyRoutesAmount(frrPod, 0)
		})

	})

	Context("with bfd", func() {
		var (
			bfdProfile   *metallb.BFDBuilder
			bfdConfigMap *configmap.Builder
		)

		BeforeAll(func() {
			By("Editing the MachineConfiguration CR named cluster to change node disruption policies")
			updateMachineConfigurationNodeDisruptionPolicy()
		})

		BeforeEach(func() {
			By("Creating BFD profile")
			frrk8sPods, err := pod.List(APIClient, NetConfig.MlbOperatorNamespace, metav1.ListOptions{
				LabelSelector: tsparams.FRRK8sDefaultLabel,
			})
			Expect(err).ToNot(HaveOccurred(), "Failed to list pods")
			bfdProfile = createBFDProfileAndVerifyIfItsReady(frrk8sPods)

			By("Creating FRR config of the external BGP peer")
			bfdConfigMap = createConfigMap(tsparams.LocalBGPASN, ipv4NodeAddrList, false, true)
		})

		Context("single hop", func() {
			BeforeEach(func() {
				By("Creating BGP peer")
				bgpPeer, err = metallb.NewBPGPeerBuilder(APIClient, "testpeer", NetConfig.MlbOperatorNamespace,
					ipv4MetalLbIPList[0], tsparams.LocalBGPASN, tsparams.LocalBGPASN).
					WithBFDProfile(bfdProfile.Definition.Name).WithPassword(tsparams.BGPPassword).
					WithGracefulRestart(true).Create()
				Expect(err).ToNot(HaveOccurred(), "Fail to create bgpPeer")

				By("Creating external FRR Pod with network and IP address")
				frrPod = createFrrPod(
					masterNodeList[0].Object.Name, bfdConfigMap.Object.Name, []string{}, staticIPAnnotation)

				By("Checking that BGP and BFD sessions are established and up")
				verifyMetalLbBFDAndBGPSessionsAreUPOnFrrPod(frrPod, ipv4NodeAddrList)

				By("Checking that BGP GracefulRestart is enabled on the neighbors")
				verifyGREnabledOnNeighbors(frrPod, ipv4NodeAddrList)
			})

			AfterEach(func() {
				By("Applying the MachineConfig to remove the created nft policy")
				createMCAndWaitforMCPStable(tsparams.DeleteAllRules)
				err := mco.NewMCBuilder(APIClient, "98-nft").Delete()
				Expect(err).ToNot(HaveOccurred(), "Failed to get the machineConfig")

				By("Checking that BGP and BFD sessions are established and up")
				verifyMetalLbBFDAndBGPSessionsAreUPOnFrrPod(frrPod, ipv4NodeAddrList)
			})

			It("Test Compatibility with BFD", reportxml.ID("77101"), func() {
				gracefulRestartTestWithBFD(
					"Cluster", addressPool[0], ipv4MetalLbIPList[0], ipv4NodeAddrList, frrPod, ipAddressPool, workerLabelMap)
			})
		})
		Context("multi hop", func() {
			var (
				speakerRoutesMap  map[string]string
				masterClientPodIP = "172.16.0.1"
				frrMasterIPs      = []string{"172.16.0.253", "172.16.0.254"}
			)
			BeforeEach(func() {
				By("Collecting information before test")
				frrk8sPods, err := pod.List(APIClient, NetConfig.MlbOperatorNamespace, metav1.ListOptions{
					LabelSelector: tsparams.FRRK8sDefaultLabel,
				})
				Expect(err).ToNot(HaveOccurred(), "Failed to list frrk8s pods")

				speakerRoutesMap, err = buildRoutesMap(frrk8sPods, ipv4MetalLbIPList)
				Expect(err).ToNot(HaveOccurred(), "Failed to build frrk8s route map")

				By("Creating internal NAD")
				masterBridgePlugin, err := nad.NewMasterBridgePlugin("internalnad", "br0").
					WithIPAM(nad.IPAMStatic()).GetMasterPluginConfig()
				Expect(err).ToNot(HaveOccurred(), "Failed to create master bridge plugin setting")
				internalNad, err := nad.NewBuilder(APIClient, "internal", tsparams.TestNamespaceName).
					WithMasterPlugin(masterBridgePlugin).Create()
				Expect(err).ToNot(HaveOccurred(), "Failed to create internal NAD")

				By("Configuring Local GW mode")
				setLocalGWMode(true)

				By("Creating FRR pod one on master node")
				createFrrPodOnMasterNodeAndWaitUntilRunning("frronmaster1",
					ipv4MetalLbIPList[0], netparam.IPSubnet24, frrMasterIPs[0], internalNad.Definition.Name,
					masterNodeList[0].Object.Name, addressPool[0], ipv4NodeAddrList[0])

				By("Creating FRR pod two on master node")
				createFrrPodOnMasterNodeAndWaitUntilRunning("frronmaster2",
					ipv4MetalLbIPList[1], netparam.IPSubnet24, frrMasterIPs[1], internalNad.Definition.Name,
					masterNodeList[0].Object.Name, addressPool[0], ipv4NodeAddrList[1])

				By("Creating FRR pod in the test namespace")
				frrPod = createFrrPod(
					masterNodeList[0].Object.Name,
					bfdConfigMap.Object.Name,
					[]string{},
					pod.StaticIPAnnotation(internalNad.Definition.Name,
						[]string{fmt.Sprintf("%s/%s", masterClientPodIP, netparam.IPSubnet24)}))

				// Add static routes from client towards Speaker via router internal IPs
				for index, workerAddress := range removePrefixFromIPList(ipv4NodeAddrList) {
					buffer, err := cmd.SetRouteOnPod(frrPod, workerAddress, frrMasterIPs[index])
					Expect(err).ToNot(HaveOccurred(), buffer.String())
				}

				By("Creating BGP peer")
				bgpPeer, err = metallb.NewBPGPeerBuilder(APIClient, "testpeer", NetConfig.MlbOperatorNamespace,
					masterClientPodIP, tsparams.LocalBGPASN, tsparams.LocalBGPASN).
					WithBFDProfile(bfdProfile.Definition.Name).WithPassword(tsparams.BGPPassword).
					WithGracefulRestart(true).Create()
				Expect(err).ToNot(HaveOccurred(), "Fail to create bgpPeer")

				By("Adding static routes to the speakers")
				for _, frrk8sPod := range frrk8sPods {
					out, err := frr.SetStaticRoute(frrk8sPod, "add", masterClientPodIP, speakerRoutesMap)
					Expect(err).ToNot(HaveOccurred(), out)
				}

				By("Checking that BGP and BFD sessions are established and up")
				verifyMetalLbBFDAndBGPSessionsAreUPOnFrrPod(frrPod, removePrefixFromIPList(ipv4NodeAddrList))
			})

			AfterEach(func() {
				By("Removing static routes from the speakers")
				frrk8sPods, err := pod.List(APIClient, NetConfig.MlbOperatorNamespace, metav1.ListOptions{
					LabelSelector: tsparams.FRRK8sDefaultLabel,
				})
				Expect(err).ToNot(HaveOccurred(), "Failed to list pods")
				for _, frrk8sPod := range frrk8sPods {
					out, err := frr.SetStaticRoute(frrk8sPod, "del", masterClientPodIP, speakerRoutesMap)
					Expect(err).ToNot(HaveOccurred(), out)
				}

				By("Reverting Local GW mode")
				setLocalGWMode(false)

				By("Applying the MachineConfig to remove the created nft policy")
				createMCAndWaitforMCPStable(tsparams.DeleteAllRules)
				err = mco.NewMCBuilder(APIClient, "98-nft").Delete()
				Expect(err).ToNot(HaveOccurred(), "Failed to get the machineConfig")
			})

			It("Scenario with BFD", reportxml.ID("77125"), func() {
				gracefulRestartTestWithBFD(
					"Local", addressPool[0], masterClientPodIP, ipv4NodeAddrList, frrPod, ipAddressPool, workerLabelMap)
			})
		})

	})
})

func verifyRoutesAmount(masterNodeFRRPod *pod.Builder, amountRoutes int) {
	Eventually(func() int {
		bgpStatus, err := frr.GetBGPStatus(masterNodeFRRPod, strings.ToLower(netparam.IPV4Family))
		Expect(err).ToNot(HaveOccurred(), "Failed to get BGP status from FRR pod")

		return len(bgpStatus.Routes)
	}, 1*time.Minute, tsparams.DefaultRetryInterval).Should(Equal(amountRoutes),
		fmt.Sprintf("Expected %d routes, but the actual count did not match", amountRoutes))
}

func gracefulRestartTest(
	extTrafficPolicy corev1.ServiceExternalTrafficPolicyType,
	destIPaddress,
	ipv4metalLbIP string,
	ipv4WorkerNodeList []string,
	frrPod *pod.Builder,
	bgpPeer *metallb.BGPPeerBuilder,
	ipAddressPool *metallb.IPAddressPoolBuilder,
	workerLabelMap map[string]string) {
	By("Creating a MetalLB service")
	setupMetalLbService(netparam.IPV4Family, ipAddressPool, extTrafficPolicy)

	By("Checking whether routes have been added to the BGP table on the external BGP peer")
	validatePrefix(frrPod, netparam.IPV4Family, removePrefixFromIPList(ipv4WorkerNodeList), []string{destIPaddress}, 32)

	By("Running http check")
	validateCurlToNginxPod(frrPod, ipv4metalLbIP, destIPaddress)

	By("Deleting the MetalLB CR to remove all frr-k8s pods and simulate a GR scenario (e.g., during an upgrade)")
	removeMetalLbCR()

	By("Checking that BGP sessions are down")
	verifyMetalLbBGPSessionsAreDown(frrPod, ipv4NodeAddrList)

	By("Checking that routes have been remove from the BGP table on the external BGP peer")
	verifyRoutesAmount(frrPod, 0)

	By("Running http check")
	validateCurlFailsToNginxPod(frrPod, ipv4metalLbIP, destIPaddress)

	By("Creating a new instance of MetalLB Speakers on workers")

	err := metallbenv.CreateNewMetalLbDaemonSetAndWaitUntilItsRunning(tsparams.DefaultTimeout, workerLabelMap)
	Expect(err).ToNot(HaveOccurred(), "Failed to recreate metalLb daemonset")

	By("Configuring BGP peers with GR enabled")

	_, err = bgpPeer.WithGracefulRestart(true).Update(false)
	Expect(err).ToNot(HaveOccurred(), "Failed to configure GracefulRestart")

	By("Checking that BGP sessions are established and up")
	verifyMetalLbBGPSessionsAreUPOnFrrPod(frrPod, ipv4NodeAddrList)

	By("Checking that BGP GracefulRestart is enabled on the neighbors")
	verifyGREnabledOnNeighbors(frrPod, ipv4NodeAddrList)

	By("Checking whether routes have been added to the BGP table on the external BGP peer")
	validatePrefix(frrPod, netparam.IPV4Family, removePrefixFromIPList(ipv4WorkerNodeList), []string{destIPaddress}, 32)

	By("Running http check")
	validateCurlToNginxPod(frrPod, ipv4metalLbIP, destIPaddress)

	By("Deleting the MetalLB CR to remove all frr-k8s pods and simulate a GR scenario (e.g., during an upgrade)")
	removeMetalLbCR()

	By("Checking that BGP sessions are down")
	verifyMetalLbBGPSessionsAreDown(frrPod, ipv4NodeAddrList)

	By("Checking whether routes have been added to the BGP table on the external BGP peer")
	validatePrefix(frrPod, netparam.IPV4Family, removePrefixFromIPList(ipv4WorkerNodeList), []string{destIPaddress}, 32)

	By("Running http check")

	_, err = cmd.Curl(frrPod, ipv4metalLbIP, destIPaddress, netparam.IPV4Family, tsparams.FRRSecondContainerName)
	Expect(err).ToNot(HaveOccurred(), "Unexpectedly succeeded in curling the Nginx pod")
}

func gracefulRestartTestWithBFD(
	extTrafficPolicy corev1.ServiceExternalTrafficPolicyType,
	destIPaddress,
	ipv4metalLbIP string,
	ipv4WorkerNodeList []string,
	frrPod *pod.Builder,
	ipAddressPool *metallb.IPAddressPoolBuilder,
	workerLabelMap map[string]string) {
	By("Creating a MetalLB service")
	setupMetalLbService(netparam.IPV4Family, ipAddressPool, extTrafficPolicy)

	By("Checking whether routes have been added to the BGP table on the external BGP peer")
	validatePrefix(
		frrPod, netparam.IPV4Family, removePrefixFromIPList(ipv4WorkerNodeList), []string{destIPaddress}, 32)

	By("Running http check before the fail over test")
	validateCurlToNginxPod(frrPod, ipv4metalLbIP, destIPaddress)

	By("Deleting the MetalLB CR to remove all frr-k8s pods and simulate a GR scenario (e.g., during an upgrade)")
	removeMetalLbCR()

	By("Checking that BGP sessions are down")
	verifyMetalLbBGPSessionsAreDown(frrPod, ipv4NodeAddrList)

	By("Checking whether routes have been added to the BGP table on the external BGP peer")
	validatePrefix(
		frrPod, netparam.IPV4Family, removePrefixFromIPList(ipv4WorkerNodeList), []string{destIPaddress}, 32)

	By("Running http check to confirm that the routes still exist")
	validateCurlToNginxPod(frrPod, ipv4metalLbIP, destIPaddress)

	By("Creating a new instance of MetalLB Speakers on workers")

	err := metallbenv.CreateNewMetalLbDaemonSetAndWaitUntilItsRunning(tsparams.DefaultTimeout, workerLabelMap)
	Expect(err).ToNot(HaveOccurred(), "Failed to recreate metalLb daemonset")

	By("Checking that BGP and BFD sessions are established and up")
	verifyMetalLbBFDAndBGPSessionsAreUPOnFrrPod(frrPod, ipv4NodeAddrList)

	By("Applying the MachineConfig to add nft rule that blocks BGP and BFD port to emulate link down event")
	createMCAndWaitforMCPStable(tsparams.BlockBGPBFDPorts)

	By("Checking that BFD sessions are down")

	err = netenv.BFDHasStatus(frrPod, ipaddr.RemovePrefix(ipv4NodeAddrList[0]), "up")
	Expect(err).Should(HaveOccurred(), "BFD is not in expected down state")

	By("Checking that BGP sessions are down")
	verifyMetalLbBGPSessionsAreDown(frrPod, ipv4NodeAddrList)

	By("Checking that routes have been remove from the BGP table on the external BGP peer")
	verifyRoutesAmount(frrPod, 0)

	By("Validating that web server is not reachable by curl command")
	validateCurlFailsToNginxPod(frrPod, ipv4metalLbIP, destIPaddress)
}

func verifyMetalLbBGPSessionsAreDown(frrPod *pod.Builder, peerAddrList []string) {
	for _, peerAddress := range removePrefixFromIPList(peerAddrList) {
		// Change the timeout timer with caution, as it is critically important for the Graceful Restart tests.
		Eventually(frr.BGPNeighborshipHasState,
			30*time.Second, tsparams.DefaultRetryInterval).
			WithArguments(frrPod, peerAddress, "Established").ShouldNot(
			BeTrue(), "Failed to receive BGP status Down")
	}
}

func verifyGREnabledOnNeighbors(frrPod *pod.Builder, peerAddrList []string) {
	for _, peerAddress := range removePrefixFromIPList(peerAddrList) {
		grStatus, err := frr.GetGracefulRestartStatus(frrPod, peerAddress)

		Expect(err).ToNot(HaveOccurred(), "Failed to get GracefulRestart status")
		Expect(grStatus.RemoteGrMode).To(Or(Equal("Helper"), Equal("Restart")),
			fmt.Sprintf("GracefulRestart is not enabled on the peer %s", peerAddress))
	}
}

func waitForGracefulRestartTimeout(frrPod *pod.Builder, peerAddress string) {
	grStatus, err := frr.GetGracefulRestartStatus(frrPod, ipaddr.RemovePrefix(peerAddress))
	Expect(err).ToNot(HaveOccurred(), "Failed to get GracefulRestart status")
	time.Sleep(time.Duration(grStatus.Timers.RestartTimerRemaining) * time.Second)
}

func validateCurlToNginxPod(frrPod *pod.Builder, srcIP, destIP string) {
	Eventually(func() error {
		_, err := cmd.Curl(
			frrPod, srcIP, destIP, netparam.IPV4Family, tsparams.FRRSecondContainerName)

		return err
	}, 40*time.Second, tsparams.DefaultRetryInterval).Should(BeNil(), "Failed to curl nginx pod")
}

func validateCurlFailsToNginxPod(frrPod *pod.Builder, srcIP, destIP string) {
	Consistently(func() error {
		_, err := cmd.Curl(
			frrPod, srcIP, destIP, netparam.IPV4Family, tsparams.FRRSecondContainerName)

		return err
	}, 30*time.Second, tsparams.DefaultRetryInterval).ShouldNot(BeNil(), "Expected curl to fail, but it succeeded")
}

func removeMetalLbCR() {
	metalLbNs, err := namespace.Pull(APIClient, NetConfig.MlbOperatorNamespace)
	Expect(err).ToNot(HaveOccurred(), "Failed to pull metalLb operator namespace")
	err = metalLbNs.CleanObjects(tsparams.DefaultTimeout, metallb.GetMetalLbIoGVR())
	Expect(err).ToNot(HaveOccurred(), "Failed to remove MetalLB object from operator namespace")
}

// updateMachineConfigurationNodeDisruptionPolicy adds a Machine Config disruption policy to skip reboot
// if the nftables service performs actions such as Reload or Restart.
func updateMachineConfigurationNodeDisruptionPolicy() {
	By("should update machineconfiguration cluster")

	jsonBytes := []byte(`
	{"spec":{"nodeDisruptionPolicy":
	  {"files": [{"actions":
	[{"restart": {"serviceName": "nftables.service"},"type": "Restart"}],
	"path": "/etc/sysconfig/nftables.conf"}],
	"units":
	[{"actions":
	[{"reload": {"serviceName":"nftables.service"},"type": "Reload"},
	{"type": "DaemonReload"}],"name": "nftables.service"}]}}}`)

	err := APIClient.Patch(context.TODO(), &ocpoperatorv1.MachineConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
	}, client.RawPatch(apimachinerytype.MergePatchType, jsonBytes))
	Expect(err).ToNot(HaveOccurred(),
		"Failed to update the machineconfiguration cluster file")
}

func createMCAndWaitforMCPStable(fileContentString string) {
	truePointer := true
	stringgzip := "gzip"
	mode := 384
	nftablesConfig := `[Unit]
Description=Netfilter Tables
Documentation=man:nft(8)
Wants=network-pre.target
Before=network-pre.target

[Service]
Type=oneshot
ProtectSystem=full
ProtectHome=true
ExecStart=/sbin/nft -f /etc/sysconfig/nftables.conf
ExecReload=/sbin/nft -f /etc/sysconfig/nftables.conf
ExecStop=/sbin/nft 'add table inet custom_table; delete table inet custom_table'
RemainAfterExit=yes

[Install]
WantedBy=multi-user.target`

	ignitionConfig := ignition.Config{
		Ignition: ignition.Ignition{
			Version: "3.4.0",
		},
		Systemd: ignition.Systemd{
			Units: []ignition.Unit{
				{
					Enabled:  &truePointer,
					Name:     "nftables.service",
					Contents: &nftablesConfig,
				},
			},
		},
		Storage: ignition.Storage{
			Files: []ignition.File{
				{
					Node: ignition.Node{
						Overwrite: &truePointer,
						Path:      "/etc/sysconfig/nftables.conf",
					},
					FileEmbedded1: ignition.FileEmbedded1{
						Contents: ignition.Resource{
							Compression: &stringgzip,
							Source:      &fileContentString,
						},
						Mode: &mode,
					},
				},
			},
		},
	}

	finalIgnitionConfig, err := json.Marshal(ignitionConfig)
	Expect(err).ToNot(HaveOccurred(), "Failed to serialize ignition config")

	_, err = mco.NewMCBuilder(APIClient, "98-nft").
		WithLabel("machineconfiguration.openshift.io/role", NetConfig.CnfMcpLabel).
		WithRawConfig(finalIgnitionConfig).
		Create()
	Expect(err).ToNot(HaveOccurred(), "Failed to create nftables machine config")

	err = netenv.WaitForMcpStable(APIClient, 10*time.Minute, 1*time.Minute, NetConfig.CnfMcpLabel)
	Expect(err).ToNot(HaveOccurred(), "Failed to wait for MCP to be stable")
}
