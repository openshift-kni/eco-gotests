package wait

import (
	"context"
	"strings"
	"time"

	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/internal/hwaccelparams"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/nfd/internal/get"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/nfd/nfdparams"

	"github.com/golang/glog"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
)

// ForLabel check that all pods in namespace are in running state.
func ForLabel(apiClient *clients.Settings, timeout time.Duration, label string) (bool, error) {
	err := wait.PollUntilContextTimeout(
		context.TODO(), 5*time.Second, timeout, true, func(ctx context.Context) (bool, error) {
			nodes, e := apiClient.CoreV1Interface.Nodes().List(
				context.TODO(), metav1.ListOptions{})
			if e != nil {
				return false, e
			}

			labelExist := false
			workernodes := filterNodesByLabel(nodes, "worker")

			for _, node := range workernodes {
				labelKeys := getLabelKeys(node.Labels)
				onelineLabels := strings.Join(labelKeys, ", ")

				if strings.Contains(onelineLabels, label) {
					labelExist = true
				} else {
					labelExist = false

					break
				}
			}

			if labelExist {
				return true, nil
			}

			return false, nil
		})

	if err != nil {
		return false, err
	}

	return true, nil
}

// ForNodeReadiness check that all nodes in namespace are Ready.
func ForNodeReadiness(apiClient *clients.Settings,
	timeout time.Duration,
	nodeSelector map[string]string) (bool, error) {
	err := wait.PollUntilContextTimeout(
		context.TODO(), 5*time.Second, timeout, true, func(ctx context.Context) (bool, error) {
			nodes, err := nodes.List(apiClient, metav1.ListOptions{LabelSelector: labels.Set(nodeSelector).String()})
			if err != nil {
				return false, err
			}

			checkReadiness := true

			for _, node := range nodes {
				isNodeReady, err := node.IsReady()
				if err != nil {
					return false, err
				}

				checkReadiness = checkReadiness && isNodeReady
			}

			return checkReadiness, nil
		})

	if err != nil {
		return false, err
	}

	return true, nil
}

// ForPodsRunning check that all pods in namespace are in running state.
func ForPodsRunning(apiClient *clients.Settings, timeout time.Duration, nsname string) (bool, error) {
	err := wait.PollUntilContextTimeout(
		context.TODO(), 5*time.Second, timeout, true, func(ctx context.Context) (bool, error) {
			pods, err := get.PodStatus(apiClient, hwaccelparams.NFDNamespace)
			if err != nil {
				return false, err
			}

			for _, pod := range pods {
				if pod.State != string(corev1.PodRunning) {
					glog.V(nfdparams.LogLevel).Infof("pod %s is in %s state", pod.Name, pod.State)

					return false, nil // not ready yet
				}
			}

			glog.V(nfdparams.LogLevel).Info("all pods are in running status")

			return true, nil
		})

	if err != nil {
		return false, err
	}

	return true, nil
}

func filterNodesByLabel(nodes *corev1.NodeList, keyword string) []corev1.Node {
	var filteredNodes []corev1.Node

	for _, node := range nodes.Items {
		// Check if any label key contains the keyword
		for nodeLabel := range node.Labels {
			if strings.Contains(nodeLabel, keyword) {
				filteredNodes = append(filteredNodes, node)

				break
			}
		}
	}

	return filteredNodes
}

func getLabelKeys(labels map[string]string) []string {
	labelKeys := make([]string, 0, len(labels))
	for k := range labels {
		labelKeys = append(labelKeys, k)
	}

	return labelKeys
}
