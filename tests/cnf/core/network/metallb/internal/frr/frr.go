package frr

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/openshift-kni/eco-gotests/pkg/pod"
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

func runningConfig(frrPod *pod.Builder) (string, error) {
	bgpStateOut, err := frrPod.ExecCommand(append(tsparams.VtySh, "sh run"), tsparams.FRRContainerName)
	if err != nil {
		return "", fmt.Errorf("error collecting frr running config from pod %s due to %w",
			frrPod.Definition.Name, err)
	}

	return bgpStateOut.String(), nil
}
