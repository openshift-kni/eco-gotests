package ecorecommon

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
	"github.com/openshift-kni/eco-goinfra/pkg/serviceaccount"

	multus "gopkg.in/k8snetworkplumbingwg/multus-cni.v4/pkg/types"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
	. "github.com/openshift-kni/eco-gotests/tests/system-tests/ecore/internal/ecoreinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/ecore/internal/ecoreparams"
)

const (
	// Names of deployments.
	sriovDeploy1OneName = "rdscore-sriov-one"
	sriovDeploy1TwoName = "rdscore-sriov-two"
	sriovDeploy2OneName = "rdscore-sriov2-one"
	sriovDeploy2TwoName = "rdscore-sriov2-two"
	sriovDeploy3OneName = "rdscore-sriov3-one"
	sriovDeploy3TwoName = "rdscore-sriov3-two"
	sriovDeploy4OneName = "rdscore-sriov4-one"
	sriovDeploy4TwoName = "rdscore-sriov4-two"
	// ConfigMap names.
	sriovDeploy1CMName = "rdscore-sriov-config"
	sriovDeploy2CMName = "rdscore-sriov2-config"
	sriovDeploy3CMName = "rdscore-sriov3-config"
	sriovDeploy4CMName = "rdscore-sriov4-config"
	// ServiceAccount names.
	sriovDeploy1SAName = "rdscore-sriov-sa-one"
	sriovDeploy2SAName = "rdscore-sriov-sa-two"
	sriovDeploy3SAName = "rdscore-sriov-sa-three"
	sriovDeploy4SAName = "rdscore-sriov-sa-four"
	// Container names within deployments.
	sriovContainerOneName = "sriov-one"
	sriovContainerTwoName = "sriov-two"
	sriovContainer3Name   = "sriov-three"
	sriovContainer4Name   = "sriov-four"
	// Labels for deployments.
	sriovDeployOneLabel  = "rds-core=sriov-deploy-one"
	sriovDeployTwoLabel  = "rds-core=sriov-deploy-two"
	sriovDeploy2OneLabel = "rds-core=sriov-deploy2-one"
	sriovDeploy2TwoLabel = "rds-core=sriov-deploy2-two"
	sriovDeploy3OneLabel = "rds-core=sriov-deploy3-one"
	sriovDeploy3TwoLabel = "rds-core=sriov-deploy3-two"
	sriovDeploy4OneLabel = "rds-core=sriov-deploy4-one"
	sriovDeploy4TwoLabel = "rds-core=sriov-deploy4-two"
	// RBAC names for the deployments.
	sriovDeployRBACName  = "privileged-rdscore-sriov"
	sriovDeployRBACName2 = "privileged-rdscore-sriov2"
	sriovDeployRBACName3 = "privileged-rdscore-sriov3"
	sriovDeployRBACName4 = "privileged-rdscore-sriov4"
	// ClusterRole to use with RBAC.
	sriovRBACRole  = "system:openshift:scc:privileged"
	sriovRBACRole2 = "system:openshift:scc:privileged"
	sriovRBACRole3 = "system:openshift:scc:privileged"
	sriovRBACRole4 = "system:openshift:scc:privileged"
)

func createServiceAccount(saName, nsName string) {
	By(fmt.Sprintf("Creating ServiceAccount %q in %q namespace",
		saName, nsName))
	glog.V(ecoreparams.ECoreLogLevel).Infof("Creating SA %q in %q namespace",
		saName, nsName)

	deploySa := serviceaccount.NewBuilder(APIClient, saName, nsName)

	var ctx SpecContext

	Eventually(func() bool {
		deploySa, err := deploySa.Create()

		if err != nil {
			glog.V(ecoreparams.ECoreLogLevel).Infof("Error creating SA %q in %q namespace: %v",
				saName, nsName, err)

			return false
		}

		glog.V(ecoreparams.ECoreLogLevel).Infof("Created SA %q in %q namespace",
			deploySa.Definition.Name, deploySa.Definition.Namespace)

		return true
	}).WithContext(ctx).WithPolling(5*time.Second).WithTimeout(1*time.Minute).Should(BeTrue(),
		fmt.Sprintf("Failed to create ServiceAccount %q in %q namespace", saName, nsName))
}

func deleteServiceAccount(saName, nsName string) {
	By("Removing Service Account")
	glog.V(ecoreparams.ECoreLogLevel).Infof("Assert SA %q exists in %q namespace",
		saName, nsName)

	var ctx SpecContext

	if deploySa, err := serviceaccount.Pull(
		APIClient, saName, nsName); err == nil {
		glog.V(ecoreparams.ECoreLogLevel).Infof("ServiceAccount %q found in %q namespace",
			saName, nsName)
		glog.V(ecoreparams.ECoreLogLevel).Infof("Deleting ServiceAccount %q in %q namespace",
			saName, nsName)

		Eventually(func() bool {
			err := deploySa.Delete()

			if err != nil {
				glog.V(ecoreparams.ECoreLogLevel).Infof("Error deleting ServiceAccount %q in %q namespace: %v",
					saName, nsName, err)

				return false
			}

			glog.V(ecoreparams.ECoreLogLevel).Infof("Deleted ServiceAccount %q in %q namespace",
				saName, nsName)

			return true
		}).WithContext(ctx).WithPolling(5*time.Second).WithTimeout(1*time.Minute).Should(BeTrue(),
			fmt.Sprintf("Failed to delete ServiceAccount %q from %q ns", saName, nsName))
	} else {
		glog.V(ecoreparams.ECoreLogLevel).Infof("ServiceAccount %q not found in %q namespace",
			saName, nsName)
	}
}

