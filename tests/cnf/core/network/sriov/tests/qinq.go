package tests

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"time"

	"github.com/openshift-kni/eco-goinfra/pkg/nad"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/cmd"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/define"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netconfig"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netparam"
	multus "gopkg.in/k8snetworkplumbingwg/multus-cni.v4/pkg/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-goinfra/pkg/sriov"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netenv"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netinittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/sriov/internal/sriovenv"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/sriov/internal/tsparams"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

var _ = Describe("QinQ", Ordered, Label(tsparams.LabelQinQTestCases), ContinueOnFailure, func() {

	var (
		err                         error
		dot1ad                      = "802.1ad"
		dot1q                       = "802.1q"
		srIovPolicyNetDevice        = "sriovnetpolicy-netdevice"
		srIovNetworkDot1AD          = "sriovnetwork-dot1ad"
		srIovNetworkDot1Q           = "sriovnetwork-dot1q"
		srIovNetworkPromiscuous     = "sriovnetwork-promiscuous"
		srIovPolicyResNameNetDevice = "sriovpolicynetdevice"
		serverNameDot1ad            = "server-1ad"
		serverNameDot1q             = "server-1q"
		clientNameDot1ad            = "client-1ad"
		clientNameDot1q             = "client-1q"
		nadCVLAN                    = "nadcvlan"
		intelDeviceIDE810           = "1593"
		mlxDevice                   = "1017"
		tcpDumpNet1CMD              = []string{"bash", "-c", "tcpdump -i net1 -e > /tmp/tcpdump"}
		tcpDumpReadFileCMD          = []string{"bash", "-c", "tail -20 /tmp/tcpdump"}
		tcpDumpDot1ADOutput         = "(ethertype 802\\.1Q-QinQ \\(0x88a8\\)).*?(ethertype 802\\.1Q, vlan 100)"
		tcpDumpDot1QOutput          = "(ethertype 802\\.1Q \\(0x8100\\)).*?(ethertype 802\\.1Q, vlan 100)"
		workerNodeList              = []*nodes.Builder{}
		promiscVFCommand            string
		srIovInterfacesUnderTest    []string
		sriovDeviceID               string
		switchCredentials           *sriovenv.SwitchCredentials
		switchConfig                *netconfig.NetworkConfig
		switchInterfaces            []string
	)

	serverIPV4IP, _, _ := net.ParseCIDR(tsparams.ServerIPv4IPAddress)
	serverIPV6IP, _, _ := net.ParseCIDR(tsparams.ServerIPv6IPAddress)

	BeforeAll(func() {
		By("Discover worker nodes")
		workerNodeList, err = nodes.List(APIClient,
			metav1.ListOptions{LabelSelector: labels.Set(NetConfig.WorkerLabelMap).String()})
		Expect(err).ToNot(HaveOccurred(), "Fail to discover worker nodes")

		By("Collecting SR-IOV interfaces for qinq testing")
		srIovInterfacesUnderTest, err = NetConfig.GetSriovInterfaces(1)
		Expect(err).ToNot(HaveOccurred(), "Failed to retrieve SR-IOV interfaces for testing")

		By("Define and create a network attachment definition with a C-VLAN 100")
		_, err = define.VlanNad(APIClient, nadCVLAN, tsparams.TestNamespaceName, "net1", 100,
			nad.IPAMStatic())
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Fail to create Network-Attachment-Definition %s",
			nadCVLAN))

		By("Define and create sriov network policy using worker node label")
		_, err = sriov.NewPolicyBuilder(
			APIClient,
			srIovPolicyNetDevice,
			NetConfig.SriovOperatorNamespace,
			srIovPolicyResNameNetDevice,
			10,
			[]string{fmt.Sprintf("%s#0-9", srIovInterfacesUnderTest[0])},
			NetConfig.WorkerLabelMap).Create()
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to create sriovnetwork policy %s",
			srIovPolicyNetDevice))

		By("Waiting until cluster MCP and SR-IOV are stable")
		err = netenv.WaitForSriovAndMCPStable(
			APIClient, tsparams.MCOWaitTimeout, time.Minute, NetConfig.CnfMcpLabel, NetConfig.SriovOperatorNamespace)
		Expect(err).ToNot(HaveOccurred(), "Failed cluster is not stable")

		By("Define and create sriov-network for the promiscuous client")
		_, err = sriov.NewNetworkBuilder(APIClient,
			srIovNetworkPromiscuous, NetConfig.SriovOperatorNamespace, tsparams.TestNamespaceName,
			srIovPolicyResNameNetDevice).WithTrustFlag(true).Create()
		Expect(err).ToNot(HaveOccurred(),
			fmt.Sprintf("Failed to create sriov network srIovNetworkPromiscuous %s", err))

		By("Verify SR-IOV Device IDs for interface under test")
		sriovDeviceID = discoverInterfaceUnderTestDeviceID(srIovInterfacesUnderTest[0],
			workerNodeList[0].Definition.Name)
		Expect(sriovDeviceID).ToNot(BeEmpty(), "Expected sriovDeviceID not to be empty")

		promiscVFCommand = fmt.Sprintf("ethtool --set-priv-flags %s vf-true-promisc-support on",
			srIovInterfacesUnderTest[0])
		if sriovDeviceID == mlxDevice {
			promiscVFCommand = fmt.Sprintf("ip link set %s promisc on",
				srIovInterfacesUnderTest[0])
		}

		By(fmt.Sprintf("Enable VF promiscuous support on %s", srIovInterfacesUnderTest[0]))
		output, err := cmd.RunCommandOnHostNetworkPod(workerNodeList[0].Definition.Name, NetConfig.SriovOperatorNamespace,
			promiscVFCommand)
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to run command on node %s", output))

		By("Configure lab switch interface to support VLAN double tagging")
		switchCredentials, err = sriovenv.NewSwitchCredentials()
		Expect(err).ToNot(HaveOccurred(), "Failed to get switch credentials")

		switchConfig = netconfig.NewNetConfig()
		switchInterfaces, err = switchConfig.GetSwitchInterfaces()
		Expect(err).ToNot(HaveOccurred(), "Failed to get switch interfaces")

		err = enableDot1ADonSwitchInterfaces(switchCredentials, switchInterfaces)
		Expect(err).ToNot(HaveOccurred(), "Failed to enable 802.1AD on the switch")
	})

	Context("802.1AD", func() {
		BeforeAll(func() {
			By("Verify SR-IOV Device IDs for interface under test")

			if sriovDeviceID != intelDeviceIDE810 {
				Skip(fmt.Sprintf("The NIC %s does not support 802.1AD", sriovDeviceID))
			}

			By("Define and create sriov-network with 802.1ad S-VLAN")
			vlan, err := strconv.Atoi(NetConfig.VLAN)
			Expect(err).ToNot(HaveOccurred(), "Failed to convert VLAN value")
			defineAndCreateSrIovNetworkWithQinQ(srIovNetworkDot1AD, srIovPolicyResNameNetDevice, dot1ad,
				uint16(vlan))
		})

		It("Verify network traffic over a 802.1ad QinQ tunnel between two SRIOV pods on the same PF",
			reportxml.ID("71676"), func() {
				By("Define and create a container in promiscuous mode")
				tcpDumpContainer := createPromiscuousClient(workerNodeList[0].Definition.Name,
					tcpDumpNet1CMD)

				By("Define and create a server container")
				serverPod := createServerTestPod(serverNameDot1ad, srIovNetworkDot1AD, nadCVLAN,
					workerNodeList[0].Definition.Name, []string{tsparams.ServerIPv4IPAddress,
						tsparams.ServerIPv6IPAddress})

				By("Define and create a client container")
				clientPod := createClientTestPod(clientNameDot1ad, srIovNetworkDot1AD, nadCVLAN,
					workerNodeList[0].Definition.Name, []string{tsparams.ClientIPv4IPAddress, tsparams.ClientIPv6IPAddress})

				By("Validate IPv4 and IPv6 connectivity between the containers over the qinq tunnel.")
				err = cmd.ICMPConnectivityCheck(serverPod,
					[]string{tsparams.ClientIPv4IPAddress, tsparams.ClientIPv6IPAddress}, "net2")
				Expect(err).ToNot(HaveOccurred(),
					"Failed to ping the client container over the 802.1AD connection")

				By("Validate IPv4 tcp traffic and dot1ad encapsulation from the client to server.")
				validateTCPTraffic(clientPod, serverIPV4IP.String())

				By("Validate IPv6 tcp traffic and dot1ad encapsulation from the client to server.")
				validateTCPTraffic(clientPod, serverIPV6IP.String())

				By("Validate that the TCP traffic is double tagged")
				readAndValidateTCPDump(tcpDumpContainer, tcpDumpReadFileCMD, tcpDumpDot1ADOutput)
			})

		It("Verify network traffic over a 802.1ad QinQ tunnel between two SRIOV containers in different nodes",
			reportxml.ID("71678"), func() {
				By("Define and create a container in promiscuous mode")
				tcpDumpContainer := createPromiscuousClient(workerNodeList[0].Definition.Name,
					tcpDumpNet1CMD)

				By("Define and create a server container")
				serverPod := createServerTestPod(serverNameDot1ad, srIovNetworkDot1AD, nadCVLAN,
					workerNodeList[1].Definition.Name, []string{tsparams.ServerIPv4IPAddress, tsparams.ServerIPv6IPAddress})

				By("Define and create a client container")
				clientPod := createClientTestPod(clientNameDot1ad, srIovNetworkDot1AD, nadCVLAN,
					workerNodeList[0].Definition.Name, []string{tsparams.ClientIPv4IPAddress, tsparams.ClientIPv6IPAddress})

				By("Validate IPv4 and IPv6 connectivity between the containers over the qinq tunnel.")
				err := cmd.ICMPConnectivityCheck(serverPod, []string{tsparams.ClientIPv4IPAddress,
					tsparams.ClientIPv6IPAddress}, "net2")
				Expect(err).ToNot(HaveOccurred(),
					"Failed to ping the client container over the 802.1ad connection")

				By("Validate IPv4 tcp traffic and dot1ad encapsulation from the client to server.")
				validateTCPTraffic(clientPod, serverIPV4IP.String())

				By("Validate IPv6 tcp traffic and dot1ad encapsulation from the client to server.")
				validateTCPTraffic(clientPod, serverIPV6IP.String())

				By("Validate that the TCP traffic is double tagged")
				readAndValidateTCPDump(tcpDumpContainer, tcpDumpReadFileCMD, tcpDumpDot1ADOutput)
			})
	})

	Context("802.1Q", func() {
		BeforeAll(func() {
			By("Define and create sriov-network with 802.1q S-VLAN")
			vlan, err := strconv.Atoi(NetConfig.VLAN)
			Expect(err).ToNot(HaveOccurred(), "Failed to convert VLAN value")
			defineAndCreateSrIovNetworkWithQinQ(srIovNetworkDot1Q, srIovPolicyResNameNetDevice, dot1q,
				uint16(vlan))
		})

		It("Verify network traffic over a 802.1q QinQ tunnel between two SRIOV pods on the same PF",
			reportxml.ID("71677"), func() {
				By("Define and create a container in promiscuous mode")
				tcpDumpContainer := createPromiscuousClient(workerNodeList[0].Definition.Name,
					tcpDumpNet1CMD)

				By("Define and create a server container")
				serverPod := createServerTestPod(serverNameDot1q, srIovNetworkDot1Q, nadCVLAN,
					workerNodeList[0].Definition.Name, []string{tsparams.ServerIPv4IPAddress, tsparams.ServerIPv6IPAddress})
				By("Define and create a client container")
				clientPod := createClientTestPod(clientNameDot1q, srIovNetworkDot1Q, nadCVLAN,
					workerNodeList[0].Definition.Name, []string{tsparams.ClientIPv4IPAddress, tsparams.ClientIPv6IPAddress})

				By("Validate IPv4 and IPv6 connectivity between the containers over the qinq tunnel.")
				err := cmd.ICMPConnectivityCheck(serverPod, []string{tsparams.ClientIPv4IPAddress,
					tsparams.ClientIPv6IPAddress}, "net2")
				Expect(err).ToNot(HaveOccurred(),
					"Failed to ping the client container over the 802.1q connection")

				By("Validate IPv4 tcp traffic and dot1ad encapsulation from the client to server.")
				validateTCPTraffic(clientPod, serverIPV4IP.String())

				By("Validate IPv6 tcp traffic and dot1ad encapsulation from the client to server.")
				validateTCPTraffic(clientPod, serverIPV6IP.String())

				By("Validate that the TCP traffic is double tagged")
				readAndValidateTCPDump(tcpDumpContainer, tcpDumpReadFileCMD, tcpDumpDot1QOutput)
			})

		It("Verify network traffic over a 802.1Q QinQ tunnel between two SRIOV containers in different nodes",
			reportxml.ID("71679"), func() {
				By("Define and create a container in promiscuous mode")
				tcpDumpContainer := createPromiscuousClient(workerNodeList[0].Definition.Name,
					tcpDumpNet1CMD)

				By("Define and create a server container")
				serverPod := createServerTestPod(serverNameDot1q, srIovNetworkDot1Q, nadCVLAN,
					workerNodeList[1].Definition.Name, []string{tsparams.ServerIPv4IPAddress, tsparams.ServerIPv6IPAddress})

				By("Define and create a client container")
				clientPod := createClientTestPod(clientNameDot1q, srIovNetworkDot1Q, nadCVLAN,
					workerNodeList[0].Definition.Name, []string{tsparams.ClientIPv4IPAddress, tsparams.ClientIPv6IPAddress})

				By("Validate IPv4 and IPv6 connectivity between the containers over the qinq tunnel.")
				err := cmd.ICMPConnectivityCheck(serverPod, []string{tsparams.ClientIPv4IPAddress,
					tsparams.ClientIPv6IPAddress}, "net2")
				Expect(err).ToNot(HaveOccurred(),
					"Failed to ping the client container over the 802.1q connection.")

				By("Validate IPv4 tcp traffic and dot1ad encapsulation from the client to server.")
				validateTCPTraffic(clientPod, serverIPV4IP.String())

				By("Validate IPv6 tcp traffic and dot1ad encapsulation from the client to server.")
				validateTCPTraffic(clientPod, serverIPV6IP.String())

				By("Validate that the TCP traffic is double tagged")
				readAndValidateTCPDump(tcpDumpContainer, tcpDumpReadFileCMD, tcpDumpDot1QOutput)
			})
	})

	AfterEach(func() {
		By("Removing all containers from test namespace")
		runningNamespace, err := namespace.Pull(APIClient, tsparams.TestNamespaceName)
		Expect(err).ToNot(HaveOccurred(), "Failed to pull namespace")

		Expect(runningNamespace.CleanObjects(
			tsparams.WaitTimeout, pod.GetGVR())).ToNot(HaveOccurred(), "Failed to the test namespace")
	})

	AfterAll(func() {
		By("Remove the double tag switch interface configurations")

		err = disableQinQOnSwitch(switchCredentials, switchInterfaces)
		Expect(err).ToNot(HaveOccurred(),
			"Failed to remove VLAN double tagging configuration from the switch")

		By(fmt.Sprintf("Disable VF promiscuous support on %s", srIovInterfacesUnderTest[0]))
		if sriovDeviceID == mlxDevice {
			promiscVFCommand = fmt.Sprintf("ip link set %s promisc off",
				srIovInterfacesUnderTest[0])
		} else {
			promiscVFCommand = fmt.Sprintf("ethtool --set-priv-flags %s vf-true-promisc-support off",
				srIovInterfacesUnderTest[0])
		}

		By(fmt.Sprintf("Disable VF promiscuous support on %s", srIovInterfacesUnderTest[0]))
		output, err := cmd.RunCommandOnHostNetworkPod(workerNodeList[0].Definition.Name, NetConfig.SriovOperatorNamespace,
			promiscVFCommand)
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to run command on node %s", output))

		By("Removing all SR-IOV Policy")
		err = sriov.CleanAllNetworkNodePolicies(APIClient, NetConfig.SriovOperatorNamespace, metav1.ListOptions{})
		Expect(err).ToNot(HaveOccurred(), "Failed to clean srIovPolicy")

		By("Removing all srIovNetworks")
		err = sriov.CleanAllNetworksByTargetNamespace(
			APIClient, NetConfig.SriovOperatorNamespace, tsparams.TestNamespaceName, metav1.ListOptions{})
		Expect(err).ToNot(HaveOccurred(), "Failed to clean sriov networks")

		By("Waiting until cluster MCP and SR-IOV are stable")
		err = netenv.WaitForSriovAndMCPStable(
			APIClient, tsparams.MCOWaitTimeout, time.Minute, NetConfig.CnfMcpLabel, NetConfig.SriovOperatorNamespace)
		Expect(err).ToNot(HaveOccurred(), "Failed cluster is not stable")
	})
})

