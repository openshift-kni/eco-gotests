package clusterversion

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/msg"
	configv1 "github.com/openshift/api/config/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	clusterVersionName = "version"
	isTrue             = "True"
)

// Builder provides a struct for clusterversion object from the cluster and a clusterversion definition.
type Builder struct {
	// clusterversion definition, used to create the clusterversion object.
	Definition *configv1.ClusterVersion
	// Created clusterversion object.
	Object *configv1.ClusterVersion
	// api client to interact with the cluster.
	apiClient *clients.Settings
	// Used in functions that define or mutate clusterversion definition. errorMsg is processed before the
	// clusterversion object is created.
	errorMsg string
}

// Pull loads an existing clusterversion into Builder struct.
func Pull(apiClient *clients.Settings) (*Builder, error) {
	glog.V(100).Infof("Pulling existing clusterversion name: %s", clusterVersionName)

	builder := Builder{
		apiClient: apiClient,
		Definition: &configv1.ClusterVersion{
			ObjectMeta: metav1.ObjectMeta{
				Name: clusterVersionName,
			},
		},
	}

	if !builder.Exists() {
		return nil, fmt.Errorf("clusterversion object %s does not exist", clusterVersionName)
	}

	builder.Definition = builder.Object

	return &builder, nil
}

// Exists checks whether the given clusterversion exists.
func (builder *Builder) Exists() bool {
	if valid, _ := builder.validate(); !valid {
		return false
	}

	glog.V(100).Infof(
		"Checking if clusterversion %s exists",
		builder.Definition.Name)

	var err error
	builder.Object, err = builder.apiClient.ConfigV1Interface.ClusterVersions().Get(
		context.TODO(), builder.Definition.Name, metav1.GetOptions{})

	return err == nil || !k8serrors.IsNotFound(err)
}

// WithDesiredUpdateImage adds the desired image to the clusterversion struct.
func (builder *Builder) WithDesiredUpdateImage(desiredUpdateImage string, force bool) *Builder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Info("Adding the desired image %s to clusterversion %s",
		desiredUpdateImage, builder.Definition.Name)

	if desiredUpdateImage == "" {
		glog.V(100).Infof("The desiredUpdateImage is empty")

		builder.errorMsg = "'desiredUpdateImage' cannot be empty"

		return builder
	}

	builder.Definition.Spec.DesiredUpdate = &configv1.Update{Image: desiredUpdateImage, Force: force}

	return builder
}

// WithDesiredUpdateChannel adds the desired channel to the clusterversion struct.
func (builder *Builder) WithDesiredUpdateChannel(updateChannel string) *Builder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Info("Adding the desired updateChannel %s to clusterversion %s",
		updateChannel, builder.Definition.Name)

	if updateChannel == "" {
		glog.V(100).Infof("The updateChannel is empty")

		builder.errorMsg = "'updateChannel' cannot be empty"

		return builder
	}

	builder.Definition.Spec.Channel = updateChannel

	return builder
}

// Update renovates the existing clusterversion object with the clusterversion definition in builder.
func (builder *Builder) Update() (*Builder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Updating clusterversion %s", builder.Definition.Name)

	if !builder.Exists() {
		return nil, fmt.Errorf("clusterversion object does not exist")
	}

	builder.Definition.CreationTimestamp = metav1.Time{}

	var err error
	builder.Object, err = builder.apiClient.ConfigV1Interface.ClusterVersions().Update(
		context.TODO(), builder.Definition, metav1.UpdateOptions{})

	return builder, err
}

// WaitUntilProgressing waits for timeout duration or until clusterversion is in Progressing state.
func (builder *Builder) WaitUntilProgressing(timeout time.Duration) error {
	return builder.WaitUntilConditionTrue("Progressing", timeout)
}

// WaitUntilAvailable waits for timeout duration or until clusterversion is in Available state.
func (builder *Builder) WaitUntilAvailable(timeout time.Duration) error {
	return builder.WaitUntilConditionTrue("Available", timeout)
}

// WaitUntilConditionTrue waits for timeout duration or until clusterversion gets to a specific status.
func (builder *Builder) WaitUntilConditionTrue(
	conditionType configv1.ClusterStatusConditionType, timeout time.Duration) error {
	if valid, err := builder.validate(); !valid {
		return err
	}

	if !builder.Exists() {
		return fmt.Errorf("%s clusterversion not found", builder.Definition.Name)
	}

	return wait.PollUntilContextTimeout(
		context.TODO(), time.Second, timeout, true, func(ctx context.Context) (bool, error) {
			var err error
			builder.Object, err = builder.apiClient.ConfigV1Interface.ClusterVersions().Get(context.TODO(),
				builder.Definition.Name, metav1.GetOptions{})

			if err != nil {
				glog.V(100).Infof("Failed to get the clusterversion with error %s", err)

				return false, nil
			}

			for _, condition := range builder.Object.Status.Conditions {
				if condition.Type == conditionType {
					return condition.Status == isTrue, nil
				}
			}

			return false, nil
		})
}

// WaitUntilUpdateIsStarted waits until there is a history entry indicating the update start.
func (builder *Builder) WaitUntilUpdateIsStarted(timeout time.Duration) error {
	return builder.WaitUntilUpdateHistoryStateTrue("Partial", timeout)
}

// WaitUntilUpdateIsCompleted waits until there is a history entry indicating the update completed.
func (builder *Builder) WaitUntilUpdateIsCompleted(timeout time.Duration) error {
	return builder.WaitUntilUpdateHistoryStateTrue("Completed", timeout)
}

