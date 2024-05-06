package ranconfig

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/golang/glog"
	"github.com/kelseyhightower/envconfig"
	"github.com/openshift-kni/eco-gotests/tests/cnf/internal/cnfconfig"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/ranparam"
	"gopkg.in/yaml.v2"
)

const (
	// PathToDefaultCnfRanParamsFile path to config file with default system tests parameters.
	PathToDefaultCnfRanParamsFile = "./default.yaml"
)

// RANConfig contains configuration for the RAN directory.
type RANConfig struct {
	*cnfconfig.CNFConfig
	MetricSamplingInterval string `yaml:"metricSamplingInterval" envconfig:"ECO_CNF_RAN_METRIC_SAMPLING_INTERVAL"`
	NoWorkloadDuration     string `yaml:"noWorkloadDuration" envconfig:"ECO_CNF_RAN_NO_WORKLOAD_DURATION"`
	WorkloadDuration       string `yaml:"workloadDuration" envconfig:"ECO_CNF_RAN_WORKLOAD_DURATION"`
	StressngTestImage      string `yaml:"stressngTestImage" envconfig:"ECO_CNF_RAN_STRESSNG_TEST_IMAGE"`
	CnfTestImage           string `yaml:"cnfTestImage" envconfig:"ECO_CNF_RAN_TEST_IMAGE"`
	HubKubeconfig          string `envconfig:"ECO_CNF_RAN_KUBECONFIG_HUB"`
	Spoke2Kubeconfig       string `envconfig:"ECO_CNF_RAN_KUBECONFIG_SPOKE2"`
	BmcUsername            string `yaml:"bmcUsername" envconfig:"ECO_CNF_RAN_BMC_USERNAME"`
	BmcPassword            string `yaml:"bmcPassword" envconfig:"ECO_CNF_RAN_BMC_PASSWORD"`
	BmcHosts               string `yaml:"bmcHosts" envconfig:"ECO_CNF_RAN_BMC_HOSTS"`
}

// NewRANConfig returns an instance of RANConfig.
func NewRANConfig() *RANConfig {
	glog.V(ranparam.LogLevel).Infof("Creating new RANConfig struct")

	var ranConfig RANConfig
	ranConfig.CNFConfig = cnfconfig.NewCNFConfig()

	_, filename, _, _ := runtime.Caller(0)
	baseDir := filepath.Dir(filename)
	configFile := filepath.Join(baseDir, PathToDefaultCnfRanParamsFile)

	err := readFile(&ranConfig, configFile)
	if err != nil {
		glog.V(ranparam.LogLevel).Infof("Error reading config file %s", configFile)

		return nil
	}

	err = readEnv(&ranConfig)
	if err != nil {
		glog.V(ranparam.LogLevel).Infof("Error reading environment variables")

		return nil
	}

	return &ranConfig
}

func readFile(ranConfig *RANConfig, configFile string) error {
	openedConfigFile, err := os.Open(configFile)
	if err != nil {
		return err
	}

	defer func() {
		_ = openedConfigFile.Close()
	}()

	decoder := yaml.NewDecoder(openedConfigFile)

	err = decoder.Decode(&ranConfig)

	return err
}

func readEnv(ranConfig *RANConfig) error {
	err := envconfig.Process("", ranConfig)

	return err
}
