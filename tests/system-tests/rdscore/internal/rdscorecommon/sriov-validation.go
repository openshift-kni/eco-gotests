package rdscorecommon

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/configmap"
	"github.com/openshift-kni/eco-goinfra/pkg/deployment"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-goinfra/pkg/rbac"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"github.com/openshift-kni/eco-goinfra/pkg/serviceaccount"

	multus "gopkg.in/k8snetworkplumbingwg/multus-cni.v4/pkg/types"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/openshift-kni/eco-gotests/tests/system-tests/rdscore/internal/rdscoreinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/rdscore/internal/rdscoreparams"
)

const (
	// Names of deployments.
	sriovDeploy1OneName = "rdscore-sriov-one"
	sriovDeploy1TwoName = "rdscore-sriov-two"
	sriovDeploy2OneName = "rdscore-sriov2-one"
	sriovDeploy2TwoName = "rdscore-sriov2-two"
	// ConfigMap names.
	sriovDeploy1CMName = "rdscore-sriov-config"
	sriovDeploy2CMName = "rdscore-sriov2-config"
	// ServiceAccount names.
	sriovDeploy1SAName = "rdscore-sriov-sa-one"
	sriovDeploy2SAName = "rdscore-sriov-sa-two"
	// Container names within deployments.
	sriovContainerOneName = "sriov-one"
	sriovContainerTwoName = "sriov-two"
	// Labels for deployments.
	sriovDeployOneLabel  = "rds-core=sriov-deploy-one"
	sriovDeployTwoLabel  = "rds-core=sriov-deploy-two"
	sriovDeploy2OneLabel = "rds-core=sriov-deploy2-one"
	sriovDeploy2TwoLabel = "rds-core=sriov-deploy2-two"
	// RBAC names for the deployments.
	sriovDeployRBACName  = "privileged-rdscore-sriov"
	sriovDeployRBACName2 = "privileged-rdscore-sriov2"
	// ClusterRole to use with RBAC.
	sriovRBACRole  = "system:openshift:scc:privileged"
	sriovRBACRole2 = "system:openshift:scc:privileged"
)

func createServiceAccount(saName, nsName string) {
	By(fmt.Sprintf("Creating ServiceAccount %q in %q namespace",
		saName, nsName))
	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Creating SA %q in %q namespace",
		saName, nsName)

	deploySa := serviceaccount.NewBuilder(APIClient, saName, nsName)

	var ctx SpecContext

	Eventually(func() bool {
		deploySa, err := deploySa.Create()

		if err != nil {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Error creating SA %q in %q namespace: %v",
				saName, nsName, err)

			return false
		}

		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Created SA %q in %q namespace",
			deploySa.Definition.Name, deploySa.Definition.Namespace)

		return true
	}).WithContext(ctx).WithPolling(5*time.Second).WithTimeout(1*time.Minute).Should(BeTrue(),
		fmt.Sprintf("Failed to create ServiceAccount %q in %q namespace", saName, nsName))
}

func deleteServiceAccount(saName, nsName string) {
	By("Removing Service Account")
	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Assert SA %q exists in %q namespace",
		saName, nsName)

	var ctx SpecContext

	if deploySa, err := serviceaccount.Pull(
		APIClient, saName, nsName); err == nil {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("ServiceAccount %q found in %q namespace",
			saName, nsName)
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Deleting ServiceAccount %q in %q namespace",
			saName, nsName)

		Eventually(func() bool {
			err := deploySa.Delete()

			if err != nil {
				glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Error deleting ServiceAccount %q in %q namespace: %v",
					saName, nsName, err)

				return false
			}

			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Deleted ServiceAccount %q in %q namespace",
				saName, nsName)

			return true
		}).WithContext(ctx).WithPolling(5*time.Second).WithTimeout(1*time.Minute).Should(BeTrue(),
			fmt.Sprintf("Failed to delete ServiceAccount %q from %q ns", saName, nsName))
	} else {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("ServiceAccount %q not found in %q namespace",
			saName, nsName)
	}
}

