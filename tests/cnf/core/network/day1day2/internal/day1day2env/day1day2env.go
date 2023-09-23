package day1day2env

import (
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/golang/glog"

	"github.com/openshift-kni/eco-goinfra/pkg/deployment"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"

	"github.com/openshift-kni/eco-goinfra/pkg/nmstate"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/day1day2/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/cmd"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netinittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netparam"
)

// DoesClusterSupportDay1Day2Tests verifies if given environment supports Day1Day2 tests.
func DoesClusterSupportDay1Day2Tests(requiredCPNodeNumber, requiredWorkerNodeNumber int) error {
	glog.V(90).Infof("Verifying if NMState operator is deployed")

	if err := isNMStateOperatorDeployed(); err != nil {
		return err
	}

	workerNodeList, err := nodes.List(
		APIClient,
		metav1.ListOptions{LabelSelector: labels.Set(NetConfig.WorkerLabelMap).String()},
	)

	if err != nil {
		return err
	}

	glog.V(90).Infof("Verifying if cluster has enough workers to run Day1Day2 tests")

	if len(workerNodeList) < requiredWorkerNodeNumber {
		return fmt.Errorf("cluster has less than %d worker nodes", requiredWorkerNodeNumber)
	}

	controlPlaneNodeList, err := nodes.List(
		APIClient,
		metav1.ListOptions{LabelSelector: labels.Set(NetConfig.ControlPlaneLabelMap).String()},
	)

	if err != nil {
		return err
	}

	glog.V(90).Infof("Verifying if cluster has enough control-plane nodes to run Day1Day2 tests")

	if len(controlPlaneNodeList) < requiredCPNodeNumber {
		return fmt.Errorf("cluster has less than %d control-plane nodes", requiredCPNodeNumber)
	}

	return nil
}

// GetSrIovPf returns SR-IOV PF name for given SR-IOV VF.
func GetSrIovPf(vfInterfaceName, nodeName string) (string, error) {
	glog.V(90).Infof("Getting PF interface name for VF %s on node %s", vfInterfaceName, nodeName)

	pfName, err := cmd.RunCommandOnHostNetworkPod(nodeName, tsparams.TestNamespaceName,
		fmt.Sprintf("ls /sys/class/net/%s/device/physfn/net/", vfInterfaceName))
	if err != nil {
		return "", err
	}

	return pfName, nil
}

// GetBondInterfaceMiimon returns miimon value for given bond interface and node.
func GetBondInterfaceMiimon(nodeName, bondInterfaceName string) (int, error) {
	glog.V(90).Infof("Getting miimon value for bond interface %s on node %s", bondInterfaceName, nodeName)

	nodeNetworkState, err := nmstate.PullNodeNetworkState(APIClient, nodeName)
	if err != nil {
		return 0, err
	}

	bondInterface, err := nodeNetworkState.GetInterfaceType(bondInterfaceName, "bond")
	if err != nil {
		return 0, err
	}

	return bondInterface.LinkAggregation.Options.Miimon, nil
}

// CheckConnectivityBetweenMasterAndWorkers creates a hostnetwork pod on the master node and ping all workers nodes.
// The Pod will be removed at the end.
func CheckConnectivityBetweenMasterAndWorkers() error {
	glog.V(90).Infof("Checking connectivity between master node and worker nodes")

	masterNodes, err := nodes.List(
		APIClient,
		metav1.ListOptions{LabelSelector: labels.Set(NetConfig.ControlPlaneLabelMap).String()},
	)

	if err != nil {
		return err
	}

	workerNodeList, err := nodes.List(
		APIClient,
		metav1.ListOptions{LabelSelector: labels.Set(NetConfig.WorkerLabelMap).String()},
	)

	if err != nil {
		return err
	}

	podMaster, err := pod.NewBuilder(
		APIClient, "mastertestpod", tsparams.TestNamespaceName, NetConfig.CnfNetTestContainer).
		DefineOnNode(masterNodes[0].Definition.Name).WithHostNetwork().
		WithPrivilegedFlag().WithTolerationToMaster().CreateAndWaitUntilRunning(netparam.DefaultTimeout)
	if err != nil {
		return err
	}

	for _, workerNode := range workerNodeList {
		err = cmd.ICMPConnectivityCheck(podMaster, []string{workerNode.Object.Status.Addresses[0].Address + "/24"})
		if err != nil {
			return fmt.Errorf("connectivity check between %s and %s failed: %w",
				masterNodes[0].Definition.Name, workerNode.Object.Name, err)
		}
	}

	_, err = podMaster.DeleteAndWait(netparam.DefaultTimeout)
	if err != nil {
		return err
	}

	return nil
}

func isNMStateOperatorDeployed() error {
	nmstateNS := namespace.NewBuilder(APIClient, NetConfig.NMStateOperatorNamespace)
	if !nmstateNS.Exists() {
		return fmt.Errorf("error NMState namespace %s doesn't exist", nmstateNS.Definition.Name)
	}

	nmstateOperatorDeployment, err := deployment.Pull(
		APIClient, "nmstate-operator", NetConfig.NMStateOperatorNamespace)

	if err != nil {
		return fmt.Errorf("error to pull nmstate-operator deployment from the cluster")
	}

	if !nmstateOperatorDeployment.IsReady(30 * time.Second) {
		return fmt.Errorf("error nmstate-operator deployment is not in ready/ready state")
	}

	return nil
}
