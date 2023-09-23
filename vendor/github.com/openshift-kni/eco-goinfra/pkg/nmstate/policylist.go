package nmstate

import (
	"context"

	"github.com/golang/glog"
	nmstateV1 "github.com/nmstate/kubernetes-nmstate/api/v1"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
)

// ListPolicy returns a list of NodeNetworkConfigurationPolicy.
func ListPolicy(apiClient *clients.Settings) ([]*PolicyBuilder, error) {
	glog.V(100).Infof("Listing NodeNetworkConfigurationPolicy")

	policyList := &nmstateV1.NodeNetworkConfigurationPolicyList{}
	err := apiClient.Client.List(context.Background(), policyList)

	if err != nil {
		glog.V(100).Infof("Failed to list NodeNetworkConfigurationPolicy due to %s", err.Error())

		return nil, err
	}

	var networkConfigurationPolicyObjects []*PolicyBuilder

	for _, policy := range policyList.Items {
		copiedPolicy := policy
		policyBuilder := &PolicyBuilder{
			apiClient:  apiClient,
			Definition: &copiedPolicy,
			Object:     &copiedPolicy}

		networkConfigurationPolicyObjects = append(networkConfigurationPolicyObjects, policyBuilder)
	}

	return networkConfigurationPolicyObjects, nil
}

// CleanAllNMStatePolicies removes all NodeNetworkConfigurationPolicies.
func CleanAllNMStatePolicies(apiClient *clients.Settings) error {
	glog.V(100).Infof("Cleaning up NodeNetworkConfigurationPolicies")

	nncpList, err := ListPolicy(apiClient)
	if err != nil {
		glog.V(100).Infof("Failed to list NodeNetworkConfigurationPolicies")

		return err
	}

	for _, nncpPolicy := range nncpList {
		_, err = nncpPolicy.Delete()
		if err != nil {
			glog.V(100).Infof("Failed to delete NodeNetworkConfigurationPolicy: %s", nncpPolicy.Object.Name)

			return err
		}
	}

	return nil
}
