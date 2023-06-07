package metallb

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-gotests/pkg/clients"
	"github.com/openshift-kni/eco-gotests/pkg/msg"
	metalLbV1Beta "go.universe.tf/metallb/api/v1beta1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	goclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// BGPAdvertisementBuilder provides struct for the BGPAdvertisement object containing connection to
// the cluster and the BGPAdvertisement definitions.
type BGPAdvertisementBuilder struct {
	Definition *metalLbV1Beta.BGPAdvertisement
	Object     *metalLbV1Beta.BGPAdvertisement
	apiClient  *clients.Settings
	errorMsg   string
}

// NewBGPAdvertisementBuilder creates a new instance of BGPAdvertisementBuilder.
func NewBGPAdvertisementBuilder(apiClient *clients.Settings, name, nsname string) *BGPAdvertisementBuilder {
	glog.V(100).Infof(
		"Initializing new BGPAdvertisement structure with the following params: %s, %s",
		name, nsname)

	builder := BGPAdvertisementBuilder{
		apiClient: apiClient,
		Definition: &metalLbV1Beta.BGPAdvertisement{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			}, Spec: metalLbV1Beta.BGPAdvertisementSpec{},
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the BGPAdvertisement is empty")

		builder.errorMsg = "BGPAdvertisement 'name' cannot be empty"
	}

	if nsname == "" {
		glog.V(100).Infof("The namespace of the BGPAdvertisement is empty")

		builder.errorMsg = "BGPAdvertisement 'nsname' cannot be empty"
	}

	return &builder
}

