package vcoreconfig

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kelseyhightower/envconfig"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/system-tests/internal/systemtestsconfig"
	"gopkg.in/yaml.v2"
)

const (
	// PathToDefaultVCoreParamsFile path to config file with default vCore parameters.
	PathToDefaultVCoreParamsFile = "./default.yaml"
)

// VCoreConfig type keeps vCore configuration.
type VCoreConfig struct {
	*systemtestsconfig.SystemTestsConfig
	Namespace                   string `yaml:"vcore_default_ns" envconfig:"ECO_SYSTEM_VCORE_NS"`
	OdfLabel                    string `yaml:"odf_label" envconfig:"ECO_SYSTEM_VCORE_ODF_LABEL"`
	VCorePpLabel                string `yaml:"vcore_pp_label" envconfig:"ECO_SYSTEM_VCORE_PP_LABEL"`
	VCoreCpLabel                string `yaml:"vcore_cp_label" envconfig:"ECO_SYSTEM_VCORE_CP_LABEL"`
	Host                        string `yaml:"host" envconfig:"ECO_SYSTEM_VCORE_HOST"`
	User                        string `yaml:"user" envconfig:"ECO_SYSTEM_VCORE_USER"`
	Pass                        string `yaml:"pass" envconfig:"ECO_SYSTEM_VCORE_PASS"`
	ControlPlaneLabelListOption metav1.ListOptions
	OdfLabelListOption          metav1.ListOptions
	VCorePpLabelListOption      metav1.ListOptions
	VCoreCpLabelListOption      metav1.ListOptions
}

// NewVCoreConfig returns instance of VCoreConfig config type.
func NewVCoreConfig() *VCoreConfig {
	log.Print("Creating new VCoreConfig struct")

	var vcoreConf VCoreConfig
	vcoreConf.SystemTestsConfig = systemtestsconfig.NewSystemTestsConfig()

	_, filename, _, _ := runtime.Caller(0)
	baseDir := filepath.Dir(filename)
	confFile := filepath.Join(baseDir, PathToDefaultVCoreParamsFile)
	err := readFile(&vcoreConf, confFile)

	if err != nil {
		log.Printf("Error to read config file %s", confFile)

		return nil
	}

	err = readEnv(&vcoreConf)

	if err != nil {
		log.Print("Error to read environment variables")

		return nil
	}

	return &vcoreConf
}

func readFile(vcoreConfig *VCoreConfig, cfgFile string) error {
	openedCfgFile, err := os.Open(cfgFile)
	if err != nil {
		return err
	}

	defer func() {
		_ = openedCfgFile.Close()
	}()

	decoder := yaml.NewDecoder(openedCfgFile)
	err = decoder.Decode(&vcoreConfig)

	if err != nil {
		return err
	}

	return nil
}

func readEnv(vcoreConfig *VCoreConfig) error {
	err := envconfig.Process("", vcoreConfig)
	if err != nil {
		return err
	}

	vcoreConfig.OdfLabel = fmt.Sprintf("%s/%s", vcoreConfig.KubernetesRolePrefix, vcoreConfig.OdfLabel)
	vcoreConfig.VCorePpLabel = fmt.Sprintf("%s/%s", vcoreConfig.KubernetesRolePrefix, vcoreConfig.VCorePpLabel)
	vcoreConfig.VCoreCpLabel = fmt.Sprintf("%s/%s", vcoreConfig.KubernetesRolePrefix, vcoreConfig.VCoreCpLabel)
	vcoreConfig.ControlPlaneLabelListOption = metav1.ListOptions{LabelSelector: vcoreConfig.ControlPlaneLabel}
	vcoreConfig.OdfLabelListOption = metav1.ListOptions{LabelSelector: vcoreConfig.OdfLabel}
	vcoreConfig.VCorePpLabelListOption = metav1.ListOptions{LabelSelector: vcoreConfig.VCorePpLabel}
	vcoreConfig.VCoreCpLabelListOption = metav1.ListOptions{LabelSelector: vcoreConfig.VCoreCpLabel}

	return nil
}
