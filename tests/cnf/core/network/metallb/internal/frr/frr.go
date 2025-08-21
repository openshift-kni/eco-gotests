package frr

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/golang/glog"

	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/pod"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/core/network/internal/netparam"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/core/network/metallb/internal/tsparams"
)

type (
	bgpDescription struct {
		BGPState string `json:"bgpState"`
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

	// BGPConnectionInfo struct includes the RemoteAS.
	BGPConnectionInfo struct {
		RemoteAS int `json:"remoteAS"`
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

// DefineBGPConfig returns string which represents BGP config file peering to all given IP addresses.
func DefineBGPConfig(localBGPASN, remoteBGPASN int, neighborsIPAddresses []string, multiHop, bfd bool) string {
	bgpConfig := tsparams.FRRBaseConfig +
		fmt.Sprintf("router bgp %d\n", localBGPASN) +
		tsparams.FRRDefaultBGPPreConfig

	for _, ipAddress := range neighborsIPAddresses {
		bgpConfig += fmt.Sprintf("  neighbor %s remote-as %d\n  neighbor %s password %s\n",
			ipAddress, remoteBGPASN, ipAddress, tsparams.BGPPassword)

		if bfd {
			bgpConfig += fmt.Sprintf("  neighbor %s bfd\n", ipAddress)
		}

		if multiHop {
			bgpConfig += fmt.Sprintf("  neighbor %s ebgp-multihop 2\n", ipAddress)
		}
	}

	bgpConfig += "!\naddress-family ipv4 unicast\n"
	for _, ipAddress := range neighborsIPAddresses {
		bgpConfig += fmt.Sprintf("  neighbor %s activate\n", ipAddress)
	}

	bgpConfig += "exit-address-family\n!\naddress-family ipv6 unicast\n"
	for _, ipAddress := range neighborsIPAddresses {
		bgpConfig += fmt.Sprintf("  neighbor %s activate\n", ipAddress)
	}

	bgpConfig += "exit-address-family\n!\nline vty\n!\nend\n"

	return bgpConfig
}

// BGPNeighborshipHasState verifies that BGP session on a pod has given state.
func BGPNeighborshipHasState(frrPod *pod.Builder, neighborIPAddress string, state string) (bool, error) {
	var result map[string]bgpDescription

	bgpStateOut, err := frrPod.ExecCommand(append(netparam.VtySh, "sh bgp neighbors json"))
	if err != nil {
		return false, err
	}

	err = json.Unmarshal(bgpStateOut.Bytes(), &result)
	if err != nil {
		return false, err
	}

	return result[neighborIPAddress].BGPState == state, nil
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

// ValidateBGPRemoteAS validates the remoteAS value for the specified BGP peer across all Speaker pods.
func ValidateBGPRemoteAS(speakerPods []*pod.Builder, bgpPeerIP string, expectedRemoteAS int) error {
	glog.V(90).Infof("Validating the frr nodes receive the correct remote bgp peer AS : %d", expectedRemoteAS)

	for _, speakerPod := range speakerPods {
		// Run the "show bgp neighbor <bgpPeerIP> json" command on each pod
		output, err := speakerPod.ExecCommand(append(netparam.VtySh,
			fmt.Sprintf("show bgp neighbor %s json", bgpPeerIP)), "frr")
		if err != nil {
			return fmt.Errorf("error collecting BGP neighbor info from pod %s: %w",
				speakerPod.Definition.Name, err)
		}

		// Parsing JSON
		var bgpData map[string]BGPConnectionInfo
		err = json.Unmarshal(output.Bytes(), &bgpData)

		if err != nil {
			return fmt.Errorf("error parsing BGP neighbor JSON for pod %s: %w", speakerPod.Definition.Name, err)
		}

		// Validate RemoteAS
		for _, bgpInfo := range bgpData {
			if bgpInfo.RemoteAS == expectedRemoteAS {
				return nil // Match found
			}
		}
	}

	// If no matches are found across all pods
	return fmt.Errorf("no BGP neighbor with RemoteAS %d found for peer %s", expectedRemoteAS, bgpPeerIP)
}

// DefineBGPConfigWithStaticRouteAndNetwork defines BGP config file with static route and network.
func DefineBGPConfigWithStaticRouteAndNetwork(localBGPASN, remoteBGPASN int, hubPodIPs,
	advertisedIPv4Routes, advertisedIPv6Routes, neighborsIPAddresses []string,
	multiHop, bfd bool) string {
	bgpConfig := tsparams.FRRBaseConfig +
		fmt.Sprintf("ip route %s/32 %s\n", neighborsIPAddresses[1], hubPodIPs[0]) +
		fmt.Sprintf("ip route %s/32 %s\n!\n", neighborsIPAddresses[0], hubPodIPs[1]) +
		fmt.Sprintf("router bgp %d\n", localBGPASN) +
		tsparams.FRRDefaultBGPPreConfig

	for _, ipAddress := range neighborsIPAddresses {
		bgpConfig += fmt.Sprintf("  neighbor %s remote-as %d\n  neighbor %s password %s\n",
			ipAddress, remoteBGPASN, ipAddress, tsparams.BGPPassword)

		if bfd {
			bgpConfig += fmt.Sprintf("  neighbor %s bfd\n", ipAddress)
		}

		if multiHop {
			bgpConfig += fmt.Sprintf("  neighbor %s ebgp-multihop 2\n", ipAddress)
		}
	}

	bgpConfig += "!\naddress-family ipv4 unicast\n"
	for _, ipAddress := range neighborsIPAddresses {
		bgpConfig += fmt.Sprintf("  neighbor %s activate\n", ipAddress)
	}

	bgpConfig += fmt.Sprintf("  network %s\n", advertisedIPv4Routes[0])
	bgpConfig += fmt.Sprintf("  network %s\n", advertisedIPv4Routes[1])
	bgpConfig += "exit-address-family\n"

	// Add network commands only once for IPv6
	bgpConfig += "!\naddress-family ipv6 unicast\n"
	for _, ipAddress := range neighborsIPAddresses {
		bgpConfig += fmt.Sprintf("  neighbor %s activate\n", ipAddress)
	}

	bgpConfig += fmt.Sprintf("  network %s\n", advertisedIPv6Routes[0])
	bgpConfig += fmt.Sprintf("  network %s\n", advertisedIPv6Routes[1])
	bgpConfig += "exit-address-family\n"

	bgpConfig += "!\nline vty\n!\nend\n"

	return bgpConfig
}

func getBgpStatus(frrPod *pod.Builder, cmd string, containerName ...string) (*bgpStatus, error) {
	cName := "frr"

	if len(containerName) > 0 {
		cName = containerName[0]
	}

	glog.V(90).Infof("Getting bgp status from container: %s of pod: %s", cName, frrPod.Definition.Name)

	bgpStateOut, err := frrPod.ExecCommand(append(netparam.VtySh, cmd))

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
	bgpStateOut, err := frrPod.ExecCommand(append(netparam.VtySh, "sh run"), tsparams.FRRContainerName)
	if err != nil {
		return "", fmt.Errorf("error collecting frr running config from pod %s due to %w",
			frrPod.Definition.Name, err)
	}

	return bgpStateOut.String(), nil
}
