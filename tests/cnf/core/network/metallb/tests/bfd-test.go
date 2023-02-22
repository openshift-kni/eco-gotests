package tests

import (
	. "github.com/onsi/ginkgo/v2"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
)

var _ = Describe("BFD", Ordered, Label(tsparams.LabelBFDTestCases), func() {

	BeforeAll(func() {

	})

	BeforeEach(func() {

	})

	Context("multi hops", Label("mutihop"), func() {

		It("should provide fast link failure detection", polarion.ID("47186"), func() {

		})

	})

	Context("single hop", Label("singlehop"), func() {

		It("provides Prometheus BFD metrics", polarion.ID("47187"), func() {

		})

		It("basic functionality should provide fast link failure detection", polarion.ID("47188"), func() {

		})

	})

	AfterEach(func() {

	})

	AfterAll(func() {

	})
})
