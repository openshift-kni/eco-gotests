package ocm

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	policiesv1beta1 "open-cluster-management.io/governance-policy-propagator/api/v1beta1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ListPolicieSetsInAllNamespaces returns a cluster-wide policySets inventory.
func ListPolicieSetsInAllNamespaces(apiClient *clients.Settings,
	options ...runtimeclient.ListOptions) (
	[]*PolicySetBuilder, error) {
	logMessage := string("Listing all policySets in all namespaces")
	passedOptions := runtimeclient.ListOptions{}

	if len(options) > 1 {
		glog.V(100).Infof("'options' parameter must be empty or single-valued")

		return nil, fmt.Errorf("error: more than one ListOptions was passed")
	}

	if len(options) == 1 {
		passedOptions = options[0]
		logMessage += fmt.Sprintf(" with the options %v", passedOptions)
	}

	glog.V(100).Infof(logMessage)

	policySetList := &policiesv1beta1.PolicySetList{}

	err := apiClient.Client.List(context.Background(), policySetList, &passedOptions)

	if err != nil {
		glog.V(100).Infof("Failed to list all policySets in all namespaces due to %s", err.Error())

		return nil, err
	}

	var policySetObjects []*PolicySetBuilder

	for _, policy := range policySetList.Items {
		copiedPolicySet := policy
		policySetBuilder := &PolicySetBuilder{
			apiClient:  apiClient,
			Object:     &copiedPolicySet,
			Definition: &copiedPolicySet,
		}

		policySetObjects = append(policySetObjects, policySetBuilder)
	}

	return policySetObjects, nil
}
