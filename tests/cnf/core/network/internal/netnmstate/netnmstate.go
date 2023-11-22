package netnmstate

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/golang/glog"
	nmstateShared "github.com/nmstate/kubernetes-nmstate/api/shared"
	"github.com/openshift-kni/eco-goinfra/pkg/daemonset"
	"github.com/openshift-kni/eco-goinfra/pkg/deployment"
	"github.com/openshift-kni/eco-goinfra/pkg/nmstate"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	. "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netinittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netparam"

	"k8s.io/apimachinery/pkg/util/wait"
)

// CreateNewNMStateAndWaitUntilItsRunning creates or recreates the new NMState instance and waits until
// all daemonsets and deployments are in Ready state.
func CreateNewNMStateAndWaitUntilItsRunning(timeout time.Duration) error {
	glog.V(90).Infof("Verifying if NMState instance is installed")

	nmstateInstance, err := nmstate.PullNMstate(APIClient, netparam.NMState)

	if err == nil {
		glog.V(90).Infof("NMState exists. Removing NMState.")

		_, err = nmstateInstance.Delete()

		if err != nil {
			return err
		}
	}

	glog.V(90).Infof("Creating a new NMState instance.")

	_, err = nmstate.NewBuilder(APIClient, netparam.NMState).Create()
	if err != nil {
		return err
	}

	err = isNMStateDeployedAndReady(timeout)
	if err != nil {
		return err
	}

	return nil
}

// CreatePolicyAndWaitUntilItsAvailable creates NodeNetworkConfigurationPolicy and waits until
// it's successfully applied.
func CreatePolicyAndWaitUntilItsAvailable(timeout time.Duration, nmstatePolicy *nmstate.PolicyBuilder) error {
	glog.V(90).Infof("Creating an NMState policy and wait for its successful application.")

	nmstatePolicy, err := nmstatePolicy.Create()
	if err != nil {
		return err
	}

	glog.V(90).Infof("Waiting for the policy to reach the Progressing state.")

	err = nmstatePolicy.WaitUntilCondition(nmstateShared.NodeNetworkConfigurationPolicyConditionProgressing, timeout)
	if err != nil {
		return err
	}

	glog.V(90).Infof("Waiting for the policy to reach the Available state.")

	err = nmstatePolicy.WaitUntilCondition(nmstateShared.NodeNetworkConfigurationPolicyConditionAvailable, timeout)
	if err != nil {
		return err
	}

	return nil
}

// ConfigureVFsAndWaitUntilItsConfigured creates NodeNetworkConfigurationPolicy with VFs configuration and waits until
// it's successfully configured.
func ConfigureVFsAndWaitUntilItsConfigured(
	policyName string,
	sriovInterfaceName string,
	nodeLabel map[string]string,
	numberOfVFs uint8,
	timeout time.Duration) error {
	glog.V(90).Infof("Creating an NMState policy with VFs (%d) on interface %s"+
		" and wait for it to be successfully configured.", numberOfVFs, sriovInterfaceName)

	nmstatePolicy := nmstate.NewPolicyBuilder(
		APIClient, policyName, nodeLabel).WithInterfaceAndVFs(sriovInterfaceName, numberOfVFs)

	err := CreatePolicyAndWaitUntilItsAvailable(timeout, nmstatePolicy)
	if err != nil {
		return err
	}

	// NodeNetworkStates exist for each node and share the same name.
	nodeList, err := nodes.List(APIClient, metav1.ListOptions{LabelSelector: labels.Set(nodeLabel).String()})

	if err != nil {
		return err
	}

	for _, node := range nodeList {
		err = AreVFsCreated(node.Definition.Name, sriovInterfaceName, int(numberOfVFs))
		if err != nil {
			return err
		}
	}

	return nil
}

// AreVFsCreated verifies that the specified number of VFs has been created by NMState
// under the given interface.
func AreVFsCreated(nmstateName, sriovInterfaceName string, numberVFs int) error {
	glog.V(90).Infof("Verifying that NMState %s has requested the specified number of VFs (%d)"+
		" under the interface %s.", nmstateName, numberVFs, sriovInterfaceName)

	nodeNetworkState, err := nmstate.PullNodeNetworkState(APIClient, nmstateName)
	if err != nil {
		return err
	}

	numVFs, err := nodeNetworkState.GetTotalVFs(sriovInterfaceName)
	if err != nil {
		return err
	}

	if numVFs != numberVFs {
		return fmt.Errorf("not all VFs are configured, expected number: %d; actual number: %d", numberVFs, numVFs)
	}

	return nil
}

