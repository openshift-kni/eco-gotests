package rdscorecommon

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift-kni/eco-goinfra/pkg/deployment"
	"github.com/openshift-kni/eco-goinfra/pkg/egressservice"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-goinfra/pkg/service"
	. "github.com/openshift-kni/eco-gotests/tests/system-tests/rdscore/internal/rdscoreinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/rdscore/internal/rdscoreparams"
)

const (
	egressSVCContainer         = "rds-egress-container"
	egressSVCDeploy1Name       = "rds-egress-deploy"
	egressSVC1Name             = "egress-svc-1"
	egressSVC1Labels           = "rds-egress=rds-core"
	egressSVCDeploy2Name       = "rds-egress-deploy2"
	egressSVC2Name             = "egress-svc-2"
	egressSVC2Labels           = "rds-egress=rds-core-2"
	servicePort          int32 = 9090
	serviceTargetPort    int32 = 9090
)

func defineEgressSVCContainer(cName, cImage string, cCmd []string) *pod.ContainerBuilder {
	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Creating container %q", egressSVCContainer)

	deployContainer := pod.NewContainerBuilder(cName, cImage, cCmd)

	cPort := corev1.ContainerPort{
		ContainerPort: servicePort,
		Protocol:      corev1.ProtocolTCP,
	}

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Adding ContainerPort definition:\n%v\n", cPort)
	deployContainer = deployContainer.WithPorts([]corev1.ContainerPort{cPort})

	return deployContainer
}

func defineEgressSVCDeployment(containerConfig *corev1.Container, deployName, deployNs string,
	deployLabels, nodeSelector map[string]string, replicas int32) *deployment.Builder {
	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Defining deployment %q in %q ns", deployName, deployNs)

	deploy := deployment.NewBuilder(APIClient, deployName, deployNs, deployLabels, *containerConfig)

	By("Setting Replicas count")

	deploy = deploy.WithReplicas(replicas)

	if len(nodeSelector) != 0 {
		By("Adding NodeSelector to the deployment")

		deploy = deploy.WithNodeSelector(nodeSelector)
	}

	return deploy
}

func defineService(svcName, svcNSName string,
	svcSelector map[string]string, svcPort corev1.ServicePort) *service.Builder {
	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Defining service %q in %q ns", svcName, svcNSName)

	return service.NewBuilder(APIClient, svcName, svcNSName, svcSelector, svcPort)
}

func deleteService(svcName, svcNSName string) {
	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Deleting service %q in %q ns", svcName, svcNSName)

	var ctx SpecContext

	svcBuilder, err := service.Pull(APIClient, svcName, svcNSName)

	switch {
	case svcBuilder == nil && err != nil:
		{
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Service %q not found in %q ns", svcName, svcNSName)
		}
	case svcBuilder != nil && err == nil:
		{
			Eventually(func() bool {
				err := svcBuilder.Delete()

				if err != nil {
					glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Error deleting Service %q in %q namespace: %v",
						svcBuilder.Definition.Name, svcBuilder.Definition.Namespace, err)

					return false
				}

				glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Deleted Service %q in %q namespace:",
					svcName, svcNSName)

				return true
			}).WithContext(ctx).WithPolling(5*time.Second).WithTimeout(1*time.Minute).Should(
				BeTrue(), fmt.Sprintf("Failed to deleted service %q in %q namespace", svcName, svcNSName))
		}
	default:
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Unhandled error while deleting service %q in %q ns",
			svcName, svcNSName)
	}
}

func deleteEgressService(svcName, svcNSName string) {
	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Deleting EgressService %q in %q ns",
		svcName, svcNSName)

	var ctx SpecContext

	svcBuilder, err := egressservice.Pull(APIClient, svcName, svcNSName)

	switch {
	case svcBuilder == nil && err != nil:
		{
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("EgressService %q not found in %q ns", svcName, svcNSName)
		}
	case svcBuilder != nil && err == nil:
		{
			Eventually(func() bool {
				svcBuilder, err := svcBuilder.Delete()

				if err != nil {
					glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Error deleting EgressService %q in %q namespace: %v",
						svcBuilder.Definition.Name, svcBuilder.Definition.Namespace, err)

					return false
				}

				glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Deleted EgressService %q in %q namespace:",
					svcBuilder.Definition.Name, svcBuilder.Definition.Namespace)

				return true
			}).WithContext(ctx).WithPolling(5*time.Second).WithTimeout(1*time.Minute).Should(
				BeTrue(), fmt.Sprintf("Failed to deleted EgressService %q in %q namespace", svcName, svcNSName))
		}
	default:
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Unhandled error while deleting EgressService %q in %q ns",
			svcName, svcNSName)
	}
}

