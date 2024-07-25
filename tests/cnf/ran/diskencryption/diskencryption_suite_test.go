package diskencryption

import (
	"runtime"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/diskencryption/internal/helper"
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
	isTTYConsole, err := helper.IsTTYConsole()
	Expect(err).ToNot(HaveOccurred(), "error checkking kernel command line for tty console")
	Expect(isTTYConsole).To(BeTrue(), "the TTY options should be configured on the kernel"+
		" boot line (nomodeset console=tty0 console=ttyS0,115200n8)")
})
