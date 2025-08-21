package bfd

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/namespace"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/reportxml"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/core/network/bfd/internal/tsparams"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/core/network/internal/netenv"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/core/network/internal/netparam"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/internal/params"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/internal/reporter"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	_ "github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/core/network/bfd/tests"
	. "github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/core/network/internal/netinittools"
)

var (
	_, currentFile, _, _ = runtime.Caller(0)
	testNS               = namespace.NewBuilder(APIClient, tsparams.TestNamespace)
)

func TestBfd(t *testing.T) {
	_, reporterConfig := GinkgoConfiguration()
	reporterConfig.JUnitReport = NetConfig.GetJunitReportPath(currentFile)

	RegisterFailHandler(Fail)
	RunSpecs(t, "BFD", Label(tsparams.Labels...), reporterConfig)
}

var _ = BeforeSuite(func() {
	By("Checking whether we have more than 1 node in the cluster")
	err := netenv.DoesClusterHasEnoughNodes(APIClient, NetConfig, 1, 2)
	if err != nil {
		Skip(fmt.Sprintf("Cluster does not have sufficient nodes required for the test. Error: %s", err.Error()))
	}
	By("Creating Test Namespace with privileges")
	_, err = testNS.WithMultipleLabels(params.PrivilegedNSLabels).Create()
	Expect(err).ToNot(HaveOccurred(), "Failed to create Test Namespace")
})

var _ = AfterSuite(func() {
	By("Deleting the Test Namespace")
	err := testNS.DeleteAndWait(netparam.DefaultTimeout)
	Expect(err).ToNot(HaveOccurred(), "Failed to delete Test Namespace")
})

var _ = JustAfterEach(func() {
	reporter.ReportIfFailed(
		CurrentSpecReport(), currentFile, tsparams.ReporterNamespacesToDump, tsparams.ReporterCRDsToDump)
})

var _ = ReportAfterSuite("", func(report Report) {
	reportxml.Create(report, NetConfig.GetReportPath(), NetConfig.TCPrefix)
})
