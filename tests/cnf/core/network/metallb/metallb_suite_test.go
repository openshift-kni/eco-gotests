package metallb

import (
	"fmt"
	"runtime"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-gotests/pkg/clients"
	"github.com/openshift-kni/eco-gotests/pkg/namespace"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netinittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/metallbenv"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/tsparams"
	_ "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/tests"
	"github.com/openshift-kni/eco-gotests/tests/internal/cluster"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
	"github.com/openshift-kni/eco-gotests/tests/internal/reporter"
)

const (
	requiredCPNodeNumber     = 1
	requiredWorkerNodeNumber = 2
)

var (
	_, currentFile, _, _ = runtime.Caller(0)
	testNS               = namespace.NewBuilder(APIClient, tsparams.TestNamespaceName)
)

func TestLB(t *testing.T) {
	_, reporterConfig := GinkgoConfiguration()
	reporterConfig.JUnitReport = NetConfig.GetJunitReportPath(currentFile)

	RegisterFailHandler(Fail)
	RunSpecs(t, "MetalLB", Label(tsparams.Labels...), reporterConfig)
}

var _ = BeforeSuite(func() {
	By("Create test namespace")
	_, err := testNS.Create()
	Expect(err).ToNot(HaveOccurred(), "error to create test namespace")

	By("Verify if metalLb tests can be executed on given cluster")
	err = metallbenv.DoesClusterSupportMetalLbTests(requiredCPNodeNumber, requiredWorkerNodeNumber)

	if err != nil {
		Skip(
			fmt.Sprintf("given cluster is not suitable for MetalLb tests due to the following error %s", err.Error()))
	}

	By("Pulling test images on cluster before running test cases")
	err = cluster.PullTestImageOnNodes(APIClient, NetConfig.WorkerLabel, NetConfig.CnfNetTestContainer, 300)
	Expect(err).ToNot(HaveOccurred(), "Failed to pull test image on nodes")

})

var _ = AfterSuite(func() {
	By("Delete test namespace")
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