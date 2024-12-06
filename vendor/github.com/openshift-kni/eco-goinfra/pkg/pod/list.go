package pod

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// List returns pod inventory in the given namespace.
func List(apiClient *clients.Settings, nsname string, options ...metav1.ListOptions) ([]*Builder, error) {
	if apiClient == nil {
		glog.V(100).Infof("The apiClient is empty")

		return nil, fmt.Errorf("podList 'apiClient' cannot be empty")
	}

	if nsname == "" {
		glog.V(100).Infof("pod 'nsname' parameter can not be empty")

		return nil, fmt.Errorf("failed to list pods, 'nsname' parameter is empty")
	}

	logMessage := fmt.Sprintf("Listing pods in the nsname %s", nsname)
	passedOptions := metav1.ListOptions{}

	if len(options) == 1 {
		passedOptions = options[0]
		logMessage += fmt.Sprintf(" with the options %v", passedOptions)
	} else if len(options) > 1 {
		glog.V(100).Infof("'options' parameter must be empty or single-valued")

		return nil, fmt.Errorf("error: more than one ListOptions was passed")
	}

	glog.V(100).Infof(logMessage)

	podList, err := apiClient.Pods(nsname).List(context.TODO(), passedOptions)

	if err != nil {
		glog.V(100).Infof("Failed to list pods in the nsname %s due to %s", nsname, err.Error())

		return nil, err
	}

	var podObjects []*Builder

	for _, runningPod := range podList.Items {
		copiedPod := runningPod
		podBuilder := &Builder{
			apiClient:  apiClient,
			Object:     &copiedPod,
			Definition: &copiedPod,
		}

		podObjects = append(podObjects, podBuilder)
	}

	return podObjects, nil
}

// ListInAllNamespaces returns a cluster-wide pod inventory.
func ListInAllNamespaces(apiClient *clients.Settings, options ...metav1.ListOptions) ([]*Builder, error) {
	logMessage := "Listing all pods in all namespaces"
	passedOptions := metav1.ListOptions{}

	if apiClient == nil {
		glog.V(100).Infof("The apiClient is empty")

		return nil, fmt.Errorf("podList 'apiClient' cannot be empty")
	}

	if len(options) > 1 {
		glog.V(100).Infof("'options' parameter must be empty or single-valued")

		return nil, fmt.Errorf("error: more than one ListOptions was passed")
	}

	if len(options) == 1 {
		passedOptions = options[0]
		logMessage += fmt.Sprintf(" with the options %v", passedOptions)
	}

	glog.V(100).Infof(logMessage)

	podList, err := apiClient.Pods("").List(context.TODO(), passedOptions)

	if err != nil {
		glog.V(100).Infof("Failed to list all pods due to %s", err.Error())

		return nil, err
	}

	var podObjects []*Builder

	for _, runningPod := range podList.Items {
		copiedPod := runningPod
		podBuilder := &Builder{
			apiClient:  apiClient,
			Object:     &copiedPod,
			Definition: &copiedPod,
		}

		podObjects = append(podObjects, podBuilder)
	}

	return podObjects, nil
}

// ListByNamePattern returns pod inventory in the given namespace filtered by name pattern.
func ListByNamePattern(apiClient *clients.Settings, namePattern, nsname string) ([]*Builder, error) {
	glog.V(100).Infof("Listing pods in the nsname %s filtered by the name pattern %s", nsname, namePattern)

	if apiClient == nil {
		glog.V(100).Infof("The apiClient is empty")

		return nil, fmt.Errorf("podList 'apiClient' cannot be empty")
	}

	if nsname == "" {
		glog.V(100).Infof("pod 'nsname' parameter can not be empty")

		return nil, fmt.Errorf("failed to list pods, 'nsname' parameter is empty")
	}

	podList, err := apiClient.Pods(nsname).List(context.TODO(), metav1.ListOptions{})

	if err != nil {
		glog.V(100).Infof("Failed to list pods filtered by the name pattern %s in the nsname %s due to %s",
			namePattern, nsname, err.Error())

		return nil, err
	}

	var podObjects []*Builder

	for _, runningPod := range podList.Items {
		if strings.Contains(runningPod.Name, namePattern) {
			copiedPod := runningPod
			podBuilder := &Builder{
				apiClient:  apiClient,
				Object:     &copiedPod,
				Definition: &copiedPod,
			}

			podObjects = append(podObjects, podBuilder)
		}
	}

	return podObjects, nil
}

