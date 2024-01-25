package ecore_system_test

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift-kni/eco-goinfra/pkg/deployment"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-goinfra/pkg/storage"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
	. "github.com/openshift-kni/eco-gotests/tests/system-tests/ecore/internal/ecoreinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/ecore/internal/ecoreparams"
)

var _ = Describe(
	"ECore ODF Persistent Storage",
	Ordered,
	ContinueOnFailure,
	Label(ecoreparams.LabelEcoreValidateODFStorage), func() {
		Describe("StorageClasses", Label("ecore_sc_exist"), polarion.ID("62960"), func() {
			It("Asserts storageClasses exist", func() {
				glog.V(ecoreparams.ECoreLogLevel).Infof("Validating StorageClasses")
				for sc, provisioner := range ECoreConfig.StorageClassesMap {
					eClass := storage.NewClassBuilder(APIClient, sc, provisioner)
					glog.V(ecoreparams.ECoreLogLevel).Infof("Assert storageClass %q exists", sc)
					scExists := eClass.Exists()
					Expect(scExists).To(BeTrue(), fmt.Sprintf("StorageClass %q not found", sc))
				}
			})
		}) // StorageClasses exist

		Context("Create PVC based on StorageClass", Label("ecore_odf_pvc_per_sc"), func() {

			var testNSName = "qe-odf-ns"

			BeforeAll(func() {
				By("Asserting test namespace already exists")
				glog.V(ecoreparams.ECoreLogLevel).Infof(fmt.Sprintf("Assert if namespace %q exists", testNSName))

				if prevNS, err := namespace.Pull(APIClient, testNSName); err == nil {
					glog.V(ecoreparams.ECoreLogLevel).Infof(fmt.Sprintf("Namespace %q exists. Removing...", testNSName))
					delErr := prevNS.DeleteAndWait(6 * time.Minute)
					Expect(delErr).ToNot(HaveOccurred(), fmt.Sprintf("Failed to delete %q namespace", testNSName))
				}

				By("Creating a test namespace")
				testNS := namespace.NewBuilder(APIClient, testNSName)
				testNS, err := testNS.Create()
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to create test namespace %s", testNSName))
				glog.V(ecoreparams.ECoreLogLevel).Infof(fmt.Sprintf("Namespace %q created", testNS.Object.Name))
			})

			It("Asserts CephFS PVC creation", func(ctx SpecContext) {
				By("Creating new PVC Builder")
				myPVC := storage.NewPVCBuilder(APIClient, "telco-cephfs-pvc", testNSName)
				glog.V(ecoreparams.ECoreLogLevel).Infof(fmt.Sprintf("PVC\n%#v", myPVC))

				By("Setting AccessMode")
				myPVC, err := myPVC.WithPVCAccessMode("ReadWriteOnce")
				Expect(err).ToNot(HaveOccurred(), "Failed to set AccessMode")
				glog.V(ecoreparams.ECoreLogLevel).Infof("PVC accessMode: ", myPVC.Definition.Spec.AccessModes)

				By("Setting PVC capacity")
				myPVC, err = myPVC.WithPVCCapacity("5G")
				Expect(err).ToNot(HaveOccurred(), "Failed to set Capacity")
				glog.V(ecoreparams.ECoreLogLevel).Infof("PVC Capacity: ", myPVC.Definition.Spec.Resources)

				By("Setting StorageClass for PVC")
				myPVC, err = myPVC.WithStorageClass("ocs-external-storagecluster-cephfs")
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to use StorageClass: %v", err))
				glog.V(ecoreparams.ECoreLogLevel).Infof("PVC StorageClass: ", myPVC.Definition.Spec.StorageClassName)

				By("Creating PVC")
				myPVC, err = myPVC.Create()
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to create PVC: %v", err))
				glog.V(ecoreparams.ECoreLogLevel).Infof("Created PVC: ", myPVC.Object.Status)

				Eventually(func(phase string) bool {
					if ok := myPVC.Exists(); ok {
						glog.V(ecoreparams.ECoreLogLevel).Infof(fmt.Sprintf("\tPVC Phase is %q", myPVC.Object.Status.Phase))

						return string(myPVC.Object.Status.Phase) == phase
					}

					return false
				}, 5*time.Minute, 3*time.Second).WithContext(ctx).WithArguments("Bound").Should(BeTrue(),
					fmt.Sprintf("Unexpeced PVC state %q", myPVC.Object.Status.Phase))
			})

			It("Asserts Ceph RBD PVC creation", func(ctx SpecContext) {
				By("Creating new PVC Builder")
				myPVC := storage.NewPVCBuilder(APIClient, "telco-cephrbd-pvc", testNSName)
				glog.V(ecoreparams.ECoreLogLevel).Infof(fmt.Sprintf("PVC\n%#v", myPVC))

				By("Setting AccessMode")
				myPVC, err := myPVC.WithPVCAccessMode("ReadWriteOnce")
				Expect(err).ToNot(HaveOccurred(), "Failed to set AccessMode")
				glog.V(ecoreparams.ECoreLogLevel).Infof("PVC accessMode: ", myPVC.Definition.Spec.AccessModes)

				By("Setting PVC capacity")
				myPVC, err = myPVC.WithPVCCapacity("5G")
				Expect(err).ToNot(HaveOccurred(), "Failed to set Capacity")
				glog.V(ecoreparams.ECoreLogLevel).Infof("PVC Capacity: ", myPVC.Definition.Spec.Resources)

				By("Setting StorageClass for PVC")
				myPVC, err = myPVC.WithStorageClass("ocs-external-storagecluster-ceph-rbd")
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to use StorageClass: %v", err))
				glog.V(ecoreparams.ECoreLogLevel).Infof("PVC StorageClass: ", myPVC.Definition.Spec.StorageClassName)

				By("Creating PVC")
				myPVC, err = myPVC.Create()
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to create PVC: %v", err))
				glog.V(ecoreparams.ECoreLogLevel).Infof("Created PVC: ", myPVC.Object.Status)

				Eventually(func(phase string) bool {
					if ok := myPVC.Exists(); ok {
						glog.V(ecoreparams.ECoreLogLevel).Infof(fmt.Sprintf("\tPVC Phase is %q", myPVC.Object.Status.Phase))

						return string(myPVC.Object.Status.Phase) == phase
					}

					return false
				}, 5*time.Minute, 3*time.Second).WithContext(ctx).WithArguments("Bound").Should(BeTrue(),
					fmt.Sprintf("Unexpeced PVC state %q", myPVC.Object.Status.Phase))
			})

			It("Asserts Ceph Nooba PVC creation", func(ctx SpecContext) {
				By("Creating new PVC Builder")
				myPVC := storage.NewPVCBuilder(APIClient, "telco-nooba-pvc", testNSName)
				glog.V(ecoreparams.ECoreLogLevel).Infof(fmt.Sprintf("PVC\n%#v", myPVC))

				By("Setting AccessMode")
				myPVC, err := myPVC.WithPVCAccessMode("ReadWriteOnce")
				Expect(err).ToNot(HaveOccurred(), "Failed to set AccessMode")
				glog.V(ecoreparams.ECoreLogLevel).Infof("PVC accessMode: ", myPVC.Definition.Spec.AccessModes)

				By("Setting PVC capacity")
				myPVC, err = myPVC.WithPVCCapacity("5G")
				Expect(err).ToNot(HaveOccurred(), "Failed to set Capacity")
				glog.V(ecoreparams.ECoreLogLevel).Infof("PVC Capacity: ", myPVC.Definition.Spec.Resources)

				By("Setting StorageClass for PVC")
				myPVC, err = myPVC.WithStorageClass("openshift-storage.noobaa.io")
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to use StorageClass: %v", err))
				glog.V(ecoreparams.ECoreLogLevel).Infof("PVC StorageClass: ", myPVC.Definition.Spec.StorageClassName)

				By("Creating PVC")
				myPVC, err = myPVC.Create()
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to create PVC: %v", err))
				glog.V(ecoreparams.ECoreLogLevel).Infof("Created PVC: ", myPVC.Object.Status)

				Eventually(func(phase string) bool {
					if ok := myPVC.Exists(); ok {
						glog.V(ecoreparams.ECoreLogLevel).Infof(fmt.Sprintf("\tPVC Phase is %q", myPVC.Object.Status.Phase))

						return string(myPVC.Object.Status.Phase) == phase
					}

					return false
				}, 5*time.Minute, 3*time.Second).WithContext(ctx).WithArguments("Pending").Should(BeTrue(),
					fmt.Sprintf("Unexpeced PVC state %q", myPVC.Object.Status.Phase))
			})
		}) // end Context

		DescribeTable("Workloads with PVC", Label("ecore_odf_pvc_workload"),
			func(fNamespace string, fStorageClass string, fPVCName string, fVolumeMode string) {

				const (
					labelsWlkdOneString = "systemtest-test=ecore-odf-pvc"
					labelsWlkdTwoString = "systemtest-test=ecore-odf-two"
				)

				var (
					ctx               SpecContext
					workloadNS        *namespace.Builder
					wlkdODFDeployName = "qe-wlkd"
					wlkdODFCmd        = []string{"/bin/sh", "-c", "sleep infinity"}
					wlkdODFImage      = ECoreConfig.StorageODFWorkloadImage
				)

				By(fmt.Sprintf("Asserting namespace %s already exists", fNamespace))
				glog.V(ecoreparams.ECoreLogLevel).Infof(fmt.Sprintf("Assert if namespace %q exists", fNamespace))

				if workloadNS, err := namespace.Pull(APIClient, fNamespace); err == nil {
					glog.V(ecoreparams.ECoreLogLevel).Infof(fmt.Sprintf("Namespace %q exists. Removing...", fNamespace))
					delErr := workloadNS.DeleteAndWait(6 * time.Minute)
					Expect(delErr).ToNot(HaveOccurred(), fmt.Sprintf("Failed to delete %q namespace", fNamespace))
				}

				By(fmt.Sprintf("Creating %s namespace", fNamespace))
				workloadNS = namespace.NewBuilder(APIClient, fNamespace)
				workloadNS, err := workloadNS.Create()
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to create test namespace %s", fNamespace))
				glog.V(ecoreparams.ECoreLogLevel).Infof(fmt.Sprintf("Namespace %q created", workloadNS.Object.Name))

				By("Creating new PVC Builder")
				myPVC := storage.NewPVCBuilder(APIClient, fPVCName, fNamespace)
				glog.V(ecoreparams.ECoreLogLevel).Infof(fmt.Sprintf("PVC\n%#v", myPVC))

				By("Setting AccessMode")
				myPVC, err = myPVC.WithPVCAccessMode("ReadWriteOnce")
				Expect(err).ToNot(HaveOccurred(), "Failed to set AccessMode")
				glog.V(ecoreparams.ECoreLogLevel).Infof("PVC accessMode: ", myPVC.Definition.Spec.AccessModes)

				By("Setting PVC capacity")
				myPVC, err = myPVC.WithPVCCapacity("5G")
				Expect(err).ToNot(HaveOccurred(), "Failed to set Capacity")
				glog.V(ecoreparams.ECoreLogLevel).Infof("PVC Capacity: ", myPVC.Definition.Spec.Resources)

				By("Setting StorageClass for PVC")
				myPVC, err = myPVC.WithStorageClass(fStorageClass)
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to use StorageClass %q: %v", fStorageClass, err))
				glog.V(ecoreparams.ECoreLogLevel).Infof("PVC StorageClass: ", myPVC.Definition.Spec.StorageClassName)

				By("Creating PVC")
				myPVC, err = myPVC.Create()
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to create PVC %q: %v", myPVC.Definition.Name, err))
				glog.V(ecoreparams.ECoreLogLevel).Infof(
					fmt.Sprintf("Created PVC %q: %v", myPVC.Definition.Name, myPVC.Object.Status))

				By("Waiting for PVC to report phase")
				Eventually(func(phase string) bool {
					if ok := myPVC.Exists(); ok {
						glog.V(ecoreparams.ECoreLogLevel).Infof(fmt.Sprintf("\tPVC Phase is %q", myPVC.Object.Status.Phase))

						return string(myPVC.Object.Status.Phase) == phase
					}

					return false
				}, 5*time.Minute, 3*time.Second).WithContext(ctx).WithArguments("Bound").Should(BeTrue(),
					fmt.Sprintf("Unexpeced PVC state %q", myPVC.Object.Status.Phase))

				By("Checking deployment doesn't exist")

				deploy, err := deployment.Pull(APIClient, wlkdODFDeployName, fNamespace)
				if deploy != nil && err == nil {
					glog.V(ecoreparams.ECoreLogLevel).Infof("Deployment %q found in %q namespace. Deleting...",
						deploy.Definition.Name, fNamespace)

					err := deploy.DeleteAndWait(300 * time.Second)
					Expect(err).ToNot(HaveOccurred(),
						fmt.Sprintf("failed to delete deployment %q", wlkdODFDeployName))

				}

				By("Asserting pods from deployments are gone")
				Eventually(func() bool {
					oldPods, _ := pod.List(APIClient, fNamespace,
						metav1.ListOptions{LabelSelector: labelsWlkdOneString})

					return len(oldPods) == 0

				}, 6*time.Minute, 3*time.Second).WithContext(ctx).Should(BeTrue(), "pods matching label(s) still present")

				By("Defining container configuration")
				deployContainer := pod.NewContainerBuilder("one", wlkdODFImage, wlkdODFCmd)

				By("Adding VolumeMount to container")
				volMount := v1.VolumeMount{
					Name:      "cephfs-pvc",
					MountPath: "/opt/cephfs-pvc/",
					ReadOnly:  false,
				}

				deployContainer = deployContainer.WithVolumeMount(volMount)
				glog.V(ecoreparams.ECoreLogLevel).Infof("Container One definition: %#v", deployContainer)

				By("Setting SecurityContext")
				var falseFlag = false

				secContext := &v1.SecurityContext{
					Privileged: &falseFlag,
					SeccompProfile: &v1.SeccompProfile{
						Type: v1.SeccompProfileTypeRuntimeDefault,
					},
					Capabilities: &v1.Capabilities{},
				}

				By("Setting SecurityContext")
				deployContainer = deployContainer.WithSecurityContext(secContext)
				glog.V(ecoreparams.ECoreLogLevel).Infof("Container One definition: %#v", deployContainer)

				By("Obtaining container definition")
				deployContainerCfg, err := deployContainer.GetContainerCfg()
				Expect(err).ToNot(HaveOccurred(), "Failed to get container definition")

				By("Defining deployment configuration")
				deploy = deployment.NewBuilder(APIClient,
					wlkdODFDeployName,
					fNamespace,
					map[string]string{strings.Split(labelsWlkdOneString, "=")[0]: strings.Split(labelsWlkdOneString, "=")[1]},
					// map[string]string{"systemtest-test": "ecore-odf-pvc"},
					deployContainerCfg)

				By("Adding Volume to the deployment")
				volMode := new(int32)
				*volMode = 511

				volDefinition := v1.Volume{
					Name: "cephfs-pvc",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
							ClaimName: fPVCName,
							ReadOnly:  false,
						},
					},
				}

				deploy = deploy.WithVolume(volDefinition)

				glog.V(ecoreparams.ECoreLogLevel).Infof("Deployment's Volume:\n %v",
					deploy.Definition.Spec.Template.Spec.Volumes)

				By("Setting Replicas count")
				deploy = deploy.WithReplicas(int32(1))

				By("Adding NodeSelector to the deployment")
				deploy = deploy.WithNodeSelector(ECoreConfig.StorageODFDeployOneSelector)

				By("Creating a deployment")
				deploy, err = deploy.CreateAndWaitUntilReady(60 * time.Second)
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to create deployment %s: %v", deploy.Definition.Name, err))

				By("Getting pods backed by deployment")
				podOneSelector := metav1.ListOptions{
					LabelSelector: labelsWlkdOneString,
				}

				podOneList, err := pod.List(APIClient, fNamespace, podOneSelector)
				Expect(err).ToNot(HaveOccurred(), "Failed to find pods for deployment one")
				Expect(len(podOneList)).To(Equal(1), "Expected only one pod")

				podOne := podOneList[0]
				glog.V(ecoreparams.ECoreLogLevel).Infof("Pod one is %v on node %s",
					podOne.Definition.Name, podOne.Definition.Spec.NodeName)

				By("Writing data to CephFS backed storage")
				msgOne := fmt.Sprintf("Running from pod %s(%s) at %d",
					podOne.Definition.Name,
					podOne.Definition.Spec.NodeName,
					time.Now().Unix())

				glog.V(ecoreparams.ECoreLogLevel).Infof("Writing msg %q from pod %s",
					msgOne, podOne.Definition.Name)

				writeDataOneCmd := []string{"/bin/bash", "-c",
					fmt.Sprintf("echo '%s' > /opt/cephfs-pvc/demo-data-file", msgOne)}

				podOneResult, err := podOne.ExecCommand(writeDataOneCmd, "one")
				Expect(err).ToNot(HaveOccurred(),
					fmt.Sprintf("Failed to send data from pod %s", podOne.Definition.Name))
				glog.V(ecoreparams.ECoreLogLevel).Infof("Result: %v - %s", podOneResult, &podOneResult)

				By("Scaling down deployment")
				deploy = deploy.WithReplicas(int32(0))

				deploy, err = deploy.Update()
				Expect(err).ToNot(HaveOccurred(),
					fmt.Sprintf("Failed to scale down deployment %s in %s namespace",
						deploy.Definition.Name, deploy.Definition.Namespace))

				By("Asserting pods from deployments are gone")
				Eventually(func() bool {
					oldPods, _ := pod.List(APIClient, fNamespace,
						metav1.ListOptions{LabelSelector: labelsWlkdOneString})

					return len(oldPods) == 0

				}, 6*time.Minute, 3*time.Second).WithContext(ctx).Should(BeTrue(), "pods matching label(s) still present")

				By("Resetting NodeSelector on the deployment")
				deploy = deploy.WithNodeSelector(ECoreConfig.StorageODFDeployTwoSelector)

				By("Scaling up deployment")
				deploy = deploy.WithReplicas(int32(1))

				deploy, err = deploy.Update()
				Expect(err).ToNot(HaveOccurred(),
					fmt.Sprintf("Failed to scale up deployment %s in %s namespace",
						deploy.Definition.Name, deploy.Definition.Namespace))

				By("Asserting pods from deployments are gone")
				Eventually(func() bool {
					oldPods, _ := pod.List(APIClient, fNamespace,
						metav1.ListOptions{LabelSelector: labelsWlkdOneString})

					return len(oldPods) == 1

				}, 6*time.Minute, 3*time.Second).WithContext(ctx).Should(BeTrue(), "pods matching label(s) not found")

				podOneList, err = pod.List(APIClient, fNamespace, podOneSelector)
				Expect(err).ToNot(HaveOccurred(), "Failed to find pods for deployment one")
				Expect(len(podOneList)).To(Equal(1), "Expected only one pod")

				podOne = podOneList[0]
				glog.V(ecoreparams.ECoreLogLevel).Infof("Pod one is %v on node %s",
					podOne.Definition.Name, podOne.Definition.Spec.NodeName)

				By("Waiting until pod is running")
				err = podOne.WaitUntilReady(5 * time.Minute)
				Expect(err).ToNot(HaveOccurred(),
					fmt.Sprintf("Pod %s in %s namespace isn't running after 5 minutes",
						podOne.Definition.Name, podOne.Definition.Namespace))

				By("Reading data from CephFS backed storage")
				readDataOneCmd := []string{"/bin/bash", "-c", "cat /opt/cephfs-pvc/demo-data-file"}

				podOneResult, err = podOne.ExecCommand(readDataOneCmd, "one")
				Expect(err).ToNot(HaveOccurred(),
					fmt.Sprintf("Failed to read data from pod %s", podOne.Definition.Name))
				glog.V(ecoreparams.ECoreLogLevel).Infof(fmt.Sprintf("Result:\n\t%s", &podOneResult))
				Expect(podOneResult.String()).Should(ContainSubstring(msgOne))

			},
			Entry("CephFS StorageClass with FileSystem PVC",
				"verification-cephfs-ns", "ocs-external-storagecluster-cephfs", "telco-cephfs-fs", "Filesystem"),
			Entry("CephRBD StorageClass with FileSystem PVC",
				"verification-ceph-rbd-ns", "ocs-external-storagecluster-ceph-rbd", "telco-ceph-rbd-fs", "Filesystem"),
		) // end DescribeTable
	})
