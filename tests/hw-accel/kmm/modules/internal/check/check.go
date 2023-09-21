package check

import (
	"fmt"
	"strings"
	"time"

	. "github.com/openshift-kni/eco-gotests/tests/internal/inittools"

	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/internal/kmmparams"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/modules/internal/get"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/modules/internal/tsparams"

	"github.com/openshift-kni/eco-goinfra/pkg/pod"

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
func ModuleLoaded(apiClient *clients.Settings, modName string, timeout time.Duration) error {
	modName = strings.Replace(modName, "-", "_", 10)

	return runCommandOnTestPods(apiClient, []string{"lsmod"}, modName, timeout)
}

// Dmesg verifies that dmesg contains message.
func Dmesg(apiClient *clients.Settings, message string, timeout time.Duration) error {
	return runCommandOnTestPods(apiClient, []string{"dmesg"}, message, timeout)
}

// ModuleSigned verifies the module is signed.
func ModuleSigned(apiClient *clients.Settings, modName, message, nsname, image string) error {
	modulePath := fmt.Sprintf("modinfo /opt/lib/modules/*/%s.ko", modName)
	command := []string{"bash", "-c", modulePath}

	kernelVersion, err := get.KernelFullVersion(apiClient, GeneralConfig.WorkerLabelMap)
	if err != nil {
		return err
	}

	processedImage := strings.ReplaceAll(image, "$KERNEL_FULL_VERSION", kernelVersion)
	testPod := pod.NewBuilder(apiClient, "image-checker", nsname, processedImage)
	_, err = testPod.CreateAndWaitUntilRunning(2 * time.Minute)

	if err != nil {
		glog.V(kmmparams.KmmLogLevel).Infof("Could not create signing verification pod. Got error : %v", err)

		return err
	}

	glog.V(kmmparams.KmmLogLevel).Infof("\n\nPodName: %v\n\n", testPod.Object.Name)

	buff, err := testPod.ExecCommand(command, "test")

	if err != nil {
		return err
	}

	_, _ = testPod.Delete()

	contents := buff.String()
	glog.V(kmmparams.KmmLogLevel).Infof("%s contents: \n \t%v\n", command, contents)

	if strings.Contains(contents, message) {
		glog.V(kmmparams.KmmLogLevel).Infof("command '%s' output contains '%s'\n", command, message)

		return nil
	}

	err = fmt.Errorf("could not find signature in module")

	return err
}

func runCommandOnTestPods(apiClient *clients.Settings,
	command []string, message string, timeout time.Duration) error {
	return wait.PollImmediate(time.Second, timeout, func() (bool, error) {
		pods, err := pod.List(apiClient, tsparams.KmmOperatorNamespace, v1.ListOptions{
			FieldSelector: "status.phase=Running",
			LabelSelector: tsparams.KmmTestHelperLabelName,
		})

		if err != nil {
			glog.V(kmmparams.KmmLogLevel).Infof("deployment list error: %s\n", err)

			return false, err
		}

		// using a map so that both ModuleLoaded and Dmesg calls don't interfere with the counter
		iter := 0
		for _, iterPod := range pods {
			glog.V(kmmparams.KmmLogLevel).Infof("\n\nPodName: %v\n\n", iterPod.Object.Name)

			buff, err := iterPod.ExecCommand(command, "test")

			if err != nil {
				return false, err
			}

			contents := buff.String()
			glog.V(kmmparams.KmmLogLevel).Infof("%s contents: \n \t%v\n", command, contents)
			if strings.Contains(contents, message) {
				glog.V(kmmparams.KmmLogLevel).Infof("command '%s' contains '%s' in pod %s\n", command, message, iterPod.Object.Name)
				iter++

				if iter == len(pods) {
					return true, nil
				}
			}
		}

		return false, err
	})
}
