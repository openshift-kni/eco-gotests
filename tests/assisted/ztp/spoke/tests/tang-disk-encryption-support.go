package spoke_test

import (
	"encoding/json"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/assisted"
	"github.com/openshift-kni/eco-goinfra/pkg/mco"
	"github.com/openshift-kni/eco-gotests/tests/assisted/ztp/internal/find"
	. "github.com/openshift-kni/eco-gotests/tests/assisted/ztp/internal/ztpinittools"
	"github.com/openshift-kni/eco-gotests/tests/assisted/ztp/spoke/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
	"github.com/openshift/assisted-service/models"
	"golang.org/x/exp/slices"
)

var (
	spokeClusterName               string
	tangEncryptionEnabledOn        string
	tangServers                    map[string]TangServer
	tangAgentClusterInstallBuilder *assisted.AgentClusterInstallBuilder
)

const (
	tangeMasterMachineConfig = "master-tang"
	tangeWorkerMachineConfig = "worker-tang"
)

// TangIgnitionConfig represents the ignitionconfig present in the config section of
// a tang machineconfig.
type TangIgnitionConfig struct {
	Storage struct {
		LUKS []struct {
			Clevis struct {
				Tang []struct {
					Thumbprint string `yaml:"thumbprint"`
					URL        string `yaml:"url"`
				}
			} `yaml:"clevis"`
			Device     string   `yaml:"device"`
			Name       string   `yaml:"name"`
			Options    []string `yaml:"options"`
			WipeVolume bool     `yaml:"wipeVolume"`
		} `yaml:"luks"`
	} `yaml:"storage"`
}

// TangServer represents an entry from the agentclusterinstall tangServers field.
type TangServer struct {
	URL        string `json:"URL"`
	Thumbprint string `json:"Thumbprint"`
}

var _ = Describe(
	"TangDiskEncryption",
	Ordered,
	ContinueOnFailure,
	Label(tsparams.LabelTangDiskEncryptionInstallTestCases), func() {
		When("on MCE 2.0 and above", func() {
			BeforeAll(func() {

				By("Get spoke cluster name")
				var err error
				spokeClusterName, err = find.SpokeClusterName()
				Expect(err).NotTo(HaveOccurred(), "error getting spoke cluster name")

				By("Get spoke cluster AgentClusterInstall")
				tangAgentClusterInstallBuilder, err = assisted.PullAgentClusterInstall(
					HubAPIClient, spokeClusterName, spokeClusterName)
				Expect(err).NotTo(HaveOccurred(), "error pulling spoke agentclusterinstall")

				if tangAgentClusterInstallBuilder.Object.Spec.DiskEncryption == nil {
					Skip("Spoke cluster was not installed with disk encryption")
				}

				if *tangAgentClusterInstallBuilder.Object.Spec.DiskEncryption.Mode != models.DiskEncryptionModeTang {
					Skip("Spoke cluster was installed with disk encryption mode other than tang")
				}

				tangEncryptionEnabledOn = *tangAgentClusterInstallBuilder.Object.Spec.DiskEncryption.EnableOn

				tangServers, err = createTangServersFromAgentClusterInstall(tangAgentClusterInstallBuilder)
				Expect(err).NotTo(HaveOccurred(), "error getting tang servers from spoke agentclusterinstall")
			})

			It("installs on all nodes", polarion.ID("51218"), func() {
				if tangEncryptionEnabledOn != models.DiskEncryptionEnableOnAll {
					Skip("Tang disk encryption enabledOn not set to all")
				}

				verifyMasterMachineConfig()
				verifyWorkerMachineConfig()
			})

			It("installs on master nodes", polarion.ID("47136"), func() {
				if tangEncryptionEnabledOn != models.DiskEncryptionEnableOnMasters {
					Skip("Tang disk encryption enabledOn not set to masters")
				}

				verifyMasterMachineConfig()
			})

			It("installs on worker nodes", polarion.ID("47137"), func() {
				if tangEncryptionEnabledOn != models.DiskEncryptionEnableOnWorkers {
					Skip("Tang disk encryption enabledOn not set to workers")
				}

				verifyWorkerMachineConfig()
			})

			It("proper positive validation is returned", polarion.ID("48320"), func() {
				if tangEncryptionEnabledOn == models.DiskEncryptionEnableOnNone {
					Skip("Tang disk encryption enabledOn set to none")
				}

				By("Fetch spoke cluster infraenv")
				tangInfraEnvBuilder, err := assisted.PullInfraEnvInstall(HubAPIClient, spokeClusterName, spokeClusterName)
				Expect(err).NotTo(HaveOccurred(), "error pulling spoke cluster infraenv")

				agentBuilders, err := tangInfraEnvBuilder.GetAllAgents()
				Expect(err).NotTo(HaveOccurred(), "error pulling agents from cluster")

				if len(agentBuilders) == 0 {
					Skip("Agent resources have been removed from hub cluster")
				}

				for _, agent := range agentBuilders {
					if tangEncryptionEnabledOn == models.DiskEncryptionEnableOnAll ||
						strings.Contains(tangEncryptionEnabledOn, string(agent.Object.Status.Role)) {
						hwValidations, ok := agent.Object.Status.ValidationsInfo["hardware"]
						Expect(ok).To(BeTrue(), "error attempting to retrieve agent hardware validationsInfo")
						for _, result := range hwValidations {
							if result.ID == "disk-encryption-requirements-satisfied" {
								Expect(result.Message).To(Equal("Installation disk can be encrypted using tang"),
									"got unexpected hardware validation message")
								Expect(result.Status).To(Equal("success"), "got unexpected hardware validation status")
							}
						}
					}
				}

			})

			It("propagates with multiple tang servers defined", polarion.ID("48329"), func() {
				if len(tangServers) == 1 {
					Skip("Only a single tang server used for installation")
				}

				var ignitionConfigs []TangIgnitionConfig
				if tangEncryptionEnabledOn == models.DiskEncryptionEnableOnAll ||
					tangEncryptionEnabledOn == models.DiskEncryptionEnableOnMasters {
					masterTangIgnition := getIgnitionConfigFromMachineConfig(tangeMasterMachineConfig)
					ignitionConfigs = append(ignitionConfigs, masterTangIgnition)
				}

				if tangEncryptionEnabledOn == models.DiskEncryptionEnableOnAll ||
					tangEncryptionEnabledOn == models.DiskEncryptionEnableOnWorkers {
					workerTangIgnition := getIgnitionConfigFromMachineConfig(tangeWorkerMachineConfig)
					ignitionConfigs = append(ignitionConfigs, workerTangIgnition)
				}

				for _, ignition := range ignitionConfigs {
					verifyTangServerConsistency(ignition)
				}
			})
		})
	})

