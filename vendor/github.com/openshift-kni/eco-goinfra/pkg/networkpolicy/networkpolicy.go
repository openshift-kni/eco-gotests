package networkpolicy

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/msg"
	netv1 "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NetworkPolicyBuilder provides struct for networkPolicy object.
type NetworkPolicyBuilder struct {
	// NetworkPolicy definition. Used to create networkPolicy object with minimum set of required elements.
	Definition *netv1.NetworkPolicy
	// Created networkPolicy object on the cluster.
	Object *netv1.NetworkPolicy
	// api client to interact with the cluster.
	apiClient *clients.Settings
	// errorMsg is processed before NetworkPolicy object is created.
	errorMsg string
}

// NewNetworkPolicyBuilder method creates new instance of builder.
func NewNetworkPolicyBuilder(apiClient *clients.Settings, name, nsname string) *NetworkPolicyBuilder {
	glog.V(100).Infof("Initializing new NetworkPolicyBuilder structure with the following params: name: %s, namespace: %s",
		name, nsname)

	builder := &NetworkPolicyBuilder{
		apiClient: apiClient,
		Definition: &netv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the networkPolicy is empty")

		builder.errorMsg = "The networkPolicy 'name' cannot be empty"
	}

	if nsname == "" {
		glog.V(100).Infof("The namespace of the networkPolicy is empty")

		builder.errorMsg = "The networkPolicy 'namespace' cannot be empty"
	}

	return builder
}

// WithNamespaceIngressRule applies ingress rule for the networkPolicy.
func (builder *NetworkPolicyBuilder) WithNamespaceIngressRule(
	namespaceIngressMatchLabels map[string]string,
	podIngressMatchLabels map[string]string) *NetworkPolicyBuilder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof(
		"Applying Ingress rule to networkPolicy %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	if len(namespaceIngressMatchLabels) == 0 && len(podIngressMatchLabels) == 0 {
		glog.V(100).Infof("At least one type of the selector for NetworkPolicy ingress rule should be defined")

		builder.errorMsg = "Both namespaceIngressMatchLabels and podIngressMatchLabels parameters are empty maps"
	}

	if builder.errorMsg != "" {
		return builder
	}

	var peerRule netv1.NetworkPolicyPeer

	if len(namespaceIngressMatchLabels) != 0 {
		glog.V(100).Infof(
			"Applying Ingress rule with namespaceIngressMatchLabels %v parameter to networkPolicy %s in namespace %s",
			namespaceIngressMatchLabels, builder.Definition.Name, builder.Definition.Namespace)

		peerRule.NamespaceSelector = &metav1.LabelSelector{
			MatchLabels: namespaceIngressMatchLabels,
		}
	}

	if len(podIngressMatchLabels) != 0 {
		glog.V(100).Infof(
			"Applying Ingress rule with podIngressMatchLabels %v parameter to networkPolicy %s in namespace %s",
			podIngressMatchLabels, builder.Definition.Name, builder.Definition.Namespace)

		peerRule.PodSelector = &metav1.LabelSelector{
			MatchLabels: podIngressMatchLabels,
		}
	}

	if builder.Definition.Spec.Ingress == nil {
		builder.Definition.Spec.Ingress = make([]netv1.NetworkPolicyIngressRule, 1)
	}

	ingressRule := netv1.NetworkPolicyIngressRule{}
	ingressRule.From = append(ingressRule.From, peerRule)

	builder.Definition.Spec.Ingress = append(builder.Definition.Spec.Ingress, ingressRule)

	return builder
}

// WithPolicyType add policyType to the networkPolicy.
func (builder *NetworkPolicyBuilder) WithPolicyType(policyType netv1.PolicyType) *NetworkPolicyBuilder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof(
		"Creating networkPolicy %s in %s namespace with the policyType defined: %v",
		builder.Definition.Name, builder.Definition.Namespace, policyType)

	if policyType == "" {
		glog.V(100).Infof("The policyType value has to be provided")

		builder.errorMsg = "The policyType is an empty string"
	}

	if builder.errorMsg != "" {
		return builder
	}

	if builder.Definition.Spec.PolicyTypes == nil {
		builder.Definition.Spec.PolicyTypes = []netv1.PolicyType{policyType}

		return builder
	}

	builder.Definition.Spec.PolicyTypes = append(builder.Definition.Spec.PolicyTypes, policyType)

	return builder
}

