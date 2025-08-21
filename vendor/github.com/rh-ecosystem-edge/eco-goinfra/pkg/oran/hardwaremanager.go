package oran

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"
	pluginv1alpha1 "github.com/openshift-kni/oran-hwmgr-plugin/api/hwmgr-plugin/v1alpha1"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/clients"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/msg"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// HardwareManagerBuilder provides a struct to interface with HardwareManager resources on a specific cluster.
type HardwareManagerBuilder struct {
	// Definition of the HardwareManager used to create the resource.
	Definition *pluginv1alpha1.HardwareManager
	// Object of the HardwareManager as it is on the cluster.
	Object *pluginv1alpha1.HardwareManager
	// apiClient used to interact with the cluster.
	apiClient runtimeclient.Client
	// errorMsg used to store latest error message from functions that do not return errors.
	errorMsg string
}

// NewHwmgrBuilder creates a new instance of a HardwareManager builder. If adaptorID is dell-hwmgr, WithDellData must be
// called before creating the builder.
func NewHwmgrBuilder(apiClient *clients.Settings,
	name, nsname string,
	adaptorID pluginv1alpha1.HardwareManagerAdaptorID) *HardwareManagerBuilder {
	glog.V(100).Infof(
		"Initializing new HardwareManager structure with the following params: "+
			"name: %s, nsname: %s, adaptorID: %s",
		name, nsname, adaptorID)

	if apiClient == nil {
		glog.V(100).Infof("The apiClient of the HardwareManager is nil")

		return nil
	}

	err := apiClient.AttachScheme(pluginv1alpha1.AddToScheme)
	if err != nil {
		glog.V(100).Infof("Failed to add plugin v1alpha1 scheme to client schemes: %v", err)

		return nil
	}

	builder := &HardwareManagerBuilder{
		apiClient: apiClient.Client,
		Definition: &pluginv1alpha1.HardwareManager{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
			Spec: pluginv1alpha1.HardwareManagerSpec{
				AdaptorID: adaptorID,
			},
		},
	}

	if name == "" {
		glog.V(100).Info("The name of the HardwareManager is empty")

		builder.errorMsg = "hardwareManager 'name' cannot be empty"

		return builder
	}

	if nsname == "" {
		glog.V(100).Info("The nsname of the HardwareManager is empty")

		builder.errorMsg = "hardwareManager 'nsname' cannot be empty"

		return builder
	}

	if adaptorID != pluginv1alpha1.SupportedAdaptors.Dell && adaptorID != pluginv1alpha1.SupportedAdaptors.Loopback {
		glog.V(100).Infof("The adaptorID of the HardwareManager is invalid. Must be loopback or dell-hwmgr, not %s",
			adaptorID)

		builder.errorMsg = "hardwareManager 'adaptorID' must be loopback or dell-hwmgr"

		return builder
	}

	return builder
}

// WithLoopbackData sets the LoopbackData of a HardwareManager with AdaptorID loopback.
func (builder *HardwareManagerBuilder) WithLoopbackData(data pluginv1alpha1.LoopbackData) *HardwareManagerBuilder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof("Setting LoopbackData for HardwareManager %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	if builder.Definition.Spec.AdaptorID != pluginv1alpha1.SupportedAdaptors.Loopback {
		glog.V(100).Infof("Cannot set HardwareManager LoopbackData if AdaptorID is %s, must be loopback",
			builder.Definition.Spec.AdaptorID)

		builder.errorMsg = "cannot set LoopbackData unless AdaptorID is loopback"

		return builder
	}

	builder.Definition.Spec.LoopbackData = &data

	return builder
}

// WithDellData sets the DellData of a HardwareManager with AdaptorID dell-hwmgr.
func (builder *HardwareManagerBuilder) WithDellData(data pluginv1alpha1.DellData) *HardwareManagerBuilder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof("Setting DellData for HardwareManager %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	if builder.Definition.Spec.AdaptorID != pluginv1alpha1.SupportedAdaptors.Dell {
		glog.V(100).Infof("Cannot set HardwareManager DellData if AdaptorID is %s, must be dell-hwmgr",
			builder.Definition.Spec.AdaptorID)

		builder.errorMsg = "cannot set DellData unless AdaptorID is dell-hwmgr"

		return builder
	}

	if data.AuthSecret == "" {
		glog.V(100).Info("The HardwareManager DellData AuthSecret cannot be empty")

		builder.errorMsg = "hardwareManager 'AuthSecret' cannot be empty"

		return builder
	}

	if data.ApiUrl == "" {
		glog.V(100).Info("The HardwareManager DellData ApiUrl cannot be empty")

		builder.errorMsg = "hardwareManager 'ApiUrl' cannot be empty"

		return builder
	}

	builder.Definition.Spec.DellData = &data

	return builder
}

