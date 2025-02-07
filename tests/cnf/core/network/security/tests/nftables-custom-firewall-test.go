package tests

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/configmap"
	"github.com/openshift-kni/eco-goinfra/pkg/mco"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/cmd"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/define"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/frrconfig"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netenv"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netinittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/security/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/internal/cluster"
	ocpoperatorv1 "github.com/openshift/api/operator/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ignition "github.com/coreos/ignition/v2/config/v3_4/types"
	"gopkg.in/k8snetworkplumbingwg/multus-cni.v4/pkg/types"
	apimachinerytype "k8s.io/apimachinery/pkg/types"
)

var _ = Describe("nftables", Ordered, Label(tsparams.LabelNftablesTestCases), ContinueOnFailure, func() {

	var (
		hubIPv4ExternalAddresses = []string{"172.16.0.10", "172.16.0.11"}
		hubIPv4Network           = "172.16.0.0/24"
		masterPodIPv4Address     = "172.16.0.1"
		portNum8888              = 8888
		cnfWorkerNodeList        []*nodes.Builder
		masterNodeList           []*nodes.Builder
		ipv4NodeAddrList         []string
		ip4Worker0NodeAddr       []string
		ipv4SecurityIPList       []string
		testPodWorker0           *pod.Builder
		testPodList              []*pod.Builder
		routeMap                 map[string]string
		mcNftablesName           = "98-nftables-cnf-worker"
		interfaceNameNet1        = "net1"
		interfaceNameBrEx        = "br-ex"
		err                      error
	)
	BeforeAll(func() {
		By("List CNF worker nodes in cluster")
		cnfWorkerNodeList, err = nodes.List(APIClient,
			metav1.ListOptions{LabelSelector: labels.Set(NetConfig.WorkerLabelMap).String()})
		Expect(err).ToNot(HaveOccurred(), "Failed to discover worker nodes")

		By("Selecting worker node for Security tests")
		ipv4NodeAddrList, err = nodes.ListExternalIPv4Networks(
			APIClient, metav1.ListOptions{LabelSelector: labels.Set(NetConfig.WorkerLabelMap).String()})
		Expect(err).ToNot(HaveOccurred(), "Failed to collect external nodes ip addresses")

		ip4Worker0NodeAddr = []string{ipv4NodeAddrList[0]}

		By("Listing master nodes")
		masterNodeList, err = nodes.List(APIClient,
			metav1.ListOptions{LabelSelector: labels.Set(NetConfig.ControlPlaneLabelMap).String()})
		Expect(err).ToNot(HaveOccurred(), "Fail to list master nodes")
		Expect(len(cnfWorkerNodeList)).To(BeNumerically(">", 1),
			"Failed to detect at least two worker nodes")

		By("verify status of nftables and activate if inactive")
		activateNftablesIfInactive()

		By("Edit the machineconfiguration cluster to include NFTables")
		updateMachineConfigurationNodeDisruptionPolicy()

		By("Create test pods on worker node 0")
		testPodWorker0 = createTestPodOnWorkers(cnfWorkerNodeList[0].Definition.Name, portNum8888)
		testPodList = []*pod.Builder{testPodWorker0}

		By("Create a static route to the external Pod network on each worker node")
		// GetMetalLbVirIP using the metallb virtal IP address variable for test pod IP addresses.
		ipv4SecurityIPList, err = NetConfig.GetMetalLbVirIP()
		Expect(err).ToNot(HaveOccurred(), "Failed to retrieve the ipv4SecurityIPList")

		routeMap, err = netenv.BuildRoutesMapWithSpecificRoutes(testPodList, cnfWorkerNodeList, ipv4SecurityIPList)
		Expect(err).ToNot(HaveOccurred(), "Failed to create route map with specific routes")

		addDeleteStaticRouteOnWorkerNodes(testPodList, routeMap, "add", hubIPv4Network)
	})

	AfterAll(func() {
		By("Remove the static route to the external Pod network on each worker node")
		addDeleteStaticRouteOnWorkerNodes(testPodList, routeMap, "del", hubIPv4Network)

		By("Disables nftables in cluster")
		disableNftablesIfActive()
	})

	Context("custom firewall", func() {
		AfterEach(func() {
			By("Define and delete a NFTables custom rule")
			createMCAndWaitforMCPStable(tsparams.CustomFirewallDelete, mcNftablesName)
			err := mco.NewMCBuilder(APIClient, mcNftablesName).Delete()
			Expect(err).ToNot(HaveOccurred(), "Failed to get the machineConfig")
		})

		It("Verify the creation of a new custom node firewall NFTables table with an ingress rule",
			reportxml.ID("77412"), func() {
				By("Setup test environment")
				masterPod := setupRemoteMultiHopTest(ipv4SecurityIPList, hubIPv4ExternalAddresses,
					ipv4NodeAddrList, cnfWorkerNodeList, masterNodeList)

				By("Verify ICMP connectivity between the master Pod and the test pods on the workers")
				err := cmd.ICMPConnectivityCheck(masterPod, ip4Worker0NodeAddr, interfaceNameNet1)
				Expect(err).ToNot(HaveOccurred(), "Failed to ping the worker nodes")

				By("Verify ingress TCP traffic over port 8888 between the master Pod and the test pods on the workers")
				err = cmd.ValidateTCPTraffic(masterPod, ip4Worker0NodeAddr, interfaceNameNet1, frrconfig.ContainerName,
					portNum8888)
				Expect(err).ToNot(HaveOccurred(),
					"Failed to send ingress TCP traffic over port 8888 to the worker nodes")

				By("Verify egress TCP traffic over port 8888 between the test Pod on the worker0 and the master pod")
				err = cmd.ValidateTCPTraffic(testPodWorker0, []string{masterPodIPv4Address},
					interfaceNameBrEx, "", portNum8888)
				Expect(err).ToNot(HaveOccurred(),
					"Failed to send egress TCP traffic over port 8888 to the pod on the master node")

				By("Define and create a NFTables custom rule blocking ingress TCP port 8888")
				createMCAndWaitforMCPStable(tsparams.CustomFirewallIngressPort8888, mcNftablesName)

				By("Verify ingress TCP traffic is blocked over port 8888 between the master Pod and the test" +
					" pods on the workers")

				err = cmd.ValidateTCPTraffic(masterPod, ipv4NodeAddrList, interfaceNameNet1, frrconfig.ContainerName,
					portNum8888)
				Expect(err).To(HaveOccurred(),
					"Successfully sent ingress TCP traffic over port 8888 to the worker nodes")

				By("Verify egress TCP traffic over port 8888 between the test Pod on the worker0 and the master pod")
				err = cmd.ValidateTCPTraffic(testPodWorker0, []string{masterPodIPv4Address},
					interfaceNameBrEx, "", portNum8888)
				Expect(err).ToNot(HaveOccurred(),
					"Failed to send egress TCP traffic over port 8888 to the pod on the master node")
			})
	})
})

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

