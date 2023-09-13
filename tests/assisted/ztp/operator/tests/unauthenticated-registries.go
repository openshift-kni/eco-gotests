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
	osImageUR                                                    []v1beta1.OSImage
	agentServiceConfigBuilderUR, tempAgentServiceConfigBuilderUR *assisted.AgentServiceConfigBuilder
	specificUnAuthenticatedRegistry                              string = "registry.redhat.io"
)

var _ = Describe(
	"UnauthenticatedRegistries",
	Ordered,
	ContinueOnFailure,
	Label(tsparams.LabelUnauthenticatedRegistriesTestCases), func() {
		When("on MCE 2.0 and above", func() {
			BeforeAll(func() {
				By("Retrieve the pre-existing AgentServiceConfig")
				var err error
				agentServiceConfigBuilderUR, err = assisted.PullAgentServiceConfig(HubAPIClient)
				Expect(err).ShouldNot(HaveOccurred(), "failed to get agentServiceConfig")

				By("Initialize osImage variable for the test from the original AgentServiceConfig")
				osImageUR = agentServiceConfigBuilderUR.Object.Spec.OSImages

				By("Delete the pre-existing AgentServiceConfig")
				err = agentServiceConfigBuilderUR.DeleteAndWait(time.Second * 10)
				Expect(err).ToNot(HaveOccurred(), "error deleting pre-existing agentserviceconfig")

			})
			AfterEach(func() {
				By("Delete AgentServiceConfig after test")
				err = tempAgentServiceConfigBuilderUR.DeleteAndWait(time.Second * 10)
				Expect(err).ToNot(HaveOccurred(), "error deleting agentserviceconfig after test")
			})
			AfterAll(func() {
				By("Re-create the original AgentServiceConfig after all tests")
				_, err = agentServiceConfigBuilderUR.Create()
				Expect(err).ToNot(HaveOccurred(), "error re-creating the original agentserviceconfig after all tests")

				_, err = agentServiceConfigBuilderUR.WaitUntilDeployed(time.Minute * 10)
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

					By("Retrieve the " + assistedConfigMapName + " configmap")
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

					By("Retrieve the " + assistedConfigMapName + " configmap")
					configMapBuilder, err := configmap.Pull(HubAPIClient, assistedConfigMapName, tsparams.MCENameSpace)
					Expect(err).ShouldNot(HaveOccurred(), fmt.Sprintf(
						"failed to get configmap %s in namespace %s", assistedConfigMapName, tsparams.MCENameSpace))

					unAuthentcatedRegistriesDefaultEntries(configMapBuilder)
				})

			It("Assert AgentServiceConfig can be created with unauthenticatedRegistries containing a specific entry",
				polarion.ID("56554"), func() {
					By("Create AgentServiceConfig with unauthenticatedRegistries containing a specific entry")
					tempAgentServiceConfigBuilderUR = assisted.NewDefaultAgentServiceConfigBuilder(HubAPIClient).
						WithUnauthenticatedRegistry(specificUnAuthenticatedRegistry)

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

					By("Retrieve the " + assistedConfigMapName + " configmap")
					configMapBuilder, err := configmap.Pull(HubAPIClient, assistedConfigMapName, tsparams.MCENameSpace)
					Expect(err).ShouldNot(HaveOccurred(), fmt.Sprintf(
						"failed to get configmap %s in namespace %s", assistedConfigMapName, tsparams.MCENameSpace))

					By("Verify the PUBLIC_CONTAINER_REGISTRIES key contains \"" + specificUnAuthenticatedRegistry +
						"\" in the " + assistedConfigMapName + " configmap")
					Expect(configMapBuilder.Definition.Data["PUBLIC_CONTAINER_REGISTRIES"]).To(
						ContainSubstring(specificUnAuthenticatedRegistry),
						"error verifying that \""+specificUnAuthenticatedRegistry+
							"\" is listed among unauthenticated registries by default")

					unAuthentcatedRegistriesDefaultEntries(configMapBuilder)
				})
		})
	})

func unAuthenticatedDefaultRegistriesList() []string {
	return []string{
		"registry.svc.ci.openshift.org",
		"quay.io",
	}
}
func unAuthentcatedRegistriesDefaultEntries(configMapBuilder *configmap.Builder) {
	for _, registry := range unAuthenticatedDefaultRegistriesList() {
		By("Verify the PUBLIC_CONTAINER_REGISTRIES key contains \"" + registry +
			"\" in the " + assistedConfigMapName + " configmap by default")
		Expect(configMapBuilder.Definition.Data["PUBLIC_CONTAINER_REGISTRIES"]).To(
			ContainSubstring(registry), "error verifying that "+registry+
				" is listed among unauthenticated registries by default")
	}
}
