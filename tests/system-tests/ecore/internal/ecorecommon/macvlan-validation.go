package ecorecommon

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
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	. "github.com/openshift-kni/eco-gotests/tests/system-tests/ecore/internal/ecoreinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/ecore/internal/ecoreparams"
	multus "gopkg.in/k8snetworkplumbingwg/multus-cni.v4/pkg/types"
)

const (
	// Names of deployments.
	macvlanDeploy10Name = "core-mcvlan1-one"
	macvlanDeploy11Name = "core-mcvlan1-two"
	macvlanDeploy20Name = "core-mcvlan2-one"
	macvlanDeploy21Name = "core-mcvlan2-two"
	macvlanDeploy30Name = "core-mcvlan3-one"
	macvlanDeploy31Name = "core-mcvlan3-two"
	macvlanDeploy40Name = "core-mcvlan4-one"
	macvlanDeploy41Name = "core-mcvlan4-two"
	// ConfigMap names.
	macvlanDeploy1CMName = "core-mcvlan1-config"
	macvlanDeploy2CMName = "core-mcvlan2-config"
	macvlanDeploy3CMName = "core-mcvlan3-config"
	macvlanDeploy4CMName = "core-mcvlan4-config"
	// ServiceAccount names.
	macvlanDeploy1SAName = "rdscore-mcvlan-sa-one"
	macvlanDeploy2SAName = "rdscore-mcvlan-sa-two"
	macvlanDeploy3SAName = "rdscore-mcvlan-sa-three"
	macvlanDeploy4SAName = "rdscore-mcvlan-sa-four"
	// Container names within deployments.
	macvlanContainer1Name = "macvlan-one"
	macvlanContainer2Name = "macvlan-two"
	macvlanContainer3Name = "macvlan-three"
	macvlanContainer4Name = "macvlan-four"
	// Labels for deployments.
	macvlanDeploy10Label = "rds-core=mcvlan-deploy-one"
	macvlanDeploy11Label = "rds-core=mcvlan-deploy-two"
	macvlanDeploy20Label = "rds-core=mcvlan-deploy2-one"
	macvlanDeploy21Label = "rds-core=mcvlan-deploy2-two"
	macvlanDeploy30Label = "rds-core=mcvlan-deploy3-one"
	macvlanDeploy31Label = "rds-core=mcvlan-deploy3-two"
	macvlanDeploy40Label = "rds-core=mcvlan-deploy4-one"
	macvlanDeploy41Label = "rds-core=mcvlan-deploy4-two"
	// RBAC names for the deployments.
	macvlanDeployRBACName1 = "privileged-core-mcvlan1"
	macvlanDeployRBACName2 = "privileged-core-mcvlan2"
	macvlanDeployRBACName3 = "privileged-core-mcvlan3"
	macvlanDeployRBACName4 = "privileged-core-mcvlan4"
	// ClusterRole to use with RBAC.
	macvlanRBACRole1 = "system:openshift:scc:privileged"
	macvlanRBACRole2 = "system:openshift:scc:privileged"
	macvlanRBACRole3 = "system:openshift:scc:privileged"
	macvlanRBACRole4 = "system:openshift:scc:privileged"
)

func assertPodsAreGone(fNamespace, podLabel string) {
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

		if len(podMatchingSelector) != 0 {
			glog.V(ecoreparams.ECoreLogLevel).Infof("Found %d pods matching label %q in namespace %q",
				len(podMatchingSelector), podLabel, fNamespace)

			return false
		}

		return true
	}).WithContext(ctx).WithPolling(15*time.Second).WithTimeout(5*time.Minute).Should(BeTrue(),
		fmt.Sprintf("Pods matching label %q still exist in %q namespace", podLabel, fNamespace))
}

