package metallb

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/metallb/mlbtypes"
	"github.com/openshift-kni/eco-goinfra/pkg/msg"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	ipAddressPoolKind = "IPAddressPool"
)

// IPAddressPoolBuilder provides struct for the IPAddressPool object containing connection to
// the cluster and the IPAddressPool definitions.
type IPAddressPoolBuilder struct {
	Definition *mlbtypes.IPAddressPool
	Object     *mlbtypes.IPAddressPool
	apiClient  *clients.Settings
	errorMsg   string
}

// IPAddressPoolAdditionalOptions additional options for IPAddressPool object.
type IPAddressPoolAdditionalOptions func(builder *IPAddressPoolBuilder) (*IPAddressPoolBuilder, error)

// NewIPAddressPoolBuilder creates a new instance of IPAddressPoolBuilder.
func NewIPAddressPoolBuilder(
	apiClient *clients.Settings, name, nsname string, addrPool []string) *IPAddressPoolBuilder {
	glog.V(100).Infof(
		"Initializing new IPAddressPool structure with the following params: %s, %s %s",
		name, nsname, addrPool)

	builder := IPAddressPoolBuilder{
		apiClient: apiClient,
		Definition: &mlbtypes.IPAddressPool{
			TypeMeta: metav1.TypeMeta{
				Kind:       ipAddressPoolKind,
				APIVersion: fmt.Sprintf("%s/%s", APIGroup, APIVersion),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			}, Spec: mlbtypes.IPAddressPoolSpec{
				Addresses: addrPool,
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the IPAddressPool is empty")

		builder.errorMsg = "IPAddressPool 'name' cannot be empty"
	}

	if nsname == "" {
		glog.V(100).Infof("The namespace of the IPAddressPool is empty")

		builder.errorMsg = "IPAddressPool 'nsname' cannot be empty"
	}

	if len(addrPool) < 1 {
		glog.V(100).Infof("The addrPool of the IPAddressPool is empty list")

		builder.errorMsg = "IPAddressPool 'addrPool' cannot be empty list"
	}

	return &builder
}

// Get returns IPAddressPool object if found.
func (builder *IPAddressPoolBuilder) Get() (*mlbtypes.IPAddressPool, error) {
	if valid, err := builder.validate(); !valid {
		return nil, err
	}

	glog.V(100).Infof(
		"Collecting IPAddressPool object %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	unsObject, err := builder.apiClient.Resource(
		GetIPAddressPoolGVR()).Namespace(builder.Definition.Namespace).Get(
		context.TODO(), builder.Definition.Name, metav1.GetOptions{})

	if err != nil {
		glog.V(100).Infof(
			"IPAddressPool object %s doesn't exist in namespace %s",
			builder.Definition.Name, builder.Definition.Namespace)

		return nil, err
	}

	return builder.convertToStructured(unsObject)
}

// Exists checks whether the given IPAddressPool exists.
func (builder *IPAddressPoolBuilder) Exists() bool {
	if valid, _ := builder.validate(); !valid {
		return false
	}

	glog.V(100).Infof(
		"Checking if IPAddressPool %s exists in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	builder.Object, err = builder.Get()

	return err == nil || !k8serrors.IsNotFound(err)
}

// PullAddressPool pulls existing addresspool from cluster.
func PullAddressPool(apiClient *clients.Settings, name, nsname string) (*IPAddressPoolBuilder, error) {
	glog.V(100).Infof("Pulling existing addresspool name %s under namespace %s from cluster", name, nsname)

	if apiClient == nil {
		glog.V(100).Infof("The apiClient is empty")

		return nil, fmt.Errorf("addresspool 'apiClient' cannot be empty")
	}

	builder := IPAddressPoolBuilder{
		apiClient: apiClient,
		Definition: &mlbtypes.IPAddressPool{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the addresspool is empty")

		return nil, fmt.Errorf("addresspool 'name' cannot be empty")
	}

	if nsname == "" {
		glog.V(100).Infof("The namespace of the addresspool is empty")

		return nil, fmt.Errorf("addresspool 'namespace' cannot be empty")
	}

	if !builder.Exists() {
		return nil, fmt.Errorf("addresspool object %s doesn't exist in namespace %s", name, nsname)
	}

	builder.Definition = builder.Object

	return &builder, nil
}

// Create makes a IPAddressPool in the cluster and stores the created object in struct.
func (builder *IPAddressPoolBuilder) Create() (*IPAddressPoolBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Creating the IPAddressPool %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace,
	)

	var err error
	if !builder.Exists() {
		unstructuredIPAddressPool, err := runtime.DefaultUnstructuredConverter.ToUnstructured(builder.Definition)

		if err != nil {
			glog.V(100).Infof("Failed to convert structured IPAddressPool to unstructured object")

			return nil, err
		}

		unsObject, err := builder.apiClient.Resource(
			GetIPAddressPoolGVR()).Namespace(builder.Definition.Namespace).Create(
			context.TODO(), &unstructured.Unstructured{Object: unstructuredIPAddressPool}, metav1.CreateOptions{})

		if err != nil {
			glog.V(100).Infof("Failed to create IPAddressPool")

			return nil, err
		}

		builder.Object, err = builder.convertToStructured(unsObject)

		if err != nil {
			return nil, err
		}
	}

	return builder, err
}

// Delete removes IPAddressPool object from a cluster.
func (builder *IPAddressPoolBuilder) Delete() (*IPAddressPoolBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Deleting the IPAddressPool object %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace,
	)

	if !builder.Exists() {
		return builder, fmt.Errorf("IPAddressPool cannot be deleted because it does not exist")
	}

	err := builder.apiClient.Resource(
		GetIPAddressPoolGVR()).Namespace(builder.Definition.Namespace).Delete(
		context.TODO(), builder.Definition.Name, metav1.DeleteOptions{})

	if err != nil {
		return builder, fmt.Errorf("can not delete IPAddressPool: %w", err)
	}

	builder.Object = nil

	return builder, nil
}

// Update renovates the existing IPAddressPool object with the IPAddressPool definition in builder.
func (builder *IPAddressPoolBuilder) Update(force bool) (*IPAddressPoolBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Updating the IPAddressPool object %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace,
	)

	unstructuredIPAddressPool, err := runtime.DefaultUnstructuredConverter.ToUnstructured(builder.Definition)

	if err != nil {
		glog.V(100).Infof("Failed to convert structured IPAddressPool to unstructured object")

		return nil, err
	}

	_, err = builder.apiClient.Resource(
		GetIPAddressPoolGVR()).Namespace(builder.Definition.Namespace).Update(
		context.TODO(), &unstructured.Unstructured{Object: unstructuredIPAddressPool}, metav1.UpdateOptions{})

	if err != nil {
		if force {
			glog.V(100).Infof(
				msg.FailToUpdateNotification("IPAddressPool", builder.Definition.Name, builder.Definition.Namespace))

			builder, err := builder.Delete()

			if err != nil {
				glog.V(100).Infof(
					msg.FailToUpdateError("IPAddressPool", builder.Definition.Name, builder.Definition.Namespace))

				return nil, err
			}

			return builder.Create()
		}
	}

	return builder, err
}

// WithAutoAssign defines the AutoAssign bool flag placed in the IPAddressPool spec.
func (builder *IPAddressPoolBuilder) WithAutoAssign(auto bool) *IPAddressPoolBuilder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof(
		"Creating IPAddressPool %s in namespace %s with this autoAssign flag: %t",
		builder.Definition.Name, builder.Definition.Namespace, auto)

	builder.Definition.Spec.AutoAssign = &auto

	return builder
}

