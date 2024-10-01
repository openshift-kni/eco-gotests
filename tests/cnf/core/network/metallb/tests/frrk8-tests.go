package tests

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/configmap"
	"github.com/openshift-kni/eco-goinfra/pkg/daemonset"
	"github.com/openshift-kni/eco-goinfra/pkg/deployment"
	"github.com/openshift-kni/eco-goinfra/pkg/metallb"
	"github.com/openshift-kni/eco-goinfra/pkg/nad"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"github.com/openshift-kni/eco-goinfra/pkg/service"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netinittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/frr"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/metallbenv"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/tsparams"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
)

var _ = Describe("FRR", Ordered, Label(tsparams.LabelFRRTestCases), ContinueOnFailure, func() {
	var (
		externalAdvertisedIPv4Routes = []string{"192.168.100.0/24", "192.168.200.0/24"}
		externalAdvertisedIPv6Routes = []string{"2001:100::0/64", "2001:200::0/64"}
		hubIPaddresses               = []string{"172.16.0.10", "172.16.0.11"}
		err                          error
	)

	BeforeAll(func() {

		By("Getting MetalLb load balancer ip addresses")
		ipv4metalLbIPList, ipv6metalLbIPList, err = metallbenv.GetMetalLbIPByIPStack()
		Expect(err).ToNot(HaveOccurred(), tsparams.MlbAddressListError)

		By("List CNF worker nodes in cluster")
		cnfWorkerNodeList, err = nodes.List(APIClient,
			metav1.ListOptions{LabelSelector: labels.Set(NetConfig.WorkerLabelMap).String()})
		Expect(err).ToNot(HaveOccurred(), "Failed to discover worker nodes")

		By("Selecting worker node for BGP tests")
		workerLabelMap, workerNodeList = setWorkerNodeListAndLabelForBfdTests(cnfWorkerNodeList, metalLbTestsLabel)
		ipv4NodeAddrList, err = nodes.ListExternalIPv4Networks(
			APIClient, metav1.ListOptions{LabelSelector: labels.Set(workerLabelMap).String()})
		Expect(err).ToNot(HaveOccurred(), "Failed to collect external nodes ip addresses")

		err = metallbenv.IsEnvVarMetalLbIPinNodeExtNetRange(ipv4NodeAddrList, ipv4metalLbIPList, nil)
		Expect(err).ToNot(HaveOccurred(), "Failed to validate metalLb exported ip address")

		By("Listing master nodes")
		masterNodeList, err = nodes.List(APIClient,
			metav1.ListOptions{LabelSelector: labels.Set(NetConfig.ControlPlaneLabelMap).String()})
		Expect(err).ToNot(HaveOccurred(), "Fail to list master nodes")
		Expect(len(masterNodeList)).To(BeNumerically(">", 0),
			"Failed to detect master nodes")
	})

	Context("IBGP Single hop", func() {

		var (
			nodeAddrList []string
			addressPool  []string
			err          error
			frrk8sPods   []*pod.Builder
		)

		BeforeAll(func() {
			By("Setting test iteration parameters")
			_, _, _, nodeAddrList, addressPool, _, err =
				metallbenv.DefineIterationParams(
					ipv4metalLbIPList, ipv6metalLbIPList, ipv4NodeAddrList, ipv6NodeAddrList, "IPv4")
			Expect(err).ToNot(HaveOccurred(), "Fail to set iteration parameters")

		})

		AfterEach(func() {
			By("Clean metallb operator and test namespaces")
			resetOperatorAndTestNS()
		})

		It("Verify that prefixes configured with alwaysBlock are not received by the FRR speakers",
			reportxml.ID("74270"), func() {
				prefixToBlock := externalAdvertisedIPv4Routes[0]

				By("Creating a new instance of MetalLB Speakers on workers blocking specific incoming prefixes")
				createNewMetalLbDaemonSetAndWaitUntilItsRunningWithAlwaysBlock(tsparams.DefaultTimeout,
					workerLabelMap, []string{prefixToBlock})

				By("Waiting until the new frr-k8s-webhook-server deployment is in Ready state.")
				frrk8sWebhookDeployment, err := deployment.Pull(
					APIClient, "frr-k8s-webhook-server", NetConfig.MlbOperatorNamespace)
				Expect(err).ToNot(HaveOccurred(), "Fail to pull frr-k8s-webhook-server")
				Expect(frrk8sWebhookDeployment.IsReady(30*time.Second)).To(BeTrue(),
					"frr-k8s-webhook-server deployment is not ready")

				By("Creating external FRR pod on master node")
				frrPod := deployTestPods(addressPool, hubIPaddresses, externalAdvertisedIPv4Routes,
					externalAdvertisedIPv6Routes)

				By("Creating BGP Peers")
				createBGPPeerAndVerifyIfItsReady(ipv4metalLbIPList[0], "", 64500,
					false, 0, frrk8sPods)

				By("Checking that BGP session is established and up")
				verifyMetalLbBGPSessionsAreUPOnFrrPod(frrPod, removePrefixFromIPList(ipv4NodeAddrList))

				By("Validating BGP route prefix")
				validatePrefix(frrPod, "IPv4", removePrefixFromIPList(nodeAddrList), addressPool, 32)

				By("Create a frrconfiguration allow all")
				createFrrConfiguration("frrconfig-allow-all", ipv4metalLbIPList[0],
					64500, nil, false, false)

				frrk8sPods, err := pod.List(APIClient, NetConfig.MlbOperatorNamespace, metav1.ListOptions{
					LabelSelector: "app=frr-k8s",
				})
				Expect(err).ToNot(HaveOccurred(), "Fail to find Frrk8 pod list")

				By("Verify that the node FRR pods advertises two routes")
				verifyExternalAdvertisedRoutes(frrPod, ipv4NodeAddrList, externalAdvertisedIPv4Routes)

				By("Validate BGP received routes")
				Eventually(func() string {
					// Get the routes
					routes, err := frr.VerifyBGPReceivedRoutesOnFrrNodes(frrk8sPods)
					Expect(err).ToNot(HaveOccurred(), "Failed to verify BGP routes")

					// Return the routes to be checked
					return routes
				}, 60*time.Second, 5*time.Second).Should(SatisfyAll(
					Not(ContainSubstring(prefixToBlock)),              // First IP address
					ContainSubstring(externalAdvertisedIPv4Routes[1]), // Second IP address
				), "Fail to find all expected received routes")
			})

		It("Verify the FRR node only receives routes that are configured in the allowed prefixes",
			reportxml.ID("74272"), func() {
				prefixToFilter := externalAdvertisedIPv4Routes[1]

				By("Creating a new instance of MetalLB Speakers on workers")
				err = metallbenv.CreateNewMetalLbDaemonSetAndWaitUntilItsRunning(tsparams.DefaultTimeout, workerLabelMap)
				Expect(err).ToNot(HaveOccurred(), "Failed to recreate metalLb daemonset")

				By("Waiting until the new frr-k8s-webhook-server deployment is in Ready state.")
				frrk8sWebhookDeployment, err := deployment.Pull(
					APIClient, "frr-k8s-webhook-server", NetConfig.MlbOperatorNamespace)
				Expect(err).ToNot(HaveOccurred(), "Fail to pull frr-k8s-webhook-server")
				Expect(frrk8sWebhookDeployment.IsReady(30*time.Second)).To(BeTrue(),
					"frr-k8s-webhook-server deployment is not ready")

				frrPod := deployTestPods(addressPool, hubIPaddresses, externalAdvertisedIPv4Routes,
					externalAdvertisedIPv6Routes)

				By("Creating BGP Peers")
				createBGPPeerAndVerifyIfItsReady(ipv4metalLbIPList[0], "", 64500,
					false, 0, frrk8sPods)

				By("Checking that BGP session is established and up")
				verifyMetalLbBGPSessionsAreUPOnFrrPod(frrPod, removePrefixFromIPList(ipv4NodeAddrList))

				By("Validating BGP route prefix")
				validatePrefix(frrPod, "IPv4", removePrefixFromIPList(nodeAddrList), addressPool, 32)

				By("Create a frrconfiguration with prefix filter")

				createFrrConfiguration("frrconfig-filtered", ipv4metalLbIPList[0],
					64500, []string{externalAdvertisedIPv4Routes[0], externalAdvertisedIPv6Routes[0]}, false, false)

				frrk8sPods, err := pod.List(APIClient, NetConfig.MlbOperatorNamespace, metav1.ListOptions{
					LabelSelector: "app=frr-k8s",
				})
				Expect(err).ToNot(HaveOccurred(), "Fail to find Frrk8 pod list")

				By("Verify that the node FRR pods advertises two routes")
				verifyExternalAdvertisedRoutes(frrPod, ipv4NodeAddrList, externalAdvertisedIPv4Routes)

				By("Validate BGP received routes")
				Eventually(func() string {
					// Get the routes
					routes, err := frr.VerifyBGPReceivedRoutesOnFrrNodes(frrk8sPods)
					Expect(err).ToNot(HaveOccurred(), "Failed to verify BGP routes")

					// Return the routes to be checked
					return routes
				}, 60*time.Second, 5*time.Second).Should(SatisfyAll(
					ContainSubstring(externalAdvertisedIPv4Routes[0]), // First IP address
					Not(ContainSubstring(prefixToFilter)),             // Second IP address
				), "Fail to find all expected received routes")
			})

		It("Verify that when the allow all mode is configured all routes are received on the FRR speaker",
			reportxml.ID("74273"), func() {

				By("Creating a new instance of MetalLB Speakers on workers")
				err = metallbenv.CreateNewMetalLbDaemonSetAndWaitUntilItsRunning(tsparams.DefaultTimeout, workerLabelMap)
				Expect(err).ToNot(HaveOccurred(), "Failed to recreate metalLb daemonset")

				By("Waiting until the new frr-k8s-webhook-server deployment is in Ready state.")
				frrk8sWebhookDeployment, err := deployment.Pull(
					APIClient, "frr-k8s-webhook-server", NetConfig.MlbOperatorNamespace)
				Expect(err).ToNot(HaveOccurred(), "Fail to pull frr-k8s-webhook-server")
				Expect(frrk8sWebhookDeployment.IsReady(30*time.Second)).To(BeTrue(),
					"frr-k8s-webhook-server deployment is not ready")

				frrPod := deployTestPods(addressPool, hubIPaddresses, externalAdvertisedIPv4Routes,
					externalAdvertisedIPv6Routes)

				By("Creating BGP Peers")
				createBGPPeerAndVerifyIfItsReady(ipv4metalLbIPList[0], "", 64500,
					false, 0, frrk8sPods)

				By("Checking that BGP session is established and up")
				verifyMetalLbBGPSessionsAreUPOnFrrPod(frrPod, removePrefixFromIPList(ipv4NodeAddrList))

				By("Validating BGP route prefix")
				validatePrefix(frrPod, "IPv4", removePrefixFromIPList(nodeAddrList), addressPool, 32)

				By("Create a frrconfiguration allow all")
				createFrrConfiguration("frrconfig-allow-all", ipv4metalLbIPList[0], 64500, nil, false, false)

				frrk8sPods, err := pod.List(APIClient, NetConfig.MlbOperatorNamespace, metav1.ListOptions{
					LabelSelector: "app=frr-k8s",
				})
				Expect(err).ToNot(HaveOccurred(), "Fail to find Frrk8 pod list")

				By("Verify that the node FRR pods advertises two routes")
				verifyExternalAdvertisedRoutes(frrPod, ipv4NodeAddrList, externalAdvertisedIPv4Routes)

				By("Validate BGP received routes")
				Eventually(func() string {
					// Get the routes
					routes, err := frr.VerifyBGPReceivedRoutesOnFrrNodes(frrk8sPods)
					Expect(err).ToNot(HaveOccurred(), "Failed to verify BGP routes")

					// Return the routes to be checked
					return routes
				}, 60*time.Second, 5*time.Second).Should(SatisfyAll(
					ContainSubstring(externalAdvertisedIPv4Routes[0]), // First IP address
					ContainSubstring(externalAdvertisedIPv4Routes[1]), // Second IP address
				), "Fail to find all expected received routes")
			})

		It("Verify that a FRR speaker can be updated by merging two different FRRConfigurations",
			reportxml.ID("74274"), func() {

				By("Creating a new instance of MetalLB Speakers on workers")
				err = metallbenv.CreateNewMetalLbDaemonSetAndWaitUntilItsRunning(tsparams.DefaultTimeout, workerLabelMap)
				Expect(err).ToNot(HaveOccurred(), "Failed to recreate metalLb daemonset")

				By("Waiting until the new frr-k8s-webhook-server deployment is in Ready state.")
				frrk8sWebhookDeployment, err := deployment.Pull(
					APIClient, "frr-k8s-webhook-server", NetConfig.MlbOperatorNamespace)
				Expect(err).ToNot(HaveOccurred(), "Fail to pull frr-k8s-webhook-server")
				Expect(frrk8sWebhookDeployment.IsReady(30*time.Second)).To(BeTrue(),
					"frr-k8s-webhook-server deployment is not ready")

				frrPod := deployTestPods(addressPool, hubIPaddresses, externalAdvertisedIPv4Routes,
					externalAdvertisedIPv6Routes)

				By("Creating BGP Peers")
				createBGPPeerAndVerifyIfItsReady(ipv4metalLbIPList[0], "", 64500,
					false, 0, frrk8sPods)

				By("Checking that BGP session is established and up")
				verifyMetalLbBGPSessionsAreUPOnFrrPod(frrPod, removePrefixFromIPList(ipv4NodeAddrList))

				By("Validating BGP route prefix")
				validatePrefix(frrPod, "IPv4", removePrefixFromIPList(nodeAddrList), addressPool, 32)

				By("Create first frrconfiguration that receieves a single route")
				createFrrConfiguration("frrconfig-filtered-1", ipv4metalLbIPList[0], 64500,
					[]string{externalAdvertisedIPv4Routes[0], externalAdvertisedIPv6Routes[0]}, false, false)

				frrk8sPods, err := pod.List(APIClient, NetConfig.MlbOperatorNamespace, metav1.ListOptions{
					LabelSelector: "app=frr-k8s",
				})
				Expect(err).ToNot(HaveOccurred(), "Fail to find Frrk8 pod list")

				By("Verify that the node FRR pods advertises two routes")
				verifyExternalAdvertisedRoutes(frrPod, ipv4NodeAddrList, externalAdvertisedIPv4Routes)

				By("Validate BGP received only the first route")
				Eventually(func() string {
					// Get the routes
					routes, err := frr.VerifyBGPReceivedRoutesOnFrrNodes(frrk8sPods)
					Expect(err).ToNot(HaveOccurred(), "Failed to verify BGP routes")

					// Return the routes to be checked
					return routes
				}, 60*time.Second, 5*time.Second).Should(SatisfyAll(
					ContainSubstring(externalAdvertisedIPv4Routes[0]),      // First IP address
					Not(ContainSubstring(externalAdvertisedIPv4Routes[1])), // Second IP address
				), "Fail to find all expected received routes")

				By("Create second frrconfiguration that receives a single route")
				createFrrConfiguration("frrconfig-filtered-2", ipv4metalLbIPList[0], 64500,
					[]string{externalAdvertisedIPv4Routes[1], externalAdvertisedIPv6Routes[1]}, false,
					false)

				By("Validate BGP received both the first and second routes")
				Eventually(func() string {
					// Get the routes
					routes, err := frr.VerifyBGPReceivedRoutesOnFrrNodes(frrk8sPods)
					Expect(err).ToNot(HaveOccurred(), "Failed to verify BGP routes")

					// Return the routes to be checked
					return routes
				}, 60*time.Second, 5*time.Second).Should(SatisfyAll(
					ContainSubstring(externalAdvertisedIPv4Routes[0]), // First IP address
					ContainSubstring(externalAdvertisedIPv4Routes[1]), // Second IP address
				), "Fail to find all expected received routes")
			})

		It("Verify that a FRR speaker rejects a contrasting FRRConfiguration merge",
			reportxml.ID("74275"), func() {

				By("Creating a new instance of MetalLB Speakers on workers")
				err = metallbenv.CreateNewMetalLbDaemonSetAndWaitUntilItsRunning(tsparams.DefaultTimeout, workerLabelMap)
				Expect(err).ToNot(HaveOccurred(), "Failed to recreate metalLb daemonset")

				By("Waiting until the new frr-k8s-webhook-server deployment is in Ready state.")
				frrk8sWebhookDeployment, err := deployment.Pull(
					APIClient, "frr-k8s-webhook-server", NetConfig.MlbOperatorNamespace)
				Expect(err).ToNot(HaveOccurred(), "Fail to pull frr-k8s-webhook-server")
				Expect(frrk8sWebhookDeployment.IsReady(30*time.Second)).To(BeTrue(),
					"frr-k8s-webhook-server deployment is not ready")

				By("Create first frrconfiguration that receive a single route")
				createFrrConfiguration("frrconfig-filtered-1", ipv4metalLbIPList[0], 64500,
					[]string{externalAdvertisedIPv4Routes[0], externalAdvertisedIPv6Routes[0]}, false,
					false)

				By("Create second frrconfiguration fails when using an incorrect AS configuration")
				createFrrConfiguration("frrconfig-filtered-2", ipv4metalLbIPList[0], 64501,
					[]string{externalAdvertisedIPv4Routes[1], externalAdvertisedIPv6Routes[1]}, false,
					true)
			})
	})

	Context("BGP Multihop", func() {
		BeforeAll(func() {
			By("Creating a new instance of MetalLB Speakers on workers")
			err := metallbenv.CreateNewMetalLbDaemonSetAndWaitUntilItsRunning(tsparams.DefaultTimeout, workerLabelMap)
			Expect(err).ToNot(HaveOccurred(), "Failed to recreate metalLb daemonset")

			By("Waiting until the new frr-k8s-webhook-server deployment is in Ready state.")
			frrk8sWebhookDeployment, err := deployment.Pull(
				APIClient, "frr-k8s-webhook-server", NetConfig.MlbOperatorNamespace)
			Expect(err).ToNot(HaveOccurred(), "Fail to pull frr-k8s-webhook-server")
			Expect(frrk8sWebhookDeployment.IsReady(30*time.Second)).To(BeTrue(),
				"frr-k8s-webhook-server deployment is not ready")

			frrk8sPods, err := pod.List(APIClient, NetConfig.MlbOperatorNamespace, metav1.ListOptions{
				LabelSelector: "app=frr-k8s",
			})
			Expect(err).ToNot(HaveOccurred(), "Fail to list speaker pods")

			By("Setting test iteration parameters")
			masterClientPodIP, _, _, _, addressPool, _, err :=
				metallbenv.DefineIterationParams(
					ipv4metalLbIPList, ipv6metalLbIPList, ipv4NodeAddrList, ipv6NodeAddrList, "IPv4")
			Expect(err).ToNot(HaveOccurred(), "Fail to set iteration parameters")

			By("Adding static routes to the speakers")
			speakerRoutesMap := buildRoutesMapWithSpecificRoutes(frrk8sPods, ipv4metalLbIPList)

			for _, frrk8sPod := range frrk8sPods {
				out, err := frr.SetStaticRoute(frrk8sPod, "add", masterClientPodIP, speakerRoutesMap)
				Expect(err).ToNot(HaveOccurred(), out)
			}

			By("Creating an IPAddressPool and BGPAdvertisement")

			ipAddressPool := setupBgpAdvertisement(addressPool, int32(32))

			By("Creating a MetalLB service")
			setupMetalLbService("IPv4", ipAddressPool, "Cluster")

			By("Creating nginx test pod on worker node")
			setupNGNXPod(workerNodeList[0].Definition.Name)

			By("Creating External NAD for master FRR pod")
			createExternalNad(tsparams.ExternalMacVlanNADName)

			By("Creating External NAD for hub FRR pods")
			createExternalNad(tsparams.HubMacVlanNADName)

			By("Creating static ip annotation for hub0")
			hub0BRstaticIPAnnotation := pod.StaticIPAnnotation("external", []string{"10.46.81.131/24", "2001:81:81::1/64"})
			hub0BRstaticIPAnnotation = append(hub0BRstaticIPAnnotation,
				pod.StaticIPAnnotation(tsparams.HubMacVlanNADName, []string{"172.16.0.10/24", "2001:100:100::10/64"})...)

			By("Creating static ip annotation for hub1")
			hub1BRstaticIPAnnotation := pod.StaticIPAnnotation("external", []string{"10.46.81.132/24", "2001:81:81::2/64"})
			hub1BRstaticIPAnnotation = append(hub1BRstaticIPAnnotation,
				pod.StaticIPAnnotation(tsparams.HubMacVlanNADName, []string{"172.16.0.11/24", "2001:100:100::11/64"})...)

			By("Creating MetalLb Hub pod configMap")
			hubConfigMap := createHubConfigMap("frr-hub-node-config")

			By("Creating FRR Hub Worker-0 Pod")
			_ = createFrrHubPod("hub-pod-worker-0",
				workerNodeList[0].Object.Name, hubConfigMap.Definition.Name, []string{}, hub0BRstaticIPAnnotation)

			By("Creating FRR Hub Worker-1 Pod")
			_ = createFrrHubPod("hub-pod-worker-1",
				workerNodeList[1].Object.Name, hubConfigMap.Definition.Name, []string{}, hub1BRstaticIPAnnotation)
		})

		AfterAll(func() {
			By("Removing static routes from the speakers")
			frrk8sPods, err := pod.List(APIClient, NetConfig.MlbOperatorNamespace, metav1.ListOptions{
				LabelSelector: tsparams.FRRK8sDefaultLabel,
			})
			Expect(err).ToNot(HaveOccurred(), "Failed to list pods")

			speakerRoutesMap := buildRoutesMapWithSpecificRoutes(frrk8sPods, ipv4metalLbIPList)

			for _, frrk8sPod := range frrk8sPods {
				out, err := frr.SetStaticRoute(frrk8sPod, "del", "172.16.0.1", speakerRoutesMap)
				Expect(err).ToNot(HaveOccurred(), out)
			}

			By("Clean metallb operator and test namespaces")
			resetOperatorAndTestNS()
		})

		It("Validate a FRR node receives and sends IPv4 and IPv6 routes from an IBGP multihop FRR instance",
			reportxml.ID("74278"), func() {

				By("Collecting information before test")
				frrk8sPods, err := pod.List(APIClient, NetConfig.MlbOperatorNamespace, metav1.ListOptions{
					LabelSelector: "app=frr-k8s",
				})
				Expect(err).ToNot(HaveOccurred(), "Failed to list speaker pods")
				By("Setting test iteration parameters")
				_, _, _, nodeAddrList, addressPool, _, err :=
					metallbenv.DefineIterationParams(
						ipv4metalLbIPList, ipv6metalLbIPList, ipv4NodeAddrList, ipv6NodeAddrList, "IPv4")
				Expect(err).ToNot(HaveOccurred(), "Fail to set iteration parameters")

				By("Creating configmap and MetalLb Master pod")
				frrPod := createMasterFrrPod(64500, nodeAddrList, hubIPaddresses, externalAdvertisedIPv4Routes,
					externalAdvertisedIPv6Routes, false)

				By("Creating BGP Peers")
				createBGPPeerAndVerifyIfItsReady("172.16.0.1", "", tsparams.LocalBGPASN,
					false, 0, frrk8sPods)

				By("Checking that BGP session is established and up")
				verifyMetalLbBGPSessionsAreUPOnFrrPod(frrPod, removePrefixFromIPList(ipv4NodeAddrList))

				By("Validating BGP route prefix")
				validatePrefix(frrPod, "IPv4", removePrefixFromIPList(nodeAddrList), addressPool, 32)

				By("Create a frrconfiguration allow all for EBGP multihop")
				createFrrConfiguration("frrconfig-allow-all", "172.16.0.1", 64500, nil,
					false, false)

				By("Verify that the node FRR pods advertises two routes")
				verifyExternalAdvertisedRoutes(frrPod, ipv4NodeAddrList, externalAdvertisedIPv4Routes)

				By("Validate BGP received routes")
				Eventually(func() string {
					// Get the routes
					routes, err := frr.VerifyBGPReceivedRoutesOnFrrNodes(frrk8sPods)
					Expect(err).ToNot(HaveOccurred(), "Failed to verify BGP routes")

					// Return the routes to be checked
					return routes
				}, 60*time.Second, 5*time.Second).Should(SatisfyAll(
					ContainSubstring(externalAdvertisedIPv4Routes[0]), // First IP address
					ContainSubstring(externalAdvertisedIPv4Routes[1]), // Second IP address
				), "Fail to find all expected received routes")
			})

		It("Validate a FRR node receives and sends IPv4 and IPv6 routes from an EBGP multihop FRR instance",
			reportxml.ID("47279"), func() {

				By("Collecting information before test")
				frrk8sPods, err := pod.List(APIClient, NetConfig.MlbOperatorNamespace, metav1.ListOptions{
					LabelSelector: "app=frr-k8s",
				})
				Expect(err).ToNot(HaveOccurred(), "Failed to list speaker pods")
				By("Setting test iteration parameters")
				_, _, _, nodeAddrList, addressPool, _, err :=
					metallbenv.DefineIterationParams(
						ipv4metalLbIPList, ipv6metalLbIPList, ipv4NodeAddrList, ipv6NodeAddrList, "IPv4")
				Expect(err).ToNot(HaveOccurred(), "Fail to set iteration parameters")

				By("Creating MetalLb Master pod configMap")
				frrPod := createMasterFrrPod(64501, nodeAddrList, hubIPaddresses, externalAdvertisedIPv4Routes,
					externalAdvertisedIPv6Routes, true)

				By("Creating BGP Peers")
				createBGPPeerAndVerifyIfItsReady("172.16.0.1", "", tsparams.RemoteBGPASN,
					true, 0, frrk8sPods)

				By("Checking that BGP session is established and up")
				verifyMetalLbBGPSessionsAreUPOnFrrPod(frrPod, removePrefixFromIPList(ipv4NodeAddrList))

				By("Validating BGP route prefix")
				validatePrefix(frrPod, "IPv4", removePrefixFromIPList(nodeAddrList), addressPool, 32)

				By("Create a frrconfiguration allow all for EBGP multihop")
				createFrrConfiguration("frrconfig-allow-all", "172.16.0.1", 64501,
					nil, true, false)

				By("Verify that the node FRR pods advertises two routes")
				verifyExternalAdvertisedRoutes(frrPod, ipv4NodeAddrList, externalAdvertisedIPv4Routes)

				By("Validate BGP received routes")
				Eventually(func() string {
					// Get the routes
					routes, err := frr.VerifyBGPReceivedRoutesOnFrrNodes(frrk8sPods)
					Expect(err).ToNot(HaveOccurred(), "Failed to verify BGP routes")

					// Return the routes to be checked
					return routes
				}, 60*time.Second, 5*time.Second).Should(SatisfyAll(
					ContainSubstring(externalAdvertisedIPv4Routes[0]), // First IP address
					ContainSubstring(externalAdvertisedIPv4Routes[1]), // Second IP address
				), "Fail to find all expected received routes")
			})
	})

})

