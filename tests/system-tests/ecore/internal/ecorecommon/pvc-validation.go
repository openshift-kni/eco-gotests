package ecorecommon

import (
	"bytes"
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

const (
	labelsWlkdOneString = "systemtest-test=rdscore-odf-pvc"
	labelsWlkdTwoString = "systemtest-test=rdscore-odf-two"
	regexPartOne        = `Deployment[[:space:]]+[[:alnum:]-_]+;Pod[[:space:]]+[[:alnum:]-_]+`
	regexPartTwo        = `\([[:alnum:]-._]+\);Timestamp[[:space:]]+[[:digit:]]+`
)

func createPVC(fPVCName, fNamespace, fStorageClass, fVolumeMode, fCapacity string) *storage.PVCBuilder {
	By("Creating new PVC Builder")

	myPVC := storage.NewPVCBuilder(APIClient, fPVCName, fNamespace)
	glog.V(ecoreparams.ECoreLogLevel).Infof(fmt.Sprintf("PVC\n%#v", myPVC))

	By("Setting AccessMode")

	myPVC, err := myPVC.WithPVCAccessMode("ReadWriteOnce")
	Expect(err).ToNot(HaveOccurred(), "Failed to set AccessMode")
	glog.V(ecoreparams.ECoreLogLevel).Infof("PVC accessMode: %v", myPVC.Definition.Spec.AccessModes)

	By("Setting PVC capacity")

	myPVC, err = myPVC.WithPVCCapacity(fCapacity)
	Expect(err).ToNot(HaveOccurred(), "Failed to set Capacity")
	glog.V(ecoreparams.ECoreLogLevel).Infof("PVC Capacity: %#v", myPVC.Definition.Spec.Resources)

	By("Setting StorageClass for PVC")

	myPVC, err = myPVC.WithStorageClass(fStorageClass)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to use StorageClass %q: %v", fStorageClass, err))
	glog.V(ecoreparams.ECoreLogLevel).Infof("PVC StorageClass: %s", myPVC.Definition.Spec.StorageClassName)

	By("Setting VolumeMode")

	myPVC, err = myPVC.WithVolumeMode(fVolumeMode)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to set VolumeMode %q: %v", fVolumeMode, err))
	glog.V(ecoreparams.ECoreLogLevel).Infof("PVC VolumeMode: %s", myPVC.Definition.Spec.VolumeMode)

	By("Creating PVC")

	myPVC, err = myPVC.Create()
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to create PVC %q: %v", myPVC.Definition.Name, err))
	glog.V(ecoreparams.ECoreLogLevel).Infof(
		fmt.Sprintf("Created PVC %q: %v", myPVC.Definition.Name, myPVC.Object.Status))

	return myPVC
}

//nolint:funlen
func createWorkloadWithPVC(fNamespace string, fStorageClass string, fPVCName string, fVolumeMode string) {
	var (
		ctx               SpecContext
		workloadNS        *namespace.Builder
		wlkdODFDeployName = "rds-core-wlkd"
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

	myPVC := createPVC(fPVCName, fNamespace, fStorageClass, fVolumeMode, "5G")

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
	glog.V(ecoreparams.ECoreLogLevel).Infof("Deployment's NodeSlector:\n\t%v",
		ECoreConfig.StorageODFDeployOneSelector)

	deploy = deploy.WithNodeSelector(ECoreConfig.StorageODFDeployOneSelector)

	By("Creating a deployment")

	deploy, err = deploy.CreateAndWaitUntilReady(5 * time.Minute)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to create deployment %s: %v", wlkdODFDeployName, err))

	By("Getting pods backed by deployment")

	podOneSelector := metav1.ListOptions{
		LabelSelector: labelsWlkdOneString,
	}

	var podOneList []*pod.Builder

	Eventually(func() bool {
		podOneList, err = pod.List(APIClient, fNamespace, podOneSelector)
		if err != nil {
			glog.V(ecoreparams.ECoreLogLevel).Infof("Failed to list pods in %q namespace: %v",
				fNamespace, err)

			return false
		}

		if len(podOneList) == 1 {
			glog.V(ecoreparams.ECoreLogLevel).Infof("Found 1 pod matching label %q in namespace %q",
				labelsWlkdOneString, fNamespace)

			return true
		}

		return false
	}).WithContext(ctx).WithPolling(15*time.Second).WithTimeout(5*time.Minute).Should(BeTrue(),
		fmt.Sprintf("Failed to find pod matching label %q in %q namespace", labelsWlkdOneString, fNamespace))

	podOne := podOneList[0]
	glog.V(ecoreparams.ECoreLogLevel).Infof("Pod one is %v on node %s",
		podOne.Definition.Name, podOne.Definition.Spec.NodeName)

	By("Writing data to persistent storage")

	msgOne := fmt.Sprintf("Deployment %s;Pod %s(%s);Timestamp %d",
		deploy.Definition.Name,
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

	glog.V(ecoreparams.ECoreLogLevel).Infof("Scaling down deployment %q in %q namespace",
		deploy.Definition.Name, deploy.Definition.Namespace)

	deploy = deploy.WithReplicas(int32(0))

	deploy, err = deploy.Update()
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to scale down deployment %s in %s namespace",
			deploy.Definition.Name, deploy.Definition.Namespace))

	By("Asserting pods from deployments are gone")

	glog.V(ecoreparams.ECoreLogLevel).Infof("Check pods from deployment %q in are gone",
		deploy.Definition.Name)

	Eventually(func() bool {
		oldPods, _ := pod.List(APIClient, fNamespace,
			metav1.ListOptions{LabelSelector: labelsWlkdOneString})

		return len(oldPods) == 0
	}, 6*time.Minute, 3*time.Second).WithContext(ctx).Should(BeTrue(), "pods matching label(s) still present")

	By("Resetting NodeSelector on the deployment")

	glog.V(ecoreparams.ECoreLogLevel).Infof("Updating nodeSelector for deployment %q",
		deploy.Definition.Name)

	deploy = deploy.WithNodeSelector(ECoreConfig.StorageODFDeployTwoSelector)

	By("Scaling up deployment")

	glog.V(ecoreparams.ECoreLogLevel).Infof("Scaling up deployment %q in %q namespace",
		deploy.Definition.Name, deploy.Definition.Namespace)

	deploy = deploy.WithReplicas(int32(1))

	deploy, err = deploy.Update()
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to scale up deployment %s in %s namespace",
			deploy.Definition.Name, deploy.Definition.Namespace))

	By("Asserting new pods from deployments are present")

	glog.V(ecoreparams.ECoreLogLevel).Infof("Check pods from deployment %q in are present",
		deploy.Definition.Name)

	Eventually(func() bool {
		podOneList, err = pod.List(APIClient, fNamespace, podOneSelector)
		if err != nil {
			glog.V(ecoreparams.ECoreLogLevel).Infof("Failed to list pods in %q namespace: %v",
				fNamespace, err)

			return false
		}

		if len(podOneList) == 1 {
			glog.V(ecoreparams.ECoreLogLevel).Infof("Found 1 pod matching label %q in namespace %q",
				labelsWlkdOneString, fNamespace)

			return true
		}

		return false
	}).WithContext(ctx).WithPolling(15*time.Second).WithTimeout(5*time.Minute).Should(BeTrue(),
		fmt.Sprintf("Failed to find pod matching label %q in %q namespace", labelsWlkdOneString, fNamespace))

	podOne = podOneList[0]
	glog.V(ecoreparams.ECoreLogLevel).Infof("Pod one is %v on node %s",
		podOne.Definition.Name, podOne.Definition.Spec.NodeName)

	By("Waiting until pod is running")

	glog.V(ecoreparams.ECoreLogLevel).Infof("Waiting 5 minutes for pod %q to be Ready",
		podOne.Definition.Name)

	err = podOne.WaitUntilReady(5 * time.Minute)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Pod %s in %s namespace isn't running after 5 minutes",
			podOne.Definition.Name, podOne.Definition.Namespace))

	By("Reading data from CephFS backed storage")

	readDataOneCmd := []string{"/bin/bash", "-c", "cat /opt/cephfs-pvc/demo-data-file"}

	glog.V(ecoreparams.ECoreLogLevel).Infof("Resetting command's output buffer")
	podOneResult.Reset()

	Eventually(func() bool {
		podOneResult, err = podOne.ExecCommand(readDataOneCmd, "one")

		if err != nil {
			glog.V(ecoreparams.ECoreLogLevel).Infof("Failed to run command on pod %s - %v",
				podOne.Definition.Name, err)

			return false
		}

		glog.V(ecoreparams.ECoreLogLevel).Infof(fmt.Sprintf("Command's result:\n\t%s",
			&podOneResult))

		return true
	}).WithContext(ctx).WithPolling(5*time.Second).WithTimeout(1*time.Minute).Should(BeTrue(),
		fmt.Sprintf("Failed to run command in pod %q", podOne.Definition.Name))

	verificationRegex := `Deployment[[:space:]]+[[:alnum:]-_]+;Pod[[:space:]]+[[:alnum:]-_]+` +
		`\([[:alnum:]-._]+\);Timestamp[[:space:]]+[[:digit:]]+`
	Expect(podOneResult.String()).Should(MatchRegexp(verificationRegex), "Command's output doesn't match regex")
}

