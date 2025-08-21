package tests

import (
	"fmt"
	"strings"

	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/machine"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/pod"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/reportxml"
	nfdDeploy "github.com/rh-ecosystem-edge/eco-gotests/tests/hw-accel/internal/deploy"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/hw-accel/internal/hwaccelparams"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/hw-accel/nfd/features/internal/helpers"
	ts "github.com/rh-ecosystem-edge/eco-gotests/tests/hw-accel/nfd/features/internal/tsparams"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/hw-accel/nfd/internal/get"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/hw-accel/nfd/internal/nfdconfig"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/hw-accel/nfd/internal/nfddelete"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/hw-accel/nfd/internal/set"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/hw-accel/nfd/internal/wait"
	. "github.com/rh-ecosystem-edge/eco-gotests/tests/internal/inittools"
	"k8s.io/client-go/util/retry"
)

var _ = Describe("NFD", Ordered, func() {
	nfdConfig := nfdconfig.NewNfdConfig()
	nfdManager := nfdDeploy.NewNfdAPIResource(APIClient,
		hwaccelparams.NFDNamespace,
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
			By("Clear labels")
			err := nfddelete.NfdLabelsByKeys(APIClient, "nfd.node.kubernetes.io", "feature.node.kubernetes.io")
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("error in cleaning labels\n %s", err))

			By("Creating nfd")
			runNodeDiscoveryAndTestLabelExistence(nfdManager, true)

			labelExist, labelsError := wait.ForLabel(APIClient, 15*time.Minute, "feature")
			if !labelExist || labelsError != nil {
				glog.Error("feature labels was not found in the given time error=%v", labelsError)
			}

		})

		It("Check pods state", reportxml.ID("54548"), func() {
			err := helpers.CheckPodStatus(APIClient)
			Expect(err).NotTo(HaveOccurred())

		})
		It("Check CPU feature labels", reportxml.ID("54222"), func() {
			skipIfConfigNotSet(nfdConfig)

			if nfdConfig.CPUFlagsHelperImage == "" {
				Skip("CPUFlagsHelperImage is not set.")
			}
			cpuFlags = get.CPUFlags(APIClient, hwaccelparams.NFDNamespace, nfdConfig.CPUFlagsHelperImage)
			nodelabels, err := get.NodeFeatureLabels(APIClient, GeneralConfig.WorkerLabelMap)

			Expect(err).NotTo(HaveOccurred())

			By("Check if features exists")

			for nodeName := range nodelabels {
				err = helpers.CheckLabelsExist(nodelabels, cpuFlags[nodeName], nil, nodeName)
				Expect(err).NotTo(HaveOccurred())
			}

		})

		It("Check Kernel config", reportxml.ID("54471"), func() {
			skipIfConfigNotSet(nfdConfig)
			nodelabels, err := get.NodeFeatureLabels(APIClient, GeneralConfig.WorkerLabelMap)
			Expect(err).NotTo(HaveOccurred())

			By("Check if custom label topology is exist")
			for nodeName := range nodelabels {
				err = helpers.CheckLabelsExist(nodelabels, ts.KernelConfig, nil, nodeName)
				Expect(err).NotTo(HaveOccurred())
			}

		})

		It("Check topology", reportxml.ID("54491"), func() {
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
		It("Check Logs", reportxml.ID("54549"), func() {
			errorKeywords := []string{"error", "exception", "failed"}
			skipIfConfigNotSet(nfdConfig)
			listOptions := metav1.ListOptions{
				AllowWatchBookmarks: false,
			}
			By("Check if NFD pod's log not contains in error messages")
			pods, err := pod.List(APIClient, hwaccelparams.NFDNamespace, listOptions)
			Expect(err).NotTo(HaveOccurred())
			for _, p := range pods {
				glog.V(ts.LogLevel).Info("retrieve logs from %v", p.Object.Name)
				log, err := get.PodLogs(APIClient, hwaccelparams.NFDNamespace, p.Object.Name)
				Expect(err).NotTo(HaveOccurred(), "Error retrieving pod logs.")
				Expect(len(log)).NotTo(Equal(0))
				for _, errorKeyword := range errorKeywords {

					logLines := strings.Split(log, "\n")
					for _, line := range logLines {
						if strings.Contains(errorKeyword, line) {
							glog.Error("error found in log:", line)
						}
					}

				}

			}

		})

		It("Check Restart Count", reportxml.ID("54538"), func() {
			skipIfConfigNotSet(nfdConfig)
			listOptions := metav1.ListOptions{
				AllowWatchBookmarks: false,
			}
			By("Check if NFD pods reset count equal to zero")
			pods, err := pod.List(APIClient, hwaccelparams.NFDNamespace, listOptions)
			Expect(err).NotTo(HaveOccurred())
			for _, p := range pods {
				glog.V(ts.LogLevel).Info("retrieve reset count from %v.", p.Object.Name)
				resetCount, err := get.PodRestartCount(APIClient, hwaccelparams.NFDNamespace, p.Object.Name)
				Expect(err).NotTo(HaveOccurred(), "Error retrieving reset count.")
				glog.V(ts.LogLevel).Info("Total resets %d.", resetCount)
				Expect(resetCount).To(Equal(int32(0)))

			}
		})

		It("Check if NUMA detected ", reportxml.ID("54408"), func() {
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

		It("Verify Feature List not contains items from Blacklist ", reportxml.ID("68298"), func() {
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
				hwaccelparams.NFDNamespace,
				nfdConfig.Image)

			labelExist, labelsError := wait.ForLabel(APIClient, 15*time.Minute, "feature")
			if !labelExist || labelsError != nil {
				glog.Error("feature labels was not found in the given time error=%v", labelsError)
			}

			nodelabels, err := get.NodeFeatureLabels(APIClient, GeneralConfig.WorkerLabelMap)
			glog.V(ts.LogLevel).Info("Received nodelabel: %v", nodelabels)
			Expect(err).NotTo(HaveOccurred())
			By("Check if features exists")
			for nodeName := range nodelabels {
				err = helpers.CheckLabelsExist(nodelabels, []string{"BMI2"}, nil, nodeName)
				Expect(err).NotTo(HaveOccurred())
			}

		})

		It("Verify Feature List contains only Whitelist", reportxml.ID("68300"), func() {
			skipIfConfigNotSet(nfdConfig)

			if nfdConfig.CPUFlagsHelperImage == "" {
				Skip("CPUFlagsHelperImage is not set.")
			}
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
				hwaccelparams.NFDNamespace,
				nfdConfig.Image)

			labelExist, labelsError := wait.ForLabel(APIClient, time.Minute*15, "feature")
			if !labelExist || labelsError != nil {
				glog.Error("feature labels was not found in the given time error=%v", labelsError)
			}
			cpuFlags = get.CPUFlags(APIClient, hwaccelparams.NFDNamespace, nfdConfig.CPUFlagsHelperImage)
			nodelabels, err := get.NodeFeatureLabels(APIClient, GeneralConfig.WorkerLabelMap)
			Expect(err).NotTo(HaveOccurred())
			By("Check if features exists")
			for nodeName := range nodelabels {
				err = helpers.CheckLabelsExist(nodelabels, []string{"BMI2"}, cpuFlags[nodeName], nodeName)
				Expect(err).NotTo(HaveOccurred())
			}

		})

		It("Add day2 workers", reportxml.ID("54539"), func() {
			skipIfConfigNotSet(nfdConfig)
			if !nfdConfig.AwsTest {
				Skip("This test works only on AWS cluster." +
					"Set ECO_HWACCEL_NFD_AWS_TESTS=true when running NFD tests against AWS cluster. ")
			}

			if nfdConfig.CPUFlagsHelperImage == "" {
				Skip("CPUFlagsHelperImage is not set.")
			}
			By("Creating machine set")
			msBuilder := machine.NewSetBuilderFromCopy(APIClient, ts.MachineSetNamespace, ts.InstanceType,
				ts.WorkerMachineSetLabel, ts.Replicas)
			Expect(msBuilder).NotTo(BeNil(), "Failed to Initialize MachineSetBuilder from copy")

			By("Create the new MachineSet")
			createdMsBuilder, err := msBuilder.Create()

			Expect(err).ToNot(HaveOccurred(), "error creating a machineset: %v", err)

			pulledMachineSetBuilder, err := machine.PullSet(APIClient,
				createdMsBuilder.Definition.ObjectMeta.Name,
				ts.MachineSetNamespace)

			Expect(err).ToNot(HaveOccurred(), "error pulling machineset: %v", err)

			By("Wait on machineset to be ready")

			err = machine.WaitForMachineSetReady(APIClient, createdMsBuilder.Definition.ObjectMeta.Name,
				ts.MachineSetNamespace, 15*time.Minute)

			Expect(err).ToNot(HaveOccurred(),
				"Failed to detect at least one replica of MachineSet %s in Ready state during 15 min polling interval: %v",
				pulledMachineSetBuilder.Definition.ObjectMeta.Name,
				err)

			nodelabels, err := get.NodeFeatureLabels(APIClient, GeneralConfig.WorkerLabelMap)

			Expect(err).NotTo(HaveOccurred())

			By("check node readiness")

			isNodeReady, err := wait.ForNodeReadiness(APIClient, 10*time.Minute, GeneralConfig.WorkerLabelMap)
			Expect(err).NotTo(HaveOccurred())
			Expect(isNodeReady).To(BeTrue(), "the new node is not ready for use")

			By("Check if features exists")
			cpuFlags = get.CPUFlags(APIClient, hwaccelparams.NFDNamespace, nfdConfig.CPUFlagsHelperImage)
			for nodeName := range nodelabels {
				glog.V(ts.LogLevel).Infof("checking labels in %v", nodeName)
				err = helpers.CheckLabelsExist(nodelabels, cpuFlags[nodeName], nil, nodeName)
				Expect(err).NotTo(HaveOccurred())
			}
			defer func() {
				err := pulledMachineSetBuilder.Delete()
				Expect(err).ToNot(HaveOccurred())
			}()

		})
	})
})

func runNodeDiscoveryAndTestLabelExistence(nfdManager *nfdDeploy.NfdAPIResource, enableTopology bool) {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_, err := get.PodStatus(APIClient, hwaccelparams.NFDNamespace)
		glog.Error(err)

		return err
	})

	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("error in deploying %s", err))

	err = nfdManager.DeployNfd(15*int(time.Minute), enableTopology, "")
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("error in deploying %s", err))
	By("Check that pods are in running state")

	res, err := wait.ForPod(APIClient, hwaccelparams.NFDNamespace)
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
