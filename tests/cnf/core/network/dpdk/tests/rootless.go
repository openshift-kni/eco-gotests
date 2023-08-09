package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/mco"
	"github.com/openshift-kni/eco-goinfra/pkg/nad"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-goinfra/pkg/scc"
	"github.com/openshift-kni/eco-goinfra/pkg/sriov"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/dpdk/internal/link"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/dpdk/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/define"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netinittools"
	"github.com/openshift-kni/eco-gotests/tests/internal/cluster"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
	"gopkg.in/k8snetworkplumbingwg/multus-cni.v4/pkg/types"
	v1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	srIovNetworkTwoResName       = "dpdkpolicytwo"
	srIovPolicyOneResName        = "dpdkpolicyone"
	setSEBool                    = "setsebool container_use_devices "
	srIovNetworkOneName          = "sriov-net-one"
	srIovNetworkTwoName          = "sriov-net-two"
	tapNetworkOne                = "tap-one"
	tapNetworkTwo                = "tap-two"
	vlanNetworkOne               = "vlan-one"
	vlanNetworkTwo               = "vlan-two"
	macVlanNetworkOne            = "mac-vlan-one"
	macVlanNetworkTwo            = "mac-vlan-two"
	ipVlanNetworkOne             = "ip-vlan-one"
	tapOneInterfaceName          = "ext0"
	tapTwoInterfaceName          = "ext1"
	defaultWhereAboutNetwork     = "1.1.1.0/24"
	defaultWhereAboutGW          = "1.1.1.254"
	networkForRouteTest          = "10.10.10.0/24"
	dpdkServerMac                = "60:00:00:00:00:02"
	dpdkClientMac                = "60:00:00:00:00:01"
	dpdkServerMacTwo             = "60:00:00:00:00:04"
	dpdkClientMacTwo             = "60:00:00:00:00:03"
	firstInterfaceBasedOnTapOne  = "ext0.1"
	secondInterfaceBasedOnTapOne = "ext0.2"
	firstInterfaceBasedOnTapTwo  = "ext1.1"
	timeoutError                 = "command terminated with exit code 124"
	maxMulticastNoiseRate        = 5000
	minimumExpectedDPDKRate      = 1000000
	customUserID                 = 2005
	customGroupID                = 2005
	dummyVlanID                  = 200
)

var (
	sleepCMD         = []string{"/bin/bash", "-c", "sleep INF"}
	rootUser         = int64(0)
	customSCCGroupID = int64(customGroupID)
	customSCCUserID  = int64(customUserID)
	hugePagesGroup   = int64(801)
	falseFlag        = false
	trueFlag         = true
	workerNodes      *nodes.Builder
	mcp              *mco.MCPBuilder

	serverSC = v1.SecurityContext{
		RunAsUser: &rootUser,
		Capabilities: &v1.Capabilities{
			Add: []v1.Capability{"IPC_LOCK", "SYS_RESOURCE", "NET_RAW"},
		},
	}

	clientPodSC = v1.PodSecurityContext{
		FSGroup:    &hugePagesGroup,
		RunAsGroup: &customSCCGroupID,
		SeccompProfile: &v1.SeccompProfile{
			Type: "RuntimeDefault",
		},
	}

	clientSC = v1.SecurityContext{
		Capabilities: &v1.Capabilities{
			Drop: []v1.Capability{"ALL"},
			Add:  []v1.Capability{"IPC_LOCK", "SYS_RESOURCE", "NET_ADMIN"},
		},
		RunAsUser:                &customSCCUserID,
		Privileged:               &falseFlag,
		RunAsNonRoot:             &trueFlag,
		AllowPrivilegeEscalation: &trueFlag,
	}

	// enabledSysctlFlags represents sysctl configuration with few enable flags for sysctl plugin.
	enabledSysctlFlags = map[string]string{
		"net.ipv6.conf.IFNAME.accept_ra":  "1",
		"net.ipv4.conf.IFNAME.arp_accept": "1",
	}
	// disabledSysctlFlags represents sysctl configuration with few disabled flags for sysctl plugin.
	disabledSysctlFlags = map[string]string{
		"net.ipv6.conf.IFNAME.accept_ra":  "0",
		"net.ipv4.conf.IFNAME.arp_accept": "0",
	}

	defaultWhereaboutIPAM = nad.IPAMWhereAbouts(defaultWhereAboutNetwork, defaultWhereAboutGW)
	vlanID                uint16
)

