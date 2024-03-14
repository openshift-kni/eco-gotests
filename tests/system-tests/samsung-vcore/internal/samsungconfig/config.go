package samsungconfig

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kelseyhightower/envconfig"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/systemtestsconfig"
	"gopkg.in/yaml.v2"
)

const (
	// PathToDefaultSamsungParamsFile path to config file with default Samsung parameters.
	PathToDefaultSamsungParamsFile = "./default.yaml"
)

// SamsungConfig type keeps Samsung configuration.
type SamsungConfig struct {
	*systemtestsconfig.SystemTestsConfig
	Namespace                   string `yaml:"samsung_default_ns" envconfig:"ECO_SYSTEM_SAMSUNG_NS"`
	OdfLabel                    string `yaml:"odf_label" envconfig:"ECO_SYSTEM_SAMSUNG_ODF_LABEL"`
	SamsungPpLabel              string `yaml:"samsung_pp_label" envconfig:"ECO_SYSTEM_SAMSUNG_PP_LABEL"`
	SamsungCnfLabel             string `yaml:"samsung_cnf_label" envconfig:"ECO_SYSTEM_SAMSUNG_CNF_LABEL"`
	Host                        string `yaml:"host" envconfig:"ECO_SYSTEM_SAMSUNG_HOST"`
	User                        string `yaml:"user" envconfig:"ECO_SYSTEM_SAMSUNG_USER"`
	Pass                        string `yaml:"pass" envconfig:"ECO_SYSTEM_SAMSUNG_PASS"`
	ControlPlaneLabelListOption metav1.ListOptions
	OdfLabelListOption          metav1.ListOptions
	SamsungPpLabelListOption    metav1.ListOptions
	SamsungCnfLabelListOption   metav1.ListOptions
}

// NewSamsungConfig returns instance of SamsungConfig config type.
func NewSamsungConfig() *SamsungConfig {
	log.Print("Creating new SamsungConfig struct")

	var samsungConf SamsungConfig
	samsungConf.SystemTestsConfig = systemtestsconfig.NewSystemTestsConfig()

	_, filename, _, _ := runtime.Caller(0)
	baseDir := filepath.Dir(filename)
	confFile := filepath.Join(baseDir, PathToDefaultSamsungParamsFile)
	err := readFile(&samsungConf, confFile)

	if err != nil {
		log.Printf("Error to read config file %s", confFile)

		return nil
	}

	err = readEnv(&samsungConf)

	if err != nil {
		log.Print("Error to read environment variables")

		return nil
	}

	return &samsungConf
}

func readFile(samsungConfig *SamsungConfig, cfgFile string) error {
	openedCfgFile, err := os.Open(cfgFile)
	if err != nil {
		return err
	}

	defer func() {
		_ = openedCfgFile.Close()
	}()

	decoder := yaml.NewDecoder(openedCfgFile)
	err = decoder.Decode(&samsungConfig)

	if err != nil {
		return err
	}

	return nil
}

func readEnv(samsungConfig *SamsungConfig) error {
	err := envconfig.Process("", samsungConfig)
	if err != nil {
		return err
	}

	samsungConfig.OdfLabel = fmt.Sprintf("%s/%s", samsungConfig.KubernetesRolePrefix, samsungConfig.OdfLabel)
	samsungConfig.SamsungPpLabel = fmt.Sprintf("%s/%s", samsungConfig.KubernetesRolePrefix, samsungConfig.SamsungPpLabel)
	samsungConfig.SamsungCnfLabel =
		fmt.Sprintf("%s/%s", samsungConfig.KubernetesRolePrefix, samsungConfig.SamsungCnfLabel)
	samsungConfig.ControlPlaneLabelListOption = metav1.ListOptions{LabelSelector: samsungConfig.ControlPlaneLabel}
	samsungConfig.OdfLabelListOption = metav1.ListOptions{LabelSelector: samsungConfig.OdfLabel}
	samsungConfig.SamsungPpLabelListOption = metav1.ListOptions{LabelSelector: samsungConfig.SamsungPpLabel}
	samsungConfig.SamsungCnfLabelListOption = metav1.ListOptions{LabelSelector: samsungConfig.SamsungCnfLabel}

	return nil
}
