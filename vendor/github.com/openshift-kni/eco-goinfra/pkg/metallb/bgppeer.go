package metallb

import (
	"context"
	"fmt"
	"net"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/msg"
	"github.com/openshift-kni/eco-goinfra/pkg/schemes/metallb/mlbtypes"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	bpgPeerKind = "BGPPeer"
)

// BGPPeerBuilder provides struct for the BGPPeer object containing connection to
// the cluster and the BGPPeer definitions.
type BGPPeerBuilder struct {
	Definition *mlbtypes.BGPPeer
	Object     *mlbtypes.BGPPeer
	apiClient  *clients.Settings
	errorMsg   string
}

// BGPPeerAdditionalOptions additional options for BGPPeer object.
type BGPPeerAdditionalOptions func(builder *BGPPeerBuilder) (*BGPPeerBuilder, error)

// NewBPGPeerBuilder creates a new instance of BGPPeer.
func NewBPGPeerBuilder(
	apiClient *clients.Settings, name, nsname, peerIP string, asn, remoteASN uint32) *BGPPeerBuilder {
	glog.V(100).Infof(
		"Initializing new BGPPeer structure with the following params: %s, %s %s %d %d",
		name, nsname, peerIP, asn, remoteASN)

	builder := BGPPeerBuilder{
		apiClient: apiClient,
		Definition: &mlbtypes.BGPPeer{
			TypeMeta: metav1.TypeMeta{
				Kind:       bpgPeerKind,
				APIVersion: fmt.Sprintf("%s/%s", APIGroup, APIVersion),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			}, Spec: mlbtypes.BGPPeerSpec{
				MyASN:   asn,
				ASN:     remoteASN,
				Address: peerIP,
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the BGPPeer is empty")

		builder.errorMsg = "BGPPeer 'name' cannot be empty"
	}

	if nsname == "" {
		glog.V(100).Infof("The namespace of the BGPPeer is empty")

		builder.errorMsg = "BGPPeer 'nsname' cannot be empty"
	}

	if net.ParseIP(peerIP) == nil {
		glog.V(100).Infof("The peerIP of the BGPPeer contains invalid ip address %s", peerIP)

		builder.errorMsg = "BGPPeer 'peerIP' of the BGPPeer contains invalid ip address"
	}

	return &builder
}

// Get returns BGPPeer object if found.
func (builder *BGPPeerBuilder) Get() (*mlbtypes.BGPPeer, error) {
	if valid, err := builder.validate(); !valid {
		return nil, err
	}

	glog.V(100).Infof(
		"Collecting BGPPeer object %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	unsObject, err := builder.apiClient.Resource(
		GetBGPPeerGVR()).Namespace(builder.Definition.Namespace).Get(
		context.TODO(), builder.Definition.Name, metav1.GetOptions{})

	if err != nil {
		glog.V(100).Infof(
			"Failed to Unmarshal BGPPeer: unstructured object to structure in namespace %s",
			builder.Definition.Name, builder.Definition.Namespace)

		return nil, err
	}

	return builder.convertToStructured(unsObject)
}

// Exists checks whether the given BGPPeer exists.
func (builder *BGPPeerBuilder) Exists() bool {
	if valid, _ := builder.validate(); !valid {
		return false
	}

	glog.V(100).Infof(
		"Checking if BGPPeer %s exists in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	builder.Object, err = builder.Get()

	return err == nil || !k8serrors.IsNotFound(err)
}

// PullBGPPeer pulls existing bgppeer from cluster.
func PullBGPPeer(apiClient *clients.Settings, name, nsname string) (*BGPPeerBuilder, error) {
	glog.V(100).Infof("Pulling existing bgppeer name %s under namespace %s from cluster", name, nsname)

	builder := BGPPeerBuilder{
		apiClient: apiClient,
		Definition: &mlbtypes.BGPPeer{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the bgppeer is empty")

		builder.errorMsg = "bgppeer 'name' cannot be empty"
	}

	if nsname == "" {
		glog.V(100).Infof("The namespace of the bgppeer is empty")

		builder.errorMsg = "bgppeer 'namespace' cannot be empty"
	}

	if !builder.Exists() {
		return nil, fmt.Errorf("bgppeer object %s does not exist in namespace %s", name, nsname)
	}

	builder.Definition = builder.Object

	return &builder, nil
}

// Create makes a BGPPeer in the cluster and stores the created object in struct.
func (builder *BGPPeerBuilder) Create() (*BGPPeerBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Creating the BGPPeer %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace,
	)

	var err error
	if !builder.Exists() {
		unstructuredBgpPeer, err := runtime.DefaultUnstructuredConverter.ToUnstructured(builder.Definition)

		if err != nil {
			glog.V(100).Infof("Failed to convert structured BGPPeer to unstructured object")

			return nil, err
		}

		unsObject, err := builder.apiClient.Resource(
			GetBGPPeerGVR()).Namespace(builder.Definition.Namespace).Create(
			context.TODO(), &unstructured.Unstructured{Object: unstructuredBgpPeer}, metav1.CreateOptions{})

		if err != nil {
			glog.V(100).Infof("Failed to create BGPPeer")

			return nil, err
		}

		builder.Object, err = builder.convertToStructured(unsObject)

		if err != nil {
			return nil, err
		}
	}

	return builder, err
}

// Delete removes BGPPeer object from a cluster.
func (builder *BGPPeerBuilder) Delete() (*BGPPeerBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Deleting the BGPPeer object %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace,
	)

	if !builder.Exists() {
		return builder, fmt.Errorf("BGPPeer cannot be deleted because it does not exist")
	}

	err := builder.apiClient.Resource(
		GetBGPPeerGVR()).Namespace(builder.Definition.Namespace).Delete(
		context.TODO(), builder.Definition.Name, metav1.DeleteOptions{})

	if err != nil {
		return builder, fmt.Errorf("can not delete BGPPeer: %w", err)
	}

	builder.Object = nil

	return builder, nil
}

// Update renovates the existing BGPPeer object with the BGPPeer definition in builder.
func (builder *BGPPeerBuilder) Update(force bool) (*BGPPeerBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Updating the BGPPeer object %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace,
	)

	unstructuredBgpPeer, err := runtime.DefaultUnstructuredConverter.ToUnstructured(builder.Definition)

	if err != nil {
		glog.V(100).Infof("Failed to convert structured BGPPeer to unstructured object")

		return nil, err
	}

	_, err = builder.apiClient.Resource(
		GetBGPPeerGVR()).Namespace(builder.Definition.Namespace).Update(
		context.TODO(), &unstructured.Unstructured{Object: unstructuredBgpPeer}, metav1.UpdateOptions{})

	if err != nil {
		if force {
			glog.V(100).Infof(
				msg.FailToUpdateNotification("BGPPeer", builder.Definition.Name, builder.Definition.Namespace))

			builder, err := builder.Delete()

			if err != nil {
				glog.V(100).Infof(
					msg.FailToUpdateError("BGPPeer", builder.Definition.Name, builder.Definition.Namespace))

				return nil, err
			}

			return builder.Create()
		}
	}

	return builder, err
}

// WithRouterID defines the routerID placed in the BGPPeer spec.
func (builder *BGPPeerBuilder) WithRouterID(routerID string) *BGPPeerBuilder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof(
		"Creating BGPPeer %s in namespace %s with this routerID: %s",
		builder.Definition.Name, builder.Definition.Namespace, routerID)

	if net.ParseIP(routerID) == nil {
		glog.V(100).Infof("The routerID of the BGPPeer contains invalid ip address %s, "+
			"routerID should be present in ip address format", routerID)

		builder.errorMsg = fmt.Sprintf("the routerID of the BGPPeer contains invalid ip address %s", routerID)
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.RouterID = routerID

	return builder
}

// WithBFDProfile defines the bfdProfile placed in the BGPPeer spec.
func (builder *BGPPeerBuilder) WithBFDProfile(bfdProfile string) *BGPPeerBuilder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof(
		"Creating BGPPeer %s in namespace %s with this bfdProfile: %s",
		builder.Definition.Name, builder.Definition.Namespace, bfdProfile)

	if bfdProfile == "" {
		glog.V(100).Infof("The bfdProfile of the BGPPeer can not be empty string")

		builder.errorMsg = "The bfdProfile is empty string"
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.BFDProfile = bfdProfile

	return builder
}

// WithSRCAddress defines the SRCAddress placed in the BGPPeer spec.
func (builder *BGPPeerBuilder) WithSRCAddress(srcAddress string) *BGPPeerBuilder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof(
		"Creating BGPPeer %s in namespace %s with this srcAddress: %s",
		builder.Definition.Name, builder.Definition.Namespace, srcAddress)

	if net.ParseIP(srcAddress) == nil {
		glog.V(100).Infof("The srcAddress of the BGPPeer contains invalid ip address %s, "+
			"srcAddress should be present in ip address format", srcAddress)

		builder.errorMsg = fmt.Sprintf("the srcAddress of the BGPPeer contains invalid ip address %s", srcAddress)
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.SrcAddress = srcAddress

	return builder
}

// WithPort defines the port placed in the BGPPeer spec.
func (builder *BGPPeerBuilder) WithPort(port uint16) *BGPPeerBuilder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof(
		"Creating BGPPeer %s in namespace %s with this port: %d",
		builder.Definition.Name, builder.Definition.Namespace, port)

	builder.Definition.Spec.Port = port

	return builder
}