func deleteClusterRBAC(rbacName string) {
	By("Deleting Cluster RBAC")

	var ctx SpecContext

	glog.V(ecoreparams.ECoreLogLevel).Infof("Assert ClusterRoleBinding %q exists", rbacName)

	if crbSa, err := rbac.PullClusterRoleBinding(
		APIClient,
		rbacName); err == nil {
		glog.V(ecoreparams.ECoreLogLevel).Infof("ClusterRoleBinding %q found. Deleting...", rbacName)

		Eventually(func() bool {
			err := crbSa.Delete()

			if err != nil {
				glog.V(ecoreparams.ECoreLogLevel).Infof("Error deleting ClusterRoleBinding %q : %v",
					rbacName, err)

				return false
			}

			glog.V(ecoreparams.ECoreLogLevel).Infof("Deleted ClusterRoleBinding %q", rbacName)

			return true
		}).WithContext(ctx).WithPolling(5*time.Second).WithTimeout(1*time.Minute).Should(BeTrue(),
			"Failed to delete Cluster RBAC")
	}
}

//nolint:unparam
func createClusterRBAC(rbacName, clusterRole, saName, nsName string) {
	By("Creating RBAC for SA")

	var ctx SpecContext

	glog.V(ecoreparams.ECoreLogLevel).Infof("Creating ClusterRoleBinding %q", rbacName)
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
			glog.V(ecoreparams.ECoreLogLevel).Infof(
				"Error Creating ClusterRoleBinding %q : %v", crbSa.Definition.Name, err)

			return false
		}

		glog.V(ecoreparams.ECoreLogLevel).Infof("ClusterRoleBinding %q created:\n\t%v",
			crbSa.Definition.Name, crbSa)

		return true
	}).WithContext(ctx).WithPolling(5*time.Second).WithTimeout(1*time.Minute).Should(BeTrue(),
		"Failed to create ClusterRoleBinding")
}

func deleteConfigMap(cmName, nsName string) {
	glog.V(ecoreparams.ECoreLogLevel).Infof("Assert ConfigMap %q exists in %q namespace",
		cmName, nsName)

	if cmBuilder, err := configmap.Pull(
		APIClient, cmName, nsName); err == nil {
		glog.V(ecoreparams.ECoreLogLevel).Infof("configMap %q found, deleting", cmName)

		var ctx SpecContext

		Eventually(func() bool {
			err := cmBuilder.Delete()
			if err != nil {
				glog.V(ecoreparams.ECoreLogLevel).Infof("Error deleting configMap %q : %v",
					cmName, err)

				return false
			}

			glog.V(ecoreparams.ECoreLogLevel).Infof("Deleted configMap %q in %q namespace",
				cmName, nsName)

			return true
		}).WithContext(ctx).WithPolling(5*time.Second).WithTimeout(1*time.Minute).Should(BeTrue(),
			"Failed to delete configMap")
	}
}

func createConfigMap(cmName, nsName string, data map[string]string) {
	glog.V(ecoreparams.ECoreLogLevel).Infof("Create ConfigMap %q in %q namespace",
		cmName, nsName)

	cmBuilder := configmap.NewBuilder(APIClient, cmName, nsName)
	cmBuilder.WithData(data)

	var ctx SpecContext

	Eventually(func() bool {

		cmResult, err := cmBuilder.Create()
		if err != nil {
			glog.V(ecoreparams.ECoreLogLevel).Infof("Error creating ConfigMap %q in %q namespace",
				cmName, nsName)

			return false
		}

		glog.V(ecoreparams.ECoreLogLevel).Infof("Created ConfigMap %q in %q namespace",
			cmResult.Definition.Name, nsName)

		return true
	}).WithContext(ctx).WithPolling(5*time.Second).WithPolling(1*time.Minute).Should(BeTrue(),
		"Failed to crete configMap")
}

