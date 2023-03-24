package pod

import (
	multus "gopkg.in/k8snetworkplumbingwg/multus-cni.v3/pkg/types"
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
