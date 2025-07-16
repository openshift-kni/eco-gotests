package get

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/nfd/internal/nfdhelpersparams"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/nfd/nfdparams"
)

// OLD VERSION - using custom cpucheck wrapper (commented out)
/*
// CPUInfo retrieves cpu info of worker node.
func CPUInfo(apiClient *clients.Settings, name, nsname, containerName, image string) map[string]string {
	workerNodesNames, err := getWorkerNodes(apiClient)
	if err != nil {
		glog.V(nfdparams.LogLevel).Infof("Error getting worker nodes: %v\n", err)

		return nil
	}

	nodeCPUFlagsMap := make(map[string]string)

	for _, nodeName := range workerNodesNames {
		podWorker := pod.NewBuilder(apiClient, nodeName+name, nsname, image)
		containerBuilder := pod.NewContainerBuilder(containerName, image, []string{"./cpucheck"})
		container, err := containerBuilder.GetContainerCfg()

		if err != nil {
			glog.V(nfdparams.LogLevel).Infof("Failed to define the default container settings %v", err)
		}

		podWorker.Definition.Spec.Containers = make([]corev1.Container, 0)
		podWorker.Definition.Spec.Containers = append(podWorker.Definition.Spec.Containers, *container)
		podWorker.Definition.Spec.RestartPolicy = corev1.RestartPolicyNever
		podWorker.Definition.Spec.NodeName = strings.ReplaceAll(nodeName, ".", "")
		podWorker.Definition.Spec.NodeSelector = map[string]string{"node-role.kubernetes.io/worker": ""}

		_, err = podWorker.Create()
		if err != nil {
			glog.V(nfdparams.LogLevel).Infof(
				"Error creating pod: %s", err.Error())

			return nil
		}

		err = podWorker.WaitUntilInStatus(corev1.PodSucceeded, 5*time.Minute)
		if err != nil {
			glog.V(nfdparams.LogLevel).Infof(
				"Error in waiting for pod: %s", err.Error())

			return nil
		}

		con := corev1.PodLogOptions{
			Container: containerName,
		}

		podLogs := apiClient.Pods(nsname).GetLogs(podWorker.Definition.Name, &con).Do(context.TODO())
		_, err = podWorker.DeleteAndWait(5 * time.Minute)

		if err != nil {
			glog.V(nfdparams.LogLevel).Infof(
				"Error creating pod: %s", err.Error())

			return nil
		}

		err = podLogs.Error()
		if err != nil {
			glog.V(nfdparams.LogLevel).Infof(
				"Failed to retrieve logs: %s", err.Error())

			return nil
		}

		body, err := podLogs.Raw()
		if err != nil {
			glog.V(nfdparams.LogLevel).Infof("Failed to retrieve logs %v", err)
		}

		nodeCPUFlagsMap[nodeName] = fmt.Sprint(string(body))
	}

	return nodeCPUFlagsMap
}

// CPUFlags returns cpu flags list.
func CPUFlags(apiClient *clients.Settings, nsName string, workerImage string) map[string][]string {
	nodeCPUFlagmap := CPUInfo(apiClient,
		nfdhelpersparams.PodName,
		nsName,
		nfdhelpersparams.ContainerName,
		workerImage)

	nodeCPUFlagsMap := make(map[string][]string)

	for nodeName, nodeLabels := range nodeCPUFlagmap {
		startIndex := strings.Index(nodeLabels, "Features")
		newlineIndex := strings.Index(nodeLabels[startIndex:], "\n")

		flags := strings.Split(strings.ReplaceAll(nodeLabels[startIndex:startIndex+newlineIndex], "Features: ", ""), ",")
		nodeCPUFlagsMap[nodeName] = flags
	}

	return nodeCPUFlagsMap
}
*/

