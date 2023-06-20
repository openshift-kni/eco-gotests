package check

import (
	"fmt"
	"strings"
	"time"

	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/internal/kmmparams"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

// NodeLabel checks if label is present on the node.
func NodeLabel(apiClient *clients.Settings, moduleName string, nodeSelector map[string]string) (bool, error) {
	nodeBuilder := nodes.NewBuilder(apiClient, nodeSelector)

	if err := nodeBuilder.Discover(); err != nil {
		glog.V(kmmparams.KmmLogLevel).Infof("could not discover %v nodes", nodeSelector)
	}

	foundLabels := 0
	label := fmt.Sprintf("kmm.node.kubernetes.io/%s.ready", moduleName)

	for _, node := range nodeBuilder.Objects {
		_, ok := node.Object.Labels[label]
		if ok {
			glog.V(kmmparams.KmmLogLevel).Infof("Found label %v that contains %v on node %v",
				label, moduleName, node.Object.Name)

			foundLabels++
			if foundLabels == len(nodeBuilder.Objects) {
				return true, nil
			}
		}
	}

	err := fmt.Errorf("not all nodes (%v) have the label '%s' ", len(nodeBuilder.Objects), label)

	return false, err
}

// ModuleLoaded verifies the module is loaded on the node.
func ModuleLoaded(apiClient *clients.Settings, modName, nsname string, timeout time.Duration) error {
	return wait.PollImmediate(time.Second, timeout, func() (bool, error) {
		pods, err := pod.List(apiClient, nsname, v1.ListOptions{FieldSelector: "status.phase=Running"})

		if err != nil {
			glog.V(kmmparams.KmmLogLevel).Infof("deployment list error: %s\n", err)

			return false, err
		}

		// injected kmod are always with underline ( _ )
		kmodName := strings.Replace(modName, "-", "_", 10)

		iter := 0
		for _, pod := range pods {
			glog.V(kmmparams.KmmLogLevel).Infof("\n\nPodName: %v\n\n", pod.Object.Name)

			buff, err := pod.ExecCommand([]string{"lsmod"}, "module-loader")

			if err != nil {
				return false, err
			}

			contents := buff.String()
			glog.V(kmmparams.KmmLogLevel).Infof("lsmod contents: \n \t%v\n", contents)
			if strings.Contains(contents, kmodName) {
				glog.V(kmmparams.KmmLogLevel).Infof("found module name loaded on pod : %v\n", pod.Object.Name)
				iter++

				if iter == len(pods)-1 {
					return true, nil
				}
			}
		}

		return false, err
	})
}
