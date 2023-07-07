package nmstateenv

import (
	"fmt"
	"time"

	"github.com/golang/glog"

	nmstateShared "github.com/nmstate/kubernetes-nmstate/api/shared"

	"github.com/openshift-kni/eco-goinfra/pkg/daemonset"
	"github.com/openshift-kni/eco-goinfra/pkg/deployment"
	"github.com/openshift-kni/eco-goinfra/pkg/nmstate"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"

	. "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netinittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/sriov/internal/tsparams"

	"k8s.io/apimachinery/pkg/util/wait"
)

// CreateNewNMStateAndWaitUntilItsRunning creates or recreates the new NMState instance and waits until
// all daemonsets and deployments are in Ready state.
func CreateNewNMStateAndWaitUntilItsRunning(timeout time.Duration) error {
	glog.V(90).Infof("Verifying if NMState instance is installed")

	nmstateInstance, err := nmstate.PullNMstate(APIClient, tsparams.NMState)

	if err == nil {
		glog.V(90).Infof("NMState exists. Removing NMState.")

		_, err = nmstateInstance.Delete()

		if err != nil {
			return err
		}
	}

	glog.V(90).Infof("Creating a new NMState instance.")

	_, err = nmstate.NewBuilder(APIClient, tsparams.NMState).Create()
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
	nodeList := nodes.NewBuilder(APIClient, nodeLabel)

	err = nodeList.Discover()
	if err != nil {
		return err
	}

	for _, node := range nodeList.Objects {
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

func isNMStateDeployedAndReady(timeout time.Duration) error {
	glog.V(90).Infof("Checking that NMState deployments and daemonsets are ready.")

	var (
		nmstateHandlerDs         *daemonset.Builder
		nmstateWebhookDeployment *deployment.Builder
		nmstateCertDeployment    *deployment.Builder
		err                      error
	)

	glog.V(90).Infof("Pulling all NMState default daemonsets and deployments.")

	err = wait.PollImmediate(5*time.Second, timeout, func() (bool, error) {
		nmstateHandlerDs, err = daemonset.Pull(
			APIClient, tsparams.NMStateHandlerDsName, NetConfig.NMStateOperatorNamespace)
		if err != nil {
			glog.V(90).Infof("Error to pull daemonset %s from namespace %s, retry",
				tsparams.NMStateHandlerDsName, NetConfig.NMStateOperatorNamespace)

			return false, nil
		}

		nmstateWebhookDeployment, err = deployment.Pull(
			APIClient, tsparams.NMStateWebhookDeploymentName, NetConfig.NMStateOperatorNamespace)
		if err != nil {
			glog.V(90).Infof("Error to pull deployment %s namespace %s, retry",
				tsparams.NMStateWebhookDeploymentName, NetConfig.NMStateOperatorNamespace)

			return false, nil
		}

		nmstateCertDeployment, err = deployment.Pull(
			APIClient, tsparams.NMStateCertDeploymentName, NetConfig.NMStateOperatorNamespace)
		if err != nil {
			glog.V(90).Infof("Error to pull deployment %s namespace %s, retry",
				tsparams.NMStateCertDeploymentName, NetConfig.NMStateOperatorNamespace)

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
