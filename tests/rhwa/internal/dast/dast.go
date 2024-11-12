package dast

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/openshift-kni/eco-gotests/tests/rhwa/internal/rhwaparams"
	"gopkg.in/yaml.v3"
)

// PrepareRapidastConfig builds the trivy command and updates the rapidast configuration file.
func PrepareRapidastConfig(destinationPath string) error {
	workingDir, err := os.Getwd()
	if err != nil {
		return err
	}

	templateDir := filepath.Join(workingDir, rhwaparams.RapidastTemplateFolder)
	rapidastConfigFile := filepath.Join(templateDir, rhwaparams.RapidastTemplateFile)

	file, err := os.Open(rapidastConfigFile)
	if err != nil {
		return err
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	yamlData := make(map[string]interface{})

	err = yaml.Unmarshal(content, &yamlData)
	if err != nil {
		return err
	}

	sc, successful := yamlData["scanners"].(map[string]interface{})
	if sc == nil || !successful {
		return fmt.Errorf("yamlData is missing key \"scanners\"")
	}

	genericTrivy, successful := sc["generic_trivy"].(map[string]interface{})
	if genericTrivy == nil || !successful {
		return fmt.Errorf("yamlData is missing key \"generic_trivy\"")
	}

	genericTrivy["inline"] = fmt.Sprintf("trivy k8s --kubeconfig=/home/rapidast/.kube/config "+
		"-n %s pod "+
		"--severity=HIGH,CRITICAL "+
		"--scanners=misconfig "+
		"--report all "+
		"--format json",
		rhwaparams.RhwaOperatorNs)

	modifiedYaml, err := yaml.Marshal(yamlData)
	if err != nil {
		return err
	}

	filePath := filepath.Join(destinationPath, rhwaparams.RapidastTemplateFile)

	rapidastConfig, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer rapidastConfig.Close()

	_, err = rapidastConfig.Write(modifiedYaml)
	if err != nil {
		return err
	}

	return nil
}