func deleteClusterRBAC(rbacName string) {
	By("Deleting Cluster RBAC")

	var ctx SpecContext

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Assert ClusterRoleBinding %q exists", rbacName)

	if crbSa, err := rbac.PullClusterRoleBinding(
		APIClient,
		rbacName); err == nil {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("ClusterRoleBinding %q found. Deleting...", rbacName)

		Eventually(func() bool {
			err := crbSa.Delete()

			if err != nil {
				glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Error deleting ClusterRoleBinding %q : %v",
					rbacName, err)

				return false
			}

			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Deleted ClusterRoleBinding %q", rbacName)

			return true
		}).WithContext(ctx).WithPolling(5*time.Second).WithTimeout(1*time.Minute).Should(BeTrue(),
			"Failed to delete Cluster RBAC")
	}
}

//nolint:unparam
func createClusterRBAC(rbacName, clusterRole, saName, nsName string) {
	By("Creating RBAC for SA")

	var ctx SpecContext

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Creating ClusterRoleBinding %q", rbacName)
	crbSa := rbac.NewClusterRoleBindingBuilder(APIClient,
		rbacName,
		clusterRole,
		rbacv1.Subject{
			Name:      saName,
			Kind:      "ServiceAccount",
			Namespace: nsName,
		})

	Eventually(func() bool {
		crbSa, err := crbSa.Create()
		if err != nil {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof(
				"Error Creating ClusterRoleBinding %q : %v", crbSa.Definition.Name, err)

			return false
		}

		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("ClusterRoleBinding %q created:\n\t%v",
			crbSa.Definition.Name, crbSa)

		return true
	}).WithContext(ctx).WithPolling(5*time.Second).WithTimeout(1*time.Minute).Should(BeTrue(),
		"Failed to create ClusterRoleBinding")
}

func deleteConfigMap(cmName, nsName string) {
	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Assert ConfigMap %q exists in %q namespace",
		cmName, nsName)

	if cmBuilder, err := configmap.Pull(
		APIClient, cmName, nsName); err == nil {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("configMap %q found, deleting", cmName)

		var ctx SpecContext

		Eventually(func() bool {
			err := cmBuilder.Delete()
			if err != nil {
				glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Error deleting configMap %q : %v",
					cmName, err)

				return false
			}

			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Deleted configMap %q in %q namespace",
				cmName, nsName)

			return true
		}).WithContext(ctx).WithPolling(5*time.Second).WithTimeout(1*time.Minute).Should(BeTrue(),
			"Failed to delete configMap")
	}
}

func createConfigMap(cmName, nsName string, data map[string]string) {
	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Create ConfigMap %q in %q namespace",
		cmName, nsName)

	cmBuilder := configmap.NewBuilder(APIClient, cmName, nsName)
	cmBuilder.WithData(data)

	var ctx SpecContext

	Eventually(func() bool {

		cmResult, err := cmBuilder.Create()
		if err != nil {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Error creating ConfigMap %q in %q namespace",
				cmName, nsName)

			return false
		}

		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Created ConfigMap %q in %q namespace",
			cmResult.Definition.Name, nsName)

		return true
	}).WithContext(ctx).WithPolling(5*time.Second).WithPolling(1*time.Minute).Should(BeTrue(),
		"Failed to crete configMap")
}

func deleteDeployments(dName, nsName string) {
	By(fmt.Sprintf("Removing test deployment %q from %q ns", dName, nsName))

	if deploy, err := deployment.Pull(APIClient, dName, nsName); err == nil {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Deleting deployment %q from %q namespace",
			deploy.Definition.Name, nsName)

		err = deploy.DeleteAndWait(300 * time.Second)
		Expect(err).ToNot(HaveOccurred(),
			fmt.Sprintf("failed to delete deployment %q", dName))
	} else {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Deployment %q not found in %q namespace",
			dName, nsName)
	}
}

func findPodWithSelector(fNamespace, podLabel string) []*pod.Builder {
	By(fmt.Sprintf("Getting pod(s) matching selector %q", podLabel))

	var (
		podMatchingSelector []*pod.Builder
		err                 error
		ctx                 SpecContext
	)

	podOneSelector := metav1.ListOptions{
		LabelSelector: podLabel,
	}

	Eventually(func() bool {
		podMatchingSelector, err = pod.List(APIClient, fNamespace, podOneSelector)
		if err != nil {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Failed to list pods in %q namespace: %v",
				fNamespace, err)

			return false
		}

		if len(podMatchingSelector) == 0 {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Found 0 pods matching label %q in namespace %q",
				podLabel, fNamespace)

			return false
		}

		return true
	}).WithContext(ctx).WithPolling(15*time.Second).WithTimeout(5*time.Minute).Should(BeTrue(),
		fmt.Sprintf("Failed to find pod matching label %q in %q namespace", podLabel, fNamespace))

	return podMatchingSelector
}

