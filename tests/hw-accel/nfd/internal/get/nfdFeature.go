package get

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/nfd/internal/params"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/nfd/nfdparams"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
)

// NodeMetadata represents the metadata of a NodeFeature resource.
type NodeMetadata struct {
	Name string `json:"name"`
	UID  string `json:"uid"`
}

// GetNodeFeatures retrieves NodeFeature resources from the cluster.
func GetNodeFeatures(apiClient *clients.Settings) ([]NodeMetadata, error) {
	defer func() {
		if r := recover(); r != nil {
			glog.Errorf("panic occurred: %v", r)
		}
	}()

	dynamicClient, err := dynamic.NewForConfig(apiClient.Config)
	if err != nil {
		glog.Errorf("error creating dynamic client: %v", err)

		return nil, fmt.Errorf("error creating dynamic client: %w", err)
	}

	for _, gvr := range params.PossibleGVRs {
		glog.Infof("Trying to list NodeFeatures with GVR: %s/%s", gvr.Group, gvr.Version)

		nodeFeatureCRs, err := dynamicClient.Resource(gvr).Namespace(nfdparams.NFDNamespace).List(
			context.Background(),
			metav1.ListOptions{})
		if err != nil {
			glog.Warningf("Failed to list NodeFeatures for GVR %s/%s: %v", gvr.Group, gvr.Version, err)

			continue
		}

		glog.Infof("Successfully listed NodeFeatures for GVR %s/%s. Found %d items.",
			gvr.Group, gvr.Version, len(nodeFeatureCRs.Items))

		var allMetadata []NodeMetadata

		for _, cr := range nodeFeatureCRs.Items {
			metadata := NodeMetadata{
				Name: cr.GetName(),
				UID:  string(cr.GetUID()),
			}
			allMetadata = append(allMetadata, metadata)
			glog.V(2).Infof("Found NodeFeature: Name=%s, UID=%s", metadata.Name, metadata.UID)
		}

		return allMetadata, nil
	}

	glog.Errorf("Failed to list NodeFeature resources in all known API groups: %v", err)

	return nil, fmt.Errorf("failed to list NodeFeature resources in all known API groups: %w", err)
}
