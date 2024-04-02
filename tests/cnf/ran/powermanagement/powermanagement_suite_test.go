package powermanagement

import (
	"runtime"
	"testing"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/ranparam"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/powermanagement/internal/tsparams"
	_ "github.com/openshift-kni/eco-gotests/tests/cnf/ran/powermanagement/tests"
	. "github.com/openshift-kni/eco-gotests/tests/internal/inittools"
)

var _, currentFile, _, _ = runtime.Caller(0)

func TestPowerSave(t *testing.T) {
	_, reporterConfig := GinkgoConfiguration()
	reporterConfig.JUnitReport = GeneralConfig.GetJunitReportPath(currentFile)

	RegisterFailHandler(Fail)
	RunSpecs(t, "Power Management Test Suite", reporterConfig)
}

var _ = BeforeSuite(func() {
	// Cleanup and create test namespace
	testNamespace := namespace.NewBuilder(APIClient, tsparams.NamespaceTesting)

	glog.V(ranparam.LogLevel).Infof("Deleting test namespace ", tsparams.NamespaceTesting)
	err := testNamespace.DeleteAndWait(tsparams.PowerSaveTimeout)
	Expect(err).ToNot(HaveOccurred(), "Failed to delete namespace ", tsparams.NamespaceTesting)

	glog.V(ranparam.LogLevel).Infof("Creating test namespace ", tsparams.NamespaceTesting)
	_, err = testNamespace.Create()
	Expect(err).ToNot(HaveOccurred(), "Failed to create namespace ", tsparams.NamespaceTesting)
})

var _ = AfterSuite(func() {
	testNamespace := namespace.NewBuilder(APIClient, tsparams.NamespaceTesting)

	glog.V(ranparam.LogLevel).Infof("Deleting test namespace", tsparams.NamespaceTesting)
	err := testNamespace.DeleteAndWait(tsparams.PowerSaveTimeout)
	Expect(err).ToNot(HaveOccurred(), "Failed to delete namespace ", tsparams.NamespaceTesting)

})
