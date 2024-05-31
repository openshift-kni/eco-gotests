package mirroring

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/platform"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/remote"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/shell"
)

// MirrorImageToTheLocalRegistry downloads image to the local mirror registry.
func MirrorImageToTheLocalRegistry(
	apiClient *clients.Settings,
	originServerURL,
	imageName,
	imageTag,
	host,
	user,
	pass,
	combinedPullSecretFile,
	localRegistryRepository string) (string, string, error) {
	if originServerURL == "" {
		glog.V(100).Infof("The originServerURL is empty")

		return "", "", fmt.Errorf("the originServerURL could not be empty")
	}

	if imageName == "" {
		glog.V(100).Infof("The imageName is empty")

		return "", "", fmt.Errorf("the imageName could not be empty")
	}

	if imageTag == "" {
		glog.V(100).Infof("The imageTag is empty")

		return "", "", fmt.Errorf("the imageTag could not be empty")
	}

	originalImageURL := fmt.Sprintf("%s/%s", originServerURL, imageName)
	isDisconnected, err := platform.IsDisconnectedDeployment(apiClient)

	if err != nil {
		return "", "", err
	}

	if !isDisconnected {
		glog.Info("Deployment type is not disconnected, images mirroring not required")

		return originalImageURL, imageTag, nil
	}

	localRegistryURL, err := platform.GetLocalMirrorRegistryURL(apiClient)

	if err != nil {
		return "", "", err
	}

	localImageURL := fmt.Sprintf("%s/%s/%s", localRegistryURL, localRegistryRepository, imageName)

	localRegistryPullSecretFilePath, err :=
		CopyRegistryAuthLocally(host, user, pass, combinedPullSecretFile)

	if err != nil {
		return "", "", err
	}

	glog.Infof("Mirror image %s to the local registry %s", originalImageURL, localRegistryURL)

	imageMirrorCmd := fmt.Sprintf("oc image mirror --insecure=true --registry-config=%s %s:%s=%s:%s",
		localRegistryPullSecretFilePath, originalImageURL, imageTag, localImageURL, imageTag)
	_, err = shell.ExecuteCmd(imageMirrorCmd)

	if err != nil {
		glog.Infof("failed to execute %s command due to %s", imageMirrorCmd, err)

		return "", "", err
	}

	return localImageURL, imageTag, nil
}

// CopyRegistryAuthLocally copy mirror registry authentication files locally.
func CopyRegistryAuthLocally(host, user, pass, combinedPullSecretFile string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	out, err := remote.ExecCmdOnHost(host, user, pass, "pwd")
	if err != nil {
		return "", err
	}

	remoteRegistryPullSecretFilePath := filepath.Join(out, combinedPullSecretFile)
	localRegistryPullSecretFilePath := filepath.Join("/tmp", combinedPullSecretFile)
	remoteDockerDirectoryPath := filepath.Join(out, ".docker")
	localDockerDirectoryPath := filepath.Join(homeDir, ".docker")

	err = remote.ScpFileFrom(
		remoteRegistryPullSecretFilePath,
		localRegistryPullSecretFilePath,
		host,
		user,
		pass)
	if err != nil {
		return "", err
	}

	err = remote.ScpDirectoryFrom(
		remoteDockerDirectoryPath,
		localDockerDirectoryPath,
		host,
		user,
		pass)
	if err != nil {
		return localRegistryPullSecretFilePath, err
	}

	return localRegistryPullSecretFilePath, nil
}