type podNetworkAnnotation struct {
	Name      string   `json:"name"`
	Interface string   `json:"interface"`
	Ips       []string `json:"ips,omitempty"`
	Mac       string   `json:"mac"`
	Default   bool     `json:"default,omitempty"`
	DNS       struct {
	} `json:"dns"`
	DeviceInfo struct {
		Type    string `json:"type"`
		Version string `json:"version"`
		Pci     struct {
			PciAddress string `json:"pci-address"`
		} `json:"pci"`
	} `json:"device-info,omitempty"`
}

var _ = Describe("rootless", Ordered, Label(tsparams.LabelSuite), ContinueOnFailure, func() {

	Context("server-tx, client-rx connectivity test on different nodes", Label("rootless"), func() {
		BeforeAll(func() {
			By("Discover worker nodes")
			workerNodes = nodes.NewBuilder(APIClient, NetConfig.WorkerLabelMap)
			err := workerNodes.Discover()
			Expect(err).ToNot(HaveOccurred(), "Fail to discover nodes")

			By(fmt.Sprintf("Pulling MCP based on label %s", NetConfig.CnfMcpLabel))
			mcp, err = mco.Pull(APIClient, NetConfig.CnfMcpLabel)
			Expect(err).ToNot(HaveOccurred(), "Fail to pull MCP ")

			By("Collecting SR-IOV interface for rootless dpdk tests")
			srIovInterfacesUnderTest, err := NetConfig.GetSriovInterfaces(1)
			Expect(err).ToNot(HaveOccurred(), "Failed to retrieve SR-IOV interfaces for testing")

			By("Creating first dpdk-policy")
			_, err = sriov.NewPolicyBuilder(
				APIClient,
				"dpdk-policy-one",
				NetConfig.SriovOperatorNamespace,
				srIovPolicyOneResName,
				5,
				[]string{fmt.Sprintf("%s#0-1", srIovInterfacesUnderTest[0])},
				NetConfig.WorkerLabelMap).WithMTU(1500).WithDevType("vfio-pci").WithVhostNet(true).Create()
			Expect(err).ToNot(HaveOccurred(), "Fail to create dpdk policy")

			By("Creating second dpdk-policy")
			_, err = sriov.NewPolicyBuilder(
				APIClient,
				"dpdk-policy-two",
				NetConfig.SriovOperatorNamespace,
				srIovNetworkTwoResName,
				5,
				[]string{fmt.Sprintf("%s#2-4", srIovInterfacesUnderTest[0])},
				NetConfig.WorkerLabelMap).WithMTU(1500).WithDevType("vfio-pci").WithVhostNet(false).Create()
			Expect(err).ToNot(HaveOccurred(), "Fail to create dpdk policy")

			By("Waiting until cluster is stable")
			err = mcp.WaitToBeStableFor(time.Minute, tsparams.MCOWaitTimeout)
			Expect(err).ToNot(HaveOccurred(), "Failed to wait until cluster is stable")

			By("Setting selinux flag container_use_devices to 1 on all compute nodes")
			err = cluster.ExecCmd(APIClient, NetConfig.WorkerLabel, setSEBool+"1")
			Expect(err).ToNot(HaveOccurred(), "Fail to enable selinux flag")

			By("Setting vlan ID")
			vlanID, err = NetConfig.GetVLAN()
			Expect(err).ToNot(HaveOccurred(), "Fail to set vlanID")
			Expect(dummyVlanID).ToNot(BeEquivalentTo(vlanID),
				"Both vlans have the same ID. Please select different ID using ECO_CNF_CORE_NET_VLAN env variable")
		})

		It("single VF, multiple tap devices, multiple mac-vlans", polarion.ID("63806"), func() {
			defineAndCreateSrIovNetworks(0, vlanID)
			defineAndCreateTapNADs(nil, nil)

			By("Creating first mac-vlan NetworkAttachmentDefinition")
			_, err := define.MacVlanNad(
				APIClient, macVlanNetworkOne, tsparams.TestNamespaceName, tapOneInterfaceName, defaultWhereaboutIPAM)
			Expect(err).ToNot(HaveOccurred(), "Fail to create first mac-vlan NetworkAttachmentDefinition")

			By("Creating second mac-vlan NetworkAttachmentDefinition")
			_, err = define.MacVlanNad(
				APIClient, macVlanNetworkTwo, tsparams.TestNamespaceName, tapTwoInterfaceName, defaultWhereaboutIPAM)
			Expect(err).ToNot(HaveOccurred(), "Fail to create second mac-vlan NetworkAttachmentDefinition")

			By("Creating server pod")

			serverPodNetConfig := pod.StaticIPAnnotationWithMacAndNamespace(
				srIovNetworkTwoName, tsparams.TestNamespaceName, dpdkServerMac)
			defineAndCreateDPDKPod(
				"serverpod",
				workerNodes.Objects[1].Definition.Name,
				serverSC,
				nil,
				serverPodNetConfig,
				defineTestServerPmdCmd(dpdkClientMac, "${PCIDEVICE_OPENSHIFT_IO_DPDKPOLICYTWO}", ""))

			By("Creating client pod")
			clientPodNetConfig := definePodNetwork([]map[string]string{
				{"netName": srIovNetworkOneName, "macAddr": dpdkClientMac},
				{"netName": tapNetworkOne, "intName": tapOneInterfaceName},
				{"netName": tapNetworkTwo, "intName": tapTwoInterfaceName},
				{"netName": macVlanNetworkOne, "intName": firstInterfaceBasedOnTapOne, "macAddr": dpdkClientMac},
				{"netName": macVlanNetworkOne, "intName": secondInterfaceBasedOnTapOne},
				{"netName": macVlanNetworkTwo, "intName": firstInterfaceBasedOnTapTwo}})

			clientPod := defineAndCreateDPDKPod(
				"clientpod",
				workerNodes.Objects[1].Definition.Name,
				clientSC,
				&clientPodSC,
				clientPodNetConfig,
				sleepCMD,
			)
			rxTrafficOnClientPod(clientPod, defineTestPmdCmd(tapOneInterfaceName, "${PCIDEVICE_OPENSHIFT_IO_DPDKPOLICYONE}"))
			checkRxOutputRateForInterfaces(
				clientPod,
				map[string]int{
					tapOneInterfaceName:          minimumExpectedDPDKRate,
					tapTwoInterfaceName:          maxMulticastNoiseRate,
					firstInterfaceBasedOnTapOne:  minimumExpectedDPDKRate,
					secondInterfaceBasedOnTapOne: maxMulticastNoiseRate},
			)
			rxTrafficOnClientPod(clientPod, defineTestPmdCmd(tapTwoInterfaceName, "${PCIDEVICE_OPENSHIFT_IO_DPDKPOLICYONE}"))
			checkRxOutputRateForInterfaces(
				clientPod,
				map[string]int{tapTwoInterfaceName: minimumExpectedDPDKRate, firstInterfaceBasedOnTapTwo: maxMulticastNoiseRate})
		})

		It("multiple VFs, one tap plus MAC-VLAN, second tap plus 2 VLANs, filter untagged and tagged traffic",
			polarion.ID("63818"), func() {
				defineAndCreateSrIovNetworks(0, vlanID)
				defineAndCreateTapNADs(nil, nil)

				By("Creating mac-vlan one")
				_, err := define.MacVlanNad(
					APIClient, macVlanNetworkOne, tsparams.TestNamespaceName, tapOneInterfaceName, defaultWhereaboutIPAM)
				Expect(err).ToNot(HaveOccurred(), "Fail to create first mac-vlan NetworkAttachmentDefinition")

				By("Creating vlan one NetworkAttachmentDefinition")
				defineAndCreateVlanNad(vlanNetworkOne, tapTwoInterfaceName, vlanID, nad.IPAMStatic())

				By("Creating vlan two NetworkAttachmentDefinition")
				defineAndCreateVlanNad(vlanNetworkTwo, tapTwoInterfaceName, dummyVlanID, nad.IPAMStatic())

				serverPodOneNetConfig := pod.StaticIPAnnotationWithMacAndNamespace(
					srIovNetworkOneName, tsparams.TestNamespaceName, dpdkServerMac)

				By("Creating first server pod")
				srvNetOne := defineTestServerPmdCmd(dpdkClientMac, "${PCIDEVICE_OPENSHIFT_IO_DPDKPOLICYONE}", "")
				defineAndCreateDPDKPod(
					"serverpod-one",
					workerNodes.Objects[0].Definition.Name,
					serverSC,
					nil,
					serverPodOneNetConfig,
					srvNetOne)

				By("Creating second server pod")
				serverPodTwoNetConfig := pod.StaticIPAnnotationWithMacAndNamespace(
					srIovNetworkTwoName, tsparams.TestNamespaceName, dpdkServerMacTwo)

				srvNetTwo := defineTestServerPmdCmd(dpdkClientMacTwo, "${PCIDEVICE_OPENSHIFT_IO_DPDKPOLICYTWO}", "")
				defineAndCreateDPDKPod(
					"serverpod-two",
					workerNodes.Objects[1].Definition.Name,
					serverSC,
					nil,
					serverPodTwoNetConfig,
					srvNetTwo)

				By("Creating client pod")
				firstVlanInterfaceBasedOnTapTwo := fmt.Sprintf("%s.%d", tapTwoInterfaceName, vlanID)
				secondVlanInterfaceBasedOnTapTwo := fmt.Sprintf("%s.%d", tapTwoInterfaceName, dummyVlanID)
				clientPodNetConfig := definePodNetwork([]map[string]string{
					{"netName": srIovNetworkOneName, "macAddr": dpdkClientMac},
					{"netName": srIovNetworkOneName, "macAddr": dpdkClientMacTwo},
					{"netName": tapNetworkOne, "intName": tapOneInterfaceName},
					{"netName": tapNetworkTwo, "intName": tapTwoInterfaceName},
					{"netName": macVlanNetworkOne, "intName": firstInterfaceBasedOnTapOne, "macAddr": dpdkClientMac},
					{"netName": vlanNetworkOne, "intName": firstVlanInterfaceBasedOnTapTwo, "ipAddr": "1.1.1.1/24"},
					{"netName": vlanNetworkTwo, "intName": secondVlanInterfaceBasedOnTapTwo, "ipAddr": "2.2.2.2/24"}})
				clientPod := defineAndCreateDPDKPod(
					"clientpod", workerNodes.Objects[1].Definition.Name, clientSC, &clientPodSC, clientPodNetConfig, sleepCMD)

				By("Collecting PCI Address")
				Eventually(
					isPciAddressAvailable, tsparams.WaitTimeout, tsparams.RetryInterval).WithArguments(clientPod).Should(BeTrue())
				pciAddressList, err := getPCIAddressListFromSrIovNetworkName(
					clientPod.Object.Annotations["k8s.v1.cni.cncf.io/network-status"], srIovNetworkOneName)
				Expect(err).ToNot(HaveOccurred(), "Fail to collect PCI addresses")

				By("Running client dpdk-testpmd")
				rxTrafficOnClientPod(clientPod, defineTestPmdCmd(tapOneInterfaceName, pciAddressList[0]))

				By("Checking the rx output of tap ext0 device")
				checkRxOutputRateForInterfaces(
					clientPod, map[string]int{
						tapOneInterfaceName:         minimumExpectedDPDKRate,
						firstInterfaceBasedOnTapOne: minimumExpectedDPDKRate})
				rxTrafficOnClientPod(clientPod, defineTestPmdCmd(tapTwoInterfaceName, pciAddressList[1]))
				checkRxOutputRateForInterfaces(clientPod, map[string]int{
					tapTwoInterfaceName:              minimumExpectedDPDKRate,
					firstVlanInterfaceBasedOnTapTwo:  minimumExpectedDPDKRate,
					secondVlanInterfaceBasedOnTapTwo: maxMulticastNoiseRate})
			})

		It("multiple VFs, one tap plus IP-VLANs, second tap plus plus VLAN and sysctl, filter untagged and tagged"+
			" traffic, add and remove routes", polarion.ID("63878"), func() {
			defineAndCreateSrIovNetworks(0, vlanID)
			defineAndCreateTapNADs(enabledSysctlFlags, disabledSysctlFlags)

			serverPodOneNetConfig := pod.StaticIPAnnotationWithMacAndNamespace(
				srIovNetworkOneName, tsparams.TestNamespaceName, dpdkServerMac)

			By("Creating ip-vlan interface")
			defineAndCreateIPVlanNad(ipVlanNetworkOne, tapOneInterfaceName, nad.IPAMStatic())

			By("Creating vlan-one interface")
			defineAndCreateVlanNad(vlanNetworkOne, tapTwoInterfaceName, vlanID, nad.IPAMWhereAbouts("2.2.2.0/24", "2.2.2.254"))

			By("Creating first server pod")
			srvNetOne := defineTestServerPmdCmd(
				dpdkClientMac, "${PCIDEVICE_OPENSHIFT_IO_DPDKPOLICYONE}", "1.1.1.50,1.1.1.100")
			defineAndCreateDPDKPod(
				"serverpod-one", workerNodes.Objects[0].Definition.Name, serverSC, nil, serverPodOneNetConfig, srvNetOne)
			serverPodTwoNetConfig := pod.StaticIPAnnotationWithMacAndNamespace(
				srIovNetworkTwoName, tsparams.TestNamespaceName, dpdkServerMacTwo)

			By("Creating second server pod")
			srvNetTwo := defineTestServerPmdCmd(dpdkClientMacTwo, "${PCIDEVICE_OPENSHIFT_IO_DPDKPOLICYTWO}", "")
			defineAndCreateDPDKPod(
				"serverpod-two", workerNodes.Objects[0].Definition.Name, serverSC, nil, serverPodTwoNetConfig, srvNetTwo)

			By("Creating client pod")
			vlanInterfaceName := fmt.Sprintf("%s.%d", tapTwoInterfaceName, vlanID)
			clientPodNetConfig := definePodNetwork([]map[string]string{
				{"netName": srIovNetworkOneName, "macAddr": dpdkClientMac},
				{"netName": srIovNetworkOneName, "macAddr": dpdkClientMacTwo},
				{"netName": tapNetworkOne, "intName": tapOneInterfaceName},
				{"netName": tapNetworkTwo, "intName": tapTwoInterfaceName},
				{"netName": ipVlanNetworkOne, "intName": firstInterfaceBasedOnTapOne, "ipAddr": "1.1.1.100/24"},
				{"netName": ipVlanNetworkOne, "intName": secondInterfaceBasedOnTapOne, "ipAddr": "1.1.1.200/24"},
				{"netName": vlanNetworkOne, "intName": vlanInterfaceName}})
			clientPod := defineAndCreateDPDKPod(
				"clientpod", workerNodes.Objects[1].Definition.Name, clientSC, &clientPodSC, clientPodNetConfig, sleepCMD)

			By("Collecting PCI Address")
			Eventually(
				isPciAddressAvailable, tsparams.WaitTimeout, tsparams.RetryInterval).WithArguments(clientPod).Should(BeTrue())
			pciAddressList, err := getPCIAddressListFromSrIovNetworkName(
				clientPod.Object.Annotations["k8s.v1.cni.cncf.io/network-status"], srIovNetworkOneName)
			Expect(err).ToNot(HaveOccurred(), "Fail to collect PCI addresses")

			rxTrafficOnClientPod(clientPod, defineTestPmdCmd(tapOneInterfaceName, pciAddressList[0]))
			checkRxOutputRateForInterfaces(
				clientPod, map[string]int{
					tapOneInterfaceName:          minimumExpectedDPDKRate,
					firstInterfaceBasedOnTapOne:  minimumExpectedDPDKRate,
					secondInterfaceBasedOnTapOne: maxMulticastNoiseRate,
				})
			rxTrafficOnClientPod(clientPod, defineTestPmdCmd(tapTwoInterfaceName, pciAddressList[1]))
			checkRxOutputRateForInterfaces(
				clientPod, map[string]int{
					tapTwoInterfaceName:          minimumExpectedDPDKRate,
					vlanInterfaceName:            minimumExpectedDPDKRate,
					secondInterfaceBasedOnTapOne: maxMulticastNoiseRate,
				})

			By("Verifying sysctl plugin configuration")

			nextHopIPAddr := "1.1.1.10"
			verifySysctlKernelParametersConfiguredOnPodInterface(clientPod, enabledSysctlFlags, tapOneInterfaceName)
			verifySysctlKernelParametersConfiguredOnPodInterface(clientPod, disabledSysctlFlags, tapTwoInterfaceName)

			By("Adding route to rootless pod")
			_, err = setRouteOnPod(clientPod, networkForRouteTest, nextHopIPAddr, firstInterfaceBasedOnTapOne)
			Expect(err).ToNot(HaveOccurred())

			By("Verifying if route exist in rootless pod")
			verifyIfRouteExist(clientPod, "10.10.10.0", nextHopIPAddr, firstInterfaceBasedOnTapOne, true)

			By("Removing route from rootless pod")
			_, err = delRouteOnPod(clientPod, networkForRouteTest, nextHopIPAddr, firstInterfaceBasedOnTapOne)
			Expect(err).ToNot(HaveOccurred())

			By("Verifying if route was removed from rootless pod")
			verifyIfRouteExist(clientPod, "10.10.10.0", nextHopIPAddr, firstInterfaceBasedOnTapOne, false)
		})
	})

	AfterEach(func() {
		By("Removing all srIovNetworks")
		err := sriov.CleanAllNetworksByTargetNamespace(
			APIClient, NetConfig.SriovOperatorNamespace, tsparams.TestNamespaceName, metaV1.ListOptions{})
		Expect(err).ToNot(HaveOccurred(), "Fail to clean srIovNetworks")

		By("Removing all pods from test namespace")
		runningNamespace, err := namespace.Pull(APIClient, tsparams.TestNamespaceName)
		Expect(err).ToNot(HaveOccurred(), "Failed to pull namespace")
		Expect(runningNamespace.CleanObjects(300*time.Second, pod.GetGVR(), nad.GetGVR())).ToNot(HaveOccurred())

	})

	AfterAll(func() {
		By("Removing all pods from test namespace")
		runningNamespace, err := namespace.Pull(APIClient, tsparams.TestNamespaceName)
		Expect(err).ToNot(HaveOccurred(), "Failed to pull namespace")
		Expect(runningNamespace.CleanObjects(tsparams.WaitTimeout, pod.GetGVR())).ToNot(HaveOccurred(),
			"Fail to clean namespace")

		By("Removing all SR-IOV Policy")
		err = sriov.CleanAllNetworkNodePolicies(APIClient, NetConfig.SriovOperatorNamespace, metaV1.ListOptions{})
		Expect(err).ToNot(HaveOccurred(), "Fail to clean srIovPolicy")

		By("Removing all srIovNetworks")
		err = sriov.CleanAllNetworksByTargetNamespace(
			APIClient, NetConfig.SriovOperatorNamespace, tsparams.TestNamespaceName, metaV1.ListOptions{})
		Expect(err).ToNot(HaveOccurred(), "Fail to clean sriov networks")

		By("Removing SecurityContextConstraints")
		testScc, err := scc.Pull(APIClient, "scc-test-admin")
		if err == nil {
			err = testScc.Delete()
			Expect(err).ToNot(HaveOccurred(), "Fail to remove scc")
		}

		By("Waiting until cluster is stable")
		err = mcp.WaitToBeStableFor(time.Minute, tsparams.MCOWaitTimeout)
		Expect(err).ToNot(HaveOccurred(), "Fail to wait until cluster is stable")
	})
})

