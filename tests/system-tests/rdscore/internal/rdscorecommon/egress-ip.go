package rdscorecommon

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"net/netip"
	"strings"
	"time"

	"github.com/openshift-kni/eco-goinfra/pkg/namespace"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	scc "github.com/openshift-kni/eco-gotests/tests/system-tests/internal/scc"

	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/apiobjectshelper"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/deployment"
	"github.com/openshift-kni/eco-goinfra/pkg/egressip"
	. "github.com/openshift-kni/eco-gotests/tests/system-tests/rdscore/internal/rdscoreinittools"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	proberRBACRole       = "system:openshift:scc:privileged"
	nonEgressIPPodLabel  = "env=centos"
	nonEgressIPNamespace = "non-egressip-ns"
)

var (
	egressIPPodLabelsMap = map[string]string{
		strings.Split(RDSCoreConfig.EgressIPPodLabel, "=")[0]: strings.Split(RDSCoreConfig.EgressIPPodLabel, "=")[1]}

	egressIPPodSelector = metav1.ListOptions{
		LabelSelector: RDSCoreConfig.EgressIPPodLabel,
	}
	nonEgressIPPodSelector = metav1.ListOptions{
		LabelSelector: nonEgressIPPodLabel,
	}
)

// createAgnhostRBAC creates the SCC privileged, serviceAccount and RBAC.
func createAgnhostRBAC(
	apiClient *clients.Settings,
	egressIPNamespace string) (string, error) {
	proberDeploySAName := fmt.Sprintf("rdscore-prober-sa-%s", egressIPNamespace)
	proberDeployRBACName := fmt.Sprintf("privileged-rdscore-rbac-%s", egressIPNamespace)

	var err error

	glog.V(100).Infof("Adding SCC privileged to the agnhost namespace")

	err = scc.AddPrivilegedSCCtoDefaultSA(egressIPNamespace)
	if err != nil {
		return "", fmt.Errorf("failed to add SCC privileged to the agnhost namespace %s: %w",
			egressIPNamespace, err)
	}

	glog.V(100).Infof("Removing ServiceAccount")

	err = apiobjectshelper.DeleteServiceAccount(apiClient, proberDeploySAName, egressIPNamespace)

	if err != nil {
		return "", fmt.Errorf("failed to remove serviceAccount %q from egressIPNamespace %q",
			proberDeploySAName, egressIPNamespace)
	}

	glog.V(100).Infof("Creating ServiceAccount")

	err = apiobjectshelper.CreateServiceAccount(apiClient, proberDeploySAName, egressIPNamespace)

	if err != nil {
		return "", fmt.Errorf("failed to create serviceAccount %q in egressIPNamespace %q",
			proberDeploySAName, egressIPNamespace)
	}

	glog.V(100).Infof("Removing Cluster RBAC")

	err = apiobjectshelper.DeleteClusterRBAC(apiClient, proberDeployRBACName)

	if err != nil {
		return "", fmt.Errorf("failed to delete prober RBAC %q", proberDeployRBACName)
	}

	glog.V(100).Infof("Creating Cluster RBAC")

	err = apiobjectshelper.CreateClusterRBAC(apiClient, proberDeployRBACName, proberRBACRole,
		proberDeploySAName, egressIPNamespace)

	if err != nil {
		return "", fmt.Errorf("failed to create prober RBAC %q in egressIPNamespace %s",
			proberDeployRBACName, egressIPNamespace)
	}

	return proberDeploySAName, nil
}

// cleanUpDeployments removes all qe deployments leftovers from the cluster according to the label.
func cleanUpDeployments(apiClient *clients.Settings) error {
	namespacesList := []string{RDSCoreConfig.EgressIPNamespaceOne, RDSCoreConfig.EgressIPNamespaceTwo,
		nonEgressIPNamespace}
	podLabelsList := []string{RDSCoreConfig.EgressIPPodLabel, nonEgressIPPodLabel}

	for _, nsName := range namespacesList {
		deploymentsList, err := deployment.List(APIClient, nsName)

		if err != nil {
			glog.V(100).Infof("Error listing deployments in the namespace %s, %v", nsName, err)

			return err
		}

		for _, deploy := range deploymentsList {
			glog.V(100).Infof("Ensure %s deploy don't exist in namespace %s",
				deploy.Definition.Name, deploy.Definition.Namespace)

			err := apiobjectshelper.DeleteDeployment(
				apiClient,
				deploy.Definition.Name,
				nsName)

			if err != nil {
				return fmt.Errorf("failed to delete deploy %s from nsname %s",
					deploy.Definition.Name, nsName)
			}
		}

		for _, label := range podLabelsList {
			err = apiobjectshelper.EnsureAllPodsRemoved(APIClient, nsName, label)

			if err != nil {
				return fmt.Errorf("failed to delete pods in namespace %s: %w", nsName, err)
			}
		}
	}

	return nil
}

