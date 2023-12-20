package upgrade

import (
	"runtime"
	"testing"

	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/1upgrade/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
	"github.com/openshift-kni/eco-gotests/tests/internal/reporter"

	_ "github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/1upgrade/tests"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/openshift-kni/eco-gotests/tests/internal/inittools"
)

var _, currentFile, _, _ = runtime.Caller(0)

func TestUpgrade(tt *testing.T) {
	_, reporterConfig := GinkgoConfiguration()
	reporterConfig.JUnitReport = GeneralConfig.GetJunitReportPath(currentFile)

	RegisterFailHandler(Fail)
	RunSpecs(tt, "1upgrade", Label(tsparams.Labels...), reporterConfig)
}

var _ = ReportAfterSuite("1upgrade", func(report Report) {
	polarion.CreateReport(
		report, GeneralConfig.GetPolarionReportPath(), GeneralConfig.PolarionTCPrefix)
})

var _ = JustAfterEach(func() {
	reporter.ReportIfFailed(
		CurrentSpecReport(), currentFile, map[string]string{"openshift-kmm": "op"}, nil, clients.SetScheme)
})