func defineAndCreateTapNADs(firstTapSysctlConfig, secondTapSysctlConfig map[string]string) {
	By("Creating first tap interface")

	_, err := define.TapNad(APIClient, tapNetworkOne, tsparams.TestNamespaceName, 0, 0, firstTapSysctlConfig)
	Expect(err).ToNot(HaveOccurred(), "Fail to create first tap NetworkAttachmentDefinition")

	_, err = define.TapNad(
		APIClient, tapNetworkTwo, tsparams.TestNamespaceName, customUserID, customGroupID, secondTapSysctlConfig)
	Expect(err).ToNot(HaveOccurred(), "Fail to create second tap NetworkAttachmentDefinition")
}

func defineAndCreateSrIovNetworks(firstSrIovNetworkVlanID, secondSrIovNetworkVlanID uint16) {
	By("Creating srIovNetwork sriov-net-one")
	defineAndCreateSrIovNetwork(srIovNetworkOneName, srIovPolicyOneResName, firstSrIovNetworkVlanID)

	By("Creating srIovNetwork sriov-net-two")
	defineAndCreateSrIovNetwork(srIovNetworkTwoName, srIovNetworkTwoResName, secondSrIovNetworkVlanID)
}

func defineAndCreateIPVlanNad(name, intName string, ipam *nad.IPAM) *nad.Builder {
	ipVlanNad, err := define.IPVlanNad(APIClient, name, tsparams.TestNamespaceName, intName, ipam)
	Expect(err).ToNot(HaveOccurred(), "Fail to create ipVlan NetworkAttachmentDefinition")

	return ipVlanNad
}

