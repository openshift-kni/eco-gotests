package sriovenv

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/daemonset"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/namespace"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/nodes"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/pod"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/sriov"

	sriovV1 "github.com/k8snetworkplumbingwg/sriov-network-operator/api/v1"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/core/network/internal/cmd"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/core/network/internal/netenv"
	. "github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/core/network/internal/netinittools"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/core/network/internal/netparam"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/core/network/sriov/internal/tsparams"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

// ValidateSriovInterfaces checks that provided interfaces by env var exist on the nodes.
func ValidateSriovInterfaces(workerNodeList []*nodes.Builder, requestedNumber int) error {
	var validSriovIntefaceList []sriovV1.InterfaceExt

	availableUpSriovInterfaces, err := sriov.NewNetworkNodeStateBuilder(APIClient,
		workerNodeList[0].Definition.Name, NetConfig.SriovOperatorNamespace).GetUpNICs()

	if err != nil {
		return fmt.Errorf("failed get SR-IOV devices from the node %s", workerNodeList[0].Definition.Name)
	}

	requestedSriovInterfaceList, err := NetConfig.GetSriovInterfaces(requestedNumber)
	if err != nil {
		return err
	}

	for _, availableUpSriovInterface := range availableUpSriovInterfaces {
		for _, requestedSriovInterface := range requestedSriovInterfaceList {
			if availableUpSriovInterface.Name == requestedSriovInterface {
				validSriovIntefaceList = append(validSriovIntefaceList, availableUpSriovInterface)
			}
		}
	}

	if len(validSriovIntefaceList) < requestedNumber {
		return fmt.Errorf("requested interfaces %v are not present on the cluster node", requestedSriovInterfaceList)
	}

	return nil
}

// CreateSriovPolicyAndWaitUntilItsApplied creates SriovNetworkNodePolicy and waits until
// it's successfully applied.
func CreateSriovPolicyAndWaitUntilItsApplied(sriovPolicy *sriov.PolicyBuilder, timeout time.Duration) error {
	glog.V(90).Infof("Creating SriovNetworkNodePolicy %s and waiting until it's successfully applied.",
		sriovPolicy.Definition.Name)

	_, err := sriovPolicy.Create()
	if err != nil {
		return err
	}

	err = netenv.WaitForSriovAndMCPStable(
		APIClient, timeout, tsparams.DefaultStableDuration, NetConfig.CnfMcpLabel, NetConfig.SriovOperatorNamespace)
	if err != nil {
		return err
	}

	return nil
}

// WaitUntilVfsCreated waits until all expected SR-IOV VFs are created.
func WaitUntilVfsCreated(
	nodeList []*nodes.Builder, sriovInterfaceName string, numberOfVfs int, timeout time.Duration) error {
	glog.V(90).Infof("Waiting for the creation of all VFs (%d) under"+
		" the %s interface in the SriovNetworkState.", numberOfVfs, sriovInterfaceName)

	for _, node := range nodeList {
		err := wait.PollUntilContextTimeout(
			context.TODO(), time.Second, timeout, true, func(ctx context.Context) (bool, error) {
				sriovNetworkState := sriov.NewNetworkNodeStateBuilder(APIClient, node.Object.Name, NetConfig.SriovOperatorNamespace)

				err := sriovNetworkState.Discover()
				if err != nil {
					return false, nil
				}

				err = isVfCreated(sriovNetworkState, numberOfVfs, sriovInterfaceName)
				if err != nil {
					return false, nil
				}

				return true, nil
			})
		if err != nil {
			return err
		}
	}

	return nil
}

// IsSriovDeployed checks SR-IOV deployment in the cluster.
// Returns nil if SR-IOV is deployed & daemonsets are ready, else returns an error.
func IsSriovDeployed() error {
	glog.V(90).Infof("Validating all SR-IOV operator resources are ready")

	sriovNS := namespace.NewBuilder(APIClient, NetConfig.SriovOperatorNamespace)
	if !sriovNS.Exists() {
		glog.V(90).Infof("SR-IOV operator namespace doesn't exist")

		return fmt.Errorf("error SR-IOV namespace %s doesn't exist", sriovNS.Definition.Name)
	}

	for _, sriovDaemonsetName := range tsparams.OperatorSriovDaemonsets {
		glog.V(90).Infof("Validating daemonset %s exists and ready", sriovDaemonsetName)
		sriovDaemonset, err := daemonset.Pull(
			APIClient, sriovDaemonsetName, NetConfig.SriovOperatorNamespace)

		if err != nil {
			glog.V(90).Infof("Pulling daemonset %s failed", sriovDaemonsetName)

			return fmt.Errorf("error to pull SR-IOV daemonset %s from cluster: %s", sriovDaemonsetName, err.Error())
		}

		if !sriovDaemonset.IsReady(3 * time.Minute) {
			glog.V(90).Infof("Daemonset %s is not ready", sriovDaemonsetName)

			return fmt.Errorf("error SR-IOV deployment %s is not in ready/ready state",
				sriovDaemonsetName)
		}
	}

	return nil
}

