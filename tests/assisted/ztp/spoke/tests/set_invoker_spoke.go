package spoke_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/configmap"
	. "github.com/openshift-kni/eco-gotests/tests/assisted/ztp/internal/ztpinittools"
	"github.com/openshift-kni/eco-gotests/tests/assisted/ztp/spoke/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
)

var _ = Describe(
	"DataInvoker",
	Label(tsparams.LabelInvokerVerificationTestCases), func() {
		When("on MCE 2.0 and above", func() {

			It("Assert the invoker is set with the proper value",
				polarion.ID("43613"), func() {
					configMapName := "openshift-install-manifests"
					nameSpaceName := "openshift-config"
					expectedInvoker := "assisted-installer-operator"
					invokerKeyName := "invoker"

					By("Assure the configmap can be pulled")
					configMapBuilder, err := configmap.Pull(SpokeAPIClient, configMapName, nameSpaceName)
					Expect(err).ToNot(HaveOccurred(),
						"error pulling configmap %s in ns %s", configMapName, nameSpaceName)

					By("Assure the configmap has the proper key with a value")
					invoker, keyExists := configMapBuilder.Object.Data[invokerKeyName]
					Expect(keyExists).To(BeTrue(), "invoker key does not exist in configmap")
					Expect(invoker).To(Equal(expectedInvoker), "error matching the invoker's value to the expected one")
				})

		})
	})
