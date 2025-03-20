package ocloudconfig

import (
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/openshift-kni/eco-goinfra/pkg/bmc"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/systemtestsconfig"
	"gopkg.in/yaml.v2"
)

const (
	// PathToDefaultOCloudParamsFile path to config file with default o-cloud parameters.
	PathToDefaultOCloudParamsFile = "./default.yaml"
)

// OCloudConfig type keeps o-cloud configuration.
type OCloudConfig struct {
	*systemtestsconfig.SystemTestsConfig

	GenerateSeedImage bool   `yaml:"ibi_generate_seed_image" envconfig:"ECO_IBI_GENERATE_SEED_IMAGE"`
	IbiBaseImagePath  string `yaml:"ibi_base_image_path" envconfig:"ECO_IBI_BASE_IMAGE_PATH"`
	IbiBaseImageURL   string `yaml:"ibi_base_image_url" envconfig:"ECO_IBI_BASE_IMAGE_URL"`
	VirtualMediaID    string `yaml:"virtual_media_id" envconfig:"ECO_VIRTUAL_MEDIA_ID"`

	LocalRegistryAuth string `yaml:"local_registry_auth" envconfig:"ECO_LOCAL_REGISTRY_AUTH"`
	SeedImage         string `yaml:"seed_image" envconfig:"ECO_SEED_IMAGE"`
	SeedVersion       string `yaml:"seed_version" envconfig:"ECO_SEED_VERSION"`
	Registry          string `yaml:"registry" envconfig:"ECO_REGISTRY"`
	SSHKey            string `yaml:"ssh_key" envconfig:"ECO_SSH_KEY"`
	PullSecret        string `yaml:"pull_secret" envconfig:"ECO_PULL_SECRET"`
	BaseImageName     string `yaml:"base_image_name" envconfig:"ECO_BASE_IMAGE_NAME"`
	InterfaceName     string `yaml:"interface_name" envconfig:"ECO_INTERFACE_NAME"`
	InterfaceIpv6     string `yaml:"interface_ipv6" envconfig:"ECO_INTERFACE_IPV6"`
	DNSIpv6           string `yaml:"dns_ipv6" envconfig:"ECO_DNS_IPV6"`
	NextHopIpv6       string `yaml:"next_hop_ipv6" envconfig:"ECO_NEXT_HOP_IPV6"`
	NextHopInterface  string `yaml:"next_hop_interface" envconfig:"ECO_NEXT_HOP_INTERFACE"`

	// BMCClient provides access to the BMC. Nil when BMC configs are not provided.
	Spoke1BMC         *bmc.BMC
	Spoke1BMCUsername string        `yaml:"oran_spoke1_bmc_username" envconfig:"ECO_ORAN_SPOKE1_BMC_USERNAME"`
	Spoke1BMCPassword string        `yaml:"oran_spoke1_bmc_password" envconfig:"ECO_ORAN_SPOKE1_BMC_PASSWORD"`
	Spoke1BMCHost     string        `yaml:"oran_spoke1_bmc_host" envconfig:"ECO_ORAN_SPOKE1_BMC_HOST"`
	Spoke1BMCTimeout  time.Duration `yaml:"oran_spoke1_bmc_timeout" envconfig:"ECO_ORAN_SPOKE1_BMC_TIMEOUT"`

	// BMCClient provides access to the BMC. Nil when BMC configs are not provided.
	Spoke2BMC         *bmc.BMC
	Spoke2BMCUsername string        `yaml:"oran_spoke2_bmc_username" envconfig:"ECO_ORAN_SPOKE2_BMC_USERNAME"`
	Spoke2BMCPassword string        `yaml:"oran_spoke2_bmc_password" envconfig:"ECO_ORAN_SPOKE2_BMC_PASSWORD"`
	Spoke2BMCHost     string        `yaml:"oran_spoke2_bmc_host" envconfig:"ECO_ORAN_SPOKE2_BMC_HOST"`
	Spoke2BMCTimeout  time.Duration `yaml:"oran_spoke2_bmc_timeout" envconfig:"ECO_ORAN_SPOKE2_BMC_TIMEOUT"`
}

// NewOCloudConfig returns instance of OCloudConfig config type.
func NewOCloudConfig() *OCloudConfig {
	log.Print("Creating new OCloudConfig struct")

	var ocloudConf OCloudConfig
	ocloudConf.SystemTestsConfig = systemtestsconfig.NewSystemTestsConfig()

	_, filename, _, _ := runtime.Caller(0)
	baseDir := filepath.Dir(filename)
	confFile := filepath.Join(baseDir, PathToDefaultOCloudParamsFile)
	err := readFile(&ocloudConf, confFile)

	if err != nil {
		log.Printf("Error to read config file %s", confFile)

		return nil
	}

	err = readEnv(&ocloudConf)

	if err != nil {
		log.Print("Error to read environment variables")

		return nil
	}

	ocloudConf.Spoke1BMC = bmc.New(ocloudConf.Spoke1BMCHost).
		WithRedfishUser(ocloudConf.Spoke1BMCUsername, ocloudConf.Spoke1BMCPassword).
		WithRedfishTimeout(ocloudConf.Spoke1BMCTimeout).
		WithSSHUser(ocloudConf.Spoke1BMCUsername, ocloudConf.Spoke1BMCPassword)

	ocloudConf.Spoke2BMC = bmc.New(ocloudConf.Spoke2BMCHost).
		WithRedfishUser(ocloudConf.Spoke2BMCUsername, ocloudConf.Spoke2BMCPassword).
		WithRedfishTimeout(ocloudConf.Spoke2BMCTimeout).
		WithSSHUser(ocloudConf.Spoke2BMCUsername, ocloudConf.Spoke2BMCPassword)

	return &ocloudConf
}

func readFile(ocloudConfig *OCloudConfig, cfgFile string) error {
	openedCfgFile, err := os.Open(cfgFile)
	if err != nil {
		return err
	}

	defer func() {
		_ = openedCfgFile.Close()
	}()

	decoder := yaml.NewDecoder(openedCfgFile)
	err = decoder.Decode(&ocloudConfig)

	if err != nil {
		return err
	}

	return nil
}

func readEnv(ocloudConfig *OCloudConfig) error {
	err := envconfig.Process("", ocloudConfig)
	if err != nil {
		return err
	}

	return nil
}
