package tests

import (
	"context"
	"fmt"
	"regexp"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/diskencryption/internal/file"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/diskencryption/internal/helper"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/diskencryption/internal/parsehelper"
	stdinmatcher "github.com/openshift-kni/eco-gotests/tests/cnf/ran/diskencryption/internal/stdin-matcher"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/diskencryption/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/cluster"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/raninittools"
)

var _ = Describe("TPM2", Ordered, func() {
	var (
		nodeList []*nodes.Builder
		err      error
	)

	BeforeAll(func() {
		if BMCClient == nil {
			Skip("collecting power usage metrics requires the BMC configuration be set.")
		}
	})

	BeforeEach(func() {

		By("checking if Secure Boot is enabled")
		var isSecureBootEnabled bool

		Eventually(func() error {
			isSecureBootEnabled, err = BMCClient.IsSecureBootEnabled()

			return err
		}).WithTimeout(tsparams.TimeoutWaitingOnBMC).
			WithPolling(tsparams.PoolingIntervalBMC).Should(BeNil())

		if !isSecureBootEnabled {

			By("enabling SecureBoot")
			Eventually(BMCClient.SecureBootEnable).WithTimeout(tsparams.TimeoutWaitingOnBMC).
				WithPolling(tsparams.PoolingIntervalBMC).
				Should(Or(BeNil(), MatchError("secure boot is already enabled")))

			By("restarting node")

			Eventually(BMCClient.SystemPowerCycle).WithTimeout(tsparams.TimeoutWaitingOnBMC).
				WithPolling(tsparams.PoolingIntervalBMC).Should(BeNil())
		}

		By("waiting until all spoke 1 pods are ready")
		err = cluster.WaitForClusterRecover(Spoke1APIClient, []string{
			RANConfig.GeneralConfig.MCONamespace}, tsparams.TimeoutClusterRecovery, time.Second)

		if err != nil {

			By("swapping back boot order, the server did not come up, maybe the boot order was swapped before")
			swapFirstSecondBootItems()

			By("restarting node")
			Eventually(BMCClient.SystemPowerCycle).WithTimeout(tsparams.TimeoutWaitingOnBMC).
				WithPolling(tsparams.PoolingIntervalBMC).Should(BeNil())

			By("waiting until all spoke 1 pods are ready")
			err = cluster.WaitForClusterRecover(Spoke1APIClient, []string{
				RANConfig.GeneralConfig.MCONamespace}, tsparams.TimeoutClusterRecovery, time.Second)
			Expect(err).ToNot(HaveOccurred(), "Failed to wait for all spoke 1 pods to be ready")
		}

		err = file.DeleteFile("/etc/host-hw-Updating.flag")
		Expect(err).ToNot(HaveOccurred(), "error deleting /etc/host-hw-Updating.fla file")

		Eventually(func() int {
			nodeList, err = nodes.List(Spoke1APIClient)
			Expect(err).To(BeNil(), "error listing nodes")

			return len(nodeList)
		}).Should(BeNumerically("==", 1), "Currently only SNO clusters are supported")

	})

	AfterEach(func() {

		By("waiting for cluster recovery after test")
		err = cluster.WaitForClusterRecover(Spoke1APIClient, []string{
			RANConfig.GeneralConfig.MCONamespace}, tsparams.TimeoutClusterRecovery, time.Second)
	})

	It("Verifies that disabling Secure Boot prevents"+
		" Disk decryption (TPM PCR 7)", func() {

		By("checking that Root disk is encrypted with tpm2 with PCR 1 and 7")
		var luksListOutput string
		luksListOutput, err = helper.GetClevisLuksListOutput()
		Expect(err).To(BeNil(), "error getting of clevis luks list command")
		isRootDiskTPM2PCR1AND7 := parsehelper.LuksListContainsPCR1And7(luksListOutput)
		Expect(err).To(BeNil(), "error when checking if the root disk is configured with TPM 1 and 7")
		if !isRootDiskTPM2PCR1AND7 {
			Skip("Root disk is not encrypted with PCR 1 and 7, skip TPM1 tests")
		}

		By("checking if Secure Boot is enabled")
		var isSecureBootEnabled bool

		Eventually(func() error {
			isSecureBootEnabled, err = BMCClient.IsSecureBootEnabled()

			return err
		}).WithTimeout(tsparams.TimeoutWaitingOnBMC).
			WithPolling(tsparams.PoolingIntervalBMC).Should(BeNil())

		if !isSecureBootEnabled {
			Skip("Secure boot is not enabled test cannot proceed")
		}

		By("checking that the reserved slot is not present")
		var isRootDiskReservedSlotPresent bool
		luksListOutput, err = helper.GetClevisLuksListOutput()
		Expect(err).To(BeNil(), "error getting of clevis luks list command")
		isRootDiskReservedSlotPresent = parsehelper.LuksListContainsReservedSlot(luksListOutput)
		Expect(err).To(BeNil(), "error checking if the output of clevis luks list contains a reserved slot")
		Expect(isRootDiskReservedSlotPresent).To(BeFalse(), "there should be no reserved slot present at this point")

		By("disabling Secure Boot")
		Eventually(BMCClient.SecureBootDisable).WithTimeout(tsparams.TimeoutWaitingOnBMC).
			WithPolling(tsparams.PoolingIntervalBMC).Should(BeNil())

		By("restarting node gracefully")
		err = cluster.SoftRebootSNO()
		Expect(err).To(BeNil(), "error rebooting node")

		By("waiting for Disk decryption Failure log to appear")
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
		ctx, cancel := context.WithTimeout(context.Background(), tsparams.TimeoutWaitRegex)
		defer cancel()
		matchIndex, err := stdinmatcher.WaitForRegex(ctx, BMCClient, matches)

		Expect(err).To(BeNil(), "WaitForRegex should not fail")
		Expect(matchIndex).To(Equal(0), "WaitForRegex should match TPM failed (0)")

		By("enabling SecureBoot")
		Eventually(BMCClient.SecureBootEnable).WithTimeout(tsparams.TimeoutWaitingOnBMC).
			WithPolling(tsparams.PoolingIntervalBMC).
			Should(Or(BeNil(), MatchError("secure boot is already enabled")))

		By("powercycling node")
		Eventually(BMCClient.SystemPowerCycle).WithTimeout(tsparams.TimeoutWaitingOnBMC).
			WithPolling(tsparams.PoolingIntervalBMC).Should(BeNil())

	})

	It("Verifies update indication with file (/etc/host-hw-Updating.flag)", func() {

		By("checking that Root disk is encrypted with tpm2 with PCR 1 and 7")
		var luksListOutput string
		luksListOutput, err = helper.GetClevisLuksListOutput()
		Expect(err).To(BeNil(), "error getting of clevis luks list command")
		isRootDiskTPM2PCR1AND7 := parsehelper.LuksListContainsPCR1And7(luksListOutput)
		Expect(err).To(BeNil())
		if !isRootDiskTPM2PCR1AND7 {
			Skip("Root disk is not encrypted with PCR 1 and 7, skip TPM1 tests")
		}

		By("checking if Secure Boot is enabled")
		var isSecureBootEnabled bool

		Eventually(func() error {
			isSecureBootEnabled, err = BMCClient.IsSecureBootEnabled()

			return err
		}).WithTimeout(tsparams.TimeoutWaitingOnBMC).
			WithPolling(tsparams.PoolingIntervalBMC).Should(BeNil())

		if !isSecureBootEnabled {
			Skip("Secure boot is not enabled test cannot proceed")
		}

		By("Checks that the reserved slot is not present")
		var isRootDiskReservedSlotPresent bool
		luksListOutput, err = helper.GetClevisLuksListOutput()
		Expect(err).To(BeNil(), "error getting of clevis luks list command")
		isRootDiskReservedSlotPresent = parsehelper.LuksListContainsReservedSlot(luksListOutput)
		Expect(err).To(BeNil(), "error checking if the output of clevis luks list contains a reserved slot")
		Expect(isRootDiskReservedSlotPresent).To(BeFalse(), "there should be no reserved slot present at this point")

		By("touch upgrade indication file")
		err = file.TouchFile("/etc/host-hw-Updating.flag")
		Expect(err).To(BeNil(), "the /etc/host-hw-Updating.flag file should be created without error")

		By("disabling Secure Boot")
		Eventually(BMCClient.SecureBootDisable).WithTimeout(tsparams.TimeoutWaitingOnBMC).
			WithPolling(tsparams.PoolingIntervalBMC).Should(BeNil())

		By("restarting node gracefully")
		err = cluster.SoftRebootSNO()
		Expect(err).To(BeNil(), "error rebooting node")

		By("waiting for pcr-rebind-boot log to appear (disk decryption succeeded)")

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

		ctx, cancel := context.WithTimeout(context.Background(), tsparams.TimeoutWaitRegex)
		defer cancel()
		matchIndex, err := stdinmatcher.WaitForRegex(ctx, BMCClient, matches)

		Expect(err).To(BeNil(), "WaitForRegex should not fail")
		Expect(matchIndex).To(Equal(1), "WaitForRegex should match pcr-rebind-boot (1)")

		By("enabling SecureBoot")
		Eventually(BMCClient.SecureBootEnable).WithTimeout(tsparams.TimeoutWaitingOnBMC).
			WithPolling(tsparams.PoolingIntervalBMC).
			Should(Or(BeNil(), MatchError("secure boot is already enabled")))

		By("waiting for cluster to recover")
		err = cluster.WaitForClusterRecover(Spoke1APIClient, []string{RANConfig.GeneralConfig.MCONamespace,
			RANConfig.GeneralConfig.MCONamespace}, 45*time.Minute, time.Second)
		Expect(err).To(BeNil())

		By("restarting node")
		err = cluster.SoftRebootSNO()
		Expect(err).To(BeNil(), "error rebooting node")

	})

	It("Verifies that changing Host boot order prevents Disk decryption (TPM PCR 1)", func() {

		By("checking that Root disk is encrypted with tpm2 with PCR 1 and 7")
		var luksListOutput string
		luksListOutput, err = helper.GetClevisLuksListOutput()
		Expect(err).To(BeNil(), "error getting of clevis luks list command")
		isRootDiskTPM2PCR1AND7 := parsehelper.LuksListContainsPCR1And7(luksListOutput)
		Expect(err).To(BeNil())
		if !isRootDiskTPM2PCR1AND7 {
			Skip("Root disk is not encrypted with PCR 1 and 7, skip TPM1 tests")
		}

		By("checking if Secure Boot is enabled")
		var isSecureBootEnabled bool

		Eventually(func() error {
			isSecureBootEnabled, err = BMCClient.IsSecureBootEnabled()

			return err
		}).WithTimeout(tsparams.TimeoutWaitingOnBMC).
			WithPolling(tsparams.PoolingIntervalBMC).Should(BeNil())

		if !isSecureBootEnabled {
			Skip("Secure boot is not enabled test cannot proceed")
		}

		By("checking that the reserved slot is not present")
		var isRootDiskReservedSlotPresent bool
		luksListOutput, err = helper.GetClevisLuksListOutput()
		Expect(err).To(BeNil(), "error getting of clevis luks list command")

		isRootDiskReservedSlotPresent = parsehelper.LuksListContainsReservedSlot(luksListOutput)
		Expect(err).To(BeNil(), "error checking if the output of clevis luks list contains a reserved slot")
		Expect(isRootDiskReservedSlotPresent).To(BeFalse(), "there should be no reserved slot present at this point")

		By("changing the server boot order")
		swapFirstSecondBootItems()

		By("restarting node gracefully")
		err = cluster.SoftRebootSNO()
		Expect(err).To(BeNil(), "error rebooting node")

		By("waiting for pcr-rebind-boot log to appear (disk decryption succeeded)")
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

		ctx, cancel := context.WithTimeout(context.Background(), tsparams.TimeoutWaitRegex)
		defer cancel()
		matchIndex, err := stdinmatcher.WaitForRegex(ctx, BMCClient, matches)

		Expect(err).To(BeNil(), "WaitForRegex should not fail")
		Expect(matchIndex).To(Equal(0), "WaitForRegex should match TPM failed (0)")

		By("changing the server boot order back to defaults")
		swapFirstSecondBootItems()

		By("powercycling node")
		Eventually(BMCClient.SystemPowerCycle).WithTimeout(tsparams.TimeoutWaitingOnBMC).
			WithPolling(tsparams.PoolingIntervalBMC).Should(BeNil())

	})
})

func swapFirstSecondBootItems() {
	By("changing the server boot order back to defaults")

	var bootRefs []string

	var err error

	Eventually(func() error {
		bootRefs, err = BMCClient.SystemBootOrderReferences()

		return err
	}).Should(BeNil())

	fmt.Printf("Current boot Order: %s\n", bootRefs)
	// Creates a boot override swapping first and second boot references
	var bootOverride []string
	bootOverride, err = parsehelper.SwapFirstAndSecondSliceItems(bootRefs)
	Expect(err).To(BeNil(), "SwapFirstAndSecondSliceItems should not fail")
	fmt.Printf("New boot Order: %s\n", bootOverride)
	Eventually(BMCClient.SetSystemBootOrderReferences).WithTimeout(tsparams.TimeoutWaitingOnBMC).
		WithPolling(tsparams.PoolingIntervalBMC).WithArguments(bootOverride).Should(BeNil())
}
