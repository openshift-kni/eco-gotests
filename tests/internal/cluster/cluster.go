package cluster

import (
	"regexp"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"

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
	. "github.com/openshift-kni/eco-gotests/tests/internal/inittools"
	"github.com/openshift-kni/eco-gotests/tests/internal/params"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/ptr"
)

// PullTestImageOnNodes pulls given image on range of relevant nodes based on nodeSelector.
func PullTestImageOnNodes(apiClient *clients.Settings, nodeSelector, image string, pullTimeout int) error {
	glog.V(90).Infof("Pulling image %s to nodes with the following label %v", image, nodeSelector)

	nodesList, err := nodes.List(
		apiClient,
		metav1.ListOptions{LabelSelector: labels.Set(map[string]string{nodeSelector: ""}).String()},
	)

	if err != nil {
		return err
	}

	for _, node := range nodesList {
		glog.V(90).Infof("Pulling image %s to node %s", image, node.Object.Name)
		podBuilder := pod.NewBuilder(
			apiClient, fmt.Sprintf("pullpod-%s", node.Object.Name), "default", image)
		err := podBuilder.PullImage(time.Duration(pullTimeout)*time.Second, []string{
			"/bin/sh", "-c", "echo image Pulled && exit 0"})

		if err != nil {
			return err
		}
	}

	return nil
}

// ExecCmd runc cmd on all nodes that match nodeSelector.
func ExecCmd(apiClient *clients.Settings, nodeSelector string, shellCmd string) error {
	glog.V(90).Infof("Executing cmd: %v on nodes based on label: %v using mcp pods", shellCmd, nodeSelector)

	nodeList, err := nodes.List(
		apiClient,
		metav1.ListOptions{LabelSelector: labels.Set(map[string]string{nodeSelector: ""}).String()},
	)
	if err != nil {
		return err
	}

	for _, node := range nodeList {
		listOptions := metav1.ListOptions{
			FieldSelector: fields.SelectorFromSet(fields.Set{"spec.nodeName": node.Definition.Name}).String(),
			LabelSelector: labels.SelectorFromSet(labels.Set{"k8s-app": GeneralConfig.MCOConfigDaemonName}).String(),
		}

		mcPodList, err := pod.List(apiClient, GeneralConfig.MCONamespace, listOptions)
		if err != nil {
			return err
		}

		for _, mcPod := range mcPodList {
			err = mcPod.WaitUntilRunning(300 * time.Second)
			if err != nil {
				return err
			}

			cmdToExec := []string{"sh", "-c", fmt.Sprintf("nsenter --mount=/proc/1/ns/mnt -- sh -c '%s'", shellCmd)}

			glog.V(90).Infof("Exec cmd %v on pod %s", cmdToExec, mcPod.Definition.Name)
			buf, err := mcPod.ExecCommand(cmdToExec)

			if err != nil {
				return fmt.Errorf("%w\n%s", err, buf.String())
			}
		}
	}

	return nil
}

// ExecCmdWithStdout runs cmd on all selected nodes and returns their stdout.
func ExecCmdWithStdout(
	apiClient *clients.Settings, shellCmd string, options ...metav1.ListOptions) (map[string]string, error) {
	if GeneralConfig.MCOConfigDaemonName == "" {
		return nil, fmt.Errorf("error: mco config daemon pod name cannot be empty")
	}

	if GeneralConfig.MCONamespace == "" {
		return nil, fmt.Errorf("error: mco namespace cannot be empty")
	}

	logMessage := fmt.Sprintf("Executing cmd: %v on nodes", shellCmd)

	passedOptions := metav1.ListOptions{}

	if len(options) > 1 {
		glog.V(90).Infof("'options' parameter must be empty or single-valued")

		return nil, fmt.Errorf("error: more than one ListOptions was passed")
	}

	if len(options) == 1 {
		passedOptions = options[0]
		logMessage += fmt.Sprintf(" with the options %v", passedOptions)
	}

	glog.V(90).Infof(logMessage)

	nodeList, err := nodes.List(
		apiClient,
		passedOptions,
	)

	if err != nil {
		return nil, err
	}

	glog.V(90).Infof("Found %d nodes matching selector", len(nodeList))

	outputMap := make(map[string]string)

	for _, node := range nodeList {
		listOptions := metav1.ListOptions{
			FieldSelector: fields.SelectorFromSet(fields.Set{"spec.nodeName": node.Definition.Name}).String(),
			LabelSelector: labels.SelectorFromSet(labels.Set{"k8s-app": GeneralConfig.MCOConfigDaemonName}).String(),
		}

		mcPodList, err := pod.List(apiClient, GeneralConfig.MCONamespace, listOptions)
		if err != nil {
			return nil, err
		}

		for _, mcPod := range mcPodList {
			err = mcPod.WaitUntilRunning(300 * time.Second)
			if err != nil {
				return nil, err
			}

			hostnameCmd := []string{"sh", "-c", "nsenter --mount=/proc/1/ns/mnt -- sh -c 'printf $(hostname)'"}
			hostnameBuf, err := mcPod.ExecCommand(hostnameCmd)

			if err != nil {
				return nil, fmt.Errorf("failed gathering node hostname: %w", err)
			}

			cmdToExec := []string{"sh", "-c", fmt.Sprintf("nsenter --mount=/proc/1/ns/mnt -- sh -c '%s'", shellCmd)}

			glog.V(90).Infof("Exec cmd %v on pod %s", cmdToExec, mcPod.Definition.Name)
			commandBuf, err := mcPod.ExecCommand(cmdToExec)

			if err != nil {
				return nil, fmt.Errorf("failed executing command '%s' on node %s: %w", shellCmd, hostnameBuf.String(), err)
			}

			hostname := regexp.MustCompile(`\r`).ReplaceAllString(hostnameBuf.String(), "")
			output := regexp.MustCompile(`\r`).ReplaceAllString(commandBuf.String(), "")

			outputMap[hostname] = output
		}
	}

	return outputMap, nil
}

