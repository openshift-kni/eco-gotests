package tests

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	nfdManager "github.com/openshift-kni/eco-gotests/tests/hw-accel/internal/hwaccelparams/deploy"
	ts "github.com/openshift-kni/eco-gotests/tests/hw-accel/nfd/features/internal/tsparams"
	getLabels "github.com/openshift-kni/eco-gotests/tests/hw-accel/nfd/internal/get"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/nfd/internal/wait"
	. "github.com/openshift-kni/eco-gotests/tests/internal/inittools"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
)

var _ = Describe("NFD", Ordered, func() {
	nfdManager := nfdManager.NewNfdAPIResource(APIClient,
		ts.Namespace,
		"op-nfd",
		"nfd",
		ts.CatalogSource,
		ts.CatalogSourceNamespace,
		"nfd",
		"stable")
	Context("Node featues", Label("discovery-of-labels"), func() {

		AfterAll(func() {
			By("Undeploy NFD instance")
			err := nfdManager.UndeployNfd("nfd-instance")
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("error in Undeploy NFD %s", err))

		})
		BeforeAll(func() {
			By("Creating nfd")

			if labelExist, labelsError := wait.WaitForLabel(APIClient, "feature"); !labelExist || labelsError != nil {
				glog.Fatalf("feature labels was not found in the given time error=%v", labelsError)
			}

			err := nfdManager.DeployNfd(180)
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("error in deploying %s", err))
		})

		It("Check labels", polarion.ID("54222"), func() {

			By("Check that pod are in running state")
			res, err := wait.WaitForPod(APIClient, ts.Namespace)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(res).To(BeTrue())
			By("Check feature labels are exist")
			Eventually(func() []string {
				nodesWithLabels, err := getLabels.NodeFeatureLabels(APIClient, GeneralConfig.WorkerLabelMap)
				Expect(err).ShouldNot(HaveOccurred())
				allNodeLabels := []string{}
				for _, labels := range nodesWithLabels {
					allNodeLabels = append(allNodeLabels, labels...)
				}

				return allNodeLabels
			}).WithTimeout(50 * time.Second).ShouldNot(HaveLen(0))

			nodelabels, err := getLabels.NodeFeatureLabels(APIClient, GeneralConfig.WorkerLabelMap)

			Expect(err).NotTo(HaveOccurred())

			By("Check if features are exist")
			for _, labels := range nodelabels {

				allFeatures := strings.Join(labels, ",")
				for _, featurelabel := range ts.FeatureLabel {

					Expect(allFeatures).To(ContainSubstring(fmt.Sprintf("%s=", featurelabel)))

				}

			}

		})
	})
})
