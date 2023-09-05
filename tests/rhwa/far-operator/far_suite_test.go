package far

import (
	"runtime"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
	"github.com/openshift-kni/eco-gotests/tests/internal/reporter"
	"github.com/openshift-kni/eco-gotests/tests/rhwa/far-operator/internal/farparams"
	_ "github.com/openshift-kni/eco-gotests/tests/rhwa/far-operator/tests"
	. "github.com/openshift-kni/eco-gotests/tests/rhwa/internal/rhwainittools"
)

var _, currentFile, _, _ = runtime.Caller(0)

func TestFAR(t *testing.T) {
	_, reporterConfig := GinkgoConfiguration()
	reporterConfig.JUnitReport = RHWAConfig.GetJunitReportPath(currentFile)

	RegisterFailHandler(Fail)
	RunSpecs(t, "FAR", Label(farparams.Labels...), reporterConfig)
}

var _ = JustAfterEach(func() {
	reporter.ReportIfFailed(
		CurrentSpecReport(), currentFile, farparams.ReporterNamespacesToDump,
		farparams.ReporterCRDsToDump, clients.SetScheme)
})

var _ = ReportAfterSuite("", func(report Report) {
	polarion.CreateReport(
		report, RHWAConfig.GetPolarionReportPath(), RHWAConfig.PolarionTCPrefix)
})