// GetPrimaryInterfaceBond returns master ovs system bond interface.
func GetPrimaryInterfaceBond(nodeName string) (string, error) {
	glog.V(90).Infof("Verifying that bond interface is a master ovs system interface on the node %s.", nodeName)

	nodeNetworkState, err := nmstate.PullNodeNetworkState(APIClient, nodeName)
	if err != nil {
		return "", err
	}

	ovsBridgeInterface, err := nodeNetworkState.GetInterfaceType("br-ex", "ovs-bridge")
	if err != nil {
		return "", err
	}

	for _, bridgePort := range ovsBridgeInterface.Bridge.Port {
		if strings.Contains(bridgePort["name"], "bond") {
			return bridgePort["name"], nil
		}
	}

	glog.V(90).Infof("There is no a bond interface in the br-ex bridge ports %v",
		ovsBridgeInterface.Bridge.Port)

	return "", nil
}

// GetBondSlaves returns slave ports under given Bond interface name.
func GetBondSlaves(bondName, nodeNetworkStateName string) ([]string, error) {
	glog.V(90).Infof("Getting slave ports under Bond interface %s", bondName)

	nodeNetworkState, err := nmstate.PullNodeNetworkState(APIClient, nodeNetworkStateName)
	if err != nil {
		return nil, err
	}

	bondInterface, err := nodeNetworkState.GetInterfaceType(bondName, "bond")
	if err != nil {
		return nil, err
	}

	return bondInterface.LinkAggregation.Port, nil
}

// GetBaseVlanInterface returns base interface under given Vlan interface name.
func GetBaseVlanInterface(vlanInterfaceName, nodeNetworkStateName string) (string, error) {
	glog.V(90).Infof("Getting base interface for Vlan interface %s", vlanInterfaceName)

	nodeNetworkState, err := nmstate.PullNodeNetworkState(APIClient, nodeNetworkStateName)
	if err != nil {
		return "", err
	}

	vlanInterface, err := nodeNetworkState.GetInterfaceType(vlanInterfaceName, "vlan")
	if err != nil {
		return "", err
	}

	return vlanInterface.Vlan.BaseIface, nil
}

func isNMStateDeployedAndReady(timeout time.Duration) error {
	glog.V(90).Infof("Checking that NMState deployments and daemonsets are ready.")

	var (
		nmstateHandlerDs         *daemonset.Builder
		nmstateWebhookDeployment *deployment.Builder
		nmstateCertDeployment    *deployment.Builder
		err                      error
	)

	glog.V(90).Infof("Pulling all NMState default daemonsets and deployments.")

	err = wait.PollUntilContextTimeout(
		context.TODO(), 5*time.Second, timeout, true, func(ctx context.Context) (bool, error) {
			nmstateHandlerDs, err = daemonset.Pull(
				APIClient, netparam.NMStateHandlerDsName, NetConfig.NMStateOperatorNamespace)
			if err != nil {
				glog.V(90).Infof("Error to pull daemonset %s from namespace %s, retry",
					netparam.NMStateHandlerDsName, NetConfig.NMStateOperatorNamespace)

				return false, nil
			}

			nmstateWebhookDeployment, err = deployment.Pull(
				APIClient, netparam.NMStateWebhookDeploymentName, NetConfig.NMStateOperatorNamespace)
			if err != nil {
				glog.V(90).Infof("Error to pull deployment %s namespace %s, retry",
					netparam.NMStateWebhookDeploymentName, NetConfig.NMStateOperatorNamespace)

				return false, nil
			}

			nmstateCertDeployment, err = deployment.Pull(
				APIClient, netparam.NMStateCertDeploymentName, NetConfig.NMStateOperatorNamespace)
			if err != nil {
				glog.V(90).Infof("Error to pull deployment %s namespace %s, retry",
					netparam.NMStateCertDeploymentName, NetConfig.NMStateOperatorNamespace)

				return false, nil
			}

			return true, nil
		})

	if err != nil {
		return err
	}

	glog.V(90).Infof("Waiting until all NMState resources are Ready.")

	if !nmstateHandlerDs.IsReady(timeout) {
		return fmt.Errorf("nmstate handler daemonset is not ready")
	}

	if !nmstateWebhookDeployment.IsReady(timeout) {
		return fmt.Errorf("nmstate webhook deployment is not ready")
	}

	if !nmstateCertDeployment.IsReady(timeout) {
		return fmt.Errorf("nmstate cert-manager deployment is not ready")
	}

	return nil
}

