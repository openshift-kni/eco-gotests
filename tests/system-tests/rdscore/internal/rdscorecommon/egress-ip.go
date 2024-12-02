package rdscorecommon

import (
	"fmt"
	"strings"
	"time"

	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	scc "github.com/openshift-kni/eco-gotests/tests/system-tests/internal/scc"

	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/apiobjectshelper"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/deployment"
	"github.com/openshift-kni/eco-goinfra/pkg/egressip"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/sniffer"
	. "github.com/openshift-kni/eco-gotests/tests/system-tests/rdscore/internal/rdscoreinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/rdscore/internal/rdscoreparams"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	snifferNamespace       = "rds-egress-ns"
	proberDeployRBACName   = "privileged-rdscore-prober"
	proberDeploySAName     = "rdscore-prober-sa"
	proberRBACRole         = "system:openshift:scc:privileged"
	numberOfRequestsToSend = 10
	proberTargetProtocol   = "http"
)

var (
	egressIPNodesList = []string{RDSCoreConfig.EgressIPNodeOne, RDSCoreConfig.EgressIPNodeTwo}
	proberTargetPort  = RDSCoreConfig.EgressIPTcpPort

	egressIPPodLabelsMap = map[string]string{
		strings.Split(RDSCoreConfig.EgressIPPodLabel,
			"=")[0]: strings.Split(RDSCoreConfig.EgressIPPodLabel, "=")[1]}

	egressIPPodSelector = metav1.ListOptions{
		LabelSelector: RDSCoreConfig.EgressIPPodLabel,
	}
)

// createAgnhostDeployment creates the route, service and deployment that will be used as
// a source for EgressIP tests. Returns the route name that can be queried to run queries against the source pods.
//
//nolint:funlen
func createAgnhostDeployment(
	apiClient *clients.Settings,
	egressIPNamespace string,
	scheduleOnHosts []string) error {
	proberServiceName := fmt.Sprintf("%s-service", egressIPNamespace)
	proberDeploymentName := fmt.Sprintf("%s-deployment", egressIPNamespace)

	var err error

	glog.V(100).Infof("Checking prober deployment don't exist")

	err = apiobjectshelper.DeleteDeployment(apiClient,
		proberDeploymentName,
		egressIPNamespace,
		RDSCoreConfig.EgressIPPodLabel)

	if err != nil {
		return fmt.Errorf("failed to delete deployment %s from egressIPNamespace %s",
			proberDeploymentName, egressIPNamespace)
	}

	glog.V(100).Infof("Sleeping 10 seconds")
	time.Sleep(10 * time.Second)

	glog.V(100).Infof("Adding SCC privileged to the prober namespace")

	err = scc.AddPrivilegedSCCtoDefaultSA(egressIPNamespace)
	if err != nil {
		return fmt.Errorf("failed to add SCC privileged to the prober namespace %s: %w",
			egressIPNamespace, err)
	}

	glog.V(100).Infof("Removing Service")

	err = apiobjectshelper.DeleteService(apiClient, proberServiceName, egressIPNamespace)

	if err != nil {
		return fmt.Errorf("failed to remove service %q from egressIPNamespace %q; %w",
			proberServiceName, egressIPNamespace, err)
	}

	glog.V(100).Infof("Removing ServiceAccount")

	err = apiobjectshelper.DeleteServiceAccount(apiClient, proberDeploySAName, egressIPNamespace)

	if err != nil {
		return fmt.Errorf("failed to remove serviceAccount %q from egressIPNamespace %q",
			proberDeploySAName, egressIPNamespace)
	}

	glog.V(100).Infof("Creating ServiceAccount")

	err = apiobjectshelper.CreateServiceAccount(apiClient, proberDeploySAName, egressIPNamespace)

	if err != nil {
		return fmt.Errorf("failed to create serviceAccount %q in egressIPNamespace %q",
			proberDeploySAName, egressIPNamespace)
	}

	glog.V(100).Infof("Removing Cluster RBAC")

	err = apiobjectshelper.DeleteClusterRBAC(apiClient, proberDeployRBACName)

	if err != nil {
		return fmt.Errorf("failed to delete prober RBAC %q", proberDeployRBACName)
	}

	glog.V(100).Infof("Creating Cluster RBAC")

	err = apiobjectshelper.CreateClusterRBAC(apiClient, proberDeployRBACName, proberRBACRole,
		proberDeploySAName, egressIPNamespace)

	if err != nil {
		return fmt.Errorf("failed to create prober RBAC %q in egressIPNamespace %s",
			proberDeployRBACName, egressIPNamespace)
	}

	glog.V(100).Infof("Defining container configuration")

	containerCmd := []string{
		"/agnhost",
		"netexec",
		"--http-port",
		fmt.Sprintf("%d", RDSCoreConfig.EgressIPTcpPort),
	}

	deployContainer := defineProberContainer(RDSCoreConfig.EgressIPDeploymentImage, containerCmd)

	glog.V(100).Infof("Obtaining container definition")

	deployContainerCfg, err := deployContainer.GetContainerCfg()
	if err != nil {
		return fmt.Errorf("failed to obtain container definition: %w", err)
	}

	glog.V(100).Infof("Defining deployment configuration")

	proberDeployment := defineProberDeployment(
		apiClient,
		deployContainerCfg,
		proberDeploymentName,
		egressIPNamespace,
		proberDeploySAName,
		scheduleOnHosts,
		egressIPPodLabelsMap)

	glog.V(100).Infof("Creating deployment")

	proberDeployment, err = proberDeployment.CreateAndWaitUntilReady(5 * time.Minute)
	if err != nil {
		return fmt.Errorf("failed to create deployment %s in namespace %s: %w",
			proberDeploymentName, egressIPNamespace, err)
	}

	if proberDeployment == nil {
		return fmt.Errorf("failed to create deployment %s in namespace %s",
			proberDeploymentName, egressIPNamespace)
	}

	return nil
}

