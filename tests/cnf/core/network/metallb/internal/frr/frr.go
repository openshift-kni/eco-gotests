package frr

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/golang/glog"

	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netparam"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/tsparams"
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

	advertisedRoute struct {
		AddrPrefix    string `json:"addrPrefix"`
		PrefixLen     int    `json:"prefixLen"`
		Network       string `json:"network"`
		NextHop       string `json:"nextHop"`
		Metric        int    `json:"metric"`
		LocPrf        int    `json:"locPrf"`
		Weight        int    `json:"weight"`
		Path          string `json:"path"`
		BGPOriginCode string `json:"bgpOriginCode"`
	}
	// BgpAdvertisedRoutes creates struct from json output.
	BgpAdvertisedRoutes struct {
		BGPTableVersion  int                        `json:"bgpTableVersion"`
		BGPLocalRouterID string                     `json:"bgpLocalRouterId"`
		DefaultLocPrf    int                        `json:"defaultLocPrf"`
		LocalAS          int                        `json:"localAS"`
		AdvertisedRoutes map[string]advertisedRoute `json:"advertisedRoutes"`
	}
	// BgpReceivedRoutes struct includes Routes map.
	BgpReceivedRoutes struct {
		Routes map[string][]RouteInfo `json:"routes"`
	}
	// RouteInfo struct includes route info.
	RouteInfo struct {
		Prefix    string    `json:"prefix"`
		PrefixLen int       `json:"prefixLen"`
		Protocol  string    `json:"protocol"`
		Metric    int       `json:"metric"`
		Uptime    string    `json:"uptime"`
		Nexthops  []Nexthop `json:"nexthops"`
	}
	// Nexthop struct includes nexthop route info.
	Nexthop struct {
		Flags          int    `json:"flags"`
		Fib            bool   `json:"fib"`
		IP             string `json:"ip"`
		Afi            string `json:"afi"`
		InterfaceIndex int    `json:"interfaceIndex"`
		InterfaceName  string `json:"interfaceName"`
		Active         bool   `json:"active"`
		Weight         int    `json:"weight"`
	}
	// BGPConnectionInfo struct includes the connectRetryTimer.
	BGPConnectionInfo struct {
		ConnectRetryTimer int `json:"connectRetryTimer"`
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
	stdout, err := frrPod.ExecCommand([]string{"curl", "localhost:7573/metrics"}, "frr")

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

// FetchBGPConnectTimeValue fetches and returns the ConnectRetryTimer value for the specified BGP peer.
func FetchBGPConnectTimeValue(frrk8sPods []*pod.Builder, bgpPeerIP string) (int, error) {
	for _, frrk8sPod := range frrk8sPods {
		// Run the "show bgp neighbor <bgpPeerIP> json" command on each pod
		output, err := frrk8sPod.ExecCommand(append(netparam.VtySh,
			fmt.Sprintf("show bgp neighbor %s json", bgpPeerIP)), "frr")
		if err != nil {
			return 0, fmt.Errorf("error collecting BGP neighbor info from pod %s: %w",
				frrk8sPod.Definition.Name, err)
		}

		// Parsing JSON
		var bgpData map[string]BGPConnectionInfo
		err = json.Unmarshal(output.Bytes(), &bgpData)

		if err != nil {
			return 0, fmt.Errorf("error parsing BGP neighbor JSON for pod %s: %w", frrk8sPod.Definition.Name, err)
		}

		// Extracting ConnectRetryTimer from the parsed JSON
		for _, bgpInfo := range bgpData {
			return bgpInfo.ConnectRetryTimer, nil
		}
	}

	return 0, fmt.Errorf("no BGP neighbor data found for peer %s", bgpPeerIP)
}

// ResetBGPConnection restarts the TCP connection.
func ResetBGPConnection(frrPod *pod.Builder) error {
	glog.V(90).Infof("Resetting BGP session to all neighbors: %s", frrPod.Definition.Name)

	_, err := frrPod.ExecCommand(append(netparam.VtySh, "clear ip bgp *"))

	return err
}
