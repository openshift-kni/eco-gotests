package tests

import (
	"fmt"
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
		err                     error
		desiredReservedCoreFreq = performancev2.CPUfrequency(2500002)
		desiredIsolatedCoreFreq = performancev2.CPUfrequency(2200002)
		originalIsolatedCPUFreq performancev2.CPUfrequency
		originalReservedCPUFreq performancev2.CPUfrequency
		isolatedCPUNumber       = 2
		ReservedCPUNumber       = 0
	)

	BeforeEach(func() {
		nodeList, err = nodes.List(raninittools.Spoke1APIClient)
		Expect(err).ToNot(HaveOccurred(), "Failed to get nodes")
		Expect(len(nodeList)).To(Equal(1), "Currently only SNO clusters are supported")

		perfProfile, err = helper.GetPerformanceProfileWithCPUSet()
		Expect(err).ToNot(HaveOccurred(), "Failed to get performance profile")

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
		err = helper.SetCPUFreqAndWaitForMcpUpdate(perfProfile, *nodeList[0],
			&originalIsolatedCPUFreq, &originalReservedCPUFreq)
		Expect(err).ToNot(HaveOccurred(), "Failed to set CPU Freq")
	})

	FContext("Reserved Core Frequency Tuning Test", func() {
		It("tests changing reserved and isolated CPU frequencies", func() {

			By("patch performance profile to set core frequencies")
			err = helper.SetCPUFreqAndWaitForMcpUpdate(perfProfile, *nodeList[0],
				&desiredIsolatedCoreFreq, &desiredReservedCoreFreq)
			Expect(err).ToNot(HaveOccurred(), "Failed to set CPU Freq")

			By("Get modified isolated core frequency")
			spokeCommand := fmt.Sprintf("cat /sys/devices/system/cpu/cpufreq/policy%v/scaling_max_freq |cat -",
				isolatedCPUNumber)
			consoleOut, err := cluster.ExecCommandOnSNO(raninittools.Spoke1APIClient, 3, spokeCommand)
			Expect(err).ToNot(HaveOccurred(), "Failed to %s", spokeCommand)

			By("Compare current isolated core freq to desired isolated core freq")
			currIsolatedCoreFreq, err := strconv.Atoi(strings.Trim(consoleOut, "\r\n "))
			Expect(err).ToNot(HaveOccurred(), "strconv.Atoi Failed")
			Expect(currIsolatedCoreFreq).To(Equal(int(desiredIsolatedCoreFreq)),
				"Isolated CPU Frequency does not match expected frequency")

			By("Get current reserved core frequency")
			spokeCommand = fmt.Sprintf("cat /sys/devices/system/cpu/cpufreq/policy%v/scaling_max_freq |cat -",
				ReservedCPUNumber)
			consoleOut, err = cluster.ExecCommandOnSNO(raninittools.Spoke1APIClient, 3, spokeCommand)
			Expect(err).ToNot(HaveOccurred(), "Failed to %s", spokeCommand)

			By("Compare current reserved core freq to desired reserved core freq")
			currReservedCoreFreq, err := strconv.Atoi(strings.Trim(consoleOut, "\r\n "))
			Expect(err).ToNot(HaveOccurred(), "strconv.Atoi Failed")
			Expect(currReservedCoreFreq).To(Equal(int(desiredReservedCoreFreq)),
				"Reserved CPU Frequency does not match expected frequency")

		})
	})
})
