package cmd

import (
	"fmt"
	"net"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
)

// ICMPConnectivityCheck checks ping against provided IPs from the client pod.
func ICMPConnectivityCheck(clientPod *pod.Builder, destIPAddresses []string) error {
	glog.V(90).Infof("Checking ping against %v from the client pod %s",
		destIPAddresses, clientPod.Definition.Name)

	for _, destIPAddress := range destIPAddresses {
		ipAddress, _, err := net.ParseCIDR(destIPAddress)
		if err != nil {
			return fmt.Errorf("invalid IP address: %s", destIPAddress)
		}

		TestCmdIcmpCommand := fmt.Sprintf("ping %s -c 5", ipAddress.String())
		if ipAddress.To4() == nil {
			TestCmdIcmpCommand = fmt.Sprintf("ping -6 %s -c 5", ipAddress.String())
		}

		output, err := clientPod.ExecCommand([]string{"bash", "-c", TestCmdIcmpCommand})
		if err != nil {
			return fmt.Errorf("ICMP connectivity failed: %s", output.String())
		}
	}

	return nil
}
