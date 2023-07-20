package tests

import (
	. "github.com/onsi/ginkgo/v2"

	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/day1day2/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
)

var _ = Describe("Day1Day2", Ordered, Label(tsparams.LabelSuite), ContinueOnFailure, func() {
	It("Day2 Bond: change miimon configuration", polarion.ID("63881"), func() {
		Skip("TODO")
	})
})
