package poddisruptionbudget

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	policyv1 "k8s.io/api/policy/v1"
)

// List returns podDisruptionBudget inventory in the given namespace.
func List(apiClient *clients.Settings, nsname string, options ...metav1.ListOptions) ([]*Builder, error) {
	if apiClient == nil {
		glog.V(100).Infof("podDisruptionBudget apiClient is empty")

		return nil, fmt.Errorf("podDisruptionBudget 'apiClient' cannot be empty")
	}

	if nsname == "" {
		glog.V(100).Infof("podDisruptionBudget 'nsname' parameter can not be empty")

		return nil, fmt.Errorf("failed to list podDisruptionBudgets, 'nsname' parameter is empty")
	}

	logMessage := fmt.Sprintf("Listing podDisruptionBudget in the nsname %s", nsname)
	passedOptions := metav1.ListOptions{}

	if len(options) > 1 {
		glog.V(100).Infof("'options' parameter must be empty or single-valued")

		return nil, fmt.Errorf("error: more than one ListOptions was passed")
	}

	if len(options) == 1 {
		passedOptions = options[0]
		logMessage += fmt.Sprintf(" with the options %v", passedOptions)
	}

	glog.V(100).Infof(logMessage)

	return list(apiClient, nsname, passedOptions)
}

// ListInAllNamespaces returns a cluster-wide podDisruptionBudget inventory.
func ListInAllNamespaces(apiClient *clients.Settings, options ...metav1.ListOptions) ([]*Builder, error) {
	logMessage := "Listing all podDisruptionBudget in all namespaces"
	passedOptions := metav1.ListOptions{}

	if apiClient == nil {
		glog.V(100).Infof("The apiClient is empty")

		return nil, fmt.Errorf("podDisruptionBudget 'apiClient' cannot be empty")
	}

	if len(options) > 1 {
		glog.V(100).Infof("'options' parameter must be empty or single-valued")

		return nil, fmt.Errorf("error: more than one ListOptions was passed")
	}

	if len(options) == 1 {
		passedOptions = options[0]
		logMessage += fmt.Sprintf(" with the options %v", passedOptions)
	}

	glog.V(100).Infof(logMessage)

	return list(apiClient, "", passedOptions)
}

// list lists the podDisruptionBudget according to the provided options.
func list(apiClient *clients.Settings, nsname string, options metav1.ListOptions) ([]*Builder, error) {
	err := apiClient.AttachScheme(policyv1.AddToScheme)
	if err != nil {
		glog.V(100).Infof("Failed to add policyv1 scheme to client schemes")

		return nil, err
	}

	pdbList, err := apiClient.PodDisruptionBudgets(nsname).List(context.TODO(), options)

	if err != nil {
		glog.V(100).Infof("Failed to list podDisruptionBudget due to %s", err.Error())

		return nil, err
	}

	var pdbObjects []*Builder

	for _, _pdb := range pdbList.Items {
		copiedPDB := _pdb
		pdbBuilder := &Builder{
			apiClient:  apiClient.PolicyV1Interface,
			Object:     &copiedPDB,
			Definition: &copiedPDB,
		}

		pdbObjects = append(pdbObjects, pdbBuilder)
	}

	return pdbObjects, nil
}
