package tests

import (
	"strings"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/deployment"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/namespace"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/olm"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/reportxml"
	. "github.com/rh-ecosystem-edge/eco-gotests/tests/hw-accel/kmm/internal/kmminittools"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/hw-accel/kmm/internal/kmmparams"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/hw-accel/kmm/modules/internal/tsparams"
	. "github.com/rh-ecosystem-edge/eco-gotests/tests/internal/inittools"
)

var _ = Describe("KMM", Ordered, Label(tsparams.LabelSuite, tsparams.LabelSanity), func() {
	Context("Module", Label("check-install"), func() {

		It("Operator should be properly installed", reportxml.ID("56674"), func() {
			if ModulesConfig.SubscriptionName == "" {
				Skip("No subscription name defined. Skipping test")
			}

			By("Checking subscription exists")
			sub, err := olm.PullSubscription(APIClient, ModulesConfig.SubscriptionName, tsparams.KmmOperatorNamespace)
			Expect(err).NotTo(HaveOccurred(), "error getting subscription")
			Expect(string(sub.Object.Status.State)).To(Equal("AtLatestKnown"))

			By("Checking operator namespace exists")
			exists := namespace.NewBuilder(APIClient, tsparams.KmmOperatorNamespace).Exists()
			Expect(exists).To(Equal(true))

			By("Listing deployment in operator namespace")
			deploymentList, err := deployment.List(APIClient, tsparams.KmmOperatorNamespace)
			Expect(err).NotTo(HaveOccurred(), "error getting deployment list")

			By("Checking deployment")
			for _, ds := range deploymentList {
				if strings.Contains(ds.Object.Name, tsparams.DeploymentName) {
					Expect(err).NotTo(HaveOccurred(), "error getting deployment")
					Expect(ds.Object.Status.ReadyReplicas).To(Equal(int32(1)))
					glog.V(kmmparams.KmmLogLevel).Infof("Successfully found deployment '%s'"+
						" with ReadyReplicas %d", ds.Object.Name, ds.Object.Status.ReadyReplicas)
				}
			}
		})
	})
})