func deleteDeployments(dName, nsName string) {
	By(fmt.Sprintf("Removing test deployment %q from %q ns", dName, nsName))

	if deploy, err := deployment.Pull(APIClient, dName, nsName); err == nil {
		glog.V(ecoreparams.ECoreLogLevel).Infof("Deleting deployment %q from %q namespace",
			deploy.Definition.Name, nsName)

		err = deploy.DeleteAndWait(300 * time.Second)
		Expect(err).ToNot(HaveOccurred(),
			fmt.Sprintf("failed to delete deployment %q", dName))
	} else {
		glog.V(ecoreparams.ECoreLogLevel).Infof("Deployment %q not found in %q namespace",
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

	return podMatchingSelector
}

func defineContainer(cName, cImage string, cCmd []string) *pod.ContainerBuilder {
	glog.V(ecoreparams.ECoreLogLevel).Infof("Creating container %q", cName)
	deployContainer := pod.NewContainerBuilder(cName, cImage, cCmd)

	By("Defining SecurityContext")

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

	By("Dropping ALL security capability")

	deployContainer = deployContainer.WithDropSecurityCapabilities([]string{"ALL"}, true)

	By("Adding VolumeMount to container")

	volMount := v1.VolumeMount{
		Name:      "configs",
		MountPath: "/opt/net/",
		ReadOnly:  false,
	}

	deployContainer = deployContainer.WithVolumeMount(volMount)

	glog.V(ecoreparams.ECoreLogLevel).Infof("%q container's  definition:\n%#v", cName, deployContainer)

	return deployContainer
}

func defineDeployment(containerConfig *v1.Container, deployName, deployNs, sriovNet, cmName, saName string,
	deployLabels, nodeSelector map[string]string) *deployment.Builder {
	glog.V(ecoreparams.ECoreLogLevel).Infof("Defining deployment %q in %q ns", deployName, deployNs)

	deploy := deployment.NewBuilder(APIClient, deployName, deployNs, deployLabels, containerConfig)

	By("Defining SR-IOV annotations")

	var networksOne []*multus.NetworkSelectionElement

	networksOne = append(networksOne,
		&multus.NetworkSelectionElement{
			Name: sriovNet})

	glog.V(ecoreparams.ECoreLogLevel).Infof("SR-IOV networks: %#v", networksOne)

	By("Adding SR-IOV annotations")

	deploy = deploy.WithSecondaryNetwork(networksOne)

	glog.V(ecoreparams.ECoreLogLevel).Infof("SR-IOV deploy one: %#v",
		deploy.Definition.Spec.Template.ObjectMeta.Annotations)

	By("Adding NodeSelector to the deployment")

	deploy = deploy.WithNodeSelector(nodeSelector)

	By("Adding Volume to the deployment")

	volMode := new(int32)
	*volMode = 511

	volDefinition := v1.Volume{
		Name: "configs",
		VolumeSource: v1.VolumeSource{
			ConfigMap: &v1.ConfigMapVolumeSource{
				DefaultMode: volMode,
				LocalObjectReference: v1.LocalObjectReference{
					Name: cmName,
				},
			},
		},
	}

	deploy = deploy.WithVolume(volDefinition)

	glog.V(ecoreparams.ECoreLogLevel).Infof("SR-IOV One Volume:\n %v",
		deploy.Definition.Spec.Template.Spec.Volumes)

	By(fmt.Sprintf("Assigning ServiceAccount %q to the deployment", saName))

	deploy = deploy.WithServiceAccountName(saName)

	By("Setting Replicas count")

	deploy = deploy.WithReplicas(int32(1))

	if len(ECoreConfig.WlkdTolerationList) > 0 {
		By("Adding TaintToleration")

		for _, toleration := range ECoreConfig.WlkdTolerationList {
			glog.V(ecoreparams.ECoreLogLevel).Infof("Adding toleration: %v", toleration)

			deploy = deploy.WithToleration(toleration)
		}
	}

	return deploy
}

func verifyMsgInPodLogs(podObj *pod.Builder, msg, cName string, timeSpan time.Time) {
	glog.V(ecoreparams.ECoreLogLevel).Infof("Parsing duration %q", timeSpan)

	var (
		podLog string
		err    error
		ctx    SpecContext
	)

	Eventually(func() bool {
		logStartTimestamp := time.Since(timeSpan)
		glog.V(ecoreparams.ECoreLogLevel).Infof("\tTime duration is %s", logStartTimestamp)

		if logStartTimestamp.Abs().Seconds() < 1 {
			logStartTimestamp, err = time.ParseDuration("1s")
			Expect(err).ToNot(HaveOccurred(), "Failed to parse time duration")
		}

		podLog, err = podObj.GetLog(logStartTimestamp, cName)

		if err != nil {
			glog.V(ecoreparams.ECoreLogLevel).Infof("Failed to get logs from pod %q: %v", podObj.Definition.Name, err)

			return false
		}

		glog.V(ecoreparams.ECoreLogLevel).Infof("Logs from pod %s:\n%s", podObj.Definition.Name, podLog)

		return true
	}).WithContext(ctx).WithPolling(5*time.Second).WithTimeout(1*time.Minute).Should(BeTrue(),
		fmt.Sprintf("Failed to get logs from pod %q", podObj.Definition.Name))

	Expect(podLog).Should(ContainSubstring(msg))
}

func verifySRIOVConnectivity(nsOneName, nsTwoName, deployOneLabels, deployTwoLabels, targetAddr string) {
	By("Getting pods backed by deployment")

	glog.V(ecoreparams.ECoreLogLevel).Infof("Looking for pod(s) matching label %q in %q namespace",
		deployOneLabels, nsOneName)

	podOneList := findPodWithSelector(nsOneName, deployOneLabels)
	Expect(len(podOneList)).To(Equal(1), "Expected only one pod")

	podOne := podOneList[0]
	glog.V(ecoreparams.ECoreLogLevel).Infof("Pod one is %q on node %q",
		podOne.Definition.Name, podOne.Definition.Spec.NodeName)

	glog.V(ecoreparams.ECoreLogLevel).Infof("Looking for pod(s) matching label %q in %q namespace",
		deployTwoLabels, nsTwoName)

	podTwoList := findPodWithSelector(nsTwoName, deployTwoLabels)
	Expect(len(podTwoList)).To(Equal(1), "Expected only one pod")

	podTwo := podTwoList[0]
	glog.V(ecoreparams.ECoreLogLevel).Infof("Pod two is %q on node %q",
		podTwo.Definition.Name, podTwo.Definition.Spec.NodeName)

	By("Sending data from pod one to pod two")

	msgOne := fmt.Sprintf("Running from pod %s(%s) at %d",
		podOne.Definition.Name,
		podOne.Definition.Spec.NodeName,
		time.Now().Unix())

	glog.V(ecoreparams.ECoreLogLevel).Infof("Sending msg %q from pod %s",
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
			glog.V(ecoreparams.ECoreLogLevel).Infof("Failed to run command within pod: %v", err)

			return false
		}

		glog.V(ecoreparams.ECoreLogLevel).Infof("Successfully run command within container %q",
			podOne.Definition.Spec.Containers[0].Name)
		glog.V(ecoreparams.ECoreLogLevel).Infof("Result: %v - %s", podOneResult, &podOneResult)

		return true
	}).WithContext(ctx).WithPolling(5*time.Second).WithTimeout(1*time.Minute).Should(BeTrue(),
		fmt.Sprintf("Failed to send data from pod %s", podOne.Definition.Name))

	verifyMsgInPodLogs(podTwo, msgOne, podTwo.Definition.Spec.Containers[0].Name, timeStart)
}