func waitForPodsGone(podNS, podSelector string) {
	By(fmt.Sprintf("Ensuring pods matching %q label in %q namespace are gone", podNS, podSelector))

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Ensuring pods matching %q label in %q namespace are gone",
		podSelector, podNS)

	var ctx SpecContext

	Eventually(func() bool {
		oldPods, err := pod.List(APIClient, podNS,
			metav1.ListOptions{LabelSelector: podSelector})

		if err != nil {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Error listing pods: %v", err)

			return false
		}

		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Found %d pods matching label", len(oldPods))

		return len(oldPods) == 0
	}).WithContext(ctx).WithPolling(5*time.Second).WithTimeout(1*time.Minute).Should(BeTrue(),
		"pods matching label() still present")
}

func verifyPodSourceAddress(clientPods []*pod.Builder, cmdToRun []string, expectedIP string) {
	By("Validating pods source address")

	for _, clientPod := range clientPods {
		var (
			parsedIP string
			ctx      SpecContext
		)

		Eventually(func() bool {

			result, err := clientPod.ExecCommand(cmdToRun, clientPod.Object.Spec.Containers[0].Name)

			if err != nil {
				glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Error running command from within a pod %q: %v",
					clientPod.Object.Name, err)

				return false
			}

			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Successfully executed command from within a pod %q: %v",
				clientPod.Object.Name, err)
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Command's output:\n\t%v", result.String())

			parsedIP, _, err = net.SplitHostPort(result.String())

			if err != nil {
				glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Failed to parse %q for host/port pair", result.String())

				return false
			}

			return true
		}).WithContext(ctx).WithPolling(5*time.Second).WithTimeout(1*time.Minute).Should(BeTrue(),
			"Failed to run command from within pod")

		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Comparing %q with expected %q", parsedIP, expectedIP)

		Expect(parsedIP).To(Equal(expectedIP),
			fmt.Sprintf("Mismatched IP address. Expected %q got %q", expectedIP, parsedIP))
	}
}

// func setupWorkloadWithClusterETP() {
//
// }