func createNewMetalLbDaemonSetAndWaitUntilItsRunningWithAlwaysBlock(timeout time.Duration,
	nodeLabel map[string]string, prefixes []string) {
	By("Verifying if metalLb daemonset is running")

	metalLbIo, err := metallb.Pull(APIClient, tsparams.MetalLbIo, NetConfig.MlbOperatorNamespace)

	if err == nil {
		By("MetalLb daemonset is running. Removing daemonset.")

		_, err = metalLbIo.Delete()
		Expect(err).ToNot(HaveOccurred(), "Failed to delete MetalLb daemonset")
	}

	By("Create new metalLb speaker's daemonSet.")

	metalLbIo = metallb.NewBuilder(APIClient, tsparams.MetalLbIo, NetConfig.MlbOperatorNamespace, nodeLabel)
	metalLbIo.WithFRRConfigAlwaysBlock(prefixes)

	_, err = metalLbIo.Create()
	Expect(err).ToNot(HaveOccurred(), "Failed to create new MetalLb daemonset")

	var metalLbDs *daemonset.Builder

	err = wait.PollUntilContextTimeout(
		context.TODO(), 3*time.Second, timeout, true, func(ctx context.Context) (bool, error) {
			metalLbDs, err = daemonset.Pull(APIClient, tsparams.MetalLbDsName, NetConfig.MlbOperatorNamespace)
			if err != nil {
				By(fmt.Sprintf("Error pulling speakers in %s namespace %s, retry",
					tsparams.MetalLbDsName, NetConfig.MlbOperatorNamespace))

				return false, nil
			}

			metalLbDs, err = daemonset.Pull(APIClient, tsparams.FrrDsName, NetConfig.MlbOperatorNamespace)
			if err != nil {
				By(fmt.Sprintf("Error pulling frrk8s in %s namespace %s, retry",
					tsparams.FRRK8sDefaultLabel, NetConfig.MlbOperatorNamespace))

				return false, nil
			}

			return true, nil
		})
	Expect(err).ToNot(HaveOccurred(), "Failed to wait for MetalLb daemonset readiness")

	By("Waiting until the new metalLb daemonset is in Ready state.")
	Expect(metalLbDs.IsReady(timeout)).To(BeTrue(), "MetalLb daemonset is not ready")
}

