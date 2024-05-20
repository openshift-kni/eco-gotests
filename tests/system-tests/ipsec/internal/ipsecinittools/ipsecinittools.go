package ipsecinittools

import (
	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-gotests/tests/internal/inittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/ipsec/internal/ipsecconfig"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	// APIClient provides API access to cluster.
	APIClient *clients.Settings
	// IpsecTestConfig provides access to IPSec system tests configuration parameters.
	IpsecTestConfig *ipsecconfig.IpsecConfig
)

// init loads all variables automatically when this package is imported.
// Once package is imported a user has full access to all vars within init function.
// It is recommended to import this package using dot import.
func init() {
	IpsecTestConfig = ipsecconfig.NewIpsecConfig()
	APIClient = inittools.APIClient
}

// GetNodeNames returns a string slice with all of the cluster node names.
func GetNodeNames() ([]string, error) {
	nodeList, err := nodes.List(
		APIClient,
		metav1.ListOptions{},
	)
	if err != nil {
		glog.V(100).Infof("Error listing nodes.")

		return nil, err
	}

	nodeNames := []string{}
	for _, node := range nodeList {
		nodeNames = append(nodeNames, node.Definition.Name)
	}

	return nodeNames, nil
}
