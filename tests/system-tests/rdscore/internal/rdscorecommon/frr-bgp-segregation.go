package rdscorecommon

import (
	"context"
	"fmt"
	"math/rand"
	"net/netip"
	"strings"
	"time"

	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/remote"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/supporttools"

	. "github.com/openshift-kni/eco-gotests/tests/system-tests/rdscore/internal/rdscoreinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/rdscore/internal/rdscoreparams"
)

const (
	lbDeploymentLabel     = "systemtest-test=rds-core-metal-app"
	stNamespace           = "rds-metallb-supporttools-ns"
	stDeploymentLabel     = "rds-core=supporttools-deploy"
	packetCaptureProtocol = "http"
	codesPattern          = "200 404"
)

var (
	bgpOneAppURLIPv4 = fmt.Sprintf("http://%s:%s",
		RDSCoreConfig.MetalLBLoadBalancerOneIPv4, RDSCoreConfig.MetalLBTrafficSegregationTargetPort)
	bgpOneAppURLIPv6 = fmt.Sprintf("http://[%s]:%s",
		RDSCoreConfig.MetalLBLoadBalancerOneIPv6, RDSCoreConfig.MetalLBTrafficSegregationTargetPort)
	bgpTwoAppURLIPv4 = fmt.Sprintf("http://%s:%s",
		RDSCoreConfig.MetalLBLoadBalancerTwoIPv4, RDSCoreConfig.MetalLBTrafficSegregationTargetPort)
	bgpTwoAppURLIPv6 = fmt.Sprintf("http://[%s]:%s",
		RDSCoreConfig.MetalLBLoadBalancerTwoIPv6, RDSCoreConfig.MetalLBTrafficSegregationTargetPort)
)

