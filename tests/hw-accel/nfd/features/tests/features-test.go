package tests

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	nfdDeploy "github.com/openshift-kni/eco-gotests/tests/hw-accel/internal/hwaccelparams/deploy"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/nfd/features/internal/nfdconfig"
	ts "github.com/openshift-kni/eco-gotests/tests/hw-accel/nfd/features/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/nfd/internal/get"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/nfd/internal/nfddelete"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/nfd/internal/search"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/nfd/internal/set"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/nfd/internal/wait"
	. "github.com/openshift-kni/eco-gotests/tests/internal/inittools"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
)

func testLabelExist(nodelabels map[string][]string, labelsToSearch, blackList []string) error {
	labelList := ts.DefaultBlackList

	if blackList != nil || len(blackList) != 0 {
		labelList = blackList
	}

	for nodeName, labels := range nodelabels {
		allFeatures := strings.Join(labels, ",")

		if len(allFeatures) == 0 {
			return fmt.Errorf("node feature labels should be greater than zero")
		}

		for _, featurelabel := range labelsToSearch {
			if search.StringInSlice(featurelabel, labelList) {
				continue
			}

			if !strings.Contains(allFeatures, fmt.Sprintf("%s=", featurelabel)) {
				return fmt.Errorf("label %s not found in node %s", featurelabel, nodeName)
			}
		}
	}

	return nil
}

func runNodeDiscoveryAndTestLabelExistence(nfdManager *nfdDeploy.NfdAPIResource, enableTopology bool) {
	err := nfdManager.DeployNfd(5*int(time.Minute), enableTopology, "")
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("error in deploying %s", err))
	By("Check that pods are in running state")

	res, err := wait.WaitForPod(APIClient, ts.Namespace)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(res).To(BeTrue())
	By("Check feature labels exists")
	Eventually(func() []string {
		nodesWithLabels, err := get.NodeFeatureLabels(APIClient, GeneralConfig.WorkerLabelMap)
		Expect(err).ShouldNot(HaveOccurred())
		allNodeLabels := []string{}
		for _, labels := range nodesWithLabels {
			allNodeLabels = append(allNodeLabels, labels...)
		}

		return allNodeLabels
	}).WithTimeout(50 * time.Second).ShouldNot(HaveLen(0))
}

func skipIfConfigNotSet(nfdConfig *nfdconfig.NfdConfig) {
	if nfdConfig.CatalogSource == "" {
		Skip("The catalog source is not set.")
	}
}

