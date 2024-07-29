package tests

import (
	"context"
	"regexp"
	"time"

	"github.com/golang/glog"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-gotests/tests/internal/cluster"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/diskencryption/internal/file"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/diskencryption/internal/helper"
	. "github.com/openshift-kni/eco-gotests/tests/system-tests/diskencryption/internal/inittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/diskencryption/internal/parsehelper"
	stdinmatcher "github.com/openshift-kni/eco-gotests/tests/system-tests/diskencryption/internal/stdin-matcher"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/diskencryption/tsparams"
)

var _ = Describe("TPM2", func() {
	var (
		err error
	)

	// BeforeEach is preparing the cluster before the tests can be run.
	// In case the cluster is in a bad state from previously running tests,
	// recovery is attempted as follows:
	// - If Secure Boot is enabled, the disk should be able to be decrypted.
	// - If not, enable it and power cycle. Otherwise the cluster will not be able to come back online
	// - If the cluster does not come back online, maybe the boot order was changed. Change it back
	// and power cycle
	// - Finally, clean the "/etc/host-hw-Updating.flag" file and wait for the cluster to come back
	// Once up check that the cluster is SNO, etc
	BeforeEach(func() {
		if BMCClient == nil {
			Skip("collecting power usage metrics requires the BMC configuration be set.")
		}

		By("checking if Secure Boot is enabled")
		var isSecureBootEnabled bool

		Eventually(func() error {
			isSecureBootEnabled, err = BMCClient.IsSecureBootEnabled()

			return err
		}).WithTimeout(tsparams.TimeoutWaitingOnBMC).
			WithPolling(tsparams.PollingIntervalBMC).ShouldNot(HaveOccurred(),
			" IsSecureBootEnabled should not return an error")

		if !isSecureBootEnabled {

			By("enabling SecureBoot")
			Eventually(BMCClient.SecureBootEnable).WithTimeout(tsparams.TimeoutWaitingOnBMC).
				WithPolling(tsparams.PollingIntervalBMC).
				Should(Or(Succeed(), MatchError("secure boot is already enabled")),
					"enabling Secure Boot should succeed even if it is already enabled")

			By("restarting node")

			Eventually(BMCClient.SystemPowerCycle).WithTimeout(tsparams.TimeoutWaitingOnBMC).
				WithPolling(tsparams.PollingIntervalBMC).ShouldNot(HaveOccurred(),
				"power cycling node should succeed")
		}

		By("waiting until all spoke 1 pods are ready")
		err = cluster.WaitForClusterRecover(APIClient, []string{
			DiskEntryptionTestConfig.GeneralConfig.MCONamespace}, tsparams.TimeoutClusterRecovery, time.Second)

		if err != nil {

			By("swapping back boot order, the server did not come up, maybe the boot order was swapped before")
			swapFirstSecondBootItems()

			By("restarting node")
			Eventually(BMCClient.SystemPowerCycle).WithTimeout(tsparams.TimeoutWaitingOnBMC).
				WithPolling(tsparams.PollingIntervalBMC).ShouldNot(HaveOccurred(),
				"power cycling node should succeed")

			By("waiting until all spoke 1 pods are ready")
			err = cluster.WaitForClusterRecover(APIClient, []string{
				DiskEntryptionTestConfig.GeneralConfig.MCONamespace}, tsparams.TimeoutClusterRecovery, time.Second)
			Expect(err).ToNot(HaveOccurred(), "Failed to wait for all spoke 1 pods to be ready")
		}

		// After this point the cluster should be back up.
		var nodeList []*nodes.Builder
		Eventually(func() int {
			nodeList, err = nodes.List(APIClient)
			Expect(err).ToNot(HaveOccurred(), "error listing nodes")

			return len(nodeList)
		}).Should(BeNumerically("==", 1), "Currently only SNO clusters are supported")

		isTTYConsole, err := helper.IsTTYConsole()
		Expect(err).ToNot(HaveOccurred(), "error checking kernel command line for tty console")
		Expect(isTTYConsole).To(BeTrue(), "the TTY options should be configured on the kernel"+
			" boot line (nomodeset console=tty0 console=ttyS0,115200n8)")

		err = file.DeleteFile("/etc/host-hw-Updating.flag")
		Expect(err).ToNot(HaveOccurred(), "error deleting /etc/host-hw-Updating.flag file")
	})

	AfterEach(func() {

		By("waiting for cluster recovery after test")
		err = cluster.WaitForClusterRecover(APIClient, []string{
			DiskEntryptionTestConfig.GeneralConfig.MCONamespace}, tsparams.TimeoutClusterRecovery, time.Second)
	})

	It("Verifies that disabling Secure Boot prevents"+
		" Disk decryption (TPM PCR 7)", func() {

		By("checking that Root disk is encrypted with tpm2 with PCR 1 and 7")
		var luksListOutput string
		luksListOutput, err = helper.GetClevisLuksListOutput()
		Expect(err).ToNot(HaveOccurred(), "error getting of clevis luks list command")
		isRootDiskTPM2PCR1AND7 := parsehelper.LuksListContainsPCR1And7(luksListOutput)
		Expect(err).ToNot(HaveOccurred(), "error when checking if the root disk is configured with TPM 1 and 7")
		if !isRootDiskTPM2PCR1AND7 {
			Skip("Root disk is not encrypted with PCR 1 and 7, skip TPM1 tests")
		}

		By("checking if Secure Boot is enabled")
		var isSecureBootEnabled bool

		Eventually(func() error {
			isSecureBootEnabled, err = BMCClient.IsSecureBootEnabled()

			return err
		}).WithTimeout(tsparams.TimeoutWaitingOnBMC).
			WithPolling(tsparams.PollingIntervalBMC).ShouldNot(HaveOccurred(),
			"IsSecureBootEnabled should not return an error")

		if !isSecureBootEnabled {
			Skip("Secure boot is not enabled test cannot proceed")
		}

		By("checking that the reserved slot is not present")
		var isRootDiskReservedSlotPresent bool
		luksListOutput, err = helper.GetClevisLuksListOutput()
		Expect(err).ToNot(HaveOccurred(), "error getting of clevis luks list command")
		isRootDiskReservedSlotPresent = parsehelper.LuksListContainsReservedSlot(luksListOutput)
		Expect(err).ToNot(HaveOccurred(), "error checking if the output of clevis luks list contains a reserved slot")
		Expect(isRootDiskReservedSlotPresent).To(BeFalse(), "there should be no reserved slot present at this point")

		By("disabling Secure Boot")
		Eventually(BMCClient.SecureBootDisable).WithTimeout(tsparams.TimeoutWaitingOnBMC).
			WithPolling(tsparams.PollingIntervalBMC).ShouldNot(HaveOccurred(),
			"disabling Secure Boot should not return an error")

		By("restarting node gracefully")
		err = cluster.SoftRebootSNO(APIClient)
		Expect(err).ToNot(HaveOccurred(), "error rebooting node")

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
		ctx, cancel := context.WithTimeout(context.TODO(), tsparams.TimeoutWaitRegex)
		defer cancel()
		matchIndex, err := stdinmatcher.WaitForRegex(ctx, BMCClient, matches)

		Expect(err).ToNot(HaveOccurred(), "WaitForRegex should not fail")
		Expect(matchIndex).To(Equal(0), "WaitForRegex should match TPM failed (0)")

		By("enabling SecureBoot")
		Eventually(BMCClient.SecureBootEnable).WithTimeout(tsparams.TimeoutWaitingOnBMC).
			WithPolling(tsparams.PollingIntervalBMC).
			Should(Or(Succeed(), MatchError("secure boot is already enabled")),
				"enabling Secure Boot should not return an error")

		By("powercycling node")
		Eventually(BMCClient.SystemPowerCycle).WithTimeout(tsparams.TimeoutWaitingOnBMC).
			WithPolling(tsparams.PollingIntervalBMC).ShouldNot(HaveOccurred(),
			"power cycling the node should succeed")

	})

	It("Verifies update indication with file (/etc/host-hw-Updating.flag)", func() {

		By("checking that Root disk is encrypted with tpm2 with PCR 1 and 7")
		var luksListOutput string
		luksListOutput, err = helper.GetClevisLuksListOutput()
		Expect(err).ToNot(HaveOccurred(), "error getting of clevis luks list command")
		isRootDiskTPM2PCR1AND7 := parsehelper.LuksListContainsPCR1And7(luksListOutput)
		if !isRootDiskTPM2PCR1AND7 {
			Skip("Root disk is not encrypted with PCR 1 and 7, skip TPM1 tests")
		}

		By("checking if Secure Boot is enabled")
		var isSecureBootEnabled bool

		Eventually(func() error {
			isSecureBootEnabled, err = BMCClient.IsSecureBootEnabled()

			return err
		}).WithTimeout(tsparams.TimeoutWaitingOnBMC).
			WithPolling(tsparams.PollingIntervalBMC).ShouldNot(HaveOccurred(),
			"IsSecureBootEnabled should not return an error")

		if !isSecureBootEnabled {
			Skip("Secure boot is not enabled test cannot proceed")
		}

		By("Checks that the reserved slot is not present")
		var isRootDiskReservedSlotPresent bool
		luksListOutput, err = helper.GetClevisLuksListOutput()
		Expect(err).ToNot(HaveOccurred(), "error getting of clevis luks list command")
		isRootDiskReservedSlotPresent = parsehelper.LuksListContainsReservedSlot(luksListOutput)
		Expect(err).ToNot(HaveOccurred(), "error checking if the output of clevis luks list contains a reserved slot")
		Expect(isRootDiskReservedSlotPresent).To(BeFalse(), "there should be no reserved slot present at this point")

		By("touch upgrade indication file")
		err = file.TouchFile("/etc/host-hw-Updating.flag")
		Expect(err).ToNot(HaveOccurred(), "the /etc/host-hw-Updating.flag file should be created without error")

		By("disabling Secure Boot")
		Eventually(BMCClient.SecureBootDisable).WithTimeout(tsparams.TimeoutWaitingOnBMC).
			WithPolling(tsparams.PollingIntervalBMC).ShouldNot(HaveOccurred(),
			"SecureBootDisable should not return an error")

		By("restarting node gracefully")
		err = cluster.SoftRebootSNO(APIClient)
		Expect(err).ToNot(HaveOccurred(), "error rebooting node")

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

		ctx, cancel := context.WithTimeout(context.TODO(), tsparams.TimeoutWaitRegex)
		defer cancel()
		matchIndex, err := stdinmatcher.WaitForRegex(ctx, BMCClient, matches)

		Expect(err).ToNot(HaveOccurred(), "WaitForRegex should not fail")
		Expect(matchIndex).To(Equal(1), "WaitForRegex should match pcr-rebind-boot (1)")

		By("enabling SecureBoot")
		Eventually(BMCClient.SecureBootEnable).WithTimeout(tsparams.TimeoutWaitingOnBMC).
			WithPolling(tsparams.PollingIntervalBMC).
			Should(Or(Succeed(), MatchError("secure boot is already enabled")),
				"enabling Secure Boot should succeed, even if it is already enabled ")

		By("waiting for cluster to recover")
		err = cluster.WaitForClusterRecover(APIClient, []string{DiskEntryptionTestConfig.GeneralConfig.MCONamespace,
			DiskEntryptionTestConfig.GeneralConfig.MCONamespace}, 45*time.Minute, time.Second)
		Expect(err).ToNot(HaveOccurred(), "cluster should recover without error")

		By("restarting node")
		err = cluster.SoftRebootSNO(APIClient)
		Expect(err).ToNot(HaveOccurred(), "error rebooting node")

	})

	It("Verifies that changing Host boot order prevents Disk decryption (TPM PCR 1)", func() {

		By("checking that Root disk is encrypted with tpm2 with PCR 1 and 7")
		var luksListOutput string
		luksListOutput, err = helper.GetClevisLuksListOutput()
		Expect(err).ToNot(HaveOccurred(), "error getting of clevis luks list command")
		isRootDiskTPM2PCR1AND7 := parsehelper.LuksListContainsPCR1And7(luksListOutput)
		if !isRootDiskTPM2PCR1AND7 {
			Skip("Root disk is not encrypted with PCR 1 and 7, skip TPM1 tests")
		}

		By("checking if Secure Boot is enabled")
		var isSecureBootEnabled bool

		Eventually(func() error {
			isSecureBootEnabled, err = BMCClient.IsSecureBootEnabled()

			return err
		}).WithTimeout(tsparams.TimeoutWaitingOnBMC).
			WithPolling(tsparams.PollingIntervalBMC).ShouldNot(HaveOccurred(),
			"IsSecureBootEnabled should not return an error")

		if !isSecureBootEnabled {
			Skip("Secure boot is not enabled test cannot proceed")
		}

		By("checking that the reserved slot is not present")
		var isRootDiskReservedSlotPresent bool
		luksListOutput, err = helper.GetClevisLuksListOutput()
		Expect(err).ToNot(HaveOccurred(), "error getting of clevis luks list command")

		isRootDiskReservedSlotPresent = parsehelper.LuksListContainsReservedSlot(luksListOutput)
		Expect(err).ToNot(HaveOccurred(), "error checking if the output of clevis luks list contains a reserved slot")
		Expect(isRootDiskReservedSlotPresent).To(BeFalse(), "there should be no reserved slot present at this point")

		By("changing the server boot order")
		swapFirstSecondBootItems()

		By("restarting node gracefully")
		err = cluster.SoftRebootSNO(APIClient)
		Expect(err).ToNot(HaveOccurred(), "error rebooting node")

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

		ctx, cancel := context.WithTimeout(context.TODO(), tsparams.TimeoutWaitRegex)
		defer cancel()
		matchIndex, err := stdinmatcher.WaitForRegex(ctx, BMCClient, matches)

		Expect(err).ToNot(HaveOccurred(), "WaitForRegex should not fail")
		Expect(matchIndex).To(Equal(0), "WaitForRegex should match TPM failed (0)")

		By("changing the server boot order back to defaults")
		swapFirstSecondBootItems()

		By("power cycling node")
		Eventually(BMCClient.SystemPowerCycle).WithTimeout(tsparams.TimeoutWaitingOnBMC).
			WithPolling(tsparams.PollingIntervalBMC).ShouldNot(HaveOccurred(),
			"power cycling the node should not return an error")

	})
})

func swapFirstSecondBootItems() {
	By("changing the server boot order back to defaults")

	var bootRefs []string

	var err error

	Eventually(func() error {
		bootRefs, err = BMCClient.SystemBootOrderReferences()

		return err
	}).ShouldNot(HaveOccurred(), "getting boot order should not return an error")

	glog.V(tsparams.LogLevel).Infof("Current boot Order: %s\n", bootRefs)
	// Creates a boot override swapping first and second boot references
	var bootOverride []string
	bootOverride, err = parsehelper.SwapFirstAndSecondSliceItems(bootRefs)
	Expect(err).ToNot(HaveOccurred(), "SwapFirstAndSecondSliceItems should not fail")
	glog.V(tsparams.LogLevel).Infof("New boot Order: %s\n", bootOverride)
	Eventually(BMCClient.SetSystemBootOrderReferences).WithTimeout(tsparams.TimeoutWaitingOnBMC).
		WithPolling(tsparams.PollingIntervalBMC).WithArguments(bootOverride).ShouldNot(HaveOccurred(),
		"changing boot order should not return an error")
}