// createAgnhostDeployment creates the agnhost deployment that will be used as a source for EgressIP tests.
func createAgnhostDeployment(
	apiClient *clients.Settings,
	nsName,
	proberDeploySAName,
	scheduleOnHost string,
	podLabelsMap map[string]string) error {
	replicaCnt := 1
	nameSuffix := strings.Split(scheduleOnHost, ".")[0]
	randSuffix := randomString(3)
	deploymentName := fmt.Sprintf("%s-agnhost-%s-%s", nsName, nameSuffix, randSuffix)

	var err error

	glog.V(100).Infof("Defining container configuration")

	containerCmd := []string{
		"/agnhost",
		"netexec",
		"--http-port",
		RDSCoreConfig.EgressIPTcpPort,
	}

	deployContainer := defineProberContainer(RDSCoreConfig.EgressIPDeploymentImage, containerCmd)

	glog.V(100).Infof("Obtaining container definition")

	deployContainerCfg, err := deployContainer.GetContainerCfg()
	if err != nil {
		return fmt.Errorf("failed to obtain container definition: %w", err)
	}

	glog.V(100).Infof("Defining deployment configuration")

	proberDeployment := defineTestPodDeployment(
		apiClient,
		deployContainerCfg,
		deploymentName,
		nsName,
		proberDeploySAName,
		scheduleOnHost,
		replicaCnt,
		podLabelsMap)

	glog.V(100).Infof("Creating deployment")

	proberDeployment, err = proberDeployment.CreateAndWaitUntilReady(5 * time.Minute)
	if err != nil {
		glog.V(100).Infof("Failed to create deployment %s in namespace %s: %v",
			deploymentName, nsName, err)

		return fmt.Errorf("failed to create deployment %s in namespace %s: %w",
			deploymentName, nsName, err)
	}

	if proberDeployment == nil {
		return fmt.Errorf("failed to create deployment %s in namespace %s",
			deploymentName, nsName)
	}

	return nil
}

func randomString(length int) string {
	chars := "abcdefghijklmnopqrstuvwxyz0123456789"

	result := make([]byte, length)
	for i := range result {
		result[i] = chars[rand.Intn(len(chars))]
	}

	return string(result)
}

func defineTestPodDeployment(
	apiClient *clients.Settings,
	containerConfig *corev1.Container,
	deployName, deployNs, saName string,
	scheduleOnHost string,
	replicaCnt int,
	deployLabels map[string]string) *deployment.Builder {
	glog.V(100).Infof("Defining deployment %q in %q ns", deployName, deployNs)

	nodeAffinity := corev1.Affinity{
		NodeAffinity: &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: []corev1.NodeSelectorTerm{
					{
						MatchExpressions: []corev1.NodeSelectorRequirement{
							{
								Key:      "kubernetes.io/hostname",
								Operator: corev1.NodeSelectorOpIn,
								Values:   []string{scheduleOnHost},
							},
						},
					},
				},
			},
		},
	}

	podsDeployment := deployment.NewBuilder(apiClient, deployName, deployNs, deployLabels, *containerConfig)

	glog.V(100).Infof("Assigning ServiceAccount %q to the deployment", saName)

	podsDeployment = podsDeployment.WithServiceAccountName(saName)

	glog.V(100).Infof("Setting Replicas count")

	podsDeployment = podsDeployment.WithReplicas(int32(replicaCnt))

	podsDeployment = podsDeployment.WithHostNetwork(false)

	podsDeployment = podsDeployment.WithAffinity(&nodeAffinity)

	return podsDeployment
}

