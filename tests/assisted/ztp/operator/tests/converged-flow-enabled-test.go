package operator_test

import (
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-gotests/tests/assisted/ztp/internal/find"
	"github.com/openshift-kni/eco-gotests/tests/assisted/ztp/operator/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
)

const (
	assistedContainer = "assisted-service"
)

var (
	convergedFlowLog string
)

var _ = Describe(
	"ConvergedFlowEnabled",
	Ordered,
	ContinueOnFailure,
	Label(tsparams.LabelConvergedFlowEnabled), func() {
		BeforeAll(func() {

			By("Registering CONVERGED_FLOW variable")
			assistedServicePod, err := find.AssistedServicePod()
			Expect(err).ToNot(HaveOccurred(), "error in retrieving pod")
			command := []string{"printenv", "ALLOW_CONVERGED_FLOW"}
			convergedFlowVariable, err := assistedServicePod.ExecCommand(command, assistedContainer)
			Expect(err.Error()).To(Or(BeEmpty(), Equal("command terminated with exit code 1")),
				"error msg is not as expected")
			if convergedFlowVariable.Len() != 0 {
				Skip("environment variable set not by default")
			}

			By("Registering converged flow status")
			convergedFlowLog, err = assistedServicePod.GetLog(90000*time.Hour, assistedContainer)
			Expect(err).ToNot(HaveOccurred(), "error occurred when getting log")
		})

		It("Validates that converged flow is enabled by default", polarion.ID("62628"), func() {

			enabledInLog := strings.Contains(convergedFlowLog, "Converged flow enabled: true")
			Expect(enabledInLog).To(BeTrue(), "environment variable not defined or not in log.")

		})

	})