// Exists checks whether the given BGPAdvertisement exists.
func (builder *BGPAdvertisementBuilder) Exists() bool {
	glog.V(100).Infof(
		"Checking if BGPAdvertisement %s exists in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	builder.Object, err = builder.Get()

	return err == nil || !k8serrors.IsNotFound(err)
}

// Get returns BGPAdvertisement object if found.
func (builder *BGPAdvertisementBuilder) Get() (*metalLbV1Beta.BGPAdvertisement, error) {
	glog.V(100).Infof(
		"Collecting BGPAdvertisement object %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	metalLb := &metalLbV1Beta.BGPAdvertisement{}
	err := builder.apiClient.Get(context.TODO(), goclient.ObjectKey{
		Name:      builder.Definition.Name,
		Namespace: builder.Definition.Namespace,
	}, metalLb)

	if err != nil {
		glog.V(100).Infof(
			"BGPAdvertisement object %s doesn't exist in namespace %s",
			builder.Definition.Name, builder.Definition.Namespace)

		return nil, err
	}

	return metalLb, err
}

// Create makes a BGPAdvertisement in the cluster and stores the created object in struct.
func (builder *BGPAdvertisementBuilder) Create() (*BGPAdvertisementBuilder, error) {
	glog.V(100).Infof("Creating the BGPAdvertisement %s in namespace %s",
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

// Delete removes BGPAdvertisement object from a cluster.
func (builder *BGPAdvertisementBuilder) Delete() (*BGPAdvertisementBuilder, error) {
	glog.V(100).Infof("Deleting the BGPAdvertisement object %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace,
	)

	if !builder.Exists() {
		return builder, fmt.Errorf("BGPAdvertisement cannot be deleted because it does not exist")
	}

	err := builder.apiClient.Delete(context.TODO(), builder.Definition)

	if err != nil {
		return builder, fmt.Errorf("can not delete BGPAdvertisement: %w", err)
	}

	builder.Object = nil

	return builder, nil
}

// Update renovates the existing BGPAdvertisement object with the BGPAdvertisement definition in builder.
func (builder *BGPAdvertisementBuilder) Update(force bool) (*BGPAdvertisementBuilder, error) {
	glog.V(100).Infof("Updating the BGPAdvertisement object %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace,
	)

	if builder.errorMsg != "" {
		return nil, fmt.Errorf(builder.errorMsg)
	}

	if !builder.Exists() {
		glog.V(100).Infof(
			"Failed to update the BGPAdvertisement object %s in namespace %s. "+
				"Resource doesn't exist",
			builder.Definition.Name, builder.Definition.Namespace,
		)

		return nil, fmt.Errorf("failed to update BGPAdvertisement, resource doesn't exist")
	}

	builder.Object.Spec = builder.Definition.Spec
	err := builder.apiClient.Update(context.TODO(), builder.Object)

	if err != nil {
		if force {
			glog.V(100).Infof(
				"Failed to update the BGPAdvertisement object %s in namespace %s. "+
					"Note: Force flag set, executed delete/create methods instead",
				builder.Definition.Name, builder.Definition.Namespace,
			)

			builder, err := builder.Delete()

			if err != nil {
				glog.V(100).Infof(
					"Failed to update the BGPAdvertisement object %s in namespace %s, "+
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

// WithAggregationLength4 adds the specified AggregationLength to the BGPAdvertisement.
func (builder *BGPAdvertisementBuilder) WithAggregationLength4(aggregationLength int32) *BGPAdvertisementBuilder {
	glog.V(100).Infof(
		"Creating BGPAdvertisement %s in namespace %s with aggregationLength: %d",
		builder.Definition.Name, builder.Definition.Namespace, aggregationLength)

	if builder.Definition == nil {
		builder.errorMsg = msg.UndefinedCrdObjectErrString("BGPAdvertisement")
	}

	if aggregationLength < 0 || aggregationLength > 32 {
		builder.errorMsg = fmt.Sprintf("AggregationLength %d is invalid, the value shoud be in range 0...32",
			aggregationLength)
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.AggregationLength = &aggregationLength

	return builder
}

// WithAggregationLength6 adds the specified AggregationLengthV6 to the BGPAdvertisement.
func (builder *BGPAdvertisementBuilder) WithAggregationLength6(aggregationLength int32) *BGPAdvertisementBuilder {
	glog.V(100).Infof(
		"Creating BGPAdvertisement %s in namespace %s with aggregationLength6: %d",
		builder.Definition.Name, builder.Definition.Namespace, aggregationLength)

	if builder.Definition == nil {
		builder.errorMsg = msg.UndefinedCrdObjectErrString("BGPAdvertisement")
	}

	if !(aggregationLength < 0 || aggregationLength > 128) {
		builder.errorMsg = fmt.Sprintf("AggregationLength %d is invalid, the value shoud be in range 0...128",
			aggregationLength)
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.AggregationLengthV6 = &aggregationLength

	return builder
}

// WithLocalPref adds the specified LocalPref to the BGPAdvertisement.
func (builder *BGPAdvertisementBuilder) WithLocalPref(localPreference uint32) *BGPAdvertisementBuilder {
	glog.V(100).Infof(
		"Creating BGPAdvertisement %s in namespace %s with LocalPref: %d",
		builder.Definition.Name, builder.Definition.Namespace, localPreference)

	if builder.Definition == nil {
		builder.errorMsg = msg.UndefinedCrdObjectErrString("BGPAdvertisement")
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.LocalPref = localPreference

	return builder
}

// WithCommunities adds the specified Communities to the BGPAdvertisement.
func (builder *BGPAdvertisementBuilder) WithCommunities(communities []string) *BGPAdvertisementBuilder {
	glog.V(100).Infof(
		"Creating BGPAdvertisement %s in namespace %s with Communities: %s",
		builder.Definition.Name, builder.Definition.Namespace, communities)

	if builder.Definition == nil {
		builder.errorMsg = msg.UndefinedCrdObjectErrString("BGPAdvertisement")
	}

	if len(communities) < 1 {
		builder.errorMsg = "error: community setting is empty list, the list should contain at least one element"
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.Communities = communities

	return builder
}

// WithIPAddressPools adds the specified IPAddressPools to the BGPAdvertisement.
func (builder *BGPAdvertisementBuilder) WithIPAddressPools(ipAddressPools []string) *BGPAdvertisementBuilder {
	glog.V(100).Infof(
		"Creating BGPAdvertisement %s in namespace %s with IPAddressPools: %s",
		builder.Definition.Name, builder.Definition.Namespace, ipAddressPools)

	if builder.Definition == nil {
		builder.errorMsg = msg.UndefinedCrdObjectErrString("BGPAdvertisement")
	}

	if len(ipAddressPools) < 1 {
		builder.errorMsg = "error: IPAddressPools setting is empty list, the list should contain at least one element"
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.IPAddressPools = ipAddressPools

	return builder
}

// WithIPAddressPoolsSelectors adds the specified IPAddressPoolSelectors to the BGPAdvertisement.
func (builder *BGPAdvertisementBuilder) WithIPAddressPoolsSelectors(
	poolSelector []metaV1.LabelSelector) *BGPAdvertisementBuilder {
	glog.V(100).Infof(
		"Creating BGPAdvertisement %s in namespace %s with IPAddressPoolSelectors: %s",
		builder.Definition.Name, builder.Definition.Namespace, poolSelector)

	if builder.Definition == nil {
		builder.errorMsg = msg.UndefinedCrdObjectErrString("BGPAdvertisement")
	}

	if len(poolSelector) < 1 {
		builder.errorMsg = "error: IPAddressPoolSelectors setting is empty list, " +
			"the list should contain at least one element"
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.IPAddressPoolSelectors = poolSelector

	return builder
}

// WithNodeSelector adds the specified NodeSelectors to the BGPAdvertisement.
func (builder *BGPAdvertisementBuilder) WithNodeSelector(
	nodeSelectors []metaV1.LabelSelector) *BGPAdvertisementBuilder {
	glog.V(100).Infof(
		"Creating BGPAdvertisement %s in namespace %s with WithIPAddressPools: %v",
		builder.Definition.Name, builder.Definition.Namespace, nodeSelectors)

	if builder.Definition == nil {
		builder.errorMsg = msg.UndefinedCrdObjectErrString("BGPAdvertisement")
	}

	if len(nodeSelectors) < 1 {
		builder.errorMsg = "error: nodeSelectors setting is empty list, the list should contain at least one element"
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.NodeSelectors = nodeSelectors

	return builder
}

// WithPeers adds the specified Peers to the BGPAdvertisement.
func (builder *BGPAdvertisementBuilder) WithPeers(peers []string) *BGPAdvertisementBuilder {
	glog.V(100).Infof(
		"Creating BGPAdvertisement %s in namespace %s with Peers: %v",
		builder.Definition.Name, builder.Definition.Namespace, peers)

	if builder.Definition == nil {
		builder.errorMsg = msg.UndefinedCrdObjectErrString("BGPAdvertisement")
	}

	if len(peers) < 1 {
		builder.errorMsg = "error: peers setting is empty list, the list should contain at least one element"
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.Peers = peers

	return builder
}

// GetBGPAdvertisementGVR returns bgpadvertisement's GroupVersionResource, which could be used for Clean function.
func GetBGPAdvertisementGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group: "metallb.io", Version: "v1beta1", Resource: "bgpadvertisements",
	}
}