func setupRemoteMultiHopTest(
	ipv4SecurityIPList []string,
	hubIPv4ExternalAddresses []string,
	ipv4NodeAddrList []string,
	cnfWorkerNodeList []*nodes.Builder,
	masterNodeList []*nodes.Builder) *pod.Builder {
	var (
		nadHubName          = "nad-hub"
		masterPodIPv4Prefix = "172.16.0.1/24"
	)

	By("Creating External NAD for master FRR pod")

	err := define.CreateExternalNad(APIClient, frrconfig.ExternalMacVlanNADName, tsparams.TestNamespaceName)
	Expect(err).ToNot(HaveOccurred(), "Failed to create a network-attachment-definition")

	By("Creating External NAD for hub FRR pods")

	err = define.CreateExternalNad(APIClient, nadHubName, tsparams.TestNamespaceName)
	Expect(err).ToNot(HaveOccurred(), "Failed to create a network-attachment-definition")

	By("Creating static ip annotation for hub0")

	hub0BRstaticIPAnnotation := frrconfig.CreateStaticIPAnnotations(frrconfig.ExternalMacVlanNADName,
		nadHubName,
		[]string{fmt.Sprintf("%s/24", ipv4SecurityIPList[0])},
		[]string{fmt.Sprintf("%s/24", hubIPv4ExternalAddresses[0])})

	By("Creating static ip annotation for hub1")

	hub1BRstaticIPAnnotation := frrconfig.CreateStaticIPAnnotations(frrconfig.ExternalMacVlanNADName, nadHubName,
		[]string{fmt.Sprintf("%s/24", ipv4SecurityIPList[1])},
		[]string{fmt.Sprintf("%s/24", hubIPv4ExternalAddresses[1])})

	By("Creating Frr Hub pod configMap")

	hubConfigMap := createFrrConfigMap("hub-node-config", "")

	By("Creating FRR Hub pod on worker node 0")

	_ = createFrrPodTest("hub-pod-worker-0",
		cnfWorkerNodeList[0].Object.Name, hubConfigMap.Definition.Name, hub0BRstaticIPAnnotation, false)

	By("Creating FRR Hub pod on worker node 1")

	_ = createFrrPodTest("hub-pod-worker-1",
		cnfWorkerNodeList[1].Object.Name, hubConfigMap.Definition.Name, hub1BRstaticIPAnnotation, false)

	By("Creating configmap and Frr Master pod")

	frrConfigMapStaticRoutes := defineConfigMapWithStaticRouteAndNetwork(hubIPv4ExternalAddresses,
		cmd.RemovePrefixFromIPList(ipv4NodeAddrList))
	masterConfigMap := createFrrConfigMap("master-configmap", frrConfigMapStaticRoutes)

	masterStaticIPAnnotation := frrconfig.CreateStaticIPAnnotations(frrconfig.ExternalMacVlanNADName, nadHubName,
		[]string{masterPodIPv4Prefix}, []string{""})

	return createFrrPodTest("master-pod", masterNodeList[0].Definition.Name,
		masterConfigMap.Definition.Name, masterStaticIPAnnotation, true)
}

