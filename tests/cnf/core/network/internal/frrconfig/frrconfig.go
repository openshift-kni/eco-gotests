package frrconfig

import (
	"fmt"
	"strings"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/ipaddr"
)

// SetStaticRoute could set or delete static route on all Speaker pods.
func SetStaticRoute(frrPod *pod.Builder, action, destIP, containerName string,
	nextHopMap map[string]string) (string, error) {
	buffer, err := frrPod.ExecCommand(
		[]string{"ip", "route", action, destIP, "via", nextHopMap[frrPod.Definition.Spec.NodeName]}, containerName)
	if err != nil {
		if strings.Contains(buffer.String(), "File exists") {
			glog.V(90).Infof("Warning: Route to %s already exist", destIP)

			return buffer.String(), nil
		}

		if strings.Contains(buffer.String(), "No such process") {
			glog.V(90).Infof("Warning: Route to %s already absent", destIP)

			return buffer.String(), nil
		}

		return buffer.String(), err
	}

	return buffer.String(), nil
}

// BuildRoutesMapWithSpecificRoutes creates a route map with specific routes.
func BuildRoutesMapWithSpecificRoutes(podList []*pod.Builder, workerNodeList []*nodes.Builder,
	nextHopList []string) (map[string]string, error) {
	if len(podList) == 0 {
		glog.Error("Pod list is empty")

		return nil, fmt.Errorf("pod list is empty")
	}

	if len(nextHopList) == 0 {
		glog.Error("Nexthop IP addresses list is empty")

		return nil, fmt.Errorf("nexthop IP addresses list is empty")
	}

	if len(nextHopList) < len(podList) {
		glog.Errorf("Number of speaker IP addresses[%d] is less than the number of pods[%d]", len(nextHopList), len(podList))

		return nil, fmt.Errorf("insufficient speaker IP addresses: got %d, need at least %d", len(nextHopList), len(podList))
	}

	routesMap := make(map[string]string)

	for _, frrPod := range podList {
		if frrPod.Definition.Spec.NodeName == workerNodeList[0].Definition.Name {
			routesMap[frrPod.Definition.Spec.NodeName] = nextHopList[1]
		} else {
			routesMap[frrPod.Definition.Spec.NodeName] = nextHopList[0]
		}
	}

	return routesMap, nil
}

// DefineBaseConfig creates a map of strings for the frr configuration.
func DefineBaseConfig(daemonsConfig, frrConfig, vtyShConfig string) map[string]string {
	configMapData := make(map[string]string)
	configMapData["daemons"] = daemonsConfig
	configMapData["frr.conf"] = frrConfig
	configMapData["vtysh.conf"] = vtyShConfig

	return configMapData
}

// RemovePrefixFromIPList removes the prefix from an IP address.
func RemovePrefixFromIPList(ipAddressList []string) []string {
	var ipAddressListWithoutPrefix []string
	for _, ipaddress := range ipAddressList {
		ipAddressListWithoutPrefix = append(ipAddressListWithoutPrefix, ipaddr.RemovePrefix(ipaddress))
	}

	return ipAddressListWithoutPrefix
}
