package sriov

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// List returns sriov networks in the given namespace.
func List(apiClient *clients.Settings, nsname string, options ...metav1.ListOptions) ([]*NetworkBuilder, error) {
	if nsname == "" {
		glog.V(100).Infof("sriov network 'nsname' parameter can not be empty")

		return nil, fmt.Errorf("failed to list sriov networks, 'nsname' parameter is empty")
	}

	passedOptions := metav1.ListOptions{}
	logMessage := fmt.Sprintf("Listing sriov networks in the namespace %s", nsname)

	if len(options) > 1 {
		glog.V(100).Infof("'options' parameter must be empty or single-valued")

		return nil, fmt.Errorf("error: more than one ListOptions was passed")
	}

	if len(options) == 1 {
		passedOptions = options[0]
		logMessage += fmt.Sprintf(" with the options %v", passedOptions)
	}

	glog.V(100).Infof(logMessage)

	networkList, err := apiClient.SriovNetworks(nsname).List(context.Background(), passedOptions)

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
	options ...metav1.ListOptions) error {
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

	networks, err := List(apiClient, operatornsname, options...)

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
