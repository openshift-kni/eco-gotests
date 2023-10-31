package frr

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/golang/glog"

	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/tsparams"
)

type (
	bgpDescription struct {
		BGPState string `json:"bgpState"`
	}

	bfdDescription struct {
		BFDStatus string `json:"status"`
		BFDPeer   string `json:"peer"`
	}

	bgpStatus struct {
		VrfID         int    `json:"vrfId"`
		VrfName       string `json:"vrfName"`
		TableVersion  int    `json:"tableVersion"`
		RouterID      string `json:"routerId"`
		DefaultLocPrf int    `json:"defaultLocPrf"`
		LocalAS       int    `json:"localAS"`
		Routes        map[string][]struct {
			Valid     bool   `json:"valid"`
			Multipath bool   `json:"multipath,omitempty"`
			PathFrom  string `json:"pathFrom"`
			Prefix    string `json:"prefix"`
			PrefixLen int    `json:"prefixLen"`
			Network   string `json:"network"`
			Metric    int    `json:"metric"`
			Weight    int    `json:"weight"`
			PeerID    string `json:"peerId"`
			Path      string `json:"path"`
			Origin    string `json:"origin"`
			Nexthops  []struct {
				IP       string `json:"ip"`
				Hostname string `json:"hostname"`
				Afi      string `json:"afi"`
				Used     bool   `json:"used"`
			} `json:"nexthops"`
			Bestpath bool `json:"bestpath,omitempty"`
		} `json:"routes"`
	}
)

// DefineBaseConfig defines minimal required FRR configuration.
func DefineBaseConfig(daemonsConfig, frrConfig, vtyShConfig string) map[string]string {
	configMapData := make(map[string]string)
	configMapData["daemons"] = daemonsConfig
	configMapData["frr.conf"] = frrConfig
	configMapData["vtysh.conf"] = vtyShConfig

	return configMapData
}

// DefineBFDConfig returns string which represents BFD config file peering to all given IP addresses.
func DefineBFDConfig(localBGPASN, remoteBGPASN int, neighborsIPAddresses []string, multiHop bool) string {
	bfdConfig := tsparams.FRRBaseConfig +
		fmt.Sprintf("router bgp %d\n", localBGPASN) +
		tsparams.FRRDefaultBGPPreConfig

	for _, ipAddress := range neighborsIPAddresses {
		bfdConfig += fmt.Sprintf("  neighbor %s remote-as %d\n  neighbor %s bfd\n  neighbor %s password %s\n",
			ipAddress, remoteBGPASN, ipAddress, ipAddress, tsparams.BGPPassword)
		if multiHop {
			bfdConfig += fmt.Sprintf("  neighbor %s ebgp-multihop 2\n", ipAddress)
		}
	}

	bfdConfig += "!\naddress-family ipv4 unicast\n"
	for _, ipAddress := range neighborsIPAddresses {
		bfdConfig += fmt.Sprintf("  neighbor %s activate\n", ipAddress)
	}

	bfdConfig += "exit-address-family\n!\naddress-family ipv6 unicast\n"
	for _, ipAddress := range neighborsIPAddresses {
		bfdConfig += fmt.Sprintf("  neighbor %s activate\n", ipAddress)
	}

	bfdConfig += "exit-address-family\n!\nline vty\n!\nend\n"

	return bfdConfig
}

// BGPNeighborshipHasState verifies that BGP session on a pod has given state.
func BGPNeighborshipHasState(frrPod *pod.Builder, neighborIPAddress string, state string) (bool, error) {
	var result map[string]bgpDescription

	bgpStateOut, err := frrPod.ExecCommand(append(tsparams.VtySh, "sh bgp neighbors json"))
	if err != nil {
		return false, err
	}

	err = json.Unmarshal(bgpStateOut.Bytes(), &result)
	if err != nil {
		return false, err
	}

	return result[neighborIPAddress].BGPState == state, nil
}

// BFDHasStatus verifies that BFD session on a pod has given status.
func BFDHasStatus(frrPod *pod.Builder, bfdPeer string, status string) error {
	bfdStatusOut, err := frrPod.ExecCommand(append(tsparams.VtySh, "sh bfd peers brief json"))
	if err != nil {
		return err
	}

	var result []bfdDescription

	err = json.Unmarshal(bfdStatusOut.Bytes(), &result)
	if err != nil {
		return err
	}

	for _, peer := range result {
		if peer.BFDPeer == bfdPeer && peer.BFDStatus != status {
			return fmt.Errorf("%s bfd status is %s (expected %s)", peer.BFDPeer, peer.BFDStatus, status)
		}
	}

	return nil
}

