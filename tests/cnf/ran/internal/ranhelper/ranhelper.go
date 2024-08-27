package ranhelper

import (
	"context"
	"os/exec"
	"time"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/ranparam"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/diskencryption/tsparams"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
)

// DoesContainerExistInPod checks if a given container exists in a given pod.
func DoesContainerExistInPod(pod *pod.Builder, containerName string) bool {
	containers := pod.Object.Status.ContainerStatuses

	for _, container := range containers {
		if container.Name == containerName {
			glog.V(ranparam.LogLevel).Infof("found %s container", containerName)

			return true
		}
	}

	return false
}

// AreClustersPresent checks all of the provided clusters and returns false if any are nil.
func AreClustersPresent(clusters []*clients.Settings) bool {
	for _, cluster := range clusters {
		if cluster == nil {
			return false
		}
	}

	return true
}

// ListNodesByLabel returns a list of nodes that have the specified label.
func ListNodesByLabel(client *clients.Settings, labelMap map[string]string) ([]*nodes.Builder, error) {
	return nodes.List(client, metav1.ListOptions{
		LabelSelector: labels.Set(labelMap).String(),
	})
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

// UnmarshalRaw converts raw bytes for a K8s CR into the actual type.
func UnmarshalRaw[T any](raw []byte) (*T, error) {
	untyped := &unstructured.Unstructured{}
	err := untyped.UnmarshalJSON(raw)

	if err != nil {
		return nil, err
	}

	var typed T
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(untyped.UnstructuredContent(), &typed)

	if err != nil {
		return nil, err
	}

	return &typed, nil
}

// ExecLocalCommand runs the provided command with the provided args locally, cancelling execution if it exceeds
// timeout.
func ExecLocalCommand(timeout time.Duration, command string, args ...string) (string, error) {
	glog.V(90).Infof("Locally executing command '%s' with args '%v'", command, args)

	ctx, cancel := context.WithTimeout(context.TODO(), timeout)

	defer cancel()

	output, err := exec.CommandContext(ctx, command, args...).Output()

	return string(output), err
}
