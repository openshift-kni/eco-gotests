package diskencryption

import (
	"runtime"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	_ "github.com/openshift-kni/eco-gotests/tests/cnf/ran/diskencryption/tests"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/diskencryption/tsparams"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/raninittools"
)

var _, currentFile, _, _ = runtime.Caller(0)

func TestTPM2(t *testing.T) {
	_, reporterConfig := GinkgoConfiguration()
	reporterConfig.JUnitReport = RANConfig.GetJunitReportPath(currentFile)

	RegisterFailHandler(Fail)

	RunSpecs(t, "RAN tpm2 tests", Label(tsparams.Labels...), reporterConfig)
}

var _ = BeforeSuite(func() {
})

var _ = AfterSuite(func() {
})
