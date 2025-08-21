package cluster

import (
	"fmt"
	"regexp"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/golang/glog"

	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/clients"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/nodes"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/pod"
	. "github.com/rh-ecosystem-edge/eco-gotests/tests/internal/inittools"
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

// ExecCmdWithStdout runs cmd on all selected nodes and returns their stdout.
func ExecCmdWithStdout(
	apiClient *clients.Settings, shellCmd string, options ...metav1.ListOptions) (map[string]string, error) {
	if GeneralConfig.MCOConfigDaemonName == "" {
		return nil, fmt.Errorf("error: mco config daemon pod name cannot be empty")
	}

	if GeneralConfig.MCONamespace == "" {
		return nil, fmt.Errorf("error: mco namespace cannot be empty")
	}

	logMessage := fmt.Sprintf("Executing cmd: %v on nodes", shellCmd)

	passedOptions := metav1.ListOptions{}

	if len(options) > 1 {
		glog.V(90).Infof("'options' parameter must be empty or single-valued")

		return nil, fmt.Errorf("error: more than one ListOptions was passed")
	}

	if len(options) == 1 {
		passedOptions = options[0]
		logMessage += fmt.Sprintf(" with the options %v", passedOptions)
	}

	glog.V(90).Infof(logMessage)

	nodeList, err := nodes.List(
		apiClient,
		passedOptions,
	)

	if err != nil {
		return nil, err
	}

	glog.V(90).Infof("Found %d nodes matching selector", len(nodeList))

	outputMap := make(map[string]string)

	for _, node := range nodeList {
		listOptions := metav1.ListOptions{
			FieldSelector: fields.SelectorFromSet(fields.Set{"spec.nodeName": node.Definition.Name}).String(),
			LabelSelector: labels.SelectorFromSet(labels.Set{"k8s-app": GeneralConfig.MCOConfigDaemonName}).String(),
		}

		mcPodList, err := pod.List(apiClient, GeneralConfig.MCONamespace, listOptions)
		if err != nil {
			return nil, err
		}

		for _, mcPod := range mcPodList {
			err = mcPod.WaitUntilRunning(300 * time.Second)
			if err != nil {
				return nil, err
			}

			hostnameCmd := []string{"sh", "-c", "nsenter --mount=/proc/1/ns/mnt -- sh -c 'printf $(hostname)'"}
			hostnameBuf, err := mcPod.ExecCommand(hostnameCmd)

			if err != nil {
				return nil, fmt.Errorf("failed gathering node hostname: %w", err)
			}

			cmdToExec := []string{"sh", "-c", fmt.Sprintf("nsenter --mount=/proc/1/ns/mnt -- sh -c '%s'", shellCmd)}

			glog.V(90).Infof("Exec cmd %v on pod %s", cmdToExec, mcPod.Definition.Name)
			commandBuf, err := mcPod.ExecCommand(cmdToExec)

			if err != nil {
				return nil, fmt.Errorf("failed executing command '%s' on node %s: %w", shellCmd, hostnameBuf.String(), err)
			}

			hostname := regexp.MustCompile(`\r`).ReplaceAllString(hostnameBuf.String(), "")
			output := regexp.MustCompile(`\r`).ReplaceAllString(commandBuf.String(), "")

			outputMap[hostname] = output
		}
	}

	return outputMap, nil
}
