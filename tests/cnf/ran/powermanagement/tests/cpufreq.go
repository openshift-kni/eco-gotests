package tests

import (
	"fmt"
	"k8s.io/utils/cpuset"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/nto" //nolint:misspell
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/cluster"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/raninittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/powermanagement/internal/helper"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/powermanagement/internal/tsparams"
	performancev2 "github.com/openshift/cluster-node-tuning-operator/pkg/apis/performanceprofile/v2"
)

var _ = Describe("Per-core runtime power states tuning", Label(tsparams.LabelCPUFrequency), func() {
	var (
		nodeList                []*nodes.Builder
		perfProfile             *nto.Builder
		desiredReservedCoreFreq = performancev2.CPUfrequency(2500002)
		desiredIsolatedCoreFreq = performancev2.CPUfrequency(2200002)
		originalIsolatedCPUFreq performancev2.CPUfrequency
		originalReservedCPUFreq performancev2.CPUfrequency
	)

	BeforeEach(func() {
		nodeList, err := nodes.List(raninittools.Spoke1APIClient)
		Expect(err).ToNot(HaveOccurred(), "Failed to get nodes")
		Expect(len(nodeList)).To(Equal(1), "Currently only SNO clusters are supported")

		perfProfile, err = helper.GetPerformanceProfileWithCPUSet()
		Expect(err).ToNot(HaveOccurred(), "Failed to get performance profile")

		// Get isolated core ID
		isolatedCPUSet, err := cpuset.Parse(string(*perfProfile.Object.Spec.CPU.Isolated))
		Expect(err).ToNot(HaveOccurred(), "Failed to get isolated cpu set")
		isolatedCPUsList := isolatedCPUSet.List()
		isolatedCPUNumber := isolatedCPUsList[0]

		// Get reserved core ID
		reservedCPUSet, err := cpuset.Parse(string(*perfProfile.Object.Spec.CPU.Isolated))
		Expect(err).ToNot(HaveOccurred(), "Failed to get isolated cpu set")
		reservedCPUsList := reservedCPUSet.List()
		ReservedCPUNumber := reservedCPUsList[0]

		By("get original isolated core frequency")
		spokeCommand := fmt.Sprintf("cat /sys/devices/system/cpu/cpufreq/policy%v/scaling_max_freq |cat -",
			isolatedCPUNumber)
		consoleOut, err := cluster.ExecCommandOnSNO(raninittools.Spoke1APIClient, 3, spokeCommand)
		Expect(err).ToNot(HaveOccurred(), "Failed to %s", spokeCommand)
		freqAsInt, err := strconv.Atoi(strings.Trim(consoleOut, "\r\n"))
		Expect(err).ToNot(HaveOccurred(), "strconv.Atoi Failed")
		originalIsolatedCPUFreq = performancev2.CPUfrequency(freqAsInt)

		By("get original reserved core frequency")
		spokeCommand = fmt.Sprintf("cat /sys/devices/system/cpu/cpufreq/policy%v/scaling_max_freq |cat -",
			ReservedCPUNumber)
		consoleOut, err = cluster.ExecCommandOnSNO(raninittools.Spoke1APIClient, 3, spokeCommand)
		Expect(err).ToNot(HaveOccurred(), "Failed to %s", spokeCommand)
		freqAsInt, err = strconv.Atoi(strings.Trim(consoleOut, "\r\n"))
		Expect(err).ToNot(HaveOccurred(), "strconv.Atoi Failed")
		originalReservedCPUFreq = performancev2.CPUfrequency(freqAsInt)
	})

	AfterEach(func() {
		By("Reverts the CPU frequencies to the original setting")
		err := helper.SetCPUFreq(perfProfile, *nodeList[0],
			&originalIsolatedCPUFreq, &originalReservedCPUFreq)
		Expect(err).ToNot(HaveOccurred(), "Failed to set CPU Freq")
	})

	FContext("Reserved Core Frequency Tuning Test", func() {
		It("tests changing reserved and isolated CPU frequencies", func() {

			By("patch performance profile to set core frequencies")
			err := helper.SetCPUFreq(perfProfile, *nodeList[0],
				&desiredIsolatedCoreFreq, &desiredReservedCoreFreq)
			Expect(err).ToNot(HaveOccurred(), "Failed to set CPU Freq")

		})
	})
})