func defineProberContainer(cImage string, cCmd []string) *pod.ContainerBuilder {
	cName := "agnhost"

	glog.V(100).Infof("Creating container %q", cName)

	deployContainer := pod.NewContainerBuilder(cName, cImage, cCmd)

	glog.V(100).Infof("Defining SecurityContext")

	var trueFlag = true

	userUID := new(int64)

	*userUID = 0

	securityContext := &corev1.SecurityContext{
		RunAsUser:  userUID,
		Privileged: &trueFlag,
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
		Capabilities: &corev1.Capabilities{
			Add: []corev1.Capability{"NET_RAW", "NET_ADMIN", "SYS_ADMIN", "IPC_LOCK"},
		},
	}

	glog.V(100).Infof("Setting SecurityContext")

	deployContainer = deployContainer.WithSecurityContext(securityContext)

	glog.V(100).Infof("Dropping ALL security capability")

	deployContainer = deployContainer.WithDropSecurityCapabilities([]string{"ALL"}, true)

	glog.V(100).Infof("%q container's  definition:\n%#v", cName, deployContainer)

	return deployContainer
}

func sendTrafficCheckIP(clientPods []*pod.Builder, isIPv6 bool, expectedIPs []string) error {
	By("Validating pods source address")

	targetIP := RDSCoreConfig.EgressIPRemoteIPv4

	if isIPv6 {
		targetIP = fmt.Sprintf("[%s]", RDSCoreConfig.EgressIPRemoteIPv6)
	}

	cmdToRun := []string{"/bin/bash", "-c",
		fmt.Sprintf("curl --connect-timeout 5 -Ls http://%s:%s/clientip",
			targetIP, RDSCoreConfig.EgressIPTcpPort)}

	for _, clientPod := range clientPods {
		var parsedIP string

		err := wait.PollUntilContextTimeout(
			context.TODO(),
			time.Second,
			time.Second*5,
			true,
			func(ctx context.Context) (bool, error) {
				result, err := clientPod.ExecCommand(cmdToRun, clientPod.Object.Spec.Containers[0].Name)

				if err != nil {
					glog.V(100).Infof("Error running command from within a pod %q: %v",
						clientPod.Object.Name, err)

					return false, nil
				}

				glog.V(100).Infof("Successfully executed command from within a pod %q: %v",
					clientPod.Object.Name, err)
				glog.V(100).Infof("Command's output:\n\t%v", result.String())

				parsedIP, _, err = net.SplitHostPort(result.String())

				if err != nil {
					glog.V(100).Infof("Failed to parse %q for host/port pair", result.String())

					return false, nil
				}

				glog.V(100).Infof("Verify IP version type correctness")

				myIP, err := netip.ParseAddr(parsedIP)

				if err != nil {
					glog.V(100).Infof("Failed to parse used ip address %q", parsedIP)

					return false, nil
				}

				if isIPv6 && myIP.Is4() || !isIPv6 && myIP.Is6() {
					glog.V(100).Infof("Wrong IP version detected; %q", parsedIP)

					return false, nil
				}

				glog.V(100).Infof("Comparing %q with expected %q", parsedIP, expectedIPs)

				for _, expectedIP := range expectedIPs {
					if parsedIP == expectedIP {
						return true, nil
					}
				}

				glog.V(100).Infof("Mismatched IP address. Expected %q got %q", expectedIPs, parsedIP)

				return false, nil
			})

		if err != nil {
			return fmt.Errorf("failed to run command from within pod %s: %w", clientPod.Object.Name, err)
		}
	}

	return nil
}

func getExpectedEgressIPList() ([]string, error) {
	By("Getting a map of source nodes and assigned Egress IPs for these nodes")

	egressIPObj, err := egressip.Pull(APIClient, RDSCoreConfig.EgressIPName)

	if err != nil {
		glog.V(100).Infof("Failed to pull egressIP %q object: %v", RDSCoreConfig.EgressIPName, err)

		return nil, fmt.Errorf("failed to retrieve egressIP %s object: %w", RDSCoreConfig.EgressIPName, err)
	}

	egressIPSet, err := egressIPObj.GetAssignedEgressIPMap()

	if err != nil {
		glog.V(100).Infof("Failed to retrieve egressIP %s assigned egressIPs map: %v",
			RDSCoreConfig.EgressIPName, err)

		return nil, fmt.Errorf("failed to retrieve egressIP %s assigned egressIPs map: %w",
			RDSCoreConfig.EgressIPName, err)
	}

	if len(egressIPSet) != 2 {
		glog.V(100).Infof("EgressIPs assigned to the wrong number of nodes: %v", egressIPSet)

		return nil, fmt.Errorf("egressIPs assigned to the wrong number of nodes: %v", egressIPSet)
	}

	for _, nodeName := range []string{RDSCoreConfig.EgressIPNodeOne, RDSCoreConfig.EgressIPNodeTwo} {
		egressIPOne := egressIPSet[RDSCoreConfig.EgressIPNodeOne]
		egressIPTwo := egressIPSet[RDSCoreConfig.EgressIPNodeTwo]

		if egressIPOne != RDSCoreConfig.EgressIPv4 || egressIPTwo != RDSCoreConfig.EgressIPv6 {
			return nil, fmt.Errorf("EgressIP address assigned to the node %s not correctly configured; "+
				"current value: %s, expected values to be set %s or %s", nodeName, egressIPOne,
				RDSCoreConfig.EgressIPRemoteIPv4, RDSCoreConfig.EgressIPRemoteIPv6)
		}
	}

	configuredEgressIPs :=
		[]string{egressIPSet[RDSCoreConfig.EgressIPNodeOne], egressIPSet[RDSCoreConfig.EgressIPNodeTwo]}

	return configuredEgressIPs, nil
}

