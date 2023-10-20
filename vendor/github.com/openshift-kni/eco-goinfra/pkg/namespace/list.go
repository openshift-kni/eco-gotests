package namespace

import (
	"context"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// List returns namespace inventory.
func List(apiClient *clients.Settings, options v1.ListOptions) ([]*Builder, error) {
	glog.V(100).Infof("Listing all namespace resources with the options %v", options)

	namespacesList, err := apiClient.CoreV1Interface.Namespaces().List(context.Background(), options)
	if err != nil {
		glog.V(100).Infof("Failed to list namespaces due to %s", err.Error())

		return nil, err
	}

	var namespaceObjects []*Builder

	for _, runningNamespace := range namespacesList.Items {
		copiedNamespace := runningNamespace
		namespaceBuilder := &Builder{
			apiClient:  apiClient,
			Object:     &copiedNamespace,
			Definition: &copiedNamespace,
		}

		namespaceObjects = append(namespaceObjects, namespaceBuilder)
	}

	return namespaceObjects, nil
}
