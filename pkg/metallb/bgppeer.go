package metallb

import (
	"context"
	"fmt"
	"net"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-gotests/pkg/clients"
	"github.com/openshift-kni/eco-gotests/pkg/msg"
	metalLbV1Beta1 "go.universe.tf/metallb/api/v1beta1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	goclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// BGPPeerBuilder provides struct for the BGPPeer object containing connection to
// the cluster and the BGPPeer definitions.
type BGPPeerBuilder struct {
	Definition *metalLbV1Beta1.BGPPeer
	Object     *metalLbV1Beta1.BGPPeer
	apiClient  *clients.Settings
	errorMsg   string
}

// NewBPGPeerBuilder creates a new instance of BGPPeer.
func NewBPGPeerBuilder(
	apiClient *clients.Settings, name, nsname, peerIP string, asn, remoteASN uint32) *BGPPeerBuilder {
	glog.V(100).Infof(
		"Initializing new BGPPeer structure with the following params: %s, %s %s %d %d",
		name, nsname, peerIP, asn, remoteASN)

	builder := BGPPeerBuilder{
		apiClient: apiClient,
		Definition: &metalLbV1Beta1.BGPPeer{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			}, Spec: metalLbV1Beta1.BGPPeerSpec{
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
func (builder *BGPPeerBuilder) Get() (*metalLbV1Beta1.BGPPeer, error) {
	glog.V(100).Infof(
		"Collecting BGPPeer object %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	bgpPeer := &metalLbV1Beta1.BGPPeer{}
	err := builder.apiClient.Get(context.TODO(), goclient.ObjectKey{
		Name:      builder.Definition.Name,
		Namespace: builder.Definition.Namespace,
	}, bgpPeer)

	if err != nil {
		glog.V(100).Infof(
			"BGPPeer object %s doesn't exist in namespace %s",
			builder.Definition.Name, builder.Definition.Namespace)

		return nil, err
	}

	return bgpPeer, err
}

// Exists checks whether the given BGPPeer exists.
func (builder *BGPPeerBuilder) Exists() bool {
	glog.V(100).Infof(
		"Checking if BGPPeer %s exists in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	builder.Object, err = builder.Get()

	return err == nil || !k8serrors.IsNotFound(err)
}

// Create makes a BGPPeer in the cluster and stores the created object in struct.
func (builder *BGPPeerBuilder) Create() (*BGPPeerBuilder, error) {
	glog.V(100).Infof("Creating the BGPPeer %s in namespace %s",
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

// Delete removes BGPPeer object from a cluster.
func (builder *BGPPeerBuilder) Delete() (*BGPPeerBuilder, error) {
	glog.V(100).Infof("Deleting the BGPPeer object %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace,
	)

	if !builder.Exists() {
		return builder, fmt.Errorf("BGPPeer cannot be deleted because it does not exist")
	}

	err := builder.apiClient.Delete(context.TODO(), builder.Definition)

	if err != nil {
		return builder, fmt.Errorf("can not delete BGPPeer: %w", err)
	}

	builder.Object = nil

	return builder, nil
}

// Update renovates the existing BGPPeer object with the BGPPeer definition in builder.
func (builder *BGPPeerBuilder) Update(force bool) (*BGPPeerBuilder, error) {
	glog.V(100).Infof("Updating the BGPPeer object %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace,
	)

	if builder.errorMsg != "" {
		return nil, fmt.Errorf(builder.errorMsg)
	}

	err := builder.apiClient.Update(context.TODO(), builder.Definition)

	if err != nil {
		if force {
			glog.V(100).Infof(
				"Failed to update the BGPPeer object %s in namespace %s. "+
					"Note: Force flag set, executed delete/create methods instead",
				builder.Definition.Name, builder.Definition.Namespace,
			)

			builder, err := builder.Delete()

			if err != nil {
				glog.V(100).Infof(
					"Failed to update the BGPPeer object %s in namespace %s, "+
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

// WithRouterID defines the routerID placed in the BGPPeer spec.
func (builder *BGPPeerBuilder) WithRouterID(routerID string) *BGPPeerBuilder {
	glog.V(100).Infof(
		"Creating BGPPeer %s in namespace %s with this routerID: %s",
		builder.Definition.Name, builder.Definition.Namespace, routerID)

	if builder.Definition == nil {
		builder.errorMsg = msg.UndefinedCrdObjectErrString("BGPPeer")
	}

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
	glog.V(100).Infof(
		"Creating BGPPeer %s in namespace %s with this bfdProfile: %s",
		builder.Definition.Name, builder.Definition.Namespace, bfdProfile)

	if builder.Definition == nil {
		builder.errorMsg = msg.UndefinedCrdObjectErrString("BGPPeer")
	}

	if bfdProfile == "" {
		glog.V(100).Infof("The bfdProfile of the BGPPeer can not be empty string")

		builder.errorMsg = "The bfdProfile is empty sting"
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.BFDProfile = bfdProfile

	return builder
}

// WithSRCAddress defines the SRCAddress placed in the BGPPeer spec.
func (builder *BGPPeerBuilder) WithSRCAddress(srcAddress string) *BGPPeerBuilder {
	glog.V(100).Infof(
		"Creating BGPPeer %s in namespace %s with this srcAddress: %s",
		builder.Definition.Name, builder.Definition.Namespace, srcAddress)

	if builder.Definition == nil {
		builder.errorMsg = msg.UndefinedCrdObjectErrString("BGPPeer")
	}

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
	glog.V(100).Infof(
		"Creating BGPPeer %s in namespace %s with this port: %d",
		builder.Definition.Name, builder.Definition.Namespace, port)

	if builder.Definition == nil {
		builder.errorMsg = msg.UndefinedCrdObjectErrString("BGPPeer")
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.Port = port

	return builder
}

// WithHoldTime defines the holdTime placed in the BGPPeer spec.
func (builder *BGPPeerBuilder) WithHoldTime(holdTime metaV1.Duration) *BGPPeerBuilder {
	glog.V(100).Infof(
		"Creating BGPPeer %s in namespace %s with this holdTime: %s",
		builder.Definition.Name, builder.Definition.Namespace, holdTime)

	if builder.Definition == nil {
		builder.errorMsg = msg.UndefinedCrdObjectErrString("BGPPeer")
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.HoldTime = holdTime

	return builder
}

// WithKeepalive defines the keepAliveTime placed in the BGPPeer spec.
func (builder *BGPPeerBuilder) WithKeepalive(keepalive metaV1.Duration) *BGPPeerBuilder {
	glog.V(100).Infof(
		"Creating BGPPeer %s in namespace %s with this keepalive: %s",
		builder.Definition.Name, builder.Definition.Namespace, keepalive)

	if builder.Definition == nil {
		builder.errorMsg = msg.UndefinedCrdObjectErrString("BGPPeer")
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.KeepaliveTime = keepalive

	return builder
}

// WithNodeSelector defines the nodeSelector placed in the BGPPeer spec.
func (builder *BGPPeerBuilder) WithNodeSelector(nodeSelector map[string]string) *BGPPeerBuilder {
	glog.V(100).Infof(
		"Creating BGPPeer %s in namespace %s with this nodeSelector: %s",
		builder.Definition.Name, builder.Definition.Namespace, nodeSelector)

	if builder.Definition == nil {
		builder.errorMsg = msg.UndefinedCrdObjectErrString("BGPPeer")
	}

	if len(nodeSelector) == 0 {
		glog.V(100).Infof("Can not redefine BGPPeer with empty nodeSelector map")

		builder.errorMsg = "BGPPeer 'nodeSelector' cannot be empty map"
	}

	if builder.errorMsg != "" {
		return builder
	}

	ndSelector := []metalLbV1Beta1.NodeSelector{{MatchLabels: nodeSelector}}
	builder.Definition.Spec.NodeSelectors = ndSelector

	return builder
}

// WithPassword defines the password placed in the BGPPeer spec.
func (builder *BGPPeerBuilder) WithPassword(password string) *BGPPeerBuilder {
	glog.V(100).Infof(
		"Creating BGPPeer %s in namespace %s with this password: %s",
		builder.Definition.Name, builder.Definition.Namespace, password)

	if builder.Definition == nil {
		builder.errorMsg = msg.UndefinedCrdObjectErrString("BGPPeer")
	}

	if password == "" {
		glog.V(100).Infof("Can not redefine BGPPeer with empty password")

		builder.errorMsg = "password can not be empty sting"
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.Password = password

	return builder
}

// WithEBGPMultiHop defines the EBGPMultiHop bool flag placed in the BGPPeer spec.
func (builder *BGPPeerBuilder) WithEBGPMultiHop(eBGPMultiHop bool) *BGPPeerBuilder {
	glog.V(100).Infof(
		"Creating BGPPeer %s in namespace %s with this eBGPMultiHop flag: %t",
		builder.Definition.Name, builder.Definition.Namespace, eBGPMultiHop)

	if builder.Definition == nil {
		builder.errorMsg = msg.UndefinedCrdObjectErrString("BGPPeer")
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.EBGPMultiHop = eBGPMultiHop

	return builder
}