// VerifySRIOVWorkloadsOnSameNode deploy worklods with SRIOV interfaces on the same node
//
//nolint:funlen
func VerifySRIOVWorkloadsOnSameNode(ctx SpecContext) {
	By("Checking SR-IOV deployments don't exist")

	deleteDeployments(sriovDeploy1OneName, ECoreConfig.NamespacePCG)
	deleteDeployments(sriovDeploy1TwoName, ECoreConfig.NamespacePCG)

	By(fmt.Sprintf("Ensuring pods from %q deployment are gone", sriovDeploy1OneName))

	glog.V(ecoreparams.ECoreLogLevel).Infof("Ensuring pods from %q deployment in %q namespace are gone",
		sriovDeploy1OneName, ECoreConfig.NamespacePCG)

	Eventually(func() bool {
		oldPods, _ := pod.List(APIClient, ECoreConfig.NamespacePCG,
			metav1.ListOptions{LabelSelector: sriovDeployOneLabel})

		return len(oldPods) == 0

	}, 6*time.Minute, 3*time.Second).WithContext(ctx).Should(BeTrue(), "pods matching label() still present")

	By(fmt.Sprintf("Ensuring pods from %q deployment are gone", sriovDeploy1TwoName))

	glog.V(ecoreparams.ECoreLogLevel).Infof("Ensuring pods from %q deployment in %q namespace are gone",
		sriovDeploy1TwoName, ECoreConfig.NamespacePCG)

	Eventually(func() bool {
		oldPods, _ := pod.List(APIClient, ECoreConfig.NamespacePCG,
			metav1.ListOptions{LabelSelector: sriovDeployTwoLabel})

		return len(oldPods) == 0

	}, 6*time.Minute, 3*time.Second).WithContext(ctx).Should(BeTrue(), "pods matching label() still present")

	By("Removing ConfigMap")

	deleteConfigMap(sriovDeploy1CMName, ECoreConfig.NamespacePCG)

	By("Creating ConfigMap")

	createConfigMap(sriovDeploy1CMName,
		ECoreConfig.NamespacePCG, ECoreConfig.WlkdSRIOVConfigMapDataPCG)

	By("Removing ServiceAccount")
	deleteServiceAccount(sriovDeploy1SAName, ECoreConfig.NamespacePCG)

	By("Creating ServiceAccount")
	createServiceAccount(sriovDeploy1SAName, ECoreConfig.NamespacePCG)

	By("Remoing Cluster RBAC")
	deleteClusterRBAC(sriovDeployRBACName)

	By("Creating Cluster RBAC")
	createClusterRBAC(sriovDeployRBACName, sriovRBACRole,
		sriovDeploy1SAName, ECoreConfig.NamespacePCG)

	By("Defining container configuration")

	deployContainer := defineContainer(sriovContainerOneName, ECoreConfig.WlkdSRIOVDeployOneImage,
		ECoreConfig.WlkdSRIOVDeployOneCmd)

	deployContainerTwo := defineContainer(sriovContainerTwoName, ECoreConfig.WlkdSRIOVDeployTwoImage,
		ECoreConfig.WlkdSRIOVDeployTwoCmd)

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
		ECoreConfig.NamespacePCG,
		ECoreConfig.WlkdSRIOVNetOne,
		sriovDeploy1CMName,
		sriovDeploy1SAName,
		deployOneLabels,
		ECoreConfig.WlkdSRIOVDeployOneSelector)

	By("Creating deployment one")

	deploy, err = deploy.CreateAndWaitUntilReady(5 * time.Minute)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to create deployment %s: %v", deploy.Definition.Name, err))

	By("Defining 2nd deployment")

	deployTwoLabels := map[string]string{
		strings.Split(sriovDeployTwoLabel, "=")[0]: strings.Split(sriovDeployTwoLabel, "=")[1]}

	deployTwo := defineDeployment(deployContainerTwoCfg,
		sriovDeploy1TwoName,
		ECoreConfig.NamespacePCG,
		ECoreConfig.WlkdSRIOVNetOne,
		sriovDeploy1CMName,
		sriovDeploy1SAName,
		deployTwoLabels,
		ECoreConfig.WlkdSRIOVDeployOneSelector)

	By("Creating 2nd deployment")

	deployTwo, err = deployTwo.CreateAndWaitUntilReady(5 * time.Minute)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to create deployment %s: %v", deployTwo.Definition.Name, err))

	glog.V(ecoreparams.ECoreLogLevel).Infof("Verify connectivity between SR-IOV workloads on the same node")

	addressesList := []string{ECoreConfig.WlkdSRIOVDeployOneTargetAddress,
		ECoreConfig.WlkdSRIOVDeployOneTargetAddressIPv6}

	for _, targetAddr := range addressesList {
		if targetAddr == "" {
			continue
		}

		glog.V(ecoreparams.ECoreLogLevel).Infof("Access %q address", targetAddr)

		verifySRIOVConnectivity(
			ECoreConfig.NamespacePCG,
			ECoreConfig.NamespacePCG,
			sriovDeployOneLabel,
			sriovDeployTwoLabel,
			targetAddr)
	}

	addressesList = []string{ECoreConfig.WlkdSRIOVDeployTwoTargetAddress,
		ECoreConfig.WlkdSRIOVDeployTwoTargetAddressIPv6}

	for _, targetAddr := range addressesList {
		if targetAddr == "" {
			continue
		}

		glog.V(ecoreparams.ECoreLogLevel).Infof("Access %q address", targetAddr)

		verifySRIOVConnectivity(
			ECoreConfig.NamespacePCG,
			ECoreConfig.NamespacePCG,
			sriovDeployTwoLabel,
			sriovDeployOneLabel,
			targetAddr)
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

	deleteDeployments(sriovDeploy2OneName, ECoreConfig.NamespacePCG)
	deleteDeployments(sriovDeploy2TwoName, ECoreConfig.NamespacePCG)

	By(fmt.Sprintf("Ensuring pods from %q deployment are gone", sriovDeploy2OneName))

	glog.V(ecoreparams.ECoreLogLevel).Infof("Ensuring pods from %q deployment in %q namespace are gone",
		sriovDeploy2OneName, ECoreConfig.NamespacePCG)

	Eventually(func() bool {
		oldPods, _ := pod.List(APIClient, ECoreConfig.NamespacePCG,
			metav1.ListOptions{LabelSelector: sriovDeploy2OneLabel})

		return len(oldPods) == 0

	}, 6*time.Minute, 3*time.Second).WithContext(ctx).Should(BeTrue(), "pods matching label() still present")

	By(fmt.Sprintf("Ensuring pods from %q deployment are gone", sriovDeploy2TwoName))

	glog.V(ecoreparams.ECoreLogLevel).Infof("Ensuring pods from %q deployment in %q namespace are gone",
		sriovDeploy2TwoName, ECoreConfig.NamespacePCG)

	Eventually(func() bool {
		oldPods, _ := pod.List(APIClient, ECoreConfig.NamespacePCG,
			metav1.ListOptions{LabelSelector: sriovDeploy2TwoLabel})

		return len(oldPods) == 0

	}, 6*time.Minute, 3*time.Second).WithContext(ctx).Should(BeTrue(), "pods matching label() still present")

	By("Removing ConfigMap")

	deleteConfigMap(sriovDeploy2CMName, ECoreConfig.NamespacePCG)

	By("Creating ConfigMap")

	createConfigMap(sriovDeploy2CMName,
		ECoreConfig.NamespacePCG, ECoreConfig.WlkdSRIOVConfigMapDataPCG)

	By("Removing ServiceAccount")
	deleteServiceAccount(sriovDeploy2SAName, ECoreConfig.NamespacePCG)

	By("Creating ServiceAccount")
	createServiceAccount(sriovDeploy2SAName, ECoreConfig.NamespacePCG)

	By("Remoing Cluster RBAC")
	deleteClusterRBAC(sriovDeployRBACName2)

	By("Creating Cluster RBAC")
	createClusterRBAC(sriovDeployRBACName2, sriovRBACRole2,
		sriovDeploy2SAName, ECoreConfig.NamespacePCG)

	By("Defining container configuration")

	deployContainer := defineContainer(sriovContainerOneName, ECoreConfig.WlkdSRIOVDeployOneImage,
		ECoreConfig.WlkdSRIOVDeploy2OneCmd)

	deployContainerTwo := defineContainer(sriovContainerTwoName, ECoreConfig.WlkdSRIOVDeployTwoImage,
		ECoreConfig.WlkdSRIOVDeploy2TwoCmd)

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
		ECoreConfig.NamespacePCG,
		ECoreConfig.WlkdSRIOVNetOne,
		sriovDeploy2CMName,
		sriovDeploy2SAName,
		deployOneLabels,
		ECoreConfig.WlkdSRIOVDeployOneSelector)

	By("Creating deployment one")

	deploy, err = deploy.CreateAndWaitUntilReady(5 * time.Minute)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to create deployment %s: %v", sriovDeploy2OneName, err))

	glog.V(ecoreparams.ECoreLogLevel).Infof("Deployment %q in %q ns is Ready",
		deploy.Definition.Name, deploy.Definition.Namespace)

	By("Defining 2nd deployment")

	deployTwoLabels := map[string]string{
		strings.Split(sriovDeploy2TwoLabel, "=")[0]: strings.Split(sriovDeploy2TwoLabel, "=")[1]}

	deployTwo := defineDeployment(deployContainerTwoCfg,
		sriovDeploy2TwoName,
		ECoreConfig.NamespacePCG,
		ECoreConfig.WlkdSRIOVNetOne,
		sriovDeploy2CMName,
		sriovDeploy2SAName,
		deployTwoLabels,
		ECoreConfig.WlkdSRIOVDeployTwoSelector)

	By("Creating 2nd deployment")

	deployTwo, err = deployTwo.CreateAndWaitUntilReady(5 * time.Minute)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to create deployment %s: %v", sriovDeploy2TwoName, err))

	glog.V(ecoreparams.ECoreLogLevel).Infof("Deployment %q in %q ns is Ready",
		deployTwo.Definition.Name, deployTwo.Definition.Namespace)

	addressesList := []string{ECoreConfig.WlkdSRIOVDeploy2OneTargetAddress,
		ECoreConfig.WlkdSRIOVDeploy2OneTargetAddressIPv6}

	for _, targetAddr := range addressesList {
		if targetAddr == "" {
			continue
		}

		glog.V(ecoreparams.ECoreLogLevel).Infof("Access %q address", targetAddr)

		verifySRIOVConnectivity(
			ECoreConfig.NamespacePCG,
			ECoreConfig.NamespacePCG,
			sriovDeploy2OneLabel,
			sriovDeploy2TwoLabel,
			targetAddr)
	}

	addressesList = []string{ECoreConfig.WlkdSRIOVDeploy2TwoTargetAddress,
		ECoreConfig.WlkdSRIOVDeploy2TwoTargetAddressIPv6}

	for _, targetAddr := range addressesList {
		if targetAddr == "" {
			continue
		}

		glog.V(ecoreparams.ECoreLogLevel).Infof("Access %q address", targetAddr)

		verifySRIOVConnectivity(
			ECoreConfig.NamespacePCG,
			ECoreConfig.NamespacePCG,
			sriovDeploy2TwoLabel,
			sriovDeploy2OneLabel,
			targetAddr)
	}
}

