package upgrade

import (
	"runtime"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	. "github.com/openshift-kni/eco-gotests/tests/accel/internal/accelinittools"
	"github.com/openshift-kni/eco-gotests/tests/accel/upgrade/internal/upgradeparams"
	_ "github.com/openshift-kni/eco-gotests/tests/accel/upgrade/tests"
	"github.com/openshift-kni/eco-gotests/tests/internal/reporter"
)

var (
	_, currentFile, _, _ = runtime.Caller(0)
	testNS               = namespace.NewBuilder(HubAPIClient, upgradeparams.TestNamespaceName)
)

func TestUpgrade(t *testing.T) {
	_, reporterConfig := GinkgoConfiguration()
	reporterConfig.JUnitReport = AccelConfig.GetJunitReportPath(currentFile)

	RegisterFailHandler(Fail)
	RunSpecs(t, "Acceleration upgrade test", Label(upgradeparams.Labels...), reporterConfig)
}

var _ = AfterSuite(func() {
	By("Deleting test namespace")
	err := testNS.DeleteAndWait(5 * time.Minute)
	Expect(err).ToNot(HaveOccurred(), "error to delete test namespace")
})

var _ = JustAfterEach(func() {
	reporter.ReportIfFailed(
		CurrentSpecReport(), currentFile, upgradeparams.ReporterNamespacesToDump,
		upgradeparams.ReporterCRDsToDump, clients.SetScheme)
})

var _ = ReportAfterSuite("", func(report Report) {
	reportxml.Create(report, AccelConfig.GetReportPath(), AccelConfig.TCPrefix)
})
