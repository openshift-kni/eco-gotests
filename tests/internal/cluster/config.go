package cluster

import (
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/clusterversion"
	"github.com/openshift-kni/eco-goinfra/pkg/network"
	"github.com/openshift-kni/eco-goinfra/pkg/proxy"
	"github.com/openshift-kni/eco-goinfra/pkg/secret"
)

// APIClientGetter is an interface that returns an APIClient from a struct.
type APIClientGetter interface {
	GetAPIClient() (*clients.Settings, error)
}

// GetOCPClusterVersion leverages APIClientGetter to retrieve the OCP clusterversion from an arbitrary cluster.
func GetOCPClusterVersion(clusterObj APIClientGetter) (*clusterversion.Builder, error) {
	apiClient, err := checkAPIClient(clusterObj)
	if err != nil {
		return nil, err
	}

	glog.V(90).Infof("Gathering OCP clusterversion from cluster at %s", apiClient.KubeconfigPath)

	clusterVersion, err := clusterversion.Pull(apiClient)
	if err != nil {
		return nil, err
	}

	return clusterVersion, nil
}

// GetOCPNetworkConfig leverages APIClientGetter to retrieve the OCP cluster network from an arbitrary cluster.
func GetOCPNetworkConfig(clusterObj APIClientGetter) (*network.ConfigBuilder, error) {
	apiClient, err := checkAPIClient(clusterObj)
	if err != nil {
		return nil, err
	}

	glog.V(90).Infof("Gathering OCP network from cluster at %s", apiClient.KubeconfigPath)

	clusterNetwork, err := network.PullConfig(apiClient)
	if err != nil {
		return nil, err
	}

	return clusterNetwork, nil
}

// GetOCPNetworkOperatorConfig leverages APIClientGetter to retrieve the OCP cluster network from an arbitrary cluster.
func GetOCPNetworkOperatorConfig(clusterObj APIClientGetter) (*network.OperatorBuilder, error) {
	apiClient, err := checkAPIClient(clusterObj)
	if err != nil {
		return nil, err
	}

	glog.V(90).Infof("Gathering OCP network from cluster at %s", apiClient.KubeconfigPath)

	clusterNetwork, err := network.PullOperator(apiClient)

	if err != nil {
		return nil, err
	}

	return clusterNetwork, nil
}

// GetOCPPullSecret leverages APIClientGetter to retrieve the OCP pull-secret from an arbitrary cluster.
func GetOCPPullSecret(clusterObj APIClientGetter) (*secret.Builder, error) {
	apiClient, err := checkAPIClient(clusterObj)
	if err != nil {
		return nil, err
	}

	glog.V(90).Infof("Gathering OCP pull-secret from cluster at %s", apiClient.KubeconfigPath)

	pullSecret, err := secret.Pull(apiClient, "pull-secret", "openshift-config")
	if err != nil {
		return nil, err
	}

	_, ok := pullSecret.Object.Data[".dockerconfigjson"]
	if !ok {
		return nil, err
	}

	return pullSecret, nil
}

// GetOCPProxy leverages APIClientGetter to retrieve the OCP cluster proxy from an arbitrary cluster.
func GetOCPProxy(clusterObj APIClientGetter) (*proxy.Builder, error) {
	apiClient, err := checkAPIClient(clusterObj)
	if err != nil {
		return nil, err
	}

	glog.V(90).Infof("Gathering OCP proxy from cluster at %s", apiClient.KubeconfigPath)

	clusterProxy, err := proxy.Pull(apiClient)
	if err != nil {
		return nil, err
	}

	return clusterProxy, err
}

// checkAPIClient determines if the APIClient returned is not nil.
func checkAPIClient(clusterObj APIClientGetter) (*clients.Settings, error) {
	glog.V(90).Infof("Getting APIClient from provided object")

	apiClient, err := clusterObj.GetAPIClient()
	if err != nil {
		return nil, err
	}

	if apiClient == nil {
		glog.V(90).Infof("The returned APIClient is nil")

		return nil, fmt.Errorf("cannot discover cluster information when APIClient is nil")
	}

	return apiClient, nil
}
