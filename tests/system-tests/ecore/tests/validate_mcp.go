package ecore_system_test

import (
	"fmt"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/mco"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"

	. "github.com/openshift-kni/eco-gotests/tests/system-tests/ecore/internal/ecoreinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/ecore/internal/ecoreparams"
)

var _ = Describe(
	"ECore MCP Validation",
	Ordered,
	ContinueOnFailure,
	Label(ecoreparams.LabelEcoreValidateMCP), func() {
		It("Assert that custom MCPs are present", polarion.ID("67022"), polarion.ID("67023"), func() {

			for _, mcpName := range ECoreConfig.MCPList {
				glog.V(ecoreparams.ECoreLogLevel).Infof(fmt.Sprintf("Checking if %q MCP exists", mcpName))
				_, err := mco.Pull(APIClient, mcpName)
				Expect(err).ToNot(HaveOccurred(), "Error pulling MCP %q", mcpName)
			}
		})
	})
