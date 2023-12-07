package networkpolicy

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/k8snetworkplumbingwg/multi-networkpolicy/pkg/apis/k8s.cni.cncf.io/v1beta1"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/msg"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// MultiNetworkPolicyBuilder provides struct for MultiNetworkPolicy object.
type MultiNetworkPolicyBuilder struct {
	// MultiNetworkPolicy definition. Used to create MultiNetworkPolicy object with minimum set of required elements.
	Definition *v1beta1.MultiNetworkPolicy
	// Created MultiNetworkPolicy object on the cluster.
	Object *v1beta1.MultiNetworkPolicy
	// api client to interact with the cluster.
	apiClient *clients.Settings
	// errorMsg is processed before MultiNetworkPolicy object is created.
	errorMsg string
}

// NewMultiNetworkPolicyBuilder method creates new instance of builder.
func NewMultiNetworkPolicyBuilder(apiClient *clients.Settings, name, nsname string) *MultiNetworkPolicyBuilder {
	glog.V(100).Infof(
		"Initializing new MultiNetworkPolicyBuilder structure with the following params: name: %s, namespace: %s",
		name, nsname)

	builder := &MultiNetworkPolicyBuilder{
		apiClient: apiClient,
		Definition: &v1beta1.MultiNetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the MultiNetworkPolicy is empty")

		builder.errorMsg = "The MultiNetworkPolicy 'name' cannot be empty"
	}

	if nsname == "" {
		glog.V(100).Infof("The namespace of the MultiNetworkPolicy is empty")

		builder.errorMsg = "The MultiNetworkPolicy 'namespace' cannot be empty"
	}

	return builder
}

// WithPodSelector adds podSelector to MultiNetworkPolicy.
func (builder *MultiNetworkPolicyBuilder) WithPodSelector(podSelector metav1.LabelSelector) *MultiNetworkPolicyBuilder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof(
		"Creating MultiNetworkPolicy %s in %s namespace with the podSelector defined: %v",
		builder.Definition.Name, builder.Definition.Namespace, podSelector)

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.PodSelector = podSelector

	return builder
}

// WithNetwork adds network name to the MultiNetworkPolicy.
func (builder *MultiNetworkPolicyBuilder) WithNetwork(networkName string) *MultiNetworkPolicyBuilder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof(
		"Creating MultiNetworkPolicy %s in %s namespace with the networkName defined: %v",
		builder.Definition.Name, builder.Definition.Namespace, networkName)

	if networkName == "" {
		glog.V(100).Infof("The networkName can not be empty string")

		builder.errorMsg = "The networkName is an empty string"
	}

	if builder.errorMsg != "" {
		return builder
	}

	if builder.Definition.Annotations == nil {
		builder.Definition.Annotations = make(map[string]string)
	}

	builder.Definition.Annotations["k8s.v1.cni.cncf.io/policy-for"] = networkName

	return builder
}

// WithEmptyIngress adds empty ingress rule to the MultiNetworkPolicy. Empty ingress denies all.
func (builder *MultiNetworkPolicyBuilder) WithEmptyIngress() *MultiNetworkPolicyBuilder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof(
		"Creating MultiNetworkPolicy %s in %s namespace with the empty Ingress rule deny all",
		builder.Definition.Name, builder.Definition.Namespace)

	builder.Definition.Spec.Ingress = []v1beta1.MultiNetworkPolicyIngressRule{}

	return builder
}

// WithIngressRule adds Ingress rule to the MultiNetworkPolicy. Empty rule is allowed and works as allow all traffic.
func (builder *MultiNetworkPolicyBuilder) WithIngressRule(
	ingressRule v1beta1.MultiNetworkPolicyIngressRule) *MultiNetworkPolicyBuilder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof(
		"Creating multiNetworkPolicy %s in %s namespace with the Ingress rule defined: %v",
		builder.Definition.Name, builder.Definition.Namespace, ingressRule)

	builder.Definition.Spec.Ingress = append(builder.Definition.Spec.Ingress, ingressRule)

	return builder
}

// WithEgressRule adds Egress rule to the MultiNetworkPolicy. Empty rule is allowed and works as allow all traffic.
func (builder *MultiNetworkPolicyBuilder) WithEgressRule(
	egressRule v1beta1.MultiNetworkPolicyEgressRule) *MultiNetworkPolicyBuilder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof(
		"Creating multiNetworkPolicy %s in %s namespace with the Egress rule defined: %v",
		builder.Definition.Name, builder.Definition.Namespace, egressRule)

	builder.Definition.Spec.Egress = append(builder.Definition.Spec.Egress, egressRule)

	return builder
}

