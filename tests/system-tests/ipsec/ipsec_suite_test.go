package ipsec_system_test

import (
	"fmt"
	"runtime"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	. "github.com/openshift-kni/eco-gotests/tests/internal/inittools"
	"github.com/openshift-kni/eco-gotests/tests/internal/reporter"
	systemtestsparams "github.com/openshift-kni/eco-gotests/tests/system-tests/internal/systemtestsparams"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/ipsec/internal/ipsecparams"
	_ "github.com/openshift-kni/eco-gotests/tests/system-tests/ipsec/tests"
)

var (
	_, currentFile, _, _ = runtime.Caller(0)
	testNS               = namespace.NewBuilder(APIClient, ipsecparams.TestNamespaceName)
)

func TestIpsec(t *testing.T) {
	_, reporterConfig := GinkgoConfiguration()
	reporterConfig.JUnitReport = GeneralConfig.GetJunitReportPath(currentFile)

	RegisterFailHandler(Fail)
	RunSpecs(t, "IPSec SystemTests Suite", Label(ipsecparams.Labels...), reporterConfig)
}

var _ = BeforeSuite(func() {
	if !testNS.Exists() {
		fmt.Printf("Namespace %s doesn't exist. Creating.", testNS.Definition.Name)

		for key, value := range systemtestsparams.PrivilegedNSLabels {
			testNS.WithLabel(key, value)
		}

		_, err := testNS.Create()
		Expect(err).ToNot(HaveOccurred(), "error creating the test namespace")
	}
})

var _ = AfterSuite(func() {
	By("Deleting test namespace")
	err := testNS.Delete()
	Expect(err).ToNot(HaveOccurred(), "error deleting the test namespace")
})

var _ = JustAfterEach(func() {
	reporter.ReportIfFailed(
		CurrentSpecReport(), currentFile, ipsecparams.ReporterNamespacesToDump,
		ipsecparams.ReporterCRDsToDump, clients.SetScheme)
})

var _ = ReportAfterSuite("", func(report Report) {
	reportxml.Create(
		report, GeneralConfig.GetReportPath(), GeneralConfig.TCPrefix)
})
