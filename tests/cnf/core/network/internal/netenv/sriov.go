package netenv

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/mco"
	"github.com/openshift-kni/eco-goinfra/pkg/sriov"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WaitForSriovAndMCPStable waits until SR-IOV and MCP stable.
func WaitForSriovAndMCPStable(
	apiClient *clients.Settings, waitingTime, stableDuration time.Duration, mcpName, sriovOperatorNamespace string) error {
	glog.V(90).Infof("Waiting for SR-IOV and MCP become stable.")
	time.Sleep(10 * time.Second)

	err := WaitForSriovStable(apiClient, waitingTime, sriovOperatorNamespace)
	if err != nil {
		return err
	}

	err = WaitForMcpStable(apiClient, waitingTime, stableDuration, mcpName)
	if err != nil {
		return err
	}

	return nil
}

// WaitForSriovStable waits until all the SR-IOV node states are in sync.
func WaitForSriovStable(apiClient *clients.Settings, waitingTime time.Duration, sriovOperatorNamespace string) error {
	networkNodeStateList, err := sriov.ListNetworkNodeState(apiClient, sriovOperatorNamespace, v1.ListOptions{})

	if err != nil {
		return fmt.Errorf("failed to fetch nodes state %w", err)
	}

	if len(networkNodeStateList) == 0 {
		return nil
	}

	for _, nodeState := range networkNodeStateList {
		err = nodeState.WaitUntilSyncStatus("Succeeded", waitingTime)
		if err != nil {
			return err
		}
	}

	return nil
}

// WaitForMcpStable waits for the stability of the MCP with the given name.
func WaitForMcpStable(apiClient *clients.Settings, waitingTime, stableDuration time.Duration, mcpName string) error {
	mcp, err := mco.Pull(apiClient, mcpName)

	if err != nil {
		return fmt.Errorf("fail to pull mcp %s from cluster due to: %s", mcpName, err.Error())
	}

	err = mcp.WaitToBeStableFor(stableDuration, waitingTime)

	if err != nil {
		return fmt.Errorf("cluster is not stable: %s", err.Error())
	}

	return nil
}