func deployTestPods(addressPool, hubIPAddresses, externalAdvertisedIPv4Routes,
	externalAdvertisedIPv6Routes []string) *pod.Builder {
	By("Creating an IPAddressPool and BGPAdvertisement")

	ipAddressPool := setupBgpAdvertisement(addressPool, int32(32))

	By("Creating a MetalLB service")
	setupMetalLbService("IPv4", ipAddressPool, "Cluster")

	By("Creating nginx test pod on worker node")
	setupNGNXPod(workerNodeList[0].Definition.Name)

	By("Creating External NAD")
	createExternalNad(tsparams.ExternalMacVlanNADName)

	By("Creating static ip annotation")

	staticIPAnnotation := pod.StaticIPAnnotation(
		externalNad.Definition.Name, []string{fmt.Sprintf("%s/%s", ipv4metalLbIPList[0], "24")})

	By("Creating MetalLb configMap")

	masterConfigMap := createConfigMapWithStaticRoutes(tsparams.LocalBGPASN, ipv4NodeAddrList, hubIPAddresses,
		externalAdvertisedIPv4Routes, externalAdvertisedIPv6Routes, false, false)

	By("Creating FRR Pod")

	frrPod := createFrrPod(
		masterNodeList[0].Object.Name, masterConfigMap.Definition.Name, []string{}, staticIPAnnotation)

	return frrPod
}