// VerifyMetallbEgressTrafficSegregation test metallb egress traffic segregation.
//
//nolint:funlen
func VerifyMetallbEgressTrafficSegregation(ctx SpecContext) {
	By("Asserting if test URLs are provided")

	if RDSCoreConfig.MetalLBTrafficSegregationTargetOneIPv4 == "" &&
		RDSCoreConfig.MetalLBTrafficSegregationTargetOneIPv6 == "" {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof(
			"Test URLs for MetalLB FRR testing not specified or are empty. Skipping...")
		Skip("Test URL for MetalLB FRR testing not specified or are empty")
	}

	if RDSCoreConfig.MetalLBTrafficSegregationTargetTwoIPv4 == "" &&
		RDSCoreConfig.MetalLBTrafficSegregationTargetTwoIPv6 == "" {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof(
			"Test URLs for the second MetalLB FRR testing not specified or are empty. Skipping...")
		Skip("Test URL for the second MetalLB FRR testing not specified or are empty")
	}

	By("Randomly choosing node for the support-tools deployment")

	segregationTestNode, err := getNodeForTest()
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to retrieve testing node object: %v", err))

	segregationTestNodeName := segregationTestNode.Definition.Name

	By("Creating the packet support-tools deployment")

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Create support-tools deployment on the node %s",
		segregationTestNodeName)

	_, err =
		supporttools.CreateTraceRouteDeployment(
			APIClient,
			stNamespace,
			stDeploymentLabel,
			RDSCoreConfig.MetalLBSupportToolsImage,
			[]string{segregationTestNodeName})
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to create supporttools deployment for nodes %v in namespace %s from image %s: %v",
			segregationTestNode, stNamespace, RDSCoreConfig.MetalLBSupportToolsImage, err))

	By("Finding support-tools pods")

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Searching for pods in %q namespace", stNamespace)

	stPodsList := findPodWithSelector(stNamespace, stDeploymentLabel)
	Expect(len(stPodsList)).ToNot(Equal(0), "No traceRoute FRR pods found in namespace %s",
		stNamespace)

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Found %d 'traceRoute' pods", len(stPodsList))

	for _, _pod := range stPodsList {
		By("Verify the correct route was used for the main MetalLB-FRR")

		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Verifying that the correct route was used "+
			"for the main FRR defined %s/%s",
			RDSCoreConfig.MetalLBFRROneIPv4, RDSCoreConfig.MetalLBFRROneIPv6)

		for _, targetHostIP := range []string{RDSCoreConfig.MetalLBTrafficSegregationTargetOneIPv4,
			RDSCoreConfig.MetalLBTrafficSegregationTargetOneIPv6} {
			if targetHostIP == "" {
				glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Empty target IP continue")

				continue
			}

			myIP, err := netip.ParseAddr(targetHostIP)
			Expect(err).ToNot(HaveOccurred(),
				fmt.Sprintf("Failed to parse host ip %s", targetHostIP))

			searchString := RDSCoreConfig.MetalLBFRROneIPv4

			if myIP.Is6() {
				searchString = RDSCoreConfig.MetalLBFRROneIPv6
			}

			err = supporttools.SendTrafficFindExpectedString(
				_pod,
				targetHostIP,
				RDSCoreConfig.MetalLBTrafficSegregationTargetPort,
				searchString)
			Expect(err).ToNot(HaveOccurred(),
				fmt.Sprintf("Failed to find required FRR IP address %s in response: %v", searchString, err))
		}

		By("Verify the correct route was used for the secondary MetalLB-FRR")

		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Verifying that the correct route was used "+
			"for the second FRR defined %s/%s", RDSCoreConfig.MetalLBFRRTwoIPv4,
			RDSCoreConfig.MetalLBFRRTwoIPv6)

		for _, targetHostIP := range []string{RDSCoreConfig.MetalLBTrafficSegregationTargetTwoIPv4,
			RDSCoreConfig.MetalLBTrafficSegregationTargetTwoIPv6} {
			if targetHostIP == "" {
				glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Empty target IP continue")

				continue
			}

			myIP, err := netip.ParseAddr(targetHostIP)
			Expect(err).ToNot(HaveOccurred(),
				fmt.Sprintf("Failed to parse host ip %s", targetHostIP))

			searchString := RDSCoreConfig.MetalLBFRRTwoIPv4

			if myIP.Is6() {
				searchString = RDSCoreConfig.MetalLBFRRTwoIPv6
			}

			err = supporttools.SendTrafficFindExpectedString(
				_pod,
				targetHostIP,
				RDSCoreConfig.MetalLBTrafficSegregationTargetPort,
				searchString)
			Expect(err).ToNot(HaveOccurred(),
				fmt.Sprintf("Failed to find required FRR IP address %s in response: %v", searchString, err))
		}
	}
}

