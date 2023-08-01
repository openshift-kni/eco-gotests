package day1day2env

import (
	"fmt"
	"time"

	"github.com/golang/glog"

	"github.com/openshift-kni/eco-goinfra/pkg/deployment"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"

	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/day1day2/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/cmd"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netinittools"
)

// DoesClusterSupportDay1Day2Tests verifies if given environment supports Day1Day2 tests.
func DoesClusterSupportDay1Day2Tests(requiredCPNodeNumber, requiredWorkerNodeNumber int) error {
	glog.V(90).Infof("Verifying if NMState operator is deployed")

	if err := isNMStateOperatorDeployed(); err != nil {
		return err
	}

	workerNodeList := nodes.NewBuilder(APIClient, NetConfig.WorkerLabelMap)

	glog.V(90).Infof("Verifying if cluster has enough workers to run Day1Day2 tests")

	err := workerNodeList.Discover()
	if err != nil {
		return err
	}

	if len(workerNodeList.Objects) < requiredWorkerNodeNumber {
		return fmt.Errorf("cluster has less than %d worker nodes", requiredWorkerNodeNumber)
	}

	controlPlaneNodeList := nodes.NewBuilder(APIClient, NetConfig.ControlPlaneLabelMap)

	glog.V(90).Infof("Verifying if cluster has enough control-plane nodes to run Day1Day2 tests")

	err = controlPlaneNodeList.Discover()
	if err != nil {
		return err
	}

	if len(controlPlaneNodeList.Objects) < requiredCPNodeNumber {
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