// VerifySRIOVConnectivityBetweenDifferentNodes test connectivity after cluster's reboot.
func VerifySRIOVConnectivityBetweenDifferentNodes(ctx SpecContext) {
	glog.V(ecoreparams.ECoreLogLevel).Infof("Verify connectivity between SR-IOV workloads on different node")

	verifySRIOVConnectivity(
		ECoreConfig.NamespacePCG,
		ECoreConfig.NamespacePCG,
		sriovDeploy2OneLabel,
		sriovDeploy2TwoLabel,
		ECoreConfig.WlkdSRIOVDeploy2OneTargetAddress)

	verifySRIOVConnectivity(
		ECoreConfig.NamespacePCG,
		ECoreConfig.NamespacePCG,
		sriovDeploy2TwoLabel,
		sriovDeploy2OneLabel,
		ECoreConfig.WlkdSRIOVDeploy2TwoTargetAddress)
}

// VerifySRIOVConnectivityOnSameNode tests connectivity after cluster's reboot.
func VerifySRIOVConnectivityOnSameNode(ctx SpecContext) {
	glog.V(ecoreparams.ECoreLogLevel).Infof("Verify connectivity between SR-IOV workloads on the same node")

	verifySRIOVConnectivity(
		ECoreConfig.NamespacePCG,
		ECoreConfig.NamespacePCG,
		sriovDeployOneLabel,
		sriovDeployTwoLabel,
		ECoreConfig.WlkdSRIOVDeployOneTargetAddress)

	verifySRIOVConnectivity(
		ECoreConfig.NamespacePCG,
		ECoreConfig.NamespacePCG,
		sriovDeployTwoLabel,
		sriovDeployOneLabel,
		ECoreConfig.WlkdSRIOVDeployTwoTargetAddress)
}

