package iperf3workload

import (
	"bytes"
	"slices"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/clients"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/deployment"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/pod"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/service"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/system-tests/ipsec/internal/ipsecparams"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	retryDurationSecs = 360
	pollIntervalSecs  = 20
)

// CreateService Create a service for a workload.
// Return nil on success, otherwise return an error.
func CreateService(apiClient *clients.Settings, nodePort int32) (*service.Builder, error) {
	glog.V(ipsecparams.IpsecLogLevel).Infof("Creating Service %q", ipsecparams.Iperf3DeploymentName)

	glog.V(ipsecparams.IpsecLogLevel).Infof("Defining ServicePort")

	svcPort, err := service.DefineServicePort(
		nodePort,
		nodePort,
		corev1.Protocol("TCP"))

	if err != nil {
		glog.V(ipsecparams.IpsecLogLevel).Infof("Error defining ServicePort: %v", err)

		return nil, err
	}

	glog.V(ipsecparams.IpsecLogLevel).Infof("Creating Service Builder")

	svcDemo := service.NewBuilder(apiClient,
		ipsecparams.Iperf3DeploymentName,
		ipsecparams.TestNamespaceName,
		ipsecparams.ContainerLabelsMap,
		*svcPort)

	glog.V(ipsecparams.IpsecLogLevel).Infof("Setting service type to NodePort")

	svcDemo = svcDemo.WithNodePort()

	glog.V(ipsecparams.IpsecLogLevel).Infof("Resetting NodePort value")

	svcDemo.Definition.Spec.Ports[0].NodePort = nodePort

	svcDemo, err = svcDemo.Create()

	if err != nil {
		glog.V(ipsecparams.IpsecLogLevel).Infof("Error creating service: %v", err)

		return nil, err
	}

	glog.V(ipsecparams.IpsecLogLevel).Infof("Created service: %q in %q namespace",
		svcDemo.Definition.Name, svcDemo.Definition.Namespace)

	return svcDemo, nil
}

// DeleteService Deletes a service.
// Return nil on success, otherwise return an error.
func DeleteService(apiClient *clients.Settings) error {
	glog.V(ipsecparams.IpsecLogLevel).Infof("Deleting Service %q in %q namespace",
		ipsecparams.Iperf3DeploymentName, ipsecparams.TestNamespaceName)

	svcDemo, err := service.Pull(apiClient, ipsecparams.Iperf3DeploymentName, ipsecparams.TestNamespaceName)

	if err != nil && svcDemo == nil {
		glog.V(ipsecparams.IpsecLogLevel).Infof("Service %q not found in %q namespace",
			ipsecparams.Iperf3DeploymentName, ipsecparams.TestNamespaceName)

		return err
	}

	err = svcDemo.Delete()
	if err != nil {
		glog.V(ipsecparams.IpsecLogLevel).Infof("Error deleting service: %v", err)

		return err
	}

	glog.V(ipsecparams.IpsecLogLevel).Infof("Deleted service %q in %q namespace",
		ipsecparams.Iperf3DeploymentName, ipsecparams.TestNamespaceName)

	return nil
}