// RemoveAllPoliciesAndWaitForSriovAndMCPStable removes all  SriovNetworkNodePolicies and waits until
// SR-IOV and MCP become stable.
func RemoveAllPoliciesAndWaitForSriovAndMCPStable() error {
	glog.V(90).Infof("Deleting all SriovNetworkNodePolicies and waiting for SR-IOV and MCP become stable.")

	err := sriov.CleanAllNetworkNodePolicies(APIClient, NetConfig.SriovOperatorNamespace, metav1.ListOptions{})
	if err != nil {
		return err
	}

	return netenv.WaitForSriovStable(APIClient, tsparams.MCOWaitTimeout, NetConfig.SriovOperatorNamespace)
}

// IsMellanoxDevice checks if a given network interface on a node is a Mellanox device.
func IsMellanoxDevice(intName, nodeName string) bool {
	glog.V(90).Infof("Checking if specific interface %s on node %s is a Mellanox device.", intName, nodeName)
	sriovNetworkState := sriov.NewNetworkNodeStateBuilder(APIClient, nodeName, NetConfig.SriovOperatorNamespace)
	driverName, err := sriovNetworkState.GetDriverName(intName)

	if err != nil {
		glog.V(90).Infof("Failed to get driver name for interface %s on node %s: %w", intName, nodeName, err)

		return false
	}

	return driverName == "mlx5_core"
}

// ConfigureSriovMlnxFirmwareOnWorkers configures SR-IOV firmware on a given Mellanox device.
func ConfigureSriovMlnxFirmwareOnWorkers(
	workerNodes []*nodes.Builder, sriovInterfaceName string, enableSriov bool, numVfs int) error {
	for _, workerNode := range workerNodes {
		glog.V(90).Infof("Configuring SR-IOV firmware on the Mellanox device %s on the workers"+
			" %v with parameters: enableSriov %t and numVfs %d", sriovInterfaceName, workerNodes, enableSriov, numVfs)

		sriovNetworkState := sriov.NewNetworkNodeStateBuilder(
			APIClient, workerNode.Object.Name, NetConfig.SriovOperatorNamespace)
		pciAddress, err := sriovNetworkState.GetPciAddress(sriovInterfaceName)

		if err != nil {
			glog.V(90).Infof("Failed to get PCI address for the interface %s", sriovInterfaceName)

			return fmt.Errorf("failed to get PCI address: %s", err.Error())
		}

		output, err := runCommandOnConfigDaemon(workerNode.Object.Name,
			[]string{"bash", "-c",
				fmt.Sprintf("mstconfig -y -d %s set SRIOV_EN=%t NUM_OF_VFS=%d && chroot /host reboot",
					pciAddress, enableSriov, numVfs)})

		if err != nil {
			glog.V(90).Infof("Failed to configure SR-IOV firmware.")

			return fmt.Errorf("failed to configure Mellanox firmware for interface %s on a node %s: %s\n %s",
				pciAddress, workerNode.Object.Name, output, err.Error())
		}
	}

	return nil
}

// CreatePodsAndRunTraffic creates test pods and verifies connectivity between them.
func CreatePodsAndRunTraffic(
	clientNodeName string,
	serverNodeName string,
	sriovResNameClient string,
	sriovResNameServer string,
	clientMac string,
	serverMac string,
	clientIPs []string,
	serverIPs []string) error {
	glog.V(90).Infof("Creating test pods and checking ICMP connectivity between them")

	clientPod, _, err := createAndWaitTestPods(
		clientNodeName,
		serverNodeName,
		sriovResNameClient,
		sriovResNameServer,
		clientMac,
		serverMac,
		clientIPs,
		serverIPs)

	if err != nil {
		glog.V(90).Infof("Failed to create test pods")

		return err
	}

	return cmd.ICMPConnectivityCheck(clientPod, serverIPs)
}

// RemoveSriovConfigurationAndWaitForSriovAndMCPStable removes all SR-IOV networks
// and policies in SR-IOV operator namespace.
func RemoveSriovConfigurationAndWaitForSriovAndMCPStable() error {
	glog.V(90).Infof("Removing all SR-IOV networks and policies")

	err := RemoveAllSriovNetworks()
	if err != nil {
		glog.V(90).Infof("Failed to remove all SR-IOV networks")

		return err
	}

	err = removeAllPoliciesAndWaitForSriovAndMCPStable()
	if err != nil {
		glog.V(90).Infof("Failed to remove all SR-IOV policies")

		return err
	}

	return nil
}

