package tests

import (
	"strings"

	"github.com/golang/glog"
	"github.com/hashicorp/go-version"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift-kni/eco-goinfra/pkg/deployment"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/olm"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/internal/get"
	. "github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/internal/kmminittools"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/internal/kmmparams"
	. "github.com/openshift-kni/eco-gotests/tests/internal/inittools"
)

var _ = Describe("KMM", Ordered, Label(kmmparams.LabelSuite, kmmparams.LabelSanity), func() {
	Context("Module", Label("check-install"), func() {

		It("Operator should be properly installed", reportxml.ID("56674"), func() {
			if ModulesConfig.SubscriptionName == "" {
				Skip("No subscription name defined. Skipping test")
			}

			By("Checking subscription exists")
			sub, err := olm.PullSubscription(APIClient, ModulesConfig.SubscriptionName, kmmparams.KmmOperatorNamespace)
			Expect(err).NotTo(HaveOccurred(), "error getting subscription")
			Expect(string(sub.Object.Status.State)).To(Equal("AtLatestKnown"))

			By("Checking operator namespace exists")
			exists := namespace.NewBuilder(APIClient, kmmparams.KmmOperatorNamespace).Exists()
			Expect(exists).To(Equal(true))

			By("Listing deployment in operator namespace")
			deploymentList, err := deployment.List(APIClient, kmmparams.KmmOperatorNamespace)
			Expect(err).NotTo(HaveOccurred(), "error getting deployment list")

			By("Checking KMM deployment")
			for _, ds := range deploymentList {
				if strings.Contains(ds.Object.Name, kmmparams.DeploymentName) {
					Expect(ds.Object.Status.ReadyReplicas).To(Equal(int32(1)))
					glog.V(kmmparams.KmmLogLevel).Infof("Successfully found deployment '%s'"+
						" with ReadyReplicas %d", ds.Object.Name, ds.Object.Status.ReadyReplicas)
				}
			}
		})

		It("Webhook server should be properly installed", reportxml.ID("72719"), func() {
			By("Checking if version is greater than 2.1.0")
			currentVersion, err := get.KmmOperatorVersion(APIClient)
			Expect(err).ToNot(HaveOccurred(), "failed to get current KMM version")
			featureFromVersion, _ := version.NewVersion("2.1.0")
			if currentVersion.LessThan(featureFromVersion) {
				Skip("Test not supported for versions lower than 2.1.0")
			}

			By("Listing deployments in operator namespace")
			deploymentList, err := deployment.List(APIClient, kmmparams.KmmOperatorNamespace)
			Expect(err).NotTo(HaveOccurred(), "error getting deployment list")

			By("Checking Webhook deployment")
			for _, ds := range deploymentList {
				if strings.Contains(ds.Object.Name, kmmparams.WebhookDeploymentName) {
					Expect(ds.Object.Status.ReadyReplicas).To(Equal(int32(1)))
					glog.V(kmmparams.KmmLogLevel).Infof("Successfully found deployment '%s'"+
						" with ReadyReplicas %d", ds.Object.Name, ds.Object.Status.ReadyReplicas)
				}
			}
		})

	})
})
