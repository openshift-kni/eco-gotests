package metallb

import (
	"runtime"
	"testing"

	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"

	"github.com/onsi/ginkgo/v2/types"
	"github.com/openshift-kni/eco-gotests/pkg/clients"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netinittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/tsparams"
	_ "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/tests"
	"github.com/openshift-kni/eco-gotests/tests/internal/reporter"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _, currentFile, _, _ = runtime.Caller(0)

func TestLB(t *testing.T) {
	_, reporterConfig := GinkgoConfiguration()
	reporterConfig.JUnitReport = NetConfig.GetJunitReportPath(currentFile)

	RegisterFailHandler(Fail)
	RunSpecs(t, "MetalLB", Label(tsparams.Labels...), reporterConfig)
}

var _ = BeforeSuite(func() {
	By("Create test namespace")
	_, err := tsparams.TestNS.Create()
	Expect(err).ToNot(HaveOccurred(), "error to create test namespace")
})

var _ = AfterSuite(func() {
	By("Delete test namespace")
	err := tsparams.TestNS.Delete()
	Expect(err).ToNot(HaveOccurred(), "error to delete test namespace")
})

var _ = ReportAfterEach(func(report types.SpecReport) {
	reporter.ReportIfFailed(
		report, currentFile, tsparams.ReporterNamespacesToDump, tsparams.ReporterCRDsToDump, clients.SetScheme)
})

var _ = ReportAfterSuite("", func(report Report) {
	polarion.CreateReport(report, NetConfig.GetPolarionReportPath(currentFile), NetConfig.PolarionTCPrefix)
})
