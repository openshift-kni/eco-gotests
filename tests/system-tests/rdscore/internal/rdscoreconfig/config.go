package rdscoreconfig

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/kelseyhightower/envconfig"
	"github.com/openshift-kni/eco-gotests/tests/internal/config"

	"gopkg.in/yaml.v2"
)

const (
	// PathToDefaultRDSCoreParamsFile path to config file with default RDSCore parameters.
	PathToDefaultRDSCoreParamsFile = "./default.yaml"
)

// BMCDetails structure to hold BMC details.
type BMCDetails struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	BMCAddress string `json:"bmc"`
}

// NodesBMCMap holds info about BMC connection for a specific node.
type NodesBMCMap map[string]BMCDetails

// Decode - method for envconfig package to parse JSON encoded environment variables.
func (nad *NodesBMCMap) Decode(value string) error {
	nodesAuthMap := make(map[string]BMCDetails)

	for _, record := range strings.Split(value, ";") {
		log.Printf("Processing: %v", record)

		parsedRecord := strings.Split(record, ",")
		if len(parsedRecord) != 4 {
			log.Printf("Error to parse data %v", value)
			log.Printf("Expected 4 entries, found %d", len(parsedRecord))

			return fmt.Errorf("error parsing data %v", value)
		}

		nodesAuthMap[parsedRecord[0]] = BMCDetails{
			Username:   parsedRecord[1],
			Password:   parsedRecord[2],
			BMCAddress: parsedRecord[3],
		}
	}

	*nad = nodesAuthMap

	return nil
}

// CoreConfig type keeps RDS Core configuration.
type CoreConfig struct {
	*config.GeneralConfig
	NamespacePCC string `yaml:"rcore_nad_workload_ns_one" envconfig:"ECO_RDS_CORE_NAD_WORKLOAD_NS_ONE"`
	NamespacePCG string `yaml:"rcore_nad_workload_ns_two" envconfig:"ECO_RDS_CORE_NAD_WORKLOAD_NS_TWO"`
	//nolint:lll
	PerformanceProfileHTName string `yaml:"rdscore_performance_profile_ht_name" envconfig:"ECO_RDS_CORE_PERFORMANCE_PROFILE_HT_NAME"`
	//nolint:lll
	NodeSelectorHTNodes map[string]string `yaml:"rdscore_node_selector_ht_nodes" envconfig:"ECO_RDSCORE_NODE_SELECTOR_HT_NODES"`
}

// NewCoreConfig returns instance of CoreConfig config type.
func NewCoreConfig() *CoreConfig {
	log.Print("Creating new CoreConfig struct")

	var rdsCoreConf CoreConfig
	rdsCoreConf.GeneralConfig = config.NewConfig()

	var confFile string

	if fileFromEnv, exists := os.LookupEnv("ECO_RDS_CORE_CONFIG_FILE_PATH"); !exists {
		_, filename, _, _ := runtime.Caller(0)
		baseDir := filepath.Dir(filename)
		confFile = filepath.Join(baseDir, PathToDefaultRDSCoreParamsFile)
	} else {
		confFile = fileFromEnv
	}

	log.Printf("Open config file %s", confFile)

	err := readFile(&rdsCoreConf, confFile)
	if err != nil {
		log.Printf("Error to read config file %s", confFile)

		return nil
	}

	err = readEnv(&rdsCoreConf)

	if err != nil {
		log.Print("Error to read environment variables")

		return nil
	}

	return &rdsCoreConf
}

func readFile(ecoreConfig *CoreConfig, cfgFile string) error {
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

func readEnv(ecoreConfig *CoreConfig) error {
	err := envconfig.Process("", ecoreConfig)
	if err != nil {
		return err
	}

	return nil
}
