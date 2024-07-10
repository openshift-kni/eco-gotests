package rdsmanagementconfig

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/kelseyhightower/envconfig"
	"github.com/openshift-kni/eco-gotests/tests/internal/config"

	"gopkg.in/yaml.v2"
)

const (
	// PathToDefaultRDSManagementParamsFile path to config file with default RDSManagement parameters.
	PathToDefaultRDSManagementParamsFile = "./default.yaml"
)

// BMCDetails structure to hold BMC details.
type BMCDetails struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	BMCAddress string `json:"bmc"`
}

// EnvMapString holds a map[string]string parsed from environment variable.
type EnvMapString map[string]string

// Decode - method for envconfig package to parse environment variable.
func (ems *EnvMapString) Decode(value string) error {
	resultMap := make(map[string]string)

	for _, record := range strings.Split(value, ";;") {
		log.Printf("Processing record: %q", record)

		key := strings.Split(record, "===")[0]
		val := strings.Split(record, "===")[1]

		multiLine := ""

		if strings.Contains(val, `\n`) {
			for _, line := range strings.Split(val, `\n`) {
				multiLine += fmt.Sprintf("\n%s", line)
			}
		} else {
			multiLine = val
		}

		resultMap[key] = multiLine
	}

	*ems = resultMap

	return nil
}

// EnvSliceString holds a []string parsed from environment variable.
type EnvSliceString []string

// Decode - method for envconfig package to parse environment variable,
// as a separator triple pipe '|||' is used.
func (ess *EnvSliceString) Decode(value string) error {
	resultSlice := []string{}

	log.Printf("EnvSliceString: Processing record: %q", value)

	resultSlice = append(resultSlice, strings.Split(value, "|||")...)

	*ess = resultSlice

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

// ManagementConfig type keeps RDS Management configuration.
type ManagementConfig struct {
	*config.GeneralConfig
	// AppsNS is the namespace where the applications are installed.
	AppsNS             		string `yaml:"rdsmanagement_apps_ns" envconfig:"ECO_RDSMANAGEMENT_APPS_NS"`
	// PerformanceAddonNamespace is the namespace of the Performance Addon operator.
	PerformanceAddonNS      string `yaml:"rdsmanagement_performance_addon_ns" envconfig:"ECO_RDSMANAGEMENT_PERFORMANCE_ADDON_NS"`
	// OpenshiftVirtualizationNamespace is the namespace of the OpenShift Virtualization operator.
	OpenshiftVirtualizationNS      string `yaml:"rdsmanagement_openshift_virtualization_ns" envconfig:"ECO_RDSMANAGEMENT_OPENSHIFT_VIRTUALIZATION_NS"`
	// QuayNamespace is the namespace of Quay.
	QuayNS      string `yaml:"rdsmanagement_quay_ns" envconfig:"ECO_RDSMANAGEMENT_QUAY_NS"`
	// MetalLBNamespace is the namespace of MetalLB.
	MetalLBNS      string `yaml:"rdsmanagement_metallb_ns" envconfig:"ECO_RDSMANAGEMENT_METALLB_NS"`
	// ACMNamespace is the namespace of ACM.
	AcmNS      string `yaml:"rdsmanagement_acm_ns" envconfig:"ECO_RDSMANAGEMENT_ACM_NS"`
	// KafkaNamespace is the namespace of Kafka.
	KafkaNS      string `yaml:"rdsmanagement_kafka_ns" envconfig:"ECO_RDSMANAGEMENT_KAFKA_NS"`
	// KafkaAdapterNS is the namespace of Kafka.
	KafkaAdapterNS      string `yaml:"rdsmanagement_kafka_adapter_ns" envconfig:"ECO_RDSMANAGEMENT_KAFKA_NS"`
	// AnsibleNS is the namespace of Ansible Automation Platform.
	AnsibleNS      string `yaml:"rdsmanagement_ansible_ns" envconfig:"ECO_RDSMANAGEMENT_ANSIBLE_NS"`
	// Amq7NS is the name of the namespace of AMQ7.
	Amq7NS      string `yaml:"rdsmanagement_amq7_ns" envconfig:"ECO_RDSMANAGEMENT_AMQ7_NS"`
	// StfNS is the namespace of STF.
	StfNS      string `yaml:"rdsmanagement_stf_ns" envconfig:"ECO_RDSMANAGEMENT_STF_NS"`

	// KubeletCPUAllocation is the CPU allocated by the kubelet.
	KubeletCPUAllocation string `yaml:"rdsmanagement_kubelet_cpu_allocation_ns" envconfig:"ECO_RDSMANAGEMENT_KUBELET_CPU_ALLOCATION_NS"`
	//KubeletMemoryAllocation is the memory allocated by the kubelet.
	KubeletMemoryAllocation string `yaml:"rdsmanagement_kubelet_memory_allocation_ns" envconfig:"ECO_RDSMANAGEMENT_KUBELET_MEMORY_ALLOCATION_NS"`

	// IDMDeployed indicates whether IDM has been deployed or not
	IDMDeployed      bool `yaml:"rdsmanagement_idm_deployed" envconfig:"ECO_RDSMANAGEMENT_IDM_DEPLOYED"`
	// SatelliteDeployed indicates whether Satellite has been deployed or not
	SatelliteDeployed      bool `yaml:"rdsmanagement_satellite_deployed" envconfig:"ECO_RDSMANAGEMENT_SATELLITE_DEPLOYED"`
	// StfDeployed indicates whether STF has been deployed or not
	StfDeployed      bool `yaml:"rdsmanagement_stf_deployed" envconfig:"ECO_RDSMANAGEMENT_STF_DEPLOYED"`
}

// NewManagementConfig returns instance of ManagementConfig config type.
func NewManagementConfig() *ManagementConfig {
	log.Print("Creating new ManagementConfig struct")

	var rdsManagementConf ManagementConfig
	rdsManagementConf.GeneralConfig = config.NewConfig()

	var confFile string

	if fileFromEnv, exists := os.LookupEnv("ECO_RDS_MANAGEMENT_CONFIG_FILE_PATH"); !exists {
		_, filename, _, _ := runtime.Caller(0)
		baseDir := filepath.Dir(filename)
		confFile = filepath.Join(baseDir, PathToDefaultRDSManagementParamsFile)
	} else {
		confFile = fileFromEnv
	}

	log.Printf("Open config file %s", confFile)

	err := readFile(&rdsManagementConf, confFile)
	if err != nil {
		log.Printf("Error to read config file %s", confFile)

		return nil
	}

	err = readEnv(&rdsManagementConf)

	if err != nil {
		log.Print("Error to read environment variables")

		return nil
	}

	return &rdsManagementConf
}

func readFile(rdsConfig *ManagementConfig, cfgFile string) error {
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

func readEnv(rdsConfig *ManagementConfig) error {
	err := envconfig.Process("", rdsConfig)
	if err != nil {
		return err
	}

	return nil
}
