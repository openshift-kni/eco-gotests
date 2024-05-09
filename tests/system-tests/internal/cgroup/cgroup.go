package cgroup

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/clusteroperator"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/nodesconfig"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/ocpcli"
	configv1 "github.com/openshift/api/config/v1"
)

var nodesConfigResourceName = "cluster"

// GetNodeLinuxCGroupVersion returns node Linux cgroup version value.
func GetNodeLinuxCGroupVersion(apiClient *clients.Settings, nodeName string) (configv1.CgroupMode, error) {
	node, err := nodes.Pull(apiClient, nodeName)

	if err != nil {
		glog.V(100).Infof("Failed to get node %s resource due to %v",
			nodeName, err.Error())

		return "", err
	}

	cmdToExecute := "stat -c %T -f /sys/fs/cgroup"

	output, err := ocpcli.ExecuteViaDebugPodOnNode(node.Object.Name, cmdToExecute)

	if err != nil {
		glog.V(100).Infof("Failed to execute command %s on the node %s due to %v",
			cmdToExecute, nodeName, err.Error())

		return "", err
	}

	if output == "" {
		glog.V(100).Infof("cgroup configuration not found for the node %s", nodeName)

		return "", fmt.Errorf("cgroup configuration not found for the node %s", nodeName)
	}

	output = strings.TrimSuffix(output, "\n")

	var result configv1.CgroupMode

	switch output {
	case "cgroup2fs":
		result = configv1.CgroupModeV2
	case "tmpfs":
		result = configv1.CgroupModeV1
	default:
		return "", fmt.Errorf("failed to parse response received from the node %s: '%v'", nodeName, output)
	}

	return result, nil
}

// SetLinuxCGroupVersion returns true when succeeded to set linux cgroup mode to the cluster.
//
//nolint:funlen
func SetLinuxCGroupVersion(apiClient *clients.Settings, expectedCGroupMode configv1.CgroupMode) error {
	nodesConfigObj, err := nodesconfig.Pull(apiClient, nodesConfigResourceName)

	if err != nil {
		glog.V(100).Infof("Failed to get nodes.config %s resource due to %v",
			nodesConfigResourceName, err.Error())

		return err
	}

	glog.V(100).Infof("Set cgroup version %s to the nodes.config %s",
		expectedCGroupMode, nodesConfigObj.Definition.Name)

	currentCGroupMode, err := nodesConfigObj.GetCGroupMode()

	if err != nil {
		glog.V(100).Infof("Failed to get current cgroup configuration from the nodes.config %s due to %v",
			nodesConfigObj.Definition.Name, err.Error())

		return err
	}

	if currentCGroupMode == configv1.CgroupModeEmpty {
		nodesList, err := nodes.List(apiClient)

		if err != nil {
			return err
		}

		glog.V(100).Infof("Verify actual cgroup version configured for node %s",
			nodesList[0].Definition.Name)

		currentCGroupMode, err = GetNodeLinuxCGroupVersion(apiClient, nodesList[0].Definition.Name)

		if err != nil {
			return err
		}
	}

	if currentCGroupMode != expectedCGroupMode {
		glog.V(100).Infof("The current cluster cgroup version is %v; it needs to be changed to the %v",
			currentCGroupMode, expectedCGroupMode)

		nodesConfigObj, err := nodesConfigObj.WithCGroupMode(expectedCGroupMode).Update()

		if err != nil {
			glog.V(100).Infof("Failed to make change to the nodesConfig cgroup due to %v",
				err)

			return err
		}

		newCGroupMode, err := nodesConfigObj.GetCGroupMode()

		if err != nil {
			glog.V(100).Infof("Failed to get current cgroup configuration from the nodes.config %s due to %v",
				nodesConfigObj.Definition.Name, err.Error())

			return err
		}

		if newCGroupMode != expectedCGroupMode {
			return fmt.Errorf("failed to change cluster linux cgroup mode; expected %v, current value is %v",
				expectedCGroupMode, newCGroupMode)
		}

		_, err = nodes.WaitForAllNodesToReboot(apiClient, 30*time.Minute)

		if err != nil {
			glog.V(100).Infof("Nodes failed to reboot after setting new cgroup mode %v config; %v",
				expectedCGroupMode, err)

			return fmt.Errorf("nodes failed to reboot after setting  a new cgroup mode %v config; %w",
				expectedCGroupMode, err)
		}

		_, err = clusteroperator.WaitForAllClusteroperatorsAvailable(apiClient, 5*time.Minute)

		if err != nil {
			return fmt.Errorf("not all clusteroperators are available; %w", err)
		}

		allNodesList, err := nodes.List(apiClient)

		if err != nil {
			glog.V(100).Infof("Failed to get cluster nodes list due to: %v", err)

			return err
		}

		for _, node := range allNodesList {
			currentNodeCGroup, err := GetNodeLinuxCGroupVersion(apiClient, node.Definition.Name)

			if err != nil {
				glog.V(100).Infof("Failed to get cGroup version from the node %s due to: %v",
					node.Definition.Name, err)

				return err
			}

			if currentNodeCGroup != expectedCGroupMode {
				return fmt.Errorf("failed to change cgroup value for the node %s, expected %v, found configured %v",
					node.Definition.Name, expectedCGroupMode, currentNodeCGroup)
			}
		}
	}

	return nil
}