// CPUInfo retrieves cpu info of worker node using standard Kubernetes client-go and /proc/cpuinfo approach.
func CPUInfo(apiClient *clients.Settings, name, nsname, containerName, image string) map[string]string {
	workerNodesNames, err := getWorkerNodes(apiClient)
	if err != nil {
		glog.V(nfdparams.LogLevel).Infof("Error getting worker nodes: %v\n", err)

		return nil
	}

	nodeCPUFlagsMap := make(map[string]string)

	for _, nodeName := range workerNodesNames {
		// Create valid pod name (DNS-1123 compliant)
		podName := sanitizePodName(fmt.Sprintf("cpu-info-%s", nodeName))

		// Create pod definition using standard Kubernetes client
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      podName,
				Namespace: nsname,
				Labels: map[string]string{
					"app": "cpu-info",
				},
			},
			Spec: corev1.PodSpec{
				RestartPolicy: corev1.RestartPolicyNever,
				NodeName:      nodeName,
				NodeSelector: map[string]string{
					"node-role.kubernetes.io/worker": "",
				},
				Containers: []corev1.Container{
					{
						Name:    containerName,
						Image:   image,
						Command: []string{"./cpucheck"},
					},
				},
			},
		}

		// Create the pod using standard clientץ
		_, err := apiClient.CoreV1Interface.Pods(nsname).Create(context.TODO(), pod, metav1.CreateOptions{})
		if err != nil {
			glog.V(nfdparams.LogLevel).Infof("Error creating pod %s: %s", podName, err.Error())

			continue
		}

		glog.V(nfdparams.LogLevel).Infof("Created pod %s on node %s", podName, nodeName)

		// Wait for pod to succeedץ
		err = waitForPodCompletion(apiClient, nsname, podName, 5*time.Minute)
		if err != nil {
			glog.V(nfdparams.LogLevel).Infof("Error waiting for pod %s: %s", podName, err.Error())
			// Clean up failed podץ

			if delErr := apiClient.CoreV1Interface.Pods(nsname).
				Delete(context.TODO(), podName, metav1.DeleteOptions{}); delErr != nil {
				glog.V(nfdparams.LogLevel).Infof(
					"Error cleaning up failed pod %s: %s", podName, delErr.Error())
			}

			continue
		}

		// Get pod logs
		podLogs := apiClient.CoreV1Interface.Pods(nsname).GetLogs(podName, &corev1.PodLogOptions{
			Container: containerName,
		}).Do(context.TODO())

		// Delete the pod
		err = apiClient.CoreV1Interface.Pods(nsname).Delete(context.TODO(), podName, metav1.DeleteOptions{})
		if err != nil {
			glog.V(nfdparams.LogLevel).Infof("Error deleting pod %s: %s", podName, err.Error())
		}

		// Process logs
		if podLogs.Error() != nil {
			glog.V(nfdparams.LogLevel).Infof("Failed to retrieve logs from pod %s: %s", podName, podLogs.Error())

			continue
		}

		body, err := podLogs.Raw()
		if err != nil {
			glog.V(nfdparams.LogLevel).Infof("Failed to retrieve logs from pod %s: %v", podName, err)

			continue
		}

		nodeCPUFlagsMap[nodeName] = string(body)

		glog.V(nfdparams.LogLevel).Infof("Successfully retrieved CPU info from node %s", nodeName)
	}

	return nodeCPUFlagsMap
}

// sanitizePodName converts node name to valid DNS-1123 pod name.
func sanitizePodName(nodeName string) string {
	// Replace invalid characters with hyphens
	reg := regexp.MustCompile(`[^a-z0-9\-]`)
	sanitized := reg.ReplaceAllString(strings.ToLower(nodeName), "-")

	// Remove leading/trailing hyphens
	sanitized = strings.Trim(sanitized, "-")

	// Ensure it doesn't exceed 63 characters
	if len(sanitized) > 63 {
		sanitized = sanitized[:63]
		sanitized = strings.TrimSuffix(sanitized, "-")
	}

	return sanitized
}

// waitForPodCompletion waits for pod to reach Succeeded or Failed state.
func waitForPodCompletion(apiClient *clients.Settings, namespace, podName string, timeout time.Duration) error {
	return wait.PollUntilContextTimeout(
		context.TODO(),
		5*time.Second,
		timeout,
		true,
		func(ctx context.Context) (bool, error) {
			pod, err := apiClient.CoreV1Interface.Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
			if err != nil {
				return false, err
			}

			switch pod.Status.Phase {
			case corev1.PodSucceeded:
				return true, nil
			case corev1.PodFailed:
				return false, fmt.Errorf("pod %s failed: %s", podName, pod.Status.Message)
			case corev1.PodPending, corev1.PodRunning, corev1.PodUnknown:
				glog.V(nfdparams.LogLevel).Infof("Pod %s is in phase %s", podName, pod.Status.Phase)

				return false, nil
			default:
				glog.V(nfdparams.LogLevel).Infof("Pod %s is in unexpected phase %s", podName, pod.Status.Phase)

				return false, nil
			}
		})
}

// CPUFlags returns cpu flags list using original parsing logic.
func CPUFlags(apiClient *clients.Settings, nsName string, workerImage string) map[string][]string {
	nodeCPUFlagmap := CPUInfo(apiClient,
		nfdhelpersparams.PodName,
		nsName,
		nfdhelpersparams.ContainerName,
		workerImage)

	nodeCPUFlagsMap := make(map[string][]string)

	for nodeName, nodeLabels := range nodeCPUFlagmap {
		startIndex := strings.Index(nodeLabels, "Features")
		newlineIndex := strings.Index(nodeLabels[startIndex:], "\n")

		flags := strings.Split(strings.ReplaceAll(nodeLabels[startIndex:startIndex+newlineIndex], "Features: ", ""), ",")
		nodeCPUFlagsMap[nodeName] = flags
	}

	return nodeCPUFlagsMap
}

func getWorkerNodes(clientset *clients.Settings) ([]string, error) {
	labelSelector := "node-role.kubernetes.io/worker="

	// Create ListOptions with the label selector
	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
	}

	workerNodes, err := nodes.List(clientset, listOptions)
	if err != nil {
		return nil, err
	}

	var workerNodeNames []string
	for _, node := range workerNodes {
		workerNodeNames = append(workerNodeNames, node.Object.Name)
	}

	return workerNodeNames, nil
}
