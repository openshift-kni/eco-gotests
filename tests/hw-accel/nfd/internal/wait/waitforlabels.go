package wait

import (
	"context"

	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"golang.org/x/exp/slices"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WaitForLabel check that all pods in namespace are in running state.
func WaitForLabel(apiClient *clients.Settings, label string) (bool, error) {
	nodes, e := apiClient.CoreV1Interface.Nodes().List(
		context.TODO(), metaV1.ListOptions{})

	if e != nil {
		return false, e
	}

	for _, node := range nodes.Items {
		labelKeys := getLabelKeys(node.Labels)
		if slices.Contains(labelKeys, label) {
			return true, nil
		}
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
