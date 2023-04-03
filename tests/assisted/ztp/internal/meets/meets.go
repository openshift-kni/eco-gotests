package meets

import (
	"fmt"

	"github.com/hashicorp/go-version"
	. "github.com/openshift-kni/eco-gotests/tests/assisted/ztp/internal/ztpinittools"
)

// HubOperatorVersionRequirement checks that hub operator version meets the version provided.
func HubOperatorVersionRequirement(requiredVersion string) (bool, string) {
	if ZTPConfig.HubConfig.OperatorVersion == "" {
		return false, "Hub operator version was not provided through environment"
	}

	hubOperatorVersion, _ := version.NewVersion(ZTPConfig.HubConfig.OperatorVersion)
	currentVersion, _ := version.NewVersion(requiredVersion)

	if hubOperatorVersion.LessThan(currentVersion) {
		return false, fmt.Sprintf("Provided hub operator version does not meet requirement: %s",
			ZTPConfig.HubConfig.OperatorVersion)
	}

	return true, ""
}

// HubDisconnectedRequirement checks that the hub is disconnected.
func HubDisconnectedRequirement() (bool, string) {
	if !ZTPConfig.HubConfig.Disconnected {
		return false, "Provided hub cluster is connected"
	}

	return true, ""
}

// HubConnectedRequirement checks that the hub is connected.
func HubConnectedRequirement() (bool, string) {
	if ZTPConfig.HubConfig.Disconnected {
		return false, "Provided hub cluster is disconnected"
	}

	return true, ""
}

// SpokeOCPVersionRequirement checks that spoke ocp version meets the version provided.
func SpokeOCPVersionRequirement(requiredVersion string) (bool, string) {
	if ZTPConfig.SpokeConfig.OCPVersion == "" {
		return false, "Spoke openshift version was not provided through environment"
	}

	spokeOCPVersion, _ := version.NewVersion(ZTPConfig.SpokeConfig.OCPVersion)
	currentVersion, _ := version.NewVersion(requiredVersion)

	if spokeOCPVersion.LessThan(currentVersion) {
		return false, fmt.Sprintf("Provided spoke openshift version does not meet requirement: %v",
			ZTPConfig.SpokeConfig.OCPVersion)
	}

	return true, ""
}

// SpokePullSecretSetRequirement check that the spoke pull-secret is not empty.
func SpokePullSecretSetRequirement() (bool, string) {
	if ZTPConfig.SpokeConfig.PullSecret == "" {
		return false, "Spoke pull-secret was not provided through environment"
	}

	return true, ""
}
