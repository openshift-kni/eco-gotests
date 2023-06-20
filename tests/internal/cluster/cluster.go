package cluster

import (
	"fmt"
	"time"

	"github.com/golang/glog"

	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
)

// PullTestImageOnNodes pulls given image on range of relevant nodes based on nodeSelector.
func PullTestImageOnNodes(apiClient *clients.Settings, nodeSelector, image string, pullTimeout int) error {
	glog.V(90).Infof("Pulling image %s to nodes with the following label %v", image, nodeSelector)

	nodesList := nodes.NewBuilder(apiClient, map[string]string{nodeSelector: ""})
	err := nodesList.Discover()

	if err != nil {
		return err
	}

	for _, node := range nodesList.Objects {
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
