package metallb

import (
	"context"
	"fmt"

	"github.com/openshift-kni/eco-gotests/pkg/msg"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-gotests/pkg/clients"
	metalLbV1Beta1 "go.universe.tf/metallb/api/v1beta1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	goclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// IPAddressPoolBuilder provides struct for the IPAddressPool object containing connection to
// the cluster and the IPAddressPool definitions.
type IPAddressPoolBuilder struct {
	Definition *metalLbV1Beta1.IPAddressPool
	Object     *metalLbV1Beta1.IPAddressPool
	apiClient  *clients.Settings
	errorMsg   string
}

// NewIPAddressPoolBuilder creates a new instance of IPAddressPoolBuilder.
func NewIPAddressPoolBuilder(
	apiClient *clients.Settings, name, nsname string, addrPool []string) *IPAddressPoolBuilder {
	glog.V(100).Infof(
		"Initializing new IPAddressPool structure with the following params: %s, %s %s",
		name, nsname, addrPool)

	builder := IPAddressPoolBuilder{
		apiClient: apiClient,
		Definition: &metalLbV1Beta1.IPAddressPool{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			}, Spec: metalLbV1Beta1.IPAddressPoolSpec{
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
func (builder *IPAddressPoolBuilder) Get() (*metalLbV1Beta1.IPAddressPool, error) {
	glog.V(100).Infof(
		"Collecting IPAddressPool object %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	ipAddressPool := &metalLbV1Beta1.IPAddressPool{}
	err := builder.apiClient.Get(context.TODO(), goclient.ObjectKey{
		Name:      builder.Definition.Name,
		Namespace: builder.Definition.Namespace,
	}, ipAddressPool)

	if err != nil {
		glog.V(100).Infof(
			"IPAddressPool object %s doesn't exist in namespace %s",
			builder.Definition.Name, builder.Definition.Namespace)

		return nil, err
	}

	return ipAddressPool, err
}

// Exists checks whether the given IPAddressPool exists.
func (builder *IPAddressPoolBuilder) Exists() bool {
	glog.V(100).Infof(
		"Checking if IPAddressPool %s exists in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	builder.Object, err = builder.Get()

	return err == nil || !k8serrors.IsNotFound(err)
}

// Create makes a IPAddressPool in the cluster and stores the created object in struct.
func (builder *IPAddressPoolBuilder) Create() (*IPAddressPoolBuilder, error) {
	glog.V(100).Infof("Creating the IPAddressPool %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace,
	)

	if builder.errorMsg != "" {
		return nil, fmt.Errorf(builder.errorMsg)
	}

	var err error
	if !builder.Exists() {
		err = builder.apiClient.Create(context.TODO(), builder.Definition)
		if err == nil {
			builder.Object = builder.Definition
		}
	}

	return builder, err
}

// Delete removes IPAddressPool object from a cluster.
func (builder *IPAddressPoolBuilder) Delete() (*IPAddressPoolBuilder, error) {
	glog.V(100).Infof("Deleting the IPAddressPool object %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace,
	)

	if !builder.Exists() {
		return builder, fmt.Errorf("IPAddressPool cannot be deleted because it does not exist")
	}

	err := builder.apiClient.Delete(context.TODO(), builder.Definition)

	if err != nil {
		return builder, fmt.Errorf("can not delete IPAddressPool: %w", err)
	}

	builder.Object = nil

	return builder, nil
}

// Update renovates the existing IPAddressPool object with the IPAddressPool definition in builder.
func (builder *IPAddressPoolBuilder) Update(force bool) (*IPAddressPoolBuilder, error) {
	glog.V(100).Infof("Updating the IPAddressPool object %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace,
	)

	if builder.errorMsg != "" {
		return nil, fmt.Errorf(builder.errorMsg)
	}

	err := builder.apiClient.Update(context.TODO(), builder.Definition)

	if err != nil {
		if force {
			glog.V(100).Infof(
				"Failed to update the IPAddressPool object %s in namespace %s. "+
					"Note: Force flag set, executed delete/create methods instead",
				builder.Definition.Name, builder.Definition.Namespace,
			)

			builder, err := builder.Delete()

			if err != nil {
				glog.V(100).Infof(
					"Failed to update the IPAddressPool object %s in namespace %s, "+
						"due to error in delete function",
					builder.Definition.Name, builder.Definition.Namespace,
				)

				return nil, err
			}

			return builder.Create()
		}
	}

	return builder, err
}

// WithAutoAssign defines the AutoAssign bool flag placed in the IPAddressPool spec.
func (builder *IPAddressPoolBuilder) WithAutoAssign(auto bool) *IPAddressPoolBuilder {
	glog.V(100).Infof(
		"Creating IPAddressPool %s in namespace %s with this autoAssign flag: %t",
		builder.Definition.Name, builder.Definition.Namespace, auto)

	if builder.Definition == nil {
		builder.errorMsg = msg.UndefinedCrdObjectErrString("IPAddressPool")
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.AutoAssign = &auto

	return builder
}

// WithAvoidBuggyIPs defines the AvoidBuggyIPs bool flag placed in the IPAddressPool spec.
func (builder *IPAddressPoolBuilder) WithAvoidBuggyIPs(avoid bool) *IPAddressPoolBuilder {
	glog.V(100).Infof(
		"Creating IPAddressPool %s in namespace %s with this avoidBuggyIPs flag: %t",
		builder.Definition.Name, builder.Definition.Namespace, avoid)

	if builder.Definition == nil {
		builder.errorMsg = msg.UndefinedCrdObjectErrString("IPAddressPool")
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.AvoidBuggyIPs = avoid

	return builder
}

// GetIPAddressPoolGVR returns ipaddresspool's GroupVersionResource, which could be used for Clean function.
func GetIPAddressPoolGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group: "metallb.io", Version: "v1beta1", Resource: "ipaddresspools",
	}
}
