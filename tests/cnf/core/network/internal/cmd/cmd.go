package cmd

import (
	"fmt"
	"net"

	. "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netinittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netparam"

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

// RunCommandOnHostNetworkPod creates hostNetwork pod  and executes given command on it.
// The Pod will be removed at the end.
func RunCommandOnHostNetworkPod(nodeName, namespace, command string) (string, error) {
	glog.V(90).Infof("Running command %s on the host network pod on node %s",
		command, nodeName)

	testPod, err := pod.NewBuilder(APIClient, "hostnetworkpod", namespace, NetConfig.CnfNetTestContainer).
		DefineOnNode(nodeName).WithPrivilegedFlag().WithHostNetwork().CreateAndWaitUntilRunning(netparam.DefaultTimeout)
	if err != nil {
		return "", err
	}

	output, err := testPod.ExecCommand([]string{"bash", "-c", command})
	if err != nil {
		return "", err
	}

	_, err = testPod.DeleteAndWait(netparam.DefaultTimeout)
	if err != nil {
		return output.String(), err
	}

	return output.String(), nil
}
