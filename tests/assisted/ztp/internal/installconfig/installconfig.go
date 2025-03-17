package installconfig

import (
	installerTypes "github.com/openshift/installer/pkg/types"
	"gopkg.in/yaml.v3"
)

// NewInstallConfigFromString returns an unmarshalled install-config from provided string.
func NewInstallConfigFromString(config string) (installerTypes.InstallConfig, error) {
	var installConfigData installerTypes.InstallConfig

	err := yaml.Unmarshal([]byte(config), &installConfigData)
	if err != nil {
		return installerTypes.InstallConfig{}, err
	}

	return installConfigData, nil
}