var _ = Describe("NFD", Ordered, func() {

	nfdConfig := nfdconfig.NewNfdConfig()
	nfdManager := nfdDeploy.NewNfdAPIResource(APIClient,
		ts.Namespace,
		"op-nfd",
		"nfd",
		nfdConfig.CatalogSource,
		ts.CatalogSourceNamespace,
		"nfd",
		"stable")
	Context("Node featues", Label("discovery-of-labels"), func() {
		cpuFlags := get.CPUFlags(APIClient, ts.Namespace)

		AfterAll(func() {
			By("Undeploy NFD instance")
			err := nfdManager.UndeployNfd("nfd-instance")
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("error in Undeploy NFD %s", err))

		})
		BeforeAll(func() {

			By("Clear labels")
			err := nfddelete.NfdLabelsByKeys(APIClient, "nfd.node.kubernetes.io", "feature.node.kubernetes.io")
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("error in cleaning labels\n %s", err))

			By("Creating nfd")
			runNodeDiscoveryAndTestLabelExistence(nfdManager, true)

			labelExist, labelsError := wait.WaitForLabel(APIClient, time.Minute*5, "feature")
			if !labelExist || labelsError != nil {
				glog.Error("feature labels was not found in the given time error=%v", labelsError)
			}

		})
		It("Check pods state", polarion.ID("54548"), func() {
			podlist, err := get.PodStatus(APIClient, ts.Namespace)
			Expect(err).NotTo(HaveOccurred())

			for _, pod := range podlist {
				By("Checking pod: " + pod.Name)

				Expect(pod.State).To((Equal("Running")))
			}

		})
		It("Check CPU feature labels", polarion.ID("54222"), func() {
			skipIfConfigNotSet(nfdConfig)
			nodelabels, err := get.NodeFeatureLabels(APIClient, GeneralConfig.WorkerLabelMap)

			Expect(err).NotTo(HaveOccurred())

			By("Check if features exists")

			err = testLabelExist(nodelabels, cpuFlags, nil)
			Expect(err).NotTo(HaveOccurred())

		})

		It("Check Kernel config", polarion.ID("54471"), func() {
			skipIfConfigNotSet(nfdConfig)
			nodelabels, err := get.NodeFeatureLabels(APIClient, GeneralConfig.WorkerLabelMap)
			Expect(err).NotTo(HaveOccurred())

			By("Check if custom label topology is exist")
			err = testLabelExist(nodelabels, ts.KernelConfig, nil)
			Expect(err).NotTo(HaveOccurred())

		})

		It("Check topology", polarion.ID("54491"), func() {
			Skip("configuration issue")
			skipIfConfigNotSet(nfdConfig)
			nodelabels, err := get.NodeFeatureLabels(APIClient, GeneralConfig.WorkerLabelMap)
			Expect(err).NotTo(HaveOccurred())

			By("Check if NFD labeling of the kernel config flags")
			err = testLabelExist(nodelabels, ts.Topology, nil)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Check if NUMA detected ", polarion.ID("54408"), func() {
			Skip("configuration issue")
			skipIfConfigNotSet(nfdConfig)
			nodelabels, err := get.NodeFeatureLabels(APIClient, GeneralConfig.WorkerLabelMap)
			Expect(err).NotTo(HaveOccurred())
			By("Check if NFD labeling nodes with custom NUMA labels")
			err = testLabelExist(nodelabels, ts.NUMA, nil)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Verify Feature List not contains items from Blacklist ", polarion.ID("68298"), func() {
			skipIfConfigNotSet(nfdConfig)
			By("delete old instance")
			err := nfdManager.DeleteNFDCR("nfd-instance")
			Expect(err).NotTo(HaveOccurred())

			err = nfddelete.NfdLabelsByKeys(APIClient, "nfd.node.kubernetes.io", "feature.node.kubernetes.io")
			Expect(err).NotTo(HaveOccurred())

			By("waiting for new image")
			set.CPUConfigLabels(APIClient,
				[]string{"BMI2"},
				nil,
				true,
				ts.Namespace,
				nfdConfig.Image)

			labelExist, labelsError := wait.WaitForLabel(APIClient, time.Minute*5, "feature")
			if !labelExist || labelsError != nil {
				glog.Error("feature labels was not found in the given time error=%v", labelsError)
			}

			nodelabels, err := get.NodeFeatureLabels(APIClient, GeneralConfig.WorkerLabelMap)
			Expect(err).NotTo(HaveOccurred())
			By("Check if features exists")

			err = testLabelExist(nodelabels, cpuFlags, []string{"BMI2"})
			Expect(err).NotTo(HaveOccurred())
		})

		It("Verify Feature List contains only Whitelist", polarion.ID("68300"), func() {
			skipIfConfigNotSet(nfdConfig)
			By("delete old instance")
			err := nfdManager.DeleteNFDCR("nfd-instance")
			Expect(err).NotTo(HaveOccurred())

			err = nfddelete.NfdLabelsByKeys(APIClient, "nfd.node.kubernetes.io", "feature.node.kubernetes.io")
			Expect(err).NotTo(HaveOccurred())

			By("waiting for new image")
			set.CPUConfigLabels(APIClient,
				nil,
				[]string{"BMI2"},
				true,
				ts.Namespace,
				nfdConfig.Image)

			labelExist, labelsError := wait.WaitForLabel(APIClient, time.Minute*5, "feature")
			if !labelExist || labelsError != nil {
				glog.Error("feature labels was not found in the given time error=%v", labelsError)
			}

			nodelabels, err := get.NodeFeatureLabels(APIClient, GeneralConfig.WorkerLabelMap)
			Expect(err).NotTo(HaveOccurred())
			By("Check if features exists")

			err = testLabelExist(nodelabels, []string{"BMI2"}, cpuFlags)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