func defineAndCreateVlanNad(name, intName string, vlanID uint16, ipam *nad.IPAM) *nad.Builder {
	vlanNad, err := define.VlanNad(APIClient, name, tsparams.TestNamespaceName, intName, vlanID, ipam)
	Expect(err).ToNot(HaveOccurred(), "Fail to create vlan NetworkAttachmentDefinition")

	return vlanNad
}

func defineAndCreateSrIovNetwork(srIovNetwork, resName string, vlanID uint16) {
	srIovNetworkObject := sriov.NewNetworkBuilder(
		APIClient, srIovNetwork, NetConfig.SriovOperatorNamespace, tsparams.TestNamespaceName, resName).
		WithMacAddressSupport()

	if vlanID != 0 {
		srIovNetworkObject.WithVLAN(vlanID)
	}

	srIovNetworkObject, err := srIovNetworkObject.Create()
	Expect(err).ToNot(HaveOccurred(), "Fail to create sriov network")

	Eventually(func() bool {
		_, err := nad.Pull(APIClient, srIovNetworkObject.Object.Name, tsparams.TestNamespaceName)

		return err == nil
	}, tsparams.WaitTimeout, tsparams.RetryInterval).Should(BeTrue(), "Fail to pull NetworkAttachmentDefinition")
}

