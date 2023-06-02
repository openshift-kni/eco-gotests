package operator_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-gotests/pkg/assisted"
	"github.com/openshift-kni/eco-gotests/pkg/storage"
	. "github.com/openshift-kni/eco-gotests/tests/assisted/ztp/internal/ztpinittools"
	"github.com/openshift-kni/eco-gotests/tests/assisted/ztp/operator/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
	"github.com/openshift/assisted-service/api/v1beta1"
	v1 "k8s.io/api/core/v1"
)

var (
	agentServiceConfigBuilder, tempAgentServiceConfigBuilder *assisted.AgentServiceConfigBuilder
	pvcbuilder                                               *storage.PVCBuilder
	imageServicePersistentVolumeClaimName                    string = "image-service-data-assisted-image-service-0"
	databaseStorage, fileSystemStorage                       v1.PersistentVolumeClaimSpec
	osImage                                                  []v1beta1.OSImage
)
var _ = Describe(
	"ImageServiceStatefulset",
	Ordered,
	ContinueOnFailure,
	Label(tsparams.LabelImageServiceStatefulsetTestCases), func() {
		When("on MCE 2.0 and above", func() {
			BeforeAll(func() {
				By("Retrieve the pre-existing AgentServiceConfig")
				var err error
				agentServiceConfigBuilder, err = assisted.PullAgentServiceConfig(HubAPIClient)
				Expect(err).ShouldNot(HaveOccurred(), "Failed to get AgentServiceConfig.")

				By("Initialize variables for the test from the original AgentServiceConfig")
				databaseStorage = agentServiceConfigBuilder.Definition.Spec.DatabaseStorage
				fileSystemStorage = agentServiceConfigBuilder.Definition.Spec.FileSystemStorage
				osImage = agentServiceConfigBuilder.Object.Spec.OSImages

				By("Delete the pre-existing AgentServiceConfig")
				err = agentServiceConfigBuilder.DeleteAndWait(time.Second * 10)
				Expect(err).ToNot(HaveOccurred(), "error deleting pre-existing agentserviceconfig")

			})
			AfterEach(func() {
				By("Delete AgentServiceConfig after test")
				err = tempAgentServiceConfigBuilder.DeleteAndWait(time.Second * 10)
				Expect(err).ToNot(HaveOccurred(), "error deleting agentserviceconfig after test")
			})
			AfterAll(func() {
				By("Re-create the original AgentServiceConfig after all tests")
				_, err = agentServiceConfigBuilder.Create()
				Expect(err).ToNot(HaveOccurred(), "error re-creating the original agentserviceconfig after all tests")

				_, err = agentServiceConfigBuilder.WaitUntilDeployed(time.Minute * 10)
				Expect(err).ToNot(HaveOccurred(),
					"error waiting until agentserviceconfig without imagestorage is deployed")
			})
			It("Assert assisted-image-service pvc is created according to imageStorage spec",
				polarion.ID("49673"), func() {
					By("Create AgentServiceConfig with default storage specs")
					tempAgentServiceConfigBuilder = assisted.NewDefaultAgentServiceConfigBuilder(HubAPIClient)

					// An attempt to restrict the osImages spec for the new agentserviceconfig
					// to prevent the download of all os images
					if len(osImage) > 0 {
						_, err = tempAgentServiceConfigBuilder.WithOSImage(osImage[0]).Create()
					} else {
						_, err = tempAgentServiceConfigBuilder.Create()
					}
					Expect(err).ToNot(HaveOccurred(),
						"error creating agentserviceconfig with default storage specs")

					By("Assure the AgentServiceConfig with default storage specs was successfully created")
					_, err = tempAgentServiceConfigBuilder.WaitUntilDeployed(time.Minute * 10)
					Expect(err).ToNot(HaveOccurred(),
						"error waiting until agentserviceconfig with default storage specs is deployed")

					By("Assure PVC exists")
					pvcbuilder, err = storage.PullPersistentVolumeClaim(
						HubAPIClient, imageServicePersistentVolumeClaimName, tsparams.MCENameSpace)
					Expect(err).ShouldNot(HaveOccurred(), fmt.Sprintf(
						"Failed to get PersistentVolumeClaim %s in NameSpace %s.",
						imageServicePersistentVolumeClaimName,
						tsparams.MCENameSpace))

					By("Assure PVC spec matches the value set in AgentServiceConfig")
					Expect(pvcbuilder.Object.Spec.Resources.Requests.Storage().String()).To(
						Equal(tempAgentServiceConfigBuilder.Object.Spec.ImageStorage.Resources.Requests.Storage().String()))
				})

			It("Assert assisted-image-service is healthy without imageStorage defined in AgentServiceConfig",
				polarion.ID("49674"), func() {
					By("Create an AgentServiceConfig without ImageStorage")
					tempAgentServiceConfigBuilder = assisted.NewAgentServiceConfigBuilder(
						HubAPIClient, databaseStorage, fileSystemStorage)

					// An attempt to restrict the osImages spec for the new agentserviceconfig
					// to prevent the download of all os images
					if len(osImage) > 0 {
						_, err = tempAgentServiceConfigBuilder.WithOSImage(osImage[0]).Create()
					} else {
						_, err = tempAgentServiceConfigBuilder.Create()
					}
					Expect(err).ToNot(HaveOccurred(), "error creating agentserviceconfig without imagestorage")

					By("Assure the AgentServiceConfig without ImageStorage was successfully created")
					_, err = tempAgentServiceConfigBuilder.WaitUntilDeployed(time.Minute * 10)
					Expect(err).ToNot(HaveOccurred(), "error waiting until agentserviceconfig without imagestorage is deployed")
					Expect(tempAgentServiceConfigBuilder.Object.Spec.ImageStorage).To(BeNil())
				})
		})
	})
