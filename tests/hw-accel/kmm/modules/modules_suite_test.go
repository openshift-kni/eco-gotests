package modules

import (
	"runtime"
	"testing"

	"github.com/openshift-kni/eco-gotests/pkg/clients"
	"github.com/openshift-kni/eco-gotests/tests/internal/reporter"

	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/modules/internal/tsparams"
	_ "github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/modules/tests"
	. "github.com/openshift-kni/eco-gotests/tests/internal/inittools"
	"github.com/openshift-kni/eco-gotests/tests/internal/reportxml"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _, currentFile, _, _ = runtime.Caller(0)

func TestModules(t *testing.T) {
	_, reporterConfig := GinkgoConfiguration()
	reporterConfig.JUnitReport = GeneralConfig.GetJunitReportPath(currentFile)

	RegisterFailHandler(Fail)
	RunSpecs(t, "KMM", Label(tsparams.Labels...), reporterConfig)
}

var _ = ReportAfterSuite("", func(report Report) {
	reportxml.Create(
		report, GeneralConfig.GetReportPath(currentFile), GeneralConfig.TCPrefix)
})

var _ = JustAfterEach(func() {
	reporter.ReportIfFailed(
		CurrentSpecReport(), currentFile, tsparams.ReporterNamespacesToDump, tsparams.ReporterCRDsToDump, clients.SetScheme)
})