func createFrrConfigMap(name, configMapStaticRoutes string) *configmap.Builder {
	configMapData := frrconfig.DefineBaseConfig(frrconfig.DaemonsFile, configMapStaticRoutes, "")
	masterConfigMap, err := configmap.NewBuilder(APIClient, name, tsparams.TestNamespaceName).
		WithData(configMapData).Create()
	Expect(err).ToNot(HaveOccurred(), "Failed to create master config map")

	return masterConfigMap
}

func createFrrPodTest(
	name, nodeName,
	configmapName string,
	secondaryNetConfig []*types.NetworkSelectionElement,
	masterPod bool) *pod.Builder {
	var frrContainer *pod.ContainerBuilder

	By("Creating FRR master container in the test namespace")

	if masterPod {
		frrContainer = pod.NewContainerBuilder(
			"frr", NetConfig.CnfNetTestContainer, []string{"/bin/bash", "-c",
				"testcmd -interface net1 -protocol tcp -port 8888 -listen"})
	} else {
		frrContainer = pod.NewContainerBuilder(
			"frr", NetConfig.CnfNetTestContainer, []string{"/bin/bash", "-c", "sleep INF"})
	}

	frrContainer.WithSecurityCapabilities([]string{"NET_ADMIN", "NET_RAW", "SYS_ADMIN"}, true)
	frrCtr, err := frrContainer.GetContainerCfg()
	Expect(err).ToNot(HaveOccurred(), "Failed to get container configuration")

	frrPod, err := pod.NewBuilder(APIClient, name, tsparams.TestNamespaceName, NetConfig.FrrImage).
		DefineOnNode(nodeName).
		WithTolerationToMaster().
		WithSecondaryNetwork(secondaryNetConfig).
		RedefineDefaultCMD([]string{}).
		WithAdditionalContainer(frrCtr).
		WithLocalVolume(configmapName, "/etc/frr").
		WithPrivilegedFlag().
		CreateAndWaitUntilRunning(5 * time.Minute)
	Expect(err).ToNot(HaveOccurred(), "Failed to create FRR test pod")

	return frrPod
}

func defineConfigMapWithStaticRouteAndNetwork(hubPodIPs, nodeIPAddresses []string) string {
	frrConfig :=
		fmt.Sprintf("ip route %s/32 %s\n", nodeIPAddresses[1], hubPodIPs[0]) +
			fmt.Sprintf("ip route %s/32 %s\n!\n", nodeIPAddresses[0], hubPodIPs[1])

	frrConfig += "!\nline vty\n!\nend\n"

	return frrConfig
}

func createTestPodOnWorkers(nodeName string, portNum int) *pod.Builder {
	testPod, err := pod.NewBuilder(
		APIClient, "testpod-"+nodeName, tsparams.TestNamespaceName, NetConfig.CnfNetTestContainer).
		DefineOnNode(nodeName).WithHostNetwork().WithHostPid(true).
		RedefineDefaultCMD([]string{"/bin/bash", "-c",
			fmt.Sprintf("testcmd -interface br-ex -protocol tcp -port %d -listen", portNum)}).
		WithPrivilegedFlag().CreateAndWaitUntilRunning(180 * time.Second)
	Expect(err).ToNot(HaveOccurred(), "Failed to create test pod")

	return testPod
}

