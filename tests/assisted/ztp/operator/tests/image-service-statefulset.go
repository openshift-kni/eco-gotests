package operator_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-gotests/pkg/assisted"
	"github.com/openshift-kni/eco-gotests/pkg/storage"
	"github.com/openshift-kni/eco-gotests/tests/assisted/ztp/internal/meets"
	. "github.com/openshift-kni/eco-gotests/tests/assisted/ztp/internal/ztpinittools"
	"github.com/openshift-kni/eco-gotests/tests/assisted/ztp/operator/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
)

var (
	agentServiceConfigBuilder             *assisted.AgentServiceConfigBuilder
	pvcbuilder                            *storage.PVCBuilder
	imageServicePersistentVolumeClaimName string = "image-service-data-assisted-image-service-0"
)
var _ = Describe(
	"ImageServiceStatefulset",
	Ordered,
	ContinueOnFailure,
	Label(tsparams.LabelImageServiceStatefulsetTestCases), func() {
		When("on ACM 2.5 and above", func() {
			BeforeAll(func() {
				By("Check that hub meets operator version requirement")
				reqMet, msg := meets.HubOperatorVersionRequirement("2.5")
				if !reqMet {
					Skip(msg)
				}

				By("Assure AgentServiceConfig exists")
				var err error
				agentServiceConfigBuilder, err = assisted.PullAgentServiceConfig(APIClient)
				Expect(err).ShouldNot(HaveOccurred(), "Failed to get AgentServiceConfig.")

				By("Assure PVC exists")
				pvcbuilder, err = storage.PullPersistentVolumeClaim(
					APIClient, imageServicePersistentVolumeClaimName, tsparams.MCENameSpace)
				Expect(err).ShouldNot(
					HaveOccurred(),
					fmt.Sprintf(
						"Failed to get PersistentVolumeClaim %s in NameSpace %s.",
						imageServicePersistentVolumeClaimName,
						tsparams.MCENameSpace))
			})

			It("Assert assisted-image-service pvc is created according to imageStorage spec",
				polarion.ID("49673"), func() {
					Expect(pvcbuilder.Object.Spec.Resources.Requests.Storage().String()).To(
						Equal(agentServiceConfigBuilder.Object.Spec.ImageStorage.Resources.Requests.Storage().String()))
				})
		})
	})