// VerifyMacVlanOnDifferentNodes verifies connectivity between freshly deployed workloads that use
// same MACVLAN definition and are scheduled on different nodes.
//
//nolint:funlen
func VerifyMacVlanOnDifferentNodes() {
	deleteConfigMap(macvlanDeploy1CMName, ECoreConfig.NamespacePCC)
	deleteConfigMap(macvlanDeploy2CMName, ECoreConfig.NamespacePCC)

	createConfigMap(macvlanDeploy1CMName, ECoreConfig.NamespacePCC, ECoreConfig.NADConfigMapPCCData)
	createConfigMap(macvlanDeploy2CMName, ECoreConfig.NamespacePCC, ECoreConfig.NADConfigMapPCCData)

	deleteClusterRBAC(macvlanDeployRBACName1)
	deleteClusterRBAC(macvlanDeployRBACName2)

	deleteDeployments(macvlanDeploy10Name, ECoreConfig.NamespacePCC)
	deleteDeployments(macvlanDeploy11Name, ECoreConfig.NamespacePCC)

	assertPodsAreGone(ECoreConfig.NamespacePCC, macvlanDeploy10Label)
	assertPodsAreGone(ECoreConfig.NamespacePCC, macvlanDeploy11Label)

	deleteServiceAccount(macvlanDeploy1SAName, ECoreConfig.NamespacePCC)
	deleteServiceAccount(macvlanDeploy2SAName, ECoreConfig.NamespacePCC)

	By("Defining container configuration")

	mvOne := defineContainer(macvlanContainer1Name,
		ECoreConfig.NADWlkdDeployOnePCCImage, ECoreConfig.NADWlkdDeployOnePCCCmd)

	mvTwo := defineContainer(macvlanContainer2Name,
		ECoreConfig.NADWlkdDeployOnePCCImage, ECoreConfig.NADWlkdDeployTwoPCCCmd)

	By("Obtaining container definition")

	deployContainerCfg, err := mvOne.GetContainerCfg()
	Expect(err).ToNot(HaveOccurred(), "Failed to get container definition")

	deployContainerTwoCfg, err := mvTwo.GetContainerCfg()
	Expect(err).ToNot(HaveOccurred(), "Failed to get container definition")

	By("Defining deployment configuration")

	deploy := defineMacVlanDeployment(macvlanDeploy10Name,
		ECoreConfig.NamespacePCC,
		macvlanDeploy10Label,
		ECoreConfig.NADWlkdOneNadName,
		macvlanDeploy1CMName,
		deployContainerCfg,
		ECoreConfig.NADWlkdDeployOnePCCSelector)

	deployTwo := defineMacVlanDeployment(macvlanDeploy11Name,
		ECoreConfig.NamespacePCC,
		macvlanDeploy11Label,
		ECoreConfig.NADWlkdOneNadName,
		macvlanDeploy2CMName,
		deployContainerTwoCfg,
		ECoreConfig.NADWlkdDeployTwoPCCSelector)

	By("Creating ServiceAccount for the deployment")

	createServiceAccount(macvlanDeploy1SAName, ECoreConfig.NamespacePCC)
	createServiceAccount(macvlanDeploy2SAName, ECoreConfig.NamespacePCC)

	By("Creating RBAC for SA")

	createClusterRBAC(macvlanDeployRBACName1, macvlanRBACRole1, macvlanDeploy1SAName, ECoreConfig.NamespacePCC)
	createClusterRBAC(macvlanDeployRBACName2, macvlanRBACRole2, macvlanDeploy2SAName, ECoreConfig.NamespacePCC)

	By("Assigning ServiceAccount to the deployment")

	deploy = deploy.WithServiceAccountName(macvlanDeploy1SAName)
	deployTwo = deployTwo.WithServiceAccountName(macvlanDeploy2SAName)

	if len(ECoreConfig.WlkdTolerationList) > 0 {
		By("Adding TaintToleration")

		for _, toleration := range ECoreConfig.WlkdTolerationList {
			glog.V(ecoreparams.ECoreLogLevel).Infof("Adding toleration: %v", toleration)

			deploy = deploy.WithToleration(toleration)
			deployTwo = deployTwo.WithToleration(toleration)
		}
	}

	By("Creating a deployment")

	deploy, err = deploy.CreateAndWaitUntilReady(5 * time.Minute)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to create deployment %s: %v", deploy.Definition.Name, err))

	deployTwo, err = deployTwo.CreateAndWaitUntilReady(5 * time.Minute)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to create deployment %s: %v", deployTwo.Definition.Name, err))

	VerifyMACVLANConnectivityBetweenDifferentNodes()
}

