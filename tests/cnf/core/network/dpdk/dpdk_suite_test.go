package dpdk

import (
	"runtime"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/mco"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/nto" //nolint:misspell
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/dpdk/internal/dpdkenv"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/dpdk/internal/tsparams"
	_ "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/dpdk/tests"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netinittools"
	"github.com/openshift-kni/eco-gotests/tests/internal/params"
	"github.com/openshift-kni/eco-gotests/tests/internal/reporter"
)

var (
	_, currentFile, _, _ = runtime.Caller(0)
	testNS               = namespace.NewBuilder(APIClient, tsparams.TestNamespaceName)
	perfProfileName      = "automationdpdk"
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

	By("Verifying if dpdk tests can be executed on given cluster")
	err = dpdkenv.DoesClusterSupportDpdkTests(APIClient, NetConfig, 26, 100)
	Expect(err).ToNot(HaveOccurred(), "Cluster doesn't support dpdk test cases")

	By("Deploying PerformanceProfile is it's not installed")
	err = dpdkenv.DeployPerformanceProfile(
		APIClient,
		NetConfig,
		perfProfileName,
		"1,3,5,7,9,11,13,15,17,19,21,23,25",
		"0,2,4,6,8,10,12,14,16,18,20",
		24)
	Expect(err).ToNot(HaveOccurred(), "Fail to deploy PerformanceProfile")
})

var _ = AfterSuite(func() {
	By("Deleting test namespace")
	err := testNS.DeleteAndWait(tsparams.WaitTimeout)
	Expect(err).ToNot(HaveOccurred(), "Fail to delete test namespace")

	By("Removing performanceProfile")
	perfProfile, err := nto.Pull(APIClient, perfProfileName)
	Expect(err).ToNot(HaveOccurred(), "Fail to pull test PerformanceProfile")
	_, err = perfProfile.Delete()
	Expect(err).ToNot(HaveOccurred(), "Fail to delete PerformanceProfile")

	By("Waiting until cluster is stable")
	mcp, err := mco.Pull(APIClient, NetConfig.CnfMcpLabel)
	Expect(err).ToNot(HaveOccurred(), "Fail to pull MCP ")
	err = mcp.WaitToBeStableFor(time.Minute, tsparams.MCOWaitTimeout)
	Expect(err).ToNot(HaveOccurred(), "Fail to wait until cluster is stable")
})

var _ = JustAfterEach(func() {
	reporter.ReportIfFailed(
		CurrentSpecReport(), currentFile, tsparams.ReporterNamespacesToDump, tsparams.ReporterCRDsToDump, clients.SetScheme)
})

var _ = ReportAfterSuite("", func(report Report) {
	reportxml.Create(report, NetConfig.GetReportPath(), NetConfig.TCPrefix)
})
