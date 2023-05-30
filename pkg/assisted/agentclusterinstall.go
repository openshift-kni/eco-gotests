package assisted

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-gotests/pkg/clients"
	"github.com/openshift-kni/eco-gotests/pkg/msg"
	hiveextV1Beta1 "github.com/openshift/assisted-service/api/hiveextension/v1beta1"
	v1 "github.com/openshift/hive/apis/hive/v1"
	coreV1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	goclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// AgentClusterInstallBuilder provides struct for the agentclusterinstall object containing connection to
// the cluster and the agentclusterinstall definitions.
type AgentClusterInstallBuilder struct {
	Definition *hiveextV1Beta1.AgentClusterInstall
	Object     *hiveextV1Beta1.AgentClusterInstall
	errorMsg   string
	apiClient  *clients.Settings
}

// agentClusterInstallCondition provides a struct for tracking a specific agentclusterinstall condition.
type agentClusterInstallCondition struct {
	v1.ClusterInstallCondition
	index int
}

// NewAgentClusterInstallBuilder creates a new instance of AgentClusterInstallBuilder.
func NewAgentClusterInstallBuilder(
	apiClient *clients.Settings,
	name string,
	nsname string,
	clusterDeployment string,
	masterCount int,
	workerCount int,
	network hiveextV1Beta1.Networking) *AgentClusterInstallBuilder {
	builder := AgentClusterInstallBuilder{
		apiClient: apiClient,
		Definition: &hiveextV1Beta1.AgentClusterInstall{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
			Spec: hiveextV1Beta1.AgentClusterInstallSpec{
				ClusterDeploymentRef: coreV1.LocalObjectReference{
					Name: clusterDeployment,
				},
				Networking: network,
				ProvisionRequirements: hiveextV1Beta1.ProvisionRequirements{
					ControlPlaneAgents: masterCount,
					WorkerAgents:       workerCount,
				},
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the agentclusterinstall is empty")

		builder.errorMsg = "agentclusterinstall 'name' cannot be empty"
	}

	if nsname == "" {
		glog.V(100).Infof("The namespace of the agentclusterinstall is empty")

		builder.errorMsg = "agentclusterinstall 'namespace' cannot be empty"
	}

	if clusterDeployment == "" {
		glog.V(100).Infof("The clusterDeployment ref for the agentclusterinstall is empty")

		builder.errorMsg = "agentclusterinstall 'clusterDeployment' cannot be empty"
	}

	return &builder
}

// WithAPIVip sets the apiVIP to use during multi-node installations.
func (builder *AgentClusterInstallBuilder) WithAPIVip(apiVIP string) *AgentClusterInstallBuilder {
	if builder.Definition == nil {
		glog.V(100).Infof("The agentclusterinstall is undefined")

		builder.errorMsg = msg.UndefinedCrdObjectErrString("AgentClusterInstall")
	}

	if net.ParseIP(apiVIP) == nil {
		glog.V(100).Infof("The apiVIP is not a properly formatted IP address")

		builder.errorMsg = "agentclusterinstall apiVIP incorrectly formatted"
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.APIVIP = apiVIP

	return builder
}

// WithAdditionalAPIVip appends apiVIP to the apiVIPs field for use during dual-stack installations.
func (builder *AgentClusterInstallBuilder) WithAdditionalAPIVip(apiVIP string) *AgentClusterInstallBuilder {
	if builder.Definition == nil {
		glog.V(100).Infof("The agentclusterinstall is undefined")

		builder.errorMsg = msg.UndefinedCrdObjectErrString("AgentClusterInstall")
	}

	if net.ParseIP(apiVIP) == nil {
		glog.V(100).Infof("The apiVIP is not a properly formatted IP address")

		builder.errorMsg = "agentclusterinstall apiVIP incorrectly formatted"
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.APIVIPs = append(builder.Definition.Spec.APIVIPs, apiVIP)

	return builder
}

// WithIngressVip sets the ingressVIP to use during multi-node installations.
func (builder *AgentClusterInstallBuilder) WithIngressVip(ingressVIP string) *AgentClusterInstallBuilder {
	if builder.Definition == nil {
		glog.V(100).Infof("The agentclusterinstall is undefined")

		builder.errorMsg = msg.UndefinedCrdObjectErrString("AgentClusterInstall")
	}

	if net.ParseIP(ingressVIP) == nil {
		glog.V(100).Infof("The ingressVIP is not a properly formatted IP address")

		builder.errorMsg = "agentclusterinstall ingressVIP incorrectly formatted"
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.IngressVIP = ingressVIP

	return builder
}

// WithAdditionalIngressVip appends ingressVIP to the ingressVIPs field for use during dual-stack installations.
func (builder *AgentClusterInstallBuilder) WithAdditionalIngressVip(ingressVIP string) *AgentClusterInstallBuilder {
	if builder.Definition == nil {
		glog.V(100).Infof("The agentclusterinstall is undefined")

		builder.errorMsg = msg.UndefinedCrdObjectErrString("AgentClusterInstall")
	}

	if net.ParseIP(ingressVIP) == nil {
		glog.V(100).Infof("The ingressVIP is not a properly formatted IP address")

		builder.errorMsg = "agentclusterinstall ingressVIP incorrectly formatted"
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.IngressVIPs = append(builder.Definition.Spec.IngressVIPs, ingressVIP)

	return builder
}

// WithUserManagedNetworking sets userManagedNetworking field.
func (builder *AgentClusterInstallBuilder) WithUserManagedNetworking(enabled bool) *AgentClusterInstallBuilder {
	if builder.Definition == nil {
		glog.V(100).Infof("The agentclusterinstall is undefined")

		builder.errorMsg = msg.UndefinedCrdObjectErrString("AgentClusterInstall")
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.Networking.UserManagedNetworking = &enabled

	return builder
}

// WithPlatformType sets platformType field (Supported values: "", None, BareMetal, VSphere).
func (builder *AgentClusterInstallBuilder) WithPlatformType(
	platform hiveextV1Beta1.PlatformType) *AgentClusterInstallBuilder {
	if builder.Definition == nil {
		glog.V(100).Infof("The agentclusterinstall is undefined")

		builder.errorMsg = msg.UndefinedCrdObjectErrString("AgentClusterInstall")
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.PlatformType = platform

	return builder
}

// WithControlPlaneAgents sets the number of masters to use for the installation.
func (builder *AgentClusterInstallBuilder) WithControlPlaneAgents(agentCount int) *AgentClusterInstallBuilder {
	if builder.Definition == nil {
		glog.V(100).Infof("The agentclusterinstall is undefined")

		builder.errorMsg = msg.UndefinedCrdObjectErrString("AgentClusterInstall")
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.ProvisionRequirements.ControlPlaneAgents = agentCount

	return builder
}

// WithWorkerAgents sets the number of workers to use during the installation.
func (builder *AgentClusterInstallBuilder) WithWorkerAgents(agentCount int) *AgentClusterInstallBuilder {
	if builder.Definition == nil {
		glog.V(100).Infof("The agentclusterinstall is undefined")

		builder.errorMsg = msg.UndefinedCrdObjectErrString("AgentClusterInstall")
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.ProvisionRequirements.WorkerAgents = agentCount

	return builder
}

// WithImageSet sets the clusterimageset to use for the installation.
func (builder *AgentClusterInstallBuilder) WithImageSet(imageSet string) *AgentClusterInstallBuilder {
	if builder.Definition == nil {
		glog.V(100).Infof("The agentclusterinstall is undefined")

		builder.errorMsg = msg.UndefinedCrdObjectErrString("AgentClusterInstall")
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.ImageSetRef = &v1.ClusterImageSetReference{Name: imageSet}

	return builder
}

// WithSSHPublicKey is the ssh key that is allowed to access the nodes.
func (builder *AgentClusterInstallBuilder) WithSSHPublicKey(pubkey string) *AgentClusterInstallBuilder {
	if builder.Definition == nil {
		glog.V(100).Infof("The agentclusterinstall is undefined")

		builder.errorMsg = msg.UndefinedCrdObjectErrString("AgentClusterInstall")
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.SSHPublicKey = pubkey

	return builder
}

// WithNetworkType sets the cluster networking type (Supported values: OpenShiftSDN, OVNKubernetes).
func (builder *AgentClusterInstallBuilder) WithNetworkType(netType string) *AgentClusterInstallBuilder {
	if builder.Definition == nil {
		glog.V(100).Infof("The agentclusterinstall is undefined")

		builder.errorMsg = msg.UndefinedCrdObjectErrString("AgentClusterInstall")
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.Networking.NetworkType = netType

	return builder
}

// WithAdditionalClusterNetwork appends additional cluster networks to be used by the cluster.
func (builder *AgentClusterInstallBuilder) WithAdditionalClusterNetwork(
	cidr string,
	prefix int32) *AgentClusterInstallBuilder {
	if builder.Definition == nil {
		glog.V(100).Infof("The agentclusterinstall is undefined")

		builder.errorMsg = msg.UndefinedCrdObjectErrString("AgentClusterInstall")
	}

	if _, _, err := net.ParseCIDR(cidr); err != nil {
		glog.V(100).Infof("The agentclusterinstall passed invalid clusterNetwork cidr: %s", cidr)

		builder.errorMsg = "Got invalid cidr for clusternetwork"
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.Networking.ClusterNetwork =
		append(builder.Definition.Spec.Networking.ClusterNetwork,
			hiveextV1Beta1.ClusterNetworkEntry{CIDR: cidr, HostPrefix: prefix})

	return builder
}

// WithAdditionalServiceNetwork appends additional service networks to be used by the cluster.
func (builder *AgentClusterInstallBuilder) WithAdditionalServiceNetwork(cidr string) *AgentClusterInstallBuilder {
	if builder.Definition == nil {
		glog.V(100).Infof("The agentclusterinstall is undefined")

		builder.errorMsg = msg.UndefinedCrdObjectErrString("AgentClusterInstall")
	}

	if _, _, err := net.ParseCIDR(cidr); err != nil {
		glog.V(100).Infof("The agentclusterinstall passed invalid serviceNetwork cidr: %s", cidr)

		builder.errorMsg = "Got invalid cidr for servicenetwork"
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.Networking.ServiceNetwork = append(builder.Definition.Spec.Networking.ServiceNetwork, cidr)

	return builder
}

// WaitForState will wait the defined timeout for the agentclusterinstall to have the defined state.
func (builder *AgentClusterInstallBuilder) WaitForState(
	state string,
	timeout time.Duration) (*AgentClusterInstallBuilder, error) {
	if builder.Definition == nil {
		glog.V(100).Infof("The agentclusterinstall is undefined")

		builder.errorMsg = msg.UndefinedCrdObjectErrString("AgentClusterInstall")
	}

	if builder.errorMsg != "" {
		return builder, nil
	}

	// Polls every second to determine if agentclusterinstall in desired state.
	var err error
	err = wait.PollImmediate(time.Second, timeout, func() (bool, error) {
		builder.Object, err = builder.Get()

		if err != nil {
			return false, nil
		}

		return builder.Object.Status.DebugInfo.State == state, err

	})

	if err == nil {
		return builder, nil
	}

	return nil, err
}

// WaitForStateInfo will wait the defined timeout for stateInfo to match the defined stateInfo string.
func (builder *AgentClusterInstallBuilder) WaitForStateInfo(
	stateInfo string,
	timeout time.Duration) (*AgentClusterInstallBuilder, error) {
	if builder.Definition == nil {
		glog.V(100).Infof("The agentclusterinstall is undefined")

		builder.errorMsg = msg.UndefinedCrdObjectErrString("AgentClusterInstall")
	}

	if builder.errorMsg != "" {
		return builder, nil
	}

	// Polls every second to determine if agentclusterinstall has the desired stateinfo message.
	var err error
	err = wait.PollImmediate(time.Second, timeout, func() (bool, error) {
		builder.Object, err = builder.Get()

		if err != nil {
			return false, nil
		}

		return builder.Object.Status.DebugInfo.StateInfo == stateInfo, err

	})

	if err == nil {
		return builder, nil
	}

	return nil, err
}

// GetCondition creates and returns a new agentClusterInstallCondition based on the condition type provided.
func (builder *AgentClusterInstallBuilder) GetCondition(conditionType string) (*agentClusterInstallCondition, error) {
	glog.V(100).Infof("Getting condition %s from agentclusterinstall %s", conditionType, builder.Definition.Name)

	if !builder.Exists() {
		glog.V(100).Infof("Getting agentclusterinstallcondtion from non-existent agentclusterinstall")

		return nil, fmt.Errorf("cannot get conditions from non-existent agentclusterinstall")
	}

	_, err := builder.waitForConditions(retryInterval * 2)
	if err != nil {
		return nil, err
	}

	for index, condition := range builder.Object.Status.Conditions {
		if conditionType == condition.Type {
			return newagentClusterInstallCondition(conditionType, index, condition)
		}
	}

	return nil, fmt.Errorf("condition '%s' does not exist within agentclusterinstall conditions", conditionType)
}

// UpdateCondition updates the agentClusterInstallCondition fields with the current state of the condition.
func (builder *AgentClusterInstallBuilder) UpdateCondition(
	condition *agentClusterInstallCondition) (*agentClusterInstallCondition, error) {
	if !builder.Exists() {
		glog.V(100).Infof("Agentclusterinstall %s does not exist on cluster", builder.Definition.Name)

		return nil, fmt.Errorf("cannot update condition on non-existent agentclusterinstall")
	}

	if condition == nil {
		glog.V(100).Infof("cannot update undefined condition agentClusterInstallCondition")

		return nil, fmt.Errorf("cannot update condition on non-existent agentclusterinstallconditon")
	}

	glog.V(100).Infof("Updating condition %s in agentclusterinstall %s", condition.Type, builder.Definition.Name)

	_, err := builder.waitForConditions(retryInterval * 2)
	if err != nil {
		return nil, err
	}

	agentClusterInstall, err := builder.Get()
	if err != nil {
		return nil, err
	}

	if len(agentClusterInstall.Status.Conditions) < condition.index+1 || condition.index < 0 {
		glog.V(100).Infof("agentClusterInstallCondition index is out of bounds")

		return nil, fmt.Errorf("cannot find condition: agentClusterInstallCondition index is out of bounds")
	}

	condition.ClusterInstallCondition = agentClusterInstall.Status.Conditions[condition.index]

	return condition, nil
}

// WaitForConditionMessage waits the specified timeout for the given condition to report the specified message.
func (builder *AgentClusterInstallBuilder) WaitForConditionMessage(
	condition *agentClusterInstallCondition,
	message string,
	timeout time.Duration) (*agentClusterInstallCondition, error) {
	if condition == nil {
		return nil, fmt.Errorf("cannot wait for undefined condition message")
	}

	glog.V(100).Infof("Waiting for message '%s' on condition %s in agentclusterinstall %s",
		message, condition.Type, builder.Definition.Name)

	// Polls every retryInterval to determine if agentclusterinstall validation has desired status.
	err := wait.PollImmediate(retryInterval, timeout, func() (bool, error) {
		_, err := builder.UpdateCondition(condition)
		if err != nil {
			return false, nil
		}

		return condition.Message == message, err
	})

	if err == nil {
		return condition, nil
	}

	return nil, err
}

// WaitForConditionStatus waits the specified timeout for the given condition to report the specified status.
func (builder *AgentClusterInstallBuilder) WaitForConditionStatus(
	condition *agentClusterInstallCondition,
	status string,
	timeout time.Duration) (*agentClusterInstallCondition, error) {
	if condition == nil {
		return nil, fmt.Errorf("cannot wait for undefined condition status")
	}

	glog.V(100).Infof("Waiting for status '%s' on condition %s in agentclusterinstall %s",
		status, condition.Type, builder.Definition.Name)

	// Polls every retryInterval to determine if agentclusterinstall validation has desired status.
	err := wait.PollImmediate(retryInterval, timeout, func() (bool, error) {
		_, err := builder.UpdateCondition(condition)
		if err != nil {
			return false, nil
		}

		return string(condition.Status) == status, err
	})

	return condition, err
}

// WaitForConditionReason waits the specified timeout for the given condition to report the specified reason.
func (builder *AgentClusterInstallBuilder) WaitForConditionReason(
	condition *agentClusterInstallCondition,
	reason string,
	timeout time.Duration) (*agentClusterInstallCondition, error) {
	if condition == nil {
		return nil, fmt.Errorf("cannot wait for undefined condition reason")
	}

	glog.V(100).Infof("Waiting for reason '%s' on condition %s in agentclusterinstall %s",
		reason, condition.Type, builder.Definition.Name)

	// Polls every retryInterval to determine if agentclusterinstall validation has desired status.
	err := wait.PollImmediate(retryInterval, timeout, func() (bool, error) {
		_, err := builder.UpdateCondition(condition)
		if err != nil {
			return false, nil
		}

		return condition.Reason == reason, err
	})

	return condition, err
}

// Get fetches the defined agentclusterinstall from the cluster.
func (builder *AgentClusterInstallBuilder) Get() (*hiveextV1Beta1.AgentClusterInstall, error) {
	glog.V(100).Infof("Getting agentclusterinstall %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	agentClusterInstall := &hiveextV1Beta1.AgentClusterInstall{}

	err := builder.apiClient.Get(context.TODO(), goclient.ObjectKey{
		Name:      builder.Definition.Name,
		Namespace: builder.Definition.Namespace,
	}, agentClusterInstall)

	if err != nil {
		return nil, err
	}

	return agentClusterInstall, err
}

// PullAgentClusterInstall pulls existing agentclusterinstall from cluster.
func PullAgentClusterInstall(apiClient *clients.Settings, name, nsname string) (*AgentClusterInstallBuilder, error) {
	glog.V(100).Infof("Pulling existing agentclusterinstall name %s under namespace %s from cluster", name, nsname)

	builder := AgentClusterInstallBuilder{
		apiClient: apiClient,
		Definition: &hiveextV1Beta1.AgentClusterInstall{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the agentclusterinstall is empty")

		builder.errorMsg = "agentclusterinstall 'name' cannot be empty"
	}

	if nsname == "" {
		glog.V(100).Infof("The namespace of the agentclusterinstall is empty")

		builder.errorMsg = "agentclusterinstall 'namespace' cannot be empty"
	}

	if !builder.Exists() {
		return nil, fmt.Errorf("agentclusterinstall object %s doesn't exist in namespace %s", name, nsname)
	}

	builder.Definition = builder.Object

	return &builder, nil
}

// Create generates a agentclusterinstall on the cluster.
func (builder *AgentClusterInstallBuilder) Create() (*AgentClusterInstallBuilder, error) {
	glog.V(100).Infof("Creating the agentclusterinstall %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	if builder.errorMsg != "" {
		return nil, fmt.Errorf(builder.errorMsg)
	}

	var err error
	if !builder.Exists() {
		err = builder.apiClient.Create(context.TODO(), builder.Definition)
		if err == nil {
			builder.Object = builder.Definition
		}
	}

	return builder, err
}

// Update modifies an existing agentclusterinstall on the cluster.
func (builder *AgentClusterInstallBuilder) Update(force bool) (*AgentClusterInstallBuilder, error) {
	glog.V(100).Infof("Updating agentclusterinstall %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	if !builder.Exists() {
		builder.errorMsg = "Cannot update non-existent agentclusterinstall"
	}

	if builder.errorMsg != "" {
		return nil, fmt.Errorf(builder.errorMsg)
	}

	err := builder.apiClient.Update(context.TODO(), builder.Definition)

	if err != nil {
		if force {
			glog.V(100).Infof(
				"Failed to update the agentclusterinstall object %s in namespace %s. "+
					"Note: Force flag set, executed delete/create methods instead",
				builder.Definition.Name, builder.Definition.Namespace,
			)

			err = builder.DeleteAndWait(time.Second * 10)
			builder.Definition.ResourceVersion = ""
			// fmt.Printf("agentclusterinstall exists: %v\n", builder.Exists())

			if err != nil {
				glog.V(100).Infof(
					"Failed to update the agentclusterinstall object %s in namespace %s, "+
						"due to error in delete function",
					builder.Definition.Name, builder.Definition.Namespace,
				)

				return nil, err
			}

			return builder.Create()
		}
	}

	return builder, err
}

// Delete removes an agentclusterinstall from the cluster.
func (builder *AgentClusterInstallBuilder) Delete() (*AgentClusterInstallBuilder, error) {
	glog.V(100).Infof("Deleting the agentclusterinstall %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	if !builder.Exists() {
		return builder, fmt.Errorf("agentclusterinstall cannot be deleted because it does not exist")
	}

	err := builder.apiClient.Delete(context.TODO(), builder.Definition)

	if err != nil {
		return builder, fmt.Errorf("cannot delete agentclusterinstall: %w", err)
	}

	builder.Object = nil

	return builder, nil
}

// DeleteAndWait deletes an agentclusterinstall and waits until it is removed from the cluster.
func (builder *AgentClusterInstallBuilder) DeleteAndWait(timeout time.Duration) error {
	glog.V(100).Infof(`Deleting agentclusterinstall %s in namespace %s and 
	waiting for the defined period until it's removed`,
		builder.Definition.Name, builder.Definition.Namespace)

	if _, err := builder.Delete(); err != nil {
		return err
	}

	// Polls the agentclusterinstall every second until it's removed.
	return wait.PollImmediate(time.Second, timeout, func() (bool, error) {
		_, err := builder.Get()
		if k8serrors.IsNotFound(err) {

			return true, nil
		}

		return false, nil
	})
}

// Exists checks if the defined agentclusterinstall has already been created.
func (builder *AgentClusterInstallBuilder) Exists() bool {
	glog.V(100).Infof("Checking if agentclusterinstall %s exists in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	builder.Object, err = builder.Get()

	return err == nil || !k8serrors.IsNotFound(err)
}

// newagentClusterInstallCondition creates a new instance of agentClusterInstallCondition.
func newagentClusterInstallCondition(
	conditionType string,
	index int,
	condition v1.ClusterInstallCondition) (*agentClusterInstallCondition, error) {
	glog.V(100).Infof(
		"Initializing new agentClusterInstallCondition structure with the following params: "+
			"conditionType: %s, index: %d, condition: %v",
		conditionType, index, condition)

	if conditionType == "" {
		glog.V(100).Infof("The conditionType is empty")

		return nil, fmt.Errorf("cannot create agentClusterInstallCondition without a condition type")
	}

	if index < 0 {
		return nil, fmt.Errorf("cannot create agentClusterInstallCondition with an index less than 0")
	}

	return &agentClusterInstallCondition{
		ClusterInstallCondition: condition,
		index:                   index,
	}, nil
}

// waitForConditions waits the speicified timeout for conditions to be available on the agentclusterinstall.
func (builder *AgentClusterInstallBuilder) waitForConditions(
	timeout time.Duration) (*AgentClusterInstallBuilder, error) {
	if !builder.Exists() {
		return builder, fmt.Errorf("cannot get conditions from undefined agentclusterinstall")
	}
	// Polls every retryInterval to determine if agentclusterinstall conditions are available.
	var err error
	err = wait.PollImmediate(retryInterval, timeout, func() (bool, error) {
		builder.Object, err = builder.Get()

		if err != nil {
			return false, nil
		}

		return builder.Object.Status.Conditions != nil, err
	})

	return builder, err
}