// CreateEgressIPTestDeployment create egressIP test setup to verify connectivity with the external server.
func CreateEgressIPTestDeployment() {
	By("Creating the EgressIP test source deployment")

	err := cleanUpDeployments(APIClient)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to cleanup deployments: %v", err))

	glog.V(100).Infof("Creating the EgressIP assigned agnhost probers in namespace %s for node %s and %s",
		RDSCoreConfig.EgressIPNamespaceOne, RDSCoreConfig.EgressIPNodeOne, RDSCoreConfig.EgressIPNodeTwo)

	proberDeploySANameOne, err := createAgnhostRBAC(APIClient, RDSCoreConfig.EgressIPNamespaceOne)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to create prober RBAC in namespace %s: %v", RDSCoreConfig.EgressIPNamespaceOne, err))

	for _, nodeToAssign := range []string{RDSCoreConfig.EgressIPNodeOne,
		RDSCoreConfig.EgressIPNodeTwo, RDSCoreConfig.NonEgressIPNode} {
		err = createAgnhostDeployment(
			APIClient,
			RDSCoreConfig.EgressIPNamespaceOne,
			proberDeploySANameOne,
			nodeToAssign,
			egressIPPodLabelsMap)
		Expect(err).ToNot(HaveOccurred(),
			fmt.Sprintf("Failed to create prober deployment for node %s in namespace %s: %v",
				nodeToAssign, RDSCoreConfig.EgressIPNamespaceOne, err))
	}

	glog.V(100).Infof("Creating the EgressIP assigned agnhost probers in namespace %s for node %s",
		RDSCoreConfig.EgressIPNamespaceTwo, RDSCoreConfig.EgressIPNodeTwo)

	proberDeploySANameTwo, err := createAgnhostRBAC(APIClient, RDSCoreConfig.EgressIPNamespaceTwo)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to create prober RBAC in namespace %s: %v", RDSCoreConfig.EgressIPNamespaceTwo, err))

	err = createAgnhostDeployment(
		APIClient,
		RDSCoreConfig.EgressIPNamespaceTwo,
		proberDeploySANameTwo,
		RDSCoreConfig.EgressIPNodeTwo,
		egressIPPodLabelsMap)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to create prober deployment for node %s in namespace %s: %v",
			RDSCoreConfig.EgressIPNodeTwo, RDSCoreConfig.EgressIPNamespaceTwo, err))

	glog.V(100).Infof("Creating the non EgressIP assigned agnhost prober in namespace %s for node %s",
		RDSCoreConfig.EgressIPNamespaceOne, RDSCoreConfig.EgressIPNodeOne)

	nonEIPPodLabelsMap := map[string]string{
		strings.Split(nonEgressIPPodLabel, "=")[0]: strings.Split(nonEgressIPPodLabel, "=")[1],
	}

	err = createAgnhostDeployment(
		APIClient,
		RDSCoreConfig.EgressIPNamespaceOne,
		proberDeploySANameOne,
		RDSCoreConfig.EgressIPNodeOne,
		nonEIPPodLabelsMap)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to create prober deployment for node %s in namespace %s: %v",
			RDSCoreConfig.EgressIPNodeOne, RDSCoreConfig.EgressIPNamespaceOne, err))
}

func verifyEgressIPConnectivityBalancedTraffic(isIPv6 bool) {
	By("Spawning the prober pod on the EgressIP non assignable host")

	expectedIPs, err := getExpectedEgressIPList()
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to retrieve configured eIP addresses list from the egressIP %s: %v",
			RDSCoreConfig.EgressIPName, err))

	proberPodObjects, err := pod.List(APIClient, RDSCoreConfig.EgressIPNamespaceOne, egressIPPodSelector)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to retrieve prober pods list from namespace %s with label %v: %v",
			RDSCoreConfig.EgressIPNamespaceOne, egressIPPodSelector, err))

	err = sendTrafficCheckIP(proberPodObjects, isIPv6, expectedIPs)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Server response was note received: %v", err))
}