// PullHwmgr pulls an existing HardwareManager into a HardwareManagerBuilder struct.
func PullHwmgr(apiClient *clients.Settings, name, nsname string) (*HardwareManagerBuilder, error) {
	glog.V(100).Infof("Pulling existing HardwareManager %s in namespace %s from cluster", name, nsname)

	if apiClient == nil {
		glog.V(100).Infof("The apiClient of the HardwareManager is nil")

		return nil, fmt.Errorf("hardwareManager 'apiClient' cannot be nil")
	}

	err := apiClient.AttachScheme(pluginv1alpha1.AddToScheme)
	if err != nil {
		glog.V(100).Infof("Failed to add plugin v1alpha1 scheme to client schemes: %v", err)

		return nil, err
	}

	builder := &HardwareManagerBuilder{
		apiClient: apiClient.Client,
		Definition: &pluginv1alpha1.HardwareManager{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
		},
	}

	if name == "" {
		glog.V(100).Info("The name of the HardwareManager is empty")

		return nil, fmt.Errorf("hardwareManager 'name' cannot be empty")
	}

	if nsname == "" {
		glog.V(100).Info("The nsname of the HardwareManager is empty")

		return nil, fmt.Errorf("hardwareManager 'nsname' cannot be empty")
	}

	if !builder.Exists() {
		glog.V(100).Info("The Hardwaremanager %s does not exist in namespace %s", name, nsname)

		return nil, fmt.Errorf("hardwareManager object %s does not exist in namespace %s", name, nsname)
	}

	builder.Definition = builder.Object

	return builder, nil
}

// Get returns the HardwareManager object if found.
func (builder *HardwareManagerBuilder) Get() (*pluginv1alpha1.HardwareManager, error) {
	if valid, err := builder.validate(); !valid {
		return nil, err
	}

	glog.V(100).Infof("Getting HardwareManager object %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	hardwareManager := &pluginv1alpha1.HardwareManager{}
	err := builder.apiClient.Get(context.TODO(), runtimeclient.ObjectKey{
		Name:      builder.Definition.Name,
		Namespace: builder.Definition.Namespace,
	}, hardwareManager)

	if err != nil {
		glog.V(100).Infof("Failed to get HardwareManager object %s in namespace %s: %v",
			builder.Definition.Name, builder.Definition.Namespace, err)

		return nil, err
	}

	return hardwareManager, nil
}

// Exists checks whether this HardwareManager exists on the cluster.
func (builder *HardwareManagerBuilder) Exists() bool {
	if valid, _ := builder.validate(); !valid {
		return false
	}

	glog.V(100).Infof("Checking if HardwareManager %s exists in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	builder.Object, err = builder.Get()

	return err == nil || !k8serrors.IsNotFound(err)
}

// Create makes a HardwareManager on the cluster if it does not already exist.
func (builder *HardwareManagerBuilder) Create() (*HardwareManagerBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return nil, err
	}

	glog.V(100).Infof("Creating HardwareManager %s in namespace %s", builder.Definition.Name, builder.Definition.Namespace)

	if builder.Exists() {
		return builder, nil
	}

	builder.Definition.ResourceVersion = ""
	err := builder.apiClient.Create(context.TODO(), builder.Definition)

	if err != nil {
		return nil, err
	}

	builder.Object = builder.Definition

	return builder, nil
}

// Update changes the existing Hardwaremanager resource on the cluster, falling back to deleting and recreating if the
// update fails when force is set.
func (builder *HardwareManagerBuilder) Update(force bool) (*HardwareManagerBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return nil, err
	}

	glog.V(100).Infof("Updating HardwareManager %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	if !builder.Exists() {
		glog.V(100).Infof("HardwareManager %s does not exist in namespace %s",
			builder.Definition.Name, builder.Definition.Namespace)

		return nil, fmt.Errorf("cannot update non-existent hardwareManager")
	}

	builder.Definition.ResourceVersion = builder.Object.ResourceVersion
	err := builder.apiClient.Update(context.TODO(), builder.Definition)

	if err != nil {
		if force {
			glog.V(100).Info(msg.FailToUpdateNotification(
				"hardwareManager", builder.Definition.Name, builder.Definition.Namespace))

			err := builder.Delete()
			builder.Definition.ResourceVersion = ""

			if err != nil {
				glog.V(100).Info(msg.FailToUpdateError("hardwareManager", builder.Definition.Name, builder.Definition.Namespace))

				return nil, err
			}

			return builder.Create()
		}

		return nil, err
	}

	builder.Object = builder.Definition

	return builder, nil
}