// WithHoldTime defines the holdTime placed in the BGPPeer spec.
func (builder *BGPPeerBuilder) WithHoldTime(holdTime metav1.Duration) *BGPPeerBuilder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof(
		"Creating BGPPeer %s in namespace %s with this holdTime: %s",
		builder.Definition.Name, builder.Definition.Namespace, holdTime)

	builder.Definition.Spec.HoldTime = holdTime

	return builder
}

// WithKeepalive defines the keepAliveTime placed in the BGPPeer spec.
func (builder *BGPPeerBuilder) WithKeepalive(keepalive metav1.Duration) *BGPPeerBuilder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof(
		"Creating BGPPeer %s in namespace %s with this keepalive: %s",
		builder.Definition.Name, builder.Definition.Namespace, keepalive)

	builder.Definition.Spec.KeepaliveTime = keepalive

	return builder
}

// WithNodeSelector defines the nodeSelector placed in the BGPPeer spec.
func (builder *BGPPeerBuilder) WithNodeSelector(nodeSelector map[string]string) *BGPPeerBuilder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof(
		"Creating BGPPeer %s in namespace %s with this nodeSelector: %s",
		builder.Definition.Name, builder.Definition.Namespace, nodeSelector)

	if len(nodeSelector) == 0 {
		glog.V(100).Infof("Can not redefine BGPPeer with empty nodeSelector map")

		builder.errorMsg = "BGPPeer 'nodeSelector' cannot be empty map"
	}

	if builder.errorMsg != "" {
		return builder
	}

	ndSelector := []mlbtypes.NodeSelector{{MatchLabels: nodeSelector}}
	builder.Definition.Spec.NodeSelectors = ndSelector

	return builder
}

