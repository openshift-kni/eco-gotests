package sriovenv

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	sriovV1 "github.com/k8snetworkplumbingwg/sriov-network-operator/api/v1"

	"github.com/openshift-kni/eco-goinfra/pkg/daemonset"
	"github.com/openshift-kni/eco-goinfra/pkg/mco"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/sriov"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	. "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netinittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/sriov/internal/tsparams"
)

// DoesClusterSupportSriovTests verifies if given environment supports SR-IOV tests.
func DoesClusterSupportSriovTests(requiredCPNodeNumber, requiredWorkerNodeNumber int) error {
	glog.V(90).Infof("Verifying if SR-IOV operator deployed")

	if err := isSriovDeployed(); err != nil {
		return err
	}

	workerNodeList := nodes.NewBuilder(APIClient, NetConfig.WorkerLabelMap)

	glog.V(90).Infof("Verifying if cluster has enough workers to run SR-IOV tests")

	err := workerNodeList.Discover()
	if err != nil {
		return err
	}

	if len(workerNodeList.Objects) < requiredWorkerNodeNumber {
		return fmt.Errorf("cluster has less than %d worker nodes", requiredWorkerNodeNumber)
	}

	controlPlaneNodeList := nodes.NewBuilder(APIClient, NetConfig.ControlPlaneLabelMap)

	glog.V(90).Infof("Verifying if cluster has enough control-plane nodes to run SR-IOV tests")

	err = controlPlaneNodeList.Discover()
	if err != nil {
		return err
	}

	if len(controlPlaneNodeList.Objects) < requiredCPNodeNumber {
		return fmt.Errorf("cluster has less than %d control-plane nodes", requiredCPNodeNumber)
	}

	glog.V(90).Infof("Verifying if workers have the same SR-IOV interfaces")

	err = compareNodeSriovInterfaces(workerNodeList)
	if err != nil {
		return err
	}

	return nil
}

// ValidateSriovInterfaces checks that provided interfaces by env var exist on the nodes.
func ValidateSriovInterfaces(workerNodeList *nodes.Builder, requestedNumber int) error {
	var validSriovIntefaceList []sriovV1.InterfaceExt

	availableUpSriovInterfaces, err := sriov.NewNetworkNodeStateBuilder(APIClient,
		workerNodeList.Objects[0].Definition.Name, NetConfig.SriovOperatorNamespace).GetUpNICs()

	if err != nil {
		return fmt.Errorf("failed get SR-IOV devices from the node %s", workerNodeList.Objects[0].Definition.Name)
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

// WaitForSriovAndMCPStable waits until SR-IOV and MCP stable.
func WaitForSriovAndMCPStable(waitingTime time.Duration) error {
	glog.V(90).Infof("Waiting for SR-IOV and MCP become stable.")

	err := waitForSriovStable(waitingTime)
	if err != nil {
		return err
	}

	err = waitForMcpStable(waitingTime, NetConfig.CnfMcpLabel)
	if err != nil {
		return err
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

	err = WaitForSriovAndMCPStable(timeout)
	if err != nil {
		return err
	}

	return nil
}

// WaitUntilVfsCreated waits until all expected SR-IOV VFs are created.
func WaitUntilVfsCreated(
	nodes *nodes.Builder, sriovInterfaceName string, numberOfVfs int, timeout time.Duration) error {
	glog.V(90).Infof("Waiting for the creation of all VFs (%d) under"+
		" the %s interface in the SriovNetworkState.", numberOfVfs, sriovInterfaceName)

	for _, node := range nodes.Objects {
		err := wait.PollImmediate(time.Second, timeout, func() (bool, error) {
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

// RemoveAllPoliciesAndWaitForSriovAndMCPStable removes all  SriovNetworkNodePolicies and waits until
// SR-IOV and MCP become stable.
func RemoveAllPoliciesAndWaitForSriovAndMCPStable() error {
	glog.V(90).Infof("Deleting all SriovNetworkNodePolicies and waiting for SR-IOV and MCP become stable.")

	err := sriov.CleanAllNetworkNodePolicies(APIClient, NetConfig.SriovOperatorNamespace, v1.ListOptions{})
	if err != nil {
		return err
	}

	return waitForSriovStable(tsparams.SriovStableTimeout)
}

// compareNodeSriovInterfaces validates if all nodes have the same interface spec.
func compareNodeSriovInterfaces(workerNodeList *nodes.Builder) error {
	baseInterfaces, err := sriov.NewNetworkNodeStateBuilder(APIClient, workerNodeList.Objects[0].Definition.Name,
		NetConfig.SriovOperatorNamespace).GetNICs()
	if err != nil {
		return fmt.Errorf("failed get SR-IOV devices from the node %s", workerNodeList.Objects[0].Definition.Name)
	}

	for _, node := range workerNodeList.Objects {
		sriovInterfaces, err := sriov.NewNetworkNodeStateBuilder(APIClient, node.Definition.Name,
			NetConfig.SriovOperatorNamespace).GetNICs()
		if err != nil {
			return fmt.Errorf("failed get SR-IOV devices from the node %s", node.Definition.Name)
		}

		for index := range sriovInterfaces {
			if baseInterfaces[index].Name != sriovInterfaces[index].Name &&
				baseInterfaces[index].Vendor != sriovInterfaces[index].Vendor &&
				baseInterfaces[index].TotalVfs != sriovInterfaces[index].TotalVfs {
				return fmt.Errorf("SR-IOV network interfaces on the Nodes %v are not identical", workerNodeList.Objects)
			}
		}
	}

	return nil
}

func isSriovDeployed() error {
	sriovNS := namespace.NewBuilder(APIClient, NetConfig.SriovOperatorNamespace)
	if !sriovNS.Exists() {
		return fmt.Errorf("error SR-IOV namespace %s doesn't exist", sriovNS.Definition.Name)
	}

	for _, sriovDaemonsetName := range tsparams.OperatorSriovDaemonsets {
		sriovDaemonset, err := daemonset.Pull(
			APIClient, sriovDaemonsetName, NetConfig.SriovOperatorNamespace)

		if err != nil {
			return fmt.Errorf("error to pull SR-IOV daemonset %s from the cluster", sriovDaemonsetName)
		}

		if !sriovDaemonset.IsReady(30 * time.Second) {
			return fmt.Errorf("error SR-IOV deployment %s is not in ready/ready state",
				sriovDaemonsetName)
		}
	}

	return nil
}

// waitForSriovStable waits until all the SR-IOV node states are in sync.
func waitForSriovStable(waitingTime time.Duration) error {
	networkNodeStateList, err := sriov.ListNetworkNodeState(APIClient, NetConfig.SriovOperatorNamespace, v1.ListOptions{})

	if err != nil {
		return fmt.Errorf("failed to fetch nodes state %w", err)
	}

	if len(networkNodeStateList) == 0 {
		return nil
	}

	for _, nodeState := range networkNodeStateList {
		err = nodeState.WaitUntilSyncStatus("Succeeded", waitingTime)
		if err != nil {
			return err
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

// waitForMcpStable waits for the stability of the MCP with the given name.
func waitForMcpStable(waitingTime time.Duration, mcpName string) error {
	mcp := mco.NewMCPBuilder(APIClient, mcpName)
	err := mcp.WaitToBeStableFor(tsparams.DefaultStableDuration, waitingTime)

	if err != nil {
		return fmt.Errorf("cluster is not stable: %s", err.Error())
	}

	return nil
}
