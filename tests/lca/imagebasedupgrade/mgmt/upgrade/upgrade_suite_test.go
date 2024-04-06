package upgrade_test

import (
	"encoding/json"
	"fmt"
	"regexp"
	"testing"

	"runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-gotests/tests/internal/cluster"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
	"github.com/openshift-kni/eco-gotests/tests/internal/reporter"
	. "github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/mgmt/internal/mgmtinittools"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/mgmt/upgrade/internal/tsparams"
	_ "github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/mgmt/upgrade/tests"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _, currentFile, _, _ = runtime.Caller(0)

func TestUpgrade(t *testing.T) {
	_, reporterConfig := GinkgoConfiguration()
	reporterConfig.JUnitReport = MGMTConfig.GetJunitReportPath(currentFile)

	RegisterFailHandler(Fail)
	RunSpecs(t, "Upgrade Suite", Label(tsparams.Labels...), reporterConfig)
}

var _ = BeforeSuite(func() {
	if MGMTConfig.SeedImageVersion == "" {
		By("Find node for pulling seed image")
		ibuNodes, err := nodes.List(APIClient)
		Expect(err).NotTo(HaveOccurred(), "error pulling nodes")
		Expect(len(ibuNodes) > 0).To(BeTrue(), "error: no node resources were found")
		seedImageNode := ibuNodes[0]

		By("Pull seed image onto node")
		_, err = cluster.ExecCmdWithStdout(
			APIClient, fmt.Sprintf("sudo podman pull %s", MGMTConfig.SeedImage), metav1.ListOptions{
				FieldSelector: fmt.Sprintf("metadata.name=%s", seedImageNode.Object.Name),
			})
		Expect(err).NotTo(HaveOccurred(), "error executing podman pull on %s", seedImageNode.Object.Name)

		By("Mount seed image on node")
		mountedFilePathOutput, err := cluster.ExecCmdWithStdout(
			APIClient, fmt.Sprintf("sudo podman image mount %s", MGMTConfig.SeedImage), metav1.ListOptions{
				FieldSelector: fmt.Sprintf("metadata.name=%s", seedImageNode.Object.Name),
			})
		Expect(err).NotTo(HaveOccurred(), "error executing podman image mount on %s", seedImageNode.Object.Name)

		mountedFilePath := regexp.MustCompile(`\n`).ReplaceAllString(mountedFilePathOutput[seedImageNode.Object.Name], "")

		By("Read manifest.json from mounted image")
		manifestJSONOutput, err := cluster.ExecCmdWithStdout(
			APIClient, fmt.Sprintf("sudo cat %s/manifest.json", mountedFilePath), metav1.ListOptions{
				FieldSelector: fmt.Sprintf("metadata.name=%s", seedImageNode.Object.Name),
			})
		Expect(err).NotTo(HaveOccurred(), "error reading manifest.json on %s", seedImageNode.Object.Name)

		manifestJSON := manifestJSONOutput[seedImageNode.Object.Name]

		var manifestJSONReader struct {
			OCPVersion string `json:"seed_cluster_ocp_version"`
		}

		err = json.Unmarshal([]byte(manifestJSON), &manifestJSONReader)
		Expect(err).NotTo(HaveOccurred(), "error unmarshalling manifest.json to struct on %s", seedImageNode.Object.Name)

		MGMTConfig.SeedImageVersion = manifestJSONReader.OCPVersion

		By("Unmount seed image")
		_, err = cluster.ExecCmdWithStdout(
			APIClient, fmt.Sprintf("sudo podman image unmount %s", MGMTConfig.SeedImage), metav1.ListOptions{
				FieldSelector: fmt.Sprintf("metadata.name=%s", seedImageNode.Object.Name),
			})
		Expect(err).NotTo(HaveOccurred(), "error executing podman image unmount on %s", seedImageNode.Object.Name)
	}
})

var _ = ReportAfterSuite("", func(report Report) {
	polarion.CreateReport(
		report, MGMTConfig.GetPolarionReportPath(), MGMTConfig.PolarionTCPrefix)
})

var _ = JustAfterEach(func() {
	reporter.ReportIfFailed(
		CurrentSpecReport(),
		currentFile,
		tsparams.ReporterNamespacesToDump,
		tsparams.ReporterCRDsToDump,
		clients.SetScheme)
})
