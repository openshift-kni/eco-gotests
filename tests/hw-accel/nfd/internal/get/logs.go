package get

import (
	"context"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/nfd/nfdparams"
	v1 "k8s.io/api/core/v1"
)

// PodLogs get a raw log from pod.
func PodLogs(apiClient *clients.Settings, nsname, podName string) (string, error) {
	podWorker, err := pod.Pull(apiClient, podName, nsname)
	podLogOpts := v1.PodLogOptions{}

	if err != nil {
		return "", err
	}

	podLogs := apiClient.Pods(nsname).GetLogs(podWorker.Definition.Name, &podLogOpts).Do(context.Background())

	body, err := podLogs.Raw()
	if err != nil {
		glog.V(nfdparams.LogLevel).Infof("Failed to retrieve logs %v", err)
	}

	return string(body), nil
}