func createFrrConfiguration(name, bgpPeerIP string, remoteAS uint32, filteredIP []string, ebgp, expectToFail bool) {
	frrConfig := metallb.NewFrrConfigurationBuilder(APIClient, name,
		NetConfig.MlbOperatorNamespace).
		WithBGPRouter(tsparams.LocalBGPASN).
		WithBGPNeighbor(bgpPeerIP, remoteAS, 0)

	// Check if there are filtered IPs and set the appropriate mode
	if len(filteredIP) > 0 {
		frrConfig.WithToReceiveModeFiltered(filteredIP, 0, 0)
	} else {
		frrConfig.WithToReceiveModeAll(0, 0)
	}

	// If eBGP is enabled, configure MultiHop
	if ebgp {
		frrConfig.WithEBGPMultiHop(0, 0)
	}

	// Set HoldTime, Keepalive, Password, and Port
	frrConfig.
		WithHoldTime(metav1.Duration{Duration: 90 * time.Second}, 0, 0).
		WithKeepalive(metav1.Duration{Duration: 30 * time.Second}, 0, 0).
		WithBGPPassword("bgp-test", 0, 0).
		WithPort(179, 0, 0)

	if expectToFail {
		_, err := frrConfig.Create()
		Expect(err).To(HaveOccurred(), "Failed expected to not create a FRR configuration for %s", name)
	} else {
		_, err := frrConfig.Create()
		Expect(err).ToNot(HaveOccurred(), "Failed to create FRR configuration for %s", name)
	}
}

