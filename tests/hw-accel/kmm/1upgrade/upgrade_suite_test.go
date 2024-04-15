package upgrade

import (
	"runtime"
	"testing"

	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/1upgrade/internal/tsparams"
	_ "github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/1upgrade/tests"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/internal/kmmparams"
	"github.com/openshift-kni/eco-gotests/tests/internal/reporter"

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
	reportxml.Create(report, GeneralConfig.GetReportPath(), GeneralConfig.TCPrefix)
})

var _ = JustAfterEach(func() {
	reporter.ReportIfFailed(
		CurrentSpecReport(), currentFile, map[string]string{kmmparams.KmmOperatorNamespace: "op"}, nil, clients.SetScheme)
})