// VerifyMACVLANConnectivityBetweenDifferentNodes verifies connectivity between workloads,
// using MACVLAN interfaces and running on different nodes.
func VerifyMACVLANConnectivityBetweenDifferentNodes() {
	verifySRIOVConnectivity(
		ECoreConfig.NamespacePCC,
		ECoreConfig.NamespacePCC,
		macvlanDeploy10Label,
		macvlanDeploy11Label,
		ECoreConfig.WlkdMACVLANDeployOneTargetAddress)

	verifySRIOVConnectivity(
		ECoreConfig.NamespacePCC,
		ECoreConfig.NamespacePCC,
		macvlanDeploy11Label,
		macvlanDeploy10Label,
		ECoreConfig.WlkdMACVLANDeployTwoTargetAddress)

	verifySRIOVConnectivity(
		ECoreConfig.NamespacePCC,
		ECoreConfig.NamespacePCC,
		macvlanDeploy10Label,
		macvlanDeploy11Label,
		ECoreConfig.WlkdMACVLANDeployOneTargetAddressIPv6)

	verifySRIOVConnectivity(
		ECoreConfig.NamespacePCC,
		ECoreConfig.NamespacePCC,
		macvlanDeploy11Label,
		macvlanDeploy10Label,
		ECoreConfig.WlkdMACVLANDeployTwoTargetAddressIPv6)
}

// VerifyMacVlanOnSameNode verifies connectivity between freshly deployed workloads that use
// same MACVLAN definition and are scheduled to the same node.
//
//nolint:funlen
func VerifyMacVlanOnSameNode() {
	deleteConfigMap(macvlanDeploy3CMName, ECoreConfig.NamespacePCC)
	deleteConfigMap(macvlanDeploy4CMName, ECoreConfig.NamespacePCC)

	createConfigMap(macvlanDeploy3CMName, ECoreConfig.NamespacePCC, ECoreConfig.NADConfigMapPCCData)
	createConfigMap(macvlanDeploy4CMName, ECoreConfig.NamespacePCC, ECoreConfig.NADConfigMapPCCData)

	deleteClusterRBAC(macvlanDeployRBACName3)
	deleteClusterRBAC(macvlanDeployRBACName4)

	deleteDeployments(macvlanDeploy20Name, ECoreConfig.NamespacePCC)
	deleteDeployments(macvlanDeploy21Name, ECoreConfig.NamespacePCC)

	assertPodsAreGone(ECoreConfig.NamespacePCC, macvlanDeploy20Label)
	assertPodsAreGone(ECoreConfig.NamespacePCC, macvlanDeploy21Label)

	deleteServiceAccount(macvlanDeploy3SAName, ECoreConfig.NamespacePCC)
	deleteServiceAccount(macvlanDeploy4SAName, ECoreConfig.NamespacePCC)

	By("Defining container configuration")

	mvOne := defineContainer(macvlanContainer3Name,
		ECoreConfig.NADWlkdDeployOnePCCImage, ECoreConfig.NADWlkdDeploy3PCCCmd)

	mvTwo := defineContainer(macvlanContainer4Name,
		ECoreConfig.NADWlkdDeployOnePCCImage, ECoreConfig.NADWlkdDeploy4PCCCmd)

	By("Obtaining container definition")

	deployContainerCfg, err := mvOne.GetContainerCfg()
	Expect(err).ToNot(HaveOccurred(), "Failed to get container definition")

	deployContainerTwoCfg, err := mvTwo.GetContainerCfg()
	Expect(err).ToNot(HaveOccurred(), "Failed to get container definition")

	By("Defining deployment configuration")

	deploy := defineMacVlanDeployment(macvlanDeploy20Name,
		ECoreConfig.NamespacePCC,
		macvlanDeploy20Label,
		ECoreConfig.NADWlkdOneNadName,
		macvlanDeploy3CMName,
		deployContainerCfg,
		ECoreConfig.NADWlkdDeployOnePCCSelector)

	deployTwo := defineMacVlanDeployment(macvlanDeploy21Name,
		ECoreConfig.NamespacePCC,
		macvlanDeploy21Label,
		ECoreConfig.NADWlkdOneNadName,
		macvlanDeploy4CMName,
		deployContainerTwoCfg,
		ECoreConfig.NADWlkdDeployOnePCCSelector)

	By("Creating ServiceAccount for the deployment")

	createServiceAccount(macvlanDeploy3SAName, ECoreConfig.NamespacePCC)
	createServiceAccount(macvlanDeploy4SAName, ECoreConfig.NamespacePCC)

	By("Creating RBAC for SA")

	createClusterRBAC(macvlanDeployRBACName3, macvlanRBACRole3, macvlanDeploy3SAName, ECoreConfig.NamespacePCC)
	createClusterRBAC(macvlanDeployRBACName4, macvlanRBACRole4, macvlanDeploy4SAName, ECoreConfig.NamespacePCC)

	By("Assigning ServiceAccount to the deployment")

	deploy = deploy.WithServiceAccountName(macvlanDeploy3SAName)
	deployTwo = deployTwo.WithServiceAccountName(macvlanDeploy4SAName)

	if len(ECoreConfig.WlkdTolerationList) > 0 {
		By("Adding TaintToleration")

		for _, toleration := range ECoreConfig.WlkdTolerationList {
			glog.V(ecoreparams.ECoreLogLevel).Infof("Adding toleration: %v", toleration)

			deploy = deploy.WithToleration(toleration)
			deployTwo = deployTwo.WithToleration(toleration)
		}
	}

	By("Creating a deployment")

	deploy, err = deploy.CreateAndWaitUntilReady(5 * time.Minute)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to create deployment %s: %v", deploy.Definition.Name, err))

	deployTwo, err = deployTwo.CreateAndWaitUntilReady(5 * time.Minute)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to create deployment %s: %v", deployTwo.Definition.Name, err))

	VerifyMACVLANConnectivityOnSameNode()
}

