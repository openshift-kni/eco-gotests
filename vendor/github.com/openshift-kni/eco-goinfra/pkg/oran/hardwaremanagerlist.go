package oran

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	pluginv1alpha1 "github.com/openshift-kni/oran-hwmgr-plugin/api/hwmgr-plugin/v1alpha1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ListHardwareManagers returns a list of HardwareManagers in all namespaces, using the provided options.
func ListHardwareManagers(
	apiClient *clients.Settings, options ...runtimeclient.ListOptions) ([]*HardwareManagerBuilder, error) {
	if apiClient == nil {
		glog.V(100).Info("HardwareManagers 'apiClient' parameter cannot be nil")

		return nil, fmt.Errorf("failed to list hardwareManagers, 'apiClient' parameter is nil")
	}

	err := apiClient.AttachScheme(pluginv1alpha1.AddToScheme)
	if err != nil {
		glog.V(100).Info("Failed to add plugin v1alpha1 scheme to client schemes")

		return nil, err
	}

	logMessage := "Listing HardwareManagers in all namespaces"
	passedOptions := runtimeclient.ListOptions{}

	if len(options) > 1 {
		glog.V(100).Info("HardwareManagers 'options' parameter must be empty or single-valued")

		return nil, fmt.Errorf("error: more than one ListOptions was passed")
	}

	if len(options) == 1 {
		passedOptions = options[0]
		logMessage += fmt.Sprintf(" with the options %v", passedOptions)
	}

	glog.V(100).Info(logMessage)

	hwmgrList := new(pluginv1alpha1.HardwareManagerList)
	err = apiClient.Client.List(context.TODO(), hwmgrList, &passedOptions)

	if err != nil {
		glog.V(100).Infof("Failed to list HardwareManagers in all namespaces due to %v", err)

		return nil, err
	}

	var hwmgrObjects []*HardwareManagerBuilder

	for _, hwmgr := range hwmgrList.Items {
		copiedHwmgr := hwmgr
		hwmgrBuilder := &HardwareManagerBuilder{
			apiClient:  apiClient.Client,
			Object:     &copiedHwmgr,
			Definition: &copiedHwmgr,
		}

		hwmgrObjects = append(hwmgrObjects, hwmgrBuilder)
	}

	return hwmgrObjects, nil
}
