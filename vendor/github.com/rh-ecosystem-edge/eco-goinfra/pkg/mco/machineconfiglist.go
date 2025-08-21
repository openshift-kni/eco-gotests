package mco

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/clients"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ListMC returns a list of builders for MachineConfigs.
func ListMC(apiClient *clients.Settings, options ...metav1.ListOptions) ([]*MCBuilder, error) {
	if apiClient == nil {
		glog.V(100).Info("MachineConfig 'apiClient' can not be empty")

		return nil, fmt.Errorf("failed to list MachineConfigs, 'apiClient' parameter is empty")
	}

	passedOptions := metav1.ListOptions{}
	logMessage := "Listing all MC resources"

	if len(options) > 1 {
		glog.V(100).Infof("'options' parameter must be empty or single-valued")

		return nil, fmt.Errorf("error: more than one ListOptions was passed")
	}

	if len(options) == 1 {
		passedOptions = options[0]
		logMessage += fmt.Sprintf(" with the options %v", passedOptions)
	}

	glog.V(100).Infof(logMessage)

	mcList, err := apiClient.MachineConfigs().List(context.TODO(), passedOptions)
	if err != nil {
		glog.V(100).Info("Failed to list MC objects due to %s", err.Error())

		return nil, err
	}

	var mcObjects []*MCBuilder

	for _, mc := range mcList.Items {
		copiedMc := mc
		mcBuilder := &MCBuilder{
			apiClient:  apiClient,
			Object:     &copiedMc,
			Definition: &copiedMc,
		}

		mcObjects = append(mcObjects, mcBuilder)
	}

	return mcObjects, nil
}
