package ecoreconfig

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/kelseyhightower/envconfig"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/systemtestsconfig"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
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
	MCPOneName          string   `yaml:"ecore_mcp_one_name" envconfig:"ECO_SYSTEM_ECORE_MCP_ONE_NAME"`
	MCPTwoName          string   `yaml:"ecore_mcp_two_name" envconfig:"ECO_SYSTEM_ECORE_MCP_TWO_NAME"`
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

	WlkdSRIOVDeploy2OneCmd []string `yaml:"ecore_wlkd2_sriov_one_cmd" envconfig:"ECO_SYSTEM_ECORE_WLKD2_SRIOV_ONE_CMD"`
	WlkdSRIOVDeploy2TwoCmd []string `yaml:"ecore_wlkd2_sriov_two_cmd" envconfig:"ECO_SYSTEM_ECORE_WLKD2_SRIOV_TWO_CMD"`

	WlkdSRIOVDeploy3OneCmd []string `yaml:"ecore_wlkd3_sriov_one_cmd" envconfig:"ECO_SYSTEM_ECORE_WLKD3_SRIOV_ONE_CMD"`
	WlkdSRIOVDeploy3TwoCmd []string `yaml:"ecore_wlkd3_sriov_two_cmd" envconfig:"ECO_SYSTEM_ECORE_WLKD3_SRIOV_TWO_CMD"`

	WlkdSRIOVDeploy4OneCmd []string `yaml:"ecore_wlkd4_sriov_one_cmd" envconfig:"ECO_SYSTEM_ECORE_WLKD4_SRIOV_ONE_CMD"`
	WlkdSRIOVDeploy4TwoCmd []string `yaml:"ecore_wlkd4_sriov_two_cmd" envconfig:"ECO_SYSTEM_ECORE_WLKD4_SRIOV_TWO_CMD"`

	KernelModulesMap map[string][]string `yaml:"ecore_kernel_modules_map" envconfig:"ECO_SYSTEM_KERNEL_MODULES_MAP"`
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
	WlkdTolerationList      TolerationList    `yaml:"ecore_tolerations_list" envconfig:"ECO_SYSTEM_ECORE_TOLERATIONS_LIST"`
	//nolint:lll
	StorageODFDeployOneSelector map[string]string `yaml:"ecore_wlkd_odf_one_selector" envconfig:"ECO_SYSTEM_WLKD_ODF_ONE_SELECTOR"`
	//nolint:lll
	StorageODFDeployTwoSelector map[string]string `yaml:"ecore_wlkd_odf_two_selector" envconfig:"ECO_SYSTEM_WLKD_ODF_TWO_SELECTOR"`
	//nolint:lll
	WlkdSRIOVDeployOneTargetAddress string `yaml:"ecore_wlkd_sriov_deploy_one_target" envconfig:"ECO_SYSTEM_WLKD_SRIOV_DEPLOY_ONE_TARGET"`
	//nolint:lll
	WlkdSRIOVDeployOneTargetAddressIPv6 string `yaml:"ecore_wlkd_sriov_deploy_one_target_ipv6" envconfig:"ECO_SYSTEM_WLKD_SRIOV_DEPLOY_ONE_TARGET_IPV6"`
	//nolint:lll
	WlkdSRIOVDeployTwoTargetAddress string `yaml:"ecore_wlkd_sriov_deploy_two_target" envconfig:"ECO_SYSTEM_WLKD_SRIOV_DEPLOY_TWO_TARGET"`
	//nolint:lll
	WlkdSRIOVDeployTwoTargetAddressIPv6 string `yaml:"ecore_wlkd_sriov_deploy_two_target_ipv6" envconfig:"ECO_SYSTEM_WLKD_SRIOV_DEPLOY_TWO_TARGET_IPV6"`
	//nolint:lll
	WlkdSRIOVDeploy2OneTargetAddress string `yaml:"ecore_wlkd2_sriov_deploy_one_target" envconfig:"ECO_SYSTEM_WLKD2_SRIOV_DEPLOY_ONE_TARGET"`
	//nolint:lll
	WlkdSRIOVDeploy2OneTargetAddressIPv6 string `yaml:"ecore_wlkd2_sriov_deploy_one_target_ipv6" envconfig:"ECO_SYSTEM_WLKD2_SRIOV_DEPLOY_ONE_TARGET_IPV6"`
	//nolint:lll
	WlkdSRIOVDeploy2TwoTargetAddress string `yaml:"ecore_wlkd2_sriov_deploy_two_target" envconfig:"ECO_SYSTEM_WLKD2_SRIOV_DEPLOY_TWO_TARGET"`
	//nolint:lll
	WlkdSRIOVDeploy2TwoTargetAddressIPv6 string `yaml:"ecore_wlkd2_sriov_deploy_two_target_ipv6" envconfig:"ECO_SYSTEM_WLKD2_SRIOV_DEPLOY_TWO_TARGET_IPV6"`
	//nolint:lll
	WlkdSRIOVDeploy3OneTargetAddress string `yaml:"ecore_wlkd3_sriov_deploy_one_target" envconfig:"ECO_SYSTEM_WLKD3_SRIOV_DEPLOY_ONE_TARGET"`
	//nolint:lll
	WlkdSRIOVDeploy3OneTargetAddressIPv6 string `yaml:"ecore_wlkd3_sriov_deploy_one_target_ipv6" envconfig:"ECO_SYSTEM_WLKD3_SRIOV_DEPLOY_ONE_TARGET_IPV6"`
	//nolint:lll
	WlkdSRIOVDeploy3TwoTargetAddress string `yaml:"ecore_wlkd3_sriov_deploy_two_target" envconfig:"ECO_SYSTEM_WLKD3_SRIOV_DEPLOY_TWO_TARGET"`
	//nolint:lll
	WlkdSRIOVDeploy3TwoTargetAddressIPv6 string `yaml:"ecore_wlkd3_sriov_deploy_two_target_ipv6" envconfig:"ECO_SYSTEM_WLKD3_SRIOV_DEPLOY_TWO_TARGET_IPV6"`
	//nolint:lll
	WlkdSRIOVDeploy4OneTargetAddress string `yaml:"ecore_wlkd4_sriov_deploy_one_target" envconfig:"ECO_SYSTEM_WLKD4_SRIOV_DEPLOY_ONE_TARGET"`
	//nolint:lll
	WlkdSRIOVDeploy4OneTargetAddressIPv6 string `yaml:"ecore_wlkd4_sriov_deploy_one_target_ipv6" envconfig:"ECO_SYSTEM_WLKD4_SRIOV_DEPLOY_ONE_TARGET_IPV6"`
	//nolint:lll
	WlkdSRIOVDeploy4TwoTargetAddress string `yaml:"ecore_wlkd4_sriov_deploy_two_target" envconfig:"ECO_SYSTEM_WLKD4_SRIOV_DEPLOY_TWO_TARGET"`
	//nolint:lll
	WlkdSRIOVDeploy4TwoTargetAddressIPv6 string `yaml:"ecore_wlkd4_sriov_deploy_two_target_ipv6" envconfig:"ECO_SYSTEM_WLKD4_SRIOV_DEPLOY_TWO_TARGET_IPV6"`
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
