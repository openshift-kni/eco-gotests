package netconfig

import (
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/kelseyhightower/envconfig"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/internal/coreconfig"
	"gopkg.in/yaml.v2"
)

const (
	// PathToDefaultCnfCoreNetParamsFile path to config file with default network parameters.
	PathToDefaultCnfCoreNetParamsFile = "./default.yaml"
)

// NetworkConfig type keeps network configuration.
type NetworkConfig struct {
	*coreconfig.CoreConfig
	CnfNetTestContainer    string `yaml:"cnf_net_test_container" envconfig:"ECO_CNF_CORE_NET_TEST_CONTAINER"`
	SriovOperatorNamespace string `yaml:"sriov_operator_namespace" envconfig:"ECO_CNF_CORE_NET_SRIOV_OPERATOR_NAMESPACE"`
	MlbOperatorNamespace   string `yaml:"metal_lb_operator_namespace" envconfig:"ECO_CNF_CORE_NET_MLB_OPERATOR_NAMESPACE"`
	//nolint:lll
	PrometheusOperatorNamespace string `yaml:"prometheus_operator_namespace" envconfig:"ECO_CNF_CORE_NET_PROMETHEUS_OPERATOR_NAMESPACE"`
	MlbAddressPoolIP            string `envconfig:"ECO_CNF_CORE_NET_MLB_ADDR_LIST"`
	SriovInterfaces             string `envconfig:"ECO_CNF_CORE_NET_SRIOV_INTERFACE_LIST"`
	FrrImage                    string `yaml:"frr_image" envconfig:"ECO_CNF_CORE_NET_FRR_IMAGE"`
}

// NewNetConfig returns instance of NetworkConfig config type.
func NewNetConfig() *NetworkConfig {
	log.Print("Creating new NetworkConfig struct")

	var netConf NetworkConfig
	netConf.CoreConfig = coreconfig.NewCoreConfig()

	_, filename, _, _ := runtime.Caller(0)
	baseDir := filepath.Dir(filename)
	confFile := filepath.Join(baseDir, PathToDefaultCnfCoreNetParamsFile)
	err := readFile(&netConf, confFile)

	if err != nil {
		log.Printf("Error to read config file %s", confFile)

		return nil
	}

	err = readEnv(&netConf)

	if err != nil {
		log.Print("Error to read environment variables")

		return nil
	}

	return &netConf
}

// GetMetalLbVirIP IPv4 checks the metalLbIP environmental variable and returns the list of give ip addresses.
func (netConfig *NetworkConfig) GetMetalLbVirIP() ([]string, error) {
	envValue := strings.Split(netConfig.MlbAddressPoolIP, ",")

	if len(envValue) < 2 {
		return nil, fmt.Errorf(
			"the number of virtial metalLb ip address is less than 2, check ECO_CNF_CORE_NET_MLB_ADDR_LIST env var")
	}

	for _, v := range envValue {
		if net.ParseIP(v) == nil {
			return nil, fmt.Errorf("the environment IP variable is not a valid IP")
		}
	}

	return envValue, nil
}

// GetSriovInterfaces checks the ECO_CNF_CORE_NET_SRIOV_INTERFACE_LIST env var
// and returns required number of SR-IOV interfaces.
func (netConfig *NetworkConfig) GetSriovInterfaces(requestedNumber int) ([]string, error) {
	requestedInterfaceList := strings.Split(netConfig.SriovInterfaces, ",")
	if len(requestedInterfaceList) < requestedNumber {
		return nil, fmt.Errorf(
			"the number of SR-IOV interfaces is less than %d,"+
				" check ECO_CNF_CORE_NET_SRIOV_INTERFACE_LIST env var", requestedNumber)
	}

	return requestedInterfaceList, nil
}

func readFile(netConfig *NetworkConfig, cfgFile string) error {
	openedCfgFile, err := os.Open(cfgFile)
	if err != nil {
		return err
	}

	defer func() {
		_ = openedCfgFile.Close()
	}()

	decoder := yaml.NewDecoder(openedCfgFile)
	err = decoder.Decode(&netConfig)

	if err != nil {
		return err
	}

	return nil
}

func readEnv(netConfig *NetworkConfig) error {
	err := envconfig.Process("", netConfig)
	if err != nil {
		return err
	}

	return nil
}