// RemoveAllSriovNetworks removes all SR-IOV networks.
func RemoveAllSriovNetworks() error {
	glog.V(90).Infof("Removing all SR-IOV networks")

	sriovNs, err := namespace.Pull(APIClient, NetConfig.SriovOperatorNamespace)
	if err != nil {
		glog.V(90).Infof("Failed to pull SR-IOV operator namespace")

		return err
	}

	err = sriovNs.CleanObjects(
		netparam.DefaultTimeout,
		sriov.GetSriovNetworksGVR())
	if err != nil {
		glog.V(90).Infof("Failed to remove SR-IOV networks from SR-IOV operator namespace")

		return err
	}

	return nil
}

// removeAllPoliciesAndWaitForSriovAndMCPStable removes all  SriovNetworkNodePolicies and waits until
// SR-IOV and MCP become stable.
func removeAllPoliciesAndWaitForSriovAndMCPStable() error {
	glog.V(90).Infof("Deleting all SriovNetworkNodePolicies and waiting for SR-IOV and MCP become stable.")

	err := sriov.CleanAllNetworkNodePolicies(APIClient, NetConfig.SriovOperatorNamespace)
	if err != nil {
		return err
	}

	return netenv.WaitForSriovAndMCPStable(
		APIClient, tsparams.MCOWaitTimeout, time.Minute, NetConfig.CnfMcpLabel, NetConfig.SriovOperatorNamespace)
}

// createAndWaitTestPods creates test pods and waits until they are in the ready state.
func createAndWaitTestPods(
	clientNodeName string,
	serverNodeName string,
	sriovResNameClient string,
	sriovResNameServer string,
	clientMac string,
	serverMac string,
	clientIPs []string,
	serverIPs []string) (client *pod.Builder, server *pod.Builder, err error) {
	glog.V(90).Infof("Creating client pod with IPs %v, mac %s, SR-IOV resourceName %s"+
		" and server pod with IPs %v, mac %s, SR-IOV resourceName %s.",
		clientIPs, clientMac, sriovResNameClient, serverIPs, serverMac, sriovResNameServer)

	clientPod, err := createAndWaitTestPodWithSecondaryNetwork("client", clientNodeName,
		sriovResNameClient, clientMac, clientIPs)
	if err != nil {
		glog.V(90).Infof("Failed to create clientPod")

		return nil, nil, err
	}

	serverPod, err := createAndWaitTestPodWithSecondaryNetwork("server", serverNodeName,
		sriovResNameServer, serverMac, serverIPs)
	if err != nil {
		glog.V(90).Infof("Failed to create serverPod")

		return nil, nil, err
	}

	return clientPod, serverPod, nil
}

// createAndWaitTestPodWithSecondaryNetwork creates test pod with secondary network
// and waits until it is in the ready state.
func createAndWaitTestPodWithSecondaryNetwork(
	podName string,
	testNodeName string,
	sriovResNameTest string,
	testMac string,
	testIPs []string) (*pod.Builder, error) {
	glog.V(90).Infof("Creating a test pod name %s", podName)

	secNetwork := pod.StaticIPAnnotationWithMacAddress(sriovResNameTest, testIPs, testMac)
	testPod, err := pod.NewBuilder(APIClient, podName, tsparams.TestNamespaceName, NetConfig.CnfNetTestContainer).
		DefineOnNode(testNodeName).WithPrivilegedFlag().
		WithSecondaryNetwork(secNetwork).CreateAndWaitUntilRunning(netparam.DefaultTimeout)

	if err != nil {
		glog.V(90).Infof("Failed to create pod %s with secondary network", podName)

		return nil, err
	}

	return testPod, nil
}

func isVfCreated(sriovNodeState *sriov.NetworkNodeStateBuilder, vfNumber int, sriovInterfaceName string) error {
	sriovNumVfs, err := sriovNodeState.GetNumVFs(sriovInterfaceName)
	if err != nil {
		return err
	}

	if sriovNumVfs != vfNumber {
		return fmt.Errorf("expected number of VFs %d is not equal to the actual number of VFs %d", vfNumber, sriovNumVfs)
	}

	return nil
}

func runCommandOnConfigDaemon(nodeName string, command []string) (string, error) {
	pods, err := pod.List(APIClient, NetConfig.SriovOperatorNamespace, metav1.ListOptions{
		LabelSelector: "app=sriov-network-config-daemon", FieldSelector: fmt.Sprintf("spec.nodeName=%s", nodeName)})
	if err != nil {
		return "", err
	}

	if len(pods) != 1 {
		return "", fmt.Errorf("there should be exactly one 'sriov-network-config-daemon' pod per node,"+
			" but found %d on node %s", len(pods), nodeName)
	}

	output, err := pods[0].ExecCommand(command)

	return output.String(), err
}