// IsProtocolConfigured verifies that given protocol is set in frr config.
func IsProtocolConfigured(frrPod *pod.Builder, protocol string) (bool, error) {
	frrConf, err := runningConfig(frrPod)
	if err != nil {
		return false, err
	}

	frrConfList := strings.Split(frrConf, "!")
	for _, configLine := range frrConfList {
		if strings.HasPrefix(strings.TrimSpace(configLine), protocol) {
			return true, nil
		}
	}

	return false, nil
}

// GetMetricsByPrefix pulls all metrics from frr pods and sort them in the list by given prefix.
func GetMetricsByPrefix(frrPod *pod.Builder, metricPrefix string) ([]string, error) {
	stdout, err := frrPod.ExecCommand([]string{"curl", "localhost:29151/metrics"}, "frr")

	if err != nil {
		return nil, err
	}

	if len(strings.Split(stdout.String(), "\n")) == 0 {
		return nil, fmt.Errorf("failed to collect metrics due to empty response")
	}

	var collectedMetrics []string

	for _, line := range strings.Split(stdout.String(), "\n") {
		if strings.HasPrefix(line, metricPrefix) {
			metricsKey := line[0:strings.Index(line, "{")]
			collectedMetrics = append(collectedMetrics, metricsKey)
		}
	}

	if len(collectedMetrics) < 1 {
		return nil, fmt.Errorf("failed to collect metrics")
	}

	return collectedMetrics, nil
}

// SetStaticRoute could set or delete static route on all Speaker pods.
func SetStaticRoute(frrPod *pod.Builder, action, destIP string, nextHopMap map[string]string) (string, error) {
	buffer, err := frrPod.ExecCommand(
		[]string{"ip", "route", action, destIP, "via", nextHopMap[frrPod.Definition.Spec.NodeName]}, "frr")
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

// GetBGPStatus returns bgp status output from frr pod.
func GetBGPStatus(frrPod *pod.Builder, protocolVersion string, containerName ...string) (*bgpStatus, error) {
	glog.V(90).Infof("Getting bgp status from pod: %s", frrPod.Definition.Name)

	return getBgpStatus(frrPod, fmt.Sprintf("show bgp %s json", protocolVersion), containerName...)
}

// GetBGPCommunityStatus returns bgp community status from frr pod.
func GetBGPCommunityStatus(frrPod *pod.Builder, ipProtocolVersion string) (*bgpStatus, error) {
	glog.V(90).Infof("Getting bgp community status from container on pod: %s", frrPod.Definition.Name)

	return getBgpStatus(frrPod, fmt.Sprintf("show bgp %s community %s json", ipProtocolVersion, "65535:65282"))
}

func getBgpStatus(frrPod *pod.Builder, cmd string, containerName ...string) (*bgpStatus, error) {
	cName := "frr"

	if len(containerName) > 0 {
		cName = containerName[0]
	}

	glog.V(90).Infof("Getting bgp status from container: %s of pod: %s", cName, frrPod.Definition.Name)

	bgpStateOut, err := frrPod.ExecCommand(append(tsparams.VtySh, cmd))

	if err != nil {
		return nil, err
	}

	bgpStatus := bgpStatus{}

	err = json.Unmarshal(bgpStateOut.Bytes(), &bgpStatus)
	if err != nil {
		glog.V(90).Infof("Failed to Unmarshal bgpStatus string: %s in to bgpStatus struct", bgpStateOut.String())

		return nil, err
	}

	if len(bgpStatus.Routes) == 0 {
		return nil, fmt.Errorf("no bgp routes present BGP status is empty")
	}

	return &bgpStatus, nil
}

func runningConfig(frrPod *pod.Builder) (string, error) {
	bgpStateOut, err := frrPod.ExecCommand(append(tsparams.VtySh, "sh run"), tsparams.FRRContainerName)
	if err != nil {
		return "", fmt.Errorf("error collecting frr running config from pod %s due to %w",
			frrPod.Definition.Name, err)
	}

	return bgpStateOut.String(), nil
}