// VerifyEgressServiceWithClusterETP verifies EgressService with externalTrafficPolicy set to Cluster.
//
//nolint:funlen
func VerifyEgressServiceWithClusterETP(ctx SpecContext) {
	deleteDeployments(egressSVCDeploy1Name, RDSCoreConfig.EgressServiceNS)
	deleteService(egressSVC1Name, RDSCoreConfig.EgressServiceNS)
	deleteEgressService(egressSVC1Name, RDSCoreConfig.EgressServiceNS)

	waitForPodsGone(RDSCoreConfig.EgressServiceNS, egressSVC1Labels)

	podContainer := defineEgressSVCContainer(egressSVCContainer,
		RDSCoreConfig.EgressServiceDeploy1Image, RDSCoreConfig.EgressServiceDeploy1CMD)

	By("Reseting SecurityContext")

	podContainer = podContainer.WithSecurityContext(&corev1.SecurityContext{RunAsGroup: nil, RunAsUser: nil})

	cfgContainer, err := podContainer.GetContainerCfg()
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to get container definition: %v", err))

	deployLabels := map[string]string{
		strings.Split(egressSVC1Labels, "=")[0]: strings.Split(egressSVC1Labels, "=")[1],
	}

	deploy := defineEgressSVCDeployment(cfgContainer, egressSVCDeploy1Name,
		RDSCoreConfig.EgressServiceNS, deployLabels, map[string]string{}, int32(2))

	Expect(deploy).ToNot(BeNil(), "Failed to create deployment")

	deploy, err = deploy.CreateAndWaitUntilReady(5 * time.Minute)

	Expect(err).ToNot(HaveOccurred(), "Deployment hasn't reached Ready status")

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof(fmt.Sprintf("Deployment %q in %q namespace is Ready",
		deploy.Object.Name, deploy.Object.Namespace))

	By("Defining a ServicePort")

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Defining ServicePort object")

	svcPort, err := service.DefineServicePort(servicePort, serviceTargetPort, corev1.ProtocolTCP)

	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to define TargetPort: %v", err))

	By("Defining Service")

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Defining Service object")

	svcLabels := map[string]string{
		strings.Split(egressSVC1Labels, "=")[0]: strings.Split(egressSVC1Labels, "=")[1],
	}

	svcBuilder := defineService(egressSVC1Name, RDSCoreConfig.EgressServiceNS, svcLabels, *svcPort)

	Expect(svcBuilder).ToNot(BeNil(), "Failed to defined service")

	By("Setting type to LoadBalancer")

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Setting Service's type to 'LoadBalancer'")

	svcBuilder.Definition.Spec.Type = "LoadBalancer"

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Service Definition:\n%#v\n", svcBuilder.Definition)

	By("Setting AddressPool annotation")

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Setting MetalLB address-pool annotation")

	svcBuilder = svcBuilder.WithAnnotation(map[string]string{
		"metallb.universe.tf/address-pool": RDSCoreConfig.EgressServiceDeploy1IPAddrPool})

	By("Creating a service")
	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Creating Service object")

	svcBuilder, err = svcBuilder.Create()
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to create Service: %v", err))

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Created Service: %#v", svcBuilder.Object)

	By("Defining EgressService")

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Defining EgressService %q in %q namespace",
		egressSVC1Name, RDSCoreConfig.EgressServiceNS)

	egrSVCBuilder := egressservice.NewEgressServiceBuilder(APIClient, egressSVC1Name,
		RDSCoreConfig.EgressServiceNS, "LoadBalancerIP")

	if len(RDSCoreConfig.EgressServiceDeploy1NodeSelector) != 0 {
		By("Setting nodeSelector for EgressService")

		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Configuring nodeSelector on EgressService %q in %q namespace",
			egrSVCBuilder.Definition.Name, egrSVCBuilder.Definition.Namespace)

		egrSVCBuilder = egrSVCBuilder.WithNodeLabelSelector(RDSCoreConfig.EgressServiceDeploy1NodeSelector)
	}

	if RDSCoreConfig.EgressServiceVRF1Network != "" {
		By("Setting VRF network for EgressService")

		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Configuring VRF network on EgressService %q in %q namespace",
			egrSVCBuilder.Definition.Name, egrSVCBuilder.Definition.Namespace)

		egrSVCBuilder = egrSVCBuilder.WithVRFNetwork("1001")
	}

	By("Creating EgressService")

	egrSVCBuilder, err = egrSVCBuilder.Create()

	Expect(err).ToNot(HaveOccurred(), "Failed to create EgressService")

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Created EgressService %q in %q namespace",
		egrSVCBuilder.Object.Name, egrSVCBuilder.Object.Namespace)

	By("Finding pod from app deployment")

	clientPods := findPodWithSelector(RDSCoreConfig.EgressServiceNS, egressSVC1Labels)

	Expect(clientPods).ToNot(BeNil(),
		fmt.Sprintf("Application pods matching %q label not found in %q namespace",
			RDSCoreConfig.EgressServiceNS, egressSVC1Labels))

	By("Getting status of service")

	Eventually(func() bool {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Check service %q in %q namespace has LoadBalancer IP",
			svcBuilder.Definition.Name, svcBuilder.Definition.Namespace)

		refreshSVC := svcBuilder.Exists()

		if !refreshSVC {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Failed to refresh service status")

			return false
		}

		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Service has %d IP addresses",
			len(svcBuilder.Object.Status.LoadBalancer.Ingress))

		return len(svcBuilder.Object.Status.LoadBalancer.Ingress) != 0
	}).WithContext(ctx).WithPolling(15*time.Second).WithTimeout(3*time.Minute).Should(BeTrue(),
		"Service does not have LoadBalancer IP address")

	loadBalancerIP := svcBuilder.Object.Status.LoadBalancer.Ingress[0].IP

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("LoadBalancer IP address: %q", loadBalancerIP)

	cmdToRun := []string{"/bin/bash", "-c",
		fmt.Sprintf("curl --connect-timeout 3 -Ls http://%s:%s/clientip",
			RDSCoreConfig.EgressServiceRemoteIP, RDSCoreConfig.EgressServiceRemotePort)}

	verifyPodSourceAddress(clientPods, cmdToRun, loadBalancerIP)
}