func createMasterFrrPod(localAS int, ipv4NodeAddrList, hubIPAddresses, externalAdvertisedIPv4Routes,
	externalAdvertisedIPv6Routes []string, ebgpMultiHop bool) *pod.Builder {
	masterConfigMap := createConfigMapWithStaticRoutes(localAS, ipv4NodeAddrList, hubIPAddresses,
		externalAdvertisedIPv4Routes, externalAdvertisedIPv6Routes, ebgpMultiHop, false)

	By("Creating static ip annotation for master FRR pod")

	masterStaticIPAnnotation := pod.StaticIPAnnotation(
		tsparams.HubMacVlanNADName, []string{"172.16.0.1/24", "2001:100:100::254/64"})

	By("Creating FRR Master Pod")

	frrPod := createFrrPod(
		masterNodeList[0].Object.Name, masterConfigMap.Definition.Name, []string{}, masterStaticIPAnnotation)

	return frrPod
}

func createConfigMapWithStaticRoutes(
	bgpAsn int, nodeAddrList, hubIPAddresses, externalAdvertisedIPv4Routes, externalAdvertisedIPv6Routes []string,
	enableMultiHop, enableBFD bool) *configmap.Builder {
	frrBFDConfig := frr.DefineBGPConfigWithStaticRouteAndNetwork(
		bgpAsn, tsparams.LocalBGPASN, hubIPAddresses, externalAdvertisedIPv4Routes,
		externalAdvertisedIPv6Routes, removePrefixFromIPList(nodeAddrList), enableMultiHop, enableBFD)
	configMapData := frr.DefineBaseConfig(tsparams.DaemonsFile, frrBFDConfig, "")
	masterConfigMap, err := configmap.NewBuilder(APIClient, "frr-master-node-config", tsparams.TestNamespaceName).
		WithData(configMapData).Create()
	Expect(err).ToNot(HaveOccurred(), "Failed to create config map")

	return masterConfigMap
}

