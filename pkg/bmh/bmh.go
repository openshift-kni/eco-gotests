package bmh

import (
	"context"

	goclient "sigs.k8s.io/controller-runtime/pkg/client"

	"fmt"

	bmhv1alpha1 "github.com/metal3-io/baremetal-operator/apis/metal3.io/v1alpha1"
	"github.com/openshift-kni/eco-gotests/pkg/clients"
	"golang.org/x/exp/slices"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Builder provides struct for bmh object which contains connection to
// the cluster and bmh definition.
type Builder struct {
	Definition *bmhv1alpha1.BareMetalHost
	Object     *bmhv1alpha1.BareMetalHost
	apiClient  *clients.Settings
	errorMsg   string
}

// NewBuilder creates new instance of Builder.
// When namespace not provided default will be used: 'openshift-machine-api'.
// When bootMode not provided default will be used: 'UEFI'.
func NewBuilder(
	apiClient *clients.Settings,
	name string,
	nsname string,
	bmcAddress string,
	bmcSecretName string,
	bootMacAddress string,
	bootMode string,
	deviceName string) *Builder {
	builder := Builder{
		apiClient: apiClient,
		Definition: &bmhv1alpha1.BareMetalHost{
			Spec: bmhv1alpha1.BareMetalHostSpec{

				BMC: bmhv1alpha1.BMCDetails{
					Address:                        bmcAddress,
					CredentialsName:                bmcSecretName,
					DisableCertificateVerification: true,
				},
				BootMode:              bmhv1alpha1.BootMode(bootMode),
				BootMACAddress:        bootMacAddress,
				Online:                true,
				ExternallyProvisioned: false,
				RootDeviceHints: &bmhv1alpha1.RootDeviceHints{
					DeviceName: deviceName,
				},
			},
			ObjectMeta: metaV1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
		},
	}

	if name == "" {
		builder.errorMsg = "BMH 'name' cannot be empty"
	}

	if nsname == "" {
		builder.errorMsg = "BMH 'nsname' cannot be empty"
	}

	if bmcAddress == "" {
		builder.errorMsg = "BMH 'bmcAddress' cannot be empty"
	}

	if bmcSecretName == "" {
		builder.errorMsg = "BMH 'bmcSecretName' cannot be empty"
	}

	bootModeAcceptable := []string{"UEFI", "UEFISecureBoot", "legacy"}
	if !slices.Contains(bootModeAcceptable, bootMode) {
		builder.errorMsg = "Not acceptable 'bootMode' value"
	}

	if bootMacAddress == "" {
		builder.errorMsg = "BMH 'bootMacAddress' cannot be empty"
	}

	if deviceName == "" {
		builder.errorMsg = "BMH 'bootMacAddress' cannot be empty"
	}

	return &builder
}

// Create generates bmh on cluster and stores created object in struct.
func (builder *Builder) Create() (*Builder, error) {
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

// Delete removes bmh from a cluster.
func (builder *Builder) Delete() (*Builder, error) {
	if !builder.Exists() {
		return builder, fmt.Errorf("bmh cannot be deleted because it does not exist")
	}

	err := builder.apiClient.Delete(context.TODO(), builder.Definition)

	if err != nil {
		return builder, fmt.Errorf("can not delete bmh: %w", err)
	}

	builder.Object = nil

	return builder, nil
}

// Exists tells whether the given bmh exists.
func (builder *Builder) Exists() bool {
	var err error
	builder.Object, err = builder.Get()

	return err == nil || !k8serrors.IsNotFound(err)
}

// Get returns bmh object if found.
func (builder *Builder) Get() (*bmhv1alpha1.BareMetalHost, error) {
	bmh := &bmhv1alpha1.BareMetalHost{}
	err := builder.apiClient.Get(context.TODO(), goclient.ObjectKey{
		Name:      builder.Definition.Name,
		Namespace: builder.Definition.Namespace,
	}, bmh)

	if err != nil {
		return nil, err
	}

	return bmh, err
}
