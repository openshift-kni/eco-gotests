package pod

import (
	multus "gopkg.in/k8snetworkplumbingwg/multus-cni.v4/pkg/types"
)

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
	baseAnnotation := StaticIPAnnotation(name, ipAddr)
	baseAnnotation[0].InterfaceRequest = intName
	baseAnnotation[0].Namespace = namespace

	return baseAnnotation
}
