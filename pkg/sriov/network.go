package sriov

import (
	"context"
	"fmt"

	"github.com/openshift-kni/eco-gotests/pkg/msg"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	srIovV1 "github.com/k8snetworkplumbingwg/sriov-network-operator/api/v1"
	"github.com/openshift-kni/eco-gotests/pkg/clients"
	"github.com/openshift-kni/eco-gotests/pkg/slice"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const srIovNetworkCrName = "SriovNetwork"

// NetworkBuilder provides struct for srIovNetwork object which contains connection to cluster and
// srIovNetwork definition.
type NetworkBuilder struct {
	// srIovNetwork definition. Used to create srIovNetwork object.
	Definition *srIovV1.SriovNetwork
	// Created srIovNetwork object.
	Object *srIovV1.SriovNetwork
	// Used in functions that define or mutate srIovNetwork definitions. errorMsg is processed before srIovNetwork
	// object is created.
	errorMsg string
	// apiClient opens api connection to the cluster.
	apiClient *clients.Settings
}

// NewNetworkBuilder creates new instance of Builder.
func NewNetworkBuilder(
	apiClient *clients.Settings, name, nsname, targetNsname, resName string) *NetworkBuilder {
	builder := NetworkBuilder{
		apiClient: apiClient,
		Definition: &srIovV1.SriovNetwork{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
			Spec: srIovV1.SriovNetworkSpec{
				ResourceName:     resName,
				NetworkNamespace: targetNsname,
			},
		},
	}

	if name == "" {
		builder.errorMsg = "SrIovNetwork 'name' cannot be empty"
	}

	if nsname == "" {
		builder.errorMsg = "SrIovNetwork 'nsname' cannot be empty"
	}

	if targetNsname == "" {
		builder.errorMsg = "SrIovNetwork 'targetNsname' cannot be empty"
	}

	if resName == "" {
		builder.errorMsg = "SrIovNetwork 'resName' cannot be empty"
	}

	return &builder
}

// WithVLAN sets vlan id in the SrIovNetwork definition. Allowed vlanId range is between 0-4094.
func (builder *NetworkBuilder) WithVLAN(vlanID uint16) *NetworkBuilder {
	if builder.Definition == nil {
		builder.errorMsg = msg.UndefinedCrdObjectErrString(srIovNetworkCrName)
	}

	if vlanID > 4094 {
		builder.errorMsg = "invalid vlanID, allowed vlanID values are between 0-4094"
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.Vlan = int(vlanID)

	return builder
}

// WithSpoof sets spoof flag based on the given argument in the SrIovNetwork definition spec.
func (builder *NetworkBuilder) WithSpoof(enabled bool) *NetworkBuilder {
	if builder.Definition == nil {
		builder.errorMsg = msg.UndefinedCrdObjectErrString(srIovNetworkCrName)

		return builder
	}

	if enabled {
		builder.Definition.Spec.SpoofChk = "on"
	} else {
		builder.Definition.Spec.SpoofChk = "off"
	}

	return builder
}

// WithLinkState sets linkState parameters in the SrIovNetwork definition spec.
func (builder *NetworkBuilder) WithLinkState(linkState string) *NetworkBuilder {
	allowedLinkStates := []string{"enable", "disable", "auto"}

	if builder.Definition == nil {
		builder.errorMsg = msg.UndefinedCrdObjectErrString(srIovNetworkCrName)
	}

	if !slice.Contains(allowedLinkStates, linkState) {
		builder.errorMsg = "invalid 'linkState' parameters"
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.LinkState = linkState

	return builder
}

// WithMaxTxRate sets maxTxRate parameters in the SrIovNetwork definition spec.
func (builder *NetworkBuilder) WithMaxTxRate(maxTxRate uint16) *NetworkBuilder {
	if builder.Definition == nil {
		builder.errorMsg = msg.UndefinedCrdObjectErrString(srIovNetworkCrName)

		return builder
	}

	maxTxRateInt := int(maxTxRate)

	builder.Definition.Spec.MaxTxRate = &maxTxRateInt

	return builder
}

// WithMinTxRate sets minTxRate parameters in the SrIovNetwork definition spec.
func (builder *NetworkBuilder) WithMinTxRate(minTxRate uint16) *NetworkBuilder {
	if builder.Definition == nil {
		builder.errorMsg = msg.UndefinedCrdObjectErrString(srIovNetworkCrName)

		return builder
	}

	maxTxRateInt := int(minTxRate)
	builder.Definition.Spec.MaxTxRate = &maxTxRateInt

	return builder
}

// WithTrustFlag sets trust flag based on the given argument in the SrIoVNetwork definition spec.
func (builder *NetworkBuilder) WithTrustFlag(enabled bool) *NetworkBuilder {
	if builder.Definition == nil {
		builder.errorMsg = msg.UndefinedCrdObjectErrString(srIovNetworkCrName)

		return builder
	}

	if enabled {
		builder.Definition.Spec.Trust = "on"
	} else {
		builder.Definition.Spec.Trust = "off"
	}

	return builder
}

// WithVlanQoS sets qoSClass parameters in the SrIovNetwork definition spec.
func (builder *NetworkBuilder) WithVlanQoS(qoSClass uint16) *NetworkBuilder {
	if builder.Definition == nil {
		builder.errorMsg = msg.UndefinedCrdObjectErrString(srIovNetworkCrName)
	}

	if qoSClass > 7 {
		builder.errorMsg = "Invalid QoS class. Supported vlan QoS class values are between 0...7"
	}

	if builder.errorMsg != "" {
		return builder
	}

	qoSClassInt := int(qoSClass)

	builder.Definition.Spec.VlanQoS = qoSClassInt

	return builder
}

// WithIPAddressSupport sets ips capabilities in the SrIovNetwork definition spec.
func (builder *NetworkBuilder) WithIPAddressSupport() *NetworkBuilder {
	return builder.withCapabilities("ips")
}

// WithMacAddressSupport sets mac capabilities in the SrIovNetwork definition spec.
func (builder *NetworkBuilder) WithMacAddressSupport() *NetworkBuilder {
	return builder.withCapabilities("mac")
}

// Create generates SrIovNetwork in a cluster and stores the created object in struct.
func (builder *NetworkBuilder) Create() (*NetworkBuilder, error) {
	if builder.errorMsg != "" {
		return nil, fmt.Errorf(builder.errorMsg)
	}

	if !builder.Exists() {
		var err error
		builder.Object, err = builder.apiClient.SriovNetworks(builder.Definition.Namespace).Create(
			context.TODO(), builder.Definition, metaV1.CreateOptions{},
		)

		if err != nil {
			return nil, err
		}
	}

	return builder, nil
}

// Delete removes SrIovNetwork object.
func (builder *NetworkBuilder) Delete() error {
	if !builder.Exists() {
		return nil
	}

	err := builder.apiClient.SriovNetworks(builder.Definition.Namespace).Delete(
		context.TODO(), builder.Object.Name, metaV1.DeleteOptions{})

	if err != nil {
		return err
	}

	builder.Object = nil

	return err
}

// Exists checks whether the given SrIovNetwork object exists in a cluster.
func (builder *NetworkBuilder) Exists() bool {
	var err error
	builder.Object, err = builder.apiClient.SriovNetworks(builder.Definition.Namespace).Get(
		context.Background(), builder.Definition.Name, metaV1.GetOptions{})

	return err == nil || !k8serrors.IsNotFound(err)
}

func (builder *NetworkBuilder) withCapabilities(capability string) *NetworkBuilder {
	if builder.Definition == nil {
		builder.errorMsg = msg.UndefinedCrdObjectErrString(srIovNetworkCrName)
	}

	builder.Definition.Spec.Capabilities = fmt.Sprintf(`{ "%s": true }`, capability)

	return builder
}