func verifyDataOnPVC(fNamespace, podLabel, verificationRegex string, cmdToRun []string) {
	By(fmt.Sprintf("Getting pod(s) matching selector %q", podLabel))

	var (
		podMatchingSelector []*pod.Builder
		err                 error
		ctx                 SpecContext
		podCommandResult    bytes.Buffer
	)

	podOneSelector := metav1.ListOptions{
		LabelSelector: podLabel,
	}

	glog.V(ecoreparams.ECoreLogLevel).Infof("Looking for pods with label %q in %q namespace",
		podLabel, fNamespace)

	Eventually(func() bool {
		podMatchingSelector, err = pod.List(APIClient, fNamespace, podOneSelector)
		if err != nil {
			glog.V(ecoreparams.ECoreLogLevel).Infof("Failed to list pods in %q namespace: %v",
				fNamespace, err)

			return false
		}

		if len(podMatchingSelector) == 0 {
			glog.V(ecoreparams.ECoreLogLevel).Infof("Found 0 pods matching label %q in namespace %q",
				podLabel, fNamespace)

			return false
		}

		return true
	}).WithContext(ctx).WithPolling(15*time.Second).WithTimeout(5*time.Minute).Should(BeTrue(),
		fmt.Sprintf("Failed to find pod matching label %q in %q namespace", podLabel, fNamespace))

	By("Waiting until pod(s) is running")

	for _, podOne := range podMatchingSelector {
		glog.V(ecoreparams.ECoreLogLevel).Infof("Waiting 5 minutes for pod %q in %q namespace to be Ready",
			podOne.Definition.Name, podOne.Definition.Namespace)

		err = podOne.WaitUntilReady(5 * time.Minute)
		Expect(err).ToNot(HaveOccurred(),
			fmt.Sprintf("Pod %s in %s namespace isn't running after 5 minutes",
				podOne.Definition.Name, podOne.Definition.Namespace))
	}

	By("Reading data from CephFS backed storage")

	for _, podOne := range podMatchingSelector {
		glog.V(ecoreparams.ECoreLogLevel).Infof("Reading data from within pod %q in %q namespace",
			podOne.Definition.Name, podOne.Definition.Namespace)
		glog.V(ecoreparams.ECoreLogLevel).Infof("Resetting command's output buffer")

		podCommandResult.Reset()

		Eventually(func() bool {
			podCommandResult, err = podOne.ExecCommand(cmdToRun, "one")

			if err != nil {
				glog.V(ecoreparams.ECoreLogLevel).Infof("Failed to run command on pod %s - %v",
					podOne.Definition.Name, err)

				return false
			}

			glog.V(ecoreparams.ECoreLogLevel).Infof(fmt.Sprintf("Command's result:\n\t%s",
				&podCommandResult))

			return true
		}).WithContext(ctx).WithPolling(5*time.Second).WithTimeout(1*time.Minute).Should(BeTrue(),
			fmt.Sprintf("Failed to run command in pod %q", podOne.Definition.Name))

		Expect(podCommandResult.String()).Should(MatchRegexp(verificationRegex), "Command's output doesn't match regex")
	}
}

