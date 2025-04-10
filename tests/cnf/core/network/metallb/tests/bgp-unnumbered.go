package tests

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/cmd"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netinittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/frr"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/metallbenv"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/tsparams"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

var _ = Describe("BGP remote-dynamicAS", Ordered, Label(tsparams.LabelDynamicRemoteASTestCases),
	ContinueOnFailure, func() {
		var (
			err error
			// dynamicASiBGP                = "internal"
			dynamicASeBGP = "external"
			// frrExternalMasterIPAddress   = "172.16.0.1"
			hubIPv4ExternalAddresses     = []string{"172.16.0.10", "172.16.0.11"}
			externalAdvertisedIPv4Routes = []string{"192.168.100.0/24", "192.168.200.0/24"}
			externalAdvertisedIPv6Routes = []string{"2001:100::0/64", "2001:200::0/64"}
		)

		BeforeAll(func() {
			By("Getting MetalLb load balancer ip addresses")
			ipv4metalLbIPList, ipv6metalLbIPList, err = metallbenv.GetMetalLbIPByIPStack()
			Expect(err).ToNot(HaveOccurred(), tsparams.MlbAddressListError)

			By("List CNF worker nodes in cluster")
			cnfWorkerNodeList, err = nodes.List(APIClient,
				metav1.ListOptions{LabelSelector: labels.Set(NetConfig.WorkerLabelMap).String()})
			Expect(err).ToNot(HaveOccurred(), "Failed to discover worker nodes")

			By("Selecting worker node 1 for BGP tests")
			fmt.Println("###################", cnfWorkerNodeList[0])
			workerLabelMap, workerNodeList = setWorkerNodeListAndLabelForBfdTests(cnfWorkerNodeList, metalLbTestsLabel)
			ipv4NodeAddrList, err = nodes.ListExternalIPv4Networks(
				APIClient, metav1.ListOptions{LabelSelector: labels.Set(workerLabelMap).String()})
			Expect(err).ToNot(HaveOccurred(), "Failed to collect external nodes ip addresses")

			err = metallbenv.IsEnvVarMetalLbIPinNodeExtNetRange(ipv4NodeAddrList, ipv4metalLbIPList, nil)
			Expect(err).ToNot(HaveOccurred(), "Failed to validate metalLb exported ip address")
		})

		AfterAll(func() {
			By("Remove custom metallb test label from nodes")
			removeNodeLabel(workerNodeList, metalLbTestsLabel)
		})

		AfterEach(func() {
			By("Clean metallb operator and test namespaces")
			resetOperatorAndTestNS()
		})

		FIt("Verify the establishment of an eBGP adjacency using neighbor peer remote-as external",
			reportxml.ID("76821"), func() {
				By("Setup test cases with Frr Node AS 64500 and external Frr AS 64501")
				frrk8sPods, frrPod := setupBGPRemoteASTestCase(hubIPv4ExternalAddresses, externalAdvertisedIPv4Routes,
					externalAdvertisedIPv6Routes, dynamicASeBGP, tsparams.RemoteBGPASN)

				By("Checking that BGP session is established and up")
				verifyMetalLbBGPSessionsAreUPOnFrrPod(frrPod, cmd.RemovePrefixFromIPList(ipv4NodeAddrList))

				By("Validating external FRR AS number received on the FRR nodes")
				Eventually(func() error {
					return frr.ValidateBGPRemoteAS(frrk8sPods, ipv4metalLbIPList[0], tsparams.RemoteBGPASN)
				}, 60*time.Second, 5*time.Second).Should(Succeed(),
					fmt.Sprintf("The remoteASN does not match the expected AS: %d", tsparams.RemoteBGPASN))
			})
	})
