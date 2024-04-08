package upgrade

import (
	"runtime"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/nfd/2upgrade/internal/tsparams"
	_ "github.com/openshift-kni/eco-gotests/tests/hw-accel/nfd/2upgrade/tests"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/nfd/nfdparams"
	. "github.com/openshift-kni/eco-gotests/tests/internal/inittools"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
	"github.com/openshift-kni/eco-gotests/tests/internal/reporter"
)

var _, currentFile, _, _ = runtime.Caller(0)

func TestUpgrade(tt *testing.T) {
	_, reporterConfig := GinkgoConfiguration()
	reporterConfig.JUnitReport = GeneralConfig.GetJunitReportPath(currentFile)

	RegisterFailHandler(Fail)
	RunSpecs(tt, tsparams.NfdUpgradeLabel, Label(nfdparams.Labels...), reporterConfig)
}

var _ = ReportAfterSuite(tsparams.NfdUpgradeLabel, func(report Report) {
	polarion.CreateReport(
		report, GeneralConfig.GetPolarionReportPath(), GeneralConfig.PolarionTCPrefix)
})

var _ = JustAfterEach(func() {
	reporter.ReportIfFailed(
		CurrentSpecReport(), currentFile, map[string]string{nfdparams.NFDNamespace: "op"}, nil, clients.SetScheme)
})
