package ecoreconfig

import (
	"encoding/json"
	"fmt"
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
	PerformanceProfileHTName  string `yaml:"ecore_performance_profile_ht_name" envconfig:"ECO_SYSTEM_ECORE_PERFORMANCE_PROFILE_HT_NAME"`
	WlkdSRIOVConfigMapNamePCG string `yaml:"ecore_wlkd_sriov_cm_name_pcg" envconfig:"ECO_SYSTEM_ECORE_SRIOV_CM_NAME_PCG"`
	//nolint:lll
	WlkdSRIOVConfigMapDataPCG map[string]string `yaml:"ecore_wlkd_sriov_cm_data_pcg" envconfig:"ECO_SYSTEM_ECORE_SRIOV_CM_DATA_PCG"`
	//nolint:lll
	WlkdSRIOVDeployOneName string `yaml:"ecore_wlkd_sriov_deploy_one_name" envconfig:"ECO_SYSTEM_ECORE_WLKD_SRIOV_DEPLOY_ONE_NAME"`
	//nolint:lll
	WlkdSRIOVDeployTwoName  string `yaml:"ecore_wlkd_sriov_deploy_two_name" envconfig:"ECO_SYSTEM_ECORE_WLKD_SRIOV_DEPLOY_TWO_NAME"`
	WlkdSRIOVDeployOneImage string `yaml:"ecore_wlkd_sriov_one_image" envconfig:"ECO_SYSTEM_ECORE_WLKD_SRIOV_ONE_IMG"`
	WlkdSRIOVDeployTwoImage string `yaml:"ecore_wlkd_sriov_two_image" envconfig:"ECO_SYSTEM_ECORE_WLKD_SRIOV_TWO_IMG"`

	WlkdSRIOVDeployOneCmd []string `yaml:"ecore_wlkd_sriov_one_cmd" envconfig:"ECO_SYSTEM_ECORE_WLKD_SRIOV_ONE_CMD"`
	WlkdSRIOVDeployTwoCmd []string `yaml:"ecore_wlkd_sriov_two_cmd" envconfig:"ECO_SYSTEM_ECORE_WLKD_SRIOV_TWO_CMD"`

	//nolint:lll
	WlkdSRIOVDeployOneSelector map[string]string `yaml:"ecore_wlkd_sriov_one_selector" envconfig:"ECO_SYSTEM_WLKD_SRIOV_ONE_SELECTOR"`
	//nolint:lll
	WlkdSRIOVDeployTwoSelector map[string]string `yaml:"ecore_wlkd_sriov_two_selector" envconfig:"ECO_SYSTEM_WLKD_SRIOV_TWO_SELECTOR"`

	WlkdSRIOVOneSa string `yaml:"ecore_wlkd_sriov_one_sa" envconfig:"ECO_SYSTEM_ECORE_WLKD_SRIOV_ONE_SA"`
	WlkdSRIOVTwoSa string `yaml:"ecore_wlkd_sriov_two_sa" envconfig:"ECO_SYSTEM_ECORE_WLKD_SRIOV_TWO_SA"`

	WlkdSRIOVNetOne string `yaml:"ecore_wlkd_sriov_net_one" envconfig:"ECO_SYSTEM_ECORE_WLKD_SRIOV_NET_ONE"`
	WlkdSRIOVNetTwo string `yaml:"ecore_wlkd_sriov_net_two" envconfig:"ECO_SYSTEM_ECORE_WLKD_SRIOV_NET_TWO"`

	//nolint:lll
	StorageODFWorkloadImage string            `yaml:"ecore_storage_odf_wlkd_image" envconfig:"ECO_SYSTEM_ECORE_ODF_WLKD_IMAGE"`
	StorageClassesMap       map[string]string `yaml:"ecore_storage_classes_map" envconfig:"ECO_SYSTEM_SC_MAP"`
	NodesCredentialsMap     NodesBMCMap       `yaml:"ecore_nodes_bmc_map" envconfig:"ECO_SYSTEM_NODES_CREDENTIALS_MAP"`
	//nolint:lll
	StorageODFDeployOneSelector map[string]string `yaml:"ecore_wlkd_odf_one_selector" envconfig:"ECO_SYSTEM_WLKD_ODF_ONE_SELECTOR"`
	//nolint:lll
	StorageODFDeployTwoSelector map[string]string `yaml:"ecore_wlkd_odf_two_selector" envconfig:"ECO_SYSTEM_WLKD_ODF_TWO_SELECTOR"`
}

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
	nodesAuthMap := new(map[string]BMCDetails)

	err := json.Unmarshal([]byte(value), nodesAuthMap)

	if err != nil {
		log.Printf("Error to parse data %v", err)

		return fmt.Errorf("invalid map json: %w", err)
	}

	*nad = *nodesAuthMap

	return nil
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