// new commands

// ExecLocalCommand runs the provided command with the provided args locally, cancelling execution if it exceeds
// timeout.
func ExecLocalCommand(timeout time.Duration, command string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), timeout)
	defer cancel()

	glog.V(params.LogLevel).Infof("Locally executing command '%s' with args '%v'", command, args)

	output, err := exec.CommandContext(ctx, command, args...).Output()

	return string(output), err
}

// ExecCmdWithRetries executes a command on the provided client on each node matching nodeSelector,
// retrying on internal errors
// retries times with a 10 second delay between retries, and ignores the stdout.
func ExecCmdWithRetries(client *clients.Settings, retries uint, nodeSelector, command string) error {
	for retry := range retries {
		err := ExecCmd(client, nodeSelector, command)
		if isErrorExecuting(err) {
			glog.V(params.LogLevel).Infof("Error during command execution, retry %d (%d max): %w", retry+1, retries, err)

			time.Sleep(10 * time.Second)

			continue
		}

		return err
	}

	return fmt.Errorf("ran out of %d retries executing command %s", retries, command)
}

// ExecCmdWithStdoutWithRetries executes a command on the provided client,
// retrying on internal errors retries times with a 10
// second delay between retries, and returns the stdout for each node.
func ExecCmdWithStdoutWithRetries(
	client *clients.Settings, retries uint, command string, options ...metav1.ListOptions) (map[string]string, error) {
	for retry := range retries {
		outputs, err := ExecCmdWithStdout(client, command, options...)
		if isErrorExecuting(err) {
			glog.V(params.LogLevel).Infof("Error during command execution, retry %d (%d max): %w", retry+1, retries, err)

			time.Sleep(10 * time.Second)

			continue
		}

		return outputs, err
	}

	return nil, fmt.Errorf("ran out of %d retries executing command %s", retries, command)
}

// ExecCommandOnSNOWithRetries executes a command on the provided single node client,
// retrying on internal errors retries times
// with a 10 second delay between retries, and returns the stdout.
func ExecCommandOnSNOWithRetries(client *clients.Settings, retries uint, command string) (string, error) {
	outputs, err := ExecCmdWithStdoutWithRetries(client, retries, command)
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
func WaitForClusterRecover(client *clients.Settings, namespaces []string, timeout, extraWait time.Duration) error {
	err := waitForClusterReachable(client, timeout)
	if err != nil {
		return err
	}

	err = waitForAllPodsHealthy(client, namespaces, timeout)
	if err != nil {
		return err
	}

	time.Sleep(extraWait)

	return nil
}

// IsPodHealthy returns true if a given pod is healthy, otherwise false.
func IsPodHealthy(pod *pod.Builder) bool {
	if pod.Object.Status.Phase == corev1.PodRunning {
		// Check if running pod is ready
		if !IsPodInCondition(pod, corev1.PodReady) {
			glog.V(params.LogLevel).Infof("pod condition is not Ready. Message: %s", pod.Object.Status.Message)

			return false
		}
	} else if pod.Object.Status.Phase != corev1.PodSucceeded {
		// Pod is not running or completed.
		glog.V(params.LogLevel).Infof("pod phase is %s. Message: %s", pod.Object.Status.Phase, pod.Object.Status.Message)

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

// SoftRebootSNO executes systemctl reboot on a node.
func SoftRebootSNO(apiClient *clients.Settings) error {
	cmdToExec := "sudo systemctl reboot"

	_, err := ExecCommandOnSNOWithRetries(apiClient, 3, cmdToExec)

	return err
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
				glog.V(params.LogLevel).Infof("Checking namespace %s for unhealthy pods", nsName)

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
						namespacePod.Object.Namespace != params.OpenshiftLoggingNamespace &&
						!(namespacePod.Object.Status.Phase == corev1.PodFailed &&
							namespacePod.Object.Spec.RestartPolicy == corev1.RestartPolicyNever) {
						glog.V(params.LogLevel).Infof("Pod %s in namespace %s was unhealthy", namespacePod.Object.Name, nsName)

						return false, nil
					}
				}
			}

			return true, nil
		})
}
