package ibiconfig

import (
	"fmt"

	"github.com/openshift-kni/lifecycle-agent/api/seedreconfig"
)

// ImageBasedInstallConfigVersion is the version supported by this package.
const ImageBasedInstallConfigVersion = "v1beta1"

const (
	defaultExtraPartitionLabel = "varlibcontainers"
)

// ImageBasedInstallConfig is the API for specifying configuration for the image-based installation iso.
type ImageBasedInstallConfig struct {
	// Params that are used to configure the installation iso and will be moved to installer
	RHCOSLiveISO string `json:"rhcosLiveIso,omitempty"`

	// RawNMStateConfig contains nmstate configuration YAML file provided as string.
	// String will be written to file and will be applied with "nmstatectl apply" command.
	// Example of nmstate configurations can be found in this link https://nmstate.io/examples.html
	// +optional
	NMStateConfig string `json:"nmStateConfig,omitempty"`

	// PullSecret is the secret to use when pulling images. Equivalent to install-config.yaml's pullSecret.
	PullSecret string `json:"pullSecret,omitempty"`

	// Proxy is the proxy settings for the installation iso. Equivalent to
	// install-config.yaml's proxy. This will be set to the installation service
	// +optional
	Proxy *seedreconfig.Proxy `json:"proxy,omitempty"`

	// ImageDigestSources lists sources/repositories for the release-image content.
	// +optional
	ImageDigestSources []ImageDigestSource `json:"imageDigestSources,omitempty"`

	// PEM-encoded X.509 certificate bundle. Will be copied to /etc/pki/ca-trust/source/anchors/
	// in the installation iso
	// +optional
	AdditionalTrustBundle string `json:"additionalTrustBundle,omitempty"`

	// SSHKey is the public Secure Shell (SSH) key to provide access to
	// instances. Equivalent to install-config.yaml's sshKey. T
	// This key will be added to the host to allow ssh access while running with installation iso
	SSHKey string `json:"sshKey,omitempty"`

	// IBIPrepareConfig parameter that should be provided to lca cli installation service
	IBIPrepareConfig `json:",inline"`
}

// IBIPrepareConfig is the API for specifying configuration for lca cli.
type IBIPrepareConfig struct {

	// configuration for lca cli
	SeedImage            string `json:"seedImage"`
	SeedVersion          string `json:"seedVersion"`
	InstallationDisk     string `json:"installationDisk"`
	PrecacheBestEffort   bool   `json:"precacheBestEffort,omitempty"`
	PrecacheDisabled     bool   `json:"precacheDisabled,omitempty"`
	Shutdown             bool   `json:"shutdown,omitempty"`
	UseContainersFolder  bool   `json:"useContainersFolder,omitempty"`
	ExtraPartitionStart  string `json:"extraPartitionStart,omitempty"`
	ExtraPartitionLabel  string `json:"extraPartitionLabel,omitempty"`
	ExtraPartitionNumber uint   `json:"extraPartitionNumber,omitempty"`
	SkipDiskCleanup      bool   `json:"skipDiskCleanup,omitempty"`
}

type ImageDigestSource struct {
	// Source is the repository that users refer to, e.g. in image pull specifications.
	Source string `json:"source"`

	// Mirrors is one or more repositories that may also contain the same images.
	Mirrors []string `json:"mirrors,omitempty"`
}

func (c *IBIPrepareConfig) Validate() error {
	if c.SeedImage == "" {
		return fmt.Errorf("seedImage is required")
	}
	if c.SeedVersion == "" {
		return fmt.Errorf("seedVersion is required")
	}
	if c.InstallationDisk == "" {
		return fmt.Errorf("installationDisk is required")
	}

	return nil
}

func (c *IBIPrepareConfig) SetDefaultValues() {
	if c.ExtraPartitionStart == "" {
		c.ExtraPartitionStart = "-40G"
	}
	if c.ExtraPartitionNumber == 0 {
		c.ExtraPartitionNumber = 5
	}
	if c.ExtraPartitionLabel == "" {
		c.ExtraPartitionLabel = defaultExtraPartitionLabel
	}
}
