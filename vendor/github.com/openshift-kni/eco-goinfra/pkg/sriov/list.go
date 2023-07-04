package sriov

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ListNetworkNodeState returns SriovNetworkNodeStates inventory in the given namespace.
func ListNetworkNodeState(
	apiClient *clients.Settings, nsname string, options metaV1.ListOptions) ([]*NetworkNodeStateBuilder, error) {
	glog.V(100).Infof("Listing SriovNetworkNodeStates in the namespace %s with the options %v", nsname, options)

	if nsname == "" {
		glog.V(100).Infof("SriovNetworkNodeStates 'nsname' parameter can not be empty")

		return nil, fmt.Errorf("failed to list SriovNetworkNodeStates, 'nsname' parameter is empty")
	}

	networkNodeStateList, err := apiClient.SriovNetworkNodeStates(nsname).List(context.Background(), options)

	if err != nil {
		glog.V(100).Infof("Failed to list SriovNetworkNodeStates in the namespace %s due to %s", nsname, err.Error())

		return nil, err
	}

	var networkNodeStateObjects []*NetworkNodeStateBuilder

	for _, networkNodeState := range networkNodeStateList.Items {
		copiedNetworkNodeState := networkNodeState
		stateBuilder := &NetworkNodeStateBuilder{
			apiClient: apiClient,
			Objects:   &copiedNetworkNodeState,
			nsName:    nsname,
			nodeName:  copiedNetworkNodeState.Name}

		networkNodeStateObjects = append(networkNodeStateObjects, stateBuilder)
	}

	return networkNodeStateObjects, nil
}

// ListPolicy returns SriovNetworkNodePolicies inventory in the given namespace.
func ListPolicy(apiClient *clients.Settings, nsname string, options metaV1.ListOptions) ([]*PolicyBuilder, error) {
	glog.V(100).Infof("Listing SriovNetworkNodePolicies in the namespace %s with the options %v",
		nsname, options)

	if nsname == "" {
		glog.V(100).Infof("SriovNetworkNodePolicies 'nsname' parameter can not be empty")

		return nil, fmt.Errorf("failed to list SriovNetworkNodePolicies, 'nsname' parameter is empty")
	}

	networkNodePoliciesList, err := apiClient.SriovNetworkNodePolicies(nsname).List(context.Background(), options)

	if err != nil {
		glog.V(100).Infof("Failed to list SriovNetworkNodePolicies in the namespace %s due to %s",
			nsname, err.Error())

		return nil, err
	}

	var networkNodePolicyObjects []*PolicyBuilder

	for _, policy := range networkNodePoliciesList.Items {
		copiedNetworkNodePolicy := policy
		policyBuilder := &PolicyBuilder{
			apiClient:  apiClient,
			Object:     &copiedNetworkNodePolicy,
			Definition: &copiedNetworkNodePolicy}

		networkNodePolicyObjects = append(networkNodePolicyObjects, policyBuilder)
	}

	return networkNodePolicyObjects, nil
}
