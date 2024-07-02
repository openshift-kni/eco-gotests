package cluster

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/raninittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/ranparam"
	"github.com/openshift-kni/eco-gotests/tests/internal/cluster"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/ptr"
)

// ExecLocalCommand runs the provided command with the provided args locally, cancelling execution if it exceeds
// timeout.
func ExecLocalCommand(timeout time.Duration, command string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), timeout)
	defer cancel()

	glog.V(ranparam.LogLevel).Infof("Locally executing command '%s' with args '%v'", command, args)

	output, err := exec.CommandContext(ctx, command, args...).Output()

	return string(output), err
}

// ExecCmd executes a command on the provided client on each node matching nodeSelector, retrying on internal errors
// retries times with a 10 second delay between retries, and ignores the stdout.
func ExecCmd(client *clients.Settings, retries uint, nodeSelector, command string) error {
	for retry := range retries {
		err := cluster.ExecCmd(client, nodeSelector, command)
		if isErrorExecuting(err) {
			glog.V(ranparam.LogLevel).Infof("Error during command execution, retry %d (%d max): %w", retry+1, retries, err)

			time.Sleep(10 * time.Second)

			continue
		}

		return err
	}

	return fmt.Errorf("ran out of %d retries executing command %s", retries, command)
}

// ExecCmdWithStdout executes a command on the provided client, retrying on internal errors retries times with a 10
// second delay between retries, and returns the stdout for each node.
func ExecCmdWithStdout(
	client *clients.Settings, retries uint, command string, options ...metav1.ListOptions) (map[string]string, error) {
	for retry := range retries {
		outputs, err := cluster.ExecCmdWithStdout(client, command, options...)
		if isErrorExecuting(err) {
			glog.V(ranparam.LogLevel).Infof("Error during command execution, retry %d (%d max): %w", retry+1, retries, err)

			time.Sleep(10 * time.Second)

			continue
		}

		return outputs, err
	}

	return nil, fmt.Errorf("ran out of %d retries executing command %s", retries, command)
}

// ExecCommandOnSNO executes a command on the provided single node client, retrying on internal errors retries times
// with a 10 second delay between retries, and returns the stdout.
func ExecCommandOnSNO(client *clients.Settings, retries uint, command string) (string, error) {
	outputs, err := ExecCmdWithStdout(client, retries, command)
	if err != nil {
		return "", err
	}

	if len(outputs) != 1 {
		return "", fmt.Errorf("expected results from one node, found %d nodes", len(outputs))
	}

	for _, output := range outputs {
		return output, nil
	}

	return "", fmt.Errorf("found unreachable code in ExecCommandOnSNO")
}

// isErrorExecuting matches errors that contain the message "error executing command in container".
func isErrorExecuting(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(err.Error(), "error executing command in container") ||
		strings.Contains(err.Error(), "container not found")
}

// WaitForClusterRecover waits up to timeout for all pods in namespaces on a provided node to recover.
func WaitForClusterRecover(client *clients.Settings, namespaces []string, timeout time.Duration) error {
	err := waitForClusterReachable(client, timeout)
	if err != nil {
		return err
	}

	return waitForAllPodsHealthy(client, namespaces, timeout)
}

// waitForClusterReachable waits up to timeout for the cluster to become available by attempting to list nodes in the
// cluster.
func waitForClusterReachable(client *clients.Settings, timeout time.Duration) error {
	return wait.PollUntilContextTimeout(
		context.TODO(), 3*time.Second, timeout, true, func(ctx context.Context) (bool, error) {
			_, err := nodes.List(client, metav1.ListOptions{TimeoutSeconds: ptr.To[int64](3)})
			if err != nil {
				return false, nil
			}

			return true, nil
		})
}

// waitForAllPodsHealthy waits up to timeout for all pods in a cluster or in the provided namespaces to be healthy.
func waitForAllPodsHealthy(client *clients.Settings, namespaces []string, timeout time.Duration) error {
	var namespacesToCheck []string

	if len(namespaces) == 0 {
		namespaceList, err := namespace.List(client)
		if err != nil {
			return err
		}

		for _, ns := range namespaceList {
			namespacesToCheck = append(namespacesToCheck, ns.Object.Name)
		}
	} else {
		namespacesToCheck = append(namespacesToCheck, namespaces...)
	}

	return wait.PollUntilContextTimeout(
		context.TODO(), 15*time.Second, timeout, true, func(ctx context.Context) (bool, error) {
			for _, nsName := range namespacesToCheck {
				glog.V(ranparam.LogLevel).Infof("Checking namespace %s for unhealthy pods", nsName)

				namespacePods, err := pod.List(client, nsName)
				if err != nil {
					return false, err
				}

				for _, namespacePod := range namespacePods {
					healthy := IsPodHealthy(namespacePod)

					// Ignore failed pod with restart policy never. This could happen in image pruner or installer pods that
					// will never restart. For those pods, instead of restarting the same pod, a new pod will be created
					// to complete the task.
					// Temp: Also excludes pods under logging namespace. As we don't have a valid logging server
					// configured, the pod gets stuck in Crashloopback. Remove this after RAN team figures out a workaround.
					if !healthy &&
						namespacePod.Object.Namespace != ranparam.OpenshiftLoggingNamespace &&
						!(namespacePod.Object.Status.Phase == corev1.PodFailed &&
							namespacePod.Object.Spec.RestartPolicy == corev1.RestartPolicyNever) {
						glog.V(ranparam.LogLevel).Infof("Pod %s in namespace %s was unhealthy", namespacePod.Object.Name, nsName)

						return false, nil
					}
				}
			}

			return true, nil
		})
}

// IsPodHealthy returns true if a given pod is healthy, otherwise false.
func IsPodHealthy(pod *pod.Builder) bool {
	if pod.Object.Status.Phase == corev1.PodRunning {
		// Check if running pod is ready
		if !IsPodInCondition(pod, corev1.PodReady) {
			glog.V(ranparam.LogLevel).Infof("pod condition is not Ready. Message: %s", pod.Object.Status.Message)

			return false
		}
	} else if pod.Object.Status.Phase != corev1.PodSucceeded {
		// Pod is not running or completed.
		glog.V(ranparam.LogLevel).Infof("pod phase is %s. Message: %s", pod.Object.Status.Phase, pod.Object.Status.Message)

		return false
	}

	return true
}

// IsPodInCondition returns true if a given pod is in expected condition, otherwise false.
func IsPodInCondition(pod *pod.Builder, condition corev1.PodConditionType) bool {
	for _, c := range pod.Object.Status.Conditions {
		if c.Type == condition && c.Status == corev1.ConditionTrue {
			return true
		}
	}

	return false
}

// RebootSNO Gracefully reboots SNO cluster.
func RebootSNO(path string) error {
	cmdToExec := "sudo systemctl reboot"

	return ExecCmd(raninittools.Spoke1APIClient, 3, ranparam.MasterNodeSelector, cmdToExec)
}