func defineContainer(cName, cImage string, cCmd []string, cRequests, cLimits map[string]string) *pod.ContainerBuilder {
	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Creating container %q", cName)
	deployContainer := pod.NewContainerBuilder(cName, cImage, cCmd)

	By("Defining SecurityContext")

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

	By("Setting SecurityContext")

	deployContainer = deployContainer.WithSecurityContext(secContext)
	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Container One definition: %#v", deployContainer)

	By("Dropping ALL security capability")

	deployContainer = deployContainer.WithDropSecurityCapabilities([]string{"ALL"}, true)

	By("Adding VolumeMount to container")

	volMount := corev1.VolumeMount{
		Name:      "configs",
		MountPath: "/opt/net/",
		ReadOnly:  false,
	}

	deployContainer = deployContainer.WithVolumeMount(volMount)

	if len(cRequests) != 0 {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Processing container's requests")

		containerRequests := corev1.ResourceList{}

		for key, val := range cRequests {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Parsing container's request: %q - %q", key, val)

			containerRequests[corev1.ResourceName(key)] = resource.MustParse(val)
		}

		deployContainer = deployContainer.WithCustomResourcesRequests(containerRequests)
	}

	if len(cLimits) != 0 {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Processing container's limits")

		containerLimits := corev1.ResourceList{}

		for key, val := range cLimits {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Parsing container's limit: %q - %q", key, val)

			containerLimits[corev1.ResourceName(key)] = resource.MustParse(val)
		}

		deployContainer = deployContainer.WithCustomResourcesLimits(containerLimits)
	}

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("%q container's  definition:\n%#v", cName, deployContainer)

	return deployContainer
}

func defineDeployment(containerConfig *corev1.Container, deployName, deployNs, sriovNet, cmName, saName string,
	deployLabels, nodeSelector map[string]string) *deployment.Builder {
	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Defining deployment %q in %q ns", deployName, deployNs)

	deploy := deployment.NewBuilder(APIClient, deployName, deployNs, deployLabels, containerConfig)

	By("Defining SR-IOV annotations")

	var networksOne []*multus.NetworkSelectionElement

	networksOne = append(networksOne,
		&multus.NetworkSelectionElement{
			Name: sriovNet})

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("SR-IOV networks: %#v", networksOne)

	By("Adding SR-IOV annotations")

	deploy = deploy.WithSecondaryNetwork(networksOne)

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("SR-IOV deploy one: %#v",
		deploy.Definition.Spec.Template.ObjectMeta.Annotations)

	By("Adding NodeSelector to the deployment")

	deploy = deploy.WithNodeSelector(nodeSelector)

	By("Adding Volume to the deployment")

	volMode := new(int32)
	*volMode = 511

	volDefinition := corev1.Volume{
		Name: "configs",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				DefaultMode: volMode,
				LocalObjectReference: corev1.LocalObjectReference{
					Name: cmName,
				},
			},
		},
	}

	deploy = deploy.WithVolume(volDefinition)

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("SR-IOV One Volume:\n %v",
		deploy.Definition.Spec.Template.Spec.Volumes)

	By(fmt.Sprintf("Assigning ServiceAccount %q to the deployment", saName))

	deploy = deploy.WithServiceAccountName(saName)

	By("Setting Replicas count")

	deploy = deploy.WithReplicas(int32(1))

	if len(RDSCoreConfig.WlkdTolerationList) > 0 {
		By("Adding TaintToleration")

		for _, toleration := range RDSCoreConfig.WlkdTolerationList {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Adding toleration: %v", toleration)

			deploy = deploy.WithToleration(toleration)
		}
	}

	return deploy
}

