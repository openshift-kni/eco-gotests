package sriov

import (
	"context"
	"fmt"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	srIovV1 "github.com/k8snetworkplumbingwg/sriov-network-operator/api/v1"
	"github.com/openshift-kni/eco-gotests/pkg/clients"
	"github.com/openshift-kni/eco-gotests/pkg/slice"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const errorMessageObjectUndefined = "can not redefine undefined SriovNetworkNodePolicy"

// PolicyBuilder provides struct for srIovPolicy object which contains connection to cluster and srIovPolicy definition.
type PolicyBuilder struct {
	// srIovPolicy definition. Used to create srIovPolicy object.
	Definition *srIovV1.SriovNetworkNodePolicy
	// Created srIovPolicy object.
	Object *srIovV1.SriovNetworkNodePolicy
	// Used in functions that defines or mutates srIovPolicy definition. errorMsg is processed before srIovPolicy
	// object is created.
	errorMsg string
	// apiClient open api connection to cluster.
	apiClient *clients.Settings
}

// NewBuilder creates new instance of Builder.
func NewBuilder(
	apiClient *clients.Settings, name, nsname, resName string, vfsNumber int,
	nicNames []string, nodeSelector map[string]string) *PolicyBuilder {
	builder := PolicyBuilder{
		apiClient: apiClient,
		Definition: &srIovV1.SriovNetworkNodePolicy{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
			Spec: srIovV1.SriovNetworkNodePolicySpec{
				NodeSelector: nodeSelector,
				NumVfs:       vfsNumber,
				ResourceName: resName,
				Priority:     1,
				NicSelector: srIovV1.SriovNetworkNicSelector{
					PfNames: nicNames,
				},
			},
		},
	}

	if name == "" {
		builder.errorMsg = "SriovNetworkNodePolicy 'name' cannot be empty"
	}

	if nsname == "" {
		builder.errorMsg = "SriovNetworkNodePolicy 'nsname' cannot be empty"
	}

	if len(nicNames) == 0 {
		builder.errorMsg = "SriovNetworkNodePolicy 'nicNames' cannot be empty list"
	}

	if len(nodeSelector) == 0 {
		builder.errorMsg = "SriovNetworkNodePolicy 'nodeSelector' cannot be empty map"
	}

	if vfsNumber <= 0 {
		builder.errorMsg = "SriovNetworkNodePolicy 'vfsNumber' cannot be zero of negative"
	}

	return &builder
}

// WithDevType sets device type in the SriovNetworkNodePolicy definition. Allowed devTypes are vfio-pci and netdevice.
func (builder *PolicyBuilder) WithDevType(devType string) *PolicyBuilder {
	allowedDevTypes := []string{"vfio-pci", "netdevice"}

	if builder.Definition == nil {
		builder.errorMsg = errorMessageObjectUndefined

		return builder
	}

	if !slice.Contains(allowedDevTypes, devType) {
		builder.errorMsg = "invalid device type, allowed devType values are: vfio-pci or netdevice"

		return builder
	}

	builder.Definition.Spec.DeviceType = devType

	return builder
}

// WithVFRange sets specific VF range for each configured PF.
func (builder *PolicyBuilder) WithVFRange(firstVF, lastVF int) *PolicyBuilder {
	if builder.Definition == nil {
		builder.errorMsg = errorMessageObjectUndefined
	}

	if firstVF > lastVF {
		builder.errorMsg = "firstPF argument can not be greater than lastPF"
	}

	if lastVF > 63 {
		builder.errorMsg = "lastVF can not be greater than 63"
	}

	if builder.errorMsg != "" {
		return builder
	}

	var partitionedPFs []string
	for _, pf := range builder.Definition.Spec.NicSelector.PfNames {
		partitionedPFs = append(partitionedPFs, fmt.Sprintf("%s#%d-%d", pf, firstVF, lastVF))
	}

	builder.Definition.Spec.NicSelector.PfNames = partitionedPFs

	return builder
}

// WithMTU sets required MTU on the given SriovNetworkNodePolicy.
func (builder *PolicyBuilder) WithMTU(mtu int) *PolicyBuilder {
	if builder.Definition == nil {
		builder.errorMsg = errorMessageObjectUndefined
	}

	if 1 > mtu || mtu > 9192 {
		builder.errorMsg = fmt.Sprintf("invalid mtu size %d allowed mtu should be in range 1...9192", mtu)
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.Mtu = mtu

	return builder
}

// WithRDMA sets RDMA mode on SriovNetworkNodePolicy object.
func (builder *PolicyBuilder) WithRDMA(rdma bool) *PolicyBuilder {
	if builder.Definition == nil {
		builder.errorMsg = errorMessageObjectUndefined

		return builder
	}

	builder.Definition.Spec.IsRdma = rdma

	return builder
}

// Create generates SriovNetworkNodePolicy on cluster and stores created object in struct.
func (builder *PolicyBuilder) Create() (*PolicyBuilder, error) {
	if builder.errorMsg != "" {
		return nil, fmt.Errorf(builder.errorMsg)
	}

	if !builder.Exists() {
		var err error
		builder.Object, err = builder.apiClient.SriovNetworkNodePolicies(builder.Definition.Namespace).Create(
			context.TODO(), builder.Definition, metaV1.CreateOptions{},
		)

		if err != nil {
			return nil, err
		}
	}

	return builder, nil
}

// Delete removes SriovNetworkNodePolicy object.
func (builder *PolicyBuilder) Delete() error {
	if !builder.Exists() {
		return nil
	}

	err := builder.apiClient.SriovNetworkNodePolicies(builder.Definition.Namespace).Delete(
		context.TODO(), builder.Object.Name, metaV1.DeleteOptions{})

	if err != nil {
		return err
	}

	builder.Object = nil

	return err
}

// Exists tells whether the given SriovNetworkNodePolicy object exists on cluster.
func (builder *PolicyBuilder) Exists() bool {
	var err error
	builder.Object, err = builder.apiClient.SriovNetworkNodePolicies(builder.Definition.Namespace).Get(
		context.Background(), builder.Definition.Name, metaV1.GetOptions{})

	return err == nil || !k8serrors.IsNotFound(err)
}
