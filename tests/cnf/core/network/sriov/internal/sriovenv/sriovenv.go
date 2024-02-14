package sriovenv

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/daemonset"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-goinfra/pkg/sriov"

	sriovV1 "github.com/k8snetworkplumbingwg/sriov-network-operator/api/v1"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netenv"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netinittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/sriov/internal/tsparams"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	err := sriov.CleanAllNetworkNodePolicies(APIClient, NetConfig.SriovOperatorNamespace, v1.ListOptions{})
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
	pods, err := pod.List(APIClient, NetConfig.SriovOperatorNamespace, v1.ListOptions{
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
