package ztp

import (
	"fmt"
	"path"
	"runtime"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/clients"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/namespace"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/reportxml"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/ran/gitopsztp/internal/gitdetails"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/ran/gitopsztp/internal/tsparams"
	_ "github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/ran/gitopsztp/tests"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/ran/internal/rancluster"
	. "github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/ran/internal/raninittools"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/internal/reporter"
)

var _, currentFile, _, _ = runtime.Caller(0)

func TestZtp(t *testing.T) {
	_, reporterConfig := GinkgoConfiguration()
	reporterConfig.JUnitReport = RANConfig.GetJunitReportPath(currentFile)

	RegisterFailHandler(Fail)
	RunSpecs(t, "RAN ZTP Suite", Label(tsparams.Labels...), reporterConfig)
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
	By("deleting and recreating test namespace to ensure blank state")
	for _, client := range []*clients.Settings{HubAPIClient, Spoke1APIClient} {
		err := namespace.NewBuilder(client, tsparams.TestNamespace).DeleteAndWait(5 * time.Minute)
		Expect(err).ToNot(HaveOccurred(), "Failed to delete ZTP test namespace")

		_, err = namespace.NewBuilder(client, tsparams.TestNamespace).Create()
		Expect(err).ToNot(HaveOccurred(), "Failed to create ZTP test namespace")
	}
})

var _ = AfterSuite(func() {
	By("deleting test namespace")
	for _, client := range []*clients.Settings{HubAPIClient, Spoke1APIClient} {
		err := namespace.NewBuilder(client, tsparams.TestNamespace).DeleteAndWait(5 * time.Minute)
		Expect(err).ToNot(HaveOccurred(), "Failed to delete ZTP test namespace")
	}
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
