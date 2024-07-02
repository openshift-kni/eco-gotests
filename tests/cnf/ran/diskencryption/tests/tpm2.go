package tests

import (
	"fmt"
	"regexp"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/diskencryption/helper"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/diskencryption/internal/file"
	stdinmatcher "github.com/openshift-kni/eco-gotests/tests/cnf/ran/diskencryption/internal/stdin-matcher"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/diskencryption/parsehelper"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/cluster"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/raninittools"
	"github.com/openshift-kni/eco-gotests/tests/internal/inittools"
)

const (
	timeoutWaitRegex       = 10 * time.Minute
	timeoutClusterRecovery = 10 * time.Minute
)

var _ = Describe("TPM2", Ordered, func() {
	var (
		nodeList []*nodes.Builder
		err      error
	)

	BeforeAll(func() {
		if raninittools.BMCClient == nil {
			Skip("Collecting power usage metrics requires the BMC configuration be set.")
		}
	})

	BeforeEach(func() {
		By("Checks if Secure Boot is enabled")
		var isSecureBootEnabled bool

		Eventually(func() error {
			isSecureBootEnabled, err = raninittools.BMCClient.IsSecureBootEnabled()

			return err
		}).WithTimeout(10 * time.Minute).
			WithPolling(30 * time.Second).Should(BeNil())

		if !isSecureBootEnabled {
			By("Enable SecureBoot")
			Eventually(raninittools.BMCClient.SecureBootEnable).WithTimeout(10 * time.Minute).
				WithPolling(30 * time.Second).
				Should(Or(BeNil(), MatchError("secure boot is already enabled")))

			By("Restart node")

			Eventually(raninittools.BMCClient.SystemPowerCycle).WithTimeout(10 * time.Minute).
				WithPolling(30 * time.Second).Should(BeNil())
		}

		By("waiting until all spoke 1 pods are ready")
		err = cluster.WaitForClusterRecover(raninittools.Spoke1APIClient, []string{
			inittools.GeneralConfig.MCONamespace}, timeoutClusterRecovery)

		if err != nil {
			By("Server did not come up, maybe the boot order was swapped, swapping back")
			swapFirstSecondBootItems()
			By("Restart node")
			Eventually(raninittools.BMCClient.SystemPowerCycle).WithTimeout(10 * time.Minute).
				WithPolling(30 * time.Second).Should(BeNil())
			By("waiting until all spoke 1 pods are ready")
			err = cluster.WaitForClusterRecover(raninittools.Spoke1APIClient, []string{
				inittools.GeneralConfig.MCONamespace}, timeoutClusterRecovery)
			Expect(err).ToNot(HaveOccurred(), "Failed to wait for all spoke 1 pods to be ready")
		}

		err = file.DeleteFile("/etc/host-hw-Updating.flag")
		Expect(err).To(Or(MatchError("error deleting file /etc/host-hw-Updating.flag, err=%!w(<nil>)"),
			MatchError("error deleting file /etc/host-hw-Updating.flag, "+
				"err=command terminated with exit code 1\nrm: cannot remove '/etc/host-hw-Updating.flag': "+
				"No such file or directory\r\n")))

		Eventually(func() int {
			nodeList, err = nodes.List(raninittools.Spoke1APIClient)
			Expect(err).To(BeNil())

			return len(nodeList)
		}).Should(BeNumerically("==", 1), "Currently only SNO clusters are supported")

	})

	AfterEach(func() {
		By("Wait for cluster recovery after test")
		err = cluster.WaitForClusterRecover(raninittools.Spoke1APIClient, []string{
			inittools.GeneralConfig.MCONamespace}, timeoutClusterRecovery)
	})

	Context("Test root disk for TPM2 encryption", func() {
		It("Verifies that disabling Secure Boot prevents Disk decryption (TPM PCR 7)", reportxml.ID("001"),
			func(ctx SpecContext) {

				By("Checks that Root disk is encrypted with tpm2 with PCR 1 and 7")
				var luksListOutput string
				luksListOutput, err = helper.GetClevisLuksListOutput()
				Expect(err).To(BeNil())
				isRootDiskTPM2PCR1AND7 := parsehelper.LuksListContainsPCR1And7(luksListOutput)
				Expect(err).To(BeNil())
				if !isRootDiskTPM2PCR1AND7 {
					Skip("Root disk is not encrypted with PCR 1 and 7, skip TPM1 tests")
				}

				By("Checks if Secure Boot is enabled")
				var isSecureBootEnabled bool

				Eventually(func() error {
					isSecureBootEnabled, err = raninittools.BMCClient.IsSecureBootEnabled()

					return err
				}).WithTimeout(10 * time.Minute).
					WithPolling(30 * time.Second).Should(BeNil())

				if !isSecureBootEnabled {
					Skip("Secure boot is not enabled test cannot proceed")
				}

				By("Checks that the reserved slot is not present")
				var isRootDiskReservedSlotPresent bool
				luksListOutput, err = helper.GetClevisLuksListOutput()
				Expect(err).To(BeNil())
				isRootDiskReservedSlotPresent = parsehelper.LuksListContainsReservedSlot(luksListOutput)
				Expect(err).To(BeNil())
				Expect(isRootDiskReservedSlotPresent).To(BeFalse())

				By("Disable Secure Boot")

				Eventually(raninittools.BMCClient.SecureBootDisable).WithTimeout(10 * time.Minute).
					WithPolling(30 * time.Second).Should(BeNil())

				By("Gracefully Restart node")
				err = helper.SoftRebootSNO()
				Expect(err).To(BeNil())

				By("Wait for Disk decryption Failure log to appear")
				matches := []stdinmatcher.Matcher{
					{
						Regex: regexp.MustCompile("TPM failed"),
						Times: 10,
					},
					{
						Regex: regexp.MustCompile("pcr-rebind-boot"),
						Times: 1,
					},
				}

				matchIndex, timeout, err := stdinmatcher.WaitForRegex(raninittools.BMCClient, timeoutWaitRegex, matches)

				fmt.Fprintf(GinkgoWriter, "timeout:%s, matchIndex %d, err: %s\n", strconv.FormatBool(timeout), matchIndex, err)

				Expect(err).To(BeNil())
				Expect(timeout).To(BeFalse())
				Expect(matchIndex).To(Equal(0)) // pcr-rebind-boot match

				By("Enable SecureBoot")
				Eventually(raninittools.BMCClient.SecureBootEnable).WithTimeout(10 * time.Minute).
					WithPolling(30 * time.Second).
					Should(Or(BeNil(), MatchError("secure boot is already enabled")))

				By("Power cycle node")
				Eventually(raninittools.BMCClient.SystemPowerCycle).WithTimeout(10 * time.Minute).
					WithPolling(30 * time.Second).Should(BeNil())

			})

		It("Verifies update indication with file (/etc/host-hw-Updating.flag)", reportxml.ID("002"), func() {

			By("Checks that Root disk is encrypted with tpm2 with PCR 1 and 7")
			var luksListOutput string
			luksListOutput, err = helper.GetClevisLuksListOutput()
			Expect(err).To(BeNil())
			isRootDiskTPM2PCR1AND7 := parsehelper.LuksListContainsPCR1And7(luksListOutput)
			Expect(err).To(BeNil())
			if !isRootDiskTPM2PCR1AND7 {
				Skip("Root disk is not encrypted with PCR 1 and 7, skip TPM1 tests")
			}

			By("Checks if Secure Boot is enabled")
			var isSecureBootEnabled bool

			Eventually(func() error {
				isSecureBootEnabled, err = raninittools.BMCClient.IsSecureBootEnabled()

				return err
			}).WithTimeout(10 * time.Minute).
				WithPolling(30 * time.Second).Should(BeNil())

			if !isSecureBootEnabled {
				Skip("Secure boot is not enabled test cannot proceed")
			}

			By("Checks that the reserved slot is not present")
			var isRootDiskReservedSlotPresent bool
			luksListOutput, err = helper.GetClevisLuksListOutput()
			Expect(err).To(BeNil())
			isRootDiskReservedSlotPresent = parsehelper.LuksListContainsReservedSlot(luksListOutput)
			Expect(err).To(BeNil())
			Expect(isRootDiskReservedSlotPresent).To(BeFalse())

			By("touch upgrade indication file")
			err = file.TouchFile("/etc/host-hw-Updating.flag")
			Expect(err).To(BeNil())

			By("Disable Secure Boot")
			Eventually(raninittools.BMCClient.SecureBootDisable).WithTimeout(10 * time.Minute).
				WithPolling(30 * time.Second).Should(BeNil())

			By("Gracefully Restart node")
			err = helper.SoftRebootSNO()
			Expect(err).To(BeNil())

			By("Wait for pcr-rebind-boot log to appear (disk decryption succeeded)")

			matches := []stdinmatcher.Matcher{
				{
					Regex: regexp.MustCompile("TPM failed"),
					Times: 10,
				},
				{
					Regex: regexp.MustCompile("pcr-rebind-boot"),
					Times: 1,
				},
			}

			matchIndex, timeout, err := stdinmatcher.WaitForRegex(raninittools.BMCClient, timeoutWaitRegex, matches)

			fmt.Fprintf(GinkgoWriter, "timeout:%s, matchIndex %d, err: %s\n", strconv.FormatBool(timeout), matchIndex, err)
			Expect(err).To(BeNil())
			Expect(timeout).To(BeFalse())
			Expect(matchIndex).To(Equal(1)) // matches pcr-rebind-boot match

			By("Enable SecureBoot")
			Eventually(raninittools.BMCClient.SecureBootEnable).WithTimeout(10 * time.Minute).
				WithPolling(30 * time.Second).
				Should(Or(BeNil(), MatchError("secure boot is already enabled")))

			By("Wait for cluster to recover")
			err = cluster.WaitForClusterRecover(raninittools.Spoke1APIClient, []string{inittools.GeneralConfig.MCONamespace,
				inittools.GeneralConfig.MCONamespace}, 45*time.Minute)
			Expect(err).To(BeNil())

			By("Restart node")
			err = helper.SoftRebootSNO()
			Expect(err).To(BeNil())

		})

		It("Verify that changing Host boot order prevents Disk decryption (TPM PCR 1)", reportxml.ID("003"),
			func(ctx SpecContext) {

				By("Checks that Root disk is encrypted with tpm2 with PCR 1 and 7")
				var luksListOutput string
				luksListOutput, err = helper.GetClevisLuksListOutput()
				Expect(err).To(BeNil())
				isRootDiskTPM2PCR1AND7 := parsehelper.LuksListContainsPCR1And7(luksListOutput)
				Expect(err).To(BeNil())
				if !isRootDiskTPM2PCR1AND7 {
					Skip("Root disk is not encrypted with PCR 1 and 7, skip TPM1 tests")
				}

				By("Checks if Secure Boot is enabled")
				var isSecureBootEnabled bool

				Eventually(func() error {
					isSecureBootEnabled, err = raninittools.BMCClient.IsSecureBootEnabled()

					return err
				}).WithTimeout(10 * time.Minute).
					WithPolling(30 * time.Second).Should(BeNil())

				if !isSecureBootEnabled {
					Skip("Secure boot is not enabled test cannot proceed")
				}

				By("Checks that the reserved slot is not present")
				var isRootDiskReservedSlotPresent bool
				luksListOutput, err = helper.GetClevisLuksListOutput()
				Expect(err).To(BeNil())
				isRootDiskReservedSlotPresent = parsehelper.LuksListContainsReservedSlot(luksListOutput)
				Expect(err).To(BeNil())
				Expect(isRootDiskReservedSlotPresent).To(BeFalse())

				By("Change the server boot order")
				swapFirstSecondBootItems()

				By("Gracefully Restart node")
				err = helper.SoftRebootSNO()
				Expect(err).To(BeNil())

				By("Wait for pcr-rebind-boot log to appear (disk decryption succeeded)")
				matches := []stdinmatcher.Matcher{
					{
						Regex: regexp.MustCompile("TPM failed"),
						Times: 10,
					},
					{
						Regex: regexp.MustCompile("pcr-rebind-boot"),
						Times: 1,
					},
				}

				matchIndex, timeout, err := stdinmatcher.WaitForRegex(raninittools.BMCClient, timeoutWaitRegex, matches)

				fmt.Fprintf(GinkgoWriter, "timeout:%s, matchIndex %d, err: %s\n", strconv.FormatBool(timeout), matchIndex, err)

				Expect(err).To(BeNil())
				Expect(timeout).To(BeFalse())
				Expect(matchIndex).To(Equal(0)) // pcr-rebind-boot match

				By("Change the server boot order back to defaults")
				swapFirstSecondBootItems()

				By("Power cycle node")
				Eventually(raninittools.BMCClient.SystemPowerCycle).WithTimeout(10 * time.Minute).
					WithPolling(30 * time.Second).Should(BeNil())

			})
	})
})

func swapFirstSecondBootItems() {
	By("Change the server boot order back to defaults")

	var bootRefs []string

	var err error

	Eventually(func() error {
		bootRefs, err = raninittools.BMCClient.SystemBootOrderReferences()

		return err
	}).Should(BeNil())

	fmt.Printf("Current boot Order: %s\n", bootRefs)
	// Creates a boot override swapping first and second boot references
	var bootOverride []string
	bootOverride, err = parsehelper.SwapFirstAndSecondSliceItems(bootRefs)
	Expect(err).To(BeNil())
	fmt.Printf("New boot Order: %s\n", bootOverride)
	Eventually(raninittools.BMCClient.SetSystemBootOrderReferences).WithTimeout(10 * time.Minute).
		WithPolling(30 * time.Second).WithArguments(bootOverride).Should(BeNil())
}
