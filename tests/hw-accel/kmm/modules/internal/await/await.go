package await

import (
	"fmt"
	"strings"
	"time"

	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/internal/kmmparams"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/modules/internal/get"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

var buildPod = make(map[string]string)

// BuildPodCompleted awaits kmm build pods to finish build.
func BuildPodCompleted(apiClient *clients.Settings, nsname string, timeout time.Duration) error {
	return wait.PollImmediate(5*time.Second, timeout, func() (bool, error) {
		var err error

		if buildPod[nsname] == "" {
			pods, err := pod.List(apiClient, nsname, v1.ListOptions{FieldSelector: "status.phase=Running"})

			if err != nil {
				glog.V(kmmparams.KmmLogLevel).Infof("build list error: %s", err)
			}

			for _, podObj := range pods {
				if strings.Contains(podObj.Object.Name, "-build") {
					buildPod[nsname] = podObj.Object.Name
					glog.V(kmmparams.KmmLogLevel).Infof("\rBuild podObj '%s' is Running\n", podObj.Object.Name)

				}
			}
		}

		if buildPod[nsname] != "" {
			fieldSelector := fmt.Sprintf("metadata.name=%s", buildPod[nsname])
			pods, _ := pod.List(apiClient, nsname, v1.ListOptions{FieldSelector: fieldSelector})
			if len(pods) == 0 {
				glog.V(kmmparams.KmmLogLevel).Infof("BuildPod %s no longer in namespace", buildPod)
				buildPod[nsname] = ""

				return true, nil
			}
		}

		return false, err
	})
}

// ModuleDeployment awaits module to de deployed.
func ModuleDeployment(apiClient *clients.Settings,
	nsname string, timeout time.Duration, selector map[string]string) error {
	return wait.PollImmediate(time.Second, timeout, func() (bool, error) {
		pods, err := pod.List(apiClient, nsname, v1.ListOptions{FieldSelector: "status.phase=Running"})

		if err != nil {
			glog.V(kmmparams.KmmLogLevel).Infof("deployment list error: %s", err)

			return false, err
		}

		nodes, err := get.NumberOfNodesForSelector(apiClient, selector)

		if err != nil {
			glog.V(kmmparams.KmmLogLevel).Infof("nodes list error: %s", err)

			return false, err
		}

		glog.V(kmmparams.KmmLogLevel).Infof("Number of nodes: %v, Number of 'Running' pods: %v\n", nodes, len(pods))
		if nodes == len(pods) {
			return true, nil
		}

		return true, err
	})
}

// ModuleUndeployed awaits module pods to be undeployed.
func ModuleUndeployed(apiClient *clients.Settings, nsName string, timeout time.Duration) error {
	return wait.PollImmediate(time.Second, timeout, func() (bool, error) {
		pods, err := pod.List(apiClient, nsName, v1.ListOptions{})

		if err != nil {
			glog.V(kmmparams.KmmLogLevel).Infof("pod list error: %s\n", err)

			return false, err
		}

		glog.V(kmmparams.KmmLogLevel).Infof("current number of pods: %v\n", len(pods))

		return len(pods) == 0, nil
	})
}
