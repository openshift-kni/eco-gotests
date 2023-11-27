package get

import (
	"fmt"
	"strings"

	"github.com/golang/glog"

	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/nfd/internal/nfdhelpersparams"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/nfd/nfdparams"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PodState an object that describe the name and state of a pod.
type PodState struct {
	Name  string
	State string
}

// PodStatus return a list pod and state.
func PodStatus(apiClient *clients.Settings, nsname string) ([]PodState, error) {
	podList, err := pod.List(apiClient, nsname, v1.ListOptions{})
	if err != nil {
		return nil, err
	}

	nfdReources := NfdResourceCount(apiClient)
	podStateList := make([]PodState, 0)

	for _, onePod := range podList {
		state := onePod.Object.Status.Phase

		glog.V(nfdparams.LogLevel).Infof("%s is in %s status", onePod.Object.Name, state)

		for _, nfdPodName := range nfdhelpersparams.ValidPodNameList {
			if strings.Contains(onePod.Object.Name, nfdPodName) {
				nfdReources[nfdPodName]--
				podStateList = append(podStateList, PodState{Name: onePod.Object.Name, State: string(state)})
			}
		}
	}

	for resourceName, resourceCount := range nfdReources {
		if resourceCount > 0 {
			return nil, fmt.Errorf("%s is equal to %d it should be 0", resourceName, resourceCount)
		}
	}

	return podStateList, nil
}
