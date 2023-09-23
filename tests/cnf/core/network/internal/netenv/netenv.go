package netenv

import (
	"fmt"

	"time"

	"github.com/golang/glog"
	v1 "github.com/k8snetworkplumbingwg/sriov-network-operator/api/v1"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/daemonset"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/sriov"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netconfig"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netparam"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/utils/strings/slices"
)

var (
	supportedSrIovDrivers = []string{"mlx5_core", "i40e", "ixgbe", "ice"}
	supportedSrIovDevices = []string{"1583", "1593", "158b", "10fb", "1015", "1017", "101d", "15b3"}
)

// DoesClusterHasEnoughNodes verifies if given cluster has enough nodes to run tests.
func DoesClusterHasEnoughNodes(
	apiClient *clients.Settings,
	netConfig *netconfig.NetworkConfig,
	requiredCPNodeNumber int,
	requiredWorkerNodeNumber int) error {
	glog.V(90).Infof("Verifying if cluster has enough workers to run tests")

	workerNodeList, err := nodes.List(
		apiClient,
		metav1.ListOptions{LabelSelector: labels.Set(netConfig.WorkerLabelMap).String()},
	)

	if err != nil {
		return err
	}

	if len(workerNodeList) < requiredWorkerNodeNumber {
		return fmt.Errorf("cluster has less than %d worker nodes", requiredWorkerNodeNumber)
	}

	controlPlaneNodeList, err := nodes.List(
		apiClient,
		metav1.ListOptions{LabelSelector: labels.Set(netConfig.ControlPlaneLabelMap).String()},
	)

	if err != nil {
		return err
	}

	glog.V(90).Infof("Verifying if cluster has enough control-plane nodes to run tests")

	if len(controlPlaneNodeList) < requiredCPNodeNumber {
		return fmt.Errorf("cluster has less than %d control-plane nodes", requiredCPNodeNumber)
	}

	return nil
}

// DoesClusterSupportSrIovTests verifies if given environment supports SR-IOV tests.
func DoesClusterSupportSrIovTests(
	apiClient *clients.Settings, netConfig *netconfig.NetworkConfig) error {
	glog.V(90).Infof("Verifying if SR-IOV operator deployed")

	if err := isSriovDeployed(apiClient, netConfig); err != nil {
		return err
	}

	workerNodeList, err := nodes.List(
		apiClient,
		metav1.ListOptions{LabelSelector: labels.Set(netConfig.WorkerLabelMap).String()},
	)

	if err != nil {
		return err
	}

	err = compareNodeSriovInterfaces(apiClient, netConfig, workerNodeList)
	if err != nil {
		return err
	}

	return nil
}

func isSriovDeployed(apiClient *clients.Settings, netConfig *netconfig.NetworkConfig) error {
	sriovNS := namespace.NewBuilder(apiClient, netConfig.SriovOperatorNamespace)
	if !sriovNS.Exists() {
		return fmt.Errorf("error SR-IOV namespace %s doesn't exist", sriovNS.Definition.Name)
	}

	for _, sriovDaemonsetName := range netparam.OperatorSriovDaemonsets {
		sriovDaemonset, err := daemonset.Pull(
			apiClient, sriovDaemonsetName, netConfig.SriovOperatorNamespace)

		if err != nil {
			return fmt.Errorf("error to pull SR-IOV daemonset %s from the cluster", sriovDaemonsetName)
		}

		if !sriovDaemonset.IsReady(30 * time.Second) {
			return fmt.Errorf("error SR-IOV daemonset %s is not in ready/ready state",
				sriovDaemonsetName)
		}
	}

	return nil
}

// compareNodeSriovInterfaces validates if all nodes have the same interface spec.
func compareNodeSriovInterfaces(
	apiClient *clients.Settings, netConfig *netconfig.NetworkConfig, workerNodeList []*nodes.Builder) error {
	baseInterfacesFirstNode, err := sriov.NewNetworkNodeStateBuilder(apiClient, workerNodeList[0].Definition.Name,
		netConfig.SriovOperatorNamespace).GetNICs()
	if err != nil {
		return fmt.Errorf("failed get SR-IOV devices from the node %s", workerNodeList[0].Definition.Name)
	}

	supportedSrIovInterfacesOnFirstComputeNode := filterSupportedSrIovDevices(baseInterfacesFirstNode)

	if len(supportedSrIovDevices) < 1 {
		return fmt.Errorf("failed to get supported SR-IOV devices from node %s", workerNodeList[0].Definition.Name)
	}

	for _, node := range workerNodeList {
		sriovInterfaces, err := sriov.NewNetworkNodeStateBuilder(apiClient, node.Definition.Name,
			netConfig.SriovOperatorNamespace).GetNICs()
		if err != nil {
			return fmt.Errorf("failed get SR-IOV devices from the node %s", node.Definition.Name)
		}

		supportedSrIovDevicesOnNode := filterSupportedSrIovDevices(sriovInterfaces)

		for _, supportedSrIovDeviceOneNode := range supportedSrIovDevicesOnNode {
			if !doesSrIovInterfaceInList(supportedSrIovInterfacesOnFirstComputeNode, supportedSrIovDeviceOneNode) {
				return fmt.Errorf("SR-IOV network interfaces %s is not present on the Nodes %v",
					supportedSrIovDeviceOneNode.Name, node.Definition.Name)
			}
		}
	}

	return nil
}

func filterSupportedSrIovDevices(baseInterfaces v1.InterfaceExts) v1.InterfaceExts {
	var supportedSrIovBaseInterfaces v1.InterfaceExts

	for _, srIovInterface := range baseInterfaces {
		if slices.Contains(supportedSrIovDrivers, srIovInterface.Driver) &&
			slices.Contains(supportedSrIovDevices, srIovInterface.DeviceID) {
			supportedSrIovBaseInterfaces = append(supportedSrIovBaseInterfaces, srIovInterface)
		}
	}

	return supportedSrIovBaseInterfaces
}

func doesSrIovInterfaceInList(firstNodeSrIovInterfaces v1.InterfaceExts, nodeSrIovInterface v1.InterfaceExt) bool {
	for index := range firstNodeSrIovInterfaces {
		if firstNodeSrIovInterfaces[index].Name == nodeSrIovInterface.Name &&
			firstNodeSrIovInterfaces[index].Vendor == nodeSrIovInterface.Vendor &&
			firstNodeSrIovInterfaces[index].Driver == nodeSrIovInterface.Driver {
			return true
		}
	}

	return false
}
