package containernshidetest

import (
	"runtime"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/containernshide/internal/tsparams"
	_ "github.com/openshift-kni/eco-gotests/tests/cnf/ran/containernshide/tests"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/raninittools"
)

var _, currentFile, _, _ = runtime.Caller(0)

func TestContainerNsHide(t *testing.T) {
	_, reporterConfig := GinkgoConfiguration()
	reporterConfig.JUnitReport = raninittools.RANConfig.GetJunitReportPath(currentFile)

	RegisterFailHandler(Fail)
	RunSpecs(t, "Container Mount Namespace Hiding", Label(tsparams.Labels...), reporterConfig)
}
