package network

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-gotests/pkg/clients"
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
	glog.V(100).Infof(
		"Checking if network.operator %s exists",
		builder.Definition.Name)

	var err error
	builder.Object, err = builder.Get()

	return err == nil || !k8serrors.IsNotFound(err)
}

// Get returns network.operator object.
func (builder *OperatorBuilder) Get() (*operatorV1.Network, error) {
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
	glog.V(100).Infof("Updating the network.operator object %s",
		builder.Definition.Name,
	)

	if builder.errorMsg != "" {
		return nil, fmt.Errorf(builder.errorMsg)
	}

	err := builder.apiClient.Update(context.TODO(), builder.Definition)

	return builder, err
}

// SetLocalGWMode switches network.operator OVN mode from/to local mode.
func (builder *OperatorBuilder) SetLocalGWMode(state bool, timeout time.Duration) (*OperatorBuilder, error) {
	var err error

	if builder.Definition.Spec.DefaultNetwork.OVNKubernetesConfig.GatewayConfig.RoutingViaHost != state {
		builder.Definition.Spec.DefaultNetwork.OVNKubernetesConfig.GatewayConfig.RoutingViaHost = state
		builder, err := builder.Update()

		if err != nil {
			return nil, err
		}

		err = builder.WaitUntilInCondition(
			operatorV1.OperatorStatusTypeProgressing, 30*time.Second, operatorV1.ConditionTrue)

		if err != nil {
			return nil, err
		}

		err = builder.WaitUntilInCondition(
			operatorV1.OperatorStatusTypeProgressing, timeout, operatorV1.ConditionFalse)

		if err != nil {
			return nil, err
		}

		return builder, builder.WaitUntilInCondition(
			operatorV1.OperatorStatusTypeAvailable, 5*time.Second, operatorV1.ConditionTrue)
	}

	return builder, err
}

// WaitUntilInCondition waits for a specific time duration until the network.operator will have a
// specified condition type with the expected status.
func (builder *OperatorBuilder) WaitUntilInCondition(
	condition string, timeout time.Duration, status operatorV1.ConditionStatus) error {
	glog.V(100).Infof("Wait until network.operator object %s is in condition %v",
		builder.Definition.Name, condition)

	err := wait.PollImmediate(3*time.Second, timeout, func() (bool, error) {
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
