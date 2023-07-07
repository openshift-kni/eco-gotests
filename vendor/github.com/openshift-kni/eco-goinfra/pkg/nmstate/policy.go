package nmstate

import (
	"context"
	"fmt"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/golang/glog"

	nmstateShared "github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstateV1 "github.com/nmstate/kubernetes-nmstate/api/v1"

	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/msg"

	coreV1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	goclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// PolicyBuilder provides struct for the NodeNetworkConfigurationPolicy object containing connection to
// the cluster and the NodeNetworkConfigurationPolicy definition.
type PolicyBuilder struct {
	// srIovPolicy definition. Used to create srIovPolicy object.
	Definition *nmstateV1.NodeNetworkConfigurationPolicy
	// Created srIovPolicy object
	Object *nmstateV1.NodeNetworkConfigurationPolicy
	// apiClient opens API connection to the cluster.
	apiClient *clients.Settings
	// errorMsg is processed before the srIovPolicy object is created.
	errorMsg string
}

// NewPolicyBuilder creates a new instance of PolicyBuilder.
func NewPolicyBuilder(apiClient *clients.Settings, name string, nodeSelector map[string]string) *PolicyBuilder {
	glog.V(100).Infof(
		"Initializing new NodeNetworkConfigurationPolicy structure with the following params: %s", name)

	builder := PolicyBuilder{
		apiClient: apiClient,
		Definition: &nmstateV1.NodeNetworkConfigurationPolicy{
			ObjectMeta: metaV1.ObjectMeta{
				Name: name,
			}, Spec: nmstateShared.NodeNetworkConfigurationPolicySpec{
				NodeSelector: nodeSelector,
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the NodeNetworkConfigurationPolicy is empty")

		builder.errorMsg = "NodeNetworkConfigurationPolicy 'name' cannot be empty"
	}

	if len(nodeSelector) == 0 {
		glog.V(100).Infof("The nodeSelector of the NodeNetworkConfigurationPolicy is empty")

		builder.errorMsg = "NodeNetworkConfigurationPolicy 'nodeSelector' cannot be empty map"
	}

	return &builder
}

// Get returns NodeNetworkConfigurationPolicy object if found.
func (builder *PolicyBuilder) Get() (*nmstateV1.NodeNetworkConfigurationPolicy, error) {
	if valid, err := builder.validate(); !valid {
		return nil, err
	}

	glog.V(100).Infof(
		"Collecting NodeNetworkConfigurationPolicy object %s", builder.Definition.Name)

	nmstatePolicy := &nmstateV1.NodeNetworkConfigurationPolicy{}
	err := builder.apiClient.Get(context.TODO(), goclient.ObjectKey{
		Name:      builder.Definition.Name,
		Namespace: builder.Definition.Namespace,
	}, nmstatePolicy)

	if err != nil {
		glog.V(100).Infof("NodeNetworkConfigurationPolicy object %s doesn't exist", builder.Definition.Name)

		return nil, err
	}

	return nmstatePolicy, err
}

// Exists checks whether the given NodeNetworkConfigurationPolicy exists.
func (builder *PolicyBuilder) Exists() bool {
	if valid, _ := builder.validate(); !valid {
		return false
	}

	glog.V(100).Infof(
		"Checking if NodeNetworkConfigurationPolicy %s exists",
		builder.Definition.Name)

	var err error
	builder.Object, err = builder.Get()

	return err == nil || !k8serrors.IsNotFound(err)
}

// Create makes a NodeNetworkConfigurationPolicy in the cluster and stores the created object in struct.
func (builder *PolicyBuilder) Create() (*PolicyBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Creating the NodeNetworkConfigurationPolicy %s", builder.Definition.Name)

	var err error
	if !builder.Exists() {
		err = builder.apiClient.Create(context.TODO(), builder.Definition)
		if err == nil {
			builder.Object = builder.Definition
		}
	}

	return builder, err
}

// Delete removes NodeNetworkConfigurationPolicy object from a cluster.
func (builder *PolicyBuilder) Delete() (*PolicyBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Deleting the NodeNetworkConfigurationPolicy object %s", builder.Definition.Name)

	if !builder.Exists() {
		return builder, fmt.Errorf("NodeNetworkConfigurationPolicy cannot be deleted because it does not exist")
	}

	err := builder.apiClient.Delete(context.TODO(), builder.Definition)

	if err != nil {
		return builder, fmt.Errorf("can not delete NodeNetworkConfigurationPolicy: %w", err)
	}

	builder.Object = nil

	return builder, nil
}

// Update renovates the existing NodeNetworkConfigurationPolicy object
// with the NodeNetworkConfigurationPolicy definition in builder.
func (builder *PolicyBuilder) Update(force bool) (*PolicyBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Updating the NodeNetworkConfigurationPolicy object %s",
		builder.Definition.Name,
	)

	err := builder.apiClient.Update(context.TODO(), builder.Definition)

	if err != nil {
		if force {
			glog.V(100).Infof(
				"Failed to update the NodeNetworkConfigurationPolicy object %s. "+
					"Note: Force flag set, executed delete/create methods instead",
				builder.Definition.Name,
			)

			builder, err := builder.Delete()

			if err != nil {
				glog.V(100).Infof(
					"Failed to update the NodeNetworkConfigurationPolicy object %s, "+
						"due to error in delete function",
					builder.Definition.Name,
				)

				return nil, err
			}

			return builder.Create()
		}
	}

	return builder, err
}

// WithInterfaceAndVFs adds SR-IOV VF configuration to the NodeNetworkConfigurationPolicy.
func (builder *PolicyBuilder) WithInterfaceAndVFs(sriovInterface string, numberOfVF uint8) *PolicyBuilder {
	if valid, err := builder.validate(); !valid {
		builder.errorMsg = err.Error()

		return builder
	}

	glog.V(100).Infof(
		"Creating NodeNetworkConfigurationPolicy %s with SR-IOV VF configuration: %d",
		builder.Definition.Name, numberOfVF)

	if sriovInterface == "" {
		glog.V(100).Infof("The sriovInterface  can not be empty string")

		builder.errorMsg = "The sriovInterface is empty sting"

		return builder
	}

	nmStateDesiredStateInterfacesWithVfs := &DesiredState{
		Interfaces: []NetworkInterface{
			{
				Name:  sriovInterface,
				Type:  "ethernet",
				State: "up",
				Ethernet: Ethernet{
					Sriov: Sriov{TotalVfs: int(numberOfVF)},
				},
			},
		},
	}

	nmStateInterfaceWithVfYaml, err := yaml.Marshal(nmStateDesiredStateInterfacesWithVfs)
	if err != nil {
		builder.errorMsg = "failed to Marshal NMState interface with VF"

		return builder
	}

	builder.Definition.Spec.DesiredState = nmstateShared.NewState(string(nmStateInterfaceWithVfYaml))

	return builder
}

// WaitUntilCondition waits for the duration of the defined timeout or until the
// NodeNetworkConfigurationPolicy gets to a specific condition.
func (builder *PolicyBuilder) WaitUntilCondition(condition nmstateShared.ConditionType, timeout time.Duration) error {
	if valid, err := builder.validate(); !valid {
		return err
	}

	glog.V(100).Infof("Waiting for the defined period until NodeNetworkConfigurationPolicy %s has condition %v",
		builder.Definition.Name, condition)

	if !builder.Exists() {
		return fmt.Errorf("cannot wait for NodeNetworkConfigurationPolicy condition because it does not exist")
	}

	// Polls every retryInterval to determine if NodeNetworkConfigurationPolicy is in desired condition.
	var err error

	return wait.PollImmediate(retryInterval, timeout, func() (bool, error) {
		builder.Object, err = builder.Get()

		if err != nil {
			return false, nil
		}

		for _, cond := range builder.Object.Status.Conditions {
			if cond.Type == condition && cond.Status == coreV1.ConditionTrue {
				return true, nil
			}
		}

		return false, nil
	})
}

// validate will check that the builder and builder definition are properly initialized before
// accessing any member fields.
func (builder *PolicyBuilder) validate() (bool, error) {
	resourceCRD := "NodeNetworkConfigurationPolicy"

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