// Delete removes a HardwareManager from the cluster if it exists.
func (builder *HardwareManagerBuilder) Delete() error {
	if valid, err := builder.validate(); !valid {
		return err
	}

	glog.V(100).Infof("Deleting HardwareManager %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	if !builder.Exists() {
		glog.V(100).Infof("HardwareManager %s in namespace %s does not exist",
			builder.Definition.Name, builder.Definition.Namespace)

		builder.Object = nil

		return nil
	}

	err := builder.apiClient.Delete(context.TODO(), builder.Object)
	if err != nil {
		return err
	}

	builder.Object = nil

	return nil
}

// WaitForCondition waits up to the provided timeout for a condition matching expected. It checks only the Type, Status,
// Reason, and Message fields. For the message, it matches if the message contains the expected. Zero fields in the
// expected condition are ignored.
func (builder *HardwareManagerBuilder) WaitForCondition(
	expected metav1.Condition, timeout time.Duration) (*HardwareManagerBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return nil, err
	}

	glog.V(100).Infof("Waiting up to %s until HardwareManager %s in namespace %s has condition %v",
		timeout, builder.Definition.Name, builder.Definition.Namespace, expected)

	if !builder.Exists() {
		glog.V(100).Infof("HardwareManager %s in namespace %s does not exist",
			builder.Definition.Name, builder.Definition.Namespace)

		return nil, fmt.Errorf("cannot wait for non-existent HardwareManager")
	}

	err := wait.PollUntilContextTimeout(
		context.TODO(), 3*time.Second, timeout, true, func(ctx context.Context) (bool, error) {
			var err error
			builder.Object, err = builder.Get()

			if err != nil {
				glog.V(100).Infof("Failed to get HardwareManager %s in namespace %s: %v",
					builder.Definition.Name, builder.Definition.Namespace, err)

				return false, nil
			}

			builder.Definition = builder.Object

			for _, condition := range builder.Object.Status.Conditions {
				if expected.Type != "" && condition.Type != expected.Type {
					continue
				}

				if expected.Status != "" && condition.Status != expected.Status {
					continue
				}

				if expected.Reason != "" && condition.Reason != expected.Reason {
					continue
				}

				if expected.Message != "" && !strings.Contains(condition.Message, expected.Message) {
					continue
				}

				return true, nil
			}

			return false, nil
		})

	if err != nil {
		return nil, err
	}

	return builder, nil
}

// validate checks that the builder, definition, and apiClient are properly initialized and there is no errorMsg.
func (builder *HardwareManagerBuilder) validate() (bool, error) {
	resourceCRD := "hardwareManager"

	if builder == nil {
		glog.V(100).Infof("The %s builder is uninitialized", resourceCRD)

		return false, fmt.Errorf("error: received nil %s builder", resourceCRD)
	}

	if builder.Definition == nil {
		glog.V(100).Infof("The %s is uninitialized", resourceCRD)

		return false, fmt.Errorf("%s", msg.UndefinedCrdObjectErrString(resourceCRD))
	}

	if builder.apiClient == nil {
		glog.V(100).Infof("The %s builder apiClient is nil", resourceCRD)

		return false, fmt.Errorf("%s builder cannot have nil apiClient", resourceCRD)
	}

	if builder.errorMsg != "" {
		glog.V(100).Infof("The %s builder has error message %s", resourceCRD, builder.errorMsg)

		return false, fmt.Errorf("%s", builder.errorMsg)
	}

	return true, nil
}
