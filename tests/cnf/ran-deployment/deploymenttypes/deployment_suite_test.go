package deployment

import (
	"fmt"
	"path"
	"runtime"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran-deployment/deploymenttypes/internal/gitdetails"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran-deployment/deploymenttypes/internal/tsparams"
	_ "github.com/openshift-kni/eco-gotests/tests/cnf/ran-deployment/deploymenttypes/tests"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran-deployment/internal/rancluster"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/ran-deployment/internal/raninittools"
	"github.com/openshift-kni/eco-gotests/tests/internal/reporter"
)

var _, currentFile, _, _ = runtime.Caller(0)

func TestDeployment(t *testing.T) {
	_, reporterConfig := GinkgoConfiguration()
	reporterConfig.JUnitReport = RANConfig.GetJunitReportPath(currentFile)

	RegisterFailHandler(Fail)
	RunSpecs(t, "RAN Deployment Types Suite", Label(tsparams.Labels...), reporterConfig)
}

var _ = BeforeSuite(func() {
	By("checking that the required clusters are present")
	if !rancluster.AreClustersPresent([]*clients.Settings{HubAPIClient, Spoke1APIClient}) {
		Skip("not all of the required clusters are present")
	}

	err := gitdetails.GetArgoCdAppGitDetails()
	Expect(err).ToNot(HaveOccurred(), "Failed to get current data from ArgoCD")
})

var _ = BeforeEach(func() {

})

var _ = AfterSuite(func() {

})

var _ = JustAfterEach(func() {
	var (
		currentDir, currentFilename = path.Split(currentFile)
		hubReportPath               = fmt.Sprintf("%shub_%s", currentDir, currentFilename)
		report                      = CurrentSpecReport()
	)

	reporter.ReportIfFailed(
		report, currentFile, tsparams.ReporterSpokeNamespacesToDump, tsparams.ReporterSpokeCRsToDump)

	if HubAPIClient != nil {
		reporter.ReportIfFailedOnCluster(
			RANConfig.HubKubeconfig,
			report,
			hubReportPath,
			tsparams.ReporterHubNamespacesToDump,
			tsparams.ReporterHubCRsToDump)
	}
})

var _ = ReportAfterSuite("", func(report Report) {
	reportxml.Create(report, RANConfig.GetReportPath(), RANConfig.TCPrefix)
})
