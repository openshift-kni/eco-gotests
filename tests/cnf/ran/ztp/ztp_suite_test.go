package ztp

import (
	"runtime"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/clients"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/namespace"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/reportxml"
	. "github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/ran/internal/raninittools"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/ran/ztp/internal/helper"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/ran/ztp/internal/tsparams"
	_ "github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/ran/ztp/tests"
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
	err := helper.GetArgoCdAppGitDetails()
	Expect(err).ToNot(HaveOccurred(), "Failed to get current data from ArgoCD")

	By("deleting and recreating ZTP test namespace to ensure a blank state")
	err = namespace.NewBuilder(HubAPIClient, tsparams.TestNamespace).DeleteAndWait(5 * time.Minute)
	Expect(err).ToNot(HaveOccurred(), "Failed to delete ZTP test namespace")

	_, err = namespace.NewBuilder(HubAPIClient, tsparams.TestNamespace).Create()
	Expect(err).ToNot(HaveOccurred(), "Failed to create ZTP test namespace")
})

var _ = AfterSuite(func() {
	err := namespace.NewBuilder(HubAPIClient, tsparams.TestNamespace).DeleteAndWait(5 * time.Minute)
	Expect(err).ToNot(HaveOccurred(), "Failed to delete ZTP test namespace")
})

var _ = ReportAfterEach(func(report SpecReport) {
	reporter.ReportIfFailed(
		report, currentFile, tsparams.ReporterNamespacesToDump, tsparams.ReporterCRDsToDump, clients.SetScheme)
})

var _ = ReportAfterSuite("", func(report Report) {
	reportxml.Create(report, RANConfig.GetReportPath(), RANConfig.TCPrefix)
})
