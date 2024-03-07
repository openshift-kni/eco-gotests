package rdscoreconfig

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	v1 "k8s.io/api/core/v1"

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

// TolerationList used to store tolerations for test workloads.
type TolerationList []v1.Toleration

// Decode - method for envconfig package to parse environment variable.
func (tl *TolerationList) Decode(value string) error {
	tmpTolerationList := []v1.Toleration{}

	for _, record := range strings.Split(value, ";") {
		log.Printf("Processing toleration record: %q", record)

		parsedToleration := v1.Toleration{}

		for _, parsedRecord := range strings.Split(record, ",") {
			switch strings.Split(parsedRecord, "=")[0] {
			case "key":
				parsedToleration.Key = strings.Split(parsedRecord, "=")[1]
			case "value":
				parsedToleration.Value = strings.Split(parsedRecord, "=")[1]
			case "effect":
				parsedToleration.Effect = v1.TaintEffect(strings.Split(parsedRecord, "=")[1])
			case "operator":
				parsedToleration.Operator = v1.TolerationOperator(strings.Split(parsedRecord, "=")[1])
			}
		}
		tmpTolerationList = append(tmpTolerationList, parsedToleration)
	}

	*tl = tmpTolerationList

	return nil
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
	WlkdSRIOVOneNS string `yaml:"rdscore_wlkd_sriov_one_ns" envconfig:"ECO_RDSCORE_WLKD_SRIOV_ONE_NS"`
	WlkdSRIOVTwoNS string `yaml:"rdscore_wlkd_sriov_two_ns" envconfig:"ECO_RDSCORE_WLKD_SRIOV_TWO_NS"`
	//nolint:lll
	PerformanceProfileHTName string         `yaml:"rdscore_performance_profile_ht_name" envconfig:"ECO_RDS_CORE_PERFORMANCE_PROFILE_HT_NAME"`
	WlkdTolerationList       TolerationList `yaml:"rdscore_tolerations_list" envconfig:"ECO_RDSCORE_TOLERATIONS_LIST"`
	//nolint:lll
	StorageODFWorkloadImage string      `yaml:"rdscore_storage_storage_wlkd_image" envconfig:"ECO_RDSCORE_STORAGE_WLKD_IMAGE"`
	NodesCredentialsMap     NodesBMCMap `yaml:"rdscore_nodes_bmc_map" envconfig:"ECO_RDSCORE_NODES_CREDENTIALS_MAP"`
	WlkdSRIOVDeployOneCmd   []string    `yaml:"rdscore_wlkd_sriov_one_cmd" envconfig:"ECO_RDSCORE_WLKD_SRIOV_ONE_CMD"`
	WlkdSRIOVDeployTwoCmd   []string    `yaml:"rdscore_wlkd_sriov_two_cmd" envconfig:"ECO_RDSCORE_WLKD_SRIOV_TWO_CMD"`
	WlkdSRIOVDeploy2OneCmd  []string    `yaml:"rdscore_wlkd2_sriov_one_cmd" envconfig:"ECO_RDSCORE_WLKD_SRIOV_2_ONE_CMD"`
	WlkdSRIOVDeploy2TwoCmd  []string    `yaml:"rdscore_wlkd2_sriov_two_cmd" envconfig:"ECO_RDSCORE_WLKD_SRIOV_2_TWO_CMD"`
	WlkdSRIOVDeployOneImage string      `yaml:"rdscore_wlkd_sriov_one_image" envconfig:"ECO_RDSCORE_WLKD_SRIOV_ONE_IMG"`
	WlkdSRIOVDeployTwoImage string      `yaml:"rdscore_wlkd_sriov_two_image" envconfig:"ECO_RDSCORE_WLKD_SRIOV_TWO_IMG"`
	//nolint:lll
	WlkdSRIOVConfigMapDataOne map[string]string `yaml:"rdscore_wlkd_sriov_cm_data_one" envconfig:"ECO_RDSCORE_SRIOV_CM_DATA_ONE"`
	//nolint:lll
	WlkdSRIOVConfigMapDataTwo map[string]string `yaml:"rdscore_wlkd_sriov_cm_data_two" envconfig:"ECO_RDSCORE_SRIOV_CM_DATA_TWO"`
	//nolint:lll
	StorageODFDeployOneSelector map[string]string `yaml:"rdscore_wlkd_odf_one_selector" envconfig:"ECO_RDSCORE_WLKD_ODF_ONE_SELECTOR"`
	//nolint:lll
	StorageODFDeployTwoSelector map[string]string `yaml:"rdscore_wlkd_odf_two_selector" envconfig:"ECO_RDSCORE_WLKD_ODF_TWO_SELECTOR"`
	//nolint:lll
	NodeSelectorHTNodes map[string]string `yaml:"rdscore_node_selector_ht_nodes" envconfig:"ECO_RDSCORE_NODE_SELECTOR_HT_NODES"`
	WlkdSRIOVNetOne     string            `yaml:"rdscore_wlkd_sriov_net_one" envconfig:"ECO_RDSCORE_WLKD_SRIOV_NET_ONE"`
	WlkdSRIOVNetTwo     string            `yaml:"rdscore_wlkd_sriov_net_two" envconfig:"ECO_RDSCORE_WLKD_SRIOV_NET_TWO"`
	WlkdSRIOVTwoSa      string            `yaml:"rdscore_wlkd_sriov_two_sa" envconfig:"ECO_RDSCORE_WLKD_SRIOV_TWO_SA"`
	//nolint:lll
	WlkdSRIOVDeployOneSelector map[string]string `yaml:"rdscore_wlkd_sriov_one_selector" envconfig:"ECO_RDSCORE_WLKD_SRIOV_ONE_SELECTOR"`
	//nolint:lll
	WlkdSRIOVDeployTwoSelector map[string]string `yaml:"rdscore_wlkd_sriov_two_selector" envconfig:"ECO_RDSCORE_WLKD_SRIOV_TWO_SELECTOR"`
	//nolint:lll
	WlkdSRIOVDeployOneTargetAddress string `yaml:"rdscore_wlkd_sriov_deploy_one_target" envconfig:"ECO_RDSCORE_SRIOV_WLKD_DEPLOY_ONE_TARGET"`
	//nolint:lll
	WlkdSRIOVDeployTwoTargetAddress string `yaml:"rdscore_wlkd_sriov_deploy_two_target" envconfig:"ECO_RDSCORE_SRIOV_WLKD_DEPLOY_TWO_TARGET"`
	//nolint:lll
	WlkdSRIOVDeploy2OneTargetAddress string `yaml:"rdscore_wlkd2_sriov_deploy_one_target" envconfig:"ECO_RDSCORE_SRIOV_WLKD2_DEPLOY_ONE_TARGET"`
	//nolint:lll
	WlkdSRIOVDeploy2TwoTargetAddress string `yaml:"rdscore_wlkd2_sriov_deploy_two_target" envconfig:"ECO_RDSCORE_SRIOV_WLKD2_DEPLOY_TWO_TARGET"`
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

func readFile(rdsConfig *CoreConfig, cfgFile string) error {
	openedCfgFile, err := os.Open(cfgFile)
	if err != nil {
		return err
	}

	defer func() {
		_ = openedCfgFile.Close()
	}()

	decoder := yaml.NewDecoder(openedCfgFile)
	err = decoder.Decode(&rdsConfig)

	if err != nil {
		return err
	}

	return nil
}

func readEnv(rdsConfig *CoreConfig) error {
	err := envconfig.Process("", rdsConfig)
	if err != nil {
		return err
	}

	return nil
}
