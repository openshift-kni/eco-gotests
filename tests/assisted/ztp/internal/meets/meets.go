package meets

import (
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/go-version"
	"github.com/openshift-kni/eco-gotests/pkg/clients"
	"github.com/openshift-kni/eco-gotests/pkg/hive"
	"github.com/openshift-kni/eco-gotests/pkg/olm"
	"github.com/openshift-kni/eco-gotests/pkg/pod"
	. "github.com/openshift-kni/eco-gotests/tests/assisted/ztp/internal/ztpinittools"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// HubInfrastructureOperatorRunningRequirement ensures that
// the infrastructure operator pod is running on the hub cluster.
func HubInfrastructureOperatorRunningRequirement() (bool, string) {
	return checkPodRunning(HubAPIClient, "infrastructure-operator", "control-plane=infrastructure-operator")
}

// HubAssistedServiceRunningRequirement ensures that
// the assisted-service pod is running on the hub cluster.
func HubAssistedServiceRunningRequirement() (bool, string) {
	return checkPodRunning(HubAPIClient, "assisted-service", "app=assisted-service")
}

// HubAssistedImageServiceRunningRequirement ensures that
// the assisted-image-service pod is running on the hub cluster.
func HubAssistedImageServiceRunningRequirement() (bool, string) {
	return checkPodRunning(HubAPIClient, "assisted-image-service", "app=assisted-image-service")
}

// HubInfrastructureOperandRunningRequirement ensures that both
// the assisted-service and assisted-image-service pods are running on the hub cluster.
func HubInfrastructureOperandRunningRequirement() (bool, string) {
	serviceRunning, msg := HubAssistedServiceRunningRequirement()
	if !serviceRunning {
		return serviceRunning, msg
	}

	return HubAssistedImageServiceRunningRequirement()
}

// HubOperatorVersionRequirement checks that hub operator version meets the version provided.
func HubOperatorVersionRequirement(requiredVersion string) (bool, string) {
	operator, err := getOperator(HubAPIClient)
	if err != nil {
		return false, fmt.Sprintf("Failed to get operator from hub cluster: %v", err)
	}

	var hubOperatorVersion *version.Version

	switch {
	case strings.Contains(operator.Object.Name, "multicluster-engine"):
		hubOperatorVersion, _ = version.NewVersion(fmt.Sprintf("%d.%d",
			operator.Object.Spec.Version.Major, operator.Object.Spec.Version.Minor))

	case strings.Contains(operator.Object.Name, "assisted-installer"):
		hubOperatorVersion, _ = version.NewVersion(requiredVersion)

	default:
		return false, "could not find expected csv providing assisted resources"
	}

	currentVersion, _ := version.NewVersion(requiredVersion)

	if hubOperatorVersion.LessThan(currentVersion) {
		return false, fmt.Sprintf("Discovered hub operator version does not meet requirement: %s",
			hubOperatorVersion)
	}

	return true, ""
}

// SpokeClusterImageSetVersionRequirement checks that the provided clusterimageset meets the version provided.
func SpokeClusterImageSetVersionRequirement(requiredVersion string) (bool, string) {
	if SpokeConfig.ClusterImageSet == "" {
		return false, "Spoke clusterimageset version was not provided through environment"
	}

	_, err := hive.PullClusterImageSet(HubAPIClient, SpokeConfig.ClusterImageSet)
	if err != nil {
		return false, fmt.Sprintf("ClusterImageSet could not be found: %v", err)
	}

	imgSetVersion, _ := version.NewVersion(SpokeConfig.ClusterImageSet)
	currentVersion, _ := version.NewVersion(requiredVersion)

	if imgSetVersion.LessThan(currentVersion) {
		return false, fmt.Sprintf("Discovered clusterimageset version does not meet requirement: %v",
			imgSetVersion.String())
	}

	return true, ""
}

// getOperator returns the clusterserviceversion of the operator that installed assisted-service.
func getOperator(apiClient *clients.Settings) (*olm.ClusterServiceVersionBuilder, error) {
	servicePod, err := getAssistedPod(apiClient, "assisted-service", "app=assisted-service")
	if err != nil {
		return nil, err
	}

	csvs, err := olm.ListClusterServiceVersion(apiClient, servicePod.Object.Namespace, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, csv := range csvs {
		if strings.Contains(csv.Object.Name, "multicluster-engine") ||
			strings.Contains(csv.Object.Name, "assisted-installer") {
			return csv, nil
		}
	}

	return nil, fmt.Errorf("could not discover operator clusterserviceversion from cluster")
}

// getAssistedPod returns a podBuilder of a pod based on provided label.
func getAssistedPod(apiClient *clients.Settings, podPrefix, label string) (*pod.Builder, error) {
	if apiClient == nil {
		return nil, fmt.Errorf("apiClient is nil")
	}

	podList, err := pod.ListInAllNamespaces(apiClient, metav1.ListOptions{LabelSelector: label})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods on cluster: %w", err)
	}

	if len(podList) == 0 {
		return nil, fmt.Errorf("%s pod is not currently running", podPrefix)
	}

	if len(podList) > 1 {
		return nil, fmt.Errorf("got unexpected pods when checking for the %s pod", podPrefix)
	}

	return podList[0], nil
}

// checkPodRunning waits for the specified pod to be running.
func checkPodRunning(apiClient *clients.Settings, podPrefix, label string) (bool, string) {
	assistedPod, err := getAssistedPod(apiClient, podPrefix, label)
	if err != nil {
		return false, err.Error()
	}

	err = assistedPod.WaitUntilInStatus(v1.PodRunning, time.Second*10)
	if err != nil {
		return false, fmt.Sprintf("%s pod found but was not running", assistedPod.Definition.Name)
	}

	return true, ""
}
