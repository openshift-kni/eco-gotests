package ecore_system_test

import (
	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/mco"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	. "github.com/openshift-kni/eco-gotests/tests/system-tests/ecore/internal/ecoreinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/ecore/internal/ecoreparams"
)

var _ = Describe(
	"ECore MCP Validation",
	Ordered,
	ContinueOnFailure,
	Label(ecoreparams.LabelEcoreValidateMCP), func() {
		DescribeTable("Verify MCP exists", func(mcpName string) {
			glog.V(ecoreparams.ECoreLogLevel).Infof("Checking if %q MCP exists", mcpName)
			_, err := mco.Pull(APIClient, mcpName)
			Expect(err).ToNot(HaveOccurred(), "Error pulling MCP %q", mcpName)
		},
			Entry("Assert 'standard' MCP", ECoreConfig.MCPOneName, reportxml.ID("67022")),
			Entry("Assert 'ht100gb' MCP", ECoreConfig.MCPTwoName, reportxml.ID("67023")),
		)
	})
