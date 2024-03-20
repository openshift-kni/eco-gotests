package sriov

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ListPolicy returns SriovNetworkNodePolicies inventory in the given namespace.
func ListPolicy(apiClient *clients.Settings, nsname string, options ...metav1.ListOptions) ([]*PolicyBuilder, error) {
	if nsname == "" {
		glog.V(100).Infof("SriovNetworkNodePolicies 'nsname' parameter can not be empty")

		return nil, fmt.Errorf("failed to list SriovNetworkNodePolicies, 'nsname' parameter is empty")
	}

	passedOptions := metav1.ListOptions{}
	logMessage := fmt.Sprintf("Listing SriovNetworkNodePolicies in the namespace %s", nsname)

	if len(options) > 1 {
		glog.V(100).Infof("'options' parameter must be empty or single-valued")

		return nil, fmt.Errorf("error: more than one ListOptions was passed")
	}

	if len(options) == 1 {
		passedOptions = options[0]
		logMessage += fmt.Sprintf(" with the options %v", passedOptions)
	}

	glog.V(100).Infof(logMessage)

	networkNodePoliciesList, err := apiClient.SriovNetworkNodePolicies(nsname).List(context.Background(), passedOptions)

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

// CleanAllNetworkNodePolicies removes all SriovNetworkNodePolicies that are not set as default.
func CleanAllNetworkNodePolicies(
	apiClient *clients.Settings, operatornsname string, options ...metav1.ListOptions) error {
	glog.V(100).Infof("Cleaning up SriovNetworkNodePolicies in the %s namespace", operatornsname)

	if operatornsname == "" {
		glog.V(100).Infof("'operatornsname' parameter can not be empty")

		return fmt.Errorf("failed to clean up SriovNetworkNodePolicies, 'operatornsname' parameter is empty")
	}

	policies, err := ListPolicy(apiClient, operatornsname, options...)

	if err != nil {
		glog.V(100).Infof("Failed to list SriovNetworkNodePolicies in namespace: %s", operatornsname)

		return err
	}

	for _, policy := range policies {
		// The "default" SriovNetworkNodePolicy is both mandatory and the default option.
		if policy.Object.Name != "default" {
			err = policy.Delete()

			if err != nil {
				glog.V(100).Infof("Failed to delete SriovNetworkNodePolicy: %s", policy.Object.Name)

				return err
			}
		}
	}

	return nil
}