// VerifyEgressServiceWithLocalETP verifies EgressService with externalTrafficPolicy set to Local.
//
//nolint:funlen
func VerifyEgressServiceWithLocalETP(ctx SpecContext) {
	deleteDeployments(egressSVCDeploy2Name, RDSCoreConfig.EgressServiceNS)
	deleteService(egressSVC2Name, RDSCoreConfig.EgressServiceNS)
	deleteEgressService(egressSVC2Name, RDSCoreConfig.EgressServiceNS)

	waitForPodsGone(RDSCoreConfig.EgressServiceNS, egressSVC2Labels)

	const (
		servicePort       int32 = 9090
		serviceTargetPort int32 = 9090
	)

	podContainer := defineEgressSVCContainer(egressSVCContainer,
		RDSCoreConfig.EgressServiceDeploy2Image, RDSCoreConfig.EgressServiceDeploy2CMD)

	By("Reseting SecurityContext")

	podContainer = podContainer.WithSecurityContext(&corev1.SecurityContext{RunAsGroup: nil, RunAsUser: nil})

	cfgContainer, err := podContainer.GetContainerCfg()
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to get container definition: %v", err))

	deployLabels := map[string]string{
		strings.Split(egressSVC2Labels, "=")[0]: strings.Split(egressSVC2Labels, "=")[1],
	}

	deploy := defineEgressSVCDeployment(cfgContainer, egressSVCDeploy2Name,
		RDSCoreConfig.EgressServiceNS, deployLabels, RDSCoreConfig.EgressServiceDeploy2NodeSelector, int32(1))

	Expect(deploy).ToNot(BeNil(), "Failed to create deployment")

	deploy, err = deploy.CreateAndWaitUntilReady(5 * time.Minute)

	Expect(err).ToNot(HaveOccurred(), "Deployment hasn't reached Ready status")

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof(fmt.Sprintf("Deployment %q in %q namespace is Ready",
		deploy.Object.Name, deploy.Object.Namespace))

	By("Defining a ServicePort")

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Defining ServicePort object")

	svcPort, err := service.DefineServicePort(servicePort, serviceTargetPort, corev1.ProtocolTCP)

	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to define TargetPort: %v", err))

	By("Defining Service")

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Defining Service object")

	svcLabels := map[string]string{
		strings.Split(egressSVC2Labels, "=")[0]: strings.Split(egressSVC2Labels, "=")[1],
	}

	svcBuilder := defineService(egressSVC2Name, RDSCoreConfig.EgressServiceNS, svcLabels, *svcPort)

	Expect(svcBuilder).ToNot(BeNil(), "Failed to defined service")

	By("Setting ExternalTrafficPolicy to 'Local'")

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Setting Service's External Traffic Policy to 'Local'")

	svcBuilder.WithExternalTrafficPolicy(corev1.ServiceExternalTrafficPolicy(corev1.ServiceInternalTrafficPolicyLocal))

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Service Definition:\n%#v\n", svcBuilder.Definition)

	By("Setting AddressPool annotation")

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Setting MetalLB address-pool annotation")

	svcBuilder = svcBuilder.WithAnnotation(map[string]string{
		"metallb.universe.tf/address-pool": RDSCoreConfig.EgressServiceDeploy2IPAddrPool})

	By("Creating a service")
	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Creating Service object")

	svcBuilder, err = svcBuilder.Create()
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to create Service: %v", err))

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Created Service: %#v", svcBuilder.Object)

	By("Defining EgressService")

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Defining EgressService %q in %q namespace",
		egressSVC2Name, RDSCoreConfig.EgressServiceNS)

	egrSVCBuilder := egressservice.NewEgressServiceBuilder(APIClient, egressSVC2Name,
		RDSCoreConfig.EgressServiceNS, "LoadBalancerIP")

	if len(RDSCoreConfig.EgressServiceDeploy1NodeSelector) != 0 {
		By("Setting nodeSelector for EgressService")

		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Configuring nodeSelector on EgressService %q in %q namespace",
			egrSVCBuilder.Definition.Name, egrSVCBuilder.Definition.Namespace)

		egrSVCBuilder = egrSVCBuilder.WithNodeLabelSelector(RDSCoreConfig.EgressServiceDeploy1NodeSelector)
	}

	if RDSCoreConfig.EgressServiceVRF1Network != "" {
		By("Setting VRF network for EgressService")

		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Configuring VRF network on EgressService %q in %q namespace",
			egrSVCBuilder.Definition.Name, egrSVCBuilder.Definition.Namespace)

		egrSVCBuilder = egrSVCBuilder.WithVRFNetwork("1001")
	}

	By("Creating EgressService")

	egrSVCBuilder, err = egrSVCBuilder.Create()

	Expect(err).ToNot(HaveOccurred(), "Failed to create EgressService")

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Created EgressService %q in %q namespace",
		egrSVCBuilder.Object.Name, egrSVCBuilder.Object.Namespace)

	By("Getting status of service")

	Eventually(func() bool {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Check service %q in %q namespace has LoadBalancer IP",
			svcBuilder.Definition.Name, svcBuilder.Definition.Namespace)

		refreshSVC := svcBuilder.Exists()

		if !refreshSVC {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Failed to refresh service status")

			return false
		}

		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Service has %d IP addresses",
			len(svcBuilder.Object.Status.LoadBalancer.Ingress))

		return len(svcBuilder.Object.Status.LoadBalancer.Ingress) != 0
	}).WithContext(ctx).WithPolling(15*time.Second).WithTimeout(3*time.Minute).Should(BeTrue(),
		"Service does not have LoadBalancer IP address")

	By("Finding pod from app deployment")

	clientPods := findPodWithSelector(RDSCoreConfig.EgressServiceNS, egressSVC2Labels)

	Expect(clientPods).ToNot(BeNil(),
		fmt.Sprintf("Application pods matching %q label not found in %q namespace",
			RDSCoreConfig.EgressServiceNS, egressSVC2Labels))

	cmdToRun := []string{"/bin/bash", "-c",
		fmt.Sprintf("curl --connect-timeout 3 -Ls http://%s:%s/clientip",
			RDSCoreConfig.EgressServiceRemoteIP, RDSCoreConfig.EgressServiceRemotePort)}

	loadBalancerIP := svcBuilder.Object.Status.LoadBalancer.Ingress[0].IP

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("LoadBalancer IP address: %q", loadBalancerIP)

	verifyPodSourceAddress(clientPods, cmdToRun, loadBalancerIP)

	By(fmt.Sprintf("Accessing workload via LoadBalancer's IP %s", loadBalancerIP))

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Accessing  workload via LoadBalancer's IP %s", loadBalancerIP)

	verifyIngressIP(loadBalancerIP, RDSCoreConfig.EgressServiceRemoteIP, servicePort)
}

