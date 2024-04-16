package sriov

import (
	"encoding/json"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/sriov"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NetworkInfo structure to store pod network information.
type NetworkInfo struct {
	Name string `json:"name"`
}

// ListNetworksByDeviceType returns a list of sriov networks matching the policies
// device type.
func ListNetworksByDeviceType(
	apiClient *clients.Settings,
	deviceType string,
) ([]string, error) {
	var devNetworks []string

	operatornsname := "openshift-sriov-network-operator"
	options := metav1.ListOptions{}
	sriovPolicies, err := sriov.ListPolicy(apiClient, operatornsname, options)

	if err != nil {
		glog.V(100).Infof("Failed to list sriov policies in namespace: %s", operatornsname)

		return nil, err
	}

	sriovNetworks, err := sriov.List(apiClient, operatornsname, options)

	if err != nil {
		glog.V(100).Infof("Failed to list sriov networks in namespace: %s", operatornsname)

		return nil, err
	}

	for _, policy := range sriovPolicies {
		if policy.Definition.Spec.DeviceType == deviceType {
			for _, network := range sriovNetworks {
				if policy.Definition.Spec.ResourceName == network.Definition.Spec.ResourceName {
					devNetworks = append(devNetworks, network.Definition.Name)
				}
			}
		}
	}

	return devNetworks, nil
}

// ExtractNetworkNames returns the name of the networks based on the pods
// network status annotations.
func ExtractNetworkNames(jsonData string) ([]string, error) {
	var networkInfo []NetworkInfo

	// Unmarshal the JSON data into the networkInfo slice.
	err := json.Unmarshal([]byte(jsonData), &networkInfo)
	if err != nil {
		return nil, err
	}

	// Extract the interface names into a separate slice.
	var networkNames []string

	for _, info := range networkInfo {
		if info.Name != "ovn-kubernetes" {
			networkNames = append(networkNames, info.Name)
		}
	}

	return networkNames, nil
}
