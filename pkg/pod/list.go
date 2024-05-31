package pod

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-gotests/pkg/clients"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// List returns pod inventory in the given namespace.
func List(apiClient *clients.Settings, nsname string, options v1.ListOptions) ([]*Builder, error) {
	glog.V(100).Infof("Listing pods in the nsname %s with the options %v", nsname, options)

	if nsname == "" {
		glog.V(100).Infof("pod 'nsname' parameter can not be empty")

		return nil, fmt.Errorf("failed to list pods, 'nsname' parameter is empty")
	}

	podList, err := apiClient.Pods(nsname).List(context.Background(), options)

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
func ListInAllNamespaces(apiClient *clients.Settings, options v1.ListOptions) ([]*Builder, error) {
	glog.V(100).Infof("Listing all pods with the options %v", options)

	podList, err := apiClient.Pods("").List(context.Background(), options)

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
