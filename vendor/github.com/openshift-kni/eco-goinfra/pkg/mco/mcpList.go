package mco

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"

	mcov1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
)

// MCPListBuilder provides struct for MachineConfigPoolList object which contains connection to cluster
// and MachineConfigPoolList definitions.
type MCPListBuilder struct {
	// MachineConfigPoolList definition. Used to create MachineConfigPoolList object with minimum
	// set of required elements.
	ObjectList *mcov1.MachineConfigPoolList
	apiClient  *clients.Settings
	// machineConfigSelector label.
	mcSelector string
}

// NewMCPListBuilder method creates new instance of MCPListBuilder.
func NewMCPListBuilder(apiClient *clients.Settings, mcpLabelSelector ...map[string]string) *MCPListBuilder {
	glog.V(100).Infof(
		"Initializing new MCPListBuilder structure with the following params: %v", mcpLabelSelector)

	builder := &MCPListBuilder{
		apiClient: apiClient,
	}

	if len(mcpLabelSelector) > 0 {
		// Serialize selector
		serialSelector := labels.Set(mcpLabelSelector[0]).String()
		builder.mcSelector = serialSelector
		glog.V(100).Infof("NewMCPListBuilder builder.mcSelector is: %v", builder.mcSelector)
	}

	return builder
}

// Discover method gets the MachineConfigPools in cluster and stores them in the builder struct.
func (builder *MCPListBuilder) Discover() error {
	glog.V(100).Infof("Getting the list of MachineConfigPool objects on this cluster")

	var (
		mcpList *mcov1.MachineConfigPoolList
		err     error
	)

	if builder.mcSelector != "" {
		mcpList, err = builder.apiClient.MachineConfigPools().List(
			context.TODO(), metav1.ListOptions{LabelSelector: builder.mcSelector})
	} else {
		mcpList, err = builder.apiClient.MachineConfigPools().List(
			context.TODO(), metav1.ListOptions{})
	}

	if err != nil {
		glog.V(100).Infof("Error to list MachineConfigPools")

		return err
	}

	if len(mcpList.Items) < 1 {
		glog.V(100).Infof("Cluster doesn't have MachineConfigPools installed ")

		return fmt.Errorf("MacineConfigPool list is empty")
	}

	builder.ObjectList = mcpList

	for _, mcp := range mcpList.Items {
		glog.V(100).Infof("builder Discover() MachineConfigPoolList contents: %v", mcp.ObjectMeta.Name)
	}

	return err
}

// WaitToBeStableFor waits on all MachineConfigPools in a MachineConfigConfigPoolList to be
// stable for a time duration up to the timeout.
func (builder *MCPListBuilder) WaitToBeStableFor(stableDuration time.Duration, timeout time.Duration) error {
	glog.V(100).Infof("WaitForMcpListToBeStableFor waits up to duration of %v for "+
		"MachineConfigPoolList to be stable for %v", timeout, stableDuration)

	isMcpListStable := true

	// Wait 5 secs in each iteration before condition function () returns true or errors or times out
	// after stableDuration
	err := wait.PollImmediate(fiveScds, timeout, func() (bool, error) {

		isMcpListStable = true

		// check if cluster is stable every 5 seconds during entire stableDuration time period
		// Here we need to run through the entire stableDuration till it times out.
		_ = wait.PollImmediate(fiveScds, stableDuration, func() (done bool, err error) {

			err = builder.Discover()

			if err != nil {
				return false, err
			}

			// iterate through the MachineConfigPools in the list.
			for _, mcp := range builder.ObjectList.Items {
				if mcp.Status.ReadyMachineCount != mcp.Status.MachineCount ||
					mcp.Status.MachineCount != mcp.Status.UpdatedMachineCount ||
					mcp.Status.DegradedMachineCount != 0 {
					isMcpListStable = false

					glog.V(100).Infof("MachineConfigPool: %v degraded and has a mismatch in "+
						"machineCount: %v "+"vs machineCountUpdated: "+"%v vs readyMachineCount: %v and "+
						"degradedMachineCount is : %v \n", mcp.ObjectMeta.Name,
						mcp.Status.MachineCount, mcp.Status.UpdatedMachineCount,
						mcp.Status.ReadyMachineCount, mcp.Status.DegradedMachineCount)

					return true, err
				}
			}

			// Here we are always returning "false, nil" so we keep iterating throughout the stableInterval
			// of the inner wait.pollImmediate loop, until we time out.
			return false, nil
		})

		if isMcpListStable {
			glog.V(100).Infof("MachineConfigPools were stable during during stableDuration: %v",
				stableDuration)

			// exit the outer wait.PollImmediate block since the mcps were stable during stableDuration.
			return true, nil
		}

		glog.V(100).Infof("MachineConfigPools were not stable during stableDuration: %v, retrying ...",
			stableDuration)

		// keep iterating in the outer wait.PollImmediate waiting for cluster to be stable.
		return false, nil

	})

	if err == nil {
		glog.V(100).Infof("Cluster was stable during stableDuration: %v", stableDuration)
	} else {
		// Here err is "timed out waiting for the condition"
		glog.V(100).Infof("Cluster was Un-stable during stableDuration: %v", stableDuration)
	}

	return err
}

// GetByLabel returns all MachineConfigPools with the specified label.
func (builder *MCPListBuilder) GetByLabel(mcpLabel string) (mcov1.MachineConfigPool, error) {
	glog.V(100).Infof("GetByLabel returns all MachineConfigPools with the specified label: %v", mcpLabel)

	mcpList, err := builder.apiClient.MachineConfigPools().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return mcov1.MachineConfigPool{}, err
	}

	for _, mcp := range mcpList.Items {
		for _, label := range mcp.Spec.MachineConfigSelector.MatchExpressions {
			for _, value := range label.Values {
				if value == mcpLabel {
					return mcp, nil
				}
			}
		}

		for _, label := range mcp.Spec.MachineConfigSelector.MatchLabels {
			if label == mcpLabel {
				return mcp, nil
			}
		}
	}

	return mcov1.MachineConfigPool{}, fmt.Errorf("cannot find MachineConfigPool"+
		" that targets machineConfig with label: %s", mcpLabel)
}