// VerifySRIOVSuite container that contains tests for SR-IOV verification.
func VerifySRIOVSuite() {
	Describe(
		"SR-IOV verification",
		Label("ecore-validate-sriov-suite"), func() {
			glog.V(ecoreparams.ECoreLogLevel).Infof("*******************************************")
			glog.V(ecoreparams.ECoreLogLevel).Infof("*** Starting SR-IOV RDS Core Test Suite ***")
			glog.V(ecoreparams.ECoreLogLevel).Infof("*******************************************")

			It("Verifices SR-IOV workloads on the same node",
				Label("sriov-same-node"), polarion.ID("71949"), MustPassRepeatedly(1),
				VerifySRIOVWorkloadsOnSameNode)

			It("Verifices SR-IOV workloads on different nodes",
				Label("sriov-different-node"), polarion.ID("71950"), MustPassRepeatedly(1),
				VerifySRIOVWorkloadsOnDifferentNodes)

			It("Verifies SR-IOV workloads on the same net and same node",
				Label("sriov-same-net-same-node"), MustPassRepeatedly(1),
				VerifySRIOVWorkloadsOnSameNodeDifferentNetworks)

			It("Verifies SR-IOV workloads on the same net and same node",
				Label("sriov-same-net-different-node"), MustPassRepeatedly(1),
				VerifySRIOVWorkloadsDifferentNodesDifferentNetworks)
		})
}

