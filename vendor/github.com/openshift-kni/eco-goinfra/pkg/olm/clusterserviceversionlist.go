package olm

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ListClusterServiceVersion returns clusterserviceversion inventory in the given namespace.
func ListClusterServiceVersion(
	apiClient *clients.Settings,
	nsname string,
	options metaV1.ListOptions) ([]*ClusterServiceVersionBuilder, error) {
	glog.V(100).Infof("Listing clusterserviceversions in the namespace %s with the options %v", nsname, options)

	if nsname == "" {
		glog.V(100).Infof("clusterserviceversion 'nsname' parameter can not be empty")

		return nil, fmt.Errorf("failed to list clusterserviceversions, 'nsname' parameter is empty")
	}

	csvList, err := apiClient.OperatorsV1alpha1Interface.ClusterServiceVersions(nsname).List(context.Background(), options)

	if err != nil {
		glog.V(100).Infof("Failed to list clusterserviceversions in the nsname %s due to %s", nsname, err.Error())

		return nil, err
	}

	var csvObjects []*ClusterServiceVersionBuilder

	for _, runningCSV := range csvList.Items {
		copiedCSV := runningCSV
		csvBuilder := &ClusterServiceVersionBuilder{
			apiClient:  apiClient,
			Object:     &copiedCSV,
			Definition: &copiedCSV,
		}

		csvObjects = append(csvObjects, csvBuilder)
	}

	return csvObjects, nil
}
