package helper

import (
	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/ranparam"
	v1 "k8s.io/api/core/v1"
)

// IsPodHealthy returns true if a given pod is healthy, otherwise false.
func IsPodHealthy(pod *pod.Builder) bool {
	if pod.Object.Status.Phase == v1.PodRunning {
		// Check if running pod is ready
		if !isPodInCondition(pod, v1.PodReady) {
			glog.V(ranparam.LogLevel).Infof("pod condition is not Ready. Message: %s", pod.Object.Status.Message)

			return false
		}
	} else if pod.Object.Status.Phase != v1.PodSucceeded {
		// Pod is not running or completed.
		glog.V(ranparam.LogLevel).Infof("pod phase is %s. Message: %s", pod.Object.Status.Phase, pod.Object.Status.Message)

		return false
	}

	return true
}

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

// isPodInCondition returns true if a given pod is in expected condition, otherwise false.
func isPodInCondition(pod *pod.Builder, condition v1.PodConditionType) bool {
	for _, c := range pod.Object.Status.Conditions {
		if c.Type == condition && c.Status == v1.ConditionTrue {
			return true
		}
	}

	return false
}