// VerifyEgressIPConnectivityBalancedTrafficIPv4 verifies egress traffic works with egressIP
// applied for the external target.
func VerifyEgressIPConnectivityBalancedTrafficIPv4() {
	verifyEgressIPConnectivityBalancedTraffic(false)
}

// VerifyEgressIPConnectivityBalancedTrafficIPv6 verifies egress traffic works with egressIP
// applied for the external target.
func VerifyEgressIPConnectivityBalancedTrafficIPv6() {
	verifyEgressIPConnectivityBalancedTraffic(true)
}

func verifyEgressIPConnectivityThreeNodes(isIPv6 bool) {
	By("Spawning the prober pods on the EgressIP assignable hosts")

	expectedIPs, err := getExpectedEgressIPList()
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to retrieve configured eIP addresses list from the egressIP %s: %v",
			RDSCoreConfig.EgressIPName, err))

	proberPodObjects, err := pod.List(APIClient, RDSCoreConfig.EgressIPNamespaceOne, egressIPPodSelector)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to retrieve prober pods list from namespace %s with label %s: %v",
			RDSCoreConfig.EgressIPNamespaceOne, RDSCoreConfig.EgressIPPodLabel, err))

	err = sendTrafficCheckIP(proberPodObjects, isIPv6, expectedIPs)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Server response was note received: %v", err))

	proberPodObjects, err = pod.List(APIClient, RDSCoreConfig.EgressIPNamespaceTwo, egressIPPodSelector)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to retrieve prober pods list from namespace %s with label %s: %v",
			RDSCoreConfig.EgressIPNamespaceTwo, RDSCoreConfig.EgressIPPodLabel, err))

	err = sendTrafficCheckIP(proberPodObjects, isIPv6, expectedIPs)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Server response was not received: %v", err))
}

// VerifyEgressIPConnectivityThreeNodesIPv4 verifies egress traffic works with egressIP
// applied for the external target.
func VerifyEgressIPConnectivityThreeNodesIPv4() {
	verifyEgressIPConnectivityThreeNodes(false)
}

// VerifyEgressIPConnectivityThreeNodesIPv6 verifies egress traffic works with egressIP
// applied for the external target.
func VerifyEgressIPConnectivityThreeNodesIPv6() {
	verifyEgressIPConnectivityThreeNodes(true)
}

// VerifyEgressIPForPodWithWrongLabel verifies egress traffic applies only for pods with
// the correct pod label defined.
func VerifyEgressIPForPodWithWrongLabel() {
	By("Spawning the prober pods on the EgressIP assignable hosts")

	expectedIPs, err := getExpectedEgressIPList()
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to retrieve configured eIP addresses list from the egressIP %s: %v",
			RDSCoreConfig.EgressIPName, err))

	proberPodObjects, err := pod.List(APIClient, RDSCoreConfig.EgressIPNamespaceOne, egressIPPodSelector)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to retrieve prober pods list from namespace %s with label %s: %v",
			RDSCoreConfig.EgressIPNamespaceOne, RDSCoreConfig.EgressIPPodLabel, err))

	err = sendTrafficCheckIP(proberPodObjects, false, expectedIPs)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Server response was note received: %v", err))

	proberPodObjects, err = pod.List(APIClient, RDSCoreConfig.EgressIPNamespaceOne, nonEgressIPPodSelector)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to retrieve prober pods list from namespace %s with label %s: %v",
			RDSCoreConfig.EgressIPNamespaceTwo, RDSCoreConfig.EgressIPPodLabel, err))

	err = sendTrafficCheckIP(proberPodObjects, false, expectedIPs)
	Expect(err).To(HaveOccurred(),
		fmt.Sprintf("Server response was received with the not correct egressIP address: %v", err))
}

