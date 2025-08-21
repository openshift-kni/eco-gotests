package helpers

import (
	"fmt"
	"strings"

	"github.com/golang/glog"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/clients"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/hw-accel/internal/hwaccelparams"
	ts "github.com/rh-ecosystem-edge/eco-gotests/tests/hw-accel/nfd/features/internal/tsparams"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/hw-accel/nfd/internal/get"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/hw-accel/nfd/internal/search"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/hw-accel/nfd/internal/wait"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/hw-accel/nfd/nfdparams"
	"k8s.io/client-go/util/retry"
)

// CheckLabelsExist check if each node contains the required label.
func CheckLabelsExist(nodelabels map[string][]string, labelsToSearch, blackList []string, nodeName string) error {
	labelList := ts.DefaultBlackList

	if blackList != nil || len(blackList) != 0 {
		labelList = blackList
	}

	allFeatures := strings.Join(nodelabels[nodeName], ",")
	if len(allFeatures) == 0 {
		return fmt.Errorf("node feature labels should be greater than zero")
	}

	for _, featurelabel := range labelsToSearch {
		if search.StringInSlice(featurelabel, labelList) {
			continue
		}

		if !strings.Contains(allFeatures, fmt.Sprintf("%s=", featurelabel)) {
			return fmt.Errorf("label %s not found in node %s", featurelabel, nodeName)
		}
	}

	return nil
}

// CheckPodStatus check if each pod is in a running status.
func CheckPodStatus(apiClient *clients.Settings) error {
	verifyPodStatus := func() error {
		_, err := wait.ForPod(apiClient, hwaccelparams.NFDNamespace)
		if err != nil {
			return err
		}

		glog.V(nfdparams.LogLevel).Info("validate all pods are running")

		podlist, err := get.PodStatus(apiClient, hwaccelparams.NFDNamespace)
		if err != nil {
			return err
		}

		for _, pod := range podlist {
			if pod.State != "Running" {
				return fmt.Errorf("pod: %v is in %v status", pod.Name, pod.State)
			}
		}

		glog.V(nfdparams.LogLevel).Info("all pods are in running status")

		return nil
	}

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		return verifyPodStatus()
	})

	return err
}
