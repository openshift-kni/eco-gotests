package tests

import (
	"encoding/json"
	"fmt"
	"time"

	ignition "github.com/coreos/ignition/v2/config/v3_1/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/mco"
	"github.com/openshift-kni/eco-goinfra/pkg/nad"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"github.com/openshift-kni/eco-goinfra/pkg/sriov"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netenv"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netinittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netparam"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/sriov/internal/sriovenv"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/sriov/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/internal/cluster"
	"gopkg.in/k8snetworkplumbingwg/multus-cni.v4/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

var _ = Describe("rdmaMetrics", Ordered, Label(tsparams.LabelRdmaMetricsTestCases),
	ContinueOnFailure, func() {
		var (
			workerNodeList           []*nodes.Builder
			sriovInterfacesUnderTest []string
			rdmaMachineConfig        *mco.MCBuilder
		)

		BeforeAll(func() {
			By("Verifying if Rdma Metrics tests can be executed on given cluster")
			err := netenv.DoesClusterHasEnoughNodes(APIClient, NetConfig, 1, 1)
			Expect(err).ToNot(HaveOccurred(),
				"Cluster doesn't support Rdma Metrics test cases as it doesn't have enough nodes")

			By("Validating SR-IOV interfaces")
			workerNodeList, err = nodes.List(APIClient,
				metav1.ListOptions{LabelSelector: labels.Set(NetConfig.WorkerLabelMap).String()})
			Expect(err).ToNot(HaveOccurred(), "Failed to discover worker nodes")

			Expect(sriovenv.ValidateSriovInterfaces(workerNodeList, 2)).ToNot(HaveOccurred(),
				"Failed to get required SR-IOV interfaces")

			sriovInterfacesUnderTest, err = NetConfig.GetSriovInterfaces(2)
			Expect(err).ToNot(HaveOccurred(), "Failed to retrieve SR-IOV interfaces for testing")

			By("Fetching SR-IOV Vendor ID for interface under test")
			sriovVendor := discoverInterfaceUnderTestVendorID(sriovInterfacesUnderTest[0], workerNodeList[0].Definition.Name)
			Expect(sriovVendor).ToNot(BeEmpty(), "Expected Sriov Vendor not to be empty")

			By("Skipping test cases if the Sriov device is not of Mellanox")
			if sriovVendor != netparam.MlxVendorID {
				Skip("Rdma metrics is supported only on Mellanox devices")
			}

		})

		Context("Rdma metrics in exclusive mode", func() {
			BeforeAll(func() {
				By("Enable RDMA exclusive mode")

				rdmaMachineConfig = enableRdmaMode()

				err := netenv.WaitForMcpStable(APIClient, tsparams.MCOWaitTimeout, 1*time.Minute, NetConfig.CnfMcpLabel)
				Expect(err).ToNot(HaveOccurred(), "Failed to wait for MCP update")

				By("Ensure RDMA is in exclusive mode")

				outputNodes, err := cluster.ExecCmdWithStdout(
					APIClient, "rdma system show", metav1.ListOptions{LabelSelector: labels.Set(NetConfig.WorkerLabelMap).String()})
				Expect(err).ToNot(HaveOccurred(), "Failed to get output of rdma system show")

				for node, output := range outputNodes {
					Expect(output).To(ContainSubstring("exclusive"), fmt.Sprintf("RDMA is not set to shared on node %s", node))
				}

			})
			It("1 Pod with 1 VF", reportxml.ID("75230"), func() {

				By("Define and Create Test Resources")
				tPol := defineAndCreateNodePolicy("rdmapolicy1", "sriovpf1", sriovInterfacesUnderTest[0], 1, 0)
				tNet := defineAndCreateSriovNetworkWithRdma("sriovnet1", tPol.Object.Spec.ResourceName, true)
				testPod := defineAndCreatePod(tNet.Object.Name, "")

				By("Verify Rdma metrics are available inside Pod but not on Host")
				verifyRdmaMetrics(testPod, "net1")
			})
			It("1 Pod with 2 VF of same PF", reportxml.ID("75231"), func() {
				By("Define and Create Test Resources")
				tPol := defineAndCreateNodePolicy("rdmapolicy1", "sriovpf1", sriovInterfacesUnderTest[0], 2, 1)
				tNet := defineAndCreateSriovNetworkWithRdma("sriovnet1", tPol.Object.Spec.ResourceName, true)
				testPod := defineAndCreatePod(tNet.Object.Name, tNet.Object.Name)

				By("Verify Rdma metrics are available inside Pod but not on Host")
				verifyRdmaMetrics(testPod, "net1")
				verifyRdmaMetrics(testPod, "net2")
			})
			It("1 Pod with 2 VF of different PF", reportxml.ID("75232"), func() {
				By("Define and Create Test Resources")
				tPol1 := defineAndCreateNodePolicy("rdmapolicy1", "sriovpf1", sriovInterfacesUnderTest[0], 1, 0)
				tPol2 := defineAndCreateNodePolicy("rdmapolicy2", "sriovpf2", sriovInterfacesUnderTest[1], 1, 0)
				tNet1 := defineAndCreateSriovNetworkWithRdma("sriovnet1", tPol1.Object.Spec.ResourceName, true)
				tNet2 := defineAndCreateSriovNetworkWithRdma("sriovnet2", tPol2.Object.Spec.ResourceName, true)
				testPod := defineAndCreatePod(tNet1.Object.Name, tNet2.Object.Name)

				By("Verify Rdma metrics are available inside Pod but not on Host")
				verifyRdmaMetrics(testPod, "net1")
				verifyRdmaMetrics(testPod, "net2")
			})

			AfterEach(func() {
				By("Removing SR-IOV configuration")
				err := sriovenv.RemoveSriovConfigurationAndWaitForSriovAndMCPStable()
				Expect(err).ToNot(HaveOccurred(), "Failed to remove SR-IOV configration")

				By("Cleaning test namespace")
				err = namespace.NewBuilder(APIClient, tsparams.TestNamespaceName).CleanObjects(
					netparam.DefaultTimeout, pod.GetGVR())
				Expect(err).ToNot(HaveOccurred(), "Failed to clean test namespace")
			})

			AfterAll(func() {
				By("Disable rdma exclusive mode")

				err := rdmaMachineConfig.Delete()
				Expect(err).ToNot(HaveOccurred(), "Failed to delete rdma exclusive mode machine config")

				err = netenv.WaitForMcpStable(
					APIClient, tsparams.MCOWaitTimeout, tsparams.DefaultStableDuration, NetConfig.CnfMcpLabel)
				Expect(err).ToNot(HaveOccurred(), "Failed to wait for MCP update")

			})
		})

		Context("Rdma metrics in shared mode", func() {
			BeforeAll(func() {
				By("Ensure RDMA is in shared mode")

				outputNodes, err := cluster.ExecCmdWithStdout(
					APIClient, "rdma system show", metav1.ListOptions{LabelSelector: labels.Set(NetConfig.WorkerLabelMap).String()})
				Expect(err).ToNot(HaveOccurred(), "Failed to get output of rdma system show")

				for node, output := range outputNodes {
					Expect(output).To(ContainSubstring("shared"), fmt.Sprintf("RDMA is not set to shared on node %s", node))
				}

			})
			It("1 Pod with 1 VF", reportxml.ID("75353"), func() {
				By("Define and Create Test Resources")

				tPol := defineAndCreateNodePolicy("rdmapolicy1", "sriovpf1", sriovInterfacesUnderTest[0], 1, 0)
				tNet := defineAndCreateSriovNetworkWithRdma("sriovnet1", tPol.Object.Spec.ResourceName, false)
				testPod := defineAndCreatePod(tNet.Object.Name, "")

				By("Fetch PCI Address and Rdma device from Pod Annotations")
				pciAddress, rdmaDevice, err := getInterfacePci(testPod, "net1")
				Expect(err).ToNot(HaveOccurred(),
					"Could not get PCI Address and/or Rdma device from Pod Annotations")

				By("Verify Rdma Metrics should not be present inside Pod")
				podOutput, err := testPod.ExecCommand(
					[]string{"bash", "-c", fmt.Sprintf("ls /sys/bus/pci/devices/%s/infiniband/%s/ports/1/hw_counters",
						pciAddress, rdmaDevice)})
				Expect(err).To(HaveOccurred(), "Expected command to fail as the path is not present on Pod")
				Expect(podOutput.String()).To(ContainSubstring("No such file or directory"),
					fmt.Sprintf("Expected error 'No such file or directory' in the output %s", podOutput.String()))

				By("Verify Rdma Metrics should be present on Host")
				nodeOutput, err := cluster.ExecCmdWithStdout(APIClient,
					fmt.Sprintf("ls /sys/bus/pci/devices/%s/infiniband/%s/ports/1/hw_counters", pciAddress, rdmaDevice),
					metav1.ListOptions{LabelSelector: fmt.Sprintf("kubernetes.io/hostname=%s", testPod.Object.Spec.NodeName)})
				Expect(err).ToNot(HaveOccurred(),
					fmt.Sprintf("Failed to run command %s on node %s",
						fmt.Sprintf("ls /sys/bus/pci/devices/%s/infiniband/%s/ports/1/hw_counters",
							pciAddress, rdmaDevice), testPod.Object.Spec.NodeName))
				Expect(nodeOutput[testPod.Object.Spec.NodeName]).To(ContainSubstring("out_of_buffer"),
					fmt.Sprintf("Failed to find the counters in the output %s", nodeOutput[testPod.Object.Spec.NodeName]))
			})

			AfterAll(func() {
				By("Removing SR-IOV configuration")
				err := sriovenv.RemoveSriovConfigurationAndWaitForSriovAndMCPStable()
				Expect(err).ToNot(HaveOccurred(), "Failed to remove SR-IOV configration")

				By("Cleaning test namespace")
				err = namespace.NewBuilder(APIClient, tsparams.TestNamespaceName).CleanObjects(
					netparam.DefaultTimeout, pod.GetGVR())
				Expect(err).ToNot(HaveOccurred(), "Failed to clean test namespace")
			})
		})
	})

func enableRdmaMode() *mco.MCBuilder {
	RdmaMachineConfigContents := `
		[Unit]
		Description=RDMA exclusive mode
		Before=kubelet.service crio.service node-valid-hostname.service
		
		[Service]
		# Need oneshot to delay kubelet
		Type=oneshot
		ExecStart=/usr/bin/bash -c "rdma system set netns exclusive"
		StandardOutput=journal+console
		StandardError=journal+console
		
		[Install]
		WantedBy=network-online.target`
	truePointer := true
	ignitionConfig := ignition.Config{
		Ignition: ignition.Ignition{
			Version: "3.1.0",
		},
		Systemd: ignition.Systemd{
			Units: []ignition.Unit{
				{
					Enabled:  &truePointer,
					Name:     "rdma.service",
					Contents: &RdmaMachineConfigContents,
				},
			},
		},
	}

	finalIgnitionConfig, err := json.Marshal(ignitionConfig)
	Expect(err).ToNot(HaveOccurred(), "Failed to serialize ignition config")

	createdMC, err := mco.NewMCBuilder(APIClient, "02-rdma-netns-exclusive-mode").
		WithLabel("machineconfiguration.openshift.io/role", NetConfig.CnfMcpLabel).
		WithRawConfig(finalIgnitionConfig).
		Create()
	Expect(err).ToNot(HaveOccurred(), "Failed to create RDMA machine config")

	return createdMC
}

func getInterfacePci(tPod *pod.Builder, podInterface string) (string, string, error) {
	type PodNetworkStatusAnnotation struct {
		Name       string   `json:"name"`
		Interface  string   `json:"interface"`
		Ips        []string `json:"ips,omitempty"`
		Mac        string   `json:"mac,omitempty"`
		Default    bool     `json:"default,omitempty"`
		Mtu        int      `json:"mtu,omitempty"`
		DeviceInfo struct {
			Type    string `json:"type"`
			Version string `json:"version"`
			Pci     struct {
				PciAddress string `json:"pci-address"`
				RdmaDevice string `json:"rdma-device"`
			} `json:"pci"`
		} `json:"device-info,omitempty"`
	}

	var annotation []PodNetworkStatusAnnotation

	err := json.Unmarshal([]byte(tPod.Object.Annotations["k8s.v1.cni.cncf.io/network-status"]), &annotation)
	Expect(err).ToNot(HaveOccurred(), "Failed to Unmarshal Pod Network Status annotation")

	for _, annotationObject := range annotation {
		if annotationObject.Interface == podInterface {
			if annotationObject.DeviceInfo.Pci.PciAddress != "" || annotationObject.DeviceInfo.Pci.RdmaDevice != "" {
				return annotationObject.DeviceInfo.Pci.PciAddress, annotationObject.DeviceInfo.Pci.RdmaDevice, nil
			}

			return "", "", fmt.Errorf("PCI Address or Rdma Device are not present in the Interface %v", annotationObject)
		}
	}

	return "", "", fmt.Errorf("interface %s not found", podInterface)
}

func defineAndCreateNodePolicy(polName, resName, sriovInterface string, numVfs, endVf int) *sriov.PolicyBuilder {
	testPolicy := sriov.NewPolicyBuilder(APIClient,
		polName, NetConfig.SriovOperatorNamespace, resName, numVfs, []string{sriovInterface}, NetConfig.WorkerLabelMap).
		WithDevType("netdevice").
		WithVFRange(0, endVf).
		WithRDMA(true)
	err := sriovenv.CreateSriovPolicyAndWaitUntilItsApplied(testPolicy, tsparams.MCOWaitTimeout)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to create SriovNodePolicy %s", polName))

	return testPolicy
}

func defineAndCreateSriovNetworkWithRdma(netName, resName string, withRdma bool) *sriov.NetworkBuilder {
	testNetBuilder := sriov.NewNetworkBuilder(
		APIClient, netName, NetConfig.SriovOperatorNamespace, tsparams.TestNamespaceName, resName).
		WithMacAddressSupport().
		WithIPAddressSupport().
		WithStaticIpam().
		WithLogLevel(netparam.LogLevelDebug)

	if withRdma {
		testNetBuilder.WithMetaPluginRdma()
	}

	testNetwork, err := testNetBuilder.Create()
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to create Sriov Network %s", netName))

	return testNetwork
}

func defineAndCreatePod(netName1, netName2 string) *pod.Builder {
	netAnnotations := []*types.NetworkSelectionElement{
		{
			Name:       netName1,
			MacRequest: tsparams.ServerMacAddress,
			IPRequest:  []string{tsparams.ServerIPv4IPAddress},
		},
	}

	if len(netName2) != 0 {
		netAnnotations = append(netAnnotations, &types.NetworkSelectionElement{
			Name:       netName2,
			MacRequest: tsparams.ClientMacAddress,
			IPRequest:  []string{tsparams.ClientIPv4IPAddress},
		})
	}

	for _, net := range netAnnotations {
		Eventually(func() error {
			_, err := nad.Pull(APIClient, net.Name, tsparams.TestNamespaceName)

			return err
		}, 10*time.Second, 1*time.Second).Should(BeNil(), fmt.Sprintf("Failed to pull NAD %s", net.Name))
	}

	tPod, err := pod.NewBuilder(APIClient, "testpod", tsparams.TestNamespaceName, NetConfig.CnfNetTestContainer).
		WithSecondaryNetwork(netAnnotations).
		WithPrivilegedFlag().
		CreateAndWaitUntilRunning(2 * time.Minute)
	Expect(err).ToNot(HaveOccurred(), "Failed to create Pod %s")

	tPod.Exists()

	return tPod
}

func verifyRdmaMetrics(inputPod *pod.Builder, iName string) {
	By("Fetch PCI Address and Rdma device from Pod Annotations")

	pciAddress, rdmaDevice, err := getInterfacePci(inputPod, iName)
	Expect(err).ToNot(HaveOccurred(),
		"Could not get PCI Address and/or Rdma device from Pod Annotations")

	By("Rdma metrics should be present inside the Pod")

	podOutput, err := inputPod.ExecCommand(
		[]string{"bash", "-c", fmt.Sprintf("ls /sys/bus/pci/devices/%s/infiniband/%s/ports/1/hw_counters",
			pciAddress, rdmaDevice)})
	Expect(err).ToNot(HaveOccurred(), "Failed to check counters directory on Pod")
	Expect(podOutput.String()).To(ContainSubstring("out_of_buffer"),
		fmt.Sprintf("Failed to find the counters in the output %s", podOutput.String()))

	By("Rdma metrics should not be present in the Host")

	_, err = cluster.ExecCmdWithStdout(APIClient,
		fmt.Sprintf("ls /sys/bus/pci/devices/%s/infiniband/%s/ports/1/hw_counters", pciAddress, rdmaDevice),
		metav1.ListOptions{LabelSelector: fmt.Sprintf("kubernetes.io/hostname=%s", inputPod.Object.Spec.NodeName)})
	Expect(err).To(HaveOccurred(), "Failed to check counters directory on Host")
	Expect(err.Error()).To(ContainSubstring("command terminated with exit code 2"), "Failure is not as expected")
}

func discoverInterfaceUnderTestVendorID(srIovInterfaceUnderTest, workerNodeName string) string {
	sriovInterfaces, err := sriov.NewNetworkNodeStateBuilder(
		APIClient, workerNodeName, NetConfig.SriovOperatorNamespace).GetUpNICs()
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("fail to discover Vendor ID for network interface %s",
		srIovInterfaceUnderTest))

	for _, srIovInterface := range sriovInterfaces {
		if srIovInterface.Name == srIovInterfaceUnderTest {
			return srIovInterface.Vendor
		}
	}

	return ""
}