func verifyMsgInPodLogs(podObj *pod.Builder, msg, cName string, timeSpan time.Time) {
	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Parsing duration %q", timeSpan)

	var (
		podLog string
		err    error
		ctx    SpecContext
	)

	Eventually(func() bool {
		logStartTimestamp := time.Since(timeSpan)
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("\tTime duration is %s", logStartTimestamp)

		if logStartTimestamp.Abs().Seconds() < 1 {
			logStartTimestamp, err = time.ParseDuration("1s")
			Expect(err).ToNot(HaveOccurred(), "Failed to parse time duration")
		}

		podLog, err = podObj.GetLog(logStartTimestamp, cName)

		if err != nil {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Failed to get logs from pod %q: %v", podObj.Definition.Name, err)

			return false
		}

		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Logs from pod %s:\n%s", podObj.Definition.Name, podLog)

		return true
	}).WithContext(ctx).WithPolling(5*time.Second).WithTimeout(1*time.Minute).Should(BeTrue(),
		fmt.Sprintf("Failed to get logs from pod %q", podObj.Definition.Name))

	Expect(podLog).Should(ContainSubstring(msg))
}

func verifySRIOVConnectivity(nsOneName, nsTwoName, deployOneLabels, deployTwoLabels, targetAddr string) {
	By("Getting pods backed by deployment")

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Looking for pod(s) matching label %q in %q namespace",
		deployOneLabels, nsOneName)

	podOneList := findPodWithSelector(nsOneName, deployOneLabels)
	Expect(len(podOneList)).To(Equal(1), "Expected only one pod")

	podOne := podOneList[0]
	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Pod one is %q on node %q",
		podOne.Definition.Name, podOne.Definition.Spec.NodeName)

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Looking for pod(s) matching label %q in %q namespace",
		deployTwoLabels, nsTwoName)

	podTwoList := findPodWithSelector(nsTwoName, deployTwoLabels)
	Expect(len(podTwoList)).To(Equal(1), "Expected only one pod")

	podTwo := podTwoList[0]
	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Pod two is %q on node %q",
		podTwo.Definition.Name, podTwo.Definition.Spec.NodeName)

	By("Sending data from pod one to pod two")

	msgOne := fmt.Sprintf("Running from pod %s(%s) at %d",
		podOne.Definition.Name,
		podOne.Definition.Spec.NodeName,
		time.Now().Unix())

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Sending msg %q from pod %s",
		msgOne, podOne.Definition.Name)

	sendDataOneCmd := []string{"/bin/bash", "-c",
		fmt.Sprintf("echo '%s' | nc %s", msgOne, targetAddr)}

	var (
		podOneResult bytes.Buffer
		err          error
		ctx          SpecContext
	)

	timeStart := time.Now()

	Eventually(func() bool {
		podOneResult, err = podOne.ExecCommand(sendDataOneCmd, podOne.Definition.Spec.Containers[0].Name)

		if err != nil {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Failed to run command within pod: %v", sendDataOneCmd)
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Failed to run command within pod: %v", err)

			return false
		}

		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Successfully run command %v within container", sendDataOneCmd)
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Successfully run command within container %q",
			podOne.Definition.Spec.Containers[0].Name)
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Result: %v - %s", podOneResult, &podOneResult)

		return true
	}).WithContext(ctx).WithPolling(5*time.Second).WithTimeout(5*time.Minute).Should(BeTrue(),
		fmt.Sprintf("Failed to send data from pod %s", podOne.Definition.Name))

	verifyMsgInPodLogs(podTwo, msgOne, podTwo.Definition.Spec.Containers[0].Name, timeStart)
}