func defineProberDeployment(
	apiClient *clients.Settings,
	containerConfig *corev1.Container,
	deployName, deployNs, saName string,
	scheduleOnHosts []string,
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
								Values:   scheduleOnHosts,
							},
						},
					},
				},
			},
		},
	}

	proberDeployment := deployment.NewBuilder(apiClient, deployName, deployNs, deployLabels, *containerConfig)

	glog.V(100).Infof("Assigning ServiceAccount %q to the deployment", saName)

	proberDeployment = proberDeployment.WithServiceAccountName(saName)

	glog.V(100).Infof("Setting Replicas count")

	proberDeployment = proberDeployment.WithReplicas(int32(len(scheduleOnHosts)))

	proberDeployment = proberDeployment.WithHostNetwork(true)

	proberDeployment = proberDeployment.WithAffinity(&nodeAffinity)

	return proberDeployment
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

// CreateEgressIPWithSingleNamespace create egressIP test setup to verify connectivity on the same namespace.
func CreateEgressIPWithSingleNamespace() {
	By("Creating the packet sniffer deployment with number of pods equals number of EgressIP nodes")

	_, err :=
		sniffer.CreatePacketSnifferDeployment(
			APIClient,
			proberTargetPort,
			proberTargetProtocol,
			RDSCoreConfig.EgressIPPacketSnifferInterface,
			snifferNamespace,
			egressIPNodesList)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to create sniffer deployment for %s and %s nodes in namespace %s: %v",
			RDSCoreConfig.EgressIPNodeOne, RDSCoreConfig.EgressIPNodeTwo, snifferNamespace, err))

	By("Creating the EgressIP test source deployment with number of pods equals number of EgressIP nodes")

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Create prober deployment based on the agnhost image")

	err = createAgnhostDeployment(
		APIClient,
		RDSCoreConfig.EgressIPNamespaceOne,
		egressIPNodesList)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to create prober deployment and ingress route for nodes %q: %v",
			egressIPNodesList, err))
}