func verifyExternalAdvertisedRoutes(frrPod *pod.Builder, ipv4NodeAddrList, externalExpectedRoutes []string) {
	// Get advertised routes from FRR pod, now returned as a map of node IPs to their advertised routes
	advertisedRoutesMap, err := frr.GetBGPAdvertisedRoutes(frrPod, removePrefixFromIPList(ipv4NodeAddrList))
	Expect(err).ToNot(HaveOccurred(), "Failed to find advertised routes")

	// Iterate through each node in the advertised routes map
	for nodeIP, actualRoutes := range advertisedRoutesMap {
		// Split the string of advertised routes into a slice of routes
		routesSlice := strings.Split(strings.TrimSpace(actualRoutes), "\n")

		// Check that the actual routes for each node contain all the expected routes
		for _, expectedRoute := range externalExpectedRoutes {
			matched, err := ContainElement(expectedRoute).Match(routesSlice)
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to match route %s for node %s", expectedRoute, nodeIP))

			Expect(matched).To(BeTrue(), fmt.Sprintf("Expected route %s not found for node %s", expectedRoute, nodeIP))
		}
	}
}

func resetOperatorAndTestNS() {
	By("Cleaning MetalLb operator namespace")

	metalLbNs, err := namespace.Pull(APIClient, NetConfig.MlbOperatorNamespace)
	Expect(err).ToNot(HaveOccurred(), "Failed to pull metalLb operator namespace")
	err = metalLbNs.CleanObjects(
		tsparams.DefaultTimeout,
		metallb.GetBGPPeerGVR(),
		metallb.GetBFDProfileGVR(),
		metallb.GetBGPPeerGVR(),
		metallb.GetBGPAdvertisementGVR(),
		metallb.GetIPAddressPoolGVR(),
		metallb.GetMetalLbIoGVR(),
		metallb.GetFrrConfigurationGVR())
	Expect(err).ToNot(HaveOccurred(), "Failed to remove object's from operator namespace")

	By("Cleaning test namespace")

	err = namespace.NewBuilder(APIClient, tsparams.TestNamespaceName).CleanObjects(
		tsparams.DefaultTimeout,
		pod.GetGVR(),
		service.GetServiceGVR(),
		configmap.GetGVR(),
		nad.GetGVR())
	Expect(err).ToNot(HaveOccurred(), "Failed to clean test namespace")
}

func buildRoutesMapWithSpecificRoutes(podList []*pod.Builder, nextHopList []string) map[string]string {
	Expect(len(podList)).ToNot(BeZero(), "Pod list is empty")
	Expect(len(nextHopList)).ToNot(BeZero(), "Nexthop IP addresses list is empty")
	Expect(len(nextHopList)).To(BeNumerically(">=", len(podList)),
		fmt.Sprintf("Number of speaker IP addresses[%d] is less than the number of pods[%d]",
			len(nextHopList), len(podList)))

	routesMap := make(map[string]string)

	for _, frrPod := range podList {
		if frrPod.Definition.Spec.NodeName == workerNodeList[0].Definition.Name {
			routesMap[frrPod.Definition.Spec.NodeName] = nextHopList[1]
		} else {
			routesMap[frrPod.Definition.Spec.NodeName] = nextHopList[0]
		}
	}

	return routesMap
}
