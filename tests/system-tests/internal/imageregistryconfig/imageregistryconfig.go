package imageregistryconfig

import (
	"fmt"
	"time"

	v1 "github.com/openshift/api/operator/v1"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/apiservers"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/clusteroperator"
	"github.com/openshift-kni/eco-goinfra/pkg/imageregistry"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/await"
	imageregistryv1 "github.com/openshift/api/imageregistry/v1"
)

var imageRegistryObjName = "cluster"
var imageRegistryNamespace = "openshift-image-registry"
var imageRegistryCoName = "image-registry"
var imageRegistryDeploymentName = "image-registry"

// SetManagementState returns true when succeeded to change imageRegistry operator management state.
func SetManagementState(apiClient *clients.Settings, expectedManagementState v1.ManagementState) error {
	irClusterOperator, err := imageregistry.Pull(apiClient, imageRegistryCoName)

	if err != nil {
		glog.V(100).Infof("Failed to get imageRegistry operator due to %v",
			err.Error())

		return err
	}

	glog.V(100).Infof("Set imageRegistry %s ManagementState to the %v",
		irClusterOperator.Definition.Name, expectedManagementState)

	currentManagementState, err := irClusterOperator.GetManagementState()

	if err != nil {
		glog.V(100).Infof("Failed to get current imageRegistry operator management state value due to %v",
			err.Error())

		return err
	}

	if currentManagementState != &expectedManagementState {
		glog.V(100).Infof("The current imageRegistry %s ManagementState is %v; it needs to be changed to the %v",
			irClusterOperator.Definition.Name, currentManagementState, expectedManagementState)

		irClusterOperator, err := irClusterOperator.WithManagementState(expectedManagementState).Update()

		if err != nil {
			glog.V(100).Infof("Failed to make change to the imageRegistry operator managementState due to %v", err)

			return err
		}

		newManagementState, err := irClusterOperator.GetManagementState()

		if err != nil {
			glog.V(100).Infof("Failed to get current imageRegistry operator managementState value due to %v", err)

			return err
		}

		if newManagementState != &expectedManagementState {
			return fmt.Errorf("failed to change imageRegistry operator managementState value;"+
				"expected %v, current value is %v", expectedManagementState, newManagementState)
		}

		return nil
	}

	return err
}

// SetStorageToTheEmptyDir sets the imageRegistry storage to an empty directory.
func SetStorageToTheEmptyDir(apiClient *clients.Settings) error {
	irClusterOperator, err := clusteroperator.Pull(apiClient, imageRegistryCoName)

	if err != nil {
		return err
	}

	if !irClusterOperator.IsDegraded() {
		err = await.WaitUntilDeploymentReady(
			apiClient,
			imageRegistryDeploymentName,
			imageRegistryNamespace,
			time.Second*2)

		if err == nil {
			return nil
		}
	}

	glog.V(100).Infof("Setting up imageRegistry storage to the EmptyDir")

	imageRegistry, err := imageregistry.Pull(apiClient, imageRegistryObjName)

	if err != nil {
		return err
	}

	emptyStorage := imageregistryv1.ImageRegistryConfigStorage{EmptyDir: nil}

	imageRegistry.Object.Spec.Storage = emptyStorage

	_, err = imageRegistry.Update()

	if err != nil {
		glog.V(100).Infof("Failed to change an imageRegistry config and setup storage to the EmptyDir")

		return err
	}

	glog.V(100).Info("Wait for the openshiftapiserver APIServerDeploymentProgressing ending, " +
		"pods have to be updated to the latest generation")

	oasBuilder, err := apiservers.PullOpenshiftAPIServer(apiClient)

	if err != nil {
		return err
	}

	err = oasBuilder.WaitAllPodsAtTheLatestGeneration(time.Minute * 10)

	if err != nil {
		return err
	}

	glog.V(100).Info("Wait for the kubeapiserver NodeInstallerProgressing ending, " +
		"nodes have to be updated to the latest revision")

	kasBuilder, err := apiservers.PullKubeAPIServer(apiClient)

	if err != nil {
		return err
	}

	err = kasBuilder.WaitAllNodesAtTheLatestRevision(time.Minute * 15)

	if err != nil {
		return err
	}

	glog.V(100).Info("Wait for the image-registry deployment succeeded")

	err = await.WaitUntilDeploymentReady(
		apiClient,
		imageRegistryDeploymentName,
		imageRegistryNamespace,
		time.Minute*5)

	if err != nil {
		glog.V(100).Infof("image-registry deployment failure due to %s", err.Error())

		return err
	}

	err = WaitForImageregistryCoIsAvailable(apiClient)

	if err != nil {
		return err
	}

	return nil
}

// WaitForImageregistryCoIsAvailable verifies imageregistryconfig co is Available.
func WaitForImageregistryCoIsAvailable(apiClient *clients.Settings) error {
	glog.V(100).Infof("Asserting clusteroperators availability")

	imageRegistryCo, err := clusteroperator.Pull(apiClient, imageRegistryCoName)

	if err != nil {
		return err
	}

	if imageRegistryCo.IsAvailable() {
		return nil
	}

	err = imageRegistryCo.WaitUntilAvailable(time.Minute * 2)

	if err != nil {
		return err
	}

	return nil
}
