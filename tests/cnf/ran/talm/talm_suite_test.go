package talm

import (
	"runtime"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/raninittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/talm/internal/helper"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/talm/internal/tsparams"
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
	err := helper.InitializeVariables()
	Expect(err).ToNot(HaveOccurred())

	// Make sure TALM is present
	err = helper.VerifyTalmIsInstalled()
	Expect(err).ToNot(HaveOccurred())

	// Delete the namespace before creating it to ensure it is in a consistent blank state
	err = helper.DeleteTalmTestNamespace()
	Expect(err).ToNot(HaveOccurred())
	err = helper.CreateTalmTestNamespace()
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	// Deleting the namespace after the suite finishes ensures all the CGUs created are deleted
	err := helper.DeleteTalmTestNamespace()
	Expect(err).ToNot(HaveOccurred())
})

var _ = ReportAfterEach(func(report SpecReport) {
	reporter.ReportIfFailed(
		report, currentFile, tsparams.ReporterNamespacesToDump, tsparams.ReporterCRDsToDump, clients.SetScheme)
})

var _ = ReportAfterSuite("", func(report Report) {
	reportxml.Create(report, RANConfig.GetReportPath(), RANConfig.TCPrefix)
})
