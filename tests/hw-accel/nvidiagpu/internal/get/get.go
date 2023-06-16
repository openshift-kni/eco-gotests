package get

import (
	"github.com/openshift-kni/eco-gotests/pkg/olm"
	"github.com/openshift-kni/eco-gotests/pkg/pod"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/nvidiagpu/internal/gpuparams"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-gotests/pkg/clients"
)

// InstalledCSVFromSubscription returns installedCSV from Subscription.
func InstalledCSVFromSubscription(apiClient *clients.Settings, gpuSubscriptionName,
	gpuSubscriptionNamespace string) (string, error) {
	subPulled, err := olm.PullSubscription(apiClient, gpuSubscriptionName, gpuSubscriptionNamespace)

	if err != nil {
		glog.V(gpuparams.GpuLogLevel).Infof(
			"error pulling Subscription %s from cluster in namespace %s", gpuSubscriptionName,
			gpuSubscriptionNamespace)

		return "", err
	}

	glog.V(gpuparams.GpuLogLevel).Infof(
		"InstalledCSV %s extracted from Subscription %s from cluster in namespace %s",
		subPulled.Object.Status.InstalledCSV, gpuSubscriptionName, gpuSubscriptionNamespace)

	return subPulled.Object.Status.InstalledCSV, nil
}

// CurrentCSVFromSubscription returns installedCSV from Subscription.
func CurrentCSVFromSubscription(apiClient *clients.Settings, gpuSubscriptionName,
	gpuSubscriptionNamespace string) (string, error) {
	subPulled, err := olm.PullSubscription(apiClient, gpuSubscriptionName, gpuSubscriptionNamespace)

	if err != nil {
		glog.V(gpuparams.GpuLogLevel).Infof(
			"error pulling Subscription %s from cluster in namespace %s", gpuSubscriptionName,
			gpuSubscriptionNamespace)

		return "", err
	}

	glog.V(gpuparams.GpuLogLevel).Infof(
		"CurrentCSV %s extracted from Subscription %s from cluster in namespace %s",
		subPulled.Object.Status.CurrentCSV, gpuSubscriptionName, gpuSubscriptionNamespace)

	return subPulled.Object.Status.CurrentCSV, nil
}

// GetFirstPodNameWithLabel returns a the first pod name matching pod labelSelector in specified namespace.
func GetFirstPodNameWithLabel(apiClient *clients.Settings, podNamespace, podLabelSelector string) (string, error) {
	podList, err := pod.List(apiClient, podNamespace, v1.ListOptions{LabelSelector: podLabelSelector})

	glog.V(gpuparams.GpuLogLevel).Infof("Length of podList matching podLabelSelector is '%v'", len(podList))
	glog.V(gpuparams.GpuLogLevel).Infof("podList[0] matching podLabelSelector is '%v'",
		podList[0].Definition.Name)

	return podList[0].Definition.Name, err
}
