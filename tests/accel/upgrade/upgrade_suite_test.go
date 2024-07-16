package upgrade

import (
	"runtime"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"github.com/openshift-kni/eco-gotests/tests/accel/upgrade/accelparams"
	. "github.com/openshift-kni/eco-gotests/tests/accel/upgrade/upgradeinittools"
	"github.com/openshift-kni/eco-gotests/tests/internal/reporter"
)

var (
	_, currentFile, _, _ = runtime.Caller(0)
	testNS               = namespace.NewBuilder(HubAPIClient, accelparams.TestNamespaceName)
)

func TestUpgrade(t *testing.T) {
	_, reporterConfig := GinkgoConfiguration()
	reporterConfig.JUnitReport = GeneralConfig.GetJunitReportPath(currentFile)

	RegisterFailHandler(Fail)
	RunSpecs(t, "Acceleration upgrade test", Label(accelparams.TeamLabel), reporterConfig)
}

var _ = AfterSuite(func() {
	By("Deleting test namespace")
	err := testNS.DeleteAndWait(accelparams.DefaultTimeout)
	Expect(err).ToNot(HaveOccurred(), "error to delete test namespace")
})

var _ = JustAfterEach(func() {
	reporter.ReportIfFailed(
		CurrentSpecReport(), currentFile, accelparams.ReporterNamespacesToDump,
		accelparams.ReporterCRDsToDump, clients.SetScheme)
})

var _ = ReportAfterSuite("", func(report Report) {
	reportxml.Create(report, GeneralConfig.GetReportPath(), GeneralConfig.TCPrefix)
})