// WaitForAllPodsInNamespaceRunning wait until all pods in namespace that match options are in running state.
func WaitForAllPodsInNamespaceRunning(
	apiClient *clients.Settings,
	nsname string,
	timeout time.Duration,
	options ...metav1.ListOptions) (bool, error) {
	if apiClient == nil {
		glog.V(100).Infof("The apiClient is empty")

		return false, fmt.Errorf("podList 'apiClient' cannot be empty")
	}

	if nsname == "" {
		glog.V(100).Infof("'nsname' parameter can not be empty")

		return false, fmt.Errorf("failed to list pods, 'nsname' parameter is empty")
	}

	logMessage := fmt.Sprintf("Waiting for all pods in %s namespace", nsname)
	passedOptions := metav1.ListOptions{}

	if len(options) > 1 {
		glog.V(100).Infof("'options' parameter must be empty or single-valued")

		return false, fmt.Errorf("error: more than one ListOptions was passed")
	}

	if len(options) == 1 {
		passedOptions = options[0]
		logMessage += fmt.Sprintf(" with the options %v", passedOptions)
	}

	glog.V(100).Infof(logMessage + " are in running state")

	podList, err := List(apiClient, nsname, passedOptions)
	if err != nil {
		glog.V(100).Infof("Failed to list all pods due to %s", err.Error())

		return false, err
	}

	for _, podObj := range podList {
		err = podObj.WaitUntilRunning(timeout)
		if err != nil {
			glog.V(100).Infof("Timout was reached while waiting for all pods in running state: %s", err.Error())

			return false, err
		}
	}

	return true, nil
}

// WaitForAllPodsInNamespacesHealthy waits until:
// - all pods in a list of namespaces that match options are in healthy state.
// - a pod in a healthy state is in running phase and optionally in ready condition.
//
// nsNames passes the list of namespaces to monitor. Monitors all namespaces when empty.
// timeout is the duration in seconds to wait for the pods to be healthy
// includeSucceeded when true, considers that pods in succeeded phase are healthy.
// skipReadiness when false, checks that the podCondition is ready.
// ignoreRestartPolicyNever when true, ignores failed pods with restart policy set to never.
// ignoreNamespaces is a list of namespaces to ignore.
// options reduces the list of namespace to only the ones matching options.
func WaitForAllPodsInNamespacesHealthy(
	apiClient *clients.Settings,
	nsNames []string,
	timeout time.Duration,
	includeSucceeded bool,
	skipReadinessCheck bool,
	ignoreRestartPolicyNever bool,
	ignoreNamespaces []string,
	options ...metav1.ListOptions,
) error {
	logMessage := fmt.Sprintf("Waiting for all pods in %v namespaces", nsNames)
	passedOptions := metav1.ListOptions{}

	if apiClient == nil {
		glog.V(100).Infof("The apiClient is empty")

		return fmt.Errorf("podList 'apiClient' cannot be empty")
	}

	if len(options) == 1 {
		passedOptions = options[0]
		logMessage += fmt.Sprintf(" with the options %v", passedOptions)
	}

	glog.V(100).Infof(logMessage + " are in running state")

	var podList []*Builder

	if len(nsNames) == 0 {
		var err error
		podList, err = ListInAllNamespaces(apiClient, passedOptions)

		if err != nil {
			glog.V(100).Infof("Failed to list all pods due to %s", err.Error())

			return err
		}
	} else {
		for _, ns := range nsNames {
			podListForNs, err := List(apiClient, ns, passedOptions)
			if err != nil {
				glog.V(100).Infof("Failed to list all pods due to %s", err.Error())

				return err
			}
			podList = append(podList, podListForNs...)
		}
	}

	for _, podObj := range podList {
		if slices.Contains(ignoreNamespaces, podObj.Definition.Namespace) {
			continue
		}

		err := podObj.WaitUntilHealthy(timeout, includeSucceeded, skipReadinessCheck, ignoreRestartPolicyNever)
		if k8serrors.IsNotFound(err) {
			glog.V(100).Infof("Pod %s in namespace %s no longer exists, skipping",
				podObj.Definition.Name, podObj.Definition.Namespace)

			continue
		}

		if err != nil {
			glog.V(100).Infof("Failed to wait for all pods to be healthy due to %s", err.Error())

			return err
		}
	}

	return nil
}
