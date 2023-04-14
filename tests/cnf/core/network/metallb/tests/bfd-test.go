package tests

import (
	"fmt"
	"strings"
	"time"

	"github.com/onsi/gomega/types"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-gotests/pkg/configmap"
	"github.com/openshift-kni/eco-gotests/pkg/daemonset"
	"github.com/openshift-kni/eco-gotests/pkg/metallb"
	"github.com/openshift-kni/eco-gotests/pkg/nad"
	"github.com/openshift-kni/eco-gotests/pkg/namespace"
	"github.com/openshift-kni/eco-gotests/pkg/nodes"
	"github.com/openshift-kni/eco-gotests/pkg/pod"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/internal/coreparams"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/define"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netinittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/frr"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/metallbenv"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
)

// test cases variables that are accessible across entire file.
var (
	ipv4metalLbIPList []string
	ipv4NodeAddrList  []string
	externalNad       *nad.Builder
	workerNodeList    *nodes.Builder
)

var _ = Describe("BFD", Ordered, Label(tsparams.LabelBFDTestCases), ContinueOnFailure, func() {

	BeforeAll(func() {
		By("Creating a new instance of MetalLB Speakers on workers")
		err := metallbenv.CreateNewMetalLbDaemonSetAndWaitUntilItsRunning(tsparams.DefaultTimeout)
		Expect(err).ToNot(HaveOccurred(), "Failed to recreate metalLb daemonset")

		By("Getting MetalLb load balancer ip addresses")
		ipv4metalLbIPList, _, err = metallbenv.GetMetalLbIPByIPStack()
		Expect(err).ToNot(HaveOccurred(), "An unexpected error occurred while "+
			"determining the IP addresses from the ECO_CNF_CORE_NET_MLB_ADDR_LIST environment variable.")

		if len(ipv4metalLbIPList) < 2 {
			Skip("MetalLb BFD tests require 2 ip addresses. Please check ECO_CNF_CORE_NET_MLB_ADDR_LIST env var")
		}

		By("Getting external nodes ip addresses")
		workerNodeList = nodes.NewBuilder(APIClient, NetConfig.WorkerLabelMap)
		Expect(workerNodeList.Discover()).ToNot(HaveOccurred(), "Failed to discover worker nodes")
		ipv4NodeAddrList, err = workerNodeList.ExternalIPv4Networks()
		Expect(err).ToNot(HaveOccurred(), "Failed to collect external nodes ip addresses")

		err = metallbenv.IsEnvVarMetalLbIPinNodeExtNetRange(ipv4NodeAddrList, ipv4metalLbIPList, nil)
		Expect(err).ToNot(HaveOccurred(), "Failed to validate metalLb exported ip address")

		By("Creating external BR-EX NetworkAttachmentDefinition")
		macVlanPlugin, err := define.MasterNadPlugin(coreparams.OvnExternalBridge, "bridge", nad.IPAMStatic())
		Expect(err).ToNot(HaveOccurred(), "Failed to define master nad plugin")
		externalNad, err = nad.NewBuilder(APIClient, tsparams.ExternalMacVlanNADName, tsparams.TestNamespaceName).
			WithMasterPlugin(macVlanPlugin).Create()
		Expect(err).ToNot(HaveOccurred(), "Failed to create external NetworkAttachmentDefinition")
		Expect(externalNad.Exists()).To(BeTrue(), "Failed to detect external NetworkAttachmentDefinition")
	})

	Context("multi hops", Label("mutihop"), func() {

		It("should provide fast link failure detection", polarion.ID("47186"), func() {

		})

	})

	Context("single hop", Label("singlehop"), func() {
		BeforeEach(func() {
			By("Creating BFD profile")
			bfdProfile, err := metallb.NewBFDBuilder(APIClient, "bfdprofile", NetConfig.MlbOperatorNamespace).
				WithRcvInterval(300).WithTransmitInterval(300).WithEchoInterval(100).
				WithEchoMode(true).WithPassiveMode(false).WithMinimumTTL(5).
				WithMultiplier(3).Create()
			Expect(err).ToNot(HaveOccurred(), "Failed to create BFD profile")
			Expect(bfdProfile.Exists()).To(BeTrue(), "BFD profile doesn't exist")

			By("Verifying that BFD profile is applied to speakers")
			speakerPods, err := pod.List(APIClient, NetConfig.MlbOperatorNamespace, v1.ListOptions{
				LabelSelector: "component=speaker",
			})
			Expect(err).ToNot(HaveOccurred(), "Failed to list pods")
			for _, speakerPod := range speakerPods {
				Eventually(frr.IsProtocolConfigured,
					time.Minute, tsparams.DefaultRetryInterval).WithArguments(speakerPod, "bfd").
					Should(BeTrue(), "BFD is not configured on the Speakers")
			}

			By("Creating BGP Peer")
			_, err = metallb.NewBPGPeerBuilder(APIClient, "testpeer", NetConfig.MlbOperatorNamespace,
				ipv4metalLbIPList[0], 64500, 64501).WithBFDProfile(bfdProfile.Definition.Name).
				WithPassword(tsparams.BGPPassword).Create()
			Expect(err).ToNot(HaveOccurred(), "Failed tp create BGP peer")

			By("Verifying if BGP protocol configured")
			for _, speakerPod := range speakerPods {
				Eventually(frr.IsProtocolConfigured,
					time.Minute, tsparams.DefaultRetryInterval).WithArguments(speakerPod, "router bgp").
					Should(BeTrue(), "BGP is not configured on the Speakers")
			}

			By("Creating MetalLb configMap")
			frrBFDConfig := frr.DefineBFDConfig(
				64501, 64500, removePrefixFromIPList(ipv4NodeAddrList), false)
			configMapData := frr.DefineBaseConfig(tsparams.DaemonsFile, frrBFDConfig, "")

			bfdConfigMap, err := configmap.NewBuilder(
				APIClient, tsparams.FRRDefaultConfigMapName, tsparams.TestNamespaceName).WithData(configMapData).Create()
			Expect(err).ToNot(HaveOccurred(), "Failed to create configmap")

			By("Creating FRR Pod With network and IP")
			masterNodeList := nodes.NewBuilder(APIClient, map[string]string{"node-role.kubernetes.io/master": ""})
			err = masterNodeList.Discover()
			Expect(err).ToNot(HaveOccurred(), "Failed to discover control-plane nodes")

			frrPod := pod.NewBuilder(
				APIClient, tsparams.FRRContainerName, tsparams.TestNamespaceName, NetConfig.FrrImage).
				DefineOnNode(masterNodeList.Objects[0].Object.Name).WithTolerationToMaster().WithPrivilegedFlag()

			By("Creating FRR container")
			frrContainer := pod.NewContainerBuilder("frr2", NetConfig.FrrImage, tsparams.SleepCMD).
				WithSecurityCapabilities([]string{"NET_ADMIN", "NET_RAW", "SYS_ADMIN"}, true)

			frrCtr, err := frrContainer.GetContainerCfg()
			Expect(err).ToNot(HaveOccurred(), "Failed to get container configuration")

			By("Creating static ip annotation")
			staticIPAnnotation := pod.StaticIPAnnotation(
				externalNad.Definition.Name, []string{fmt.Sprintf("%s/24", ipv4metalLbIPList[0])})

			By("Creating FRR pod in the test namespace")
			frrPod.WithAdditionalContainer(frrCtr).WithSecondaryNetwork(staticIPAnnotation)
			frrPod.WithLocalVolume(bfdConfigMap.Definition.Name, "/etc/frr").RedefineDefaultCMD([]string{})
			_, err = frrPod.CreateAndWaitUntilRunning(time.Minute)
			Expect(err).ToNot(HaveOccurred(), "Failed to create frr test pod")

			By("Checking that BGP and BFD sessions are established and up")
			verifyMetalLbBFDAndBGPSessionsAreUPOnFrrPod(frrPod, ipv4NodeAddrList)
		})

		It("basic functionality should provide fast link failure detection", polarion.ID("47188"), func() {
			By("Changing the label selector for MetalLb speakers")
			metalLbIo, err := metallb.Pull(APIClient, tsparams.MetalLbIo, NetConfig.MlbOperatorNamespace)
			Expect(err).ToNot(HaveOccurred(), "Failed to pull metallb.io object")
			_, err = metalLbIo.WithSpeakerNodeSelector(tsparams.MetalLbSpeakerLabel).Update(false)
			Expect(err).ToNot(HaveOccurred(), "Failed to update metallb object with the new MetalLb label")

			By("Verifying that the MetalLb speakers are not running on nodes after label update")
			metalLbDs, err := daemonset.Pull(APIClient, tsparams.MetalLbDsName, NetConfig.MlbOperatorNamespace)
			Expect(err).ToNot(HaveOccurred(), "Failed to pull metalLb speaker daemonSet")
			metalLbDaemonSetShouldMatchConditionAndBeInReadyState(
				metalLbDs, BeZero(), "Failed to scale down metalLb speaker pods to zero")

			By("Adding metalLb label to compute nodes")
			Expect(workerNodeList.Discover()).ToNot(HaveOccurred(), "Failed to discover worker nodes")
			for _, worker := range workerNodeList.Objects {
				_, err = worker.WithNewLabel(mapFirstKeyValue(tsparams.MetalLbSpeakerLabel)).Update()
				Expect(err).ToNot(HaveOccurred(), "Failed to append new metalLb label to nodes objects")
			}
			metalLbDaemonSetShouldMatchConditionAndBeInReadyState(
				metalLbDs, Not(BeZero()), "Failed to run metalLb speakers on top of nodes with test label")

			By("Checking that BGP and BFD sessions are established and up")
			frrPod, err := pod.Pull(APIClient, tsparams.FRRContainerName, tsparams.TestNamespaceName)
			Expect(err).ToNot(HaveOccurred(), "Failed to pull frr test pod")
			verifyMetalLbBFDAndBGPSessionsAreUPOnFrrPod(frrPod, ipv4NodeAddrList)

			By("Removing Speaker pod from one of the compute nodes")
			firstWorkerNode, err := nodes.PullNode(APIClient, workerNodeList.Objects[0].Object.Name)
			Expect(err).ToNot(HaveOccurred(), "Failed to pull worker node object")
			_, err = firstWorkerNode.RemoveLabel(mapFirstKeyValue(tsparams.MetalLbSpeakerLabel)).Update()
			Expect(err).ToNot(HaveOccurred(), "Failed to remove metalLb label from worker node")

			By("Verifying that cluster has reduced the number of speakers by 1")
			metalLbDaemonSetShouldMatchConditionAndBeInReadyState(
				metalLbDs, BeEquivalentTo(len(workerNodeList.Objects)-1), "The number of running speaker pods is not expected")

			By("Verifying that FRR pod still has BFD and BGP session UP with one of the MetalLb speakers")
			secondWorkerNode, err := nodes.PullNode(APIClient, workerNodeList.Objects[1].Object.Name)
			Expect(err).ToNot(HaveOccurred(), "Failed to pull compute node object")
			secondWorkerIP, err := secondWorkerNode.ExternalIPv4Network()
			Expect(err).ToNot(HaveOccurred(), "Failed to collect external node ip")

			// Sleep until BFD timeout
			time.Sleep(1200 * time.Millisecond)
			bpgUp, err := frr.BGPNeighborshipHasState(frrPod, removePrefixFromIP(secondWorkerIP), "Established")
			Expect(err).ToNot(HaveOccurred(), "Failed to collect bgp state from FRR router")
			Expect(bpgUp).Should(BeTrue(), "BGP is not in expected established state")
			Expect(frr.BFDHasStatus(frrPod, removePrefixFromIP(secondWorkerIP), "up")).Should(BeNil(),
				"BFD is not in expected up state")

			By("Verifying that FRR pod lost BFD and BGP session with one of the MetalLb speakers")
			firstWorkerNodeIP, err := firstWorkerNode.ExternalIPv4Network()
			Expect(err).ToNot(HaveOccurred(), "Failed to collect external node ip")
			bpgUp, err = frr.BGPNeighborshipHasState(frrPod, removePrefixFromIP(firstWorkerNodeIP), "Established")
			Expect(err).ToNot(HaveOccurred(), "Failed to collect BGP state")
			Expect(bpgUp).Should(BeFalse(), "BGP is not in expected down state")
			Expect(frr.BFDHasStatus(frrPod, removePrefixFromIP(firstWorkerNodeIP), "down")).
				ShouldNot(HaveOccurred(), "BFD is not in expected down state")

			By("Bringing Speaker pod back by labeling node")
			_, err = firstWorkerNode.WithNewLabel(mapFirstKeyValue(tsparams.MetalLbSpeakerLabel)).Update()
			Expect(err).ToNot(HaveOccurred(), "Failed to append metalLb label to worker node")

			By("Check if speakers daemonSet is UP and running")
			metalLbDaemonSetShouldMatchConditionAndBeInReadyState(
				metalLbDs, BeEquivalentTo(len(workerNodeList.Objects)), "The number of running speak pods is not expected")
			verifyMetalLbBFDAndBGPSessionsAreUPOnFrrPod(frrPod, ipv4NodeAddrList)

		})

		AfterEach(func() {
			By("Removing label from Workers")
			Expect(workerNodeList.Discover()).ToNot(HaveOccurred(), "Failed to discover worker nodes")
			for _, worker := range workerNodeList.Objects {
				_, err := worker.RemoveLabel(mapFirstKeyValue(tsparams.MetalLbSpeakerLabel)).Update()
				Expect(err).ToNot(HaveOccurred(), "Failed to remove label from worker")
			}

			By("Cleaning MetalLb operator namespace")
			metalLbNs, err := namespace.Pull(APIClient, NetConfig.MlbOperatorNamespace)
			Expect(err).ToNot(HaveOccurred(), "Failed to pull metalLb operator namespace")
			err = metalLbNs.CleanObjects(tsparams.DefaultTimeout, metallb.GetBGPPeerGVR(), metallb.GetBFDProfileGVR())
			Expect(err).ToNot(HaveOccurred(), "Failed to remove object's from operator namespace")

			By("Cleaning test namespace")
			err = namespace.NewBuilder(APIClient, tsparams.TestNamespaceName).
				CleanObjects(tsparams.DefaultTimeout, pod.GetGVR())
			Expect(err).ToNot(HaveOccurred(), "Failed to clean objects from test namespace")

			By("Deleting configmap")
			bfdConfigMap, err := configmap.Pull(
				APIClient, tsparams.FRRDefaultConfigMapName, tsparams.TestNamespaceName)
			Expect(err).ToNot(HaveOccurred(), "Failed to pull frr configmap")
			Expect(bfdConfigMap.Delete()).ToNot(HaveOccurred(), "Failed to delete frr configmap")
		})
	})

	AfterAll(func() {
		By("Cleaning Metallb namespace")
		metalLbNs, err := namespace.Pull(APIClient, NetConfig.MlbOperatorNamespace)
		Expect(err).ToNot(HaveOccurred(), "Failed to pull metalLb namespace")
		err = metalLbNs.CleanObjects(tsparams.DefaultTimeout, metallb.GetMetalLbIoGVR())
		Expect(err).ToNot(HaveOccurred(), "Failed to clean metalLb operator namespace")

		By("Cleaning test namespace")
		err = namespace.NewBuilder(APIClient, tsparams.TestNamespaceName).CleanObjects(
			tsparams.DefaultTimeout, pod.GetGVR(), nad.GetGVR())
		Expect(err).ToNot(HaveOccurred(), "Failed to clean test namespace")
	})
})