// WithOptionMiimon returns a func that mutate miimon value.
func WithOptionMiimon(
	miimon uint64, bondInterfaceName string) func(*nmstate.PolicyBuilder) (*nmstate.PolicyBuilder, error) {
	return func(builder *nmstate.PolicyBuilder) (*nmstate.PolicyBuilder, error) {
		glog.V(90).Infof("Changing miimon value for the bondInterface to %d", miimon)

		if bondInterfaceName == "" {
			glog.V(90).Infof("The bondInterfaceName can not be empty string")

			return builder, fmt.Errorf("the bondInterfaceName is empty string")
		}

		var currentState nmstate.DesiredState

		err := yaml.Unmarshal(builder.Definition.Spec.DesiredState.Raw, &currentState)
		if err != nil {
			glog.V(90).Infof("Failed Unmarshal DesiredState")

			return builder, fmt.Errorf("failed Unmarshal DesiredState: %w", err)
		}

		var foundInterface bool

		for i, networkInterface := range currentState.Interfaces {
			if networkInterface.Name == bondInterfaceName && networkInterface.Type == "bond" {
				currentState.Interfaces[i].LinkAggregation.Options.Miimon = int(miimon)
				foundInterface = true
			}
		}

		if !foundInterface {
			glog.V(90).Infof("Failed to find given Bond interface")

			return builder, fmt.Errorf("failed to find Bond interface %s", bondInterfaceName)
		}

		desiredStateYaml, err := yaml.Marshal(currentState)
		if err != nil {
			glog.V(90).Infof("Failed Marshal DesiredState")

			return builder, fmt.Errorf("failed to Marshal a new Desired state: %w", err)
		}

		builder.Definition.Spec.DesiredState = nmstateShared.NewState(string(desiredStateYaml))

		return builder, nil
	}
}

// WithOptionMaxTxRateOnFirstVf returns a func that mutate MaxTxRate value on the first VF.
func WithOptionMaxTxRateOnFirstVf(
	maxTxRate uint64, sriovInterfaceName string) func(*nmstate.PolicyBuilder) (*nmstate.PolicyBuilder, error) {
	return func(builder *nmstate.PolicyBuilder) (*nmstate.PolicyBuilder, error) {
		glog.V(90).Infof("Changing MaxTxRate value for the first VF on given SR-IOV interface to %d",
			maxTxRate)

		if sriovInterfaceName == "" {
			glog.V(90).Infof("The sriovInterfaceName can not be empty string")

			return builder, fmt.Errorf("the sriovInterfaceName is empty string")
		}

		var currentState nmstate.DesiredState

		err := yaml.Unmarshal(builder.Definition.Spec.DesiredState.Raw, &currentState)
		if err != nil {
			glog.V(90).Infof("Failed Unmarshal DesiredState")

			return builder, fmt.Errorf("failed Unmarshal DesiredState: %w", err)
		}

		var foundInterface bool

		for i, networkInterface := range currentState.Interfaces {
			if networkInterface.Name == sriovInterfaceName && networkInterface.Type == "ethernet" {
				value := int(maxTxRate)
				currentState.Interfaces[i].Ethernet.Sriov.Vfs = []nmstate.Vf{{ID: 0, MaxTxRate: &value}}
				foundInterface = true

				break
			}
		}

		if !foundInterface {
			glog.V(90).Infof("Failed to find given SR-IOV interface")

			return builder, fmt.Errorf("failed to find SR-IOV interface %s", sriovInterfaceName)
		}

		desiredStateYaml, err := yaml.Marshal(currentState)
		if err != nil {
			glog.V(90).Infof("Failed Marshal DesiredState")

			return builder, fmt.Errorf("failed to Marshal a new Desired state: %w", err)
		}

		builder.Definition.Spec.DesiredState = nmstateShared.NewState(string(desiredStateYaml))

		return builder, nil
	}
}