// VerifySRIOVWorkloadsOnSameNode deploy worklods with SRIOV interfaces on the same node
//
//nolint:funlen
func VerifySRIOVWorkloadsOnSameNode(ctx SpecContext) {
	By("Checking SR-IOV deployments don't exist")

	deleteDeployments(sriovDeploy1OneName, RDSCoreConfig.WlkdSRIOVOneNS)
	deleteDeployments(sriovDeploy1TwoName, RDSCoreConfig.WlkdSRIOVOneNS)

	By(fmt.Sprintf("Ensuring pods from %q deployment are gone", sriovDeploy1OneName))

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Ensuring pods from %q deployment in %q namespace are gone",
		sriovDeploy1OneName, RDSCoreConfig.WlkdSRIOVOneNS)

	Eventually(func() bool {
		oldPods, _ := pod.List(APIClient, RDSCoreConfig.WlkdSRIOVOneNS,
			metav1.ListOptions{LabelSelector: sriovDeployOneLabel})

		return len(oldPods) == 0

	}, 6*time.Minute, 3*time.Second).WithContext(ctx).Should(BeTrue(), "pods matching label() still present")

	By(fmt.Sprintf("Ensuring pods from %q deployment are gone", sriovDeploy1TwoName))

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Ensuring pods from %q deployment in %q namespace are gone",
		sriovDeploy1TwoName, RDSCoreConfig.WlkdSRIOVOneNS)

	Eventually(func() bool {
		oldPods, _ := pod.List(APIClient, RDSCoreConfig.WlkdSRIOVOneNS,
			metav1.ListOptions{LabelSelector: sriovDeployTwoLabel})

		return len(oldPods) == 0

	}, 6*time.Minute, 3*time.Second).WithContext(ctx).Should(BeTrue(), "pods matching label() still present")

	By("Sleeping 90 seconds")
	time.Sleep(90 * time.Second)

	By("Removing ConfigMap")

	deleteConfigMap(sriovDeploy1CMName, RDSCoreConfig.WlkdSRIOVOneNS)

	By("Creating ConfigMap")

	createConfigMap(sriovDeploy1CMName,
		RDSCoreConfig.WlkdSRIOVOneNS, RDSCoreConfig.WlkdSRIOVConfigMapDataOne)

	By("Removing ServiceAccount")
	deleteServiceAccount(sriovDeploy1SAName, RDSCoreConfig.WlkdSRIOVOneNS)

	By("Creating ServiceAccount")
	createServiceAccount(sriovDeploy1SAName, RDSCoreConfig.WlkdSRIOVOneNS)

	By("Remoing Cluster RBAC")
	deleteClusterRBAC(sriovDeployRBACName)

	By("Creating Cluster RBAC")
	createClusterRBAC(sriovDeployRBACName, sriovRBACRole,
		sriovDeploy1SAName, RDSCoreConfig.WlkdSRIOVOneNS)

	By("Defining container configuration")

	deployContainer := defineContainer(sriovContainerOneName, RDSCoreConfig.WlkdSRIOVDeployOneImage,
		RDSCoreConfig.WlkdSRIOVDeployOneCmd, RDSCoreConfig.WldkSRIOVDeployOneResRequests,
		RDSCoreConfig.WldkSRIOVDeployOneResLimits)

	deployContainerTwo := defineContainer(sriovContainerTwoName, RDSCoreConfig.WlkdSRIOVDeployTwoImage,
		RDSCoreConfig.WlkdSRIOVDeployTwoCmd, RDSCoreConfig.WldkSRIOVDeployOneResRequests,
		RDSCoreConfig.WldkSRIOVDeployOneResLimits)

	By("Obtaining container definition")

	deployContainerCfg, err := deployContainer.GetContainerCfg()
	Expect(err).ToNot(HaveOccurred(), "Failed to get container definition")

	deployContainerTwoCfg, err := deployContainerTwo.GetContainerCfg()
	Expect(err).ToNot(HaveOccurred(), "Failed to get container definition")

	By("Defining 1st deployment configuration")

	deployOneLabels := map[string]string{
		strings.Split(sriovDeployOneLabel, "=")[0]: strings.Split(sriovDeployOneLabel, "=")[1]}

	deploy := defineDeployment(deployContainerCfg,
		sriovDeploy1OneName,
		RDSCoreConfig.WlkdSRIOVOneNS,
		RDSCoreConfig.WlkdSRIOVNetOne,
		sriovDeploy1CMName,
		sriovDeploy1SAName,
		deployOneLabels,
		RDSCoreConfig.WlkdSRIOVDeployOneSelector)

	By("Creating deployment one")

	deploy, err = deploy.CreateAndWaitUntilReady(5 * time.Minute)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to create deployment %s: %v", deploy.Definition.Name, err))

	By("Defining 2nd deployment")

	deployTwoLabels := map[string]string{
		strings.Split(sriovDeployTwoLabel, "=")[0]: strings.Split(sriovDeployTwoLabel, "=")[1]}

	deployTwo := defineDeployment(deployContainerTwoCfg,
		sriovDeploy1TwoName,
		RDSCoreConfig.WlkdSRIOVOneNS,
		RDSCoreConfig.WlkdSRIOVNetOne,
		sriovDeploy1CMName,
		sriovDeploy1SAName,
		deployTwoLabels,
		RDSCoreConfig.WlkdSRIOVDeployOneSelector)

	By("Creating 2nd deployment")

	deployTwo, err = deployTwo.CreateAndWaitUntilReady(5 * time.Minute)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to create deployment %s: %v", deployTwo.Definition.Name, err))

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Verify connectivity between SR-IOV workloads on the same node")

	addressesList := []string{RDSCoreConfig.WlkdSRIOVDeployOneTargetAddress,
		RDSCoreConfig.WlkdSRIOVDeployOneTargetAddressIPv6}

	for _, targetAddress := range addressesList {
		if targetAddress == "" {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Skipping empty address %q", targetAddress)

			continue
		}

		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Access workload via %q", targetAddress)

		verifySRIOVConnectivity(
			RDSCoreConfig.WlkdSRIOVOneNS,
			RDSCoreConfig.WlkdSRIOVOneNS,
			sriovDeployOneLabel,
			sriovDeployTwoLabel,
			targetAddress)
	}

	addressesList = []string{RDSCoreConfig.WlkdSRIOVDeployTwoTargetAddress,
		RDSCoreConfig.WlkdSRIOVDeployTwoTargetAddressIPv6}

	for _, targetAddress := range addressesList {
		if targetAddress == "" {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Skipping empty address %q", targetAddress)

			continue
		}

		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Access workload via %q", targetAddress)

		verifySRIOVConnectivity(
			RDSCoreConfig.WlkdSRIOVOneNS,
			RDSCoreConfig.WlkdSRIOVOneNS,
			sriovDeployTwoLabel,
			sriovDeployOneLabel,
			targetAddress)
	}
}