func defineAndCreateDPDKPod(
	podName,
	nodeName string,
	securityContext v1.SecurityContext,
	podSC *v1.PodSecurityContext,
	serverPodNetConfig []*types.NetworkSelectionElement,
	podCmd []string) *pod.Builder {
	dpdkContainer := pod.NewContainerBuilder(podName, NetConfig.DpdkTestContainer, podCmd)
	dpdkContainerCfg, err := dpdkContainer.WithSecurityContext(&securityContext).
		WithResourceLimit("2Gi", "1Gi", 4).
		WithResourceRequest("2Gi", "1Gi", 4).WithEnvVar("RUN_TYPE", "testcmd").
		GetContainerCfg()
	Expect(err).ToNot(HaveOccurred(), "Fail to set dpdk container")

	dpdkPod := pod.NewBuilder(APIClient, podName, tsparams.TestNamespaceName,
		NetConfig.DpdkTestContainer).WithSecondaryNetwork(serverPodNetConfig).
		DefineOnNode(nodeName).RedefineDefaultContainer(*dpdkContainerCfg).WithHugePages()

	if podSC != nil {
		dpdkPod = dpdkPod.WithSecurityContext(podSC)
	}

	dpdkPod, err = dpdkPod.CreateAndWaitUntilRunning(tsparams.WaitTimeout)
	Expect(err).ToNot(HaveOccurred(), "Fail to create server pod")

	return dpdkPod
}

