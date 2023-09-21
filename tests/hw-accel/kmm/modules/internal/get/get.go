package get

import (
	"encoding/base64"
	"fmt"

	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/internal/kmmparams"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
)

// NumberOfNodesForSelector returns the number or worker nodes.
func NumberOfNodesForSelector(apiClient *clients.Settings, selector map[string]string) (int, error) {
	nodeBuilder := nodes.NewBuilder(apiClient, selector)

	if err := nodeBuilder.Discover(); err != nil {
		fmt.Println("could not discover number of nodes")

		return 0, err
	}

	glog.V(kmmparams.KmmLogLevel).Infof(
		"NumberOfNodesForSelector return %v nodes", len(nodeBuilder.Objects))

	return len(nodeBuilder.Objects), nil
}

// ClusterArchitecture returns first node architecture of the nodes that match nodeSelector (e.g. worker nodes).
func ClusterArchitecture(apiClient *clients.Settings, nodeSelector map[string]string) (string, error) {
	nodeLabel := "kubernetes.io/arch"

	return getLabelFromNodeSelector(apiClient, nodeLabel, nodeSelector)
}

// KernelFullVersion returns first node architecture of the nodes that match nodeSelector (e.g. worker nodes).
func KernelFullVersion(apiClient *clients.Settings, nodeSelector map[string]string) (string, error) {
	nodeLabel := "kmm.node.kubernetes.io/kernel-version.full"

	return getLabelFromNodeSelector(apiClient, nodeLabel, nodeSelector)
}

func getLabelFromNodeSelector(apiClient *clients.Settings, nodeLabel string,
	nodeSelector map[string]string) (string, error) {
	nodeBuilder := nodes.NewBuilder(apiClient, nodeSelector)
	// Check if at least one node matching the nodeSelector has the specific nodeLabel label set to true
	// For example, look in all the worker nodes for specific label
	if err := nodeBuilder.Discover(); err != nil {
		glog.V(kmmparams.KmmLogLevel).Infof("could not discover %v nodes", nodeSelector)

		return "", err
	}

	for _, node := range nodeBuilder.Objects {
		labelValue, ok := node.Object.Labels[nodeLabel]

		if ok {
			glog.V(kmmparams.KmmLogLevel).Infof("Found label '%v' with label value '%v' on node '%v'",
				nodeLabel, labelValue, node.Object.Name)

			return labelValue, nil
		}
	}

	err := fmt.Errorf("could not find one node with label '%s'", nodeLabel)

	return "", err
}

// MachineConfigPoolName returns machineconfigpool's name for a specified label.
func MachineConfigPoolName(apiClient *clients.Settings) string {
	nodeBuilder := nodes.NewBuilder(apiClient, map[string]string{"kubernetes.io/os": "linux"})

	if err := nodeBuilder.Discover(); err != nil {
		glog.V(kmmparams.KmmLogLevel).Infof("could not discover nodes")

		return ("")
	}

	if len(nodeBuilder.Objects) == 1 {
		glog.V(kmmparams.KmmLogLevel).Infof("Using 'master' as mcp")

		return "master"
	}

	glog.V(kmmparams.KmmLogLevel).Infof("Using 'worker' as mcp")

	return "worker"
}

// SigningData returns struct used for creating secrets for module signing.
func SigningData(key string, value string) map[string][]byte {
	val, err := base64.StdEncoding.DecodeString(value)

	if err != nil {
		glog.V(kmmparams.KmmLogLevel).Infof("Error decoding signing key")
	}

	secretContents := map[string][]byte{key: val}

	return secretContents
}