// VerifySRIOVWorkloadsOnDifferentNodes deploy worklods with SRIOV interfaces on the same node
// Test config:
//
//	Same SR-IOV network
//	Same Namespace
//	Different nodes
//
//nolint:funlen
func VerifySRIOVWorkloadsOnDifferentNodes(ctx SpecContext) {
	By("Checking SR-IOV deployments don't exist")

	deleteDeployments(sriovDeploy2OneName, RDSCoreConfig.WlkdSRIOVOneNS)
	deleteDeployments(sriovDeploy2TwoName, RDSCoreConfig.WlkdSRIOVOneNS)

	By(fmt.Sprintf("Ensuring pods from %q deployment are gone", sriovDeploy2OneName))

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Ensuring pods from %q deployment in %q namespace are gone",
		sriovDeploy2OneName, RDSCoreConfig.WlkdSRIOVOneNS)

	Eventually(func() bool {
		oldPods, _ := pod.List(APIClient, RDSCoreConfig.WlkdSRIOVOneNS,
			metav1.ListOptions{LabelSelector: sriovDeploy2OneLabel})

		return len(oldPods) == 0

	}, 6*time.Minute, 3*time.Second).WithContext(ctx).Should(BeTrue(), "pods matching label() still present")

	By(fmt.Sprintf("Ensuring pods from %q deployment are gone", sriovDeploy2TwoName))

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Ensuring pods from %q deployment in %q namespace are gone",
		sriovDeploy2TwoName, RDSCoreConfig.WlkdSRIOVOneNS)

	Eventually(func() bool {
		oldPods, _ := pod.List(APIClient, RDSCoreConfig.WlkdSRIOVOneNS,
			metav1.ListOptions{LabelSelector: sriovDeploy2TwoLabel})

		return len(oldPods) == 0

	}, 6*time.Minute, 3*time.Second).WithContext(ctx).Should(BeTrue(), "pods matching label() still present")

	By("Sleeping 90 seconds")
	time.Sleep(90 * time.Second)

	By("Removing ConfigMap")

	deleteConfigMap(sriovDeploy2CMName, RDSCoreConfig.WlkdSRIOVOneNS)

	By("Creating ConfigMap")

	createConfigMap(sriovDeploy2CMName,
		RDSCoreConfig.WlkdSRIOVOneNS, RDSCoreConfig.WlkdSRIOVConfigMapDataOne)

	By("Removing ServiceAccount")
	deleteServiceAccount(sriovDeploy2SAName, RDSCoreConfig.WlkdSRIOVOneNS)

	By("Creating ServiceAccount")
	createServiceAccount(sriovDeploy2SAName, RDSCoreConfig.WlkdSRIOVOneNS)

	By("Remoing Cluster RBAC")
	deleteClusterRBAC(sriovDeployRBACName2)

	By("Creating Cluster RBAC")
	createClusterRBAC(sriovDeployRBACName2, sriovRBACRole2,
		sriovDeploy2SAName, RDSCoreConfig.WlkdSRIOVOneNS)

	By("Defining container configuration")

	deployContainer := defineContainer(sriovContainerOneName, RDSCoreConfig.WlkdSRIOVDeployOneImage,
		RDSCoreConfig.WlkdSRIOVDeploy2OneCmd, RDSCoreConfig.WldkSRIOVDeployOneResRequests,
		RDSCoreConfig.WldkSRIOVDeployOneResLimits)

	deployContainerTwo := defineContainer(sriovContainerTwoName, RDSCoreConfig.WlkdSRIOVDeployTwoImage,
		RDSCoreConfig.WlkdSRIOVDeploy2TwoCmd, RDSCoreConfig.WldkSRIOVDeployOneResRequests,
		RDSCoreConfig.WldkSRIOVDeployOneResLimits)

	By("Obtaining container definition")

	deployContainerCfg, err := deployContainer.GetContainerCfg()
	Expect(err).ToNot(HaveOccurred(), "Failed to get container definition")

	deployContainerTwoCfg, err := deployContainerTwo.GetContainerCfg()
	Expect(err).ToNot(HaveOccurred(), "Failed to get container definition")

	By("Defining 1st deployment configuration")

	deployOneLabels := map[string]string{
		strings.Split(sriovDeploy2OneLabel, "=")[0]: strings.Split(sriovDeploy2OneLabel, "=")[1]}

	deploy := defineDeployment(deployContainerCfg,
		sriovDeploy2OneName,
		RDSCoreConfig.WlkdSRIOVOneNS,
		RDSCoreConfig.WlkdSRIOVNetOne,
		sriovDeploy2CMName,
		sriovDeploy2SAName,
		deployOneLabels,
		RDSCoreConfig.WlkdSRIOVDeployOneSelector)

	By("Creating deployment one")

	deploy, err = deploy.CreateAndWaitUntilReady(5 * time.Minute)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to create deployment %s: %v", sriovDeploy2OneName, err))

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Deployment %q in %q ns is Ready",
		deploy.Definition.Name, deploy.Definition.Namespace)

	By("Defining 2nd deployment")

	deployTwoLabels := map[string]string{
		strings.Split(sriovDeploy2TwoLabel, "=")[0]: strings.Split(sriovDeploy2TwoLabel, "=")[1]}

	deployTwo := defineDeployment(deployContainerTwoCfg,
		sriovDeploy2TwoName,
		RDSCoreConfig.WlkdSRIOVOneNS,
		RDSCoreConfig.WlkdSRIOVNetOne,
		sriovDeploy2CMName,
		sriovDeploy2SAName,
		deployTwoLabels,
		RDSCoreConfig.WlkdSRIOVDeployTwoSelector)

	By("Creating 2nd deployment")

	deployTwo, err = deployTwo.CreateAndWaitUntilReady(5 * time.Minute)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to create deployment %s: %v", sriovDeploy2TwoName, err))

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Deployment %q in %q ns is Ready",
		deployTwo.Definition.Name, deployTwo.Definition.Namespace)

	addressesList := []string{RDSCoreConfig.WlkdSRIOVDeploy2OneTargetAddress,
		RDSCoreConfig.WlkdSRIOVDeploy2OneTargetAddressIPv6}

	for _, targetAddress := range addressesList {
		if targetAddress == "" {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Skipping empty address %q", targetAddress)

			continue
		}

		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Access workload via %q", targetAddress)

		verifySRIOVConnectivity(
			RDSCoreConfig.WlkdSRIOVOneNS,
			RDSCoreConfig.WlkdSRIOVOneNS,
			sriovDeploy2OneLabel,
			sriovDeploy2TwoLabel,
			targetAddress)
	}

	addressesList = []string{RDSCoreConfig.WlkdSRIOVDeploy2TwoTargetAddress,
		RDSCoreConfig.WlkdSRIOVDeploy2TwoTargetAddressIPv6}

	for _, targetAddress := range addressesList {
		if targetAddress == "" {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Skipping empty address %q", targetAddress)

			continue
		}

		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Access workload via %q", targetAddress)

		verifySRIOVConnectivity(
			RDSCoreConfig.WlkdSRIOVOneNS,
			RDSCoreConfig.WlkdSRIOVOneNS,
			sriovDeploy2TwoLabel,
			sriovDeploy2OneLabel,
			targetAddress)
	}
}

