package tests

import (
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/cgu"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/raninittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/talm/internal/helper"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/talm/internal/tsparams"
	"k8s.io/utils/ptr"
)

var _ = Describe("TALM backup tests", Label(tsparams.LabelBackupTestCases), func() {
	var (
		loopbackDevicePath string
		err                error
	)

	BeforeEach(func() {
		By("checking that the talm version is at least 4.11")
		versionInRange, err := helper.IsVersionStringInRange(tsparams.TalmVersion, "4.11", "")
		Expect(err).ToNot(HaveOccurred(), "Failed to compared talm version string")

		if !versionInRange {
			Skip("backup tests require talm 4.11 or higher")
		}
	})

	When("there is a single spoke", func() {
		BeforeEach(func() {
			By("checking that the hub and spoke 1 are present")
			Expect([]*clients.Settings{raninittools.HubAPIClient, raninittools.Spoke1APIClient}).
				ToNot(ContainElement(BeNil()), "Failed due to missing API client")
		})

		AfterEach(func() {
			By("cleaning up resources on hub")
			errorList := helper.CleanupTestResourcesOnHub(raninittools.HubAPIClient, tsparams.TestNamespace, "")
			Expect(errorList).To(BeEmpty(), "Failed to clean up test resources on hub")

			By("cleaning up resources on spoke 1")
			errorList = helper.CleanupTestResourcesOnSpokes([]*clients.Settings{raninittools.Spoke1APIClient}, "")
			Expect(errorList).To(BeEmpty(), "Failed to clean up test resources on spoke 1")
		})

		Context("with full disk for spoke1", func() {
			BeforeEach(func() {
				By("setting up filesystem to simulate low space")
				loopbackDevicePath, err = helper.PrepareEnvWithSmallMountPoint(raninittools.Spoke1APIClient)
				Expect(err).ToNot(HaveOccurred(), "Failed to prepare mount point")
			})

			AfterEach(func() {
				By("starting disk-full env clean up")
				err = helper.DiskFullEnvCleanup(raninittools.Spoke1APIClient, loopbackDevicePath)
				Expect(err).ToNot(HaveOccurred(), "Failed to clean up mount point")
			})

			// 50835 - Insufficient Backup Partition Size
			It("should have a failed cgu for single spoke", reportxml.ID("50835"), func() {
				By("applying all the required CRs for backup")
				cguBuilder := cgu.NewCguBuilder(raninittools.HubAPIClient, tsparams.CguName, tsparams.TestNamespace, 1).
					WithCluster(tsparams.Spoke1Name).
					WithManagedPolicy(tsparams.PolicyName)
				cguBuilder.Definition.Spec.Backup = true

				_, err = helper.SetupCguWithNamespace(cguBuilder, "")
				Expect(err).ToNot(HaveOccurred(), "Failed to setup cgu")

				By("waiting for cgu to fail for spoke1")
				assertBackupStatus(tsparams.Spoke1Name, "UnrecoverableError")
			})
		})

		Context("with CGU disabled", func() {
			BeforeEach(func() {
				By("checking that the talm version is at least 4.12")
				versionInRange, err := helper.IsVersionStringInRange(tsparams.TalmVersion, "4.12", "")
				Expect(err).ToNot(HaveOccurred(), "Failed to compare talm version string")

				if !versionInRange {
					Skip("CGU disabled requires talm 4.12 or higher")
				}
			})

			// 54294 - Cluster Backup and Precaching in a Disabled CGU
			It("verifies backup begins and succeeds after CGU is enabled", reportxml.ID("54294"), func() {
				By("creating a disabled cgu with backup enabled")
				cguBuilder := cgu.NewCguBuilder(raninittools.HubAPIClient, tsparams.CguName, tsparams.TestNamespace, 1).
					WithCluster(tsparams.Spoke1Name).
					WithManagedPolicy(tsparams.PolicyName)
				cguBuilder.Definition.Spec.Backup = true
				cguBuilder.Definition.Spec.Enable = ptr.To(false)
				cguBuilder.Definition.Spec.RemediationStrategy.Timeout = 30

				cguBuilder, err = helper.SetupCguWithNamespace(cguBuilder, "")
				Expect(err).ToNot(HaveOccurred(), "Failed to setup cgu")

				By("checking backup does not begin when CGU is disabled")
				// don't want to overwrite cguBuilder since it'll be nil after the error
				_, err = cguBuilder.WaitUntilBackupStarts(2 * time.Minute)
				Expect(err).To(HaveOccurred(), "Backup started when CGU is disabled")

				By("enabling CGU")
				cguBuilder.Definition.Spec.Enable = ptr.To(true)
				cguBuilder, err = cguBuilder.Update(true)
				Expect(err).ToNot(HaveOccurred(), "Failed to enable CGU")

				By("waiting for backup to begin")
				_, err = cguBuilder.WaitUntilBackupStarts(1 * time.Minute)
				Expect(err).ToNot(HaveOccurred(), "Failed to start backup")

				By("waiting for cgu to indicate backup succeeded for spoke")
				assertBackupStatus(tsparams.Spoke1Name, "Succeeded")
			})

		})
	})

	When("there are two spokes", func() {
		BeforeEach(func() {
			By("checking that hub and two spokes are present")
			Expect([]*clients.Settings{raninittools.HubAPIClient, raninittools.Spoke1APIClient, raninittools.Spoke2APIClient}).
				ToNot(ContainElement(BeNil()), "Failed due to missing API client")

			By("setting up filesystem to simulate low space")
			loopbackDevicePath, err = helper.PrepareEnvWithSmallMountPoint(raninittools.Spoke1APIClient)
			Expect(err).ToNot(HaveOccurred(), "Failed to prepare mount point")
		})

		AfterEach(func() {
			By("cleaning up resources on hub")
			errorList := helper.CleanupTestResourcesOnHub(raninittools.HubAPIClient, tsparams.TestNamespace, "")
			Expect(errorList).To(BeEmpty(), "Failed to clean up test resources on hub")

			By("starting disk-full env clean up")
			err = helper.DiskFullEnvCleanup(raninittools.Spoke1APIClient, loopbackDevicePath)
			Expect(err).ToNot(HaveOccurred(), "Failed to clean up mount point")

			By("cleaning up resources on spokes")
			errorList = helper.CleanupTestResourcesOnSpokes(
				[]*clients.Settings{raninittools.Spoke1APIClient, raninittools.Spoke2APIClient}, "")
			Expect(errorList).To(BeEmpty(), "Failed to clean up test resources on spokes")
		})

		It("should not affect backup on second spoke in same batch", func() {
			By("applying all the required CRs for backup")
			// max concurrency of 2 so both spokes are in the same batch
			cguBuilder := cgu.NewCguBuilder(raninittools.HubAPIClient, tsparams.CguName, tsparams.TestNamespace, 2).
				WithCluster(tsparams.Spoke1Name).
				WithCluster(tsparams.Spoke2Name).
				WithManagedPolicy(tsparams.PolicyName)
			cguBuilder.Definition.Spec.Backup = true

			_, err = helper.SetupCguWithNamespace(cguBuilder, "")
			Expect(err).ToNot(HaveOccurred(), "Failed to setup cgu")

			By("waiting for cgu to indicate it failed for spoke1")
			assertBackupStatus(tsparams.Spoke1Name, "UnrecoverableError")

			By("waiting for cgu to indicate it succeeded for spoke2")
			assertBackupStatus(tsparams.Spoke2Name, "Succeeded")
		})

	})
})

// assertBackupStatus asserts that the cgu backup status becomes expected within 10 minutes.
func assertBackupStatus(spokeName, expected string) {
	Eventually(func() string {
		cguBuilder, err := cgu.Pull(raninittools.HubAPIClient, tsparams.CguName, tsparams.TestNamespace)
		Expect(err).ToNot(HaveOccurred(),
			"Failed to pull cgu %s in namespace %s", tsparams.CguName, tsparams.TestNamespace)

		if cguBuilder.Object.Status.Backup == nil {
			glog.V(tsparams.LogLevel).Info("backup struct not ready yet")

			return ""
		}

		_, ok := cguBuilder.Object.Status.Backup.Status[spokeName]
		if !ok {
			glog.V(tsparams.LogLevel).Info("cluster name as key did not appear yet")

			return ""
		}

		glog.V(tsparams.LogLevel).Infof("[%s] %s backup status: %s\n", cguBuilder.Object.Name, spokeName,
			cguBuilder.Object.Status.Backup.Status[spokeName])

		return cguBuilder.Object.Status.Backup.Status[spokeName]
	}, 10*time.Minute, 10*time.Second).Should(Equal(expected))
}