func verifySourceIP(svcName, svcNS, podLabels string, cmdToRun []string) {
	By(fmt.Sprintf("Pulling %q service configuration", svcName))

	var (
		svcBuilder *service.Builder
		err        error
		ctx        SpecContext
	)

	Eventually(func() bool {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Pulling %q service from %q namespace",
			svcName, svcNS)

		svcBuilder, err = service.Pull(APIClient, svcName, svcNS)

		if err != nil {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Error pulling %q service from %q namespace: %v",
				svcName, svcNS, err)

			return false
		}

		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Successfully pulled %q service from %q namespace",
			svcBuilder.Definition.Name, svcBuilder.Definition.Namespace)

		return true
	}).WithContext(ctx).WithPolling(5*time.Second).WithTimeout(1*time.Minute).Should(BeTrue(),
		fmt.Sprintf("Error obtaining service %q configuration", svcName))

	By(fmt.Sprintf("Asserting service %q has LoadBalancer IP address", svcName))

	Eventually(func() bool {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Check service %q in %q namespace has LoadBalancer IP",
			svcBuilder.Definition.Name, svcBuilder.Definition.Namespace)

		refreshSVC := svcBuilder.Exists()

		if !refreshSVC {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Failed to refresh service status")

			return false
		}

		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Service has %d IP addresses",
			len(svcBuilder.Object.Status.LoadBalancer.Ingress))

		return len(svcBuilder.Object.Status.LoadBalancer.Ingress) != 0
	}).WithContext(ctx).WithPolling(15*time.Second).WithTimeout(3*time.Minute).Should(BeTrue(),
		"Service does not have LoadBalancer IP address")

	loadBalancerIP := svcBuilder.Object.Status.LoadBalancer.Ingress[0].IP

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("LoadBalancer IP address: %q", loadBalancerIP)

	By("Finding pod from app deployment")

	clientPods := findPodWithSelector(svcNS, podLabels)

	Expect(clientPods).ToNot(BeNil(),
		fmt.Sprintf("Application pods matching %q label not found in %q namespace",
			svcName, svcNS))

	verifyPodSourceAddress(clientPods, cmdToRun, loadBalancerIP)
}