// WaitUntilUpdateHistoryStateTrue waits until there is a history entry indicating an updateHistoryState.
func (builder *Builder) WaitUntilUpdateHistoryStateTrue(
	updateHistoryState configv1.UpdateState, timeout time.Duration) error {
	if valid, err := builder.validate(); !valid {
		return err
	}

	if !builder.Exists() {
		return fmt.Errorf("%s clusterversion not found", builder.Definition.Name)
	}

	return wait.PollUntilContextTimeout(
		context.TODO(), time.Second, timeout, true, func(ctx context.Context) (bool, error) {
			var err error
			builder.Object, err = builder.apiClient.ConfigV1Interface.ClusterVersions().Get(context.TODO(),
				builder.Definition.Name, metav1.GetOptions{})

			if err != nil {
				glog.V(100).Infof("Failed to get the clusterversion with error %s", err)

				return false, nil
			}

			updateImage := builder.Object.Status.Desired.Image

			for _, updateHistory := range builder.Object.Status.History {
				if updateHistory.Image == updateImage && updateHistory.State == updateHistoryState {
					return true, nil
				}
			}

			return false, nil
		})
}

// GetNextUpdateVersionImage fetches the next recommended or conditional update for the cluster.
func (builder *Builder) GetNextUpdateVersionImage(stream string, acceptConditionalVersions bool) (string, error) {
	if valid, err := builder.validate(); !valid {
		return "", err
	}

	glog.V(100).Infof("Getting the update version image in stream %s for clusterversion %s",
		stream, builder.Definition.Name)

	if !builder.Exists() {
		return "", fmt.Errorf("%s clusterversion not found", builder.Definition.Name)
	}

	if stream == "" {
		return "", fmt.Errorf("stream can not be empty")
	}

	currentVersion := builder.Object.Status.Desired.Version

	for _, availableUpdate := range builder.Object.Status.AvailableUpdates {
		isStreamUpdate, err := builder.isStreamUpdate(currentVersion, availableUpdate.Version, stream)
		if isStreamUpdate && err == nil {
			return availableUpdate.Image, nil
		}
	}

	if acceptConditionalVersions {
		for _, conditionalUpdate := range builder.Object.Status.ConditionalUpdates {
			isStreamUpdate, err := builder.isStreamUpdate(currentVersion, conditionalUpdate.Release.Version, stream)
			if isStreamUpdate && err == nil {
				return conditionalUpdate.Release.Image, nil
			}
		}
	}

	return "", fmt.Errorf("update version in %s stream not found", stream)
}

// isStreamUpdate checks if updateVersion is a 'stream' (X, Y or Z) update for version.
func (builder *Builder) isStreamUpdate(version, updateVersion, stream string) (isStreamUpdate bool, err error) {
	glog.V(100).Infof("Verify if updateVersion %s is a stream %s update for version %s", updateVersion, stream, version)

	if !slices.Contains([]string{X, Z, Y}, stream) {
		glog.V(100).Infof("invalid stream %s", stream)

		return false, fmt.Errorf("invalid stream %s", stream)
	}

	semVersion, semVersionError := semver.NewVersion(version)

	if semVersionError != nil {
		return false, fmt.Errorf("the version %s is invalid", version)
	}

	semUpdateVersion, updateVersionError := semver.NewVersion(updateVersion)
	glog.V(100).Infof("Testing %s and %s", semVersion.String(), semUpdateVersion.String())

	if updateVersionError != nil {
		return false, fmt.Errorf("the Update Version %s is invalid", updateVersion)
	}

	switch major := semVersion.Major(); {
	case major == semUpdateVersion.Major() && semVersion.Minor() == semUpdateVersion.Minor() &&
		semVersion.Patch() < semUpdateVersion.Patch() && stream == Z:
		glog.V(100).Infof("This version is a z update: %s", semUpdateVersion.String())

		return true, nil

	case major == semUpdateVersion.Major() &&
		semVersion.Minor() < semUpdateVersion.Minor() && stream == Y:
		glog.V(100).Infof("This version is a y update: %s", semUpdateVersion.String())

		return true, nil

	case semVersion.Major() < semUpdateVersion.Major() && stream == X:
		glog.V(100).Infof("This version is an x update: %s", semUpdateVersion.String())

		return true, nil

	default:
		glog.V(100).Infof("The version %s is not an update for %s ",
			semUpdateVersion.String(), semVersion.String())

		return false, nil
	}
}

// validate will check that the builder and builder definition are properly initialized before
// accessing any member fields.
func (builder *Builder) validate() (bool, error) {
	resourceCRD := "ClusterVersion"

	if builder == nil {
		glog.V(100).Infof("The %s builder is uninitialized", resourceCRD)

		return false, fmt.Errorf("error: received nil %s builder", resourceCRD)
	}

	if builder.Definition == nil {
		glog.V(100).Infof("The %s is undefined", resourceCRD)

		return false, fmt.Errorf(msg.UndefinedCrdObjectErrString(resourceCRD))
	}

	if builder.apiClient == nil {
		glog.V(100).Infof("The %s builder apiclient is nil", resourceCRD)

		return false, fmt.Errorf("%s builder cannot have nil apiClient", resourceCRD)
	}

	if builder.errorMsg != "" {
		glog.V(100).Infof("The %s builder has error message: %s", resourceCRD, builder.errorMsg)

		return false, fmt.Errorf(builder.errorMsg)
	}

	return true, nil
}
