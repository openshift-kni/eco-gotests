package wait

import (
	"time"

	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const podIsReadyTimeOut = 300

// WaitForPod check that all pods in namespace are in running state.
func WaitForPod(apiClient *clients.Settings, nsname string) (bool, error) {
	podList, err := pod.List(apiClient, nsname, v1.ListOptions{})
	if err != nil {
		return false, err
	}

	for _, onePod := range podList {
		err = onePod.WaitUntilRunning(podIsReadyTimeOut * time.Second)
		if err != nil {
			return false, err
		}
	}

	return true, nil
}
