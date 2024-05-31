package bmh

import (
	"context"
	"time"

	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/util/wait"

	goclient "sigs.k8s.io/controller-runtime/pkg/client"

	"fmt"

	bmhv1alpha1 "github.com/metal3-io/baremetal-operator/apis/metal3.io/v1alpha1"
	"github.com/openshift-kni/eco-gotests/pkg/clients"
	"github.com/openshift-kni/eco-gotests/pkg/msg"
	"golang.org/x/exp/slices"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Builder provides struct for the bmh object containing connection to
// the cluster and the bmh definitions.
type Builder struct {
	Definition *bmhv1alpha1.BareMetalHost
	Object     *bmhv1alpha1.BareMetalHost
	apiClient  *clients.Settings
	errorMsg   string
}

// NewBuilder creates a new instance of Builder.
func NewBuilder(
	apiClient *clients.Settings,
	name string,
	nsname string,
	bmcAddress string,
	bmcSecretName string,
	bootMacAddress string,
	bootMode string) *Builder {
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

	return &builder
}

// WithRootDeviceDeviceName sets rootDeviceHints DeviceName to specified value.
func (builder *Builder) WithRootDeviceDeviceName(deviceName string) *Builder {
	if builder.Definition == nil {
		glog.V(100).Infof("The baremetalhost is undefined")

		builder.errorMsg = msg.UndefinedCrdObjectErrString("BareMetalHost")
	}

	if deviceName == "" {
		glog.V(100).Infof("The baremetalhost rootDeviceHint deviceName is empty")

		builder.errorMsg = "the baremetalhost rootDeviceHint deviceName cannot be empty"
	}

	if builder.errorMsg != "" {
		return builder
	}

	if builder.Definition.Spec.RootDeviceHints == nil {
		builder.Definition.Spec.RootDeviceHints = &bmhv1alpha1.RootDeviceHints{}
	}

	builder.Definition.Spec.RootDeviceHints.DeviceName = deviceName

	return builder
}

// WithRootDeviceHTCL sets rootDeviceHints HTCL to specified value.
func (builder *Builder) WithRootDeviceHTCL(hctl string) *Builder {
	if builder.Definition == nil {
		glog.V(100).Infof("The baremetalhost is undefined")

		builder.errorMsg = msg.UndefinedCrdObjectErrString("BareMetalHost")
	}

	if hctl == "" {
		glog.V(100).Infof("The baremetalhost rootDeviceHint hctl is empty")

		builder.errorMsg = "the baremetalhost rootDeviceHint hctl cannot be empty"
	}

	if builder.errorMsg != "" {
		return builder
	}

	if builder.Definition.Spec.RootDeviceHints == nil {
		builder.Definition.Spec.RootDeviceHints = &bmhv1alpha1.RootDeviceHints{}
	}

	builder.Definition.Spec.RootDeviceHints.HCTL = hctl

	return builder
}

// WithRootDeviceModel sets rootDeviceHints Model to specified value.
func (builder *Builder) WithRootDeviceModel(model string) *Builder {
	if builder.Definition == nil {
		glog.V(100).Infof("The baremetalhost is undefined")

		builder.errorMsg = msg.UndefinedCrdObjectErrString("BareMetalHost")
	}

	if model == "" {
		glog.V(100).Infof("The baremetalhost rootDeviceHint model is empty")

		builder.errorMsg = "the baremetalhost rootDeviceHint model cannot be empty"
	}

	if builder.errorMsg != "" {
		return builder
	}

	if builder.Definition.Spec.RootDeviceHints == nil {
		builder.Definition.Spec.RootDeviceHints = &bmhv1alpha1.RootDeviceHints{}
	}

	builder.Definition.Spec.RootDeviceHints.Model = model

	return builder
}

// WithRootDeviceVendor sets rootDeviceHints Vendor to specified value.
func (builder *Builder) WithRootDeviceVendor(vendor string) *Builder {
	if builder.Definition == nil {
		glog.V(100).Infof("The baremetalhost is undefined")

		builder.errorMsg = msg.UndefinedCrdObjectErrString("BareMetalHost")
	}

	if vendor == "" {
		glog.V(100).Infof("The baremetalhost rootDeviceHint vendor is empty")

		builder.errorMsg = "the baremetalhost rootDeviceHint vendor cannot be empty"
	}

	if builder.errorMsg != "" {
		return builder
	}

	if builder.Definition.Spec.RootDeviceHints == nil {
		builder.Definition.Spec.RootDeviceHints = &bmhv1alpha1.RootDeviceHints{}
	}

	builder.Definition.Spec.RootDeviceHints.Model = vendor

	return builder
}

// WithRootDeviceSerialNumber sets rootDeviceHints serialNumber to specified value.
func (builder *Builder) WithRootDeviceSerialNumber(serialNumber string) *Builder {
	if builder.Definition == nil {
		glog.V(100).Infof("The baremetalhost is undefined")

		builder.errorMsg = msg.UndefinedCrdObjectErrString("BareMetalHost")
	}

	if serialNumber == "" {
		glog.V(100).Infof("The baremetalhost rootDeviceHint serialNumber is empty")

		builder.errorMsg = "the baremetalhost rootDeviceHint serialNumber cannot be empty"
	}

	if builder.errorMsg != "" {
		return builder
	}

	if builder.Definition.Spec.RootDeviceHints == nil {
		builder.Definition.Spec.RootDeviceHints = &bmhv1alpha1.RootDeviceHints{}
	}

	builder.Definition.Spec.RootDeviceHints.SerialNumber = serialNumber

	return builder
}

// WithRootDeviceMinSizeGigabytes sets rootDeviceHints MinSizeGigabytes to specified value.
func (builder *Builder) WithRootDeviceMinSizeGigabytes(size int) *Builder {
	if builder.Definition == nil {
		glog.V(100).Infof("The baremetalhost is undefined")

		builder.errorMsg = msg.UndefinedCrdObjectErrString("BareMetalHost")
	}

	if size < 0 {
		glog.V(100).Infof("The baremetalhost rootDeviceHint size is less than 0")

		builder.errorMsg = "the baremetalhost rootDeviceHint size cannot be less than 0"
	}

	if builder.errorMsg != "" {
		return builder
	}

	if builder.Definition.Spec.RootDeviceHints == nil {
		builder.Definition.Spec.RootDeviceHints = &bmhv1alpha1.RootDeviceHints{}
	}

	builder.Definition.Spec.RootDeviceHints.MinSizeGigabytes = size

	return builder
}

// WithRootDeviceWWN sets rootDeviceHints WWN to specified value.
func (builder *Builder) WithRootDeviceWWN(wwn string) *Builder {
	if builder.Definition == nil {
		glog.V(100).Infof("The baremetalhost is undefined")

		builder.errorMsg = msg.UndefinedCrdObjectErrString("BareMetalHost")
	}

	if wwn == "" {
		glog.V(100).Infof("The baremetalhost rootDeviceHint wwn is empty")

		builder.errorMsg = "the baremetalhost rootDeviceHint wwn cannot be empty"
	}

	if builder.errorMsg != "" {
		return builder
	}

	if builder.Definition.Spec.RootDeviceHints == nil {
		builder.Definition.Spec.RootDeviceHints = &bmhv1alpha1.RootDeviceHints{}
	}

	builder.Definition.Spec.RootDeviceHints.WWN = wwn

	return builder
}

// WithRootDeviceWWNWithExtension sets rootDeviceHints WWNWithExtension to specified value.
func (builder *Builder) WithRootDeviceWWNWithExtension(wwnWithExtension string) *Builder {
	if builder.Definition == nil {
		glog.V(100).Infof("The baremetalhost is undefined")

		builder.errorMsg = msg.UndefinedCrdObjectErrString("BareMetalHost")
	}

	if wwnWithExtension == "" {
		glog.V(100).Infof("The baremetalhost rootDeviceHint wwnWithExtension is empty")

		builder.errorMsg = "the baremetalhost rootDeviceHint wwnWithExtension cannot be empty"
	}

	if builder.errorMsg != "" {
		return builder
	}

	if builder.Definition.Spec.RootDeviceHints == nil {
		builder.Definition.Spec.RootDeviceHints = &bmhv1alpha1.RootDeviceHints{}
	}

	builder.Definition.Spec.RootDeviceHints.WWNWithExtension = wwnWithExtension

	return builder
}

// WithRootDeviceWWNVendorExtension sets rootDeviceHint WWNVendorExtension to specified value.
func (builder *Builder) WithRootDeviceWWNVendorExtension(wwnVendorExtension string) *Builder {
	if builder.Definition == nil {
		glog.V(100).Infof("The baremetalhost is undefined")

		builder.errorMsg = msg.UndefinedCrdObjectErrString("BareMetalHost")
	}

	if wwnVendorExtension == "" {
		glog.V(100).Infof("The baremetalhost rootDeviceHint wwnVendorExtension is empty")

		builder.errorMsg = "the baremetalhost rootDeviceHint wwnVendorExtension cannot be empty"
	}

	if builder.errorMsg != "" {
		return builder
	}

	if builder.Definition.Spec.RootDeviceHints == nil {
		builder.Definition.Spec.RootDeviceHints = &bmhv1alpha1.RootDeviceHints{}
	}

	builder.Definition.Spec.RootDeviceHints.WWNVendorExtension = wwnVendorExtension

	return builder
}

// WithRootDeviceRotationalDisk sets rootDeviceHint Rotational to specified value.
func (builder *Builder) WithRootDeviceRotationalDisk(rotational bool) *Builder {
	if builder.Definition == nil {
		glog.V(100).Infof("The baremetalhost is undefined")

		builder.errorMsg = msg.UndefinedCrdObjectErrString("BareMetalHost")
	}

	if builder.errorMsg != "" {
		return builder
	}

	if builder.Definition.Spec.RootDeviceHints == nil {
		builder.Definition.Spec.RootDeviceHints = &bmhv1alpha1.RootDeviceHints{}
	}

	builder.Definition.Spec.RootDeviceHints.Rotational = &rotational

	return builder
}

// Create makes a bmh in the cluster and stores the created object in struct.
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

// Exists checks whether the given bmh exists.
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

// CreateAndWaitUntilProvisioned creates bmh object and waits until bmh is provisioned.
func (builder *Builder) CreateAndWaitUntilProvisioned(timeout time.Duration) (*Builder, error) {
	builder, err := builder.Create()
	if err != nil {
		return nil, err
	}

	err = builder.WaitUntilProvisioned(timeout)

	return builder, err
}

// WaitUntilProvisioned waits for timeout duration or until bmh is provisioned.
func (builder *Builder) WaitUntilProvisioned(timeout time.Duration) error {
	return builder.WaitUntilInStatus(bmhv1alpha1.StateProvisioned, timeout)
}

// WaitUntilProvisioning waits for timeout duration or until bmh is provisioning.
func (builder *Builder) WaitUntilProvisioning(timeout time.Duration) error {
	return builder.WaitUntilInStatus(bmhv1alpha1.StateProvisioning, timeout)
}

// WaitUntilReady waits for timeout duration or until bmh is ready.
func (builder *Builder) WaitUntilReady(timeout time.Duration) error {
	return builder.WaitUntilInStatus(bmhv1alpha1.StateReady, timeout)
}

// WaitUntilAvailable waits for timeout duration or until bmh is available.
func (builder *Builder) WaitUntilAvailable(timeout time.Duration) error {
	return builder.WaitUntilInStatus(bmhv1alpha1.StateAvailable, timeout)
}

// WaitUntilInStatus waits for timeout duration or until bmh gets to a specific status.
func (builder *Builder) WaitUntilInStatus(status bmhv1alpha1.ProvisioningState, timeout time.Duration) error {
	if builder.errorMsg != "" {
		return fmt.Errorf(builder.errorMsg)
	}

	return wait.PollImmediate(time.Second, timeout, func() (bool, error) {
		var err error
		builder.Object, err = builder.Get()
		if err != nil {
			return false, nil
		}

		if builder.Object.Status.Provisioning.State == status {
			return true, nil
		}

		return false, err
	})
}

// DeleteAndWaitUntilDeleted delete bmh object and waits until deleted.
func (builder *Builder) DeleteAndWaitUntilDeleted(timeout time.Duration) (*Builder, error) {
	builder, err := builder.Delete()
	if err != nil {
		return builder, err
	}

	err = builder.WaitUntilDeleted(timeout)

	return nil, err
}

// WaitUntilDeleted waits for timeout duration or until bmh is deleted.
func (builder *Builder) WaitUntilDeleted(timeout time.Duration) error {
	err := wait.Poll(time.Second, timeout, func() (bool, error) {
		_, err := builder.Get()
		if err == nil {
			glog.V(100).Infof("bmh %s/%s still present",
				builder.Definition.Namespace,
				builder.Definition.Name)

			return false, nil
		}
		if k8serrors.IsNotFound(err) {
			glog.V(100).Infof("bmh %s/%s is gone",
				builder.Definition.Namespace,
				builder.Definition.Name)

			return true, nil
		}
		glog.V(100).Infof("failed to get bmh %s/%s: %v",
			builder.Definition.Namespace,
			builder.Definition.Name, err)

		return false, err
	})

	return err
}
