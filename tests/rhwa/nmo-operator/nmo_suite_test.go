package nmo

import (
	"runtime"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
	"github.com/openshift-kni/eco-gotests/tests/internal/reporter"
	. "github.com/openshift-kni/eco-gotests/tests/rhwa/internal/rhwainittools"
	"github.com/openshift-kni/eco-gotests/tests/rhwa/nmo-operator/internal/nmoparams"
	_ "github.com/openshift-kni/eco-gotests/tests/rhwa/nmo-operator/tests"
)

var _, currentFile, _, _ = runtime.Caller(0)

func TestNMO(t *testing.T) {
	_, reporterConfig := GinkgoConfiguration()
	reporterConfig.JUnitReport = RHWAConfig.GetJunitReportPath(currentFile)

	RegisterFailHandler(Fail)
	RunSpecs(t, "NMO", Label(nmoparams.Labels...), reporterConfig)
}

var _ = JustAfterEach(func() {
	reporter.ReportIfFailed(
		CurrentSpecReport(), currentFile, nmoparams.ReporterNamespacesToDump,
		nmoparams.ReporterCRDsToDump, clients.SetScheme)
})

var _ = ReportAfterSuite("", func(report Report) {
	polarion.CreateReport(
		report, RHWAConfig.GetPolarionReportPath(currentFile), RHWAConfig.PolarionTCPrefix)
})
