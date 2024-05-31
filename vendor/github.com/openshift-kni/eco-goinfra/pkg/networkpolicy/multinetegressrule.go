package networkpolicy

import (
	"fmt"
	"net"

	"github.com/golang/glog"
	"github.com/k8snetworkplumbingwg/multi-networkpolicy/pkg/apis/k8s.cni.cncf.io/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// EgressAdditionalOptions additional options for MultiNetworkPolicyEgressRule object.
type EgressAdditionalOptions func(builder *EgressRuleBuilder) (*EgressRuleBuilder, error)

// EgressRuleBuilder provides a struct for EgressRules's object definition.
type EgressRuleBuilder struct {
	// EgressRule definition, used to create the EgressRule object.
	definition *v1beta1.MultiNetworkPolicyEgressRule
	// Used to store latest error message upon defining or mutating EgressRule definition.
	errorMsg string
}

// NewEgressRuleBuilder creates a new instance of EgressRuleBuilder.
func NewEgressRuleBuilder() *EgressRuleBuilder {
	glog.V(100).Infof("Initializing new Egress rule structure")

	// Empty rule allowed.
	builder := &EgressRuleBuilder{
		definition: &v1beta1.MultiNetworkPolicyEgressRule{},
	}

	return builder
}

// WithPortAndProtocol adds port and protocol to Egress rule.
func (builder *EgressRuleBuilder) WithPortAndProtocol(port uint16, protocol corev1.Protocol) *EgressRuleBuilder {
	glog.V(100).Infof("Adding port %d and protocol %s to EgressRule", port, protocol)

	if port == 0 {
		glog.V(100).Infof("Port number can not be 0")

		builder.errorMsg = "port number can not be 0"

		return builder
	}

	formattedPort := intstr.FromInt(int(port))

	builder.definition.Ports = append(
		builder.definition.Ports, v1beta1.MultiNetworkPolicyPort{Port: &formattedPort, Protocol: &protocol})

	return builder
}

// WithOptions adds generic options to Egress rule.
func (builder *EgressRuleBuilder) WithOptions(options ...EgressAdditionalOptions) *EgressRuleBuilder {
	glog.V(100).Infof("Setting EgressRule additional options")

	for _, option := range options {
		if option != nil {
			builder, err := option(builder)

			if err != nil {
				glog.V(100).Infof("Error occurred in mutation function")

				builder.errorMsg = err.Error()

				return builder
			}
		}
	}

	return builder
}

// WithPeerPodSelector adds pod selector to Egress rule.
func (builder *EgressRuleBuilder) WithPeerPodSelector(podSelector metav1.LabelSelector) *EgressRuleBuilder {
	glog.V(100).Infof("Adding peer pod selector %v to EgressRule", podSelector)

	if builder.errorMsg != "" {
		return builder
	}

	builder.definition.To = append(builder.definition.To, v1beta1.MultiNetworkPolicyPeer{PodSelector: &podSelector})

	return builder
}

// WithPeerPodSelectorAndCIDR adds pod selector and CIDR to Egress rule.
func (builder *EgressRuleBuilder) WithPeerPodSelectorAndCIDR(
	podSelector metav1.LabelSelector, cidr string, except ...[]string) *EgressRuleBuilder {
	glog.V(100).Infof("Adding peer pod selector %v to EgressRule", podSelector)

	_, _, err := net.ParseCIDR(cidr)

	if err != nil {
		glog.V(100).Infof("Invalid CIDR %s", cidr)

		builder.errorMsg = fmt.Sprintf("Invalid CIDR argument %s", cidr)

		return builder
	}

	builder.WithPeerPodSelector(podSelector)

	// Append IPBlock config to the previously added rule
	builder.definition.To[len(builder.definition.To)-1].IPBlock = &v1beta1.IPBlock{
		CIDR: cidr,
	}

	if len(except) > 0 {
		builder.definition.To[len(builder.definition.To)-1].IPBlock.Except = except[0]
	}

	return builder
}

// GetEgressRuleCfg returns MultiNetworkPolicyEgressRule.
func (builder *EgressRuleBuilder) GetEgressRuleCfg() (*v1beta1.MultiNetworkPolicyEgressRule, error) {
	glog.V(100).Infof("Returning configuration for egress rule")

	if builder.errorMsg != "" {
		glog.V(100).Infof("Failed to build Egress rule configuration due to %s", builder.errorMsg)

		return nil, fmt.Errorf(builder.errorMsg)
	}

	return builder.definition, nil
}
