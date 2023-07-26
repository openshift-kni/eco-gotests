package netenv

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/daemonset"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/sriov"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netconfig"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netparam"
)

// DoesClusterHasEnoughNodes verifies if given cluster has enough nodes to run tests.
func DoesClusterHasEnoughNodes(
	apiClient *clients.Settings,
	netConfig *netconfig.NetworkConfig,
	requiredCPNodeNumber int,
	requiredWorkerNodeNumber int) error {
	glog.V(90).Infof("Verifying if cluster has enough workers to run tests")

	workerNodeList := nodes.NewBuilder(apiClient, netConfig.WorkerLabelMap)

	err := workerNodeList.Discover()

	if err != nil {
		return err
	}

	if len(workerNodeList.Objects) < requiredWorkerNodeNumber {
		return fmt.Errorf("cluster has less than %d worker nodes", requiredWorkerNodeNumber)
	}

	controlPlaneNodeList := nodes.NewBuilder(apiClient, netConfig.ControlPlaneLabelMap)

	glog.V(90).Infof("Verifying if cluster has enough control-plane nodes to run tests")

	err = controlPlaneNodeList.Discover()

	if err != nil {
		return err
	}

	if len(controlPlaneNodeList.Objects) < requiredCPNodeNumber {
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

	workerNodeList := nodes.NewBuilder(apiClient, netConfig.WorkerLabelMap)
	err := workerNodeList.Discover()

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
	apiClient *clients.Settings, netConfig *netconfig.NetworkConfig, workerNodeList *nodes.Builder) error {
	baseInterfaces, err := sriov.NewNetworkNodeStateBuilder(apiClient, workerNodeList.Objects[0].Definition.Name,
		netConfig.SriovOperatorNamespace).GetNICs()
	if err != nil {
		return fmt.Errorf("failed get SR-IOV devices from the node %s", workerNodeList.Objects[0].Definition.Name)
	}

	for _, node := range workerNodeList.Objects {
		sriovInterfaces, err := sriov.NewNetworkNodeStateBuilder(apiClient, node.Definition.Name,
			netConfig.SriovOperatorNamespace).GetNICs()
		if err != nil {
			return fmt.Errorf("failed get SR-IOV devices from the node %s", node.Definition.Name)
		}

		for index := range sriovInterfaces {
			if baseInterfaces[index].Name != sriovInterfaces[index].Name &&
				baseInterfaces[index].Vendor != sriovInterfaces[index].Vendor &&
				baseInterfaces[index].TotalVfs != sriovInterfaces[index].TotalVfs {
				return fmt.Errorf("SR-IOV network interfaces %s and %s on the Nodes %v are not identical",
					baseInterfaces[index].Name, sriovInterfaces[index].Name, workerNodeList.Objects)
			}
		}
	}

	return nil
}
