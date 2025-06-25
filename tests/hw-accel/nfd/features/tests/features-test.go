package tests

import (
	"context"
	"fmt"
	"strings"

	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/machine"
	Nodes "github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	nfdDeploy "github.com/openshift-kni/eco-gotests/tests/hw-accel/internal/deploy"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/internal/hwaccelparams"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/nfd/features/internal/helpers"
	ts "github.com/openshift-kni/eco-gotests/tests/hw-accel/nfd/features/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/nfd/internal/get"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/nfd/internal/nfdconfig"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/nfd/internal/nfddelete"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/nfd/internal/set"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/nfd/internal/wait"
	. "github.com/openshift-kni/eco-gotests/tests/internal/inittools"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
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
			runNodeDiscoveryAndTestLabelExistence(nfdManager, nfdConfig, true)

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
		It("Check if NFD Feature is deleted when node is deleted", reportxml.ID("68304"), func() {
			skipIfConfigNotSet(nfdConfig)

			if nfdConfig.SkipDestructiveNodeTests {
				Skip("Skipping destructive node deletion test. Set ECO_HWACCEL_NFD_SKIP_DESTRUCTIVE_NODE_TESTS=false to enable.")
			}

			By("Get initial node features")
			initialNodeFeatures, err := get.GetNodeFeatures(APIClient)
			glog.V(ts.LogLevel).Infof("Initial node features: %v", initialNodeFeatures)
			Expect(err).NotTo(HaveOccurred(), "Failed to get initial node features")

			initialFeatureCount := len(initialNodeFeatures)
			glog.V(ts.LogLevel).Infof("Initial node features count: %d", initialFeatureCount)

			By("Get current nodes")
			nodes, err := Nodes.List(APIClient, metav1.ListOptions{
				LabelSelector: labels.Set(GeneralConfig.WorkerLabelMap).String(),
			})
			Expect(err).NotTo(HaveOccurred(), "Failed to get nodes")
			glog.V(ts.LogLevel).Infof("Current nodes count: %d", len(nodes))
			if len(nodes) == 0 {
				Skip("No worker nodes found to test node deletion")
			}

			// Select the first worker node for deletion
			nodeToDelete := nodes[0]
			glog.V(ts.LogLevel).Infof("Selected node for deletion: %s", nodeToDelete.Object.Name)

			By("Delete the selected node")
			err = nodeToDelete.Delete()
			Expect(err).NotTo(HaveOccurred(), "Failed to delete node %s", nodeToDelete.Definition.Name)

			By("Wait for node deletion to complete")
			err = retry.OnError(retry.DefaultRetry, func(err error) bool {
				return true // Retry on any error
			}, func() error {
				_, err := APIClient.CoreV1Interface.Nodes().Get(
					context.Background(),
					nodeToDelete.Definition.Name,
					metav1.GetOptions{})
				if err != nil {
					// Node is deleted, which is what we want
					return nil
				}

				return fmt.Errorf("node %s still exists", nodeToDelete.Definition.Name)
			})

			Expect(err).NotTo(HaveOccurred(), "Node deletion did not complete within expected time")

			By("Wait for NFD features to be cleaned up")
			err = retry.OnError(retry.DefaultRetry, func(err error) bool {
				return true // Retry on any error
			}, func() error {
				currentNodeFeatures, err := get.GetNodeFeatures(APIClient)
				if err != nil {
					return err
				}

				currentFeatureCount := len(currentNodeFeatures)
				glog.V(ts.LogLevel).Infof("Current node features count: %d", currentFeatureCount)

				// Check if the number of features decreased (indicating cleanup)
				if currentFeatureCount < initialFeatureCount {
					return nil // Success - features were cleaned up
				}

				return fmt.Errorf("node features not yet cleaned up. Expected less than %d, got %d",
					initialFeatureCount, currentFeatureCount)
			})
			glog.V(ts.LogLevel).Infof("NFD features cleanup check completed: %v", err)
			Expect(err).NotTo(HaveOccurred(), "NFD features were not cleaned up after node deletion%v", err)

			By("Verify the deleted node's features are no longer present")
			finalNodeFeatures, err := get.GetNodeFeatures(APIClient)
			Expect(err).NotTo(HaveOccurred(), "Failed to get final node features")

			// Check that no features exist for the deleted node
			for _, feature := range finalNodeFeatures {
				Expect(feature.Name).NotTo(Equal(nodeToDelete.Definition.Name),
					"Found features for deleted node %s", nodeToDelete.Definition.Name)
			}

			glog.V(ts.LogLevel).Infof("Successfully verified NFD features cleanup after node deletion")
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

func runNodeDiscoveryAndTestLabelExistence(
	nfdManager *nfdDeploy.NfdAPIResource,
	nfdConfig *nfdconfig.NfdConfig,
	enableTopology bool) {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_, err := get.PodStatus(APIClient, hwaccelparams.NFDNamespace)
		glog.Error(err)

		return err
	})

	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("error in deploying %s", err))
	err = nfdManager.DeployNfd(15*int(time.Minute), enableTopology, nfdConfig.Image)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("error in deploying %s", err))
	By("Check that pods are in running state")

	res, err := wait.ForPodsRunning(APIClient, 15*time.Minute, hwaccelparams.NFDNamespace)
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
	}).WithTimeout(5 * time.Minute).ShouldNot(HaveLen(0))
}

func skipIfConfigNotSet(nfdConfig *nfdconfig.NfdConfig) {
	if nfdConfig.CatalogSource == "" {
		Skip("The catalog source is not set.")
	}
}
