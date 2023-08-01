package ran_du_system_test

import (
	"runtime"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	. "github.com/openshift-kni/eco-gotests/tests/internal/inittools"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
	"github.com/openshift-kni/eco-gotests/tests/internal/reporter"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/ran-du/internal/randuparams"
	_ "github.com/openshift-kni/eco-gotests/tests/system-tests/ran-du/tests"
)

var _, currentFile, _, _ = runtime.Caller(0)

func TestRanDu(t *testing.T) {
	_, reporterConfig := GinkgoConfiguration()
	reporterConfig.JUnitReport = GeneralConfig.GetJunitReportPath(currentFile)

	RegisterFailHandler(Fail)
	RunSpecs(t, "RanDU SystemTests Suite", Label(randuparams.Labels...), reporterConfig)
}

var _ = JustAfterEach(func() {
	reporter.ReportIfFailed(
		CurrentSpecReport(), currentFile, randuparams.ReporterNamespacesToDump,
		randuparams.ReporterCRDsToDump, clients.SetScheme)
})

var _ = ReportAfterSuite("", func(report Report) {
	polarion.CreateReport(
		report, GeneralConfig.GetPolarionReportPath(), GeneralConfig.PolarionTCPrefix)
})