// VerifySRIOVConnectivityBetweenDifferentNodes test connectivity after cluster's reboot.
func VerifySRIOVConnectivityBetweenDifferentNodes(ctx SpecContext) {
	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Verify connectivity between SR-IOV workloads on different node")

	addressesList := []string{RDSCoreConfig.WlkdSRIOVDeploy2OneTargetAddress,
		RDSCoreConfig.WlkdSRIOVDeploy2OneTargetAddressIPv6}

	for _, targetAddress := range addressesList {
		if targetAddress == "" {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Skipping empty address %q", targetAddress)

			continue
		}

		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Access workload via %q", targetAddress)

		verifySRIOVConnectivity(
			RDSCoreConfig.WlkdSRIOVOneNS,
			RDSCoreConfig.WlkdSRIOVOneNS,
			sriovDeploy2OneLabel,
			sriovDeploy2TwoLabel,
			targetAddress)
	}

	addressesList = []string{RDSCoreConfig.WlkdSRIOVDeploy2TwoTargetAddress,
		RDSCoreConfig.WlkdSRIOVDeploy2TwoTargetAddressIPv6}

	for _, targetAddress := range addressesList {
		if targetAddress == "" {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Skipping empty address %q", targetAddress)

			continue
		}

		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Access workload via %q", targetAddress)

		verifySRIOVConnectivity(
			RDSCoreConfig.WlkdSRIOVOneNS,
			RDSCoreConfig.WlkdSRIOVOneNS,
			sriovDeploy2TwoLabel,
			sriovDeploy2OneLabel,
			targetAddress)
	}
}

