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
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/nfd/internal/get"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/nfd/internal/search"

	"github.com/openshift-kni/eco-gotests/tests/hw-accel/nfd/internal/wait"
	. "github.com/openshift-kni/eco-gotests/tests/internal/inittools"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
)

func testLabelExist(nodelabels map[string][]string, labelsToSearch []string) {
	for _, labels := range nodelabels {
		allFeatures := strings.Join(labels, ",")

		for _, featurelabel := range labelsToSearch {
			if search.StringInSlice(featurelabel, ts.DefaultBlackList) {
				continue
			}

			Expect(allFeatures).To(ContainSubstring(fmt.Sprintf("%s=", featurelabel)))
		}
	}
}

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

			err := nfdManager.DeployNfd(5*int(time.Minute), false)
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("error in deploying %s", err))
			By("Check that pod are in running state")
			res, err := wait.WaitForPod(APIClient, ts.Namespace)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(res).To(BeTrue())
			By("Check feature labels are exist")
			Eventually(func() []string {
				nodesWithLabels, err := get.NodeFeatureLabels(APIClient, GeneralConfig.WorkerLabelMap)
				Expect(err).ShouldNot(HaveOccurred())
				allNodeLabels := []string{}
				for _, labels := range nodesWithLabels {
					allNodeLabels = append(allNodeLabels, labels...)
				}

				return allNodeLabels
			}).WithTimeout(50 * time.Second).ShouldNot(HaveLen(0))
		})

		It("Check CPU feature labels", polarion.ID("54222"), func() {

			nodelabels, err := get.NodeFeatureLabels(APIClient, GeneralConfig.WorkerLabelMap)

			Expect(err).NotTo(HaveOccurred())

			By("Check if features are exist")

			testLabelExist(nodelabels, get.CPUFlags(APIClient, ts.Namespace))

		})

		It("Check Kernel config", polarion.ID("54471"), func() {

			nodelabels, err := get.NodeFeatureLabels(APIClient, GeneralConfig.WorkerLabelMap)
			Expect(err).NotTo(HaveOccurred())

			By("Check if custom label topolgy is exist")
			testLabelExist(nodelabels, ts.KernelConfig)

		})

		It("Check topology", polarion.ID("54491"), func() {
			Skip("configuration issue")
			nodelabels, err := get.NodeFeatureLabels(APIClient, GeneralConfig.WorkerLabelMap)
			Expect(err).NotTo(HaveOccurred())

			By("Check if NFD labeling of the kernel config flags")
			testLabelExist(nodelabels, ts.Topology)

		})

		It("Check if NUMA detected ", polarion.ID("54408"), func() {

			nodelabels, err := get.NodeFeatureLabels(APIClient, GeneralConfig.WorkerLabelMap)
			Expect(err).NotTo(HaveOccurred())
			By("Check if NFD labeling nodes with custom NUMA labels")
			testLabelExist(nodelabels, ts.NUMA)

		})

	})
})
