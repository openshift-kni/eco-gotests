package network

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/msg"
	operatorV1 "github.com/openshift/api/operator/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	goclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// OperatorBuilder provides a struct for network.operator object from the cluster and a network.operator definition.
type OperatorBuilder struct {
	// network.operator definition, used to create the network.operator object.
	Definition *operatorV1.Network
	// Created network.operator object.
	Object *operatorV1.Network
	// api client to interact with the cluster.
	apiClient *clients.Settings
	errorMsg  string
}

// PullOperator loads an existing network.operator into OperatorBuilder struct.
func PullOperator(apiClient *clients.Settings) (*OperatorBuilder, error) {
	glog.V(100).Infof("Pulling existing network.operator name: %s", clusterNetworkName)

	builder := OperatorBuilder{
		apiClient: apiClient,
		Definition: &operatorV1.Network{
			ObjectMeta: metaV1.ObjectMeta{
				Name: clusterNetworkName,
			},
		},
	}

	if !builder.Exists() {
		return nil, fmt.Errorf("network.operator object %s doesn't exist", clusterNetworkName)
	}

	builder.Definition = builder.Object

	return &builder, nil
}

// Exists checks whether the given network.operator exists.
func (builder *OperatorBuilder) Exists() bool {
	if valid, _ := builder.validate(); !valid {
		return false
	}

	glog.V(100).Infof(
		"Checking if network.operator %s exists",
		builder.Definition.Name)

	var err error
	builder.Object, err = builder.Get()

	return err == nil || !k8serrors.IsNotFound(err)
}

// Get returns network.operator object.
func (builder *OperatorBuilder) Get() (*operatorV1.Network, error) {
	if valid, err := builder.validate(); !valid {
		return nil, err
	}

	clusterNetwork := &operatorV1.Network{}
	err := builder.apiClient.Get(context.TODO(), goclient.ObjectKey{
		Name: builder.Definition.Name,
	}, clusterNetwork)

	if err != nil {
		return nil, err
	}

	return clusterNetwork, err
}

// Update renovates the existing network.operator object with the new definition in builder.
func (builder *OperatorBuilder) Update() (*OperatorBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Updating the network.operator object %s",
		builder.Definition.Name,
	)

	err := builder.apiClient.Update(context.TODO(), builder.Definition)

	return builder, err
}

// SetLocalGWMode switches network.operator OVN mode from/to local mode.
func (builder *OperatorBuilder) SetLocalGWMode(state bool, timeout time.Duration) (*OperatorBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	var err error

	if builder.Definition.Spec.DefaultNetwork.OVNKubernetesConfig.GatewayConfig.RoutingViaHost != state {
		builder.Definition.Spec.DefaultNetwork.OVNKubernetesConfig.GatewayConfig.RoutingViaHost = state
		builder, err := builder.Update()

		if err != nil {
			return nil, err
		}

		err = builder.WaitUntilInCondition(
			operatorV1.OperatorStatusTypeProgressing, 60*time.Second, operatorV1.ConditionTrue)

		if err != nil {
			return nil, err
		}

		err = builder.WaitUntilInCondition(
			operatorV1.OperatorStatusTypeProgressing, timeout, operatorV1.ConditionFalse)

		if err != nil {
			return nil, err
		}

		return builder, builder.WaitUntilInCondition(
			operatorV1.OperatorStatusTypeAvailable, 60*time.Second, operatorV1.ConditionTrue)
	}

	return builder, err
}

// SetMultiNetworkPolicy enables network.operator multinetworkpolicy feature.
func (builder *OperatorBuilder) SetMultiNetworkPolicy(state bool, timeout time.Duration) (*OperatorBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Applying MultiNetworkPolicy flag %t to network.operator %s", state, builder.Definition.Name)

	var err error

	if *builder.Definition.Spec.UseMultiNetworkPolicy != state {
		builder.Definition.Spec.UseMultiNetworkPolicy = &state
		builder, err := builder.Update()

		if err != nil {
			return nil, err
		}

		err = builder.WaitUntilInCondition(
			operatorV1.OperatorStatusTypeProgressing, 60*time.Second, operatorV1.ConditionTrue)

		if err != nil {
			return nil, err
		}

		err = builder.WaitUntilInCondition(
			operatorV1.OperatorStatusTypeProgressing, timeout, operatorV1.ConditionFalse)

		if err != nil {
			return nil, err
		}

		return builder, builder.WaitUntilInCondition(
			operatorV1.OperatorStatusTypeAvailable, 60*time.Second, operatorV1.ConditionTrue)
	}

	return builder, err
}

// WaitUntilInCondition waits for a specific time duration until the network.operator will have a
// specified condition type with the expected status.
func (builder *OperatorBuilder) WaitUntilInCondition(
	condition string, timeout time.Duration, status operatorV1.ConditionStatus) error {
	if valid, err := builder.validate(); !valid {
		return err
	}

	glog.V(100).Infof("Wait until network.operator object %s is in condition %v",
		builder.Definition.Name, condition)

	err := wait.PollUntilContextTimeout(
		context.TODO(), 3*time.Second, timeout, true, func(ctx context.Context) (bool, error) {
			if !builder.Exists() {
				return false, fmt.Errorf("network.operator object doesn't exist")
			}

			for _, c := range builder.Object.Status.OperatorStatus.Conditions {
				if c.Type == condition && c.Status == status {
					return true, nil
				}
			}

			return false, nil

		})

	return err
}

// validate will check that the builder and builder definition are properly initialized before
// accessing any member fields.
func (builder *OperatorBuilder) validate() (bool, error) {
	resourceCRD := "Network.Operator"

	if builder == nil {
		glog.V(100).Infof("The %s builder is uninitialized", resourceCRD)

		return false, fmt.Errorf("error: received nil %s builder", resourceCRD)
	}

	if builder.Definition == nil {
		glog.V(100).Infof("The %s is undefined", resourceCRD)

		builder.errorMsg = msg.UndefinedCrdObjectErrString(resourceCRD)
	}

	if builder.apiClient == nil {
		glog.V(100).Infof("The %s builder apiclient is nil", resourceCRD)

		builder.errorMsg = fmt.Sprintf("%s builder cannot have nil apiClient", resourceCRD)
	}

	if builder.errorMsg != "" {
		glog.V(100).Infof("The %s builder has error message: %s", resourceCRD, builder.errorMsg)

		return false, fmt.Errorf(builder.errorMsg)
	}

	return true, nil
}