func createMCAndWaitforMCPStable(fileContentString, mcNftablesName string) {
	// Convert the string to base64.
	encoded := base64.StdEncoding.EncodeToString([]byte(fileContentString))
	encodedWithPrefix := "data:;base64," + encoded
	truePointer := true

	mode := 384
	sysDContents := `
            [Unit]  
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
					Contents: &sysDContents,
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
							Source: &encodedWithPrefix,
						},
						Mode: &mode,
					},
				},
			},
		},
	}
	finalIgnitionConfig, err := json.Marshal(ignitionConfig)
	Expect(err).ToNot(HaveOccurred(), "Failed to serialize ignition config")
	_, err = mco.NewMCBuilder(APIClient, mcNftablesName).
		WithLabel("machineconfiguration.openshift.io/role", NetConfig.CnfMcpLabel).
		WithRawConfig(finalIgnitionConfig).
		Create()
	Expect(err).ToNot(HaveOccurred(), "Failed to create nftables machine config")

	err = netenv.WaitForMcpStable(APIClient, 35*time.Minute, 30*time.Second, NetConfig.CnfMcpLabel)
	Expect(err).ToNot(HaveOccurred(), "Failed to wait for MCP to be stable")
}

func addDeleteStaticRouteOnWorkerNodes(testPodList []*pod.Builder, routeMap map[string]string, routeAction,
	hubIPv4Network string) {
	for _, testPod := range testPodList {
		_, err := netenv.SetStaticRoute(testPod, routeAction, hubIPv4Network, "", routeMap)
		Expect(err).ToNot(HaveOccurred(), "Failed to create or delete static route")
	}
}

func activateNftablesIfInactive() {
	// Get the current status of the nftables service
	statuses, err := cluster.ExecCmdWithStdoutWithRetries(APIClient,
		3, 10*time.Second,
		"systemctl is-active nftables.service | cat -")
	Expect(err).ToNot(HaveOccurred(), "Failed to check if nftables service status")
	Expect(statuses).ToNot(BeEmpty(), "Failed to find statuses for nftables service")

	// Iterate through the statuses of the nodes
	for nodeName, status := range statuses {
		// If the status is inactive, activate it
		status = strings.TrimSpace(status)
		if status == "inactive" {
			// Execute the command to start nftables service
			_, err := cluster.ExecCmdWithStdoutWithRetries(APIClient,
				3, 10*time.Second,
				"systemctl start nftables.service")
			Expect(err).ToNot(HaveOccurred(), "Failed to start nftables service on "+nodeName)

			// Verify that nftables is now active
			verifyNftablesStatus("active", nodeName)
		}
	}
}

func disableNftablesIfActive() {
	// Get the current status of the nftables service
	statuses, err := cluster.ExecCmdWithStdoutWithRetries(APIClient,
		3, 10*time.Second,
		"systemctl is-active nftables.service | cat -")
	Expect(err).ToNot(HaveOccurred(), "Failed to check nftables service status")
	Expect(statuses).ToNot(BeEmpty(), "Failed to find statuses for nftables service")

	// Iterate through the statuses of the nodes
	for nodeName, status := range statuses {
		status = strings.TrimSpace(status)
		if status == "active" {
			// Execute the command to stop and disable nftables service
			_, err := cluster.ExecCmdWithStdoutWithRetries(APIClient,
				3, 10*time.Second,
				"systemctl stop nftables.service")
			Expect(err).ToNot(HaveOccurred(), "Failed to disable nftables service on "+nodeName)
			fmt.Printf("Disabled nftables on %s\n", nodeName)

			// Verify that nftables is now inactive
			verifyNftablesStatus("inactive", nodeName)
		}
	}
}

func verifyNftablesStatus(expectedStatus, nodeName string) {
	statuses, err := cluster.ExecCmdWithStdoutWithRetries(APIClient,
		3, 10*time.Second,
		"systemctl is-active nftables.service | cat -")
	Expect(err).ToNot(HaveOccurred(), "Failed to verify nftables status on "+nodeName)

	actualStatus, exists := statuses[nodeName]
	Expect(exists).To(BeTrue(), "Node "+nodeName+" not found in nftables status results")

	actualStatus = strings.TrimSpace(actualStatus)
	Expect(actualStatus).To(Equal(expectedStatus),
		fmt.Sprintf("Expected nftables status on %s to be %s, but got %s", nodeName, expectedStatus, actualStatus))
}