func verifyMetalLbBFDAndBGPSessionsAreUPOnFrrPod(frrPod *pod.Builder, peerAddrList []string) {
	for _, peerAddress := range removePrefixFromIPList(peerAddrList) {
		Eventually(frr.BGPNeighborshipHasState,
			time.Minute*2, tsparams.DefaultRetryInterval).
			WithArguments(frrPod, peerAddress, "Established").Should(
			BeTrue(), "Failed to receive BGP status UP")
		Eventually(frr.BFDHasStatus,
			time.Minute, tsparams.DefaultRetryInterval).
			WithArguments(frrPod, peerAddress, "up").
			ShouldNot(HaveOccurred(), "Failed to receive BFD status UP")
	}
}

func metalLbDaemonSetShouldMatchConditionAndBeInReadyState(
	metalLbDs *daemonset.Builder, expectedCondition types.GomegaMatcher, errorMessage string) {
	Eventually(func() int32 {
		if metalLbDs.Exists() {
			return metalLbDs.Object.Status.NumberAvailable
		}

		return 0
	}, tsparams.DefaultTimeout, tsparams.DefaultRetryInterval).Should(expectedCondition, errorMessage)
	Expect(metalLbDs.IsReady(120*time.Second)).To(BeTrue(), "MetalLb daemonSet is not Ready")
}

func removePrefixFromIPList(ipAddrs []string) []string {
	var ipAddrsWithoutPrefix []string
	for _, ipaddr := range ipAddrs {
		ipAddrsWithoutPrefix = append(ipAddrsWithoutPrefix, removePrefixFromIP(ipaddr))
	}

	return ipAddrsWithoutPrefix
}

func removePrefixFromIP(ipAddr string) string {
	return strings.Split(ipAddr, "/")[0]
}

func mapFirstKeyValue(inputMap map[string]string) (string, string) {
	for key, value := range inputMap {
		return key, value
	}

	return "", ""
}
