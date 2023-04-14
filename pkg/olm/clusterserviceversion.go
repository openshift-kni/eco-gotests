package olm

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-gotests/pkg/clients"
	oplmV1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterServiceVersionBuilder provides a struct for clusterserviceversion object
// from the cluster and a clusterserviceversion definition.
type ClusterServiceVersionBuilder struct {
	// ClusterServiceVersionBuilder definition. Used to create
	// ClusterServiceVersionBuilder object with minimum set of required elements.
	Definition *oplmV1alpha1.ClusterServiceVersion
	// Created ClusterServiceVersionBuilder object on the cluster.
	Object *oplmV1alpha1.ClusterServiceVersion
	// api client to interact with the cluster.
	apiClient *clients.Settings
	// errorMsg is processed before ClusterServiceVersionBuilder object is created.
	errorMsg string
}

// ListClusterServiceVersion returns clusterserviceversion inventory in the given namespace.
func ListClusterServiceVersion(
	apiClient *clients.Settings,
	nsname string,
	options metaV1.ListOptions) ([]*ClusterServiceVersionBuilder, error) {
	glog.V(100).Infof("Listing clusterserviceversions in the namespace %s with the options %v", nsname, options)

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

// PullClusterServiceVersion loads an existing clusterserviceversion into Builder struct.
func PullClusterServiceVersion(apiClient *clients.Settings, name string) (*ClusterServiceVersionBuilder, error) {
	glog.V(100).Infof("Pulling existing clusterserviceversion name: %s", name)

	builder := ClusterServiceVersionBuilder{
		apiClient: apiClient,
		Definition: &oplmV1alpha1.ClusterServiceVersion{
			ObjectMeta: metaV1.ObjectMeta{
				Name: name,
			},
		},
	}

	if name == "" {
		builder.errorMsg = "clusterserviceversion 'name' cannot be empty"
	}

	if !builder.Exists() {
		return nil, fmt.Errorf("clusterserviceversion object %s doesn't exist", name)
	}

	builder.Definition = builder.Object

	return &builder, nil
}

// Exists checks whether the given clusterserviceversion exists.
func (builder *ClusterServiceVersionBuilder) Exists() bool {
	glog.V(100).Infof(
		"Checking if clusterserviceversion %s exists",
		builder.Definition.Name)

	var err error
	builder.Object, err = builder.apiClient.OperatorsV1alpha1Interface.ClusterServiceVersions(
		builder.Definition.Namespace).Get(
		context.Background(), builder.Definition.Name, metaV1.GetOptions{})

	return err == nil || !k8serrors.IsNotFound(err)
}
