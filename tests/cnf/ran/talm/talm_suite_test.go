package talm

import (
	"fmt"
	"path"
	"runtime"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/raninittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/talm/internal/helper"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/talm/internal/tsparams"
	_ "github.com/openshift-kni/eco-gotests/tests/cnf/ran/talm/tests"
	"github.com/openshift-kni/eco-gotests/tests/internal/reporter"
)

var _, currentFile, _, _ = runtime.Caller(0)

func TestTalm(t *testing.T) {
	_, reporterConfig := GinkgoConfiguration()
	reporterConfig.JUnitReport = RANConfig.GetJunitReportPath(currentFile)

	RegisterFailHandler(Fail)
	RunSpecs(t, "TALM Suite", Label(tsparams.Labels...), reporterConfig)
}

var _ = BeforeSuite(func() {
	err := helper.VerifyTalmIsInstalled()
	Expect(err).ToNot(HaveOccurred(), "Failed to verify that TALM is installed")

	By("deleting and recreating TALM test namespace to ensure a blank slate")
	err = helper.DeleteTalmTestNamespace()
	Expect(err).ToNot(HaveOccurred(), "Failed to delete TALM test namespace")
	err = helper.CreateTalmTestNamespace()
	Expect(err).ToNot(HaveOccurred(), "Failed to create TALM test namespace")
})

var _ = AfterSuite(func() {
	// Deleting the namespace after the suite finishes ensures all the CGUs created are deleted
	err := helper.DeleteTalmTestNamespace()
	Expect(err).ToNot(HaveOccurred(), "Failed to delete TALM test namespace")
})

var _ = JustAfterEach(func() {
	var (
		currentDir, currentFilename = path.Split(currentFile)
		hubReportPath               = fmt.Sprintf("%shub_%s", currentDir, currentFilename)
		spoke2ReportPath            = fmt.Sprintf("%sspoke2_%s", currentDir, currentFilename)
		report                      = CurrentSpecReport()
	)

	reporter.ReportIfFailed(
		report, currentFile, tsparams.ReporterSpokeNamespacesToDump, tsparams.ReporterSpokeCRsToDump, clients.SetScheme)

	if HubAPIClient != nil {
		reporter.ReportIfFailedOnCluster(
			RANConfig.HubKubeconfig,
			report,
			hubReportPath,
			tsparams.ReporterHubNamespacesToDump,
			tsparams.ReporterHubCRsToDump,
			clients.SetScheme)
	}

	if Spoke2APIClient != nil {
		reporter.ReportIfFailedOnCluster(
			RANConfig.Spoke2Kubeconfig,
			report,
			spoke2ReportPath,
			tsparams.ReporterSpokeNamespacesToDump,
			tsparams.ReporterSpokeCRsToDump,
			clients.SetScheme)
	}
})

var _ = ReportAfterSuite("", func(report Report) {
	reportxml.Create(report, RANConfig.GetReportPath(), RANConfig.TCPrefix)
})