// VerifySRIOVConnectivityOnSameNode tests connectivity after cluster's reboot.
func VerifySRIOVConnectivityOnSameNode(ctx SpecContext) {
	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Verify connectivity between SR-IOV workloads on the same node")

	addressesList := []string{RDSCoreConfig.WlkdSRIOVDeployOneTargetAddress,
		RDSCoreConfig.WlkdSRIOVDeployOneTargetAddressIPv6}

	for _, targetAddress := range addressesList {
		if targetAddress == "" {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Skipping empty address %q", targetAddress)

			continue
		}

		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Access workload via %q", targetAddress)

		verifySRIOVConnectivity(
			RDSCoreConfig.WlkdSRIOVOneNS,
			RDSCoreConfig.WlkdSRIOVOneNS,
			sriovDeployOneLabel,
			sriovDeployTwoLabel,
			targetAddress)
	}

	addressesList = []string{RDSCoreConfig.WlkdSRIOVDeployTwoTargetAddress,
		RDSCoreConfig.WlkdSRIOVDeployTwoTargetAddressIPv6}

	for _, targetAddress := range addressesList {
		if targetAddress == "" {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Skipping empty address %q", targetAddress)

			continue
		}

		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Access workload via %q", targetAddress)

		verifySRIOVConnectivity(
			RDSCoreConfig.WlkdSRIOVOneNS,
			RDSCoreConfig.WlkdSRIOVOneNS,
			sriovDeployTwoLabel,
			sriovDeployOneLabel,
			targetAddress)
	}
}

// VerifySRIOVSuite container that contains tests for SR-IOV verification.
func VerifySRIOVSuite() {
	Describe(
		"SR-IOV verification",
		Label(rdscoreparams.LabelValidateSRIOV), func() {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("*******************************************")
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("*** Starting SR-IOV RDS Core Test Suite ***")
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("*******************************************")

			It("Verifices SR-IOV workloads on the same node",
				Label("sriov-same-node"), reportxml.ID("71949"), MustPassRepeatedly(3),
				VerifySRIOVWorkloadsOnSameNode)

			It("Verifices SR-IOV workloads on different nodes",
				Label("sriov-different-node"), reportxml.ID("71950"), MustPassRepeatedly(3),
				VerifySRIOVWorkloadsOnDifferentNodes)
		})
}
