package set

import (
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/internal/hwaccelparams/deploy"
	"gopkg.in/yaml.v2"
)

// Config worker-config.
type Config struct {
	Sources Sources `yaml:"sources"`
}

// CPUConfig cpu feature config.
type CPUConfig struct {
	CPUID struct {
		AttributeBlacklist []string `yaml:"attributeBlacklist,omitempty"`
		AttributeWhitelist []string `yaml:"attributeWhitelist,omitempty"`
	} `yaml:"cpuid,omitempty"`
}

// PCIDevice pci config.
type PCIDevice struct {
	DeviceClassWhitelist []string `yaml:"deviceClassWhitelist,omitempty"`
	DeviceLabelFields    []string `yaml:"deviceLabelFields,omitempty"`
}

// Sources contains all sources.
type Sources struct {
	CPU    *CPUConfig    `yaml:"cpu,omitempty"`
	PCI    []PCIDevice   `yaml:"pci,omitempty"`
	USB    []interface{} `yaml:"usb,omitempty"`    // Add the necessary struct for USB if needed
	Custom []interface{} `yaml:"custom,omitempty"` // Add the necessary struct for Custom if needed
}

// CPUConfigLabels set cpu blacklist/whitelist.
func CPUConfigLabels(apiClient *clients.Settings,
	blackListLabels,
	whiteListLabels []string,
	enableTopology bool,
	namespace,
	image string) {
	var cfg Config

	if cfg.Sources.CPU == nil {
		cfg.Sources.CPU = &CPUConfig{}
	}

	cfg.Sources.CPU.CPUID.AttributeBlacklist = append(cfg.Sources.CPU.CPUID.AttributeBlacklist, blackListLabels...)
	cfg.Sources.CPU.CPUID.AttributeWhitelist = append(cfg.Sources.CPU.CPUID.AttributeWhitelist, whiteListLabels...)

	modifiedCPUYAML, err := yaml.Marshal(cfg)
	if err != nil {
		panic(err)
	}

	err = deploy.DeployNfdWithCustomConfig(namespace, enableTopology, string(modifiedCPUYAML), image)
	if err != nil {
		panic(err)
	}
}
