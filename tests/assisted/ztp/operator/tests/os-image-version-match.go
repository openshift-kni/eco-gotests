package operator_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/assisted"
	"github.com/openshift-kni/eco-gotests/tests/assisted/ztp/internal/find"
	. "github.com/openshift-kni/eco-gotests/tests/assisted/ztp/internal/ztpinittools"
	"github.com/openshift-kni/eco-gotests/tests/assisted/ztp/operator/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
	agentInstallV1Beta1 "github.com/openshift/assisted-service/api/v1beta1"
)

var (
	envOSImages   []EnvOSImage
	envOSVersions []EnvOSImage
)

// EnvOSImage is a helper struct to decode OSImages correctly.
type EnvOSImage struct {
	OpenshiftVersion string `json:"openshift_version"`
	Version          string `json:"version"`
	URL              string `json:"url"`
	CPUArchitecture  string `json:"cpu_architecture"`
}

func osImagesToEnvOsImages(osImages []agentInstallV1Beta1.OSImage) []EnvOSImage {
	var envImages []EnvOSImage
	for _, osImage := range osImages {
		envImages = append(envImages, EnvOSImage{
			OpenshiftVersion: osImage.OpenshiftVersion,
			Version:          osImage.Version,
			URL:              osImage.Url,
			CPUArchitecture:  osImage.CPUArchitecture,
		})
	}

	return envImages
}

var _ = Describe(
	"OsImageVersionMatch",
	Ordered,
	ContinueOnFailure,
	Label(tsparams.LabelOSImageVersionTestCases),
	func() {
		BeforeAll(func() {

			By("Getting osImages environment variable from the assisted-service pod")
			assistedServicePod, err := find.AssistedServicePod()
			Expect(err).ToNot(HaveOccurred(), "error in retrieving pod")
			osImagesBuffer, err := assistedServicePod.ExecCommand([]string{"printenv", "OS_IMAGES"}, "assisted-service")
			Expect(err).ToNot(HaveOccurred(), "error occurred when executing command in the pod")
			Expect(osImagesBuffer).ToNot(BeNil(), "error in executing command, osimagesbuffer is nil")
			err = json.Unmarshal(osImagesBuffer.Bytes(), &envOSImages)
			Expect(err).ToNot(HaveOccurred(), "error occurred while unmarshaling")

			By("Identifying OsVersions from AgentServiceConfig")
			assistedAgentServiceConfig, err := assisted.PullAgentServiceConfig(HubAPIClient)
			Expect(err).ToNot(HaveOccurred(), "error occurred when pulling agentserviceconfig")
			Expect(assistedAgentServiceConfig.Object.Spec.OSImages).ToNot(BeNil(), "osimages field is nil")
			envOSVersions = osImagesToEnvOsImages(assistedAgentServiceConfig.Object.Spec.OSImages)
		})

		When("Test for os-version matches between the pod and the AgentServiceConfig", func() {
			It("Tries to match versions", polarion.ID("41936"), func() {
				for _, envOSVersion := range envOSVersions {
					Expect(func() bool {
						for _, envOSImage := range envOSImages {
							if envOSImage == envOSVersion {
								return true
							}
						}

						return false
					}()).To(BeTrue(), "os versions do not match between the pod and the agentserviceconfig")
				}
			})
		})

	})