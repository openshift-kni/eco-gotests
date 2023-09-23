package cluster

import (
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/golang/glog"

	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	. "github.com/openshift-kni/eco-gotests/tests/internal/inittools"
)

// PullTestImageOnNodes pulls given image on range of relevant nodes based on nodeSelector.
func PullTestImageOnNodes(apiClient *clients.Settings, nodeSelector, image string, pullTimeout int) error {
	glog.V(90).Infof("Pulling image %s to nodes with the following label %v", image, nodeSelector)

	nodesList, err := nodes.List(
		apiClient,
		metav1.ListOptions{LabelSelector: labels.Set(map[string]string{nodeSelector: ""}).String()},
	)

	if err != nil {
		return err
	}

	for _, node := range nodesList {
		glog.V(90).Infof("Pulling image %s to node %s", image, node.Object.Name)
		podBuilder := pod.NewBuilder(
			apiClient, fmt.Sprintf("pullpod-%s", node.Object.Name), "default", image)
		err := podBuilder.PullImage(time.Duration(pullTimeout)*time.Second, []string{
			"/bin/sh", "-c", "echo image Pulled && exit 0"})

		if err != nil {
			return err
		}
	}

	return nil
}

// ExecCmd runc cmd on all nodes that match nodeSelector.
func ExecCmd(apiClient *clients.Settings, nodeSelector string, shellCmd string) error {
	glog.V(90).Infof("Executing cmd: %v on nodes based on label: %v using mcp pods", shellCmd, nodeSelector)

	nodeList, err := nodes.List(
		apiClient,
		metav1.ListOptions{LabelSelector: labels.Set(map[string]string{nodeSelector: ""}).String()},
	)
	if err != nil {
		return err
	}

	for _, node := range nodeList {
		listOptions := metav1.ListOptions{
			FieldSelector: fields.SelectorFromSet(fields.Set{"spec.nodeName": node.Definition.Name}).String(),
			LabelSelector: labels.SelectorFromSet(labels.Set{"k8s-app": GeneralConfig.MCOConfigDaemonName}).String(),
		}

		mcPodList, err := pod.List(apiClient, GeneralConfig.MCONamespace, listOptions)
		if err != nil {
			return err
		}

		for _, mcPod := range mcPodList {
			err = mcPod.WaitUntilRunning(300 * time.Second)
			if err != nil {
				return err
			}

			cmdToExec := []string{"sh", "-c", fmt.Sprintf("nsenter --mount=/proc/1/ns/mnt -- sh -c '%s'", shellCmd)}

			glog.V(90).Infof("Exec cmd %v on pod %s", cmdToExec, mcPod.Definition.Name)
			buf, err := mcPod.ExecCommand(cmdToExec)

			if err != nil {
				return fmt.Errorf("%w\n%s", err, buf.String())
			}
		}
	}

	return nil
}
