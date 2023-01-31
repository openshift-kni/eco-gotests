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
	goclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// BFDBuilder provides struct for the BFDProfile object containing connection to
// the cluster and the BFDProfile definitions.
type BFDBuilder struct {
	Definition *metalLbV1Beta1.BFDProfile
	Object     *metalLbV1Beta1.BFDProfile
	apiClient  *clients.Settings
	errorMsg   string
}

// NewBFDBuilder creates a new instance of BFDBuilder.
func NewBFDBuilder(apiClient *clients.Settings, name, nsname string) *BFDBuilder {
	glog.V(100).Infof(
		"Initializing new BFDBuilder structure with the following params: %s, %s",
		name, nsname)

	builder := BFDBuilder{
		apiClient: apiClient,
		Definition: &metalLbV1Beta1.BFDProfile{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the BFDProfile is empty")

		builder.errorMsg = "BFDProfile 'name' cannot be empty"
	}

	if nsname == "" {
		glog.V(100).Infof("The namespace of the BFDProfile is empty")

		builder.errorMsg = "BFDProfile 'nsname' cannot be empty"
	}

	return &builder
}

// Get returns BFDProfile object if found.
func (builder *BFDBuilder) Get() (*metalLbV1Beta1.BFDProfile, error) {
	glog.V(100).Infof(
		"Collecting BFDProfile object %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	bfdProfile := &metalLbV1Beta1.BFDProfile{}
	err := builder.apiClient.Get(context.TODO(), goclient.ObjectKey{
		Name:      builder.Definition.Name,
		Namespace: builder.Definition.Namespace,
	}, bfdProfile)

	if err != nil {
		glog.V(100).Infof(
			"BFDProfile object %s doesn't exist in namespace %s",
			builder.Definition.Name, builder.Definition.Namespace)

		return nil, err
	}

	return bfdProfile, err
}

// Exists checks whether the given BFDProfile exists.
func (builder *BFDBuilder) Exists() bool {
	glog.V(100).Infof(
		"Checking if BFDProfile %s exists in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	builder.Object, err = builder.Get()

	return err == nil || !k8serrors.IsNotFound(err)
}

// Create makes a BFDProfile in the cluster and stores the created object in struct.
func (builder *BFDBuilder) Create() (*BFDBuilder, error) {
	glog.V(100).Infof("Creating the BFDProfile %s in namespace %s",
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

// Delete removes BFDProfile object from a cluster.
func (builder *BFDBuilder) Delete() (*BFDBuilder, error) {
	glog.V(100).Infof("Deleting the BFDProfile object %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace,
	)

	if !builder.Exists() {
		return builder, fmt.Errorf("BFDProfile cannot be deleted because it does not exist")
	}

	err := builder.apiClient.Delete(context.TODO(), builder.Definition)

	if err != nil {
		return builder, fmt.Errorf("can not delete BFDProfile: %w", err)
	}

	builder.Object = nil

	return builder, nil
}

// Update renovates the existing BFDProfile object with the BFDProfile definition in builder.
func (builder *BFDBuilder) Update(force bool) (*BFDBuilder, error) {
	glog.V(100).Infof("Updating the BFDProfile object %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace,
	)

	if builder.errorMsg != "" {
		return nil, fmt.Errorf(builder.errorMsg)
	}

	err := builder.apiClient.Update(context.TODO(), builder.Definition)

	if err != nil {
		if force {
			glog.V(100).Infof(
				"Failed to update the BFDProfile object %s in namespace %s. "+
					"Note: Force flag set, executed delete/create methods instead",
				builder.Definition.Name, builder.Definition.Namespace,
			)

			builder, err := builder.Delete()

			if err != nil {
				glog.V(100).Infof(
					"Failed to update the BFDProfile object %s in namespace %s, "+
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

// WithRcvInterval defines the receiveInterval placed in the BFDProfile.
func (builder *BFDBuilder) WithRcvInterval(rcvInterval uint32) *BFDBuilder {
	return builder.withInterval("receiveInterval", rcvInterval)
}

// WithTransmitInterval defines the transmitInterval placed in the BFDProfile.
func (builder *BFDBuilder) WithTransmitInterval(rcvInterval uint32) *BFDBuilder {
	return builder.withInterval("transmitInterval", rcvInterval)
}

// WithEchoInterval defines the ecoInterval placed in the BFDProfile.
func (builder *BFDBuilder) WithEchoInterval(ecoInterval uint32) *BFDBuilder {
	return builder.withInterval("ecoInterval", ecoInterval)
}

// WithMultiplier defines the detectMultiplier placed in the BFDProfile.
func (builder *BFDBuilder) WithMultiplier(multiplier uint32) *BFDBuilder {
	glog.V(100).Infof(
		"Creating BFDProfile %s in namespace %s with this detectMultiplier: %d",
		builder.Definition.Name, builder.Definition.Namespace, multiplier)

	if builder.Definition == nil {
		builder.errorMsg = msg.UndefinedCrdObjectErrString("BFDProfile")
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.DetectMultiplier = &multiplier

	return builder
}

// WithEchoMode defines the echoMode placed in the BFDProfile.
func (builder *BFDBuilder) WithEchoMode(echoMode bool) *BFDBuilder {
	return builder.withBoolFlagFor("echoMode", echoMode)
}

// WithPassiveMode defines the passiveMode placed in the BFDProfile.
func (builder *BFDBuilder) WithPassiveMode(passiveMode bool) *BFDBuilder {
	return builder.withBoolFlagFor("passiveMode", passiveMode)
}

// WithMinimumTTL defines the minimumTTTL placed in the BFDProfile.
func (builder *BFDBuilder) WithMinimumTTL(minimumTTL uint32) *BFDBuilder {
	glog.V(100).Infof(
		"Creating BFDProfile %s in namespace %s with this minimumTTL: %d",
		builder.Definition.Name, builder.Definition.Namespace, minimumTTL)

	if builder.Definition == nil {
		builder.errorMsg = msg.UndefinedCrdObjectErrString("BFDProfile")
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.MinimumTTL = &minimumTTL

	return builder
}

func (builder *BFDBuilder) withBoolFlagFor(flagName string, flagValue bool) *BFDBuilder {
	glog.V(100).Infof(
		"Creating BFDProfile %s in namespace %s with flag %s: %t",
		builder.Definition.Name, builder.Definition.Namespace, flagName, flagValue)

	if builder.Definition == nil {
		builder.errorMsg = msg.UndefinedCrdObjectErrString("BFDProfile")
	}

	if builder.errorMsg != "" {
		return builder
	}

	switch flagName {
	case "echoMode":
		builder.Definition.Spec.EchoMode = &flagValue
	case "passiveMode":
		builder.Definition.Spec.PassiveMode = &flagValue
	default:
		builder.errorMsg = "invalid bool flag name parameter"
	}

	return builder
}

func (builder *BFDBuilder) withInterval(intervalName string, interval uint32) *BFDBuilder {
	glog.V(100).Infof(
		"Creating BFDProfile %s in namespace %s with interval %s: %d",
		builder.Definition.Name, builder.Definition.Namespace, intervalName, interval)

	if builder.Definition == nil {
		builder.errorMsg = msg.UndefinedCrdObjectErrString("BFDProfile")
	}

	if builder.errorMsg != "" {
		return builder
	}

	switch intervalName {
	case "transmitInterval":
		builder.Definition.Spec.TransmitInterval = &interval
	case "receiveInterval":
		builder.Definition.Spec.ReceiveInterval = &interval
	case "ecoInterval":
		builder.Definition.Spec.EchoInterval = &interval
	default:
		builder.errorMsg = "invalid interval parameters"
	}

	return builder
}
