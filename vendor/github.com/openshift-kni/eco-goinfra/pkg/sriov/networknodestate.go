package sriov

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/glog"
	srIovV1 "github.com/k8snetworkplumbingwg/sriov-network-operator/api/v1"
	clientSrIov "github.com/k8snetworkplumbingwg/sriov-network-operator/pkg/client/clientset/versioned"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

// NetworkNodeStateBuilder provides struct for SriovNetworkNodeState object which contains connection to cluster and
// SriovNetworkNodeState definitions.
type NetworkNodeStateBuilder struct {
	// Dynamically discovered SriovNetworkNodeState object.
	Objects *srIovV1.SriovNetworkNodeState
	// apiClient opens api connection to the cluster.
	apiClient clientSrIov.Interface
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
		apiClient: apiClient.ClientSrIov,
		nodeName:  nodeName,
		nsName:    nsname,
	}

	if nodeName == "" {
		glog.V(100).Infof("The name of the nodeName is empty")

		builder.errorMsg = "SriovNetworkNodeState 'nodeName' is empty"
	}

	if nsname == "" {
		glog.V(100).Infof("The namespace of the SriovNetworkNodeState is empty")

		builder.errorMsg = "SriovNetworkNodeState 'nsname' is empty"
	}

	return builder
}

// Discover method gets the SriovNetworkNodeState items and stores them in the NetworkNodeStateBuilder struct.
func (builder *NetworkNodeStateBuilder) Discover() error {
	if valid, err := builder.validate(); !valid {
		return err
	}

	glog.V(100).Infof("Getting the SriovNetworkNodeState object in namespace %s for node %s",
		builder.nsName, builder.nodeName)

	var err error
	builder.Objects, err = builder.apiClient.SriovnetworkV1().SriovNetworkNodeStates(builder.nsName).Get(
		context.TODO(), builder.nodeName, v1.GetOptions{})

	return err
}

// GetUpNICs returns a list of SrIov interfaces in UP state.
func (builder *NetworkNodeStateBuilder) GetUpNICs() (srIovV1.InterfaceExts, error) {
	if valid, err := builder.validate(); !valid {
		return nil, err
	}

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
	if valid, err := builder.validate(); !valid {
		return nil, err
	}

	if err := builder.Discover(); err != nil {
		glog.V(100).Infof("Error to discover sriov interfaces for node %s", builder.nodeName)

		return nil, err
	}

	glog.V(100).Infof("Collected sriov interfaces list %v for node %s",
		builder.Objects.Status.Interfaces, builder.nodeName)

	return builder.Objects.Status.Interfaces, nil
}

// WaitUntilSyncStatus waits for the duration of the defined timeout or until the
// SriovNetworkNodeState gets to a specific syncStatus.
func (builder *NetworkNodeStateBuilder) WaitUntilSyncStatus(syncStatus string, timeout time.Duration) error {
	if valid, err := builder.validate(); !valid {
		return err
	}

	glog.V(100).Infof("Waiting for the defined period until SriovNetworkNodeState %s has syncStatus %s",
		builder.Objects.Name, syncStatus)

	if syncStatus == "" {
		glog.V(100).Infof("The syncStatus parameter is empty")

		return fmt.Errorf("syncStatus can't be empty")
	}

	// Polls every retryInterval to determine if SriovNetworkNodeState is in desired syncStatus.
	return wait.PollUntilContextTimeout(
		context.TODO(), time.Second, timeout, true, func(ctx context.Context) (bool, error) {
			err := builder.Discover()

			if err != nil {
				return false, nil
			}

			return builder.Objects.Status.SyncStatus == syncStatus, nil
		})
}

// GetNumVFs returns num-vfs under the given interface.
func (builder *NetworkNodeStateBuilder) GetNumVFs(sriovInterfaceName string) (int, error) {
	glog.V(100).Infof("Getting num-vfs under interface %s from SriovNetworkNodeState %s",
		sriovInterfaceName, builder.nodeName)

	interf, err := builder.findInterfaceByName(sriovInterfaceName)
	if err != nil {
		return 0, err
	}

	return interf.NumVfs, nil
}

// GetDriverName returns driver name under the given interface.
func (builder *NetworkNodeStateBuilder) GetDriverName(sriovInterfaceName string) (string, error) {
	glog.V(100).Infof("Getting driver name for interface %s from SriovNetworkNodeState %s",
		sriovInterfaceName, builder.nodeName)

	interf, err := builder.findInterfaceByName(sriovInterfaceName)
	if err != nil {
		return "", err
	}

	return interf.Driver, nil
}

// GetPciAddress returns PciAddress under the given interface.
func (builder *NetworkNodeStateBuilder) GetPciAddress(sriovInterfaceName string) (string, error) {
	glog.V(100).Infof("Getting PCI address for interface %s from SriovNetworkNodeState %s",
		sriovInterfaceName, builder.nodeName)

	interf, err := builder.findInterfaceByName(sriovInterfaceName)
	if err != nil {
		return "", err
	}

	return interf.PciAddress, nil
}

func (builder *NetworkNodeStateBuilder) findInterfaceByName(sriovInterfaceName string) (*srIovV1.InterfaceExt, error) {
	if valid, err := builder.validate(); !valid {
		return nil, err
	}

	if err := builder.Discover(); err != nil {
		glog.V(100).Infof("Error to discover sriov network node state for node %s", builder.nodeName)

		builder.errorMsg = "failed to discover sriov network node state"
	}

	if sriovInterfaceName == "" {
		glog.V(100).Infof("The sriovInterface can not be empty string")

		builder.errorMsg = "the sriovInterface is an empty sting"
	}

	if builder.errorMsg != "" {
		return nil, fmt.Errorf(builder.errorMsg)
	}

	for _, interf := range builder.Objects.Status.Interfaces {
		if interf.Name == sriovInterfaceName {
			return &interf, nil
		}
	}

	return nil, fmt.Errorf("interface %s was not found", sriovInterfaceName)
}

// validate will check that the builder and builder definition are properly initialized before
// accessing any member fields.
func (builder *NetworkNodeStateBuilder) validate() (bool, error) {
	resourceCRD := "SriovNetworkNodeState"

	if builder == nil {
		glog.V(100).Infof("The %s builder is uninitialized", resourceCRD)

		return false, fmt.Errorf("error: received nil %s builder", resourceCRD)
	}

	if builder.apiClient == nil {
		glog.V(100).Infof("The %s builder apiclient is nil", resourceCRD)

		builder.errorMsg = fmt.Sprintf("%s builder cannot have nil apiClient", resourceCRD)
	}

	if builder.errorMsg != "" {
		glog.V(100).Infof("The %s builder has error message: %s", resourceCRD, builder.errorMsg)

		return false, fmt.Errorf(builder.errorMsg)
	}

	return true, nil
}