// CreateWorkload Create a workload with the iperf3 image, the iperf3 command will be
// launched from either LaunchIperf3Client() or LaunchIperf3Server().
// Return nil on success, otherwise return an error.
func CreateWorkload(apiClient *clients.Settings, nodeName string, iperf3ToolImage string) (*deployment.Builder, error) {
	deployContainer := pod.NewContainerBuilder(ipsecparams.Iperf3DeploymentName,
		iperf3ToolImage,
		ipsecparams.ContainerCmdSleep)

	deployContainer = deployContainer.WithSecurityContext(&corev1.SecurityContext{RunAsGroup: nil, RunAsUser: nil})

	deployContainerCfg, err := deployContainer.GetContainerCfg()
	if err != nil {
		glog.V(ipsecparams.IpsecLogLevel).Infof("Error getting container cfg: %s", err)

		return nil, err
	}

	createDeploy := deployment.NewBuilder(apiClient,
		ipsecparams.Iperf3DeploymentName,
		ipsecparams.TestNamespaceName,
		ipsecparams.ContainerLabelsMap,
		deployContainerCfg)
	createDeploy = createDeploy.WithNodeSelector(map[string]string{"kubernetes.io/hostname": nodeName})

	_, err = createDeploy.CreateAndWaitUntilReady(300 * time.Second)
	if err != nil {
		glog.V(ipsecparams.IpsecLogLevel).Infof("Error deploying container: %s", err)

		return nil, err
	}

	return createDeploy, nil
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
		oldPods, err = pod.List(apiClient, ipsecparams.TestNamespaceName,
			metav1.ListOptions{LabelSelector: ipsecparams.ContainerLabelsStr})

		if err == nil {
			pollSuccess = true
			continueLooping = false

			glog.V(ipsecparams.IpsecLogLevel).Infof("Found %d pods matching label %q ",
				len(oldPods), ipsecparams.ContainerLabelsStr)
		} else {
			time.Sleep(pollIntervalSecs)

			totalPollTime += pollIntervalSecs
			if totalPollTime > retryDurationSecs {
				continueLooping = false
			}
		}
	}

	if !pollSuccess {
		glog.V(ipsecparams.IpsecLogLevel).Infof("Error listing pods in %q namespace",
			ipsecparams.TestNamespaceName)

		return err
	}

	if len(oldPods) == 0 {
		glog.V(ipsecparams.IpsecLogLevel).Infof("No pods matching label %q found in %q namespace",
			ipsecparams.ContainerLabelsStr, ipsecparams.TestNamespaceName)
	}

	for _, _pod := range oldPods {
		glog.V(ipsecparams.IpsecLogLevel).Infof("Deleting pod %q in %q namspace",
			_pod.Definition.Name, _pod.Definition.Namespace)

		_pod, err = _pod.DeleteAndWait(300 * time.Second)
		if err != nil {
			glog.V(ipsecparams.IpsecLogLevel).Infof("Failed to delete pod %q: %v",
				_pod.Definition.Name, err)

			return err
		}
	}

	return nil
}

// LaunchIperf3Command launches the iperf3 command in an already running workload
// Return nil on success, otherwise return an error.
func LaunchIperf3Command(apiClient *clients.Settings, iperf3Command []string) bool {
	// deployName       =>  ipsecparams.Iperf3DeploymentName
	// deployNS         =>  ipsecparams.TestNamespaceName
	// deployLabel      =>  ipsecparams.ContainerLabelsStr
	// containerName    =>  ipsecparams.Iperf3DeploymentName
	glog.V(ipsecparams.IpsecLogLevel).Infof("Check deployment %q exists in %q namespace",
		ipsecparams.Iperf3DeploymentName, ipsecparams.TestNamespaceName)

	pullDeploy, _ := deployment.Pull(apiClient, ipsecparams.Iperf3DeploymentName, ipsecparams.TestNamespaceName)

	if pullDeploy == nil {
		glog.V(ipsecparams.IpsecLogLevel).Infof("Deployment %q not found in %q ns",
			ipsecparams.Iperf3DeploymentName, ipsecparams.TestNamespaceName)
	}

	var (
		appPods []*pod.Builder
		err     error
		output  bytes.Buffer
	)

	glog.V(ipsecparams.IpsecLogLevel).Infof("Finding pod backed by deployment")

	totalPollTime := 0
	pollSuccess := false

	continueLooping := true
	for continueLooping {
		appPods, err = pod.List(apiClient,
			ipsecparams.TestNamespaceName,
			metav1.ListOptions{LabelSelector: ipsecparams.ContainerLabelsStr})

		if err == nil {
			pollSuccess = true
			continueLooping = false

			glog.V(ipsecparams.IpsecLogLevel).Infof("Found %d pods matching label %q",
				len(appPods), ipsecparams.ContainerLabelsStr)
		} else {
			time.Sleep(pollIntervalSecs)

			totalPollTime += pollIntervalSecs
			if totalPollTime > retryDurationSecs {
				continueLooping = false
			}
		}
	}

	if !pollSuccess {
		glog.V(ipsecparams.IpsecLogLevel).Infof("Failed to find pods matching label %q",
			ipsecparams.ContainerLabelsStr)

		return false
	}

	for _, _pod := range appPods {
		cmdIperf3 := append(slices.Clone(ipsecparams.ContainerCmdBash), strings.Join(iperf3Command, " "))
		glog.V(ipsecparams.IpsecLogLevel).Infof("Running command %q from within a pod %q",
			cmdIperf3, _pod.Definition.Name)

		output, err = _pod.ExecCommand(cmdIperf3, ipsecparams.Iperf3DeploymentName)

		if err != nil {
			glog.V(ipsecparams.IpsecLogLevel).Infof(
				"Error running iperf3 lookup from within pod, output: [%s], err [%s]",
				output, err)

			return false
		}

		glog.V(ipsecparams.IpsecLogLevel).Infof("Command's Output:\n%v\n", output.String())
	}

	return true
}
