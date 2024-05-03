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

// IngressAdditionalOptions additional options for MultiNetworkPolicyIngressRule object.
type IngressAdditionalOptions func(builder *IngressRuleBuilder) (*IngressRuleBuilder, error)

// IngressRuleBuilder provides a struct for IngressRules's object definition.
type IngressRuleBuilder struct {
	// IngressRule definition, used to create the IngressRule object.
	definition *v1beta1.MultiNetworkPolicyIngressRule
	// Used to store latest error message upon defining or mutating IngressRule definition.
	errorMsg string
}

// NewIngressRuleBuilder creates a new instance of IngressRuleBuilder.
func NewIngressRuleBuilder() *IngressRuleBuilder {
	glog.V(100).Infof("Initializing new Ingress rule structure")

	builder := &IngressRuleBuilder{
		definition: &v1beta1.MultiNetworkPolicyIngressRule{},
	}

	return builder
}

// WithPortAndProtocol adds port and protocol to Ingress rule.
func (builder *IngressRuleBuilder) WithPortAndProtocol(port uint16, protocol corev1.Protocol) *IngressRuleBuilder {
	glog.V(100).Infof("Adding port %d protocol %s to IngressRule", port, protocol)

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

// WithOptions adds generic options to Ingress rule.
func (builder *IngressRuleBuilder) WithOptions(options ...IngressAdditionalOptions) *IngressRuleBuilder {
	glog.V(100).Infof("Setting IngressRule additional options")

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

// WithPeerPodSelector adds peer pod selector to Ingress rule.
func (builder *IngressRuleBuilder) WithPeerPodSelector(podSelector metav1.LabelSelector) *IngressRuleBuilder {
	glog.V(100).Infof("Adding peer pod selector %v to Ingress Rule", podSelector)

	if builder.errorMsg != "" {
		return builder
	}

	builder.definition.From = append(
		builder.definition.From, v1beta1.MultiNetworkPolicyPeer{
			PodSelector: &podSelector,
		})

	return builder
}

// WithCIDR adds CIDR to Ingress rule.
func (builder *IngressRuleBuilder) WithCIDR(cidr string, except ...[]string) *IngressRuleBuilder {
	glog.V(100).Infof("Adding peer CIDR %s to Ingress Rule", cidr)

	_, _, err := net.ParseCIDR(cidr)

	if err != nil {
		glog.V(100).Infof("Invalid CIDR %s", cidr)

		builder.errorMsg = fmt.Sprintf("Invalid CIDR argument %s", cidr)

		return builder
	}

	if len(except) > 0 {
		glog.V(100).Infof("Adding CIDR except %s to Ingress Rule", except[0])
	}

	builder.definition.From = append(builder.definition.From, v1beta1.MultiNetworkPolicyPeer{})

	// Append IPBlock config to the previously added rule
	builder.definition.From[len(builder.definition.From)-1].IPBlock = &v1beta1.IPBlock{
		CIDR: cidr,
	}

	if len(except) > 0 {
		builder.definition.From[len(builder.definition.From)-1].IPBlock.Except = except[0]
	}

	return builder
}

// WithPeerPodSelectorAndCIDR adds port and protocol,CIDR to Ingress rule.
func (builder *IngressRuleBuilder) WithPeerPodSelectorAndCIDR(
	podSelector metav1.LabelSelector, cidr string, except ...[]string) *IngressRuleBuilder {
	glog.V(100).Infof("Adding peer pod selector %v and CIDR %s to IngressRule", podSelector, cidr)

	if builder.errorMsg != "" {
		return builder
	}

	builder.WithPeerPodSelector(podSelector)
	builder.WithCIDR(cidr, except...)

	return builder
}

// GetIngressRuleCfg returns MultiNetworkPolicyIngressRule.
func (builder *IngressRuleBuilder) GetIngressRuleCfg() (*v1beta1.MultiNetworkPolicyIngressRule, error) {
	glog.V(100).Infof("Returning configuration for ingress rule")

	if builder.errorMsg != "" {
		glog.V(100).Infof("Failed to build Ingress rule configuration due to %s", builder.errorMsg)

		return nil, fmt.Errorf(builder.errorMsg)
	}

	return builder.definition, nil
}