// VerifyMACVLANConnectivityOnSameNode verifies connectivity between workloads that use MACVLAN net.
func VerifyMACVLANConnectivityOnSameNode() {
	verifySRIOVConnectivity(
		ECoreConfig.NamespacePCC,
		ECoreConfig.NamespacePCC,
		macvlanDeploy20Label,
		macvlanDeploy21Label,
		ECoreConfig.WlkdMACVLANDeploy3TargetAddress)

	verifySRIOVConnectivity(
		ECoreConfig.NamespacePCC,
		ECoreConfig.NamespacePCC,
		macvlanDeploy21Label,
		macvlanDeploy20Label,
		ECoreConfig.WlkdMACVLANDeploy4TargetAddress)

	verifySRIOVConnectivity(
		ECoreConfig.NamespacePCC,
		ECoreConfig.NamespacePCC,
		macvlanDeploy20Label,
		macvlanDeploy21Label,
		ECoreConfig.WlkdMACVLANDeploy3TargetAddressIPv6)

	verifySRIOVConnectivity(
		ECoreConfig.NamespacePCC,
		ECoreConfig.NamespacePCC,
		macvlanDeploy21Label,
		macvlanDeploy20Label,
		ECoreConfig.WlkdMACVLANDeploy4TargetAddressIPv6)
}

func defineMacVlanDeployment(dName, nsName, dLabels, netDefName, volName string,
	dContainer *v1.Container,
	nodeSelector map[string]string) *deployment.Builder {
	By("Defining deployment configuration")

	deploy := deployment.NewBuilder(APIClient,
		dName,
		nsName,
		map[string]string{strings.Split(dLabels, "=")[0]: strings.Split(dLabels, "=")[1]},
		dContainer)

	By("Adding MACVLAN annotations")

	var networks []*multus.NetworkSelectionElement

	networks = append(networks,
		&multus.NetworkSelectionElement{
			Name:      netDefName,
			Namespace: nsName})

	glog.V(ecoreparams.ECoreLogLevel).Infof("MACVlan networks:\n\t%#v", networks)

	deploy = deploy.WithSecondaryNetwork(networks)

	if len(nodeSelector) > 0 {
		By("Adding NodeSelector to the deployment")

		deploy = deploy.WithNodeSelector(nodeSelector)
	}

	By("Adding Volume to the deployment")

	volMode := new(int32)
	*volMode = 511

	volDefinition := v1.Volume{
		Name: "configs",
		VolumeSource: v1.VolumeSource{
			ConfigMap: &v1.ConfigMapVolumeSource{
				DefaultMode: volMode,
				LocalObjectReference: v1.LocalObjectReference{
					Name: volName,
				},
			},
		},
	}

	deploy = deploy.WithVolume(volDefinition)

	By("Setting Replicas count")

	deploy = deploy.WithReplicas(int32(1))

	return deploy
}
