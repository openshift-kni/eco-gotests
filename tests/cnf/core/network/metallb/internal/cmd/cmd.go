package cmd

import (
	"bytes"
	"fmt"

	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netparam"

	"github.com/openshift-kni/eco-goinfra/pkg/pod"
)

// DefineNGNXAndSleep runs NGNX server.
func DefineNGNXAndSleep() []string {
	return []string{"/bin/bash", "-c", "nginx && sleep INF"}
}

// DefineRouteAndSleep creates route and appends sleep CMD.
func DefineRouteAndSleep(dstNet, nextHop string) []string {
	cmd := DefineRoute(dstNet, nextHop)
	cmd[2] += " && sleep INF"

	return cmd
}

// DefineRoute creates route.
func DefineRoute(dstNet, nextHop string) []string {
	return []string{"/bin/bash", "-c", fmt.Sprintf("ip route add %s via %s", dstNet, nextHop)}
}

// SetRouteOnPod adds route to the given pod.
func SetRouteOnPod(client *pod.Builder, dstNet, nextHop string) (bytes.Buffer, error) {
	cmd := DefineRoute(dstNet, nextHop)

	return client.ExecCommand(cmd)
}

// Curl exec curl cmd inside given pod.
func Curl(client *pod.Builder, sourceIPAddr, destIPAddr, ipFamily, containerName string) (string, error) {
	command := fmt.Sprintf("curl --interface %s %s --max-time 10", sourceIPAddr, destIPAddr)

	if ipFamily == netparam.IPV6Family {
		command = fmt.Sprint("curl --interface ", sourceIPAddr, "[", destIPAddr, "]", "--max-time 5")
	}

	curlStatus, err := client.ExecCommand([]string{"bash", "-c", command}, containerName)

	if err != nil {
		return curlStatus.String(), fmt.Errorf("curl command failed - %w", err)
	}

	return curlStatus.String(), nil
}