// VerifyMetallbIngressTrafficSegregation test ingress connectivity to the workloads running on the cluster exposed
// via service(of type loadbalancer) with traffic segregation.
//
//nolint:funlen
func VerifyMetallbIngressTrafficSegregation(ctx SpecContext) {
	By("Asserting if test URLs are provided")

	var err error

	if RDSCoreConfig.MetalLBFRROneIPv4 == "" && RDSCoreConfig.MetalLBFRROneIPv6 == "" {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof(
			"First MetalLB FRR IP not specified or are empty. Skipping...")
		Skip("First MetalLB FRR IP not specified or are empty")
	}

	if RDSCoreConfig.MetalLBFRRTwoIPv4 == "" && RDSCoreConfig.MetalLBFRRTwoIPv6 == "" {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof(
			"Second MetalLB FRR IP not specified or are empty. Skipping...")
		Skip("Second MetalLB FRR IP not specified or are empty")
	}

	if RDSCoreConfig.MetalLBTrafficSegregationTCPDumpIntOne == "" &&
		RDSCoreConfig.MetalLBTrafficSegregationTCPDumpIntTwo == "" {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof(
			"One of the capture interfaces not specified or are empty. Skipping...")
		Skip("One of the capture interfaces not specified or are empty")
	}

	By("Ensure LB application deployment created for the first bgp route")

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Searching for pods in %q namespace",
		RDSCoreConfig.MetalLBLoadBalancerOneNamespace)

	lbOnePodsList := findPodWithSelector(RDSCoreConfig.MetalLBLoadBalancerOneNamespace, lbDeploymentLabel)

	var lbOneNodeName string

	if len(lbOnePodsList) == 0 {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof(
			"FRR One Load Balance app not found deployed. Skipping...")
		Skip("FRR One Load Balance app not found deployed")
	}

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Getting node name for the loadbalancer one application")

	lbOneNodeName = lbOnePodsList[0].Object.Spec.NodeName

	By("Ensure LB application deployment created for the second bgp route")

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Searching for pods in %q namespace",
		RDSCoreConfig.MetalLBLoadBalancerTwoNamespace)

	lbTwoPodsList := findPodWithSelector(RDSCoreConfig.MetalLBLoadBalancerTwoNamespace, lbDeploymentLabel)

	var lbTwoNodeName string

	if len(lbTwoPodsList) == 0 {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof(
			"FRR Two Load Balance app not found deployed. Skipping...")
		Skip("FRR Two Load Balance app not found deployed")
	}

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Getting node name for the loadbalancer two application")

	lbTwoNodeName = lbTwoPodsList[0].Object.Spec.NodeName

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("debug for the node selector: %s", lbTwoNodeName)

	By("Make sure that 1st external FRR container has learnt route to the 1st deployment")

	ipRouteList := []string{
		fmt.Sprintf("%s/32", RDSCoreConfig.MetalLBLoadBalancerOneIPv4),
		fmt.Sprintf("%s/128", RDSCoreConfig.MetalLBLoadBalancerOneIPv6),
	}

	for _, ipRoute := range ipRouteList {
		err := verifyIPRouteBGP(
			RDSCoreConfig.HypervisorHost,
			RDSCoreConfig.HypervisorUser,
			RDSCoreConfig.HypervisorPass,
			RDSCoreConfig.MetalLBFRRContainerNameOne,
			ipRoute)
		Expect(err).ToNot(HaveOccurred(),
			fmt.Sprintf("first external FRR container %s route check for the %s failed: %v",
				RDSCoreConfig.MetalLBFRRContainerNameOne, ipRoute, err))
	}

	By("Make sure that 2nd external FRR container has learnt route to the 2nd deployment")

	ipRouteList = []string{
		fmt.Sprintf("%s/32", RDSCoreConfig.MetalLBLoadBalancerTwoIPv4),
		fmt.Sprintf("%s/128", RDSCoreConfig.MetalLBLoadBalancerTwoIPv6),
	}

	for _, ipRoute := range ipRouteList {
		err := verifyIPRouteBGP(
			RDSCoreConfig.HypervisorHost,
			RDSCoreConfig.HypervisorUser,
			RDSCoreConfig.HypervisorPass,
			RDSCoreConfig.MetalLBFRRContainerNameTwo,
			ipRoute)
		Expect(err).ToNot(HaveOccurred(),
			fmt.Sprintf("second external FRR container %s route check for the %s failed: %v",
				RDSCoreConfig.MetalLBFRRContainerNameTwo, ipRoute, err))
	}

	By("Creating the tcpdump deployment for the first FRR packets capturing")

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Create tcpdump deployment for the node %v", lbOneNodeName)

	_, err =
		supporttools.CreateTCPDumpDeployment(
			APIClient,
			stNamespace,
			stDeploymentLabel,
			RDSCoreConfig.MetalLBSupportToolsImage,
			RDSCoreConfig.MetalLBTrafficSegregationTCPDumpIntOne,
			packetCaptureProtocol,
			[]string{lbOneNodeName})
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to create supporttools deployment for the node %v in namespace %s from image %s: %v",
			lbOneNodeName, stNamespace, RDSCoreConfig.MetalLBSupportToolsImage, err))

	By("Finding support-tools pods")

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Searching for tcpdump pods in %q namespace", stNamespace)

	stPodsList := findPodWithSelector(stNamespace, stDeploymentLabel)
	Expect(len(stPodsList)).ToNot(Equal(0), "No tcpdump pods found in namespace %s",
		stNamespace)

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Found %d 'traceRoute' pods", len(stPodsList))

	By("Assert that 1st mockup app is reachable from the node with 1st FRR container")

	timeStart := time.Now()

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("start run traffic at %v", timeStart)

	for _, appURL := range []string{bgpOneAppURLIPv4, bgpOneAppURLIPv6} {
		err := verifyAppIsReachableFromFRRContainer(
			RDSCoreConfig.HypervisorHost,
			RDSCoreConfig.HypervisorUser,
			RDSCoreConfig.HypervisorPass,
			RDSCoreConfig.MetalLBFRRContainerNameOne,
			appURL)
		Expect(err).ToNot(HaveOccurred(),
			fmt.Sprintf("first mockup app %s is not reachable from the container %s: %v",
				appURL, RDSCoreConfig.MetalLBFRRContainerNameOne, err))
	}

	By("Check the traffic flows for the first FRR through the expected interface")

	for _, _pod := range stPodsList {
		searchString := "10.18.126.21"

		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Making sure that the traffic flows for the first FRR "+
			"through the %s interface", RDSCoreConfig.MetalLBTrafficSegregationTCPDumpIntOne)

		numberFound, logs, err := supporttools.ScanTCPDumpPodLogs(
			APIClient,
			_pod,
			packetCaptureProtocol,
			stNamespace,
			stDeploymentLabel,
			searchString,
			timeStart)
		Expect(err).ToNot(HaveOccurred(),
			fmt.Sprintf("failed to tcpdump log results for the pod %s in namespace %s: %v",
				_pod.Definition.Name, _pod.Definition.Namespace, err))

		if numberFound == 0 {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("The searching string %s was not found in log: %s",
				searchString, logs)
		}

		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("The searching string %s was found in log: %s",
			searchString, logs)
	}

	By("Creating the tcpdump deployment for the second FRR packets capturing")

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Create tcpdump deployment for the node %v", lbTwoNodeName)

	_, err =
		supporttools.CreateTCPDumpDeployment(
			APIClient,
			stNamespace,
			stDeploymentLabel,
			RDSCoreConfig.MetalLBSupportToolsImage,
			RDSCoreConfig.MetalLBTrafficSegregationTCPDumpIntTwo,
			packetCaptureProtocol,
			[]string{lbTwoNodeName})
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to create supporttools deployment for the node %v in namespace %s from image %s: %v",
			lbTwoNodeName, stNamespace, RDSCoreConfig.MetalLBSupportToolsImage, err))

	By("Finding support-tools pods")

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Searching for tcpdump pods in %q namespace", stNamespace)

	stPodsList = findPodWithSelector(stNamespace, stDeploymentLabel)
	Expect(len(stPodsList)).ToNot(Equal(0), "No tcpdump pods found in namespace %s",
		stNamespace)

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Found %d 'traceRoute' pods", len(stPodsList))

	timeStart = time.Now()

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("start run traffic at %v", timeStart)

	By("Assert that 2nd mockup app is reachable from the node with 2nd FRR container")

	for _, appURL := range []string{bgpTwoAppURLIPv4, bgpTwoAppURLIPv6} {
		err := verifyAppIsReachableFromFRRContainer(
			RDSCoreConfig.HypervisorHost,
			RDSCoreConfig.HypervisorUser,
			RDSCoreConfig.HypervisorPass,
			RDSCoreConfig.MetalLBFRRContainerNameTwo,
			appURL)
		Expect(err).ToNot(HaveOccurred(),
			fmt.Sprintf("second mockup app %s is not reachable from the container %s: %v",
				appURL, RDSCoreConfig.MetalLBFRRContainerNameTwo, err))

		By("Check the traffic flows for the second FRR through the expected interface")

		for _, _pod := range stPodsList {
			searchString := "10.18.124.21"

			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Making sure that the traffic flows for the second FRR "+
				"through the %s interface", RDSCoreConfig.MetalLBTrafficSegregationTCPDumpIntTwo)

			numberFound, logs, err := supporttools.ScanTCPDumpPodLogs(
				APIClient,
				_pod,
				packetCaptureProtocol,
				stNamespace,
				stDeploymentLabel,
				searchString,
				timeStart)
			Expect(err).ToNot(HaveOccurred(),
				fmt.Sprintf("failed to tcpdump log results for the pod %s in namespace %s: %v",
					_pod.Definition.Name, _pod.Definition.Namespace, err))

			if numberFound == 0 {
				glog.V(rdscoreparams.RDSCoreLogLevel).Infof("The searching string %s was not found in log: %s",
					searchString, logs)
			}

			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("The searching string %s was found in log: %s",
				searchString, logs)
		}
	}
}