func defineAndCreateSrIovNetworkWithQinQ(srIovNetwork, resName, vlanProtocol string, vlan uint16) {
	srIovNetworkObject, err := sriov.NewNetworkBuilder(
		APIClient, srIovNetwork, NetConfig.SriovOperatorNamespace, tsparams.TestNamespaceName, resName).
		WithVlanProto(vlanProtocol).WithVLAN(vlan).Create()
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to create sriov network %s", err))

	Eventually(func() bool {
		_, err := nad.Pull(APIClient, srIovNetworkObject.Object.Name, tsparams.TestNamespaceName)

		return err == nil
	}, tsparams.WaitTimeout, tsparams.RetryInterval).Should(BeTrue(), "Fail to pull NetworkAttachmentDefinition")
}

func createPromiscuousClient(nodeName string, tcpDumpCMD []string) *pod.Builder {
	sriovNetworkDefault := pod.StaticIPAnnotation("sriovnetwork-promiscuous", []string{"192.168.100.1/24"})

	clientDefault, err := pod.NewBuilder(APIClient, "client-promiscuous", tsparams.TestNamespaceName,
		NetConfig.CnfNetTestContainer).DefineOnNode(nodeName).WithPrivilegedFlag().RedefineDefaultCMD(tcpDumpCMD).
		WithSecondaryNetwork(sriovNetworkDefault).CreateAndWaitUntilRunning(netparam.DefaultTimeout)
	Expect(err).ToNot(HaveOccurred(), "Failed to define and run promiscuous pod")

	return clientDefault
}

