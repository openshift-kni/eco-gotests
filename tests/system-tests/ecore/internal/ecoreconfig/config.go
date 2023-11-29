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
	NamespacePCC        string   `yaml:"ecore_nad_workload_ns_pcc" envconfig:"ECO_SYSTEM_ECORE_NAD_WORKLOAD_NS_PCC"`
	NamespacePCG        string   `yaml:"ecore_nad_workload_ns_pcg" envconfig:"ECO_SYSTEM_ECORE_NAD_WORKLOAD_NS_PCG"`
	NADListPCC          []string `yaml:"ecore_nad_nad_list_pcc" envconfig:"ECO_SYSTEM_ECORE_NAD_NAD_LIST_PCC"`
	NADListPCG          []string `yaml:"ecore_nad_nad_list_pcg" envconfig:"ECO_SYSTEM_ECORE_NAD_NAD_LIST_PCG"`
	MCPList             []string `yaml:"ecore_mcp_list" envconfig:"ECO_SYSTEM_ECORE_MCP_LIST"`
	PolicyNS            string   `yaml:"ecore_policy_ns" envconfig:"ECO_SYSTEM_ECORE_POLICY"`
	NADConfigMapPCCName string   `yaml:"ecore_nad_workload_pcc_cm_name" envconfig:"ECO_SYSTEM_ECORE_NAD_CM_NAME_PCC"`
	KubeletConfigCPName string   `yaml:"ecore_kublet_config_name_cp" envconfig:"ECO_SYSTEM_ECORE_KUBELET_CONFIG_NAME_CP"`
	//nolint:lll
	NodeSelectorHTNodes map[string]string `yaml:"ecore_node_selector_ht_nodes" envconfig:"ECO_SYSTEM_ECORE_NODE_SELECTOR_HT_NODES"`
	//nolint:lll
	NADConfigMapPCCData map[string]string `yaml:"ecore_nad_workload_pcc_cm_data" envconfig:"ECO_SYSTEM_ECORE_NAD_CM_DATA_PCC"`
	//nolint:lll
	NADWlkdDeployOnePCCName string   `yaml:"ecore_nad_workload_one_pcc_name" envconfig:"ECO_SYSTEM_ECORE_NAD_WLKD_DEPLOY_ONE_PCC_NAME"`
	NADWlkdDeployOnePCCCmd  []string `yaml:"ecore_nad_wlkd_one_pcc_cmd" envconfig:"ECO_SYSTEM_ECORE_NAD_WLKD_ONE_PCC_CMD"`
	NADWlkdDeployTwoPCCCmd  []string `yaml:"ecore_nad_wlkd_two_pcc_cmd" envconfig:"ECO_SYSTEM_ECORE_NAD_WLKD_TWO_PCC_CMD"`
	//nolint:lll
	NADWlkdDeployTwoPCCName  string `yaml:"ecore_nad_workload_two_pcc_name" envconfig:"ECO_SYSTEM_ECORE_NAD_WLKD_DEPLOY_TWO_PCC_NAME"`
	NADWlkdDeployOnePCCImage string `yaml:"ecore_nad_wlkd_one_image_pcc" envconfig:"ECO_SYSTEM_ECORE_NAD_WLKD_ONE_IMG_PCC"`
	NADWlkdDeployTwoPCCImage string `yaml:"ecore_nad_wlkd_two_image_pcc" envconfig:"ECO_SYSTEM_ECORE_NAD_WLKD_TWO_IMG_PCC"`
	NADWlkdOneNadName        string `yaml:"ecore_nad_wlkd_one_nad_name" envconfig:"ECO_SYSTEM_ECORE_NAD_WLKD_ONE_NAD_NAME"`
	NADWlkdTwoNadName        string `yaml:"ecore_nad_wlkd_two_nad_name" envconfig:"ECO_SYSTEM_ECORE_NAD_WLKD_TWO_NAD_NAME"`
	NADWlkdOnePCCSa          string `yaml:"ecore_nad_wlkd_one_pcc_sa" envconfig:"ECO_SYSTEM_ECORE_NAD_WLKD_ONE_PCC_SA"`
	NADWlkdTwoPCCSa          string `yaml:"ecore_nad_wlkd_two_pcc_sa" envconfig:"ECO_SYSTEM_ECORE_NAD_WLKD_TWO_PCC_SA"`
	NADConfigMapPCGName      string `yaml:"ecore_nad_workload_pcg_cm_name" envconfig:"ECO_SYSTEM_ECORE_NAD_CM_NAME_PCG"`
	//nolint:lll
	NADConfigMapPCGData map[string]string `yaml:"ecore_nad_workload_pcg_cm_data" envconfig:"ECO_SYSTEM_ECORE_NAD_CM_DATA_PCG"`
	//nolint:lll
	NADWlkdDeployOnePCCSelector map[string]string `yaml:"ecore_nad_wlkd_one_pcc_selector" envconfig:"ECO_SYSTEM_NAD_WLKD_ONE_PCC_SELECTOR"`
	//nolint:lll
	NADWlkdDeployTwoPCCSelector map[string]string `yaml:"ecore_nad_wlkd_two_pcc_selector" envconfig:"ECO_SYSTEM_NAD_WLKD_TWO_PCC_SELECTOR"`
	//nolint:lll
	NADWlkdDeployOnePCGSelector map[string]string `yaml:"ecore_nad_wlkd_one_pcg_selector" envconfig:"ECO_SYSTEM_NAD_WLKD_ONE_PCG_SELECTOR"`
	//nolint:lll
	KubeletConfigStandardName string `yaml:"ecore_kublet_config_name_standard" envconfig:"ECO_SYSTEM_ECORE_KUBELET_CONFIG_NAME_STANDARD"`
	//nolint:lll
	PerformanceProfileHTName string `yaml:"ecore_performance_profile_ht_name" envconfig:"ECO_SYSTEM_ECORE_PERFORMANCE_PROFILE_HT_NAME"`
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