// VerifyMetallbMockupAppNotReachableFromOtherFRR test mockup app is not reachable from the not correct FRR container.
//
//nolint:funlen
func VerifyMetallbMockupAppNotReachableFromOtherFRR(ctx SpecContext) {
	By("Asserting if test URLs are provided")

	if RDSCoreConfig.MetalLBFRROneIPv4 == "" && RDSCoreConfig.MetalLBFRROneIPv6 == "" {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof(
			"First MetalLB FRR IP not specified or are empty. Skipping...")
		Skip("First MetalLB FRR IP not specified or are empty")
	}

	if RDSCoreConfig.MetalLBFRRTwoIPv4 == "" && RDSCoreConfig.MetalLBFRRTwoIPv6 == "" {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof(
			"Second MetalLB FRR IP not specified or are empty. Skipping...")
		Skip("Second MetalLB FRR IP not specified or are empty")
	}

	if RDSCoreConfig.MetalLBTrafficSegregationTCPDumpIntOne == "" &&
		RDSCoreConfig.MetalLBTrafficSegregationTCPDumpIntTwo == "" {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof(
			"One of the capture interfaces not specified or are empty. Skipping...")
		Skip("One of the capture interfaces not specified or are empty")
	}

	By("Ensure LB application deployment created for the first bgp route")

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Searching for pods in %q namespace",
		RDSCoreConfig.MetalLBLoadBalancerOneNamespace)

	lbOnePodsList := findPodWithSelector(RDSCoreConfig.MetalLBLoadBalancerOneNamespace, lbDeploymentLabel)

	if len(lbOnePodsList) == 0 {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof(
			"FRR One Load Balance app not found deployed. Skipping...")
		Skip("FRR One Load Balance app not found deployed")
	}

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Getting node name for the loadbalancer one application")

	By("Ensure LB application deployment created for the second bgp route")

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Searching for pods in %q namespace",
		RDSCoreConfig.MetalLBLoadBalancerTwoNamespace)

	lbTwoPodsList := findPodWithSelector(RDSCoreConfig.MetalLBLoadBalancerTwoNamespace, lbDeploymentLabel)

	var lbTwoNodeName string

	if len(lbTwoPodsList) == 0 {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof(
			"FRR Two Load Balance app not found deployed. Skipping...")
		Skip("FRR Two Load Balance app not found deployed")
	}

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Getting node name for the loadbalancer two application")

	lbTwoNodeName = lbTwoPodsList[0].Object.Spec.NodeName

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("debug for the node selector: %s", lbTwoNodeName)

	By("Make sure that 1st external FRR container has learnt route to the 1st deployment")

	ipRouteList := []string{
		fmt.Sprintf("%s/32", RDSCoreConfig.MetalLBLoadBalancerOneIPv4),
		fmt.Sprintf("%s/128", RDSCoreConfig.MetalLBLoadBalancerOneIPv6),
	}

	for _, ipRoute := range ipRouteList {
		err := verifyIPRouteBGP(
			RDSCoreConfig.HypervisorHost,
			RDSCoreConfig.HypervisorUser,
			RDSCoreConfig.HypervisorPass,
			RDSCoreConfig.MetalLBFRRContainerNameOne,
			ipRoute)
		Expect(err).ToNot(HaveOccurred(),
			fmt.Sprintf("first external FRR container %s route check for the %s failed: %v",
				RDSCoreConfig.MetalLBFRRContainerNameOne, ipRoute, err))
	}

	By("Make sure that 2nd external FRR container has learnt route to the 2nd deployment")

	ipRouteList = []string{
		fmt.Sprintf("%s/32", RDSCoreConfig.MetalLBLoadBalancerTwoIPv4),
		fmt.Sprintf("%s/128", RDSCoreConfig.MetalLBLoadBalancerTwoIPv6),
	}

	for _, ipRoute := range ipRouteList {
		err := verifyIPRouteBGP(
			RDSCoreConfig.HypervisorHost,
			RDSCoreConfig.HypervisorUser,
			RDSCoreConfig.HypervisorPass,
			RDSCoreConfig.MetalLBFRRContainerNameTwo,
			ipRoute)
		Expect(err).ToNot(HaveOccurred(),
			fmt.Sprintf("second external FRR container %s route check for the %s failed: %v",
				RDSCoreConfig.MetalLBFRRContainerNameTwo, ipRoute, err))
	}

	By("Assert that 1st mockup app is not reachable from the node with 2st FRR container")

	timeStart := time.Now()

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("start run traffic at %v", timeStart)

	for _, appURL := range []string{bgpOneAppURLIPv4, bgpOneAppURLIPv6} {
		err := verifyAppIsReachableFromFRRContainer(
			RDSCoreConfig.HypervisorHost,
			RDSCoreConfig.HypervisorUser,
			RDSCoreConfig.HypervisorPass,
			RDSCoreConfig.MetalLBFRRContainerNameTwo,
			appURL)
		Expect(err).To(HaveOccurred(),
			fmt.Sprintf("first mockup app %s is reachable from the container %s: %v",
				appURL, RDSCoreConfig.MetalLBFRRContainerNameTwo, err))
	}

	By("Assert that 2nd mockup app is reachable from the node with 2nd FRR container")

	for _, appURL := range []string{bgpTwoAppURLIPv4, bgpTwoAppURLIPv6} {
		err := verifyAppIsReachableFromFRRContainer(
			RDSCoreConfig.HypervisorHost,
			RDSCoreConfig.HypervisorUser,
			RDSCoreConfig.HypervisorPass,
			RDSCoreConfig.MetalLBFRRContainerNameOne,
			appURL)
		Expect(err).To(HaveOccurred(),
			fmt.Sprintf("second mockup app %s is not reachable from the container %s: %v",
				appURL, RDSCoreConfig.MetalLBFRRContainerNameOne, err))
	}
}

