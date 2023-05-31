package operator_test

import (
	"testing"

	"runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift-kni/eco-gotests/pkg/clients"
	"github.com/openshift-kni/eco-gotests/tests/assisted/ztp/internal/meets"
	. "github.com/openshift-kni/eco-gotests/tests/assisted/ztp/internal/ztpinittools"
	"github.com/openshift-kni/eco-gotests/tests/assisted/ztp/operator/internal/tsparams"
	_ "github.com/openshift-kni/eco-gotests/tests/assisted/ztp/operator/tests"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
	"github.com/openshift-kni/eco-gotests/tests/internal/reporter"
)

var _, currentFile, _, _ = runtime.Caller(0)

func TestOperator(t *testing.T) {
	_, reporterConfig := GinkgoConfiguration()
	reporterConfig.JUnitReport = ZTPConfig.GetJunitReportPath(currentFile)

	RegisterFailHandler(Fail)
	RunSpecs(t, "Operator Suite", Label(tsparams.Labels...), reporterConfig)
}

var _ = BeforeSuite(func() {
	By("Check that assisted is running")
	operandRunning, msg := meets.HubInfrastructureOperandRunningRequirement()
	if !operandRunning {
		Skip(msg)
	}
})

var _ = ReportAfterSuite("", func(report Report) {
	polarion.CreateReport(
		report, ZTPConfig.GetPolarionReportPath(currentFile), ZTPConfig.PolarionTCPrefix)
})

var _ = JustAfterEach(func() {
	reporter.ReportIfFailed(
		CurrentSpecReport(),
		currentFile,
		tsparams.ReporterNamespacesToDump,
		tsparams.ReporterCRDsToDump,
		clients.SetScheme)
})