package cnfconfig

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/golang/glog"
	"github.com/kelseyhightower/envconfig"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/cnf/internal/cnfparams"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/internal/ibuconfig"
	"gopkg.in/yaml.v2"
)

const (
	// PathToDefaultIbuCnfParamsFile path to config file with default system tests parameters.
	PathToDefaultIbuCnfParamsFile = "./default.yaml"
)

// CNFConfig type contains cnf configuration.
type CNFConfig struct {
	*ibuconfig.IBUConfig
	IBUWorkloadImage      string `yaml:"ibu_workload_image" envconfig:"ECO_LCA_IBU_CNF_WORKLOAD_IMAGE"`
	TargetHubKubeConfig   string `envconfig:"ECO_LCA_IBU_CNF_KUBECONFIG_TARGET_HUB"`
	TargetSNOKubeConfig   string `envconfig:"ECO_LCA_IBU_CNF_KUBECONFIG_TARGET_SNO"`
	IbuWorkloadNS         string `yaml:"ibu_workload_validation_ns" envconfig:"ECO_LCA_IBU_CNF_WORKLOAD_NS"`
	IbuWorkloadPVNS       string `yaml:"ibu_workload_validation_pv_ns" envconfig:"ECO_LCA_IBU_CNF_WORKLOAD_PV_NS"`
	IbuWorkloadPVPod      string `yaml:"ibu_workload_validation_pv_pod_name" envconfig:"ECO_LCA_IBU_CNF_WORKLOAD_PV_POD"`
	IbuWorkloadPVFilePath string `yaml:"ibu_workload_validation_pv_file_path" envconfig:"ECO_LCA_IBU_CNF_WORKLOAD_PV_FILE"`
	IbuKcatImage          string `yaml:"ibu_kcat_image" envconfig:"ECO_LCA_IBU_CNF_KCAT_IMAGE"`
	IbuKcatBroker         string `yaml:"ibu_kcat_broker" envconfig:"ECO_LCA_IBU_CNF_KCAT_BROKER"`
	IbuKcatTopic          string `yaml:"ibu_kcat_topic" envconfig:"ECO_LCA_IBU_CNF_KCAT_TOPIC"`
}

// NewCNFConfig returns instance of CNFConfig type.
func NewCNFConfig() *CNFConfig {
	glog.V(cnfparams.CNFLogLevel).Info("Creating new CNFConfig struct")

	var cnfConfig CNFConfig
	cnfConfig.IBUConfig = ibuconfig.NewIBUConfig()

	_, filename, _, _ := runtime.Caller(0)
	baseDir := filepath.Dir(filename)
	configFile := filepath.Join(baseDir, PathToDefaultIbuCnfParamsFile)

	err := readFile(&cnfConfig, configFile)
	if err != nil {
		glog.V(cnfparams.CNFLogLevel).Infof("Error reading config file %s", configFile)

		return nil
	}

	err = readEnv(&cnfConfig)
	if err != nil {
		glog.V(cnfparams.CNFLogLevel).Infof("Error reading environment variables")

		return nil
	}

	err = envconfig.Process("eco_lca_ibu_cnf_", &cnfConfig)
	if err != nil {
		return nil
	}

	return &cnfConfig
}

func readFile(cnfConfig *CNFConfig, configFile string) error {
	openedConfigFile, err := os.Open(configFile)
	if err != nil {
		return err
	}

	defer func() {
		_ = openedConfigFile.Close()
	}()

	decoder := yaml.NewDecoder(openedConfigFile)

	err = decoder.Decode(&cnfConfig)

	return err
}

func readEnv(cnfConfig *CNFConfig) error {
	err := envconfig.Process("", cnfConfig)

	return err
}