func verifyIPRouteBGP(host, user, pass, containerName, lbIP string) error {
	var result string

	var err error

	glog.V(100).Infof("Verify IP version type")

	parsedIP := strings.Split(lbIP, "/")[0]

	glog.V(100).Infof("IP: %s", parsedIP)

	myIP, err := netip.ParseAddr(parsedIP)

	if err != nil {
		glog.V(100).Infof("Failed to parse provided loadbalancer ip address %q; %v", parsedIP, err)

		return fmt.Errorf("failed to parse provided loadbalancer ip address %q; %w", parsedIP, err)
	}

	showIPRouteBGPOnFRRContainerCmd := fmt.Sprintf("sudo podman exec -it %s bash "+
		"-c 'vtysh -c \"show ip route bgp\"'", containerName)

	if myIP.Is6() {
		showIPRouteBGPOnFRRContainerCmd = fmt.Sprintf("sudo podman exec -it %s bash "+
			"-c 'vtysh -c \"show ipv6 route bgp\"'", containerName)
	}

	glog.V(100).Infof("Running command %q from within container %s",
		showIPRouteBGPOnFRRContainerCmd, containerName)

	err = wait.PollUntilContextTimeout(
		context.TODO(),
		time.Second,
		time.Second*30,
		true,
		func(ctx context.Context) (bool, error) {
			result, err = remote.ExecCmdOnHost(host, user, pass, showIPRouteBGPOnFRRContainerCmd)

			if err != nil {
				glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Failed to run command due to: %v", err)

				return false, nil
			}

			glog.V(100).Infof(fmt.Sprintf("the IP routes %s were found on the FRR container %s: %s",
				lbIP, containerName, result))

			return true, nil
		})

	if err != nil {
		glog.V(100).Infof("Failed to verify IP route bgp on the FRR container %s: %v", containerName, err)

		return fmt.Errorf("failed to verify IP route bgp on the FRR container %s: %w", containerName, err)
	}

	if !strings.Contains(result, lbIP) {
		glog.V(100).Infof("No IP route bgp %s on the FRR container %s was found", lbIP, containerName)

		return fmt.Errorf("no IP route bgp %s on the FRR container %s was found", lbIP, containerName)
	}

	return nil
}