func defineTestServerPmdCmd(ethPeer, pciAddress, txIPs string) []string {
	baseCmd := fmt.Sprintf("dpdk-testpmd -a %s -- --forward-mode txonly --eth-peer=0,%s", pciAddress, ethPeer)
	if txIPs != "" {
		baseCmd += fmt.Sprintf(" --tx-ip=%s", txIPs)
	}

	baseCmd += " --stats-period 5"

	return []string{"/bin/bash", "-c", baseCmd}
}

func definePodNetwork(podNetMapList []map[string]string) []*types.NetworkSelectionElement {
	var clientPodNetConfig []*types.NetworkSelectionElement

	for _, podNet := range podNetMapList {
		_, isMacAddressSet := podNet["macAddr"]
		_, isIntNameSet := podNet["intName"]
		_, isIPPresent := podNet["ipAddr"]

		if isIntNameSet && isIPPresent {
			clientPodNetConfig = append(clientPodNetConfig, pod.StaticIPAnnotationWithInterfaceAndNamespace(
				podNet["netName"], tsparams.TestNamespaceName, podNet["intName"], []string{podNet["ipAddr"]})...)

			continue
		}

		if isMacAddressSet && isIntNameSet {
			clientPodNetConfig = append(clientPodNetConfig, pod.StaticIPAnnotationWithInterfaceMacAndNamespace(
				podNet["netName"], tsparams.TestNamespaceName, podNet["intName"], podNet["macAddr"])...)

			continue
		}

		if isMacAddressSet {
			clientPodNetConfig = append(clientPodNetConfig, pod.StaticIPAnnotationWithMacAndNamespace(
				podNet["netName"], tsparams.TestNamespaceName, podNet["macAddr"])...)

			continue
		}

		clientPodNetConfig = append(
			clientPodNetConfig, pod.StaticIPAnnotationWithInterfaceAndNamespace(
				podNet["netName"], tsparams.TestNamespaceName, podNet["intName"], nil)...)
	}

	return clientPodNetConfig
}