// VerifyEgressServiceConnectivityETPCluster verifies source IP address when external traffic policy
// is set to Cluster.
func VerifyEgressServiceConnectivityETPCluster() {
	cmdToRun := []string{"/bin/bash", "-c",
		fmt.Sprintf("curl --connect-timeout 3 -Ls http://%s:%s/clientip",
			RDSCoreConfig.EgressServiceRemoteIP, RDSCoreConfig.EgressServiceRemotePort)}

	verifySourceIP(egressSVC1Name, RDSCoreConfig.EgressServiceNS, egressSVC1Labels, cmdToRun)
}

// VerifyEgressServiceConnectivityETPLocal verifies source IP address when external traffic policy
// is set to Local.
func VerifyEgressServiceConnectivityETPLocal() {
	cmdToRun := []string{"/bin/bash", "-c",
		fmt.Sprintf("curl --connect-timeout 3 -Ls http://%s:%s/clientip",
			RDSCoreConfig.EgressServiceRemoteIP, RDSCoreConfig.EgressServiceRemotePort)}

	verifySourceIP(egressSVC2Name, RDSCoreConfig.EgressServiceNS, egressSVC2Labels, cmdToRun)
}

// VerifyEgressServiceIngressConnectivity verifies ingress IP address while accessing backend pods
// via loadbalancer.
func VerifyEgressServiceIngressConnectivity() {
	By(fmt.Sprintf("Pulling %q service configuration", egressSVC2Name))

	var (
		svcBuilder *service.Builder
		err        error
		ctx        SpecContext
	)

	Eventually(func() bool {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Pulling %q service from %q namespace",
			egressSVC2Name, RDSCoreConfig.EgressServiceNS)

		svcBuilder, err = service.Pull(APIClient, egressSVC2Name, RDSCoreConfig.EgressServiceNS)

		if err != nil {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Error pulling %q service from %q namespace: %v",
				egressSVC2Name, RDSCoreConfig.EgressServiceNS, err)

			return false
		}

		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Successfully pulled %q service from %q namespace",
			svcBuilder.Definition.Name, svcBuilder.Definition.Namespace)

		return true
	}).WithContext(ctx).WithPolling(5*time.Second).WithTimeout(1*time.Minute).Should(BeTrue(),
		fmt.Sprintf("Error obtaining service %q configuration", egressSVC2Name))

	By(fmt.Sprintf("Asserting service %q has LoadBalancer IP address", svcBuilder.Definition.Name))

	Eventually(func() bool {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Check service %q in %q namespace has LoadBalancer IP",
			svcBuilder.Definition.Name, svcBuilder.Definition.Namespace)

		refreshSVC := svcBuilder.Exists()

		if !refreshSVC {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Failed to refresh service status")

			return false
		}

		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Service has %d IP addresses",
			len(svcBuilder.Object.Status.LoadBalancer.Ingress))

		return len(svcBuilder.Object.Status.LoadBalancer.Ingress) != 0
	}).WithContext(ctx).WithPolling(15*time.Second).WithTimeout(3*time.Minute).Should(BeTrue(),
		"Service does not have LoadBalancer IP address")

	loadBalancerIP := svcBuilder.Object.Status.LoadBalancer.Ingress[0].IP

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Accessing  workload via LoadBalancer's IP %s", loadBalancerIP)

	verifyIngressIP(loadBalancerIP, RDSCoreConfig.EgressServiceRemoteIP, servicePort)
}

func verifyIngressIP(loadBalancerIP, expectedIP string, servicePort int32) {
	var (
		cmdResult []byte
		err       error
		ctx       SpecContext
	)

	By(fmt.Sprintf("Accessing backend pods via %s IP", loadBalancerIP))

	Eventually(func() bool {
		cmdExternal := exec.Command("curl", "--connect-timeout", "3", "-s",
			fmt.Sprintf("http://%s:%d/clientip", loadBalancerIP, servicePort))

		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Running command: %q", cmdExternal.String())

		cmdResult, err = cmdExternal.CombinedOutput()

		if err != nil {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Error running command: %v", err)

			return false
		}

		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Successfully executed command: %q",
			cmdExternal.String())

		return true
	}).WithContext(ctx).WithPolling(5*time.Second).WithTimeout(1*time.Minute).Should(
		BeTrue(), "Failed to executed command")

	By("Parsing command's output")

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Parsing command's output:\n\t%v(%v)\n",
		string(cmdResult), err)

	addr, _, err := net.SplitHostPort(string(cmdResult))

	By("Comparing ingress IP address")

	Expect(err).ToNot(HaveOccurred(), "Failed to parse Host/Port pairs from command's output")

	Expect(addr).To(BeEquivalentTo(expectedIP), "Wrong IP address used")
}
