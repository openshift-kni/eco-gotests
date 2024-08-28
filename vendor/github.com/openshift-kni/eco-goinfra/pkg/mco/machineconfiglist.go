package mco

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ListMC returns a list of builders for MachineConfigs.
func ListMC(apiClient *clients.Settings, options ...runtimeclient.ListOptions) ([]*MCBuilder, error) {
	if apiClient == nil {
		glog.V(100).Info("MachineConfig 'apiClient' can not be empty")

		return nil, fmt.Errorf("failed to list MachineConfigs, 'apiClient' parameter is empty")
	}

	err := apiClient.AttachScheme(mcv1.Install)
	if err != nil {
		glog.V(100).Info("Failed to add machineconfig v1 scheme to client schemes")

		return nil, err
	}

	passedOptions := runtimeclient.ListOptions{}
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

	mcList := new(mcv1.MachineConfigList)
	err = apiClient.Client.List(context.TODO(), mcList, &passedOptions)

	if err != nil {
		glog.V(100).Info("Failed to list MC objects due to %s", err.Error())

		return nil, err
	}

	var mcObjects []*MCBuilder

	for _, mc := range mcList.Items {
		copiedMc := mc
		mcBuilder := &MCBuilder{
			apiClient:  apiClient.Client,
			Object:     &copiedMc,
			Definition: &copiedMc,
		}

		mcObjects = append(mcObjects, mcBuilder)
	}

	return mcObjects, nil
}
