package tests

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/deployment"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/olm"
	. "github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/internal/kmminittools"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/modules/internal/tsparams"
	. "github.com/openshift-kni/eco-gotests/tests/internal/inittools"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
)

var _ = Describe("KMM", Ordered, Label(tsparams.LabelSuite), func() {
	Context("Module", Label("check-install"), func() {

		It("Operator should be properly installed", polarion.ID("56674"), func() {
			if ModulesConfig.SubscriptionName == "" {
				Skip("No subscription name defined. Skipping test")
			}

			By("Checking subscription exists")
			sub, err := olm.PullSubscription(APIClient, ModulesConfig.SubscriptionName, tsparams.KmmNamespace)
			Expect(err).NotTo(HaveOccurred(), "error getting subscription")
			Expect(string(sub.Object.Status.State)).To(Equal("AtLatestKnown"))

			By("Checking operator namespace exists")
			exists := namespace.NewBuilder(APIClient, tsparams.KmmNamespace).Exists()
			Expect(exists).To(Equal(true))

			By("Checking operator deployment")
			ds, err := deployment.Pull(APIClient, tsparams.DeploymentName, tsparams.KmmNamespace)
			Expect(err).NotTo(HaveOccurred(), "error getting deployment")
			Expect(ds.Object.Status.ReadyReplicas).To(Equal(int32(1)))
		})
	})
})
