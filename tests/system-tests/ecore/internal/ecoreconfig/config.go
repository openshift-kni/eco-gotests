package ecoreconfig

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
	// PathToDefaultECoreParamsFile path to config file with default ECore parameters.
	PathToDefaultECoreParamsFile = "./default.yaml"
)

// ECoreConfig type keeps ECore configuration.
type ECoreConfig struct {
	*systemtestsconfig.SystemTestsConfig
	NamespacePCC string   `yaml:"ecore_nad_workload_ns_pcc" envconfig:"ECO_SYSTEM_ECORE_NAD_WORKLOAD_NS_PCC"`
	NamespacePCG string   `yaml:"ecore_nad_workload_ns_pcg" envconfig:"ECO_SYSTEM_ECORE_NAD_WORKLOAD_NS_PCG"`
	NADListPCC   []string `yaml:"ecore_nad_nad_list_pcc" envconfig:"ECO_SYSTEM_ECORE_NAD_NAD_LIST_PCC"`
	NADListPCG   []string `yaml:"ecore_nad_nad_list_pcg" envconfig:"ECO_SYSTEM_ECORE_NAD_NAD_LIST_PCG"`
}

// NewECoreConfig returns instance of ECoreConfig config type.
func NewECoreConfig() *ECoreConfig {
	log.Print("Creating new ECoreConfig struct")

	var ecoreConf ECoreConfig
	ecoreConf.SystemTestsConfig = systemtestsconfig.NewSystemTestsConfig()

	var confFile string

	if fileFromEnv, exists := os.LookupEnv("ECO_SYSTEM_ECORE_CONFIG_FILE_PATH"); !exists {
		_, filename, _, _ := runtime.Caller(0)
		baseDir := filepath.Dir(filename)
		confFile = filepath.Join(baseDir, PathToDefaultECoreParamsFile)
	} else {
		confFile = fileFromEnv
	}

	log.Printf("Open config file %s", confFile)

	err := readFile(&ecoreConf, confFile)
	if err != nil {
		log.Printf("Error to read config file %s", confFile)

		return nil
	}

	err = readEnv(&ecoreConf)

	if err != nil {
		log.Print("Error to read environment variables")

		return nil
	}

	return &ecoreConf
}

func readFile(ecoreConfig *ECoreConfig, cfgFile string) error {
	openedCfgFile, err := os.Open(cfgFile)
	if err != nil {
		return err
	}

	defer func() {
		_ = openedCfgFile.Close()
	}()

	decoder := yaml.NewDecoder(openedCfgFile)
	err = decoder.Decode(&ecoreConfig)

	if err != nil {
		return err
	}

	return nil
}

func readEnv(ecoreConfig *ECoreConfig) error {
	err := envconfig.Process("", ecoreConfig)
	if err != nil {
		return err
	}

	return nil
}
