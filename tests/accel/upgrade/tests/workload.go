package upgrade

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/deployment"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-goinfra/pkg/route"
	"github.com/openshift-kni/eco-goinfra/pkg/service"

	upgradeinittools "github.com/openshift-kni/eco-gotests/tests/accel/internal/accelinittools"
	"github.com/openshift-kni/eco-gotests/tests/accel/upgrade/internal/accelparams"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	retryDurationSecs = 360
	pollIntervalSecs  = 20
)

// CreateWorkload Create a workload with test image.
func CreateWorkload(apiClient *clients.Settings, workloadImage string) (*deployment.Builder, error) {
	glog.V(90).Infof("Creating Deployment %q", accelparams.DeploymentName)

	workloadDeployment, err := deployment.NewBuilder(
		upgradeinittools.HubAPIClient, accelparams.DeploymentName, accelparams.TestNamespaceName, map[string]string{
			"app": accelparams.DeploymentName,
		}, &corev1.Container{
			Name:  accelparams.DeploymentName,
			Image: accelparams.IbuWorkloadImage,
			Ports: []corev1.ContainerPort{
				{
					Name:          "http",
					ContainerPort: 8080,
				},
			},
		}).WithLabel("app", accelparams.DeploymentName).CreateAndWaitUntilReady(time.Second * 60)

	if err != nil {
		return nil, fmt.Errorf("failed to create workload with error: %w", err)
	}

	return workloadDeployment, nil
}

// DeleteWorkload Delete a workload.
// Return nil on success, otherwise return an error.
func DeleteWorkload(apiClient *clients.Settings) error {
	var (
		oldPods []*pod.Builder
		err     error
	)

	totalPollTime := 0
	pollSuccess := false

	continueLooping := true
	for continueLooping {
		oldPods, err = pod.List(apiClient, accelparams.TestNamespaceName,
			metav1.ListOptions{LabelSelector: accelparams.ContainerLabelsStr})

		if err == nil {
			pollSuccess = true
			continueLooping = false

			glog.V(90).Infof("Found %d pods matching label %q ",
				len(oldPods), accelparams.ContainerLabelsStr)
		} else {
			time.Sleep(pollIntervalSecs)

			totalPollTime += pollIntervalSecs
			if totalPollTime > retryDurationSecs {
				continueLooping = false
			}
		}
	}

	if !pollSuccess {
		glog.V(90).Infof("Error listing pods in %q namespace",
			accelparams.TestNamespaceName)

		return err
	}

	if len(oldPods) == 0 {
		glog.V(90).Infof("No pods matching label %q found in %q namespace",
			accelparams.ContainerLabelsStr, accelparams.TestNamespaceName)
	}

	for _, _pod := range oldPods {
		glog.V(90).Infof("Deleting pod %q in %q namspace",
			_pod.Definition.Name, _pod.Definition.Namespace)

		_pod, err = _pod.DeleteAndWait(300 * time.Second)
		if err != nil {
			glog.V(90).Infof("Failed to delete pod %q: %v",
				_pod.Definition.Name, err)

			return err
		}
	}

	return nil
}

// CreateService Create a service for a workload.
// Return nil on success, otherwise return an error.
func CreateService(apiClient *clients.Settings, port int32) (*service.Builder, error) {
	glog.V(90).Infof("Creating Service %q", accelparams.DeploymentName)

	glog.V(90).Infof("Defining ServicePort")

	svcPort, err := service.DefineServicePort(
		accelparams.ServicePort,
		accelparams.ServicePort,
		corev1.Protocol("TCP"))

	if err != nil {
		glog.V(90).Infof("Error defining service port: %v", err)

		return nil, err
	}

	glog.V(90).Infof("Creating Service Builder")

	svcDemo := service.NewBuilder(apiClient,
		accelparams.DeploymentName,
		accelparams.TestNamespaceName,
		accelparams.ContainerLabelsMap,
		*svcPort)

	svcDemo, err = svcDemo.Create()

	if err != nil {
		glog.V(90).Infof("Error creating service: %v", err)

		return nil, err
	}

	glog.V(90).Infof("Created service: %q in %q namespace",
		svcDemo.Definition.Name, svcDemo.Definition.Namespace)

	return svcDemo, nil
}

// DeleteService Deletes a service.
// Return nil on success, otherwise return an error.
func DeleteService(apiClient *clients.Settings) error {
	glog.V(90).Infof("Deleting Service %q in %q namespace",
		accelparams.DeploymentName, accelparams.TestNamespaceName)

	svcDemo, err := service.Pull(apiClient, accelparams.DeploymentName, accelparams.TestNamespaceName)

	if err != nil && svcDemo == nil {
		glog.V(90).Infof("Service %q not found in %q namespace",
			accelparams.DeploymentName, accelparams.TestNamespaceName)

		return err
	}

	err = svcDemo.Delete()
	if err != nil {
		glog.V(90).Infof("Error deleting service: %v", err)

		return err
	}

	glog.V(90).Infof("Deleted service %q in %q namespace",
		accelparams.DeploymentName, accelparams.TestNamespaceName)

	return nil
}

// CreateWorkloadRoute will create a route for the workload service.
func CreateWorkloadRoute(apiClient *clients.Settings) (*route.Builder, error) {
	workloadRoute, err := route.NewBuilder(
		apiClient, accelparams.DeploymentName, accelparams.TestNamespaceName, accelparams.DeploymentName).Create()

	if err != nil {
		glog.V(90).Infof("Error creating route: %v", err)

		return nil, err
	}

	return workloadRoute, err
}

// DeleteNamespace will delete the workload test namespace.
func DeleteNamespace(apiClient *clients.Settings) error {
	glog.V(90).Infof("Deleting namespace %q", accelparams.TestNamespaceName)

	nsDemo, err := namespace.Pull(apiClient, accelparams.TestNamespaceName)

	if err != nil && nsDemo == nil {
		glog.V(90).Infof("Namespace %q not found", accelparams.TestNamespaceName)

		return err
	}

	err = nsDemo.DeleteAndWait(5 * time.Minute)
	if err != nil {
		glog.V(90).Infof("Error deleting namespace: %v", err)

		return err
	}

	glog.V(90).Infof("Deleted namespace %q", accelparams.TestNamespaceName)

	return nil
}
