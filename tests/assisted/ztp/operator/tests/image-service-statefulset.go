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
	"github.com/openshift-kni/eco-gotests/tests/internal/reportxml"
	"github.com/openshift/assisted-service/api/v1beta1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
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
				reportxml.ID("49673"), func() {
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
				reportxml.ID("49674"), func() {
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
			It("Assert imageStorage can be defined after the agentServiceConfig is created",
				reportxml.ID("49675"), func() {
					arbitraryStorageSize := "14Gi"

					By("Create an AgentServiceConfig without ImageStorage")
					tempAgentServiceConfigBuilder = assisted.NewAgentServiceConfigBuilder(
						HubAPIClient, databaseStorage, fileSystemStorage)

					// An attempt to restrict the osImages spec for the new agentserviceconfig
					// to prevent the download of all os images
					if len(osImage) > 0 {
						_ = tempAgentServiceConfigBuilder.WithOSImage(osImage[0])
					}
					_, err = tempAgentServiceConfigBuilder.Create()

					Expect(err).ToNot(HaveOccurred(), "error creating agentserviceconfig without imagestorage")

					By("Assure the AgentServiceConfig without ImageStorage was successfully created")
					_, err = tempAgentServiceConfigBuilder.WaitUntilDeployed(time.Minute * 10)
					Expect(err).ToNot(HaveOccurred(), "error waiting until agentserviceconfig without imagestorage is deployed")
					Expect(tempAgentServiceConfigBuilder.Object.Spec.ImageStorage).To(BeNil(), "error getting empty imagestorage")

					By("Assure the respective PVC doesn't exist")
					pvcbuilder, err = storage.PullPersistentVolumeClaim(
						HubAPIClient, imageServicePersistentVolumeClaimName, tsparams.MCENameSpace)
					Expect(err).Should(HaveOccurred(), fmt.Sprintf(
						"PersistentVolumeClaim %s in NameSpace %s shouldn't exist.",
						imageServicePersistentVolumeClaimName,
						tsparams.MCENameSpace))

					By("Initialize PersistentVolumeClaimSpec to be used by imageStorage")
					imageStoragePersistentVolumeClaim, err := assisted.GetDefaultStorageSpec(arbitraryStorageSize)
					Expect(err).ToNot(HaveOccurred(),
						"error initializing PersistentVolumeClaimSpec to be used by imageStorage")

					By("Pull current agentserviceconfig before updating")
					tempAgentServiceConfigBuilder, err = assisted.PullAgentServiceConfig(HubAPIClient)
					Expect(err).ToNot(HaveOccurred(), "error pulling agentserviceconfig from cluster")

					By("Update AgentServiceConfig with ImageStorage spec")
					_, err = tempAgentServiceConfigBuilder.WithImageStorage(imageStoragePersistentVolumeClaim).Update(false)
					Expect(err).ToNot(HaveOccurred(), "error updating the agentServiceConfig")

					By("Wait until AgentServiceConfig is updated with ImageStorage")
					tempAgentServiceConfigBuilder, err = tempAgentServiceConfigBuilder.WaitUntilDeployed(time.Minute * 10)
					Expect(err).ToNot(HaveOccurred(), "error waiting until agentserviceconfig with imagestorage is deployed")

					By("Assure the respective PVC was created")
					pvcbuilder, err = storage.PullPersistentVolumeClaim(
						HubAPIClient, imageServicePersistentVolumeClaimName, tsparams.MCENameSpace)
					Expect(err).ShouldNot(HaveOccurred(), fmt.Sprintf(
						"failed to get PersistentVolumeClaim %s in NameSpace %s.",
						imageServicePersistentVolumeClaimName,
						tsparams.MCENameSpace))

					By("Assert the PVC was created according to the spec")
					Expect(pvcbuilder.Object.Spec.Resources.Requests.Storage().String()).To(Equal(arbitraryStorageSize),
						"error matching pvc storage size with the expected storage size")
				})

			It("Assert respective PVC is released when imageStorage is removed from AgentServiceConfig",
				reportxml.ID("49676"), func() {
					By("Create an AgentServiceConfig with default storage specs")
					tempAgentServiceConfigBuilder = assisted.NewDefaultAgentServiceConfigBuilder(HubAPIClient)

					// An attempt to restrict the osImages spec for the new agentserviceconfig
					// to prevent the download of all os images
					if len(osImage) > 0 {
						_ = tempAgentServiceConfigBuilder.WithOSImage(osImage[0])
					}
					_, err = tempAgentServiceConfigBuilder.Create()
					Expect(err).ToNot(HaveOccurred(), "error creating agentserviceconfig with default storage specs")

					By("Assure the AgentServiceConfig with default storage specs was successfully created")
					_, err = tempAgentServiceConfigBuilder.WaitUntilDeployed(time.Minute * 10)
					Expect(err).ToNot(HaveOccurred(), "error waiting until agentserviceconfig with default storage specs is deployed")
					Expect(tempAgentServiceConfigBuilder.Object.Spec.ImageStorage).NotTo(BeNil(),
						"error validating that imagestorage spec is not empty")

					By("Assure PVC was created for the ImageStorage")
					pvcbuilder, err = storage.PullPersistentVolumeClaim(
						HubAPIClient, imageServicePersistentVolumeClaimName, tsparams.MCENameSpace)
					Expect(err).ShouldNot(HaveOccurred(), fmt.Sprintf(
						"failed to get PersistentVolumeClaim %s in NameSpace %s.",
						imageServicePersistentVolumeClaimName,
						tsparams.MCENameSpace))

					By("Pull current AgentServiceConfig before updating")
					tempAgentServiceConfigBuilder, err = assisted.PullAgentServiceConfig(HubAPIClient)
					Expect(err).ToNot(HaveOccurred(), "error pulling agentserviceconfig from cluster")

					By("Remove the imageStorage from the AgentServiceConfig")
					tempAgentServiceConfigBuilder.Object.Spec.ImageStorage = nil
					_, err = tempAgentServiceConfigBuilder.Update(false)
					Expect(err).ToNot(HaveOccurred(), "error updating the agentserviceconfig")

					By("Wait up to 1 minute for the imageservice PVC to be removed")
					err = wait.PollImmediate(5*time.Second, time.Minute*1, func() (bool, error) {
						pvcbuilder, err = storage.PullPersistentVolumeClaim(
							HubAPIClient, imageServicePersistentVolumeClaimName, tsparams.MCENameSpace)
						if err != nil {
							return true, nil
						}

						return false, nil
					})
					Expect(err).ToNot(HaveOccurred(), "error waiting until the imageservice PVC is removed.")
				})

		})
	})
