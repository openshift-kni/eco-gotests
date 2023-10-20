package olm

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ListClusterServiceVersion returns clusterserviceversion inventory in the given namespace.
func ListClusterServiceVersion(
	apiClient *clients.Settings,
	nsname string,
	options metaV1.ListOptions) ([]*ClusterServiceVersionBuilder, error) {
	glog.V(100).Infof("Listing clusterserviceversion in the namespace %s with the options %v", nsname, options)

	if nsname == "" {
		glog.V(100).Infof("clusterserviceversion 'nsname' parameter can not be empty")

		return nil, fmt.Errorf("failed to list clusterserviceversion, 'nsname' parameter is empty")
	}

	csvList, err := apiClient.OperatorsV1alpha1Interface.ClusterServiceVersions(nsname).List(context.Background(), options)

	if err != nil {
		glog.V(100).Infof("Failed to list clusterserviceversion in the nsname %s due to %s", nsname, err.Error())

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

// ListClusterServiceVersionWithNamePattern returns a cluster-wide clusterserviceversion inventory
// filtered by the name pattern.
func ListClusterServiceVersionWithNamePattern(
	apiClient *clients.Settings,
	namePattern string,
	nsname string,
	options metaV1.ListOptions) ([]*ClusterServiceVersionBuilder, error) {
	if namePattern == "" {
		glog.V(100).Info(
			"The namePattern field to filter out all relevant clusterserviceversion cannot be empty")

		return nil, fmt.Errorf(
			"the namePattern field to filter out all relevant clusterserviceversion cannot be empty")
	}

	glog.V(100).Infof("Listing clusterserviceversion filtered by the name pattern %s in %s namespace",
		namePattern, nsname)

	notFilteredCsvList, err := ListClusterServiceVersion(apiClient, nsname, options)

	if err != nil {
		glog.V(100).Infof("Failed to list all clusterserviceversions in namespace %s due to %s",
			nsname, err.Error())

		return nil, err
	}

	var finalCsvList []*ClusterServiceVersionBuilder

	for _, foundCsv := range notFilteredCsvList {
		if strings.Contains(foundCsv.Definition.Name, namePattern) {
			finalCsvList = append(finalCsvList, foundCsv)
		}
	}

	return finalCsvList, nil
}
