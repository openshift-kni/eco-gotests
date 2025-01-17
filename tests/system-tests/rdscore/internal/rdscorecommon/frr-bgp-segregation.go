package rdscorecommon

import (
	"fmt"
	"net/netip"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/traceroute"

	. "github.com/openshift-kni/eco-gotests/tests/system-tests/rdscore/internal/rdscoreinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/rdscore/internal/rdscoreparams"
)

const (
	trNamespace       = "rds-metallb-traceroute-ns"
	trDeploymentLabel = "rds-core=traceroute-deploy"
)

// ReachURLviaSecondFRRroute test URL via second route learned by MetalLB FRR.
func ReachURLviaSecondFRRroute(ctx SpecContext) {
	reachURLviaFRRroute(ctx, RDSCoreConfig.MetalLBSecondFRRTestURLIPv4, RDSCoreConfig.MetalLBSecondFRRTestURLIPv6)
}

// VerifyMetallbTrafficSegregation test metallb traffic segregation.
//
//nolint:funlen
func VerifyMetallbTrafficSegregation(ctx SpecContext) {
	By("Asserting if test URLs are provided")

	if RDSCoreConfig.MetalLBFirstFRRTargetIPv4 == "" && RDSCoreConfig.MetalLBFirstFRRTargetIPv6 == "" {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof(
			"Test URLs for MetalLB FRR testing not specified or are empty. Skipping...")
		Skip("Test URL for MetalLB FRR testing not specified or are empty")
	}

	if RDSCoreConfig.MetalLBSecondFRRTargetIPv4 == "" && RDSCoreConfig.MetalLBSecondFRRTargetIPv6 == "" {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof(
			"Test URLs for the second MetalLB FRR testing not specified or are empty. Skipping...")
		Skip("Test URL for the second MetalLB FRR testing not specified or are empty")
	}

	By("Finding MetalLB-FRR pods")

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Searching for pods in %q namespace with %q label",
		rdscoreparams.MetalLBOperatorNamespace, rdscoreparams.MetalLBFRRPodSelector)

	frrPodList := findPodWithSelector(rdscoreparams.MetalLBOperatorNamespace,
		rdscoreparams.MetalLBFRRPodSelector)
	Expect(len(frrPodList)).ToNot(Equal(0), "No MetalLB FRR pods found")

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Found %d 'frr' pods", len(frrPodList))

	By("Creating the packet traceroute deployment")

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Create traceroute deployment nodes list based on the found "+
		"frr pods list: %v", frrPodList)

	var snifferNodesList []string
	for _, frrPod := range frrPodList {
		snifferNodesList = append(snifferNodesList, frrPod.Object.Spec.NodeName)
	}

	Expect(len(snifferNodesList)).ToNot(Equal(0), "Failed to build traceroute nodes list based "+
		"on the frr pods list %v", frrPodList)

	_, err :=
		traceroute.CreateTraceRouteDeployment(
			APIClient,
			trNamespace,
			trDeploymentLabel,
			RDSCoreConfig.MetalLBSupportToolsImage,
			snifferNodesList)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to create traceroute deployment for nodes %v in namespace %s from image %s: %v",
			snifferNodesList, trNamespace, RDSCoreConfig.MetalLBSupportToolsImage, err))

	By("Finding support-tools (traceroute) pods")

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Searching for pods in %q namespace", trNamespace)

	trPodList := findPodWithSelector(trNamespace, trDeploymentLabel)
	Expect(len(frrPodList)).ToNot(Equal(0), "No traceRoute FRR pods found in namespace %s",
		trNamespace)

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Found %d 'traceRoute' pods", len(frrPodList))

	for _, _pod := range trPodList {
		By("Verify the correct route was used for the main MetalLB-FRR")

		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Verifying that the correct route was used "+
			"for the main FRR defined %s/%s", RDSCoreConfig.MetalLBFirstFRRIPv4, RDSCoreConfig.MetalLBFirstFRRIPv6)

		for _, targetHostIP := range []string{RDSCoreConfig.MetalLBFirstFRRTargetIPv4,
			RDSCoreConfig.MetalLBFirstFRRTargetIPv6} {
			if targetHostIP == "" {
				glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Empty target IP continue")

				continue
			}

			myIP, err := netip.ParseAddr(targetHostIP)
			Expect(err).ToNot(HaveOccurred(),
				fmt.Sprintf("Failed to parse host ip %s", targetHostIP))

			searchString := RDSCoreConfig.MetalLBFirstFRRIPv4

			if myIP.Is6() {
				searchString = RDSCoreConfig.MetalLBFirstFRRIPv6
			}

			err = traceroute.SendTrafficFindExpectedString(
				_pod,
				targetHostIP,
				RDSCoreConfig.MetalLBFRRTargetPort,
				searchString)
			Expect(err).ToNot(HaveOccurred(),
				fmt.Sprintf("Failed to find required FRR IP address %s in response: %v", searchString, err))
		}

		By("Verify the correct route was used for the secondary MetalLB-FRR")

		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Verifying that the correct route was used "+
			"for the second FRR defined %s/%s", RDSCoreConfig.MetalLBSecondFRRIPv4, RDSCoreConfig.MetalLBSecondFRRIPv6)

		for _, targetHostIP := range []string{RDSCoreConfig.MetalLBSecondFRRTargetIPv4,
			RDSCoreConfig.MetalLBSecondFRRTargetIPv6} {
			if targetHostIP == "" {
				glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Empty target IP continue")

				continue
			}

			myIP, err := netip.ParseAddr(targetHostIP)
			Expect(err).ToNot(HaveOccurred(),
				fmt.Sprintf("Failed to parse host ip %s", targetHostIP))

			searchString := RDSCoreConfig.MetalLBSecondFRRIPv4

			if myIP.Is6() {
				searchString = RDSCoreConfig.MetalLBSecondFRRIPv6
			}

			err = traceroute.SendTrafficFindExpectedString(
				_pod,
				targetHostIP,
				RDSCoreConfig.MetalLBFRRTargetPort,
				searchString)
			Expect(err).ToNot(HaveOccurred(),
				fmt.Sprintf("Failed to find required FRR IP address %s in response: %v", searchString, err))
		}
	}
}
