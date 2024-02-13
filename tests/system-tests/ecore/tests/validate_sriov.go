package ecore_system_test

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/configmap"
	"github.com/openshift-kni/eco-goinfra/pkg/deployment"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-goinfra/pkg/rbac"
	"github.com/openshift-kni/eco-goinfra/pkg/serviceaccount"
	"github.com/openshift-kni/eco-goinfra/pkg/sriov"
	multus "gopkg.in/k8snetworkplumbingwg/multus-cni.v4/pkg/types"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/openshift-kni/eco-gotests/tests/system-tests/ecore/internal/ecoreinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/ecore/internal/ecoreparams"
)

var _ = Describe(
	"ECore SR-IOV Definitions",
	Ordered,
	ContinueOnFailure,
	Label(ecoreparams.LabelEcoreValidateSriov), func() {
		Describe("Verify SR-IOV networks", func() {
			It("Asserts SR-IOV network(s) exist", func() {

				glog.V(ecoreparams.ECoreLogLevel).Infof(fmt.Sprintf("Listing SR-IOV networks in %q ns",
					ecoreparams.SRIOVOperatorNS))
				sriovNetworks, err := sriov.List(APIClient, ecoreparams.SRIOVOperatorNS, metav1.ListOptions{})
				Expect(err).ToNot(HaveOccurred(), "Failed to list SR-IOV networks")
				Expect(len(sriovNetworks)).ToNot(Equal(0), "Found 0 SR-IOV networks")
				for _, net := range sriovNetworks {
					glog.V(ecoreparams.ECoreLogLevel).Infof(fmt.Sprintf("SR-IOV network %q found", net.Definition.Name))
				}
			})
		}) // Verify SR-IOV networks

		Describe("SR-IOV Workloads", Label("ecore_sriov_workloads"), func() {

			var cmBuilder *configmap.Builder
			var deploySa *serviceaccount.Builder
			var crbSa *rbac.ClusterRoleBindingBuilder

			const labelsWlkdOneString = "systemtest-test=ecore-sriov-one"
			const labelsWlkdTwoString = "systemtest-test=ecore-sriov-two"

			BeforeAll(func() {
				glog.V(ecoreparams.ECoreLogLevel).Infof("Assert ConfigMap %q exists in %q namespace",
					ECoreConfig.WlkdSRIOVConfigMapNamePCG, ECoreConfig.NamespacePCG)

				if cmBuilder, err := configmap.Pull(
					APIClient,
					ECoreConfig.WlkdSRIOVConfigMapNamePCG,
					ECoreConfig.NamespacePCG); err == nil {
					glog.V(ecoreparams.ECoreLogLevel).Infof("configMap found, delete it")

					err := cmBuilder.Delete()
					Expect(err).ToNot(HaveOccurred(),
						fmt.Sprintf("Failed to delete configMap %q", ECoreConfig.WlkdSRIOVConfigMapNamePCG))
				}

				glog.V(ecoreparams.ECoreLogLevel).Infof("Create ConfigMap %q in %q namespace",
					ECoreConfig.WlkdSRIOVConfigMapNamePCG, ECoreConfig.NamespacePCG)
				cmBuilder = configmap.NewBuilder(
					APIClient,
					ECoreConfig.WlkdSRIOVConfigMapNamePCG,
					ECoreConfig.NamespacePCG)
				cmBuilder.WithData(ECoreConfig.WlkdSRIOVConfigMapDataPCG)

				_, err := cmBuilder.Create()
				Expect(err).ToNot(HaveOccurred(), "Failed to create configMap")

				DeferCleanup(func() {
					err := cmBuilder.Delete()
					Expect(err).ToNot(HaveOccurred(),
						fmt.Sprintf("Failed to delete CM %q from %q ns",
							ECoreConfig.NADConfigMapPCCName, ECoreConfig.NamespacePCC))
				})

				By("Creating ServiceAccount for the deployment")
				glog.V(ecoreparams.ECoreLogLevel).Infof("Assert SA %q exists in %q namespace",
					ECoreConfig.WlkdSRIOVOneSa, ECoreConfig.NamespacePCG)

				if deploySa, err := serviceaccount.Pull(
					APIClient, ECoreConfig.WlkdSRIOVOneSa, ECoreConfig.NamespacePCG); err == nil {
					glog.V(ecoreparams.ECoreLogLevel).Infof("ServiceAccount %q found in %q namespace",
						ECoreConfig.WlkdSRIOVOneSa, ECoreConfig.NamespacePCG)
					err := deploySa.Delete()
					Expect(err).ToNot(HaveOccurred(),
						fmt.Sprintf("Failed to delete ServiceAccount %q from %q ns",
							ECoreConfig.WlkdSRIOVOneSa, ECoreConfig.NamespacePCG))
				}

				deploySa = serviceaccount.NewBuilder(
					APIClient, ECoreConfig.WlkdSRIOVOneSa, ECoreConfig.NamespacePCG)

				deploySa, err = deploySa.Create()
				Expect(err).ToNot(HaveOccurred(), "Failed to create ServiceAccount",
					deploySa.Definition.Name)

				DeferCleanup(func() {
					err := deploySa.Delete()
					Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to delete ServiceAccount %s",
						ECoreConfig.WlkdSRIOVOneSa))
				})

				By("Creating RBAC for SA")
				rbacName := "privileged-telco-sriov"

				glog.V(ecoreparams.ECoreLogLevel).Infof("Assert ClusterRoleBinding %q exists", rbacName)
				if crbSa, err := rbac.PullClusterRoleBinding(
					APIClient,
					rbacName); err == nil {
					glog.V(ecoreparams.ECoreLogLevel).Infof("ClusterRoleBinding %q found", rbacName)
					err := crbSa.Delete()
					Expect(err).ToNot(HaveOccurred(),
						fmt.Sprintf("Failed to delete ClusterRoleBinding %q", rbacName))
				}

				glog.V(ecoreparams.ECoreLogLevel).Infof("Creating ClusterRoleBinding %q", rbacName)
				crbSa = rbac.NewClusterRoleBindingBuilder(APIClient,
					rbacName,
					"system:openshift:scc:privileged",
					rbacv1.Subject{
						Name:      ECoreConfig.WlkdSRIOVOneSa,
						Kind:      "ServiceAccount",
						Namespace: ECoreConfig.NamespacePCG,
					})

				crbSa, err = crbSa.Create()
				Expect(err).ToNot(HaveOccurred(), "Failed to create ClusterRoleBinding")

				glog.V(ecoreparams.ECoreLogLevel).Infof("ClusterRoleBinding %q created %v", rbacName, crbSa)
				DeferCleanup(func() {
					err := crbSa.Delete()
					Expect(err).ToNot(HaveOccurred(),
						fmt.Sprintf("Failed to delete ClusterRoleBinding %q", rbacName))
				})

			}) // BeforeAll

			AfterAll(func(ctx SpecContext) {
				By("Removing test deployments")
				for _, dName := range []string{ECoreConfig.WlkdSRIOVDeployOneName, ECoreConfig.WlkdSRIOVDeployTwoName} {
					deploy, err := deployment.Pull(APIClient, dName, ECoreConfig.NamespacePCG)
					if deploy != nil && err == nil {
						glog.V(ecoreparams.ECoreLogLevel).Infof("Deployment %q found in %q namespace. Deleting...",
							deploy.Definition.Name, ECoreConfig.NamespacePCG)

						err := deploy.DeleteAndWait(300 * time.Second)
						Expect(err).ToNot(HaveOccurred(),
							fmt.Sprintf("failed to delete deployment %q", dName))

					}
				}

				By("Asserting pods from deployments are gone")
				labelsWlkdOne := labelsWlkdOneString
				labelsWlkdTwo := labelsWlkdTwoString

				for _, label := range []string{labelsWlkdOne, labelsWlkdTwo} {
					Eventually(func() bool {
						oldPods, _ := pod.List(APIClient, ECoreConfig.NamespacePCG,
							metav1.ListOptions{LabelSelector: label})

						return len(oldPods) == 0

					}, 6*time.Minute, 3*time.Second).WithContext(ctx).Should(BeTrue(), "pods matching label(s) still present")
				}
			})

			Context("Different SR-IOV networks", func() {
				It("Assert SR-IOV workloads on the same node", func(ctx SpecContext) {

					By("Checking SR-IOV deployments don't exist")
					for _, dName := range []string{ECoreConfig.WlkdSRIOVDeployOneName, ECoreConfig.WlkdSRIOVDeployTwoName} {
						deploy, err := deployment.Pull(APIClient, dName, ECoreConfig.NamespacePCG)
						if deploy != nil && err == nil {
							glog.V(ecoreparams.ECoreLogLevel).Infof("Deployment %q found in %q namespace. Deleting...",
								deploy.Definition.Name, ECoreConfig.NamespacePCG)

							err := deploy.DeleteAndWait(300 * time.Second)
							Expect(err).ToNot(HaveOccurred(),
								fmt.Sprintf("failed to delete deployment %q", dName))

						}
					}

					By("Asserting pods from deployments are gone")
					labelsWlkdOne := labelsWlkdOneString
					labelsWlkdTwo := labelsWlkdTwoString

					for _, label := range []string{labelsWlkdOne, labelsWlkdTwo} {
						Eventually(func() bool {
							oldPods, _ := pod.List(APIClient, ECoreConfig.NamespacePCG,
								metav1.ListOptions{LabelSelector: label})

							return len(oldPods) == 0

						}, 6*time.Minute, 3*time.Second).WithContext(ctx).Should(BeTrue(), "pods matching label(s) still present")
					}

					By("Defining container configuration")
					deployContainer := pod.NewContainerBuilder("one", ECoreConfig.WlkdSRIOVDeployOneImage,
						ECoreConfig.WlkdSRIOVDeployOneCmd)

					deployContainerTwo := pod.NewContainerBuilder("two", ECoreConfig.WlkdSRIOVDeployTwoImage,
						ECoreConfig.WlkdSRIOVDeployTwoCmd)

					By("Setting SecurityContext")
					var trueFlag = true
					userUID := new(int64)
					*userUID = 0

					secContext := &v1.SecurityContext{
						RunAsUser:  userUID,
						Privileged: &trueFlag,
						SeccompProfile: &v1.SeccompProfile{
							Type: v1.SeccompProfileTypeRuntimeDefault,
						},
						Capabilities: &v1.Capabilities{
							Add: []v1.Capability{"NET_RAW", "NET_ADMIN", "SYS_ADMIN", "IPC_LOCK"},
						},
					}

					By("Setting SecurityContext")
					deployContainer = deployContainer.WithSecurityContext(secContext)
					glog.V(ecoreparams.ECoreLogLevel).Infof("Container One definition: %#v", deployContainer)

					deployContainerTwo = deployContainerTwo.WithSecurityContext(secContext)
					glog.V(ecoreparams.ECoreLogLevel).Infof("Container Two definition: %#v", deployContainerTwo)

					By("Dropping ALL security capability")
					deployContainer = deployContainer.WithDropSecurityCapabilities([]string{"ALL"}, true)
					deployContainerTwo = deployContainerTwo.WithDropSecurityCapabilities([]string{"ALL"}, true)

					By("Adding VolumeMount to container")
					volMount := v1.VolumeMount{
						Name:      "configs",
						MountPath: "/opt/net/",
						ReadOnly:  false,
					}

					deployContainer = deployContainer.WithVolumeMount(volMount)
					deployContainerTwo = deployContainerTwo.WithVolumeMount(volMount)

					glog.V(ecoreparams.ECoreLogLevel).Infof("Container One definition: %#v", deployContainer)
					glog.V(ecoreparams.ECoreLogLevel).Infof("Container Two definition: %#v", deployContainerTwo)

					By("Obtaining container definition")
					deployContainerCfg, err := deployContainer.GetContainerCfg()
					Expect(err).ToNot(HaveOccurred(), "Failed to get container definition")

					deployContainerTwoCfg, err := deployContainerTwo.GetContainerCfg()
					Expect(err).ToNot(HaveOccurred(), "Failed to get container definition")

					By("Defining deployment configuration")
					deploy := deployment.NewBuilder(APIClient,
						ECoreConfig.WlkdSRIOVDeployOneName,
						ECoreConfig.NamespacePCG,
						map[string]string{"systemtest-test": "ecore-sriov-one"},
						deployContainerCfg)

					deployTwo := deployment.NewBuilder(APIClient,
						ECoreConfig.WlkdSRIOVDeployTwoName,
						ECoreConfig.NamespacePCG,
						map[string]string{"systemtest-test": "ecore-sriov-two"},
						deployContainerTwoCfg)

					By("Adding SR-IOV annotations")
					var networksOne, networksTwo []*multus.NetworkSelectionElement

					networksOne = append(networksOne,
						&multus.NetworkSelectionElement{
							Name: ECoreConfig.WlkdSRIOVNetOne})

					networksTwo = append(networksTwo,
						&multus.NetworkSelectionElement{
							Name: ECoreConfig.WlkdSRIOVNetTwo})

					glog.V(ecoreparams.ECoreLogLevel).Infof("SR-IOV networks: %#v", networksOne)
					glog.V(ecoreparams.ECoreLogLevel).Infof("SR-IOV networks: %#v", networksTwo)

					deploy = deploy.WithSecondaryNetwork(networksOne)
					deployTwo = deployTwo.WithSecondaryNetwork(networksTwo)

					glog.V(ecoreparams.ECoreLogLevel).Infof("SR-IOV deploy one: %#v",
						deploy.Definition.Spec.Template.ObjectMeta.Annotations)
					glog.V(ecoreparams.ECoreLogLevel).Infof("SR-IOV deploy two: %#v",
						deployTwo.Definition.Spec.Template.ObjectMeta.Annotations)

					By("Adding NodeSelector to the deployment")
					deploy = deploy.WithNodeSelector(ECoreConfig.WlkdSRIOVDeployOneSelector)
					deployTwo = deployTwo.WithNodeSelector(ECoreConfig.WlkdSRIOVDeployOneSelector)

					By("Adding Toleration")
					toleration := v1.Toleration{
						Key:      "sriov",
						Value:    "true",
						Effect:   v1.TaintEffectNoSchedule,
						Operator: v1.TolerationOpEqual,
					}

					deploy = deploy.WithToleration(toleration)
					deployTwo = deployTwo.WithToleration(toleration)

					glog.V(ecoreparams.ECoreLogLevel).Infof("SR-IOV Toleration: %#v",
						deploy.Definition.Spec.Template.Spec.Tolerations)
					glog.V(ecoreparams.ECoreLogLevel).Infof("SR-IOV Toleration: %#v",
						deployTwo.Definition.Spec.Template.Spec.Tolerations)

					By("Adding Volume to the deployment")
					volMode := new(int32)
					*volMode = 511

					volDefinition := v1.Volume{
						Name: "configs",
						VolumeSource: v1.VolumeSource{
							ConfigMap: &v1.ConfigMapVolumeSource{
								DefaultMode: volMode,
								LocalObjectReference: v1.LocalObjectReference{
									Name: ECoreConfig.WlkdSRIOVConfigMapNamePCG,
								},
							},
						},
					}

					deploy = deploy.WithVolume(volDefinition)
					deployTwo = deployTwo.WithVolume(volDefinition)

					glog.V(ecoreparams.ECoreLogLevel).Infof("SR-IOV One Volume:\n %v",
						deploy.Definition.Spec.Template.Spec.Volumes)
					glog.V(ecoreparams.ECoreLogLevel).Infof("SR-IOV Two Volume:\n %#v",
						deployTwo.Definition.Spec.Template.Spec.Volumes)

					By(fmt.Sprintf("Assigning ServiceAccount %q to the deployment", ECoreConfig.WlkdSRIOVOneSa))
					deploy = deploy.WithServiceAccountName(ECoreConfig.WlkdSRIOVOneSa)
					deployTwo = deployTwo.WithServiceAccountName(ECoreConfig.WlkdSRIOVOneSa)

					By("Setting Replicas count")
					deploy = deploy.WithReplicas(int32(1))
					deployTwo = deployTwo.WithReplicas(int32(1))

					By("Creating a deployment")
					deploy, err = deploy.CreateAndWaitUntilReady(60 * time.Second)
					Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to create deployment %s: %v", deploy.Definition.Name, err))

					deployTwo, err = deployTwo.CreateAndWaitUntilReady(60 * time.Second)
					Expect(err).ToNot(HaveOccurred(),
						fmt.Sprintf("Failed to create deployment %s: %v", deployTwo.Definition.Name, err))

					By("Getting pods backed by deployment")
					podOneSelector := metav1.ListOptions{
						LabelSelector: labelsWlkdOneString,
					}

					podOneList, err := pod.List(APIClient, ECoreConfig.NamespacePCG, podOneSelector)
					Expect(err).ToNot(HaveOccurred(), "Failed to find pods for deployment one")
					Expect(len(podOneList)).To(Equal(1), "Expected only one pod")

					podOne := podOneList[0]
					glog.V(ecoreparams.ECoreLogLevel).Infof("Pod one is %v on node %s",
						podOne.Definition.Name, podOne.Definition.Spec.NodeName)

					podTwoSelector := metav1.ListOptions{
						LabelSelector: labelsWlkdTwoString,
					}

					podTwoList, err := pod.List(APIClient, ECoreConfig.NamespacePCG, podTwoSelector)
					Expect(err).ToNot(HaveOccurred(), "Failed to find pods for deployment two")
					Expect(len(podTwoList)).To(Equal(1), "Expected only two pod")

					podTwo := podTwoList[0]
					glog.V(ecoreparams.ECoreLogLevel).Infof("Pod two is %v on node %s",
						podTwo.Definition.Name, podTwo.Definition.Spec.NodeName)

					By("Sending data from pod one to pod two")
					msgOne := fmt.Sprintf("Running from pod %s(%s) at %d",
						podOne.Definition.Name,
						podOne.Definition.Spec.NodeName,
						time.Now().Unix())

					glog.V(ecoreparams.ECoreLogLevel).Infof("Sending msg %q from pod %s",
						msgOne, podOne.Definition.Name)

					sendDataOneCmd := []string{"/bin/bash", "-c",
						fmt.Sprintf("echo '%s' | nc 10.46.126.77 2222", msgOne)}

					podOneResult, err := podOne.ExecCommand(sendDataOneCmd, "one")
					Expect(err).ToNot(HaveOccurred(),
						fmt.Sprintf("Failed to send data from pod %s", podOne.Definition.Name))
					glog.V(ecoreparams.ECoreLogLevel).Infof("Result: %v - %s", podOneResult, &podOneResult)

					logStartTimestamp, err := time.ParseDuration("5s")
					Expect(err).ToNot(HaveOccurred(), "Failed to parse time duration")

					podTwoLog, err := podTwo.GetLog(logStartTimestamp, "two")
					Expect(err).ToNot(HaveOccurred(),
						fmt.Sprintf("Failed to get logs from pod %s", podTwo.Definition.Name))

					glog.V(ecoreparams.ECoreLogLevel).Infof("Logs from pod %s:\n%s",
						podTwo.Definition.Name, podTwoLog)
					Expect(podTwoLog).Should(ContainSubstring(msgOne))

					By("Sending data from pod two to pod one")
					msgTwo := fmt.Sprintf("Running from pod %s(%s) at %d",
						podTwo.Definition.Name,
						podTwo.Definition.Spec.NodeName,
						time.Now().Unix())

					glog.V(ecoreparams.ECoreLogLevel).Infof("Sending msg %q from pod %s",
						msgTwo, podTwo.Definition.Name)

					sendDataTwoCmd := []string{"/bin/bash", "-c",
						fmt.Sprintf("echo '%s' | nc 10.46.126.75 1111", msgTwo)}

					podTwoResult, err := podTwo.ExecCommand(sendDataTwoCmd, "two")
					Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to send data from pod %s", podTwo.Definition.Name))
					glog.V(ecoreparams.ECoreLogLevel).Infof("Result: %v - %s", podTwoResult, &podTwoResult)

					logStartTimestamp, err = time.ParseDuration("5s")
					Expect(err).ToNot(HaveOccurred(), "Failed to parse time duration")

					podOneLog, err := podOne.GetLog(logStartTimestamp, "one")
					Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to get logs from pod %s", podOne.Definition.Name))

					glog.V(ecoreparams.ECoreLogLevel).Infof("Logs from pod %s:\n%s", podOne.Definition.Name, podOneLog)
					Expect(podOneLog).Should(ContainSubstring(msgTwo))
				}) // Assert SR-IOV workloads on the same node

				It("Assert SR-IOV workloads on different nodes", func(ctx SpecContext) {

					By("Checking SR-IOV deployments don't exist")
					for _, dName := range []string{ECoreConfig.WlkdSRIOVDeployOneName, ECoreConfig.WlkdSRIOVDeployTwoName} {
						deploy, err := deployment.Pull(APIClient, dName, ECoreConfig.NamespacePCG)
						if deploy != nil && err == nil {
							glog.V(ecoreparams.ECoreLogLevel).Infof("Deployment %q found in %q namespace. Deleting...",
								deploy.Definition.Name, ECoreConfig.NamespacePCG)

							err := deploy.DeleteAndWait(300 * time.Second)
							Expect(err).ToNot(HaveOccurred(),
								fmt.Sprintf("failed to delete deployment %q", dName))

						}
					}

					By("Asserting pods from deployments are gone")
					labelsWlkdOne := labelsWlkdOneString
					labelsWlkdTwo := labelsWlkdTwoString

					for _, label := range []string{labelsWlkdOne, labelsWlkdTwo} {
						Eventually(func() bool {
							oldPods, _ := pod.List(APIClient, ECoreConfig.NamespacePCG,
								metav1.ListOptions{LabelSelector: label})

							return len(oldPods) == 0

						}, 6*time.Minute, 3*time.Second).WithContext(ctx).Should(BeTrue(), "pods matching label(s) still present")
					}

					By("Defining container configuration")
					deployContainer := pod.NewContainerBuilder("one", ECoreConfig.WlkdSRIOVDeployOneImage,
						ECoreConfig.WlkdSRIOVDeployOneCmd)

					deployContainerTwo := pod.NewContainerBuilder("two", ECoreConfig.WlkdSRIOVDeployTwoImage,
						ECoreConfig.WlkdSRIOVDeployTwoCmd)

					By("Setting SecurityContext")
					var trueFlag = true
					userUID := new(int64)
					*userUID = 0

					secContext := &v1.SecurityContext{
						RunAsUser:  userUID,
						Privileged: &trueFlag,
						SeccompProfile: &v1.SeccompProfile{
							Type: v1.SeccompProfileTypeRuntimeDefault,
						},
						Capabilities: &v1.Capabilities{
							Add: []v1.Capability{"NET_RAW", "NET_ADMIN", "SYS_ADMIN", "IPC_LOCK"},
						},
					}

					By("Setting SecurityContext")
					deployContainer = deployContainer.WithSecurityContext(secContext)
					glog.V(ecoreparams.ECoreLogLevel).Infof("Container One definition: %#v", deployContainer)

					deployContainerTwo = deployContainerTwo.WithSecurityContext(secContext)
					glog.V(ecoreparams.ECoreLogLevel).Infof("Container Two definition: %#v", deployContainerTwo)

					By("Dropping ALL security capability")
					deployContainer = deployContainer.WithDropSecurityCapabilities([]string{"ALL"}, true)
					deployContainerTwo = deployContainerTwo.WithDropSecurityCapabilities([]string{"ALL"}, true)

					By("Adding VolumeMount to container")
					volMount := v1.VolumeMount{
						Name:      "configs",
						MountPath: "/opt/net/",
						ReadOnly:  false,
					}

					deployContainer = deployContainer.WithVolumeMount(volMount)
					deployContainerTwo = deployContainerTwo.WithVolumeMount(volMount)

					glog.V(ecoreparams.ECoreLogLevel).Infof("Container One definition: %#v", deployContainer)
					glog.V(ecoreparams.ECoreLogLevel).Infof("Container Two definition: %#v", deployContainerTwo)

					By("Obtaining container definition")
					deployContainerCfg, err := deployContainer.GetContainerCfg()
					Expect(err).ToNot(HaveOccurred(), "Failed to get container definition")

					deployContainerTwoCfg, err := deployContainerTwo.GetContainerCfg()
					Expect(err).ToNot(HaveOccurred(), "Failed to get container definition")

					By("Defining deployment configuration")
					deploy := deployment.NewBuilder(APIClient,
						ECoreConfig.WlkdSRIOVDeployOneName,
						ECoreConfig.NamespacePCG,
						map[string]string{"systemtest-test": "ecore-sriov-one"},
						deployContainerCfg)

					deployTwo := deployment.NewBuilder(APIClient,
						ECoreConfig.WlkdSRIOVDeployTwoName,
						ECoreConfig.NamespacePCG,
						map[string]string{"systemtest-test": "ecore-sriov-two"},
						deployContainerTwoCfg)

					By("Adding SR-IOV annotations")
					var networksOne, networksTwo []*multus.NetworkSelectionElement

					networksOne = append(networksOne,
						&multus.NetworkSelectionElement{
							Name: ECoreConfig.WlkdSRIOVNetOne})

					networksTwo = append(networksTwo,
						&multus.NetworkSelectionElement{
							Name: ECoreConfig.WlkdSRIOVNetTwo})

					glog.V(ecoreparams.ECoreLogLevel).Infof("SR-IOV networks: %#v", networksOne)
					glog.V(ecoreparams.ECoreLogLevel).Infof("SR-IOV networks: %#v", networksTwo)

					deploy = deploy.WithSecondaryNetwork(networksOne)
					deployTwo = deployTwo.WithSecondaryNetwork(networksTwo)

					glog.V(ecoreparams.ECoreLogLevel).Infof("SR-IOV deploy one: %#v",
						deploy.Definition.Spec.Template.ObjectMeta.Annotations)
					glog.V(ecoreparams.ECoreLogLevel).Infof("SR-IOV deploy two: %#v",
						deployTwo.Definition.Spec.Template.ObjectMeta.Annotations)

					By("Adding NodeSelector to the deployment")
					deploy = deploy.WithNodeSelector(ECoreConfig.WlkdSRIOVDeployOneSelector)
					deployTwo = deployTwo.WithNodeSelector(ECoreConfig.WlkdSRIOVDeployTwoSelector)

					By("Adding Toleration")
					toleration := v1.Toleration{
						Key:      "sriov",
						Value:    "true",
						Effect:   v1.TaintEffectNoSchedule,
						Operator: v1.TolerationOpEqual,
					}

					deploy = deploy.WithToleration(toleration)
					deployTwo = deployTwo.WithToleration(toleration)

					glog.V(ecoreparams.ECoreLogLevel).Infof("SR-IOV Toleration: %#v",
						deploy.Definition.Spec.Template.Spec.Tolerations)
					glog.V(ecoreparams.ECoreLogLevel).Infof("SR-IOV Toleration: %#v",
						deployTwo.Definition.Spec.Template.Spec.Tolerations)

					By("Adding Volume to the deployment")
					volMode := new(int32)
					*volMode = 511

					volDefinition := v1.Volume{
						Name: "configs",
						VolumeSource: v1.VolumeSource{
							ConfigMap: &v1.ConfigMapVolumeSource{
								DefaultMode: volMode,
								LocalObjectReference: v1.LocalObjectReference{
									Name: ECoreConfig.WlkdSRIOVConfigMapNamePCG,
								},
							},
						},
					}

					deploy = deploy.WithVolume(volDefinition)
					deployTwo = deployTwo.WithVolume(volDefinition)

					glog.V(ecoreparams.ECoreLogLevel).Infof("SR-IOV One Volume:\n %v",
						deploy.Definition.Spec.Template.Spec.Volumes)
					glog.V(ecoreparams.ECoreLogLevel).Infof("SR-IOV Two Volume:\n %#v",
						deployTwo.Definition.Spec.Template.Spec.Volumes)

					By(fmt.Sprintf("Assigning ServiceAccount %q to the deployment", ECoreConfig.WlkdSRIOVOneSa))
					deploy = deploy.WithServiceAccountName(ECoreConfig.WlkdSRIOVOneSa)
					deployTwo = deployTwo.WithServiceAccountName(ECoreConfig.WlkdSRIOVOneSa)

					By("Setting Replicas count")
					deploy = deploy.WithReplicas(int32(1))
					deployTwo = deployTwo.WithReplicas(int32(1))

					By("Creating a deployment")
					deploy, err = deploy.CreateAndWaitUntilReady(60 * time.Second)
					Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to create deployment %s: %v", deploy.Definition.Name, err))

					deployTwo, err = deployTwo.CreateAndWaitUntilReady(60 * time.Second)
					Expect(err).ToNot(HaveOccurred(),
						fmt.Sprintf("Failed to create deployment %s: %v", deployTwo.Definition.Name, err))

					By("Getting pods backed by deployment")
					podOneSelector := metav1.ListOptions{
						LabelSelector: labelsWlkdOneString,
					}

					podOneList, err := pod.List(APIClient, ECoreConfig.NamespacePCG, podOneSelector)
					Expect(err).ToNot(HaveOccurred(), "Failed to find pods for deployment one")
					Expect(len(podOneList)).To(Equal(1), "Expected only one pod")

					podOne := podOneList[0]
					glog.V(ecoreparams.ECoreLogLevel).Infof("Pod one is %v on node %s",
						podOne.Definition.Name, podOne.Definition.Spec.NodeName)

					podTwoSelector := metav1.ListOptions{
						LabelSelector: labelsWlkdTwoString,
					}

					podTwoList, err := pod.List(APIClient, ECoreConfig.NamespacePCG, podTwoSelector)
					Expect(err).ToNot(HaveOccurred(), "Failed to find pods for deployment two")
					Expect(len(podTwoList)).To(Equal(1), "Expected only two pod")

					podTwo := podTwoList[0]
					glog.V(ecoreparams.ECoreLogLevel).Infof("Pod two is %v on node %s",
						podTwo.Definition.Name, podTwo.Definition.Spec.NodeName)

					By("Sending data from pod one to pod two")
					msgOne := fmt.Sprintf("Running from pod %s(%s) at %d",
						podOne.Definition.Name,
						podOne.Definition.Spec.NodeName,
						time.Now().Unix())

					glog.V(ecoreparams.ECoreLogLevel).Infof("Sending msg %q from pod %s",
						msgOne, podOne.Definition.Name)

					sendDataOneCmd := []string{"/bin/bash", "-c",
						fmt.Sprintf("echo '%s' | nc 10.46.126.77 2222", msgOne)}

					podOneResult, err := podOne.ExecCommand(sendDataOneCmd, "one")
					Expect(err).ToNot(HaveOccurred(),
						fmt.Sprintf("Failed to send data from pod %s", podOne.Definition.Name))
					glog.V(ecoreparams.ECoreLogLevel).Infof("Result: %v - %s", podOneResult, &podOneResult)

					logStartTimestamp, err := time.ParseDuration("5s")
					Expect(err).ToNot(HaveOccurred(), "Failed to parse time duration")

					podTwoLog, err := podTwo.GetLog(logStartTimestamp, "two")
					Expect(err).ToNot(HaveOccurred(),
						fmt.Sprintf("Failed to get logs from pod %s", podTwo.Definition.Name))

					glog.V(ecoreparams.ECoreLogLevel).Infof("Logs from pod %s:\n%s",
						podTwo.Definition.Name, podTwoLog)
					Expect(podTwoLog).Should(ContainSubstring(msgOne))

					By("Sending data from pod two to pod one")
					msgTwo := fmt.Sprintf("Running from pod %s(%s) at %d",
						podTwo.Definition.Name,
						podTwo.Definition.Spec.NodeName,
						time.Now().Unix())

					glog.V(ecoreparams.ECoreLogLevel).Infof("Sending msg %q from pod %s",
						msgTwo, podTwo.Definition.Name)

					sendDataTwoCmd := []string{"/bin/bash", "-c",
						fmt.Sprintf("echo '%s' | nc 10.46.126.75 1111", msgTwo)}

					podTwoResult, err := podTwo.ExecCommand(sendDataTwoCmd, "two")
					Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to send data from pod %s", podTwo.Definition.Name))
					glog.V(ecoreparams.ECoreLogLevel).Infof("Result: %v - %s", podTwoResult, &podTwoResult)

					logStartTimestamp, err = time.ParseDuration("5s")
					Expect(err).ToNot(HaveOccurred(), "Failed to parse time duration")

					podOneLog, err := podOne.GetLog(logStartTimestamp, "one")
					Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to get logs from pod %s", podOne.Definition.Name))

					glog.V(ecoreparams.ECoreLogLevel).Infof("Logs from pod %s:\n%s", podOne.Definition.Name, podOneLog)
					Expect(podOneLog).Should(ContainSubstring(msgTwo))
				}) // Assert SR-IOV workloads on the same node

			}) // Different SR-IOV networks

			Context("Same SR-IOV network", func() {
				It("Assert SR-IOV workloads on the same node", func(ctx SpecContext) {

					By("Checking SR-IOV deployments don't exist")
					for _, dName := range []string{ECoreConfig.WlkdSRIOVDeployOneName, ECoreConfig.WlkdSRIOVDeployTwoName} {
						deploy, err := deployment.Pull(APIClient, dName, ECoreConfig.NamespacePCG)
						if deploy != nil && err == nil {
							glog.V(ecoreparams.ECoreLogLevel).Infof("Deployment %q found in %q namespace. Deleting...",
								deploy.Definition.Name, ECoreConfig.NamespacePCG)

							err := deploy.DeleteAndWait(300 * time.Second)
							Expect(err).ToNot(HaveOccurred(),
								fmt.Sprintf("failed to delete deployment %q", dName))

						}
					}

					By("Asserting pods from deployments are gone")
					labelsWlkdOne := labelsWlkdOneString
					labelsWlkdTwo := labelsWlkdTwoString

					for _, label := range []string{labelsWlkdOne, labelsWlkdTwo} {
						Eventually(func() bool {
							oldPods, _ := pod.List(APIClient, ECoreConfig.NamespacePCG,
								metav1.ListOptions{LabelSelector: label})

							return len(oldPods) == 0

						}, 6*time.Minute, 3*time.Second).WithContext(ctx).Should(BeTrue(), "pods matching label(s) still present")
					}

					By("Defining container configuration")
					deployContainer := pod.NewContainerBuilder("one", ECoreConfig.WlkdSRIOVDeployOneImage,
						ECoreConfig.WlkdSRIOVDeployOneCmd)

					deployContainerTwo := pod.NewContainerBuilder("two", ECoreConfig.WlkdSRIOVDeployTwoImage,
						ECoreConfig.WlkdSRIOVDeployTwoCmd)

					By("Setting SecurityContext")
					var trueFlag = true
					userUID := new(int64)
					*userUID = 0

					secContext := &v1.SecurityContext{
						RunAsUser:  userUID,
						Privileged: &trueFlag,
						SeccompProfile: &v1.SeccompProfile{
							Type: v1.SeccompProfileTypeRuntimeDefault,
						},
						Capabilities: &v1.Capabilities{
							Add: []v1.Capability{"NET_RAW", "NET_ADMIN", "SYS_ADMIN", "IPC_LOCK"},
						},
					}

					By("Setting SecurityContext")
					deployContainer = deployContainer.WithSecurityContext(secContext)
					glog.V(ecoreparams.ECoreLogLevel).Infof("Container One definition: %#v", deployContainer)

					deployContainerTwo = deployContainerTwo.WithSecurityContext(secContext)
					glog.V(ecoreparams.ECoreLogLevel).Infof("Container Two definition: %#v", deployContainerTwo)

					By("Dropping ALL security capability")
					deployContainer = deployContainer.WithDropSecurityCapabilities([]string{"ALL"}, true)
					deployContainerTwo = deployContainerTwo.WithDropSecurityCapabilities([]string{"ALL"}, true)

					By("Adding VolumeMount to container")
					volMount := v1.VolumeMount{
						Name:      "configs",
						MountPath: "/opt/net/",
						ReadOnly:  false,
					}

					deployContainer = deployContainer.WithVolumeMount(volMount)
					deployContainerTwo = deployContainerTwo.WithVolumeMount(volMount)

					glog.V(ecoreparams.ECoreLogLevel).Infof("Container One definition: %#v", deployContainer)
					glog.V(ecoreparams.ECoreLogLevel).Infof("Container Two definition: %#v", deployContainerTwo)

					By("Obtaining container definition")
					deployContainerCfg, err := deployContainer.GetContainerCfg()
					Expect(err).ToNot(HaveOccurred(), "Failed to get container definition")

					deployContainerTwoCfg, err := deployContainerTwo.GetContainerCfg()
					Expect(err).ToNot(HaveOccurred(), "Failed to get container definition")

					By("Defining deployment configuration")
					deploy := deployment.NewBuilder(APIClient,
						ECoreConfig.WlkdSRIOVDeployOneName,
						ECoreConfig.NamespacePCG,
						map[string]string{"systemtest-test": "ecore-sriov-one"},
						deployContainerCfg)

					deployTwo := deployment.NewBuilder(APIClient,
						ECoreConfig.WlkdSRIOVDeployTwoName,
						ECoreConfig.NamespacePCG,
						map[string]string{"systemtest-test": "ecore-sriov-two"},
						deployContainerTwoCfg)

					By("Adding SR-IOV annotations")
					var networksOne, networksTwo []*multus.NetworkSelectionElement

					networksOne = append(networksOne,
						&multus.NetworkSelectionElement{
							Name: ECoreConfig.WlkdSRIOVNetOne})

					networksTwo = append(networksTwo,
						&multus.NetworkSelectionElement{
							Name: ECoreConfig.WlkdSRIOVNetOne})

					glog.V(ecoreparams.ECoreLogLevel).Infof("SR-IOV networks: %#v", networksOne)
					glog.V(ecoreparams.ECoreLogLevel).Infof("SR-IOV networks: %#v", networksTwo)

					deploy = deploy.WithSecondaryNetwork(networksOne)
					deployTwo = deployTwo.WithSecondaryNetwork(networksTwo)

					glog.V(ecoreparams.ECoreLogLevel).Infof("SR-IOV deploy one: %#v",
						deploy.Definition.Spec.Template.ObjectMeta.Annotations)
					glog.V(ecoreparams.ECoreLogLevel).Infof("SR-IOV deploy two: %#v",
						deployTwo.Definition.Spec.Template.ObjectMeta.Annotations)

					By("Adding NodeSelector to the deployment")
					deploy = deploy.WithNodeSelector(ECoreConfig.WlkdSRIOVDeployOneSelector)
					deployTwo = deployTwo.WithNodeSelector(ECoreConfig.WlkdSRIOVDeployOneSelector)

					By("Adding Toleration")
					toleration := v1.Toleration{
						Key:      "sriov",
						Value:    "true",
						Effect:   v1.TaintEffectNoSchedule,
						Operator: v1.TolerationOpEqual,
					}

					deploy = deploy.WithToleration(toleration)
					deployTwo = deployTwo.WithToleration(toleration)

					glog.V(ecoreparams.ECoreLogLevel).Infof("SR-IOV Toleration: %#v",
						deploy.Definition.Spec.Template.Spec.Tolerations)
					glog.V(ecoreparams.ECoreLogLevel).Infof("SR-IOV Toleration: %#v",
						deployTwo.Definition.Spec.Template.Spec.Tolerations)

					By("Adding Volume to the deployment")
					volMode := new(int32)
					*volMode = 511

					volDefinition := v1.Volume{
						Name: "configs",
						VolumeSource: v1.VolumeSource{
							ConfigMap: &v1.ConfigMapVolumeSource{
								DefaultMode: volMode,
								LocalObjectReference: v1.LocalObjectReference{
									Name: ECoreConfig.WlkdSRIOVConfigMapNamePCG,
								},
							},
						},
					}

					deploy = deploy.WithVolume(volDefinition)
					deployTwo = deployTwo.WithVolume(volDefinition)

					glog.V(ecoreparams.ECoreLogLevel).Infof("SR-IOV One Volume:\n %v",
						deploy.Definition.Spec.Template.Spec.Volumes)
					glog.V(ecoreparams.ECoreLogLevel).Infof("SR-IOV Two Volume:\n %#v",
						deployTwo.Definition.Spec.Template.Spec.Volumes)

					By(fmt.Sprintf("Assigning ServiceAccount %q to the deployment", ECoreConfig.WlkdSRIOVOneSa))
					deploy = deploy.WithServiceAccountName(ECoreConfig.WlkdSRIOVOneSa)
					deployTwo = deployTwo.WithServiceAccountName(ECoreConfig.WlkdSRIOVOneSa)

					By("Setting Replicas count")
					deploy = deploy.WithReplicas(int32(1))
					deployTwo = deployTwo.WithReplicas(int32(1))

					By("Creating a deployment")
					deploy, err = deploy.CreateAndWaitUntilReady(60 * time.Second)
					Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to create deployment %s: %v", deploy.Definition.Name, err))

					deployTwo, err = deployTwo.CreateAndWaitUntilReady(60 * time.Second)
					Expect(err).ToNot(HaveOccurred(),
						fmt.Sprintf("Failed to create deployment %s: %v", deployTwo.Definition.Name, err))

					By("Getting pods backed by deployment")
					podOneSelector := metav1.ListOptions{
						LabelSelector: labelsWlkdOneString,
					}

					podOneList, err := pod.List(APIClient, ECoreConfig.NamespacePCG, podOneSelector)
					Expect(err).ToNot(HaveOccurred(), "Failed to find pods for deployment one")
					Expect(len(podOneList)).To(Equal(1), "Expected only one pod")

					podOne := podOneList[0]
					glog.V(ecoreparams.ECoreLogLevel).Infof("Pod one is %v on node %s",
						podOne.Definition.Name, podOne.Definition.Spec.NodeName)

					podTwoSelector := metav1.ListOptions{
						LabelSelector: labelsWlkdTwoString,
					}

					podTwoList, err := pod.List(APIClient, ECoreConfig.NamespacePCG, podTwoSelector)
					Expect(err).ToNot(HaveOccurred(), "Failed to find pods for deployment two")
					Expect(len(podTwoList)).To(Equal(1), "Expected only two pod")

					podTwo := podTwoList[0]
					glog.V(ecoreparams.ECoreLogLevel).Infof("Pod two is %v on node %s",
						podTwo.Definition.Name, podTwo.Definition.Spec.NodeName)

					By("Sending data from pod one to pod two")
					msgOne := fmt.Sprintf("Running from pod %s(%s) at %d",
						podOne.Definition.Name,
						podOne.Definition.Spec.NodeName,
						time.Now().Unix())

					glog.V(ecoreparams.ECoreLogLevel).Infof("Sending msg %q from pod %s",
						msgOne, podOne.Definition.Name)

					sendDataOneCmd := []string{"/bin/bash", "-c",
						fmt.Sprintf("echo '%s' | nc 10.46.126.77 2222", msgOne)}

					podOneResult, err := podOne.ExecCommand(sendDataOneCmd, "one")
					Expect(err).ToNot(HaveOccurred(),
						fmt.Sprintf("Failed to send data from pod %s", podOne.Definition.Name))
					glog.V(ecoreparams.ECoreLogLevel).Infof("Result: %v - %s", podOneResult, &podOneResult)

					logStartTimestamp, err := time.ParseDuration("5s")
					Expect(err).ToNot(HaveOccurred(), "Failed to parse time duration")

					podTwoLog, err := podTwo.GetLog(logStartTimestamp, "two")
					Expect(err).ToNot(HaveOccurred(),
						fmt.Sprintf("Failed to get logs from pod %s", podTwo.Definition.Name))

					glog.V(ecoreparams.ECoreLogLevel).Infof("Logs from pod %s:\n%s",
						podTwo.Definition.Name, podTwoLog)
					Expect(podTwoLog).Should(ContainSubstring(msgOne))

					By("Sending data from pod two to pod one")
					msgTwo := fmt.Sprintf("Running from pod %s(%s) at %d",
						podTwo.Definition.Name,
						podTwo.Definition.Spec.NodeName,
						time.Now().Unix())

					glog.V(ecoreparams.ECoreLogLevel).Infof("Sending msg %q from pod %s",
						msgTwo, podTwo.Definition.Name)

					sendDataTwoCmd := []string{"/bin/bash", "-c",
						fmt.Sprintf("echo '%s' | nc 10.46.126.75 1111", msgTwo)}

					podTwoResult, err := podTwo.ExecCommand(sendDataTwoCmd, "two")
					Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to send data from pod %s", podTwo.Definition.Name))
					glog.V(ecoreparams.ECoreLogLevel).Infof("Result: %v - %s", podTwoResult, &podTwoResult)

					logStartTimestamp, err = time.ParseDuration("5s")
					Expect(err).ToNot(HaveOccurred(), "Failed to parse time duration")

					podOneLog, err := podOne.GetLog(logStartTimestamp, "one")
					Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to get logs from pod %s", podOne.Definition.Name))

					glog.V(ecoreparams.ECoreLogLevel).Infof("Logs from pod %s:\n%s", podOne.Definition.Name, podOneLog)
					Expect(podOneLog).Should(ContainSubstring(msgTwo))
				}) // Assert SR-IOV workloads on the same node

				It("Assert SR-IOV workloads on different nodes", func(ctx SpecContext) {

					By("Checking SR-IOV deployments don't exist")
					for _, dName := range []string{ECoreConfig.WlkdSRIOVDeployOneName, ECoreConfig.WlkdSRIOVDeployTwoName} {
						deploy, err := deployment.Pull(APIClient, dName, ECoreConfig.NamespacePCG)
						if deploy != nil && err == nil {
							glog.V(ecoreparams.ECoreLogLevel).Infof("Deployment %q found in %q namespace. Deleting...",
								deploy.Definition.Name, ECoreConfig.NamespacePCG)

							err := deploy.DeleteAndWait(300 * time.Second)
							Expect(err).ToNot(HaveOccurred(),
								fmt.Sprintf("failed to delete deployment %q", dName))

						}
					}

					By("Asserting pods from deployments are gone")
					labelsWlkdOne := labelsWlkdOneString
					labelsWlkdTwo := labelsWlkdTwoString

					for _, label := range []string{labelsWlkdOne, labelsWlkdTwo} {
						Eventually(func() bool {
							oldPods, _ := pod.List(APIClient, ECoreConfig.NamespacePCG,
								metav1.ListOptions{LabelSelector: label})

							return len(oldPods) == 0

						}, 6*time.Minute, 3*time.Second).WithContext(ctx).Should(BeTrue(), "pods matching label(s) still present")
					}

					By("Defining container configuration")
					deployContainer := pod.NewContainerBuilder("one", ECoreConfig.WlkdSRIOVDeployOneImage,
						ECoreConfig.WlkdSRIOVDeployOneCmd)

					deployContainerTwo := pod.NewContainerBuilder("two", ECoreConfig.WlkdSRIOVDeployTwoImage,
						ECoreConfig.WlkdSRIOVDeployTwoCmd)

					By("Setting SecurityContext")
					var trueFlag = true
					userUID := new(int64)
					*userUID = 0

					secContext := &v1.SecurityContext{
						RunAsUser:  userUID,
						Privileged: &trueFlag,
						SeccompProfile: &v1.SeccompProfile{
							Type: v1.SeccompProfileTypeRuntimeDefault,
						},
						Capabilities: &v1.Capabilities{
							Add: []v1.Capability{"NET_RAW", "NET_ADMIN", "SYS_ADMIN", "IPC_LOCK"},
						},
					}

					By("Setting SecurityContext")
					deployContainer = deployContainer.WithSecurityContext(secContext)
					glog.V(ecoreparams.ECoreLogLevel).Infof("Container One definition: %#v", deployContainer)

					deployContainerTwo = deployContainerTwo.WithSecurityContext(secContext)
					glog.V(ecoreparams.ECoreLogLevel).Infof("Container Two definition: %#v", deployContainerTwo)

					By("Dropping ALL security capability")
					deployContainer = deployContainer.WithDropSecurityCapabilities([]string{"ALL"}, true)
					deployContainerTwo = deployContainerTwo.WithDropSecurityCapabilities([]string{"ALL"}, true)

					By("Adding VolumeMount to container")
					volMount := v1.VolumeMount{
						Name:      "configs",
						MountPath: "/opt/net/",
						ReadOnly:  false,
					}

					deployContainer = deployContainer.WithVolumeMount(volMount)
					deployContainerTwo = deployContainerTwo.WithVolumeMount(volMount)

					glog.V(ecoreparams.ECoreLogLevel).Infof("Container One definition: %#v", deployContainer)
					glog.V(ecoreparams.ECoreLogLevel).Infof("Container Two definition: %#v", deployContainerTwo)

					By("Obtaining container definition")
					deployContainerCfg, err := deployContainer.GetContainerCfg()
					Expect(err).ToNot(HaveOccurred(), "Failed to get container definition")

					deployContainerTwoCfg, err := deployContainerTwo.GetContainerCfg()
					Expect(err).ToNot(HaveOccurred(), "Failed to get container definition")

					By("Defining deployment configuration")
					deploy := deployment.NewBuilder(APIClient,
						ECoreConfig.WlkdSRIOVDeployOneName,
						ECoreConfig.NamespacePCG,
						map[string]string{"systemtest-test": "ecore-sriov-one"},
						deployContainerCfg)

					deployTwo := deployment.NewBuilder(APIClient,
						ECoreConfig.WlkdSRIOVDeployTwoName,
						ECoreConfig.NamespacePCG,
						map[string]string{"systemtest-test": "ecore-sriov-two"},
						deployContainerTwoCfg)

					By("Adding SR-IOV annotations")
					var networksOne, networksTwo []*multus.NetworkSelectionElement

					networksOne = append(networksOne,
						&multus.NetworkSelectionElement{
							Name: ECoreConfig.WlkdSRIOVNetOne})

					networksTwo = append(networksTwo,
						&multus.NetworkSelectionElement{
							Name: ECoreConfig.WlkdSRIOVNetOne})

					glog.V(ecoreparams.ECoreLogLevel).Infof("SR-IOV networks: %#v", networksOne)
					glog.V(ecoreparams.ECoreLogLevel).Infof("SR-IOV networks: %#v", networksTwo)

					deploy = deploy.WithSecondaryNetwork(networksOne)
					deployTwo = deployTwo.WithSecondaryNetwork(networksTwo)

					glog.V(ecoreparams.ECoreLogLevel).Infof("SR-IOV deploy one: %#v",
						deploy.Definition.Spec.Template.ObjectMeta.Annotations)
					glog.V(ecoreparams.ECoreLogLevel).Infof("SR-IOV deploy two: %#v",
						deployTwo.Definition.Spec.Template.ObjectMeta.Annotations)

					By("Adding NodeSelector to the deployment")
					deploy = deploy.WithNodeSelector(ECoreConfig.WlkdSRIOVDeployOneSelector)
					deployTwo = deployTwo.WithNodeSelector(ECoreConfig.WlkdSRIOVDeployTwoSelector)

					By("Adding Toleration")
					toleration := v1.Toleration{
						Key:      "sriov",
						Value:    "true",
						Effect:   v1.TaintEffectNoSchedule,
						Operator: v1.TolerationOpEqual,
					}

					deploy = deploy.WithToleration(toleration)
					deployTwo = deployTwo.WithToleration(toleration)

					glog.V(ecoreparams.ECoreLogLevel).Infof("SR-IOV Toleration: %#v",
						deploy.Definition.Spec.Template.Spec.Tolerations)
					glog.V(ecoreparams.ECoreLogLevel).Infof("SR-IOV Toleration: %#v",
						deployTwo.Definition.Spec.Template.Spec.Tolerations)

					By("Adding Volume to the deployment")
					volMode := new(int32)
					*volMode = 511

					volDefinition := v1.Volume{
						Name: "configs",
						VolumeSource: v1.VolumeSource{
							ConfigMap: &v1.ConfigMapVolumeSource{
								DefaultMode: volMode,
								LocalObjectReference: v1.LocalObjectReference{
									Name: ECoreConfig.WlkdSRIOVConfigMapNamePCG,
								},
							},
						},
					}

					deploy = deploy.WithVolume(volDefinition)
					deployTwo = deployTwo.WithVolume(volDefinition)

					glog.V(ecoreparams.ECoreLogLevel).Infof("SR-IOV One Volume:\n %v",
						deploy.Definition.Spec.Template.Spec.Volumes)
					glog.V(ecoreparams.ECoreLogLevel).Infof("SR-IOV Two Volume:\n %#v",
						deployTwo.Definition.Spec.Template.Spec.Volumes)

					By(fmt.Sprintf("Assigning ServiceAccount %q to the deployment", ECoreConfig.WlkdSRIOVOneSa))
					deploy = deploy.WithServiceAccountName(ECoreConfig.WlkdSRIOVOneSa)
					deployTwo = deployTwo.WithServiceAccountName(ECoreConfig.WlkdSRIOVOneSa)

					By("Setting Replicas count")
					deploy = deploy.WithReplicas(int32(1))
					deployTwo = deployTwo.WithReplicas(int32(1))

					By("Creating a deployment")
					deploy, err = deploy.CreateAndWaitUntilReady(60 * time.Second)
					Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to create deployment %s: %v", deploy.Definition.Name, err))

					deployTwo, err = deployTwo.CreateAndWaitUntilReady(60 * time.Second)
					Expect(err).ToNot(HaveOccurred(),
						fmt.Sprintf("Failed to create deployment %s: %v", deployTwo.Definition.Name, err))

					By("Getting pods backed by deployment")
					podOneSelector := metav1.ListOptions{
						LabelSelector: labelsWlkdOneString,
					}

					podOneList, err := pod.List(APIClient, ECoreConfig.NamespacePCG, podOneSelector)
					Expect(err).ToNot(HaveOccurred(), "Failed to find pods for deployment one")
					Expect(len(podOneList)).To(Equal(1), "Expected only one pod")

					podOne := podOneList[0]
					glog.V(ecoreparams.ECoreLogLevel).Infof("Pod one is %v on node %s",
						podOne.Definition.Name, podOne.Definition.Spec.NodeName)

					podTwoSelector := metav1.ListOptions{
						LabelSelector: labelsWlkdTwoString,
					}

					podTwoList, err := pod.List(APIClient, ECoreConfig.NamespacePCG, podTwoSelector)
					Expect(err).ToNot(HaveOccurred(), "Failed to find pods for deployment two")
					Expect(len(podTwoList)).To(Equal(1), "Expected only two pod")

					podTwo := podTwoList[0]
					glog.V(ecoreparams.ECoreLogLevel).Infof("Pod two is %v on node %s",
						podTwo.Definition.Name, podTwo.Definition.Spec.NodeName)

					By("Sending data from pod one to pod two")
					msgOne := fmt.Sprintf("Running from pod %s(%s) at %d",
						podOne.Definition.Name,
						podOne.Definition.Spec.NodeName,
						time.Now().Unix())

					glog.V(ecoreparams.ECoreLogLevel).Infof("Sending msg %q from pod %s",
						msgOne, podOne.Definition.Name)

					sendDataOneCmd := []string{"/bin/bash", "-c",
						fmt.Sprintf("echo '%s' | nc 10.46.126.77 2222", msgOne)}

					podOneResult, err := podOne.ExecCommand(sendDataOneCmd, "one")
					Expect(err).ToNot(HaveOccurred(),
						fmt.Sprintf("Failed to send data from pod %s", podOne.Definition.Name))
					glog.V(ecoreparams.ECoreLogLevel).Infof("Result: %v - %s", podOneResult, &podOneResult)

					logStartTimestamp, err := time.ParseDuration("5s")
					Expect(err).ToNot(HaveOccurred(), "Failed to parse time duration")

					podTwoLog, err := podTwo.GetLog(logStartTimestamp, "two")
					Expect(err).ToNot(HaveOccurred(),
						fmt.Sprintf("Failed to get logs from pod %s", podTwo.Definition.Name))

					glog.V(ecoreparams.ECoreLogLevel).Infof("Logs from pod %s:\n%s",
						podTwo.Definition.Name, podTwoLog)
					Expect(podTwoLog).Should(ContainSubstring(msgOne))

					By("Sending data from pod two to pod one")
					msgTwo := fmt.Sprintf("Running from pod %s(%s) at %d",
						podTwo.Definition.Name,
						podTwo.Definition.Spec.NodeName,
						time.Now().Unix())

					glog.V(ecoreparams.ECoreLogLevel).Infof("Sending msg %q from pod %s",
						msgTwo, podTwo.Definition.Name)

					sendDataTwoCmd := []string{"/bin/bash", "-c",
						fmt.Sprintf("echo '%s' | nc 10.46.126.75 1111", msgTwo)}

					podTwoResult, err := podTwo.ExecCommand(sendDataTwoCmd, "two")
					Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to send data from pod %s", podTwo.Definition.Name))
					glog.V(ecoreparams.ECoreLogLevel).Infof("Result: %v - %s", podTwoResult, &podTwoResult)

					logStartTimestamp, err = time.ParseDuration("5s")
					Expect(err).ToNot(HaveOccurred(), "Failed to parse time duration")

					podOneLog, err := podOne.GetLog(logStartTimestamp, "one")
					Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to get logs from pod %s", podOne.Definition.Name))

					glog.V(ecoreparams.ECoreLogLevel).Infof("Logs from pod %s:\n%s", podOne.Definition.Name, podOneLog)
					Expect(podOneLog).Should(ContainSubstring(msgTwo))
				}) // Assert SR-IOV workloads on the same node

			}) // Different SR-IOV networks

		}) // SR-IOV Workloads
	})