func createTangServersFromAgentClusterInstall(
	builder *assisted.AgentClusterInstallBuilder) (map[string]TangServer, error) {
	var tangServers []TangServer

	err := json.Unmarshal([]byte(builder.Object.Spec.DiskEncryption.TangServers), &tangServers)
	if err != nil {
		return nil, err
	}

	var tangServerMap = make(map[string]TangServer)

	for _, server := range tangServers {
		tangServerMap[server.URL] = server
	}

	return tangServerMap, nil
}

func createTangIgnitionFromMachineConfig(builder *mco.MCBuilder) (TangIgnitionConfig, error) {
	var ignitionConfig TangIgnitionConfig

	err := json.Unmarshal(builder.Object.Spec.Config.Raw, &ignitionConfig)
	if err != nil {
		return TangIgnitionConfig{}, err
	}

	return ignitionConfig, nil
}

func verifyMasterMachineConfig() {
	verifyLuksIgnitionConfig(getIgnitionConfigFromMachineConfig(tangeMasterMachineConfig))
}

func verifyWorkerMachineConfig() {
	verifyLuksIgnitionConfig(getIgnitionConfigFromMachineConfig(tangeWorkerMachineConfig))
}

func getIgnitionConfigFromMachineConfig(machineConfigName string) TangIgnitionConfig {
	By("Check that " + machineConfigName + " machine config is created")
	tangMachineConfigBuilder, err := mco.PullMachineConfig(SpokeConfig.APIClient, machineConfigName)
	Expect(err).NotTo(HaveOccurred(), "error pulling "+machineConfigName+" machineconfig")

	By("Construct TangIgnitionConfig from " + machineConfigName + " machineconfig")

	ignitionConfig, err := createTangIgnitionFromMachineConfig(tangMachineConfigBuilder)
	Expect(err).NotTo(HaveOccurred(), "error extracting ignition config from "+machineConfigName+" machineconfig")
	Expect(len(ignitionConfig.Storage.LUKS)).To(Equal(1), "error received multiple luks devices and expected 1")

	return ignitionConfig
}

func verifyLuksIgnitionConfig(ignitionConfig TangIgnitionConfig) {
	verifyTangServerConsistency(ignitionConfig)
	luksEntry := ignitionConfig.Storage.LUKS[0]

	Expect(luksEntry.Device).To(Equal("/dev/disk/by-partlabel/root"), "found incorrect root device label")
	Expect(luksEntry.Name).To(Equal("root"), "found incorrect luks name")
	Expect(slices.Contains(luksEntry.Options, "--cipher")).To(BeTrue(), "luks device options did not contain --cipher")
	Expect(slices.Contains(luksEntry.Options, "aes-cbc-essiv:sha256")).To(BeTrue(),
		"luks device options did not contain aes-cbc-essiv:sha256")
	Expect(luksEntry.WipeVolume).To(BeTrue(), "luks device has wipevolume set to false")
}

func verifyTangServerConsistency(ignitionConfig TangIgnitionConfig) {
	luksEntry := ignitionConfig.Storage.LUKS[0]

	Expect(len(luksEntry.Clevis.Tang)).To(Equal(len(tangServers)),
		"machineconfig tang server entries do not match entries in agentclusterinstall")

	for _, ignitionConfigServer := range luksEntry.Clevis.Tang {
		aciServer, ok := tangServers[ignitionConfigServer.URL]
		Expect(ok).To(BeTrue(), "ignition config contains incorrect tang url: "+ignitionConfigServer.URL)
		Expect(ignitionConfigServer.Thumbprint).To(Equal(aciServer.Thumbprint),
			"machineconfig and agentclusterinstall tang thumbprints do not match")
	}
}