// WithAvoidBuggyIPs defines the AvoidBuggyIPs bool flag placed in the IPAddressPool spec.
func (builder *IPAddressPoolBuilder) WithAvoidBuggyIPs(avoid bool) *IPAddressPoolBuilder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof(
		"Creating IPAddressPool %s in namespace %s with this avoidBuggyIPs flag: %t",
		builder.Definition.Name, builder.Definition.Namespace, avoid)

	builder.Definition.Spec.AvoidBuggyIPs = avoid

	return builder
}

// WithOptions creates IPAddressPool with generic mutation options.
func (builder *IPAddressPoolBuilder) WithOptions(options ...IPAddressPoolAdditionalOptions) *IPAddressPoolBuilder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof("Setting IPAddressPool additional options")

	if builder.Definition == nil {
		glog.V(100).Infof("The IPAddressPool is undefined")

		builder.errorMsg = msg.UndefinedCrdObjectErrString("IPAddressPool")
	}

	if builder.errorMsg != "" {
		return builder
	}

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

// GetIPAddressPoolGVR returns ipaddresspool's GroupVersionResource, which could be used for Clean function.
func GetIPAddressPoolGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group: APIGroup, Version: APIVersion, Resource: "ipaddresspools",
	}
}

// validate will check that the builder and builder definition are properly initialized before
// accessing any member fields.
func (builder *IPAddressPoolBuilder) validate() (bool, error) {
	resourceCRD := "IPAddressPool"

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

func (builder *IPAddressPoolBuilder) convertToStructured(
	unsObject *unstructured.Unstructured) (*mlbtypes.IPAddressPool, error) {
	ipAddressPool := &mlbtypes.IPAddressPool{}

	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unsObject.Object, ipAddressPool)
	if err != nil {
		glog.V(100).Infof(
			"Failed to convert from unstructured to ipAddressPool object in namespace %s",
			builder.Definition.Name, builder.Definition.Namespace)

		return nil, err
	}

	return ipAddressPool, err
}