// VerifyEgressIPConnectivityForSingleNamespace verifies egress traffic works with egressIP
// applied for the external target in the same namespace.
//
//nolint:funlen
func VerifyEgressIPConnectivityForSingleNamespace() {
	By("Getting a map of source nodes and assigned Egress IPs for these nodes")

	egressIPObj, err := egressip.Pull(APIClient, RDSCoreConfig.EgressIPName)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to retrieve egressIP %s object: %v", RDSCoreConfig.EgressIPName, err))

	egressIPSet, err := egressIPObj.GetAssignedEgressIPMap()
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to retrieve egressIP %s assigned egressIPs map: %v", RDSCoreConfig.EgressIPName, err))
	Expect(len(egressIPSet)).To(Equal(2),
		fmt.Sprintf("EgressIPs assigned to the wrong number of nodes: %v", egressIPSet))

	By("Spawning the prober pods on the EgressIP assignable hosts")

	for clusterNode, proberTargetHost := range egressIPSet {
		for i := 0; i < 2; i++ {
			By(fmt.Sprintf("Sending requests from prober and making sure that %d requests with search string and "+
				"EgressIPs %v were seen", numberOfRequestsToSend, egressIPSet))

			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Get prober pod object")

			proberPodObjects, err := pod.List(APIClient, RDSCoreConfig.EgressIPNamespaceOne, egressIPPodSelector)
			Expect(err).ToNot(HaveOccurred(),
				fmt.Sprintf("Failed to retrieve prober pods list from namespace %s with label %s: %v",
					RDSCoreConfig.EgressIPNamespaceOne, RDSCoreConfig.EgressIPPodLabel, err))

			var proberPodObj *pod.Builder

			for _, podObj := range proberPodObjects {
				if podObj.Object.Spec.NodeName == clusterNode {
					glog.V(rdscoreparams.RDSCoreLogLevel).Infof("pod: %s", podObj.Definition.Name)
					proberPodObj = podObj
				}
			}

			Expect(proberPodObj).ToNot(Equal(nil),
				fmt.Sprintf("prober pod not found running on the %s node", clusterNode))

			err = sniffer.SendTrafficCheckLogs(
				APIClient,
				proberPodObj,
				snifferNamespace,
				RDSCoreConfig.EgressIPPodLabel,
				proberTargetHost,
				proberTargetProtocol,
				proberTargetPort,
				numberOfRequestsToSend,
				numberOfRequestsToSend)
			Expect(err).ToNot(HaveOccurred(),
				fmt.Sprintf("Failed to find required number of requests %d: %v", numberOfRequestsToSend, err))
		}
	}

	By("Verify defined, but not assigned egressIP address is reachable")

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Get prober pod object")

	podSelector := metav1.ListOptions{
		LabelSelector: RDSCoreConfig.EgressIPPodLabel,
	}

	proberPodObjects, err := pod.List(APIClient, RDSCoreConfig.EgressIPNamespaceOne, podSelector)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to retrieve prober pods list from namespace %s with label %s: %v",
			RDSCoreConfig.EgressIPNamespaceOne, RDSCoreConfig.EgressIPPodLabel, err))

	err = sniffer.SendTrafficCheckLogs(
		APIClient,
		proberPodObjects[0],
		snifferNamespace,
		RDSCoreConfig.EgressIPPodLabel,
		RDSCoreConfig.EgressIPRemoteIPThree,
		proberTargetProtocol,
		proberTargetPort,
		numberOfRequestsToSend,
		numberOfRequestsToSend)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to find required number of requests %d: %v", numberOfRequestsToSend, err))

	By("Verify undefined egressIP address is not reachable")

	err = sniffer.SendTrafficCheckLogs(
		APIClient,
		proberPodObjects[0],
		snifferNamespace,
		RDSCoreConfig.EgressIPPodLabel,
		RDSCoreConfig.EgressIPRemoteIPFour,
		proberTargetProtocol,
		proberTargetPort,
		numberOfRequestsToSend,
		numberOfRequestsToSend)
	Expect(err).To(HaveOccurred(), "No packets should be seen for the undefined egressIP address")
}

// VerifyEgressIPWithSingleNamespace verifies egress traffic works with egressIP
// applied for the external target.
func VerifyEgressIPWithSingleNamespace() {
	By("Creating egressIP test setup to verify connectivity on the same namespace")

	CreateEgressIPWithSingleNamespace()

	By("Verify egressIP connectivity for the same namespace")

	VerifyEgressIPConnectivityForSingleNamespace()
}