//nolint:funlen
func verifyAppIsReachableFromFRRContainer(host, user, pass, containerName, appURL string) error {
	var netNs, output string

	var err error

	getContainerNetNsCmd :=
		fmt.Sprintf("sudo podman container inspect --format \"{{.NetworkSettings.SandboxKey}}\" %s | xargs basename",
			containerName)

	glog.V(100).Infof("Running command %q from within container %s", getContainerNetNsCmd, containerName)

	err = wait.PollUntilContextTimeout(
		context.TODO(),
		time.Second,
		time.Second*3,
		true,
		func(ctx context.Context) (bool, error) {
			netNs, err = remote.ExecCmdOnHost(host, user, pass, getContainerNetNsCmd)

			if err != nil {
				glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Failed to run command due to: %v", err)

				return false, nil
			}

			netNs = strings.TrimSpace(netNs)
			netNs = strings.Trim(netNs, "\n")

			glog.V(100).Infof(fmt.Sprintf("netns for the container %s successfully retrieved: %s",
				containerName, netNs))

			return true, nil
		})

	if err != nil {
		glog.V(100).Infof("Failed to retrieve netns for the FRR container %s: %v", containerName, err)

		return fmt.Errorf("failed to retrieve netns for the FRR container %s: %w", containerName, err)
	}

	if netNs == "" {
		glog.V(100).Infof("netns for the FRR container %s not found", containerName)

		return fmt.Errorf("netns for the FRR container %s not found", containerName)
	}

	verifyAppIsReachableCmd := fmt.Sprintf("sudo ip netns exec %s curl -Ls --max-time 5 -o /dev/null "+
		"-w '%%{http_code}' %s", netNs, appURL)

	glog.V(100).Infof("Running command %q from within netns %s of the container %s",
		verifyAppIsReachableCmd, netNs, containerName)

	err = wait.PollUntilContextTimeout(
		context.TODO(),
		time.Second,
		time.Second*3,
		true,
		func(ctx context.Context) (bool, error) {
			for i := 0; i < 3; i++ {
				output, err = remote.ExecCmdOnHost(host, user, pass, verifyAppIsReachableCmd)

				if err != nil {
					glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Failed to run command due to: %v", err)

					return false, nil
				}

				glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Command's Output:\n%v\n", output)
			}

			return true, nil
		})

	if err != nil {
		glog.V(100).Infof("URL %s is not reachable from the netns %s of the container %s; %v",
			appURL, netNs, containerName, err)

		return fmt.Errorf("the URL %s is not reachable from the netns %s of the container %s; %w",
			appURL, netNs, containerName, err)
	}

	if !strings.Contains(codesPattern, strings.Trim(output, "'")) {
		glog.V(100).Infof("received HTTP response not as expected; expected: %s, received: %s",
			codesPattern, output)

		return fmt.Errorf("received HTTP response not as expected; expected: %s, received: %s",
			codesPattern, output)
	}

	glog.V(100).Infof(fmt.Sprintf("URL %s is reachable from the netns %s of the container %s",
		appURL, netNs, containerName))

	return nil
}

func getNodeForTest() (*nodes.Builder, error) {
	glog.V(rdscoreparams.RDSCoreLogLevel).Infof(
		"Find available nodes list for the LoadBalancer application deployment")

	nodesList, err := nodes.List(APIClient, RDSCoreConfig.WorkerLabelListOption)

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve nodes list due to %w", err)
	}

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Choose one random node for deployment")

	nodeIndex := rand.Intn(len(nodesList) - 1)
	randomNode := nodesList[nodeIndex]

	return randomNode, nil
}