func defineTestPmdCmd(interfaceName string, pciAddress string) string {
	return fmt.Sprintf("timeout 20 dpdk-testpmd "+
		"--vdev=virtio_user0,path=/dev/vhost-net,queues=2,queue_size=1024,iface=%s "+
		"-a %s -- --stats-period 5", interfaceName, pciAddress)
}

func rxTrafficOnClientPod(clientPod *pod.Builder, clientRxCmd string) {
	clientOut, err := clientPod.ExecCommand([]string{"/bin/bash", "-c", clientRxCmd})
	if err.Error() != timeoutError {
		Expect(err).ToNot(HaveOccurred(), "Fail to exec cmd")
	}

	By("Parsing output from the DPDK application")
	Expect(checkRxOnly(clientOut.String())).Should(BeTrue(), "Fail to process output from dpdk application")
}

func checkRxOnly(out string) bool {
	lines := strings.Split(out, "\n")
	Expect(len(lines)).To(BeNumerically(">=", 3),
		"Fail line list contains less than 3 elements")

	for i, line := range lines {
		if strings.Contains(line, "NIC statistics for port") {
			if len(lines) > i && getNumberOfPackets(lines[i+1], "RX") > 0 {
				return true
			}
		}
	}

	return false
}

func getNumberOfPackets(line, firstFieldSubstr string) int {
	splitLine := strings.Fields(line)
	Expect(splitLine[0]).To(ContainSubstring(firstFieldSubstr), "Fail to find expected substring")
	Expect(len(splitLine)).To(Equal(6), "the slice doesn't contain 6 elements")
	numberOfPackets, err := strconv.Atoi(splitLine[1])
	Expect(err).ToNot(HaveOccurred(), "Fail to convert string to integer")

	return numberOfPackets
}

