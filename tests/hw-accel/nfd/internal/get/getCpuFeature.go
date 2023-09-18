package get

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"

	"github.com/openshift-kni/eco-gotests/tests/hw-accel/nfd/internal/nfdhelpersparams"
	v1 "k8s.io/api/core/v1"
)

// CPUInfo retrieves cpu info of worker node.
func CPUInfo(apiClient *clients.Settings, name, nsname, containerName, image string) string {
	podWorker := pod.NewBuilder(apiClient, name, nsname, image)
	containerBuilder := pod.NewContainerBuilder(containerName, image, []string{"./cpucheck"})
	container, err := containerBuilder.GetContainerCfg()

	if err != nil {
		fmt.Println(err)
		glog.V(100).Infof("Failed to define the default container settings %v", err)
	}

	podWorker.Definition.Spec.Containers = make([]v1.Container, 0)
	podWorker.Definition.Spec.Containers = append(podWorker.Definition.Spec.Containers, *container)

	_, err = podWorker.CreateAndWaitUntilRunning(5 * time.Minute)
	if err != nil {
		glog.V(100).Infof(
			"Error creating pod: %s", err.Error())

		return ""
	}

	con := v1.PodLogOptions{
		Container: containerName,
	}

	podLogs := apiClient.Pods(nsname).GetLogs(podWorker.Definition.Name, &con).Do(context.Background())

	err = podLogs.Error()
	if err != nil {
		glog.V(100).Infof(
			"Failed to retrieve logs: %s", err.Error())

		return ""
	}

	body, err := podLogs.Raw()
	if err != nil {
		glog.V(100).Infof("Failed to retrieve logs %v", err)
	}

	return fmt.Sprint(string(body))
}

// CPUFlags returns cpu flags list.
func CPUFlags(apiClient *clients.Settings, nsName string) []string {
	s := CPUInfo(apiClient,
		nfdhelpersparams.PodName,
		nsName,
		nfdhelpersparams.ContainerName,
		nfdhelpersparams.CPUImage)
	arr := strings.Split(s, "\n")

	for _, t := range arr {
		if strings.Contains(t, "Features") {
			flags := strings.Split(strings.ReplaceAll(t, "Features: ", ""), ",")

			return flags
		}
	}

	return []string{}
}
