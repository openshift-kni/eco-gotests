package cnfconfig

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/golang/glog"
	"github.com/kelseyhightower/envconfig"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedinstall/cnf/internal/cnfparams"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedinstall/internal/ibiconfig"
	"gopkg.in/yaml.v2"
)

const (
	// PathToDefaultIbuCnfParamsFile path to config file with default system tests parameters.
	PathToDefaultIbuCnfParamsFile = "./default.yaml"
)

// CNFConfig type contains cnf configuration.
type CNFConfig struct {
	*ibiconfig.IBIConfig
	TargetHubKubeConfig string `envconfig:"ECO_LCA_IBI_CNF_KUBECONFIG_TARGET_HUB"`
	TargetSNOKubeConfig string `envconfig:"ECO_LCA_IBI_CNF_KUBECONFIG_TARGET_SNO"`
	SpokeName           string `envconfig:"ECO_CNF_RAN_SPOKE1_NAME"`
}

// NewCNFConfig returns instance of CNFConfig type.
func NewCNFConfig() *CNFConfig {
	glog.V(cnfparams.CNFLogLevel).Info("Creating new CNFConfig struct")

	var cnfConfig CNFConfig
	cnfConfig.IBIConfig = ibiconfig.NewIBIConfig()

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