func createServerTestPod(name, sVlan, cVlan, nodeName string, ipAddress []string) *pod.Builder {
	By(fmt.Sprintf("Define and run test pod  %s", name))

	annotation := defineNetworkAnnotation(sVlan, cVlan, ipAddress)

	serverCmd := []string{"bash", "-c", "sleep 5; testcmd -interface net2 -protocol tcp -port 4444 -listen"}
	serverBuild, err := pod.NewBuilder(APIClient, name, tsparams.TestNamespaceName,
		NetConfig.CnfNetTestContainer).DefineOnNode(nodeName).WithSecondaryNetwork(annotation).
		RedefineDefaultCMD(serverCmd).WithPrivilegedFlag().CreateAndWaitUntilRunning(netparam.DefaultTimeout)
	Expect(err).ToNot(HaveOccurred(), "Failed to define and run server container")

	return serverBuild
}

func createClientTestPod(name, sVlan, cVlan, nodeName string, ipAddress []string) *pod.Builder {
	By(fmt.Sprintf("Define and run test pod  %s", name))

	annotation := defineNetworkAnnotation(sVlan, cVlan, ipAddress)

	clientBuild, err := pod.NewBuilder(APIClient, name, tsparams.TestNamespaceName,
		NetConfig.CnfNetTestContainer).DefineOnNode(nodeName).WithSecondaryNetwork(annotation).
		WithPrivilegedFlag().CreateAndWaitUntilRunning(netparam.DefaultTimeout)
	Expect(err).ToNot(HaveOccurred(), "Failed to define and run default client")

	return clientBuild
}