// VerifyCephFSPVC Verify workload with CephFS PVC.
func VerifyCephFSPVC(ctx SpecContext) {
	createWorkloadWithPVC("rds-cephfs-ns", "ocs-external-storagecluster-cephfs", "rds-cephfs-fs", "Filesystem")
}

// VerifyCephRBDPVC Verify workload with CephRBD PVC.
func VerifyCephRBDPVC(ctx SpecContext) {
	createWorkloadWithPVC("rds-cephrbd-ns", "ocs-external-storagecluster-ceph-rbd", "rds-cephrbd-fs", "Filesystem")
}

// VerifyDataOnCephFSPVC verify data on CephFS PVC.
func VerifyDataOnCephFSPVC(ctx SpecContext) {
	glog.V(ecoreparams.ECoreLogLevel).Infof("Verify data on CephFS PVC")

	verificationRegex := regexPartOne + regexPartTwo

	cmdToRun := []string{"/bin/bash", "-c", "cat /opt/cephfs-pvc/demo-data-file"}

	verifyDataOnPVC("rds-cephfs-ns", labelsWlkdOneString, verificationRegex, cmdToRun)
}

// VerifyDataOnCephRBDPVC verify data on CephRBD PVC.
func VerifyDataOnCephRBDPVC(ctx SpecContext) {
	glog.V(ecoreparams.ECoreLogLevel).Infof("Verify data on CephRBD PVC")

	verificationRegex := regexPartOne + regexPartTwo

	cmdToRun := []string{"/bin/bash", "-c", "cat /opt/cephfs-pvc/demo-data-file"}

	verifyDataOnPVC("rds-cephrbd-ns", labelsWlkdOneString, verificationRegex, cmdToRun)
}

// VerifyPersistentStorageSuite container that contains tests for persistent storage verification.
func VerifyPersistentStorageSuite() {
	Describe(
		"Persistent storage validation",
		Label("ecore-persistent-storage"), func() {
			It("Verifies CephFS",
				Label("odf-cephfs-pvc"), polarion.ID("71850"), MustPassRepeatedly(3), VerifyCephFSPVC)

			It("Verifies CephRBD",
				Label("odf-cephrbd-pvc"), polarion.ID("71989"), MustPassRepeatedly(3), VerifyCephRBDPVC)
		})
}