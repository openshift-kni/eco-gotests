package tests

import (
	"context"
	"encoding/json"
	"flag"
	"strings"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/configmap"
	"github.com/openshift-kni/eco-goinfra/pkg/deployment"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/nvidiagpu"
	"github.com/openshift-kni/eco-goinfra/pkg/olm"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/nvidiagpu/gpudeploy/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/nvidiagpu/internal/check"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/nvidiagpu/internal/get"
	gpuburn "github.com/openshift-kni/eco-gotests/tests/hw-accel/nvidiagpu/internal/gpu-burn"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/nvidiagpu/internal/gpuparams"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/nvidiagpu/internal/wait"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/openshift-kni/eco-gotests/tests/internal/inittools"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
)

var workerNodeSelector = map[string]string{
	"node-role.kubernetes.io/worker": ""}

const (
	nvidiaGPUNamespace        = "nvidia-gpu-operator"
	nfdRhcosLabel             = "feature.node.kubernetes.io/system-os_release.ID"
	nfdRhcosLabelValue        = "rhcos"
	nvidiaGPULabel            = "feature.node.kubernetes.io/pci-10de.present"
	gpuOperatorGroupName      = "gpu-og"
	gpuSubscriptionName       = "gpu-subscription"
	gpuSubscriptionNamespace  = "nvidia-gpu-operator"
	gpuCatalogSource          = "certified-operators"
	gpuCatalogSourceNamespace = "openshift-marketplace"
	gpuPackage                = "gpu-operator-certified"
	gpuClusterPolicyName      = "gpu-cluster-policy"
	gpuBurnNamespace          = "test-gpu-burn"
	gpuBurnPodName            = "gpu-burn-pod"
	gpuBurnPodLabel           = "app=gpu-burn-app"
	gpuBurnImageName          = "quay.io/wabouham/gpu_burn_amd64:ubi9"
	gpuBurnConfigmapName      = "gpu-burn-entrypoint"
)

var gpuInstallPlanApproval v1alpha1.Approval = "Automatic"

