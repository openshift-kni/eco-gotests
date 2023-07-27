package spkconfig

import (
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/kelseyhightower/envconfig"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/systemtestsconfig"
	"gopkg.in/yaml.v2"
)

const (
	// PathToDefaultSPKParamsFile path to config file with default SPK parameters.
	PathToDefaultSPKParamsFile = "./default.yaml"
)

// SPKConfig type keeps SPK configuration.
type SPKConfig struct {
	*systemtestsconfig.SystemTestsConfig
	Namespace         string `yaml:"spk_workload_ns" envconfig:"ECO_SYSTEM_SPK_WORKLOAD_NS"`
	IngressTCPIPv4URL string `yaml:"spk_ingress_tcp_ipv4_url" envconfig:"ECO_SYSTEM_SPK_INGRESS_TCP_IPV4_URL"`
	IngressTCPIPv6URL string `yaml:"spk_ingress_tcp_ipv6_url" envconfig:"ECO_SYSTEM_SPK_INGRESS_TCP_IPV6_URL"`
}

// NewSPKConfig returns instance of SPKConfig config type.
func NewSPKConfig() *SPKConfig {
	log.Print("Creating new SPKConfig struct")

	var spkConf SPKConfig
	spkConf.SystemTestsConfig = systemtestsconfig.NewSystemTestsConfig()

	_, filename, _, _ := runtime.Caller(0)
	baseDir := filepath.Dir(filename)
	confFile := filepath.Join(baseDir, PathToDefaultSPKParamsFile)
	err := readFile(&spkConf, confFile)

	if err != nil {
		log.Printf("Error to read config file %s", confFile)

		return nil
	}

	err = readEnv(&spkConf)

	if err != nil {
		log.Print("Error to read environment variables")

		return nil
	}

	return &spkConf
}

func readFile(spkConfig *SPKConfig, cfgFile string) error {
	openedCfgFile, err := os.Open(cfgFile)
	if err != nil {
		return err
	}

	defer func() {
		_ = openedCfgFile.Close()
	}()

	decoder := yaml.NewDecoder(openedCfgFile)
	err = decoder.Decode(&spkConfig)

	if err != nil {
		return err
	}

	return nil
}

func readEnv(spkConfig *SPKConfig) error {
	err := envconfig.Process("", spkConfig)
	if err != nil {
		return err
	}

	return nil
}