// VerifyEgressIPForNamespaceWithWrongLabel verifies egress traffic applies only for the pods
// run in the namespace assigned to the eIP service.
func VerifyEgressIPForNamespaceWithWrongLabel() {
	glog.V(100).Infof("Create new, not assigned to the eIP service namespace %s", nonEgressIPNamespace)

	_, err := namespace.NewBuilder(APIClient, nonEgressIPNamespace).Create()
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to create namespace %s: %v", nonEgressIPNamespace, err))

	proberDeploySAName, err := createAgnhostRBAC(APIClient, nonEgressIPNamespace)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to create prober RBAC in namespace %s: %v", nonEgressIPNamespace, err))

	err = createAgnhostDeployment(
		APIClient,
		nonEgressIPNamespace,
		proberDeploySAName,
		RDSCoreConfig.EgressIPNodeOne,
		egressIPPodLabelsMap)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to create prober deployment for node %s in namespace %s: %v",
			RDSCoreConfig.EgressIPNodeOne, nonEgressIPNamespace, err))

	By("Spawning the prober pods on the EgressIP assignable hosts")

	expectedIPs, err := getExpectedEgressIPList()
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to retrieve configured eIP addresses list from the egressIP %s: %v",
			RDSCoreConfig.EgressIPName, err))

	proberPodObjects, err := pod.List(APIClient, nonEgressIPNamespace, egressIPPodSelector)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to retrieve prober pods list from namespace %s with label %s: %v",
			nonEgressIPNamespace, RDSCoreConfig.EgressIPPodLabel, err))

	err = sendTrafficCheckIP(proberPodObjects, false, expectedIPs)
	Expect(err).To(HaveOccurred(),
		fmt.Sprintf("Server response was received with the not correct egressIP address: %v", err))
}

// VerifyEgressIPOneNamespaceThreeNodesBalancedEIPTrafficIPv4 verifies egress traffic works with egressIP
// applied for the external target.
func VerifyEgressIPOneNamespaceThreeNodesBalancedEIPTrafficIPv4() {
	By("Creating egressIP test setup: 7 pods run in the 3 different namespaces on 3 nodes")

	CreateEgressIPTestDeployment()

	By("Verify egressIP connectivity for the same namespace")

	for iter := 0; iter < 5; iter++ {
		VerifyEgressIPConnectivityBalancedTrafficIPv4()
	}
}

// VerifyEgressIPTwoNamespacesThreeNodesIPv4 verifies egress traffic works with egressIP
// applied for the external target.
func VerifyEgressIPTwoNamespacesThreeNodesIPv4() {
	By("Creating egressIP test setup to verify connectivity on the two namespaces")

	CreateEgressIPTestDeployment()

	By("Verify egressIP connectivity for the different namespaces")

	VerifyEgressIPConnectivityThreeNodesIPv4()
}

// VerifyEgressIPOneNamespaceThreeNodesBalancedEIPTrafficIPv6 verifies egress traffic works with egressIP
// applied for the external target.
func VerifyEgressIPOneNamespaceThreeNodesBalancedEIPTrafficIPv6() {
	By("Creating egressIP test setup: 7 pods run in the 3 different namespaces on 3 nodes")

	CreateEgressIPTestDeployment()

	By("Verify egressIP connectivity for the same namespace")

	for iter := 0; iter < 5; iter++ {
		VerifyEgressIPConnectivityBalancedTrafficIPv6()
	}
}

// VerifyEgressIPTwoNamespacesThreeNodesIPv6 verifies egress traffic works with egressIP
// applied for the external target.
func VerifyEgressIPTwoNamespacesThreeNodesIPv6() {
	By("Creating egressIP test setup to verify connectivity on the two namespaces")

	CreateEgressIPTestDeployment()

	By("Verify egressIP connectivity for the different namespaces")

	VerifyEgressIPConnectivityThreeNodesIPv6()
}

// VerifyEgressIPOneNamespaceOneNodeWrongPodLabel verifies egress traffic works with egressIP
// applied for the external target with the single namespace and wrong pod label.
func VerifyEgressIPOneNamespaceOneNodeWrongPodLabel() {
	By("Creating egressIP test setup to verify connectivity on the two namespaces")

	CreateEgressIPTestDeployment()

	By("Verify egressIP connectivity for the pod with the not correctly defined label")

	VerifyEgressIPForPodWithWrongLabel()
}

// VerifyEgressIPWrongNsLabel verifies egress traffic works with egressIP
// applied for the external target when two egressIP pods from the different namespaces run on the same node
// when one of the namespaces not assigned to the egressIP.
func VerifyEgressIPWrongNsLabel() {
	By("Creating egressIP test setup to verify connectivity on the two namespaces")

	CreateEgressIPTestDeployment()

	By("Verify egressIP connectivity for the pods run in the namespace not assigned to the eIP")

	VerifyEgressIPForNamespaceWithWrongLabel()
}
