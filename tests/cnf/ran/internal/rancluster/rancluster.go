package rancluster

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/raninittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/ranparam"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/diskencryption/tsparams"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
)

// AreClustersPresent checks all of the provided clusters and returns false if any are nil.
func AreClustersPresent(clusters []*clients.Settings) bool {
	for _, cluster := range clusters {
		if cluster == nil {
			return false
		}
	}

	return true
}

// GetPlusOneWorkerName gets the name of the one worker in a SNO+1 cluster.
func GetPlusOneWorkerName(client *clients.Settings) (string, error) {
	workers, err := ListNodesByLabel(client, RANConfig.WorkerLabelMap)
	if err != nil {
		return "", err
	}

	for _, worker := range workers {
		if !IsNodeControlPlane(worker) {
			return worker.Definition.Name, nil
		}
	}

	return "", fmt.Errorf("could not find a worker node for cluster")
}

// IsClusterStable checks if the provided cluster does not have any unschedulable nodes.
func IsClusterStable(client *clients.Settings) (bool, error) {
	nodeList, err := nodes.List(client)
	if err != nil {
		return false, err
	}

	for _, node := range nodeList {
		if node.Definition.Spec.Unschedulable {
			return false, nil
		}
	}

	return true, nil
}

// IsNodeControlPlane checks whether the provided node is a control plane node.
func IsNodeControlPlane(node *nodes.Builder) bool {
	_, exists := node.Definition.Labels[RANConfig.ControlPlaneLabel]

	return exists
}

// IsSnoPlusOne checks if the specified cluster has one control plane and one worker node.
func IsSnoPlusOne(client *clients.Settings) (bool, error) {
	controlPlanes, err := ListNodesByLabel(client, RANConfig.ControlPlaneLabelMap)
	if err != nil {
		return false, err
	}

	if len(controlPlanes) != 1 {
		return false, nil
	}

	glog.V(tsparams.LogLevel).Info("Exactly one control plane node found")

	workers, err := ListNodesByLabel(client, RANConfig.WorkerLabelMap)
	if err != nil {
		return false, err
	}

	trueWorkers := 0

	for _, worker := range workers {
		if !IsNodeControlPlane(worker) {
			trueWorkers++
		}
	}

	if trueWorkers != 1 {
		return false, nil
	}

	glog.V(tsparams.LogLevel).Info("Exactly one worker node found")

	return true, nil
}

// ListNodesByLabel returns a list of nodes that have the specified label.
func ListNodesByLabel(client *clients.Settings, labelMap map[string]string) ([]*nodes.Builder, error) {
	return nodes.List(client, metav1.ListOptions{
		LabelSelector: labels.Set(labelMap).String(),
	})
}

// WaitForNumberOfNodes waits up to timeout until the number of nodes on the cluster matches the expected.
func WaitForNumberOfNodes(client *clients.Settings, expected int, timeout time.Duration) error {
	return wait.PollUntilContextTimeout(
		context.TODO(), 5*time.Second, timeout, true, func(ctx context.Context) (bool, error) {
			nodeList, err := nodes.List(client)
			if err != nil {
				return false, err
			}

			if len(nodeList) == expected {
				return true, nil
			}

			glog.V(tsparams.LogLevel).Infof("Expected %d nodes but found %d nodes", expected, len(nodeList))

			return false, nil
		})
}

// CheckSpokeClusterType checks and returns a spoke cluster type based on number of control plane nodes.
func CheckSpokeClusterType(client *clients.Settings) (ranparam.ClusterType, error) {
	controlPlaneNodesList, err := ListNodesByLabel(client, RANConfig.ControlPlaneLabelMap)
	if err != nil {
		return "", err
	}

	if len(controlPlaneNodesList) == 1 {
		return ranparam.SNOCluster, nil
	}

	return ranparam.HighlyAvailableCluster, fmt.Errorf("could not determine spoke cluster type")
}