// WithPolicyType adds policyType to the MultiNetworkPolicy.
func (builder *MultiNetworkPolicyBuilder) WithPolicyType(
	policyType v1beta1.MultiPolicyType) *MultiNetworkPolicyBuilder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof(
		"Creating multiNetworkPolicy %s in %s namespace with the Policy type defined: %v",
		builder.Definition.Name, builder.Definition.Namespace, policyType)

	if policyType == "" {
		glog.V(100).Infof("The policy type can not be an empty string")

		builder.errorMsg = "The policy Type is an empty string"
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.PolicyTypes = append(builder.Definition.Spec.PolicyTypes, policyType)

	return builder
}

// PullMultiNetworkPolicy loads an existing MultiNetworkPolicy into the Builder struct.
func PullMultiNetworkPolicy(apiClient *clients.Settings, name, nsname string) (*MultiNetworkPolicyBuilder, error) {
	glog.V(100).Infof("Pulling existing MultiNetworkPolicy name: %s, namespace: %s", name, nsname)

	builder := MultiNetworkPolicyBuilder{
		apiClient: apiClient,
		Definition: &v1beta1.MultiNetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the MultiNetworkPolicy is empty")

		builder.errorMsg = "MultiNetworkPolicy 'name' cannot be empty"
	}

	if nsname == "" {
		glog.V(100).Infof("The namespace of the MultiNetworkPolicy is empty")

		builder.errorMsg = "MultiNetworkPolicy 'namespace' cannot be empty"
	}

	if builder.errorMsg != "" {
		return nil, fmt.Errorf("failed to pull MultiNetworkPolicy object due to the following error: %s",
			builder.errorMsg)
	}

	if !builder.Exists() {
		glog.V(100).Infof(
			"Failed to pull MultiNetworkPolicy object %s from namespace %s. Object doesn't exist",
			name, nsname)

		return nil, fmt.Errorf("MultiNetworkPolicy object %s doesn't exist in namespace %s", name, nsname)
	}

	builder.Definition = builder.Object

	return &builder, nil
}

// Create makes a MultiNetworkPolicy in cluster and stores the created object in struct.
func (builder *MultiNetworkPolicyBuilder) Create() (*MultiNetworkPolicyBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Creating the MultiNetworkPolicy %s in %s namespace",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	if !builder.Exists() {
		builder.Object, err = builder.apiClient.MultiNetworkPolicies(builder.Definition.Namespace).Create(
			context.TODO(), builder.Definition, metav1.CreateOptions{})
	}

	return builder, err
}

// Exists checks whether the given MultiNetworkPolicy exists.
func (builder *MultiNetworkPolicyBuilder) Exists() bool {
	if valid, _ := builder.validate(); !valid {
		return false
	}

	glog.V(100).Infof("Checking if MultiNetworkPolicy %s exists in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	builder.Object, err = builder.apiClient.MultiNetworkPolicies(builder.Definition.Namespace).Get(
		context.Background(), builder.Definition.Name, metav1.GetOptions{})

	return err == nil || !k8serrors.IsNotFound(err)
}

// Delete removes a MultiNetworkPolicy object from a cluster.
func (builder *MultiNetworkPolicyBuilder) Delete() error {
	if valid, err := builder.validate(); !valid {
		return err
	}

	glog.V(100).Infof("Deleting the MultiNetworkPolicy object %s from %s namespace",
		builder.Definition.Name, builder.Definition.Namespace)

	if !builder.Exists() {
		return fmt.Errorf("multiNetworkPolicy cannot be deleted because it does not exist")
	}

	err := builder.apiClient.MultiNetworkPolicies(builder.Definition.Namespace).Delete(
		context.TODO(), builder.Definition.Name, metav1.DeleteOptions{})

	if err != nil {
		return fmt.Errorf("cannot delete MultiNetworkPolicy: %w", err)
	}

	builder.Object = nil

	return err
}

// Update renovates the existing MultiNetworkPolicy object with MultiNetworkPolicy definition in builder.
func (builder *MultiNetworkPolicyBuilder) Update() (*MultiNetworkPolicyBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Updating MultiNetworkPolicy %s in %s namespace ",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	builder.Object, err = builder.apiClient.MultiNetworkPolicies(builder.Definition.Namespace).Update(
		context.TODO(), builder.Definition, metav1.UpdateOptions{})

	return builder, err
}

// GetMultiNetworkGVR returns MultiNetworkPolicy's GroupVersionResource which could be used for Clean function.
func GetMultiNetworkGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: "k8s.cni.cncf.io", Version: "v1beta1", Resource: "multi-networkpolicies"}
}

// validate will check that the builder and builder definition are properly initialized before
// accessing any member fields.
func (builder *MultiNetworkPolicyBuilder) validate() (bool, error) {
	resourceCRD := "MultiNetworkPolicy"

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
