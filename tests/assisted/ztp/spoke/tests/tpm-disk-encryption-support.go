package spoke_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/assisted"
	"github.com/openshift-kni/eco-gotests/tests/assisted/ztp/internal/find"
	. "github.com/openshift-kni/eco-gotests/tests/assisted/ztp/internal/ztpinittools"
	"github.com/openshift-kni/eco-gotests/tests/assisted/ztp/spoke/internal/diskencryption"
	"github.com/openshift-kni/eco-gotests/tests/assisted/ztp/spoke/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
	"github.com/openshift/assisted-service/models"
	"golang.org/x/exp/slices"
)

var (
	tpmSpokeClusterName           string
	tpmEncryptionEnabledOn        string
	tpmAgentClusterInstallBuilder *assisted.AgentClusterInstallBuilder
)

const (
	tpmMasterMachineConfig = "master-tpm"
	tpmWorkerMachineConfig = "worker-tpm"
)

var _ = Describe(
	"TPMDiskEncryption",
	Ordered,
	ContinueOnFailure,
	Label(tsparams.LabelTPMDiskEncryptionInstallTestCases), func() {
		When("on MCE 2.0 and above", func() {
			BeforeAll(func() {

				By("Get spoke cluster name")
				var err error
				tpmSpokeClusterName, err = find.SpokeClusterName()
				Expect(err).NotTo(HaveOccurred(), "error getting spoke cluster name")

				By("Get spoke cluster AgentClusterInstall")
				tpmAgentClusterInstallBuilder, err = assisted.PullAgentClusterInstall(
					HubAPIClient, tpmSpokeClusterName, tpmSpokeClusterName)
				Expect(err).NotTo(HaveOccurred(), "error pulling agentclusterinstall")

				if tpmAgentClusterInstallBuilder.Object.Spec.DiskEncryption == nil {
					Skip("Spoke cluster was not installed with disk encryption")
				}

				if *tpmAgentClusterInstallBuilder.Object.Spec.DiskEncryption.Mode != models.DiskEncryptionModeTpmv2 {
					Skip("Spoke cluster was installed with disk encryption mode other than tpm")
				}

				tpmEncryptionEnabledOn = *tpmAgentClusterInstallBuilder.Object.Spec.DiskEncryption.EnableOn

				Expect(err).NotTo(HaveOccurred(), "error getting tpm servers from agentclusterinstall")
			})

			It("installs on all nodes", polarion.ID("47135"), func() {
				if tpmEncryptionEnabledOn != models.DiskEncryptionEnableOnAll {
					Skip("tpm disk encryption enabledOn not set to all")
				}

				verifyTpmMasterMachineConfig()
				verifyTpmWorkerMachineConfig()
			})

			It("installs on master nodes", polarion.ID("47133"), func() {
				if tpmEncryptionEnabledOn != models.DiskEncryptionEnableOnMasters {
					Skip("tpm disk encryption enabledOn not set to masters")
				}

				verifyTpmMasterMachineConfig()
			})

			It("installs on worker nodes", polarion.ID("47134"), func() {
				if tpmEncryptionEnabledOn != models.DiskEncryptionEnableOnWorkers {
					Skip("tpm disk encryption enabledOn not set to workers")
				}

				verifyTpmWorkerMachineConfig()
			})

			It("proper positive validation is returned", polarion.ID("48319"), func() {
				if tpmEncryptionEnabledOn == models.DiskEncryptionEnableOnNone {
					Skip("tpm disk encryption enabledOn set to none")
				}

				By("Pull cluster infraenv")
				tpmInfraEnvBuilder, err := assisted.PullInfraEnvInstall(HubAPIClient, tpmSpokeClusterName, tpmSpokeClusterName)
				Expect(err).NotTo(HaveOccurred(), "error pulling cluster infraenv")

				agentBuilders, err := tpmInfraEnvBuilder.GetAllAgents()
				Expect(err).NotTo(HaveOccurred(), "error pulling agents from cluster")

				if len(agentBuilders) == 0 {
					Skip("Agent resources have been removed from hub cluster")
				}

				for _, agent := range agentBuilders {
					if tpmEncryptionEnabledOn == models.DiskEncryptionEnableOnAll ||
						strings.Contains(tpmEncryptionEnabledOn, string(agent.Object.Status.Role)) {
						hwValidations, ok := agent.Object.Status.ValidationsInfo["hardware"]
						Expect(ok).To(BeTrue(), "error attempting to retrieve agent hardware validationsInfo")
						for _, result := range hwValidations {
							if result.ID == "disk-encryption-requirements-satisfied" {
								Expect(result.Message).To(Equal("Installation disk can be encrypted using tpmv2"),
									"got unexpected hardware validation message")
								Expect(result.Status).To(Equal("success"), "got unexpected hardware validation status")
							}
						}
					}
				}

			})

		})
	})

func verifyTpmMasterMachineConfig() {
	ignitionConfig, err := diskencryption.GetIgnitionConfigFromMachineConfig(SpokeConfig.APIClient, tpmMasterMachineConfig)
	Expect(err).NotTo(HaveOccurred(), "error getting ignition config from "+tpmMasterMachineConfig+" machineconfig")

	verifyLuksTpmIgnitionConfig(ignitionConfig)
}

func verifyTpmWorkerMachineConfig() {
	ignitionConfig, err := diskencryption.GetIgnitionConfigFromMachineConfig(SpokeConfig.APIClient, tpmWorkerMachineConfig)
	Expect(err).NotTo(HaveOccurred(), "error getting ignition config from "+tpmWorkerMachineConfig+" machineconfig")

	verifyLuksTpmIgnitionConfig(ignitionConfig)
}

func verifyLuksTpmIgnitionConfig(ignitionConfig *diskencryption.IgnitionConfig) {
	luksEntry := ignitionConfig.Storage.LUKS[0]

	Expect(luksEntry.Clevis.Tpm).To(BeTrue(), "tpm not set to true in machineconfig")
	Expect(luksEntry.Device).To(Equal("/dev/disk/by-partlabel/root"), "found incorrect root device label")
	Expect(luksEntry.Name).To(Equal("root"), "found incorrect luks name")
	Expect(slices.Contains(luksEntry.Options, "--cipher")).To(BeTrue(), "luks device options did not contain --cipher")
	Expect(slices.Contains(luksEntry.Options, "aes-cbc-essiv:sha256")).To(BeTrue(),
		"luks device options did not contain aes-cbc-essiv:sha256")
	Expect(luksEntry.WipeVolume).To(BeTrue(), "luks device has wipevolume set to false")
}