func defineNetworkAnnotation(sVlan, cVlan string, ipAddress []string) []*multus.NetworkSelectionElement {
	annotation := []*multus.NetworkSelectionElement{}
	svlanAnnotation := pod.StaticAnnotation(sVlan)
	cvlanAnnotation := pod.StaticIPAnnotation(cVlan, ipAddress)
	annotation = append(annotation, svlanAnnotation, cvlanAnnotation[0])

	return annotation
}

func discoverInterfaceUnderTestDeviceID(srIovInterfaceUnderTest, workerNodeName string) string {
	sriovInterfaces, err := sriov.NewNetworkNodeStateBuilder(
		APIClient, workerNodeName, NetConfig.SriovOperatorNamespace).GetUpNICs()
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("fail to discover device ID for network interface %s",
		srIovInterfaceUnderTest))

	for _, srIovInterface := range sriovInterfaces {
		if srIovInterface.Name == srIovInterfaceUnderTest {
			return srIovInterface.DeviceID
		}
	}

	return ""
}

func validateTCPTraffic(clientPod *pod.Builder, destIPAddr string) {
	command := []string{
		"testcmd",
		fmt.Sprintf("--interface=%s", "net2"),
		fmt.Sprintf("--server=%s", destIPAddr),
		"--protocol=tcp",
		"--mtu=100",
		"--port=4444",
	}

	outPut, err := clientPod.ExecCommand(
		command)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Fail to run testcmd on %s command output: %s",
		clientPod.Definition.Name, outPut.String()))
}

