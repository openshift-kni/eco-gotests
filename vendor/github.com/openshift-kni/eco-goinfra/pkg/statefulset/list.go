package statefulset

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// List returns statefulset inventory in the given namespace.
func List(apiClient *clients.Settings, nsname string, options metaV1.ListOptions) ([]*Builder, error) {
	glog.V(100).Infof("Listing statefulsets in the namespace %s with the options %v", nsname, options)

	if nsname == "" {
		glog.V(100).Infof("statefulset 'nsname' parameter can not be empty")

		return nil, fmt.Errorf("failed to list statefulsets, 'nsname' parameter is empty")
	}

	statefulsetList, err := apiClient.StatefulSets(nsname).List(context.Background(), options)

	if err != nil {
		glog.V(100).Infof("Failed to list statefulsets in the namespace %s due to %s", nsname, err.Error())

		return nil, err
	}

	var statefulsetObjects []*Builder

	for _, runningStatefulSet := range statefulsetList.Items {
		copiedStatefulSet := runningStatefulSet
		statefulsetBuilder := &Builder{
			apiClient:  apiClient,
			Object:     &copiedStatefulSet,
			Definition: &copiedStatefulSet,
		}

		statefulsetObjects = append(statefulsetObjects, statefulsetBuilder)
	}

	return statefulsetObjects, nil
}