func checkRxOutputRateForInterfaces(clientPod *pod.Builder, interfaceTrafficRateMap map[string]int) {
	for interfaceName, TrafficRate := range interfaceTrafficRateMap {
		comparator := ">"
		if TrafficRate == 5000 {
			comparator = "<"
		}

		By(fmt.Sprintf("Checking the rx output of %s device", interfaceName))
		Expect(getLinkRx(clientPod, interfaceName)).To(BeNumerically(comparator, TrafficRate),
			"Fail traffic rate is not in expected range")
	}
}

func getLinkRx(runningPod *pod.Builder, linkName string) int {
	linkRawInfo, err := runningPod.ExecCommand(
		[]string{"/bin/bash", "-c", fmt.Sprintf("ip --json -s link show dev %s", linkName)})
	Expect(err).ToNot(HaveOccurred(), "Failed to collect link info")
	linkInfo, err := link.NewBuilder(linkRawInfo)
	Expect(err).ToNot(HaveOccurred(), "Failed to collect link info")

	return linkInfo.GetRxByte()
}

func isPciAddressAvailable(clientPod *pod.Builder) bool {
	if !clientPod.Exists() {
		return false
	}

	podNetAnnotation := clientPod.Object.Annotations["k8s.v1.cni.cncf.io/network-status"]
	if podNetAnnotation == "" {
		return false
	}

	var err error

	pciAddressList, err := getPCIAddressListFromSrIovNetworkName(podNetAnnotation, srIovNetworkOneName)

	if err != nil {
		return false
	}

	if len(pciAddressList) < 2 {
		return false
	}

	return true
}

func getPCIAddressListFromSrIovNetworkName(podNetworkStatus, srIovNetworkName string) ([]string, error) {
	var podNetworkStatusType []podNetworkAnnotation
	err := json.Unmarshal([]byte(podNetworkStatus), &podNetworkStatusType)

	if err != nil {
		return nil, err
	}

	var pciAddressList []string

	for _, networkAnnotation := range podNetworkStatusType {
		if strings.Contains(networkAnnotation.Name, srIovNetworkName) {
			pciAddressList = append(pciAddressList, networkAnnotation.DeviceInfo.Pci.PciAddress)
		}
	}

	return pciAddressList, nil
}

func verifySysctlKernelParametersConfiguredOnPodInterface(
	podUnderTest *pod.Builder, sysctlPluginConfig map[string]string, interfaceName string) {
	for key, value := range sysctlPluginConfig {
		sysctlKernelParam := strings.Replace(key, "IFNAME", interfaceName, 1)

		By(fmt.Sprintf("Validate sysctl flag: %s has the right value in pod's interface: %s",
			sysctlKernelParam, interfaceName))

		cmdBuffer, err := podUnderTest.ExecCommand([]string{"sysctl", "-n", sysctlKernelParam})
		Expect(err).ToNot(HaveOccurred(), "Fail to execute cmd command on the pod")
		Expect(strings.TrimSpace(cmdBuffer.String())).To(BeIdenticalTo(value),
			"sysctl kernel param is not in expected state")
	}
}

func defineRoute(dstNet, nextHop, devName, mode string) []string {
	return []string{"/bin/bash", "-c", fmt.Sprintf("route %s -net %s gw %s dev %s", mode, dstNet, nextHop, devName)}
}

func setRouteOnPod(client *pod.Builder, dstNet, nextHop, devName string) (bytes.Buffer, error) {
	cmd := defineRoute(dstNet, nextHop, devName, "add")

	return client.ExecCommand(cmd)
}

func delRouteOnPod(client *pod.Builder, dstNet, nextHop, devName string) (bytes.Buffer, error) {
	cmd := defineRoute(dstNet, nextHop, devName, "del")

	return client.ExecCommand(cmd)
}

func verifyIfRouteExist(clientPod *pod.Builder, dstNetwork, gateway, nextHostInterface string, expectedState bool) {
	buff, err := clientPod.ExecCommand([]string{"route", "-n"})
	Expect(err).ToNot(HaveOccurred())

	var doesRoutePresent bool

	for _, line := range strings.Split(buff.String(), "\n") {
		if strings.Contains(line, dstNetwork) &&
			strings.Contains(line, gateway) &&
			strings.Contains(line, nextHostInterface) {
			doesRoutePresent = true
		}
	}

	Expect(doesRoutePresent).To(BeIdenticalTo(expectedState), "Fail to find required route")
}
