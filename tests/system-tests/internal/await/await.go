package await

import (
	"fmt"
	"time"

	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/deployment"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/pod"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/statefulset"

	"github.com/golang/glog"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/clients"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WaitUntilAllDeploymentsReady waits for the duration of the defined timeout or until all deployments
// in the namespace reach the Ready condition.
func WaitUntilAllDeploymentsReady(apiClient *clients.Settings, nsname string, timeout time.Duration) (bool, error) {
	deployments, err := deployment.List(apiClient, nsname, metaV1.ListOptions{})

	if err != nil {
		glog.V(100).Infof("deployment list error: %s", err)

		return false, err
	}

	for _, testDeployment := range deployments {
		if !testDeployment.IsReady(timeout) {
			return false, fmt.Errorf("deployment %s not ready in time. available replicas: %d",
				testDeployment.Definition.Name, testDeployment.Object.Status.AvailableReplicas)
		}
	}

	return true, nil
}

// WaitUntilAllStatefulSetsReady waits for the duration of the defined timeout or until all deployments
// in the namespace reach the Ready condition.
func WaitUntilAllStatefulSetsReady(apiClient *clients.Settings, nsname string, timeout time.Duration) (bool, error) {
	statefulsets, err := statefulset.List(apiClient, nsname, metaV1.ListOptions{})

	if err != nil {
		glog.V(100).Infof("statefulsets list error: %s", err)

		return false, err
	}

	for _, testStatefulset := range statefulsets {
		if !testStatefulset.IsReady(timeout) {
			return false, fmt.Errorf("statefulset %s not ready in time. available replicas: %d",
				testStatefulset.Definition.Name, testStatefulset.Object.Status.AvailableReplicas)
		}
	}

	return true, nil
}

// WaitUntilAllPodsReady waits for the duration of the defined timeout or until all deployments
// in the namespace reach the Ready condition.
func WaitUntilAllPodsReady(apiClient *clients.Settings, nsname string, timeout time.Duration) (bool, error) {
	pods, err := pod.List(apiClient, nsname, metaV1.ListOptions{})

	if err != nil {
		glog.V(100).Infof("pods list error: %s", err)

		return false, err
	}

	for _, testPod := range pods {
		err = testPod.WaitUntilReady(timeout)
		if err != nil {
			glog.V(100).Infof("pod %s did not become ready in time: %s", testPod.Object.Name, err)

			return false, err
		}
	}

	return true, nil
}
