package tests

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	nfdDeploy "github.com/openshift-kni/eco-gotests/tests/hw-accel/internal/deploy"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/nfd/features/internal/helpers"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/nfd/features/internal/nfdconfig"
	ts "github.com/openshift-kni/eco-gotests/tests/hw-accel/nfd/features/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/nfd/internal/get"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/nfd/internal/nfddelete"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/nfd/internal/set"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/nfd/internal/wait"
	. "github.com/openshift-kni/eco-gotests/tests/internal/inittools"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
	"k8s.io/client-go/util/retry"
)

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
		var cpuFlags map[string][]string

		AfterAll(func() {
			By("Undeploy NFD instance")
			err := nfdManager.UndeployNfd("nfd-instance")
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("error in Undeploy NFD %s", err))

		})
		BeforeAll(func() {
			cpuFlags = get.CPUFlags(APIClient, ts.Namespace)
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
			err := helpers.CheckPodStatus(APIClient)
			Expect(err).NotTo(HaveOccurred())

		})
		It("Check CPU feature labels", polarion.ID("54222"), func() {
			skipIfConfigNotSet(nfdConfig)
			nodelabels, err := get.NodeFeatureLabels(APIClient, GeneralConfig.WorkerLabelMap)

			Expect(err).NotTo(HaveOccurred())

			By("Check if features exists")

			for nodeName := range nodelabels {
				err = helpers.CheckLabelsExist(nodelabels, cpuFlags[nodeName], nil, nodeName)
				Expect(err).NotTo(HaveOccurred())
			}

		})

		It("Check Kernel config", polarion.ID("54471"), func() {
			skipIfConfigNotSet(nfdConfig)
			nodelabels, err := get.NodeFeatureLabels(APIClient, GeneralConfig.WorkerLabelMap)
			Expect(err).NotTo(HaveOccurred())

			By("Check if custom label topology is exist")
			for nodeName := range nodelabels {
				err = helpers.CheckLabelsExist(nodelabels, ts.KernelConfig, nil, nodeName)
				Expect(err).NotTo(HaveOccurred())
			}

		})

		It("Check topology", polarion.ID("54491"), func() {
			Skip("configuration issue")
			skipIfConfigNotSet(nfdConfig)
			nodelabels, err := get.NodeFeatureLabels(APIClient, GeneralConfig.WorkerLabelMap)
			Expect(err).NotTo(HaveOccurred())

			By("Check if NFD labeling of the kernel config flags")
			for nodeName := range nodelabels {
				err = helpers.CheckLabelsExist(nodelabels, ts.Topology, nil, nodeName)
				Expect(err).NotTo(HaveOccurred())
			}

		})

		It("Check if NUMA detected ", polarion.ID("54408"), func() {
			Skip("configuration issue")
			skipIfConfigNotSet(nfdConfig)
			nodelabels, err := get.NodeFeatureLabels(APIClient, GeneralConfig.WorkerLabelMap)
			Expect(err).NotTo(HaveOccurred())
			By("Check if NFD labeling nodes with custom NUMA labels")
			for nodeName := range nodelabels {
				err = helpers.CheckLabelsExist(nodelabels, ts.NUMA, nil, nodeName)
				Expect(err).NotTo(HaveOccurred())
			}

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
			for nodeName := range nodelabels {
				err = helpers.CheckLabelsExist(nodelabels, []string{"BMI2"}, nil, nodeName)
				Expect(err).NotTo(HaveOccurred())
			}

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
			for nodeName := range nodelabels {
				err = helpers.CheckLabelsExist(nodelabels, []string{"BMI2"}, cpuFlags[nodeName], nodeName)
				Expect(err).NotTo(HaveOccurred())
			}

		})
	})
})

func runNodeDiscoveryAndTestLabelExistence(nfdManager *nfdDeploy.NfdAPIResource, enableTopology bool) {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_, err := get.PodStatus(APIClient, ts.Namespace)
		glog.Error(err)

		return err
	})
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("error in deploying %s", err))

	err = nfdManager.DeployNfd(5*int(time.Minute), enableTopology, "")
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