// readAndValidateTCPDump checks that the inner C-VLAN is present verifying that the packet was double tagged.
func readAndValidateTCPDump(clientPod *pod.Builder, testCmd []string, pattern string) {
	By("Start to capture traffic on the promiscuous client")

	output, err := clientPod.ExecCommand(testCmd)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Error executing command: %s", output.String()))

	err = validateDot1Encapsulation(output.String(), pattern)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to validate qinq encapsulation %s", output.String()))
}

func validateDot1Encapsulation(fileOutput, dot1X string) error {
	// Compile the regular expression
	regex := regexp.MustCompile(dot1X)
	fmt.Println("REGEX", regex.String())
	match := regex.FindStringSubmatch(fileOutput)

	// Check if the regular expression matched at all
	if len(match) == 0 {
		return fmt.Errorf("regular expression did not match")
	}

	if len(match) != 3 {
		return fmt.Errorf("failed to match double encapsulation")
	}
	// Output the matches
	fmt.Println("Matched S-VLAN", match[1])
	fmt.Println("Matched C-VLAN", match[2])

	return nil
}

func enableDot1ADonSwitchInterfaces(credentials *sriovenv.SwitchCredentials, switchInterfaces []string) error {
	jnpr, err := cmd.NewSession(credentials.SwitchIP, credentials.User, credentials.Password)
	if err != nil {
		return err
	}
	defer jnpr.Close()

	for _, switchInterface := range switchInterfaces {
		commands := []string{fmt.Sprintf("set interfaces %s vlan-tagging encapsulation extended-vlan-bridge",
			switchInterface)}

		err = jnpr.Config(commands)
		if err != nil {
			return err
		}
	}

	return nil
}

func disableQinQOnSwitch(switchCredentials *sriovenv.SwitchCredentials, switchInterfaces []string) error {
	jnpr, err := cmd.NewSession(switchCredentials.SwitchIP, switchCredentials.User, switchCredentials.Password)
	if err != nil {
		return err
	}
	defer jnpr.Close()

	for _, switchInterface := range switchInterfaces {
		commands := []string{fmt.Sprintf("delete interfaces %s vlan-tagging", switchInterface)}
		commands = append(commands, fmt.Sprintf("delete interfaces %s encapsulation extended-vlan-bridge",
			switchInterface))
		err = jnpr.Config(commands)

		if err != nil {
			return err
		}
	}

	return nil
}