// CreateEgressIPMixedNodesAndNamespaces create egressIP test setup to verify connectivity on the
// different namespaces and nodes.
func CreateEgressIPMixedNodesAndNamespaces() {
	By("Creating the packet sniffer deployment with number of pods equals number of EgressIP nodes")

	_, err :=
		sniffer.CreatePacketSnifferDeployment(
			APIClient,
			proberTargetPort,
			proberTargetProtocol,
			RDSCoreConfig.EgressIPPacketSnifferInterface,
			snifferNamespace,
			egressIPNodesList)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to create sniffer deployment for %s and %s nodes in namespace %s: %v",
			RDSCoreConfig.EgressIPNodeOne, RDSCoreConfig.EgressIPNodeTwo, snifferNamespace, err))

	By(fmt.Sprintf("Create prober pod deployment in namespace %s on the node %s",
		RDSCoreConfig.EgressIPNamespaceOne, RDSCoreConfig.EgressIPNodeOne))

	err = createAgnhostDeployment(
		APIClient,
		RDSCoreConfig.EgressIPNamespaceOne,
		[]string{RDSCoreConfig.EgressIPNodeOne})
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to create prober deployment for node %s: %v", RDSCoreConfig.EgressIPNodeOne, err))

	By(fmt.Sprintf("Create prober pod deployment in namespace %s on the node %s",
		RDSCoreConfig.EgressIPNamespaceTwo, RDSCoreConfig.EgressIPNodeTwo))

	err = createAgnhostDeployment(
		APIClient,
		RDSCoreConfig.EgressIPNamespaceTwo,
		[]string{RDSCoreConfig.EgressIPNodeTwo})
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to create prober deployment in namespace %s for node %s: %v",
			RDSCoreConfig.EgressIPNamespaceTwo, RDSCoreConfig.EgressIPNodeTwo, err))

	By(fmt.Sprintf("Create prober deployment in namespace %s on the node %s",
		RDSCoreConfig.EgressIPNamespaceTwo, RDSCoreConfig.NonEgressIPNodeOne))

	err = createAgnhostDeployment(
		APIClient,
		RDSCoreConfig.EgressIPNamespaceTwo,
		[]string{RDSCoreConfig.NonEgressIPNodeOne})
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to create prober deployment for node %s: %v", RDSCoreConfig.NonEgressIPNodeOne, err))

	By(fmt.Sprintf("Create prober deployment in namespace %s on the node %s",
		RDSCoreConfig.EgressIPNamespaceThree, RDSCoreConfig.EgressIPNodeOne))

	err = createAgnhostDeployment(
		APIClient,
		RDSCoreConfig.EgressIPNamespaceThree,
		[]string{RDSCoreConfig.EgressIPNodeOne})
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to create prober deployment for node %s: %v", RDSCoreConfig.EgressIPNodeOne, err))
}