// WithPassword defines the password placed in the BGPPeer spec.
func (builder *BGPPeerBuilder) WithPassword(password string) *BGPPeerBuilder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof(
		"Creating BGPPeer %s in namespace %s with this password: %s",
		builder.Definition.Name, builder.Definition.Namespace, password)

	if password == "" {
		glog.V(100).Infof("Can not redefine BGPPeer with empty password")

		builder.errorMsg = "password can not be empty string"
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.Password = password

	return builder
}

// WithEBGPMultiHop defines the EBGPMultiHop bool flag placed in the BGPPeer spec.
func (builder *BGPPeerBuilder) WithEBGPMultiHop(eBGPMultiHop bool) *BGPPeerBuilder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof(
		"Creating BGPPeer %s in namespace %s with this eBGPMultiHop flag: %t",
		builder.Definition.Name, builder.Definition.Namespace, eBGPMultiHop)

	builder.Definition.Spec.EBGPMultiHop = eBGPMultiHop

	return builder
}

// WithOptions creates BGPPeer with generic mutation options.
func (builder *BGPPeerBuilder) WithOptions(options ...BGPPeerAdditionalOptions) *BGPPeerBuilder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof("Setting BGPPeer additional options")

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

// GetBGPPeerGVR returns bgppeer's GroupVersionResource which could be used for Clean function.
func GetBGPPeerGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group: APIGroup, Version: APIVersion, Resource: "bgppeers",
	}
}

// validate will check that the builder and builder definition are properly initialized before
// accessing any member fields.
func (builder *BGPPeerBuilder) validate() (bool, error) {
	resourceCRD := "BGPPeer"

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

func (builder *BGPPeerBuilder) convertToStructured(unsObject *unstructured.Unstructured) (*mlbtypes.BGPPeer, error) {
	bgpPeer := &mlbtypes.BGPPeer{}

	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unsObject.Object, bgpPeer)
	if err != nil {
		glog.V(100).Infof(
			"Failed to convert from unstructured to BGPPeer object in namespace %s",
			builder.Definition.Name, builder.Definition.Namespace)

		return nil, err
	}

	return bgpPeer, err
}
