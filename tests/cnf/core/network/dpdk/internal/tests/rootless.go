package tests

import (
	. "github.com/onsi/ginkgo/v2"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/dpdk/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
)

var _ = Describe("rootless", Ordered, Label(tsparams.LabelSuite), ContinueOnFailure, func() {

	Context("server-tx, client-rx connectivity test on different nodes", Label("rootless"), func() {
		BeforeEach(func() {

		})

		It("single VF, multiple tap devices, multiple mac-vlans", polarion.ID("63806"), func() {
			Skip("TODO")
		})
	})

	BeforeAll(func() {

	})

	AfterAll(func() {

	})
})
