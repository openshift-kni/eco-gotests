package tests

import (
	"fmt"
	"log"
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

var _ = Describe("CPU frequency tuning tests change the core frequencies of isolated and reserved cores",
	Label(tsparams.LabelCPUFrequency), func() {
		var (
			perfProfile             *nto.Builder
			desiredReservedCoreFreq = performancev2.CPUfrequency(2500002)
			desiredIsolatedCoreFreq = performancev2.CPUfrequency(2200002)
			originalIsolatedCPUFreq performancev2.CPUfrequency
			originalReservedCPUFreq performancev2.CPUfrequency
			err                     error
		)

		BeforeEach(func() {

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

			By("getting original isolated core frequency")
			originalIsolatedCPUFreq, err = getCPUFreq(isolatedCPUNumber)
			Expect(err).ToNot(HaveOccurred(), "Failed to get original isolated core frequency")

			By("getting original reserved core frequency")
			originalReservedCPUFreq, err = getCPUFreq(reservedCPUNumber)
			Expect(err).ToNot(HaveOccurred(), "Failed to get original reserved core frequency")

			log.Println("****Original RCF: %v", originalReservedCPUFreq)

		})

		AfterEach(func() {
			By("reverting the CPU frequencies to the original setting")
			err := helper.SetCPUFreq(perfProfile, &originalIsolatedCPUFreq, &originalReservedCPUFreq)
			Expect(err).ToNot(HaveOccurred(), "Failed to set CPU Freq")
		})

		FWhen("reserved and isolated core frequency is configured via PerformanceProfile", func() {

			It("sets the reserved and isolated core frequency correctly on the DUT", func() {
				err := helper.SetCPUFreq(perfProfile, &desiredIsolatedCoreFreq, &desiredReservedCoreFreq)
				Expect(err).ToNot(HaveOccurred(), "Failed to set CPU Freq")

			})
		})
	})

// getCPUFreq gets the current frequency of a given CPU core.
func getCPUFreq(coreID int) (performancev2.CPUfrequency, error) {
	spokeCommand := fmt.Sprintf("cat /sys/devices/system/cpu/cpufreq/policy%v/scaling_max_freq",
		coreID)
	cmdOut, err := cluster.ExecCommandOnSNO(raninittools.Spoke1APIClient, 3, spokeCommand)
	Expect(err).ToNot(HaveOccurred(), "Failed to %s, error:%s", spokeCommand, cmdOut)
	freqAsInt, err := strconv.Atoi(strings.TrimSpace(cmdOut))
	Expect(err).ToNot(HaveOccurred(), "strconv.Atoi Failed")

	cpuFreq := performancev2.CPUfrequency(freqAsInt)

	return cpuFreq, err
}
