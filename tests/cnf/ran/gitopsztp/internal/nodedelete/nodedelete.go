package nodedelete

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/assisted"
	"github.com/openshift-kni/eco-goinfra/pkg/bmh"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/gitopsztp/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/ranhelper"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/raninittools"
	"k8s.io/apimachinery/pkg/util/wait"
)

// IsSnoPlusOne checks if the specified cluster has one control plane and one worker node.
func IsSnoPlusOne(client *clients.Settings) (bool, error) {
	controlPlanes, err := ranhelper.ListNodesByLabel(client, RANConfig.ControlPlaneLabelMap)
	if err != nil {
		return false, err
	}

	if len(controlPlanes) != 1 {
		return false, nil
	}

	glog.V(tsparams.LogLevel).Info("Exactly one control plane node found")

	workers, err := ranhelper.ListNodesByLabel(client, RANConfig.WorkerLabelMap)
	if err != nil {
		return false, err
	}

	trueWorkers := 0

	for _, worker := range workers {
		if !isNodeControlPlane(worker) {
			trueWorkers++
		}
	}

	if trueWorkers != 1 {
		return false, nil
	}

	glog.V(tsparams.LogLevel).Info("Exactly one worker node found")

	return true, nil
}

// GetPlusOneWorkerName gets the name of the one worker in a SNO+1 cluster.
func GetPlusOneWorkerName(client *clients.Settings) (string, error) {
	workers, err := ranhelper.ListNodesByLabel(client, RANConfig.WorkerLabelMap)
	if err != nil {
		return "", err
	}

	for _, worker := range workers {
		if !isNodeControlPlane(worker) {
			return worker.Definition.Name, nil
		}
	}

	return "", fmt.Errorf("could not find a worker node for cluster")
}

// GetBmhNamespace returns the namespace for the specified BareMetalHost, if it exists.
func GetBmhNamespace(client *clients.Settings, bmhName string) (string, error) {
	bmhList, err := bmh.ListInAllNamespaces(client)
	if err != nil {
		return "", err
	}

	for _, bmhBuilder := range bmhList {
		if bmhBuilder.Definition.Name == bmhName {
			return bmhBuilder.Definition.Namespace, nil
		}
	}

	return "", fmt.Errorf("BareMetalHost %s not found", bmhName)
}

// WaitForBMHDeprovisioning waits up to the specified timeout till the BMH and agent with the provided name and
// namespace are no longer found.
func WaitForBMHDeprovisioning(client *clients.Settings, name, namespace string, timeout time.Duration) error {
	return wait.PollUntilContextTimeout(
		context.TODO(), tsparams.ArgoCdChangeInterval, timeout, true, func(ctx context.Context) (bool, error) {
			glog.V(tsparams.LogLevel).Infof("Checking if BareMetalHost %s in namespace %s is deprovisioned", name, namespace)

			_, err := bmh.Pull(client, name, namespace)
			if err == nil {
				return false, nil
			}

			_, err = assisted.PullAgent(client, name, namespace)
			if err == nil {
				return false, nil
			}

			return true, nil
		})
}

// isNodeControlPlane checks whether the provided node is a control plane node.
func isNodeControlPlane(node *nodes.Builder) bool {
	_, exists := node.Definition.Labels[RANConfig.ControlPlaneLabel]

	return exists
}
