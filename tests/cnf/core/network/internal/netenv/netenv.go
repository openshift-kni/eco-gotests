package netenv

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/daemonset"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netconfig"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netparam"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
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

// IsSriovDeployed verifies that the sriov operator is deployed.
func IsSriovDeployed(apiClient *clients.Settings, netConfig *netconfig.NetworkConfig) error {
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

// BFDHasStatus verifies that BFD session on a pod has given status.
func BFDHasStatus(frrPod *pod.Builder, bfdPeer string, status string) error {
	bfdStatusOut, err := frrPod.ExecCommand(append(netparam.VtySh, "sh bfd peers brief json"))
	if err != nil {
		return err
	}

	var result []netparam.BFDDescription

	err = json.Unmarshal(bfdStatusOut.Bytes(), &result)
	if err != nil {
		return err
	}

	for _, peer := range result {
		if peer.BFDPeer == bfdPeer && peer.BFDStatus != status {
			return fmt.Errorf("%s bfd status is %s (expected %s)", peer.BFDPeer, peer.BFDStatus, status)
		}
	}

	return nil
}
