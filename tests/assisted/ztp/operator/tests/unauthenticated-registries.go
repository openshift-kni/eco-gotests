package operator_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/assisted"
	"github.com/openshift-kni/eco-goinfra/pkg/configmap"
	. "github.com/openshift-kni/eco-gotests/tests/assisted/ztp/internal/ztpinittools"
	"github.com/openshift-kni/eco-gotests/tests/assisted/ztp/operator/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
	"github.com/openshift/assisted-service/api/v1beta1"
)

const (
	assistedConfigMapName = "assisted-service"
)

var (
	osImageUR                          []v1beta1.OSImage
	tempAgentServiceConfigBuilderUR    *assisted.AgentServiceConfigBuilder
	errorVerifyingMsg                  = "error verifying that \""
	verifyPublicContainerRegistriesMsg = "Verify the PUBLIC_CONTAINER_REGISTRIES key contains\""
	retrieveAssistedConfigMapMsg       = fmt.Sprintf("Retrieve the  %s configmap", assistedConfigMapName)
)

var _ = Describe(
	"UnauthenticatedRegistries",
	Ordered,
	ContinueOnFailure,
	Label(tsparams.LabelUnauthenticatedRegistriesTestCases), func() {
		When("on MCE 2.0 and above", func() {
			BeforeAll(func() {
				By("Initialize osImage variable for the test from the original AgentServiceConfig")
				osImageUR = ZTPConfig.HubAgentServiceConfg.Object.Spec.OSImages

				By("Delete the pre-existing AgentServiceConfig")
				err = ZTPConfig.HubAgentServiceConfg.DeleteAndWait(time.Second * 10)
				Expect(err).ToNot(HaveOccurred(), "error deleting pre-existing agentserviceconfig")

			})
			AfterEach(func() {
				By("Delete AgentServiceConfig after test")
				err = tempAgentServiceConfigBuilderUR.DeleteAndWait(time.Second * 10)
				Expect(err).ToNot(HaveOccurred(), "error deleting agentserviceconfig after test")
			})
			AfterAll(func() {
				By("Re-create the original AgentServiceConfig after all tests")
				_, err = ZTPConfig.HubAgentServiceConfg.Create()
				Expect(err).ToNot(HaveOccurred(), "error re-creating the original agentserviceconfig after all tests")

				_, err = ZTPConfig.HubAgentServiceConfg.WaitUntilDeployed(time.Minute * 10)
				Expect(err).ToNot(HaveOccurred(),
					"error waiting until the original agentserviceconfig is deployed")
			})
			It("Assert AgentServiceConfig can be created without unauthenticatedRegistries in spec",
				polarion.ID("56552"), func() {
					By("Create AgentServiceConfig with default specs")
					tempAgentServiceConfigBuilderUR = assisted.NewDefaultAgentServiceConfigBuilder(HubAPIClient)

					// An attempt to restrict the osImage spec for the new agentserviceconfig
					// to prevent the download of all os images
					if len(osImageUR) > 0 {
						_, err = tempAgentServiceConfigBuilderUR.WithOSImage(osImageUR[0]).Create()
					} else {
						_, err = tempAgentServiceConfigBuilderUR.Create()
					}
					Expect(err).ToNot(HaveOccurred(),
						"error creating agentserviceconfig with default specs")

					By("Assure the AgentServiceConfig with default specs was successfully created")
					_, err = tempAgentServiceConfigBuilderUR.WaitUntilDeployed(time.Minute * 10)
					Expect(err).ToNot(HaveOccurred(),
						"error waiting until agentserviceconfig with default storage specs is deployed")

					By(retrieveAssistedConfigMapMsg)
					configMapBuilder, err := configmap.Pull(HubAPIClient, assistedConfigMapName, tsparams.MCENameSpace)
					Expect(err).ShouldNot(HaveOccurred(), fmt.Sprintf(
						"failed to get configmap %s in namespace %s", assistedConfigMapName, tsparams.MCENameSpace))

					unAuthentcatedRegistriesDefaultEntries(configMapBuilder)
				})

			It("Assert AgentServiceConfig can be created with unauthenticatedRegistries containing a default entry",
				polarion.ID("56553"), func() {
					By("Create AgentServiceConfig with unauthenticatedRegistries containing a default entry")
					tempAgentServiceConfigBuilderUR = assisted.NewDefaultAgentServiceConfigBuilder(HubAPIClient).
						WithUnauthenticatedRegistry(unAuthenticatedDefaultRegistriesList()[1])

					// An attempt to restrict the osImage spec for the new agentserviceconfig
					// to prevent the download of all os images
					if len(osImageUR) > 0 {
						_, err = tempAgentServiceConfigBuilderUR.WithOSImage(osImageUR[0]).Create()
					} else {
						_, err = tempAgentServiceConfigBuilderUR.Create()
					}
					Expect(err).ToNot(HaveOccurred(),
						"error creating agentserviceconfig with unauthenticatedRegistries containing a default entry")

					By("Assure the AgentServiceConfig with unauthenticatedRegistries containing a default entry was created")
					_, err = tempAgentServiceConfigBuilderUR.WaitUntilDeployed(time.Minute * 10)
					Expect(err).ToNot(HaveOccurred(),
						"error waiting until agentserviceconfig with unauthenticatedRegistries containing a default entry is deployed")

					By(retrieveAssistedConfigMapMsg)
					configMapBuilder, err := configmap.Pull(HubAPIClient, assistedConfigMapName, tsparams.MCENameSpace)
					Expect(err).ShouldNot(HaveOccurred(), fmt.Sprintf(
						"failed to get configmap %s in namespace %s", assistedConfigMapName, tsparams.MCENameSpace))

					unAuthentcatedRegistriesDefaultEntries(configMapBuilder)
				})

			It("Assert AgentServiceConfig can be created with unauthenticatedRegistries containing a specific entry",
				polarion.ID("56554"), func() {
					By("Create AgentServiceConfig with unauthenticatedRegistries containing a specific entry")
					tempAgentServiceConfigBuilderUR = assisted.NewDefaultAgentServiceConfigBuilder(HubAPIClient).
						WithUnauthenticatedRegistry(unAuthenticatedNonDefaultRegistriesList()[0])

					// An attempt to restrict the osImage spec for the new agentserviceconfig
					// to prevent the download of all os images
					if len(osImageUR) > 0 {
						_, err = tempAgentServiceConfigBuilderUR.WithOSImage(osImageUR[0]).Create()
					} else {
						_, err = tempAgentServiceConfigBuilderUR.Create()
					}
					Expect(err).ToNot(HaveOccurred(),
						"error creating agentserviceconfig with unauthenticatedRegistries containing a specific entry")

					By("Assure the AgentServiceConfig with unauthenticatedRegistries containing a specific entry was created")
					_, err = tempAgentServiceConfigBuilderUR.WaitUntilDeployed(time.Minute * 10)
					Expect(err).ToNot(HaveOccurred(),
						"error waiting until agentserviceconfig with unauthenticatedRegistries containing a specific entry is deployed")

					By(retrieveAssistedConfigMapMsg)
					configMapBuilder, err := configmap.Pull(HubAPIClient, assistedConfigMapName, tsparams.MCENameSpace)
					Expect(err).ShouldNot(HaveOccurred(), fmt.Sprintf(
						"failed to get configmap %s in namespace %s", assistedConfigMapName, tsparams.MCENameSpace))

					By(fmt.Sprintf("%s %s \" in the %s configmap",
						verifyPublicContainerRegistriesMsg,
						unAuthenticatedNonDefaultRegistriesList()[0],
						assistedConfigMapName))

					Expect(configMapBuilder.Definition.Data["PUBLIC_CONTAINER_REGISTRIES"]).To(
						ContainSubstring(unAuthenticatedNonDefaultRegistriesList()[0]),
						errorVerifyingMsg+unAuthenticatedNonDefaultRegistriesList()[0]+
							"\" is listed among unauthenticated registries by default")

					unAuthentcatedRegistriesDefaultEntries(configMapBuilder)
				})
			It("Assert AgentServiceConfig can be created with unauthenticatedRegistries containing multiple entries",
				polarion.ID("56555"), func() {
					By("Create AgentServiceConfig with unauthenticatedRegistries containing multiples entries")
					tempAgentServiceConfigBuilderUR = assisted.NewDefaultAgentServiceConfigBuilder(HubAPIClient)
					for _, registry := range unAuthenticatedNonDefaultRegistriesList() {
						tempAgentServiceConfigBuilderUR.WithUnauthenticatedRegistry(registry)
					}

					// An attempt to restrict the osImage spec for the new agentserviceconfig
					// to prevent the download of all os images
					if len(osImageUR) > 0 {
						_, err = tempAgentServiceConfigBuilderUR.WithOSImage(osImageUR[0]).Create()
					} else {
						_, err = tempAgentServiceConfigBuilderUR.Create()
					}
					Expect(err).ToNot(HaveOccurred(),
						"error creating agentserviceconfig with unauthenticatedRegistries containing a specific entry")

					By("Assure the AgentServiceConfig with unauthenticatedRegistries containing multiple entries was created")
					_, err = tempAgentServiceConfigBuilderUR.WaitUntilDeployed(time.Minute * 10)
					Expect(err).ToNot(HaveOccurred(),
						"error waiting until agentserviceconfig with unauthenticatedRegistries containing multiple entries is deployed")

					By(retrieveAssistedConfigMapMsg)
					configMapBuilder, err := configmap.Pull(HubAPIClient, assistedConfigMapName, tsparams.MCENameSpace)
					Expect(err).ShouldNot(HaveOccurred(), fmt.Sprintf(
						"failed to get configmap %s in namespace %s", assistedConfigMapName, tsparams.MCENameSpace))

					for _, registry := range unAuthenticatedNonDefaultRegistriesList() {
						By(fmt.Sprintf("%s %s \" in the %s configmap",
							verifyPublicContainerRegistriesMsg,
							registry,
							assistedConfigMapName))

						Expect(configMapBuilder.Definition.Data["PUBLIC_CONTAINER_REGISTRIES"]).To(
							ContainSubstring(registry),
							errorVerifyingMsg+registry+
								"\" is listed among unauthenticated registries")
					}
					unAuthentcatedRegistriesDefaultEntries(configMapBuilder)
				})
			It("Assert AgentServiceConfig can be created with unauthenticatedRegistries containing an incorrect entry",
				polarion.ID("56556"), func() {
					By("Create AgentServiceConfig with unauthenticatedRegistries containing an incorrect entry")
					incorrectRegistry := "register.redhat.io"
					tempAgentServiceConfigBuilderUR = assisted.NewDefaultAgentServiceConfigBuilder(HubAPIClient).
						WithUnauthenticatedRegistry(incorrectRegistry)

					// An attempt to restrict the osImage spec for the new agentserviceconfig
					// to prevent the download of all os images
					if len(osImageUR) > 0 {
						_, err = tempAgentServiceConfigBuilderUR.WithOSImage(osImageUR[0]).Create()
					} else {
						_, err = tempAgentServiceConfigBuilderUR.Create()
					}
					Expect(err).ToNot(HaveOccurred(),
						"error creating agentserviceconfig with unauthenticatedRegistries containing an incorrect entry")

					By("Assure the AgentServiceConfig with unauthenticatedRegistries containing an incorrect entry was created")
					_, err = tempAgentServiceConfigBuilderUR.WaitUntilDeployed(time.Minute * 10)
					Expect(err).ToNot(HaveOccurred(),
						"error waiting until agentserviceconfig with unauthenticatedRegistries containing an incorrect entry is deployed")

					By(retrieveAssistedConfigMapMsg)
					configMapBuilder, err := configmap.Pull(HubAPIClient, assistedConfigMapName, tsparams.MCENameSpace)
					Expect(err).ShouldNot(HaveOccurred(), fmt.Sprintf(
						"failed to get configmap %s in namespace %s", assistedConfigMapName, tsparams.MCENameSpace))

					By(verifyPublicContainerRegistriesMsg + incorrectRegistry +
						"\" in the " + assistedConfigMapName + " configmap")
					Expect(configMapBuilder.Definition.Data["PUBLIC_CONTAINER_REGISTRIES"]).To(
						ContainSubstring(incorrectRegistry),
						errorVerifyingMsg+incorrectRegistry+
							"\" is listed among unauthenticated registries by default")

					unAuthentcatedRegistriesDefaultEntries(configMapBuilder)
				})
		})
	})

// unAuthenticatedDefaultRegistriesList return the list of default registries.
func unAuthenticatedDefaultRegistriesList() []string {
	return []string{
		"registry.svc.ci.openshift.org",
		"quay.io",
	}
}

// unAuthenticatedNonDefaultRegistriesList return the list of non default registries.
func unAuthenticatedNonDefaultRegistriesList() []string {
	return []string{
		"registry.redhat.io",
		"registry.ci.openshift.org",
		"docker.io",
	}
}

// unAuthentcatedRegistriesDefaultEntries verifies the existence of default registries
// in the assisted-service configmap.
func unAuthentcatedRegistriesDefaultEntries(configMapBuilder *configmap.Builder) {
	for _, registry := range unAuthenticatedDefaultRegistriesList() {
		By(verifyPublicContainerRegistriesMsg + registry +
			"\" in the " + assistedConfigMapName + " configmap by default")
		Expect(configMapBuilder.Definition.Data["PUBLIC_CONTAINER_REGISTRIES"]).To(
			ContainSubstring(registry), errorVerifyingMsg+registry+
				" is listed among unauthenticated registries by default")
	}
}