// VerifyEgressIPConnectivityMixedNodesAndNamespaces verifies egress traffic works with egressIP
// applied for the external target on the different namespaces and nodes.
//
//nolint:funlen
func VerifyEgressIPConnectivityMixedNodesAndNamespaces() {
	By("Getting a map of source nodes and assigned Egress IPs for these nodes")

	egressIPObj, err := egressip.Pull(APIClient, RDSCoreConfig.EgressIPName)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to retrieve egressIP %s object: %v", RDSCoreConfig.EgressIPName, err))

	egressIPSet, err := egressIPObj.GetAssignedEgressIPMap()
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to retrieve egressIP %s assigned egressIPs map: %v", RDSCoreConfig.EgressIPName, err))
	Expect(len(egressIPSet)).To(Equal(2),
		fmt.Sprintf("EgressIPs assigned to the wrong number of nodes: %v", egressIPSet))

	By(fmt.Sprintf("Sending requests from prober pod in namespace %s "+
		"on the node %s and making sure that %d requests were seen",
		RDSCoreConfig.EgressIPNamespaceOne, RDSCoreConfig.EgressIPNodeOne, numberOfRequestsToSend))

	proberPodObjects, err := pod.List(APIClient, RDSCoreConfig.EgressIPNamespaceOne, egressIPPodSelector)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to retrieve prober pods list from namespace %s with label %s: %v",
			RDSCoreConfig.EgressIPNamespaceOne, RDSCoreConfig.EgressIPPodLabel, err))

	err = sniffer.SendTrafficCheckLogs(
		APIClient,
		proberPodObjects[0],
		snifferNamespace,
		RDSCoreConfig.EgressIPPodLabel,
		egressIPSet[RDSCoreConfig.EgressIPNodeOne],
		proberTargetProtocol,
		proberTargetPort,
		numberOfRequestsToSend,
		numberOfRequestsToSend)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to find required number of requests %d: %v", numberOfRequestsToSend, err))

	By(fmt.Sprintf("Sending requests from prober pod in namespace %s "+
		"on the node %s and making sure that %d requests were seen",
		RDSCoreConfig.EgressIPNamespaceTwo, RDSCoreConfig.EgressIPNodeTwo, numberOfRequestsToSend))

	proberPodObjects, err = pod.List(APIClient, RDSCoreConfig.EgressIPNamespaceTwo, egressIPPodSelector)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to retrieve prober pods list from namespace %s with label %s: %v",
			RDSCoreConfig.EgressIPNamespaceTwo, RDSCoreConfig.EgressIPPodLabel, err))

	err = sniffer.SendTrafficCheckLogs(
		APIClient,
		proberPodObjects[0],
		snifferNamespace,
		RDSCoreConfig.EgressIPPodLabel,
		egressIPSet[RDSCoreConfig.EgressIPNodeTwo],
		proberTargetProtocol,
		proberTargetPort,
		numberOfRequestsToSend,
		numberOfRequestsToSend)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to find required number of requests %d: %v", numberOfRequestsToSend, err))

	By(fmt.Sprintf("Sending requests from prober pod in namespace %s "+
		"on the node %s and making sure that %d requests were seen",
		RDSCoreConfig.EgressIPNamespaceTwo, RDSCoreConfig.NonEgressIPNodeOne, numberOfRequestsToSend))

	proberPodObjects, err = pod.List(APIClient, RDSCoreConfig.EgressIPNamespaceTwo, egressIPPodSelector)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to retrieve prober pods list from namespace %s with label %s: %v",
			RDSCoreConfig.EgressIPNamespaceTwo, RDSCoreConfig.EgressIPPodLabel, err))

	err = sniffer.SendTrafficCheckLogs(
		APIClient,
		proberPodObjects[0],
		snifferNamespace,
		RDSCoreConfig.EgressIPPodLabel,
		egressIPSet[RDSCoreConfig.EgressIPNodeTwo],
		proberTargetProtocol,
		proberTargetPort,
		numberOfRequestsToSend,
		numberOfRequestsToSend)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to find required number of requests %d: %v", numberOfRequestsToSend, err))

	By(fmt.Sprintf("Sending requests from prober pod in namespace %s "+
		"on the node %s and making sure that %d requests were seen",
		RDSCoreConfig.EgressIPNamespaceThree, RDSCoreConfig.EgressIPNodeOne, numberOfRequestsToSend))

	proberPodObjects, err = pod.List(APIClient, RDSCoreConfig.EgressIPNamespaceThree, egressIPPodSelector)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to retrieve prober pods list from namespace %s with label %s: %v",
			RDSCoreConfig.EgressIPNamespaceThree, RDSCoreConfig.EgressIPPodLabel, err))

	err = sniffer.SendTrafficCheckLogs(
		APIClient,
		proberPodObjects[0],
		snifferNamespace,
		RDSCoreConfig.EgressIPPodLabel,
		egressIPSet[RDSCoreConfig.EgressIPNodeOne],
		proberTargetProtocol,
		proberTargetPort,
		numberOfRequestsToSend,
		numberOfRequestsToSend)
	Expect(err).To(HaveOccurred(), "traffic receive for the pod from the non-egressIP assigned namespace")
}

// VerifyEgressIPDifferentNodesAndNamespaces verifies egress traffic works with egressIP applied
// for the external target with two additional namespaces in use on a different nodes.
func VerifyEgressIPDifferentNodesAndNamespaces() {
	By("Creating egressIP test setup to verify connectivity on the different namespaces and nodes")

	CreateEgressIPMixedNodesAndNamespaces()

	By("Verify egressIP connectivity for the same namespace")

	VerifyEgressIPConnectivityMixedNodesAndNamespaces()
}
