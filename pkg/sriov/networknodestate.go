package sriov

import (
	"context"
	"fmt"

	"github.com/golang/glog"

	srIovV1 "github.com/k8snetworkplumbingwg/sriov-network-operator/api/v1"
	"github.com/openshift-kni/eco-gotests/pkg/clients"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NetworkNodeStateBuilder provides struct for SriovNetworkNodeState object which contains connection to cluster and
// SriovNetworkNodeState definitions.
type NetworkNodeStateBuilder struct {
	// Dynamically discovered SriovNetworkNodeState object.
	Objects *srIovV1.SriovNetworkNodeState
	// apiClient opens api connection to the cluster.
	apiClient *clients.Settings
	// nodeName defines on what node SriovNetworkNodeState resource should be queried.
	nodeName string
	// nsName defines SrIov operator namespace.
	nsName string
	// errorMsg used in discovery function before sending api request to cluster.
	errorMsg string
}

// NewNetworkNodeStateBuilder creates new instance of NetworkNodeStateBuilder.
func NewNetworkNodeStateBuilder(apiClient *clients.Settings, nodeName, nsname string) *NetworkNodeStateBuilder {
	glog.V(100).Infof(
		"Initializing new NetworkNodeStateBuilder structure with the following params: %s, %s",
		nodeName, nsname)

	builder := &NetworkNodeStateBuilder{
		apiClient: apiClient,
		nodeName:  nodeName,
		nsName:    nsname,
	}

	if nodeName == "" {
		glog.V(100).Infof("The name of the nodeName is empty")

		builder.errorMsg = "error: 'nodeName' is empty"
	}

	if nsname == "" {
		glog.V(100).Infof("The namespace of the SriovNetworkNodeState is empty")

		builder.errorMsg = "error: 'nsname' is empty"
	}

	return builder
}

// Discover method gets the SriovNetworkNodeState items and stores them in the NetworkNodeStateBuilder struct.
func (builder *NetworkNodeStateBuilder) Discover() error {
	if builder.errorMsg != "" {
		return fmt.Errorf(builder.errorMsg)
	}

	glog.V(100).Infof("Getting the SriovNetworkNodeState object in namespace %s for node %s",
		builder.nsName, builder.nodeName)

	var err error
	builder.Objects, err = builder.apiClient.SriovNetworkNodeStates(builder.nsName).Get(
		context.TODO(), builder.nodeName, v1.GetOptions{})

	return err
}

// GetUpNICs returns a list of SrIov interfaces in UP state.
func (builder *NetworkNodeStateBuilder) GetUpNICs() (srIovV1.InterfaceExts, error) {
	glog.V(100).Infof("Collection of sriov interfaces in UP state for node %s", builder.nodeName)
	sriovNics, err := builder.GetNICs()

	if err != nil {
		glog.V(100).Infof("Error to discover sriov interfaces for node %s", builder.nodeName)

		return nil, err
	}

	var sriovNicsUp srIovV1.InterfaceExts

	for _, nic := range sriovNics {
		if nic.LinkSpeed != "" && nic.LinkSpeed != "-1 Mb/s" {
			glog.V(100).Infof("Interface %s is UP on node %s. Append to list", nic.Name, builder.nodeName)
			sriovNicsUp = append(sriovNicsUp, nic)
		}
	}

	glog.V(100).Infof("Collected sriov UP interfaces list %v for node %s",
		builder.Objects.Status.Interfaces, builder.nodeName)

	return sriovNicsUp, nil
}

// GetNICs returns a list of SrIov interfaces.
func (builder *NetworkNodeStateBuilder) GetNICs() (srIovV1.InterfaceExts, error) {
	if err := builder.Discover(); err != nil {
		glog.V(100).Infof("Error to discover sriov interfaces for node %s", builder.nodeName)

		return nil, err
	}

	glog.V(100).Infof("Collected sriov interfaces list %v for node %s",
		builder.Objects.Status.Interfaces, builder.nodeName)

	return builder.Objects.Status.Interfaces, nil
}
