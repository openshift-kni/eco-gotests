package ecore_system_test

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/configmap"
	"github.com/openshift-kni/eco-goinfra/pkg/deployment"
	"github.com/openshift-kni/eco-goinfra/pkg/nad"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-goinfra/pkg/rbac"
	"github.com/openshift-kni/eco-goinfra/pkg/serviceaccount"
	multus "gopkg.in/k8snetworkplumbingwg/multus-cni.v4/pkg/types"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/openshift-kni/eco-gotests/tests/system-tests/ecore/internal/ecoreinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/ecore/internal/ecoreparams"
)

var _ = Describe(
	"ECore MacVlan Definitions",
	Ordered,
	ContinueOnFailure,
	Label(ecoreparams.LabelEcoreValidateNAD), func() {
		BeforeAll(func() {

			By(fmt.Sprintf("Asserting namespace %q exists", ECoreConfig.NamespacePCC))

			_, err := namespace.Pull(APIClient, ECoreConfig.NamespacePCC)
			Expect(err).To(BeNil(), fmt.Sprintf("Test namespace %s does not exist", ECoreConfig.NamespacePCC))

			By(fmt.Sprintf("Asserting namespace %q exists", ECoreConfig.NamespacePCG))

			_, err = namespace.Pull(APIClient, ECoreConfig.NamespacePCG)
			Expect(err).To(BeNil(), fmt.Sprintf("Test namespace %s does not exist", ECoreConfig.NamespacePCG))

		})

		Describe("Verifying NADs are created", func() {
			It("Asserts net-attach-def exist in PCC ns", Label("ecore_validate_nad_pcc_ns"), func() {

				for _, nadName := range ECoreConfig.NADListPCC {
					By(fmt.Sprintf("Asserting %q exists in %q ns", nadName, ECoreConfig.NamespacePCC))
					glog.V(ecoreparams.ECoreLogLevel).Infof("Checking NAD %q in %q ns", nadName, ECoreConfig.NamespacePCC)

					_, err := nad.Pull(APIClient, nadName, ECoreConfig.NamespacePCC)
					Expect(err).ToNot(HaveOccurred(),
						fmt.Sprintf("Failed to find net-attach-def %q", nadName))
				}

			})

			It("Asserts net-attach-def exist in PCG ns", Label("ecore_validate_nad_pcg_ns"), func() {

				for _, nadName := range ECoreConfig.NADListPCG {
					By(fmt.Sprintf("Asserting %q exists in %q ns", nadName, ECoreConfig.NamespacePCG))
					glog.V(ecoreparams.ECoreLogLevel).Infof("Checking NAD %q in %q ns", nadName, ECoreConfig.NamespacePCG)

					_, err := nad.Pull(APIClient, nadName, ECoreConfig.NamespacePCG)
					Expect(err).ToNot(HaveOccurred(),
						fmt.Sprintf("Failed to find net-attach-def %q", nadName))
				}

			})
		}) // Verifying NADs are created

		Describe("Configuring workloads", Label("ecore_validate_cm"), func() {

			var cmBuilder *configmap.Builder
			const rbacName = "privileged-telco-qe"

			Context("Workloads in the PCC namespace", func() {
				BeforeEach(func() {
					if cmBuilder, err := configmap.Pull(
						APIClient,
						ECoreConfig.NADConfigMapPCCName, ECoreConfig.NamespacePCC); err == nil {

						glog.V(ecoreparams.ECoreLogLevel).Infof("Deleting ConfigMap %q from %q ns",
							ECoreConfig.NADConfigMapPCCName,
							ECoreConfig.NamespacePCC)

						err := cmBuilder.Delete()
						Expect(err).ToNot(HaveOccurred(),
							fmt.Sprintf("Failed to delete CM %q from %q ns",
								ECoreConfig.NADConfigMapPCCName, ECoreConfig.NamespacePCC))
					}

					glog.V(ecoreparams.ECoreLogLevel).Infof("Creating ConfigMap")
					cmBuilder = configmap.NewBuilder(APIClient, ECoreConfig.NADConfigMapPCCName, ECoreConfig.NamespacePCC)
					cmBuilder = cmBuilder.WithData(ECoreConfig.NADConfigMapPCCData)

					_, err := cmBuilder.Create()
					Expect(err).ToNot(HaveOccurred(), "Failed to create configMap")

					DeferCleanup(func() {
						err := cmBuilder.Delete()
						Expect(err).ToNot(HaveOccurred(),
							fmt.Sprintf("Failed to delete CM %q from %q ns",
								ECoreConfig.NADConfigMapPCCName, ECoreConfig.NamespacePCC))
					})
				})

				BeforeEach(func() {

					if sAccount, err := serviceaccount.Pull(
						APIClient,
						ECoreConfig.NADWlkdOnePCCSa,
						ECoreConfig.NamespacePCC); err == nil {

						glog.V(ecoreparams.ECoreLogLevel).Infof("Deleting SA %q from %q ns",
							ECoreConfig.NADWlkdOnePCCSa, ECoreConfig.NamespacePCC)

						err := sAccount.Delete()
						Expect(err).ToNot(HaveOccurred(),
							fmt.Sprintf("Failed to delete SA %q from %q ns",
								ECoreConfig.NADWlkdOnePCCSa, ECoreConfig.NamespacePCC))
					}
				})

				AfterEach(func() {
					By("Cleaning ServiceAccount")
					if sAccount, err := serviceaccount.Pull(
						APIClient,
						ECoreConfig.NADWlkdOnePCCSa,
						ECoreConfig.NamespacePCC); err == nil {

						glog.V(ecoreparams.ECoreLogLevel).Infof("Deleting SA %q from %q ns",
							ECoreConfig.NADWlkdOnePCCSa, ECoreConfig.NamespacePCC)

						err := sAccount.Delete()
						Expect(err).ToNot(HaveOccurred(),
							fmt.Sprintf("Failed to delete SA %q from %q ns",
								ECoreConfig.NADWlkdOnePCCSa, ECoreConfig.NamespacePCC))
					}
				})

				// Remove deployment
				BeforeEach(func(ctx SpecContext) {
					By("Asserting deployments do not exist")
					for _, dName := range []string{ECoreConfig.NADWlkdDeployOnePCCName, ECoreConfig.NADWlkdDeployTwoPCCName} {
						deploy, _ := deployment.Pull(APIClient, dName, ECoreConfig.NamespacePCC)
						if deploy != nil {
							glog.V(ecoreparams.ECoreLogLevel).Infof("Existing deployment %q found. Removing...", dName)
							err := deploy.DeleteAndWait(300 * time.Second)
							Expect(err).ToNot(HaveOccurred(),
								fmt.Sprintf("failed to delete deployment %q", dName))
						}
					}

					By("Asserting pods from deployments are gone")
					labelsWlkdOne := "systemtest-test=ecore-wlkd-macvlan-one"
					labelsWlkdTwo := "systemtest-test=ecore-wlkd-macvlan-two"

					for _, label := range []string{labelsWlkdOne, labelsWlkdTwo} {
						Eventually(func() bool {
							oldPods, _ := pod.List(APIClient, ECoreConfig.NamespacePCC,
								metav1.ListOptions{LabelSelector: label})

							return len(oldPods) == 0

						}, 6*time.Minute, 3*time.Second).WithContext(ctx).Should(BeTrue(), "pods matching label(s) still present")
					}
				})

				AfterEach(func(ctx SpecContext) {
					By("Cleaning existing deployments")
					for _, dName := range []string{ECoreConfig.NADWlkdDeployOnePCCName, ECoreConfig.NADWlkdDeployTwoPCCName} {
						deploy, _ := deployment.Pull(APIClient, dName, ECoreConfig.NamespacePCC)
						if deploy != nil {
							glog.V(ecoreparams.ECoreLogLevel).Infof("Existing deployment %q found. Removing...", dName)
							err := deploy.DeleteAndWait(300 * time.Second)
							Expect(err).ToNot(HaveOccurred(),
								fmt.Sprintf("failed to delete deployment %q", dName))
						}
					}

					By("Asserting pods from deployments are gone")
					labelsWlkdOne := "systemtest-test=ecore-wlkd-macvlan-one"
					labelsWlkdTwo := "systemtest-test=ecore-wlkd-macvlan-two"

					for _, label := range []string{labelsWlkdOne, labelsWlkdTwo} {
						Eventually(func() bool {
							oldPods, _ := pod.List(APIClient, ECoreConfig.NamespacePCC,
								metav1.ListOptions{LabelSelector: label})

							return len(oldPods) == 0

						}, 6*time.Minute, 3*time.Second).WithContext(ctx).Should(BeTrue(), "pods matching label(s) still present")
					}
				})

				// Remove Cluster Role Binding
				BeforeEach(func() {
					By("Removing ClusterRoleBinding")
					glog.V(ecoreparams.ECoreLogLevel).Infof("Assert ClusterRoleBinding %q exists", rbacName)

					if crb, err := rbac.PullClusterRoleBinding(APIClient, rbacName); err == nil {
						glog.V(ecoreparams.ECoreLogLevel).Infof("ClusterRoleBinding %q found. Removing...", rbacName)
						err := crb.Delete()
						Expect(err).ToNot(HaveOccurred(),
							fmt.Sprintf("Failed to delete ClusterRoleBinding %q", rbacName))

					}
				})

				AfterEach(func() {
					By("Cleaning ClusterRoleBinding")
					glog.V(ecoreparams.ECoreLogLevel).Infof("Assert ClusterRoleBinding %q exists", rbacName)

					if crb, err := rbac.PullClusterRoleBinding(APIClient, rbacName); err == nil {
						glog.V(ecoreparams.ECoreLogLevel).Infof("ClusterRoleBinding %q found. Removing...", rbacName)
						err := crb.Delete()
						Expect(err).ToNot(HaveOccurred(),
							fmt.Sprintf("Failed to delete ClusterRoleBinding %q", rbacName))

					}
				})

				It("Can reach workload on the other node via MACVLAN interface", Label("wlkd_nad_pcc_different_nodes"), func() {

					By("Defining container configuration")
					deployContainer := pod.NewContainerBuilder("withmacvlan-c-one", ECoreConfig.NADWlkdDeployOnePCCImage,
						ECoreConfig.NADWlkdDeployOnePCCCmd)

					deployContainerTwo := pod.NewContainerBuilder("withmacvlan-c-two", ECoreConfig.NADWlkdDeployTwoPCCImage,
						ECoreConfig.NADWlkdDeployTwoPCCCmd)

					By("Setting SecurityContext")
					var trueFlag = true
					userUID := new(int64)
					*userUID = 0

					secContext := &corev1.SecurityContext{
						RunAsUser:  userUID,
						Privileged: &trueFlag,
						SeccompProfile: &corev1.SeccompProfile{
							Type: corev1.SeccompProfileTypeRuntimeDefault,
						},
						Capabilities: &corev1.Capabilities{
							Add: []corev1.Capability{"NET_RAW", "NET_ADMIN", "SYS_ADMIN", "IPC_LOCK"},
						},
					}

					glog.V(ecoreparams.ECoreLogLevel).Infof("*** SecurityContext is %v", secContext)

					By("Setting SecurityContext")
					deployContainer = deployContainer.WithSecurityContext(secContext)
					glog.V(ecoreparams.ECoreLogLevel).Infof("Container One definition: %#v", deployContainer)

					deployContainerTwo = deployContainerTwo.WithSecurityContext(secContext)
					glog.V(ecoreparams.ECoreLogLevel).Infof("Container Two definition: %#v", deployContainerTwo)

					By("Dropping ALL security capability")
					deployContainer = deployContainer.WithDropSecurityCapabilities([]string{"ALL"}, true)
					deployContainerTwo = deployContainerTwo.WithDropSecurityCapabilities([]string{"ALL"}, true)

					By("Adding VolumeMount to container")
					volMount := corev1.VolumeMount{
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
						ECoreConfig.NADWlkdDeployOnePCCName,
						ECoreConfig.NamespacePCC,
						map[string]string{"systemtest-test": "ecore-wlkd-macvlan-one"},
						deployContainerCfg)

					deployTwo := deployment.NewBuilder(APIClient,
						ECoreConfig.NADWlkdDeployTwoPCCName,
						ECoreConfig.NamespacePCC,
						map[string]string{"systemtest-test": "ecore-wlkd-macvlan-two"},
						deployContainerTwoCfg)

					By("Adding MACVLAN annotations")
					var networks []*multus.NetworkSelectionElement
					networks = append(networks,
						&multus.NetworkSelectionElement{
							Name:      ECoreConfig.NADWlkdOneNadName,
							Namespace: ECoreConfig.NamespacePCC})

					glog.V(ecoreparams.ECoreLogLevel).Infof("MACVlan networks: %#v", networks)

					deploy = deploy.WithSecondaryNetwork(networks)
					deployTwo = deployTwo.WithSecondaryNetwork(networks)

					By("Adding NodeSelector to the deployment")
					deploy = deploy.WithNodeSelector(ECoreConfig.NADWlkdDeployOnePCCSelector)
					deployTwo = deployTwo.WithNodeSelector(ECoreConfig.NADWlkdDeployTwoPCCSelector)

					By("Adding Volume to the deployment")
					volMode := new(int32)
					*volMode = 511

					volDefinition := corev1.Volume{
						Name: "configs",
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								DefaultMode: volMode,
								LocalObjectReference: corev1.LocalObjectReference{
									Name: ECoreConfig.NADConfigMapPCCName,
								},
							},
						},
					}

					deploy = deploy.WithVolume(volDefinition)
					deployTwo = deployTwo.WithVolume(volDefinition)

					By("Creating ServiceAccount for the deployment")
					deploySa := serviceaccount.NewBuilder(
						APIClient, ECoreConfig.NADWlkdOnePCCSa, ECoreConfig.NamespacePCC)

					deploySa, err = deploySa.Create()
					Expect(err).ToNot(HaveOccurred(), "Failed to create ServiceAccount",
						deploySa.Definition.Name)

					By("Creating RBAC for SA")
					crbSa := rbac.NewClusterRoleBindingBuilder(APIClient,
						rbacName,
						"system:openshift:scc:privileged",
						rbacv1.Subject{
							Name:      ECoreConfig.NADWlkdOnePCCSa,
							Kind:      "ServiceAccount",
							Namespace: ECoreConfig.NamespacePCC,
						})

					crbSa, err = crbSa.Create()
					Expect(err).ToNot(HaveOccurred(), "Failed to create ClusterRoleBinding")

					glog.V(ecoreparams.ECoreLogLevel).Infof("ClusterRoleBinding %q created %v", rbacName, crbSa)

					By(fmt.Sprintf("Assigning ServiceAccount %q to the deployment", ECoreConfig.NADWlkdOnePCCSa))
					deploy = deploy.WithServiceAccountName(ECoreConfig.NADWlkdOnePCCSa)
					deployTwo = deployTwo.WithServiceAccountName(ECoreConfig.NADWlkdOnePCCSa)

					By("Creating a deployment")
					deploy, err = deploy.CreateAndWaitUntilReady(60 * time.Second)
					Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to create deployment %s: %v", deploy.Definition.Name, err))

					deployTwo, err = deployTwo.CreateAndWaitUntilReady(60 * time.Second)
					Expect(err).ToNot(HaveOccurred(),
						fmt.Sprintf("Failed to create deployment %s: %v", deployTwo.Definition.Name, err))

					By("Getting pods backed by deployment")
					podOneSelector := metav1.ListOptions{
						LabelSelector: "systemtest-test=ecore-wlkd-macvlan-one",
					}

					podOneList, err := pod.List(APIClient, ECoreConfig.NamespacePCC, podOneSelector)
					Expect(err).ToNot(HaveOccurred(), "Failed to find pods for deployment one")
					Expect(len(podOneList)).To(Equal(1), "Expected only one pod")

					podOne := podOneList[0]
					glog.V(ecoreparams.ECoreLogLevel).Infof("Pod one is %v", podOne.Definition.Name)

					podTwoSelector := metav1.ListOptions{
						LabelSelector: "systemtest-test=ecore-wlkd-macvlan-two",
					}

					podTwoList, err := pod.List(APIClient, ECoreConfig.NamespacePCC, podTwoSelector)
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
						fmt.Sprintf("echo '%s' | nc 10.46.125.71 2222", msgOne)}

					podOneResult, err := podOne.ExecCommand(sendDataOneCmd, "withmacvlan-c-one")
					Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to send data from pod %s", podOne.Definition.Name))
					glog.V(ecoreparams.ECoreLogLevel).Infof("Result: %v - %s", podOneResult, &podOneResult)

					logStartTimestamp, err := time.ParseDuration("5s")
					Expect(err).ToNot(HaveOccurred(), "Failed to parse time duration")

					podTwoLog, err := podTwo.GetLog(logStartTimestamp, "withmacvlan-c-two")
					Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to get logs from pod %s", podTwo.Definition.Name))

					glog.V(ecoreparams.ECoreLogLevel).Infof("Logs from pod %s:\n%s", podTwo.Definition.Name, podTwoLog)
					Expect(podTwoLog).Should(ContainSubstring(msgOne))

					By("Sending data from pod two to pod one")
					msgTwo := fmt.Sprintf("Running from pod %s(%s) at %d",
						podTwo.Definition.Name,
						podTwo.Definition.Spec.NodeName,
						time.Now().Unix())

					glog.V(ecoreparams.ECoreLogLevel).Infof("Sending msg %q from pod %s",
						msgTwo, podTwo.Definition.Name)

					sendDataTwoCmd := []string{"/bin/bash", "-c",
						fmt.Sprintf("echo '%s' | nc 10.46.125.70 1111", msgTwo)}

					podTwoResult, err := podTwo.ExecCommand(sendDataTwoCmd, "withmacvlan-c-two")
					Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to send data from pod %s", podTwo.Definition.Name))
					glog.V(ecoreparams.ECoreLogLevel).Infof("Result: %v - %s", podTwoResult, &podTwoResult)

					logStartTimestamp, err = time.ParseDuration("5s")
					Expect(err).ToNot(HaveOccurred(), "Failed to parse time duration")

					podOneLog, err := podOne.GetLog(logStartTimestamp, "withmacvlan-c-one")
					Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to get logs from pod %s", podOne.Definition.Name))

					glog.V(ecoreparams.ECoreLogLevel).Infof("Logs from pod %s:\n%s", podOne.Definition.Name, podOneLog)
					Expect(podOneLog).Should(ContainSubstring(msgTwo))

				})

				It("Can reach workload on the same node via MACVLAN interface", func(ctx SpecContext) {
					By("Asserting CM exists")
					_, err := configmap.Pull(APIClient, ECoreConfig.NADConfigMapPCCName, ECoreConfig.NamespacePCC)
					Expect(err).ToNot(HaveOccurred(), "Failed to get CM")

					By("Defining container configuration")
					deployContainer := pod.NewContainerBuilder("withmacvlan-c-one", ECoreConfig.NADWlkdDeployOnePCCImage,
						ECoreConfig.NADWlkdDeployOnePCCCmd)

					deployContainerTwo := pod.NewContainerBuilder("withmacvlan-c-two", ECoreConfig.NADWlkdDeployTwoPCCImage,
						ECoreConfig.NADWlkdDeployTwoPCCCmd)

					By("Setting SecurityContext")
					var trueFlag = true
					userUID := new(int64)
					*userUID = 0

					secContext := &corev1.SecurityContext{
						RunAsUser:  userUID,
						Privileged: &trueFlag,
						SeccompProfile: &corev1.SeccompProfile{
							Type: corev1.SeccompProfileTypeRuntimeDefault,
						},
						Capabilities: &corev1.Capabilities{
							Add: []corev1.Capability{"NET_RAW", "NET_ADMIN", "SYS_ADMIN", "IPC_LOCK"},
						},
					}

					glog.V(ecoreparams.ECoreLogLevel).Infof("*** SecurityContext is %v", secContext)

					By("Setting SecurityContext")
					deployContainer = deployContainer.WithSecurityContext(secContext)
					glog.V(ecoreparams.ECoreLogLevel).Infof("Container One definition: %#v", deployContainer)

					deployContainerTwo = deployContainerTwo.WithSecurityContext(secContext)
					glog.V(ecoreparams.ECoreLogLevel).Infof("Container Two definition: %#v", deployContainerTwo)

					By("Dropping ALL security capability")
					deployContainer = deployContainer.WithDropSecurityCapabilities([]string{"ALL"}, true)
					deployContainerTwo = deployContainerTwo.WithDropSecurityCapabilities([]string{"ALL"}, true)

					By("Adding VolumeMount to container")
					volMount := corev1.VolumeMount{
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
						ECoreConfig.NADWlkdDeployOnePCCName,
						ECoreConfig.NamespacePCC,
						map[string]string{"systemtest-test": "ecore-wlkd-macvlan-one"},
						deployContainerCfg)

					deployTwo := deployment.NewBuilder(APIClient,
						ECoreConfig.NADWlkdDeployTwoPCCName,
						ECoreConfig.NamespacePCC,
						map[string]string{"systemtest-test": "ecore-wlkd-macvlan-two"},
						deployContainerTwoCfg)

					By("Adding MACVLAN annotations")
					var networks []*multus.NetworkSelectionElement
					networks = append(networks,
						&multus.NetworkSelectionElement{
							Name:      ECoreConfig.NADWlkdOneNadName,
							Namespace: ECoreConfig.NamespacePCC})

					glog.V(ecoreparams.ECoreLogLevel).Infof("MACVlan networks: %#v", networks)

					deploy = deploy.WithSecondaryNetwork(networks)
					deployTwo = deployTwo.WithSecondaryNetwork(networks)

					By("Adding NodeSelector to the deployment")
					deploy = deploy.WithNodeSelector(ECoreConfig.NADWlkdDeployOnePCCSelector)
					deployTwo = deployTwo.WithNodeSelector(ECoreConfig.NADWlkdDeployOnePCCSelector)

					By("Adding Volume to the deployment")
					volMode := new(int32)
					*volMode = 511

					volDefinition := corev1.Volume{
						Name: "configs",
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								DefaultMode: volMode,
								LocalObjectReference: corev1.LocalObjectReference{
									Name: ECoreConfig.NADConfigMapPCCName,
								},
							},
						},
					}

					deploy = deploy.WithVolume(volDefinition)
					deployTwo = deployTwo.WithVolume(volDefinition)

					By("Creating ServiceAccount for the deployment")
					deploySa := serviceaccount.NewBuilder(
						APIClient, ECoreConfig.NADWlkdOnePCCSa, ECoreConfig.NamespacePCC)

					deploySa, err = deploySa.Create()
					Expect(err).ToNot(HaveOccurred(), "Failed to create ServiceAccount",
						deploySa.Definition.Name)

					By("Creating RBAC for SA")
					crbSa := rbac.NewClusterRoleBindingBuilder(APIClient,
						rbacName,
						"system:openshift:scc:privileged",
						rbacv1.Subject{
							Name:      ECoreConfig.NADWlkdOnePCCSa,
							Kind:      "ServiceAccount",
							Namespace: ECoreConfig.NamespacePCC,
						})

					crbSa, err = crbSa.Create()
					Expect(err).ToNot(HaveOccurred(), "Failed to create ClusterRoleBinding")

					glog.V(ecoreparams.ECoreLogLevel).Infof("ClusterRoleBinding %q created %v", rbacName, crbSa)

					By(fmt.Sprintf("Assigning ServiceAccount %q to the deployment", ECoreConfig.NADWlkdOnePCCSa))
					deploy = deploy.WithServiceAccountName(ECoreConfig.NADWlkdOnePCCSa)
					deployTwo = deployTwo.WithServiceAccountName(ECoreConfig.NADWlkdOnePCCSa)

					By("Creating a deployment")
					deploy, err = deploy.CreateAndWaitUntilReady(60 * time.Second)
					Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to create deployment %s: %v", deploy.Definition.Name, err))

					deployTwo, err = deployTwo.CreateAndWaitUntilReady(60 * time.Second)
					Expect(err).ToNot(HaveOccurred(),
						fmt.Sprintf("Failed to create deployment %s: %v", deployTwo.Definition.Name, err))

					By("Getting pods backed by deployment")
					podOneSelector := metav1.ListOptions{
						LabelSelector: "systemtest-test=ecore-wlkd-macvlan-one",
					}

					podOneList, err := pod.List(APIClient, ECoreConfig.NamespacePCC, podOneSelector)
					Expect(err).ToNot(HaveOccurred(), "Failed to find pods for deployment one")
					Expect(len(podOneList)).To(Equal(1), "Expected only one pod")

					podOne := podOneList[0]
					glog.V(ecoreparams.ECoreLogLevel).Infof("Pod one is %v", podOne.Definition.Name)

					podTwoSelector := metav1.ListOptions{
						LabelSelector: "systemtest-test=ecore-wlkd-macvlan-two",
					}

					podTwoList, err := pod.List(APIClient, ECoreConfig.NamespacePCC, podTwoSelector)
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
						fmt.Sprintf("echo '%s' | nc 10.46.125.71 2222", msgOne)}

					podOneResult, err := podOne.ExecCommand(sendDataOneCmd, "withmacvlan-c-one")
					Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to send data from pod %s", podOne.Definition.Name))
					glog.V(ecoreparams.ECoreLogLevel).Infof("Result: %v - %s", podOneResult, &podOneResult)

					logStartTimestamp, err := time.ParseDuration("5s")
					Expect(err).ToNot(HaveOccurred(), "Failed to parse time duration")

					podTwoLog, err := podTwo.GetLog(logStartTimestamp, "withmacvlan-c-two")
					Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to get logs from pod %s", podTwo.Definition.Name))

					glog.V(ecoreparams.ECoreLogLevel).Infof("Logs from pod %s:\n%s", podTwo.Definition.Name, podTwoLog)
					Expect(podTwoLog).Should(ContainSubstring(msgOne))

					By("Sending data from pod two to pod one")
					msgTwo := fmt.Sprintf("Running from pod %s(%s) at %d",
						podTwo.Definition.Name,
						podTwo.Definition.Spec.NodeName,
						time.Now().Unix())

					glog.V(ecoreparams.ECoreLogLevel).Infof("Sending msg %q from pod %s",
						msgTwo, podTwo.Definition.Name)

					sendDataTwoCmd := []string{"/bin/bash", "-c",
						fmt.Sprintf("echo '%s' | nc 10.46.125.70 1111", msgTwo)}

					podTwoResult, err := podTwo.ExecCommand(sendDataTwoCmd, "withmacvlan-c-two")
					Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to send data from pod %s", podTwo.Definition.Name))
					glog.V(ecoreparams.ECoreLogLevel).Infof("Result: %v - %s", podTwoResult, &podTwoResult)

					logStartTimestamp, err = time.ParseDuration("5s")
					Expect(err).ToNot(HaveOccurred(), "Failed to parse time duration")

					podOneLog, err := podOne.GetLog(logStartTimestamp, "withmacvlan-c-one")
					Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to get logs from pod %s", podOne.Definition.Name))

					glog.V(ecoreparams.ECoreLogLevel).Infof("Logs from pod %s:\n%s", podOne.Definition.Name, podOneLog)
					Expect(podOneLog).Should(ContainSubstring(msgTwo))

				})

			}) // Context -> Workloads on the same node
		})

	})
