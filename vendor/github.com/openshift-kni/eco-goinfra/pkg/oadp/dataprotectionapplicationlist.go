package oadp

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/oadp/oadptypes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ListDataProtectionApplication returns dataprotectionapplication inventory in the given namespace.
func ListDataProtectionApplication(
	apiClient *clients.Settings, nsname string, options ...metav1.ListOptions) ([]*DPABuilder, error) {
	glog.V(100).Infof("Listing dataprotectionapplications in namespace %s", nsname)

	if apiClient == nil {
		glog.V(100).Infof("The apiClient cannot be nil")

		return nil, fmt.Errorf("the apiClient is nil")
	}

	if nsname == "" {
		glog.V(100).Infof("dataprotectionapplication 'nsname' parameter can not be empty")

		return nil, fmt.Errorf("failed to list dataprotectionapplications, 'nsname' parameter is empty")
	}

	logMessage := fmt.Sprintf("Listing dataprotectionapplications in the nsname %s", nsname)
	passedOptions := metav1.ListOptions{}

	if len(options) == 1 {
		passedOptions = options[0]
		logMessage += fmt.Sprintf(" with the options %v", passedOptions)
	} else if len(options) > 1 {
		glog.V(100).Infof("'options' parameter must be empty or single-valued")

		return nil, fmt.Errorf("error: more than one ListOptions was passed")
	}

	glog.V(100).Infof(logMessage)

	unstructObjList, err := apiClient.Resource(GetDataProtectionApplicationGVR()).
		Namespace(nsname).List(context.TODO(), passedOptions)
	if err != nil {
		glog.V(100).Infof("Failed to list dataprotectionapplications in the nsname %s due to %s", nsname, err.Error())

		return nil, err
	}

	var dpaObjects []*DPABuilder

	for _, runningDPA := range unstructObjList.Items {
		dataprotectionapplication := &oadptypes.DataProtectionApplication{}

		err = runtime.DefaultUnstructuredConverter.FromUnstructured(runningDPA.Object, dataprotectionapplication)
		if err != nil {
			glog.V(100).Infof(
				"Failed to convert from unstructured list to DataProtectionApplicationList object in namespace %s",
				nsname)

			return nil, err
		}

		dpaBuilder := &DPABuilder{
			apiClient:  apiClient,
			Object:     dataprotectionapplication,
			Definition: dataprotectionapplication,
		}

		dpaObjects = append(dpaObjects, dpaBuilder)
	}

	return dpaObjects, nil
}
