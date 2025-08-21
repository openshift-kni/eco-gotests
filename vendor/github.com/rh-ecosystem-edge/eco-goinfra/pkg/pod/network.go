package pod

import (
	"fmt"

	"github.com/golang/glog"
	multus "gopkg.in/k8snetworkplumbingwg/multus-cni.v4/pkg/types"
)

// StaticAnnotation defines network annotation for pod object.
func StaticAnnotation(name string) *multus.NetworkSelectionElement {
	return &multus.NetworkSelectionElement{
		Name: name,
	}
}

// StaticIPAnnotation defines static ip address network annotation for pod object.
func StaticIPAnnotation(name string, ipAddr []string) []*multus.NetworkSelectionElement {
	return []*multus.NetworkSelectionElement{
		{
			Name:      name,
			IPRequest: ipAddr,
		},
	}
}

// StaticIPAnnotationWithInterfaceAndNamespace defines static ip address, interface name and namespace
// network annotation for pod object.
func StaticIPAnnotationWithInterfaceAndNamespace(
	name, namespace, intName string, ipAddr []string) []*multus.NetworkSelectionElement {
	baseAnnotation := StaticIPAnnotationWithNamespace(name, namespace, ipAddr)
	baseAnnotation[0].InterfaceRequest = intName

	return baseAnnotation
}

// StaticIPAnnotationWithMacAddress defines static ip address and static macaddress network annotation for pod object.
func StaticIPAnnotationWithMacAddress(name string, ipAddr []string, macAddr string) []*multus.NetworkSelectionElement {
	baseAnnotation := StaticIPAnnotation(name, ipAddr)
	baseAnnotation[0].MacRequest = macAddr

	return baseAnnotation
}

// StaticIPAnnotationWithNamespace defines static ip address and namespace network annotation for pod object.
func StaticIPAnnotationWithNamespace(name, namespace string, ipAddr []string) []*multus.NetworkSelectionElement {
	baseAnnotation := StaticIPAnnotation(name, ipAddr)
	baseAnnotation[0].Namespace = namespace

	return baseAnnotation
}

// StaticIPAnnotationWithMacAndNamespace defines static ip address and namespace, mac address network annotation
// for pod object.
func StaticIPAnnotationWithMacAndNamespace(name, namespace, macAddr string) []*multus.NetworkSelectionElement {
	baseAnnotation := StaticIPAnnotationWithNamespace(name, namespace, nil)
	baseAnnotation[0].MacRequest = macAddr

	return baseAnnotation
}

// StaticIPAnnotationWithInterfaceMacAndNamespace defines static ip address and namespace, interface name,
// mac address network annotation for pod object.
func StaticIPAnnotationWithInterfaceMacAndNamespace(
	name, namespace, intName, macAddr string) []*multus.NetworkSelectionElement {
	baseAnnotation := StaticIPAnnotationWithMacAndNamespace(name, namespace, macAddr)
	baseAnnotation[0].InterfaceRequest = intName

	return baseAnnotation
}

// StaticIPBondAnnotationWithInterface defines static name for bonded interfaces and name, interface and IP for the
// main bond int.
func StaticIPBondAnnotationWithInterface(
	bondNadName, bondIntName string, sriovNetworkNameList, ipAddrBond []string) []*multus.NetworkSelectionElement {
	annotation := []*multus.NetworkSelectionElement{}
	for _, sriovNetName := range sriovNetworkNameList {
		annotation = append(annotation, StaticAnnotation(sriovNetName))
	}

	bond := StaticIPAnnotation(bondNadName, ipAddrBond)
	bond[0].InterfaceRequest = bondIntName

	return append(annotation, bond[0])
}

// StaticIPMultiNetDualStackAnnotation defines network annotation for multiple interfaces with dual stack addresses.
func StaticIPMultiNetDualStackAnnotation(sriovNets, ipAddr []string) ([]*multus.NetworkSelectionElement, error) {
	if len(sriovNets) == 0 {
		glog.V(100).Infof("sriovNets cannot be empty")

		return nil, fmt.Errorf("sriovNets []string cannot be empty")
	}

	annotation := []*multus.NetworkSelectionElement{}

	// Verify ipAddr has an even number of IP addresses and not empty.
	if len(ipAddr) == 0 || len(ipAddr)%2 != 0 {
		glog.V(100).Infof("ipAddr needs to contain an even number of IP addresses")

		return nil, fmt.Errorf("ipAddr []string cannot be empy or an odd number")
	}

	for i, sriovNetName := range sriovNets {
		if i*2+1 < len(ipAddr) {
			annotation = append(annotation, StaticIPAnnotation(sriovNetName, []string{ipAddr[i*2], ipAddr[i*2+1]})...)
		}
	}

	return annotation, nil
}
