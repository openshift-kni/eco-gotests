package sriov

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// List returns sriov networks in the given namespace.
func List(apiClient *clients.Settings, nsname string, options metaV1.ListOptions) ([]*NetworkBuilder, error) {
	glog.V(100).Infof("Listing sriov networks in the namespace %s with the options %v", nsname, options)

	if nsname == "" {
		glog.V(100).Infof("sriov network 'nsname' parameter can not be empty")

		return nil, fmt.Errorf("failed to list sriov networks, 'nsname' parameter is empty")
	}

	networkList, err := apiClient.SriovNetworks(nsname).List(context.Background(), options)

	if err != nil {
		glog.V(100).Infof("Failed to list sriov networks in the namespace %s due to %s", nsname, err.Error())

		return nil, err
	}

	var networkObjects []*NetworkBuilder

	for _, runningNetwork := range networkList.Items {
		copiedNetwork := runningNetwork
		networkBuilder := &NetworkBuilder{
			apiClient:  apiClient,
			Object:     &copiedNetwork,
			Definition: &copiedNetwork,
		}

		networkObjects = append(networkObjects, networkBuilder)
	}

	return networkObjects, nil
}

// CleanAllNetworksByTargetNamespace deletes all networks matched by their NetworkNamespace spec.
func CleanAllNetworksByTargetNamespace(
	apiClient *clients.Settings,
	operatornsname string,
	targetnsname string,
	options metaV1.ListOptions) error {
	glog.V(100).Infof("Cleaning up sriov networks in the %s namespace with %s NetworkNamespace spec",
		operatornsname, targetnsname)

	if operatornsname == "" {
		glog.V(100).Infof("'operatornsname' parameter can not be empty")

		return fmt.Errorf("failed to clean up sriov networks, 'operatornsname' parameter is empty")
	}

	if targetnsname == "" {
		glog.V(100).Infof("'targetnsname' parameter can not be empty")

		return fmt.Errorf("failed to clean up sriov networks, 'targetnsname' parameter is empty")
	}

	networks, err := List(apiClient, operatornsname, options)

	if err != nil {
		glog.V(100).Infof("Failed to list sriov networks in namespace: %s", operatornsname)

		return err
	}

	for _, network := range networks {
		if network.Object.Spec.NetworkNamespace == targetnsname {
			err = network.Delete()
			if err != nil {
				glog.V(100).Infof("Failed to delete sriov networks: %s", network.Object.Name)

				return err
			}
		}
	}

	return nil
}
