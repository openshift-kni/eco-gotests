package tests

import (
	. "github.com/onsi/ginkgo/v2"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/sriov/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
)

var _ = Describe("ExternallyManage", Ordered, Label(tsparams.LabelExternallyManageTestCases),
	ContinueOnFailure, func() {
		It("SR-IOV: ExternallyManage: Verifying connectivity with different IP protocols", polarion.ID("63527"), func() {
			Skip("TODO")
		})
	})