// WithPodSelector add podSelector to the networkPolicy.
func (builder *NetworkPolicyBuilder) WithPodSelector(podSelectorMatchLabels map[string]string) *NetworkPolicyBuilder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof(
		"Creating networkPolicy %s in %s namespace with podSelector defined: %v",
		builder.Definition.Name, builder.Definition.Namespace, podSelectorMatchLabels)

	if len(podSelectorMatchLabels) == 0 {
		glog.V(100).Infof("The podSelector could not be empty")

		builder.errorMsg = "The podSelector is an empty string"
	}

	if builder.errorMsg != "" {
		return builder
	}

	podSelector := metav1.LabelSelector{MatchLabels: podSelectorMatchLabels}
	builder.Definition.Spec.PodSelector = podSelector

	return builder
}

// Pull loads an existing networkPolicy into the Builder struct.
func Pull(apiClient *clients.Settings, name, nsname string) (*NetworkPolicyBuilder, error) {
	glog.V(100).Infof("Pulling existing networkPolicy name: %s namespace:%s", name, nsname)

	builder := NetworkPolicyBuilder{
		apiClient: apiClient,
		Definition: &netv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the networkPolicy is empty")

		builder.errorMsg = "networkPolicy 'name' cannot be empty"
	}

	if nsname == "" {
		glog.V(100).Infof("The namespace of the networkPolicy is empty")

		builder.errorMsg = "networkPolicy 'namespace' cannot be empty"
	}

	if builder.errorMsg != "" {
		return nil, fmt.Errorf("failed to pull networkPolicy object due to the following error: %s",
			builder.errorMsg)
	}

	if !builder.Exists() {
		glog.V(100).Infof("Failed to pull networkPolicy object %s from namespace %s. Object doesn't exist",
			name, nsname)

		return nil, fmt.Errorf("networkPolicy object %s doesn't exist in namespace %s", name, nsname)
	}

	builder.Definition = builder.Object

	return &builder, nil
}

// Create makes a networkPolicy in cluster and stores the created object in struct.
func (builder *NetworkPolicyBuilder) Create() (*NetworkPolicyBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Creating the networkPolicy %s in %s namespace",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	if !builder.Exists() {
		builder.Object, err = builder.apiClient.NetworkPolicies(builder.Definition.Namespace).Create(
			context.TODO(), builder.Definition, metav1.CreateOptions{})
	}

	return builder, err
}

// Exists checks whether the given NetworkPolicy exists.
func (builder *NetworkPolicyBuilder) Exists() bool {
	if valid, _ := builder.validate(); !valid {
		return false
	}

	glog.V(100).Infof("Checking if networkPolicy %s exists in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	builder.Object, err = builder.apiClient.NetworkPolicies(builder.Definition.Namespace).Get(
		context.Background(), builder.Definition.Name, metav1.GetOptions{})

	return err == nil || !k8serrors.IsNotFound(err)
}

// Delete removes a networkPolicy object from a cluster.
func (builder *NetworkPolicyBuilder) Delete() error {
	if valid, err := builder.validate(); !valid {
		return err
	}

	glog.V(100).Infof("Deleting the networkPolicy object %s from %s namespace",
		builder.Definition.Name, builder.Definition.Namespace)

	if !builder.Exists() {
		return fmt.Errorf("networkPolicy cannot be deleted because it does not exist")
	}

	err := builder.apiClient.NetworkPolicies(builder.Definition.Namespace).Delete(
		context.TODO(), builder.Definition.Name, metav1.DeleteOptions{})

	if err != nil {
		return fmt.Errorf("cannot delete MachineConfig: %w", err)
	}

	builder.Object = nil

	return err
}

// Update renovates the existing networkPolicy object with networkPolicy definition in builder.
func (builder *NetworkPolicyBuilder) Update() (*NetworkPolicyBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Updating networkPolicy %s in %s namespace ",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	builder.Object, err = builder.apiClient.NetworkPolicies(builder.Definition.Namespace).Update(
		context.TODO(), builder.Definition, metav1.UpdateOptions{})

	return builder, err
}

// validate will check that the builder and builder definition are properly initialized before
// accessing any member fields.
func (builder *NetworkPolicyBuilder) validate() (bool, error) {
	resourceCRD := "NetworkPolicy"

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