var _ = Describe("GPU", Ordered, Label(tsparams.LabelSuite), func() {

	Context("DeployGpu", Label("deploy-gpu-with-dtk"), func() {

		BeforeAll(func() {
			// Init glog lib and dump logs to stdout
			_ = flag.Lookup("logtostderr").Value.Set("true")
			// _ = flag.Lookup("v").Value.Set(GeneralConfig.VerboseLevel)
			_ = flag.Lookup("v").Value.Set("100")
		})

		// "feature.node.kubernetes.io/system-os_release.ID=rhcos"
		BeforeEach(func() {
			By("Check if NFD is installed")
			nfdLabelDetected, err := check.AllNodeLabel(APIClient, nfdRhcosLabel, nfdRhcosLabelValue, workerNodeSelector)
			Expect(err).ToNot(HaveOccurred(), "error calling check.NodeLabel:  %v ", err)
			Expect(nfdLabelDetected).NotTo(BeFalse(), "NFD node label check failed to match "+
				"label %s and label value %s on all nodes", nfdRhcosLabel, nfdRhcosLabelValue)
			// expect if it returns true
			glog.V(gpuparams.GpuLogLevel).Infof("The check for NFD label returned: %v", nfdLabelDetected)

			// Note for later:  decide if you want to deploy NFD if not deployed
			nfdInstalled, err := check.NFDDeploymentsReady(APIClient)
			Expect(err).ToNot(HaveOccurred(), "error checking if NFD deployments are ready:  %v ", err)
			glog.V(gpuparams.GpuLogLevel).Infof("The check for NFD label returned: %v", nfdInstalled)

			By("Check if at least one worker node is GPU enabled")
			gpuNodeFound, err := check.NodeWithLabel(APIClient, nvidiaGPULabel, workerNodeSelector)
			Expect(err).ToNot(HaveOccurred(), "error checking if one node is GPU enabled:  %v ", err)
			glog.V(gpuparams.GpuLogLevel).Infof("The check for Nvidia GPU label returned: %v", gpuNodeFound)

		})

		AfterEach(func() {
			// Later use this for clean up corner cases
		})

		It("Deploy NVIDIA GPU Operator with DTK without cluster-wide entitlement",
			polarion.ID("48452"), func() {

				By("Check if 'gpu-operator-certified' packagemanifest exists in 'certified-operators' catalog")
				gpuPkgManifestBuilderByCatalog, err := olm.PullPackageManifestByCatalog(APIClient,
					gpuPackage, gpuCatalogSourceNamespace, gpuCatalogSource)
				Expect(err).ToNot(HaveOccurred(), "error getting GPU packagemanifest %s from catalog %s:"+
					"  %v", gpuPackage, gpuCatalogSource, err)

				glog.V(gpuparams.GpuLogLevel).Infof("The gpu packagemanifest name returned: %s",
					gpuPkgManifestBuilderByCatalog.Object.Name)

				gpuChannel := gpuPkgManifestBuilderByCatalog.Object.Status.DefaultChannel
				glog.V(gpuparams.GpuLogLevel).Infof("The gpu channel retrieved from packagemanifest is:  %v",
					gpuChannel)

				By("Check if NVIDIA GPU Operator namespace exists, otherwise created it")
				// APIClient comes from package inittools
				nsBuilder := namespace.NewBuilder(APIClient, nvidiaGPUNamespace)
				if nsBuilder.Exists() {
					glog.V(gpuparams.GpuLogLevel).Infof("The namespace '%s' already exists",
						nsBuilder.Object.Name)
				} else {
					// create the namespace
					glog.V(gpuparams.GpuLogLevel).Infof("Creating the namespace:  %v", nvidiaGPUNamespace)
					createdNsBuilder, err := nsBuilder.Create()
					Expect(err).ToNot(HaveOccurred(), "error creating namespace '%s' :  %v ",
						nsBuilder.Definition.Name, err)

					glog.V(gpuparams.GpuLogLevel).Infof("Successfully created namespace '%s'",
						createdNsBuilder.Object.Name)

					glog.V(gpuparams.GpuLogLevel).Infof("Labeling the newly created namespace '%s'",
						nsBuilder.Object.Name)

					labeledNsBuilder := createdNsBuilder.WithMultipleLabels(map[string]string{
						"openshift.io/cluster-monitoring":    "true",
						"pod-security.kubernetes.io/enforce": "privileged",
					})

					newLabeledNsBuilder, err := labeledNsBuilder.Update()
					Expect(err).ToNot(HaveOccurred(), "error labeling namespace %v :  %v ",
						newLabeledNsBuilder.Definition.Name, err)

					glog.V(gpuparams.GpuLogLevel).Infof("The nvidia-gpu-operator labeled namespace has "+
						"labels:  %v", newLabeledNsBuilder.Object.Labels)

				}

				defer func() {
					err := nsBuilder.Delete()
					Expect(err).ToNot(HaveOccurred())
				}()

				By("Create OperatorGroup in NVIDIA GPU Operator Namespace")
				ogBuilder := olm.NewOperatorGroupBuilder(APIClient, gpuOperatorGroupName, nvidiaGPUNamespace)
				if ogBuilder.Exists() {
					glog.V(gpuparams.GpuLogLevel).Infof("The ogBuilder that exists has name:  %v",
						ogBuilder.Object.Name)
				} else {
					glog.V(gpuparams.GpuLogLevel).Infof("Create a new operatorgroup with name:  %v",
						ogBuilder.Object.Name)

					ogBuilderCreated, err := ogBuilder.Create()
					Expect(err).ToNot(HaveOccurred(), "error creating operatorgroup %v :  %v ",
						ogBuilderCreated.Definition.Name, err)

				}

				defer func() {
					err := ogBuilder.Delete()
					Expect(err).ToNot(HaveOccurred())
				}()

				By("Create Subscription in NVIDIA GPU Operator Namespace")
				subBuilder := olm.NewSubscriptionBuilder(APIClient, gpuSubscriptionName, gpuSubscriptionNamespace,
					gpuCatalogSource, gpuCatalogSourceNamespace, gpuPackage)

				subBuilder.WithChannel(gpuChannel)
				// need to determine efficient way to extract this from the packagemanifest
				// subBuilder.WithStartingCSV(gpuStartingCSV)
				subBuilder.WithInstallPlanApproval(gpuInstallPlanApproval)

				glog.V(gpuparams.GpuLogLevel).Infof("Creating the subscription, i.e Deploy the GPU operator")
				createdSub, err := subBuilder.Create()

				Expect(err).ToNot(HaveOccurred(), "error creating subscription %v :  %v ",
					createdSub.Definition.Name, err)

				glog.V(gpuparams.GpuLogLevel).Infof("Newly created subscription: %s was successfully created",
					createdSub.Object.Name)

				if createdSub.Exists() {
					glog.V(gpuparams.GpuLogLevel).Infof("The newly created subscription: %s in namespace: %v "+
						"has starting CSV:  %v", createdSub.Object.Name, createdSub.Object.Namespace,
						createdSub.Definition.Spec.StartingCSV)
				}

				defer func() {
					err := createdSub.Delete()
					Expect(err).ToNot(HaveOccurred())
				}()

				By("Sleeping for 15 seconds to allow NVIDIA GPU Operator deployment to be created")
				glog.V(gpuparams.GpuLogLevel).Infof("Sleeping for 15 seconds to allow operator deployment " +
					"pod to be created")
				time.Sleep(15 * time.Second)

				By("Check if the GPU operator deployment is ready")
				gpuOperatorDeployment, err := deployment.Pull(APIClient, "gpu-operator", nvidiaGPUNamespace)

				Expect(err).ToNot(HaveOccurred(), "Error trying to pull GPU operator "+
					"deployment is: %v", err)

				glog.V(gpuparams.GpuLogLevel).Infof("Pulled GPU operator deployment is:  %v ",
					gpuOperatorDeployment.Definition.Name)

				if gpuOperatorDeployment.IsReady(180 * time.Second) {
					glog.V(gpuparams.GpuLogLevel).Infof("Pulled GPU operator deployment is:  %v is Ready ",
						gpuOperatorDeployment.Definition.Name)
				}

				By("Get currentCSV from subscription")
				gpuCurrentCSVFromSub, err := get.CurrentCSVFromSubscription(APIClient, gpuSubscriptionName,
					gpuSubscriptionNamespace)
				Expect(err).ToNot(HaveOccurred(), "error pulling currentCSV from cluster:  %v", err)
				Expect(gpuCurrentCSVFromSub).ToNot(BeEmpty())
				glog.V(gpuparams.GpuLogLevel).Infof("currentCSV %s extracted from Subscripion %s",
					gpuCurrentCSVFromSub, gpuSubscriptionName)

				By("Wait for ClusterServiceVersion to be in Succeeded phase")
				glog.V(gpuparams.GpuLogLevel).Infof("Waiting for ClusterServiceVersion to be Succeeded phase")
				err = wait.CSVSucceeded(APIClient, gpuCurrentCSVFromSub, nvidiaGPUNamespace, 60*time.Second,
					5*time.Minute)
				glog.V(gpuparams.GpuLogLevel).Infof("error waiting for ClusterServiceVersion to be "+
					"in Succeeded phase:  %v ", err)
				Expect(err).ToNot(HaveOccurred(), "error waiting for ClusterServiceVersion to be "+
					"in Succeeded phase:  %v ", err)

				By("Pull existing CSV in NVIDIA GPU Operator Namespace")
				clusterCSV, err := olm.PullClusterServiceVersion(APIClient, gpuCurrentCSVFromSub, nvidiaGPUNamespace)
				Expect(err).ToNot(HaveOccurred(), "error pulling CSV %v from cluster:  %v",
					gpuCurrentCSVFromSub, err)

				glog.V(gpuparams.GpuLogLevel).Infof("clusterCSV from cluster lastUpdatedTime is : %v ",
					clusterCSV.Definition.Status.LastUpdateTime)

				glog.V(gpuparams.GpuLogLevel).Infof("clusterCSV from cluster Phase is : \"%v\"",
					clusterCSV.Definition.Status.Phase)

				succeeded := v1alpha1.ClusterServiceVersionPhase("Succeeded")
				Expect(clusterCSV.Definition.Status.Phase).To(Equal(succeeded), "CSV Phase is not "+
					"succeeded")

				defer func() {
					err := clusterCSV.Delete()
					Expect(err).ToNot(HaveOccurred())
				}()

				By("Get ALM examples block form CSV")
				almExamples, err := clusterCSV.GetAlmExamples()
				Expect(err).ToNot(HaveOccurred(), "Error from pulling almExamples from csv "+
					"from cluster:  %v ", err)
				glog.V(gpuparams.GpuLogLevel).Infof("almExamples block from clusterCSV  is : %v ", almExamples)

				By("Deploy ClusterPolicy")
				glog.V(gpuparams.GpuLogLevel).Infof("Creating ClusterPolicy from CSV almExamples")
				clusterPolicyBuilder := nvidiagpu.NewBuilderFromObjectString(APIClient, almExamples)
				createdClusterPolicyBuilder, err := clusterPolicyBuilder.Create()
				Expect(err).ToNot(HaveOccurred(), "Error Creating ClusterPolicy from csv "+
					"almExamples  %v ", err)
				glog.V(gpuparams.GpuLogLevel).Infof("ClusterPolicy '%' successfully created",
					createdClusterPolicyBuilder.Definition.Name)

				defer func() {
					_, err := createdClusterPolicyBuilder.Delete()
					Expect(err).ToNot(HaveOccurred())
				}()

				By("Pull the ClusterPolicy just created from cluster, with updated fields")
				pulledClusterPolicy, err := nvidiagpu.Pull(APIClient, gpuClusterPolicyName)
				Expect(err).ToNot(HaveOccurred(), "error pulling ClusterPolicy %s from cluster: "+
					" %v ", gpuClusterPolicyName, err)

				cpJSON, err := json.MarshalIndent(pulledClusterPolicy, "", " ")

				if err == nil {
					glog.V(gpuparams.GpuLogLevel).Infof("The ClusterPolicy just created has name:  %v",
						pulledClusterPolicy.Definition.Name)
					glog.V(gpuparams.GpuLogLevel).Infof("The ClusterPolicy just created marshalled "+
						"in json: %v", string(cpJSON))
				} else {
					glog.V(gpuparams.GpuLogLevel).Infof("Error Marshalling ClusterPolicy into json:  %v",
						err)
				}

				By("Wait up to 12 minutes for ClusterPolicy to be ready")
				glog.V(gpuparams.GpuLogLevel).Infof("Waiting for ClusterPolicy to be ready")
				err = wait.ClusterPolicyReady(APIClient, gpuClusterPolicyName, 60*time.Second, 12*time.Minute)

				glog.V(gpuparams.GpuLogLevel).Infof("error waiting for ClusterPolicy to be Ready:  %v ", err)
				Expect(err).ToNot(HaveOccurred(), "error waiting for ClusterPolicy to be Ready:  %v ",
					err)

				By("Pull the ready ClusterPolicy from cluster, with updated fields")
				pulledReadyClusterPolicy, err := nvidiagpu.Pull(APIClient, gpuClusterPolicyName)
				Expect(err).ToNot(HaveOccurred(), "error pulling ClusterPolicy %s from cluster: "+
					" %v ", gpuClusterPolicyName, err)

				cpReadyJSON, err := json.MarshalIndent(pulledReadyClusterPolicy, "", " ")

				if err == nil {
					glog.V(gpuparams.GpuLogLevel).Infof("The ready ClusterPolicy just has name:  %v",
						pulledReadyClusterPolicy.Definition.Name)
					glog.V(gpuparams.GpuLogLevel).Infof("The ready ClusterPolicy just marshalled "+
						"in json: %v", string(cpReadyJSON))
				} else {
					glog.V(gpuparams.GpuLogLevel).Infof("Error Marshalling the ready ClusterPolicy into json:  %v",
						err)
				}

				By("Create GPU Burn namespace 'test-gpu-burn'")
				gpuBurnNsBuilder := namespace.NewBuilder(APIClient, gpuBurnNamespace)
				if gpuBurnNsBuilder.Exists() {
					glog.V(gpuparams.GpuLogLevel).Infof("The namespace '%s' already exists",
						gpuBurnNsBuilder.Object.Name)
				} else {
					glog.V(gpuparams.GpuLogLevel).Infof("Creating the gpu burn namespace '%s'",
						gpuBurnNamespace)
					createdGPUBurnNsBuilder, err := gpuBurnNsBuilder.Create()
					Expect(err).ToNot(HaveOccurred(), "error creating gpu burn "+
						"namespace '%s' :  %v ", gpuBurnNamespace, err)

					glog.V(gpuparams.GpuLogLevel).Infof("Successfully created namespace '%s'",
						createdGPUBurnNsBuilder.Object.Name)

					glog.V(gpuparams.GpuLogLevel).Infof("Labeling the newly created namespace '%s'",
						createdGPUBurnNsBuilder.Object.Name)

					labeledGPUBurnNsBuilder := createdGPUBurnNsBuilder.WithMultipleLabels(map[string]string{
						"openshift.io/cluster-monitoring":    "true",
						"pod-security.kubernetes.io/enforce": "privileged",
					})

					newGPUBurnLabeledNsBuilder, err := labeledGPUBurnNsBuilder.Update()
					Expect(err).ToNot(HaveOccurred(), "error labeling namespace %v :  %v ",
						newGPUBurnLabeledNsBuilder.Definition.Name, err)

					glog.V(gpuparams.GpuLogLevel).Infof("The nvidia-gpu-operator labeled namespace has "+
						"labels:  %v", newGPUBurnLabeledNsBuilder.Object.Labels)
				}

				defer func() {
					err := gpuBurnNsBuilder.Delete()
					Expect(err).ToNot(HaveOccurred())
				}()

				By("Deploy GPU Burn configmap in test-gpu-burn namespace")
				gpuBurnConfigMap, err := gpuburn.CreateGPUBurnConfigMap(APIClient, gpuBurnConfigmapName,
					gpuBurnNamespace)
				Expect(err).ToNot(HaveOccurred(), "Error Creating gpu burn configmap: %v", err)

				glog.V(gpuparams.GpuLogLevel).Infof("The created gpuBurnConfigMap has name: %s",
					gpuBurnConfigMap.Name)

				configmapBuilder, err := configmap.Pull(APIClient, gpuBurnConfigmapName, gpuBurnNamespace)
				Expect(err).ToNot(HaveOccurred(), "Error pulling gpu-burn configmap '%s' from "+
					"namespace '%s': %v", gpuBurnConfigmapName, gpuBurnNamespace, err)

				glog.V(gpuparams.GpuLogLevel).Infof("The pulled gpuBurnConfigMap has name: %s",
					configmapBuilder.Definition.Name)

				defer func() {
					err := configmapBuilder.Delete()
					Expect(err).ToNot(HaveOccurred())
				}()

				By("Deploy gpu-burn pod in test-gpu-burn namespace")
				gpuBurnPod, err := gpuburn.CreateGPUBurnPod(APIClient, gpuBurnPodName, gpuBurnNamespace,
					gpuBurnImageName, 5*time.Minute)
				Expect(err).ToNot(HaveOccurred(), "Error creating gpu burn pod: %v", err)

				glog.V(gpuparams.GpuLogLevel).Infof("Creating gpu-burn pod '%s' in namespace '%s'",
					gpuBurnPodName, gpuBurnNamespace)

				_, err = APIClient.Pods(gpuBurnPod.Namespace).Create(context.TODO(), gpuBurnPod, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred(), "Error creating gpu-burn '%s' in "+
					"namespace '%s': %v", gpuBurnPodName, gpuBurnNamespace, err)

				glog.V(gpuparams.GpuLogLevel).Infof("The created gpuBurnPod has name: %s has status: %v ",
					gpuBurnPod.Name, gpuBurnPod.Status)

				By("Get the gpu-burn pod with label \"app=gpu-burn-app\"")
				gpuPodName, err := get.GetFirstPodNameWithLabel(APIClient, gpuBurnNamespace, gpuBurnPodLabel)
				Expect(err).ToNot(HaveOccurred(), "error getting gpu-burn pod with label "+
					"'app=gpu-burn-app' from namespace '%s' :  %v ", gpuBurnNamespace, err)
				glog.V(gpuparams.GpuLogLevel).Infof("gpuPodName is %s ", gpuPodName)

				By("Pull the gpu-burn pod object from the cluster")
				gpuPodPulled, err := pod.Pull(APIClient, gpuPodName, gpuBurnNamespace)
				Expect(err).ToNot(HaveOccurred(), "error pulling gpu-burn pod from "+
					"namespace '%s' :  %v ", gpuBurnNamespace, err)

				defer func() {
					_, err := gpuPodPulled.Delete()
					Expect(err).ToNot(HaveOccurred())
				}()

				By("Wait for up to 3 minutes for gpu-burn pod to be in Running phase")
				err = gpuPodPulled.WaitUntilInStatus(corev1.PodRunning, 3*time.Minute)
				Expect(err).ToNot(HaveOccurred(), "timeout waiting for gpu-burn pod in "+
					"namespace '%s' to go to Running phase:  %v ", gpuBurnNamespace, err)
				glog.V(gpuparams.GpuLogLevel).Infof("gpu-burn pod now in Running phase")

				By("Wait for gpu-burn pod to run to completion and be in Succeeded phase/Completed status")
				err = gpuPodPulled.WaitUntilInStatus(corev1.PodSucceeded, 8*time.Minute)
				Expect(err).ToNot(HaveOccurred(), "timeout waiting for gpu-burn pod '%s' in "+
					"namespace '%s'to go Succeeded phase/Completed status:  %v ", gpuBurnPodName, gpuBurnNamespace, err)
				glog.V(gpuparams.GpuLogLevel).Infof("gpu-burn pod now in Succeeded Phase/Completed status")

				By("Get the gpu-burn pod logs")
				glog.V(gpuparams.GpuLogLevel).Infof("Get the gpu-burn pod logs")

				gpuBurnLogs, err := gpuPodPulled.GetLog(500*time.Second, "gpu-burn-ctr")

				Expect(err).ToNot(HaveOccurred(), "error getting gpu-burn pod '%s' logs "+
					"from gpu burn namespace '%s' :  %v ", gpuBurnNamespace, err)
				glog.V(gpuparams.GpuLogLevel).Infof("Gpu-burn pod '%s' logs:\n%s",
					gpuPodPulled.Definition.Name, gpuBurnLogs)

				By("Parse the gpu-burn pod logs and check for successful execution")
				match1 := strings.Contains(gpuBurnLogs, "GPU 0: OK")
				match2 := strings.Contains(gpuBurnLogs, "100.0%  proc'd:")

				Expect(match1 && match2).ToNot(BeFalse(), "gpu-burn pod execution was FAILED")
				glog.V(gpuparams.GpuLogLevel).Infof("Gpu-burn pod execution was successful")
			})
	})
})