// VerifySRIOVWorkloadsOnSameNodeDifferentNetworks deploy worklods with SRIOV interfaces on the same node
//
//nolint:funlen
func VerifySRIOVWorkloadsOnSameNodeDifferentNetworks(ctx SpecContext) {
	By("Checking SR-IOV deployments don't exist")

	deleteDeployments(sriovDeploy3OneName, ECoreConfig.NamespacePCG)
	deleteDeployments(sriovDeploy3TwoName, ECoreConfig.NamespacePCG)

	By(fmt.Sprintf("Ensuring pods from %q deployment are gone", sriovDeploy3OneName))

	glog.V(ecoreparams.ECoreLogLevel).Infof("Ensuring pods from %q deployment in %q namespace are gone",
		sriovDeploy3OneName, ECoreConfig.NamespacePCG)

	Eventually(func() bool {
		oldPods, _ := pod.List(APIClient, ECoreConfig.NamespacePCG,
			metav1.ListOptions{LabelSelector: sriovDeploy3OneLabel})

		return len(oldPods) == 0

	}, 6*time.Minute, 3*time.Second).WithContext(ctx).Should(BeTrue(), "pods matching label() still present")

	By(fmt.Sprintf("Ensuring pods from %q deployment are gone", sriovDeploy3TwoName))

	glog.V(ecoreparams.ECoreLogLevel).Infof("Ensuring pods from %q deployment in %q namespace are gone",
		sriovDeploy3TwoName, ECoreConfig.NamespacePCG)

	Eventually(func() bool {
		oldPods, _ := pod.List(APIClient, ECoreConfig.NamespacePCG,
			metav1.ListOptions{LabelSelector: sriovDeploy3TwoLabel})

		return len(oldPods) == 0

	}, 6*time.Minute, 3*time.Second).WithContext(ctx).Should(BeTrue(), "pods matching label() still present")

	By("Removing ConfigMap")

	deleteConfigMap(sriovDeploy3CMName, ECoreConfig.NamespacePCG)

	By("Creating ConfigMap")

	createConfigMap(sriovDeploy3CMName,
		ECoreConfig.NamespacePCG, ECoreConfig.WlkdSRIOVConfigMapDataPCG)

	By("Removing ServiceAccount")
	deleteServiceAccount(sriovDeploy3SAName, ECoreConfig.NamespacePCG)

	By("Creating ServiceAccount")
	createServiceAccount(sriovDeploy3SAName, ECoreConfig.NamespacePCG)

	By("Remoing Cluster RBAC")
	deleteClusterRBAC(sriovDeployRBACName3)

	By("Creating Cluster RBAC")
	createClusterRBAC(sriovDeployRBACName3, sriovRBACRole3,
		sriovDeploy3SAName, ECoreConfig.NamespacePCG)

	By("Defining container configuration")

	deployContainer := defineContainer(sriovContainerOneName, ECoreConfig.WlkdSRIOVDeployOneImage,
		ECoreConfig.WlkdSRIOVDeploy3OneCmd)

	deployContainerTwo := defineContainer(sriovContainerTwoName, ECoreConfig.WlkdSRIOVDeployTwoImage,
		ECoreConfig.WlkdSRIOVDeploy3TwoCmd)

	By("Obtaining container definition")

	deployContainerCfg, err := deployContainer.GetContainerCfg()
	Expect(err).ToNot(HaveOccurred(), "Failed to get container definition")

	deployContainerTwoCfg, err := deployContainerTwo.GetContainerCfg()
	Expect(err).ToNot(HaveOccurred(), "Failed to get container definition")

	By("Defining 1st deployment configuration")

	deployOneLabels := map[string]string{
		strings.Split(sriovDeploy3OneLabel, "=")[0]: strings.Split(sriovDeploy3OneLabel, "=")[1]}

	deploy := defineDeployment(deployContainerCfg,
		sriovDeploy3OneName,
		ECoreConfig.NamespacePCG,
		ECoreConfig.WlkdSRIOVNetOne,
		sriovDeploy3CMName,
		sriovDeploy3SAName,
		deployOneLabels,
		ECoreConfig.WlkdSRIOVDeployOneSelector)

	By("Creating deployment one")

	deploy, err = deploy.CreateAndWaitUntilReady(5 * time.Minute)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to create deployment %s: %v", deploy.Definition.Name, err))

	By("Defining 2nd deployment")

	deployTwoLabels := map[string]string{
		strings.Split(sriovDeploy3TwoLabel, "=")[0]: strings.Split(sriovDeploy3TwoLabel, "=")[1]}

	deployTwo := defineDeployment(deployContainerTwoCfg,
		sriovDeploy3TwoName,
		ECoreConfig.NamespacePCG,
		ECoreConfig.WlkdSRIOVNetTwo,
		sriovDeploy3CMName,
		sriovDeploy3SAName,
		deployTwoLabels,
		ECoreConfig.WlkdSRIOVDeployOneSelector)

	By("Creating 2nd deployment")

	deployTwo, err = deployTwo.CreateAndWaitUntilReady(5 * time.Minute)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to create deployment %s: %v", deployTwo.Definition.Name, err))

	glog.V(ecoreparams.ECoreLogLevel).Infof("Verify connectivity between SR-IOV workloads on the same node")

	addressesList := []string{ECoreConfig.WlkdSRIOVDeploy3OneTargetAddress,
		ECoreConfig.WlkdSRIOVDeploy3OneTargetAddressIPv6}

	for _, targetAddr := range addressesList {
		if targetAddr == "" {
			continue
		}

		glog.V(ecoreparams.ECoreLogLevel).Infof("Access %q address", targetAddr)

		verifySRIOVConnectivity(
			ECoreConfig.NamespacePCG,
			ECoreConfig.NamespacePCG,
			sriovDeploy3OneLabel,
			sriovDeploy3TwoLabel,
			targetAddr)
	}

	addressesList = []string{ECoreConfig.WlkdSRIOVDeploy3TwoTargetAddress,
		ECoreConfig.WlkdSRIOVDeploy3TwoTargetAddressIPv6}

	for _, targetAddr := range addressesList {
		if targetAddr == "" {
			continue
		}

		glog.V(ecoreparams.ECoreLogLevel).Infof("Access %q address", targetAddr)

		verifySRIOVConnectivity(
			ECoreConfig.NamespacePCG,
			ECoreConfig.NamespacePCG,
			sriovDeploy3TwoLabel,
			sriovDeploy3OneLabel,
			targetAddr)
	}
}

