package tests

import (
	"fmt"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/nto" //nolint:misspell
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/cluster"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/raninittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/powermanagement/internal/helper"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/powermanagement/internal/tsparams"
	performancev2 "github.com/openshift/cluster-node-tuning-operator/pkg/apis/performanceprofile/v2"
	"k8s.io/utils/cpuset"
)

var _ = Describe("CPU frequency tuning tests", Label(tsparams.LabelCPUFrequency), func() {
	var (
		perfProfile             *nto.Builder
		desiredReservedCoreFreq = performancev2.CPUfrequency(2500001)
		desiredIsolatedCoreFreq = performancev2.CPUfrequency(2200001)
		originalIsolatedCPUFreq performancev2.CPUfrequency
		originalReservedCPUFreq performancev2.CPUfrequency
	)

	BeforeEach(func() {
		var err error

		perfProfile, err = helper.GetPerformanceProfileWithCPUSet()
		Expect(err).ToNot(HaveOccurred(), "Failed to get performance profile")

		By("getting isolated core ID")
		isolatedCPUSet, err := cpuset.Parse(string(*perfProfile.Object.Spec.CPU.Isolated))
		Expect(err).ToNot(HaveOccurred(), "Failed to get isolated cpu set")
		isolatedCPUsList := isolatedCPUSet.List()
		isolatedCPUNumber := isolatedCPUsList[0]

		By("getting reserved core ID")
		reservedCPUSet, err := cpuset.Parse(string(*perfProfile.Object.Spec.CPU.Reserved))
		Expect(err).ToNot(HaveOccurred(), "Failed to get reserved cpu set")
		reservedCPUsList := reservedCPUSet.List()
		reservedCPUNumber := reservedCPUsList[0]

		By("get original isolated core frequency")
		spokeCommand := fmt.Sprintf("cat /sys/devices/system/cpu/cpufreq/policy%v/scaling_max_freq |cat -",
			isolatedCPUNumber)
		consoleOut, err := cluster.ExecCommandOnSNO(raninittools.Spoke1APIClient, 3, spokeCommand)
		Expect(err).ToNot(HaveOccurred(), "Failed to %s, error:%s", spokeCommand, consoleOut)
		freqAsInt, err := strconv.Atoi(strings.TrimSpace(consoleOut))
		Expect(err).ToNot(HaveOccurred(), "strconv.Atoi Failed")
		originalIsolatedCPUFreq = performancev2.CPUfrequency(freqAsInt)

		By("get original reserved core frequency")
		spokeCommand = fmt.Sprintf("cat /sys/devices/system/cpu/cpufreq/policy%v/scaling_max_freq |cat -",
			reservedCPUNumber)
		consoleOut, err = cluster.ExecCommandOnSNO(raninittools.Spoke1APIClient, 3, spokeCommand)
		Expect(err).ToNot(HaveOccurred(), "Failed to %s, error: %s", spokeCommand, consoleOut)
		freqAsInt, err = strconv.Atoi(strings.TrimSpace(consoleOut))
		Expect(err).ToNot(HaveOccurred(), "strconv.Atoi Failed")
		originalReservedCPUFreq = performancev2.CPUfrequency(freqAsInt)

	})

	AfterEach(func() {
		By("Reverts the CPU frequencies to the original setting")
		err := helper.SetCPUFreq(perfProfile, &originalIsolatedCPUFreq, &originalReservedCPUFreq)
		Expect(err).ToNot(HaveOccurred(), "Failed to set CPU Freq")
	})

	Context("Reserved Core Frequency Tuning Test", func() {

		It("tests changing reserved and isolated CPU frequencies using performance profile to set core frequencies", func() {
			err := helper.SetCPUFreq(perfProfile, &desiredIsolatedCoreFreq, &desiredReservedCoreFreq)
			Expect(err).ToNot(HaveOccurred(), "Failed to set CPU Freq")

		})
	})
})
