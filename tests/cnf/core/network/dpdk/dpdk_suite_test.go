package dpdk

import (
	"runtime"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	_ "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/dpdk/internal/tests"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/dpdk/internal/tsparams"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netinittools"
	"github.com/openshift-kni/eco-gotests/tests/internal/params"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
	"github.com/openshift-kni/eco-gotests/tests/internal/reporter"
)

var (
	_, currentFile, _, _ = runtime.Caller(0)
	testNS               = namespace.NewBuilder(APIClient, tsparams.TestNamespaceName)
)

func TestLB(t *testing.T) {
	_, reporterConfig := GinkgoConfiguration()
	reporterConfig.JUnitReport = NetConfig.GetJunitReportPath(currentFile)

	RegisterFailHandler(Fail)
	RunSpecs(t, "dpdk", Label(tsparams.Labels...), reporterConfig)
}

var _ = BeforeSuite(func() {
	By("Creating privileged test namespace")
	for key, value := range params.PrivilegedNSLabels {
		testNS.WithLabel(key, value)
	}

	_, err := testNS.Create()
	Expect(err).ToNot(HaveOccurred(), "error to create test namespace")
})

var _ = AfterSuite(func() {
	By("Deleting test namespace")
	err := testNS.Delete()
	Expect(err).ToNot(HaveOccurred(), "error to delete test namespace")
})

var _ = JustAfterEach(func() {
	reporter.ReportIfFailed(
		CurrentSpecReport(), currentFile, tsparams.ReporterNamespacesToDump, tsparams.ReporterCRDsToDump, clients.SetScheme)
})

var _ = ReportAfterSuite("", func(report Report) {
	polarion.CreateReport(report, NetConfig.GetPolarionReportPath(currentFile), NetConfig.PolarionTCPrefix)
})