// VerifySRIOVWorkloadsDifferentNodesDifferentNetworks deploy worklods with SRIOV interfaces on different nodes
//
//nolint:funlen
func VerifySRIOVWorkloadsDifferentNodesDifferentNetworks(ctx SpecContext) {
	By("Checking SR-IOV deployments don't exist")

	deleteDeployments(sriovDeploy4OneName, ECoreConfig.NamespacePCG)
	deleteDeployments(sriovDeploy4TwoName, ECoreConfig.NamespacePCG)

	By(fmt.Sprintf("Ensuring pods from %q deployment are gone", sriovDeploy4OneName))

	glog.V(ecoreparams.ECoreLogLevel).Infof("Ensuring pods from %q deployment in %q namespace are gone",
		sriovDeploy4OneName, ECoreConfig.NamespacePCG)

	Eventually(func() bool {
		oldPods, _ := pod.List(APIClient, ECoreConfig.NamespacePCG,
			metav1.ListOptions{LabelSelector: sriovDeploy4OneLabel})

		return len(oldPods) == 0

	}, 6*time.Minute, 3*time.Second).WithContext(ctx).Should(BeTrue(), "pods matching label() still present")

	By(fmt.Sprintf("Ensuring pods from %q deployment are gone", sriovDeploy4TwoName))

	glog.V(ecoreparams.ECoreLogLevel).Infof("Ensuring pods from %q deployment in %q namespace are gone",
		sriovDeploy4TwoName, ECoreConfig.NamespacePCG)

	Eventually(func() bool {
		oldPods, _ := pod.List(APIClient, ECoreConfig.NamespacePCG,
			metav1.ListOptions{LabelSelector: sriovDeploy4TwoLabel})

		return len(oldPods) == 0

	}, 6*time.Minute, 3*time.Second).WithContext(ctx).Should(BeTrue(), "pods matching label() still present")

	By("Removing ConfigMap")

	deleteConfigMap(sriovDeploy4CMName, ECoreConfig.NamespacePCG)

	By("Creating ConfigMap")

	createConfigMap(sriovDeploy4CMName,
		ECoreConfig.NamespacePCG, ECoreConfig.WlkdSRIOVConfigMapDataPCG)

	By("Removing ServiceAccount")
	deleteServiceAccount(sriovDeploy4SAName, ECoreConfig.NamespacePCG)

	By("Creating ServiceAccount")
	createServiceAccount(sriovDeploy4SAName, ECoreConfig.NamespacePCG)

	By("Remoing Cluster RBAC")
	deleteClusterRBAC(sriovDeployRBACName4)

	By("Creating Cluster RBAC")
	createClusterRBAC(sriovDeployRBACName4, sriovRBACRole4,
		sriovDeploy4SAName, ECoreConfig.NamespacePCG)

	By("Defining container configuration")

	deployContainer := defineContainer(sriovContainerOneName, ECoreConfig.WlkdSRIOVDeployOneImage,
		ECoreConfig.WlkdSRIOVDeploy4OneCmd)

	deployContainerTwo := defineContainer(sriovContainerTwoName, ECoreConfig.WlkdSRIOVDeployTwoImage,
		ECoreConfig.WlkdSRIOVDeploy4TwoCmd)

	By("Obtaining container definition")

	deployContainerCfg, err := deployContainer.GetContainerCfg()
	Expect(err).ToNot(HaveOccurred(), "Failed to get container definition")

	deployContainerTwoCfg, err := deployContainerTwo.GetContainerCfg()
	Expect(err).ToNot(HaveOccurred(), "Failed to get container definition")

	By("Defining 1st deployment configuration")

	deployOneLabels := map[string]string{
		strings.Split(sriovDeploy4OneLabel, "=")[0]: strings.Split(sriovDeploy4OneLabel, "=")[1]}

	deploy := defineDeployment(deployContainerCfg,
		sriovDeploy4OneName,
		ECoreConfig.NamespacePCG,
		ECoreConfig.WlkdSRIOVNetOne,
		sriovDeploy4CMName,
		sriovDeploy4SAName,
		deployOneLabels,
		ECoreConfig.WlkdSRIOVDeployOneSelector)

	By("Creating deployment one")

	deploy, err = deploy.CreateAndWaitUntilReady(5 * time.Minute)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to create deployment %s: %v", deploy.Definition.Name, err))

	By("Defining 2nd deployment")

	deployTwoLabels := map[string]string{
		strings.Split(sriovDeploy4TwoLabel, "=")[0]: strings.Split(sriovDeploy4TwoLabel, "=")[1]}

	deployTwo := defineDeployment(deployContainerTwoCfg,
		sriovDeploy4TwoName,
		ECoreConfig.NamespacePCG,
		ECoreConfig.WlkdSRIOVNetTwo,
		sriovDeploy4CMName,
		sriovDeploy4SAName,
		deployTwoLabels,
		ECoreConfig.WlkdSRIOVDeployTwoSelector)

	By("Creating 2nd deployment")

	deployTwo, err = deployTwo.CreateAndWaitUntilReady(5 * time.Minute)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to create deployment %s: %v", deployTwo.Definition.Name, err))

	glog.V(ecoreparams.ECoreLogLevel).Infof("Verify connectivity between SR-IOV workloads on different nodes")

	addressesList := []string{ECoreConfig.WlkdSRIOVDeploy4OneTargetAddress,
		ECoreConfig.WlkdSRIOVDeploy4OneTargetAddressIPv6}

	for _, targetAddr := range addressesList {
		if targetAddr == "" {
			continue
		}

		glog.V(ecoreparams.ECoreLogLevel).Infof("Access %q address", targetAddr)

		verifySRIOVConnectivity(
			ECoreConfig.NamespacePCG,
			ECoreConfig.NamespacePCG,
			sriovDeploy4OneLabel,
			sriovDeploy4TwoLabel,
			targetAddr)
	}

	addressesList = []string{ECoreConfig.WlkdSRIOVDeploy4TwoTargetAddress,
		ECoreConfig.WlkdSRIOVDeploy4TwoTargetAddressIPv6}

	for _, targetAddr := range addressesList {
		if targetAddr == "" {
			continue
		}

		glog.V(ecoreparams.ECoreLogLevel).Infof("Access %q address", targetAddr)

		verifySRIOVConnectivity(
			ECoreConfig.NamespacePCG,
			ECoreConfig.NamespacePCG,
			sriovDeploy4TwoLabel,
			sriovDeploy4OneLabel,
			targetAddr)
	}
}
