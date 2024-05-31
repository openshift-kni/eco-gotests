package spoke_test

import (
	"testing"

	"runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift-kni/eco-gotests/pkg/clients"
	. "github.com/openshift-kni/eco-gotests/tests/assisted/ztp/internal/ztpinittools"
	"github.com/openshift-kni/eco-gotests/tests/assisted/ztp/spoke/internal/tsparams"
	_ "github.com/openshift-kni/eco-gotests/tests/assisted/ztp/spoke/tests"
	"github.com/openshift-kni/eco-gotests/tests/internal/reporter"
	"github.com/openshift-kni/eco-gotests/tests/internal/reportxml"
)

var _, currentFile, _, _ = runtime.Caller(0)

func TestSpoke(t *testing.T) {
	_, reporterConfig := GinkgoConfiguration()
	reporterConfig.JUnitReport = ZTPConfig.GetJunitReportPath(currentFile)

	RegisterFailHandler(Fail)
	RunSpecs(t, "Spoke Suite", Label(tsparams.Labels...), reporterConfig)
}

var _ = ReportAfterSuite("", func(report Report) {
	reportxml.Create(
		report, ZTPConfig.GetReportPath(currentFile), ZTPConfig.TCPrefix)
})

var _ = JustAfterEach(func() {
	reporter.ReportIfFailed(
		CurrentSpecReport(),
		currentFile,
		tsparams.ReporterNamespacesToDump,
		tsparams.ReporterCRDsToDump,
		clients.SetScheme)
})
