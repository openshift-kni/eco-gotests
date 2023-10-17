package wait

import (
	"context"
	"strings"
	"time"

	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

// WaitForLabel check that all pods in namespace are in running state.
func WaitForLabel(apiClient *clients.Settings, timeout time.Duration, label string) (bool, error) {
	err := wait.PollImmediate(time.Second, timeout, func() (bool, error) {
		nodes, e := apiClient.CoreV1Interface.Nodes().List(
			context.TODO(), metaV1.ListOptions{})
		if e != nil {
			return false, e
		}
		for _, node := range nodes.Items {
			labelKeys := getLabelKeys(node.Labels)

			for _, nodeLabel := range labelKeys {

				if strings.Contains(nodeLabel, label) {

					return true, nil
				}

			}
		}

		return false, nil
	})
	if err != nil {
		return false, err
	}

	return true, nil
}

func getLabelKeys(labels map[string]string) []string {
	labelKeys := make([]string, 0, len(labels))
	for k := range labels {
		labelKeys = append(labelKeys, k)
	}

	return labelKeys
}
