package dast

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-gotests/tests/rhwa/internal/rhwaparams"
	"gopkg.in/yaml.v3"
)

func prepareRapidastConfig(destinationPath string) error {

	workingDir, err := os.Getwd()
	Expect(err).ToNot(HaveOccurred(), err)

	templateDir := filepath.Join(workingDir, rhwaparams.RapidastTemplateFolder)
	rapidastConfigFile := filepath.Join(templateDir, rhwaparams.RapidastTemplateFile)

	f, err := os.Open(rapidastConfigFile)
	Expect(err).NotTo((HaveOccurred()))
	defer f.Close()

	content, err := io.ReadAll(f)
	Expect(err).NotTo((HaveOccurred()))

	yamlData := make(map[string]interface{})
	err = yaml.Unmarshal(content, &yamlData)
	Expect(err).NotTo((HaveOccurred()))

	sc := yamlData["scanners"].(map[string]interface{})
	genericTrivy := sc["generic_trivy"].(map[string]interface{})
	genericTrivy["inline"] = fmt.Sprintf("trivy k8s --kubeconfig=/home/rapidast/.kube/config -n %s pod --severity=HIGH,CRITICAL --scanners=misconfig --report all --format json",
		rhwaparams.RhwaOperatorNs)

	modifiedYaml, err := yaml.Marshal(yamlData)
	Expect(err).NotTo((HaveOccurred()))

	filePath := filepath.Join(destinationPath, rhwaparams.RapidastTemplateFile)
	rapidastConfig, err := os.Create(filePath)
	Expect(err).NotTo((HaveOccurred()))
	defer rapidastConfig.Close()

	_, err = rapidastConfig.Write(modifiedYaml)
	Expect(err).NotTo((HaveOccurred()))

	err = rapidastConfig.Close()
	Expect(err).NotTo((HaveOccurred()))

	return nil
}
