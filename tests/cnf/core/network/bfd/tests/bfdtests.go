package tests

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/configmap"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/daemonset"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/nodes"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/pod"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/rbac"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/reportxml"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/core/network/bfd/internal/netbfdhelper"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/core/network/bfd/internal/tsparams"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/core/network/internal/netinittools"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("BFD", Ordered, Label(tsparams.LabelSuite), ContinueOnFailure, func() {
	var (
		masterNodePod        *pod.Builder
		workerNodesAddresses []string
	)
	BeforeAll(func() {

		By("Getting ip addresses of master nodes")
		masterNodesAddresses, err := netbfdhelper.NodeIPs(netinittools.APIClient, netinittools.NetConfig.ControlPlaneLabel)
		Expect(err).ToNot(HaveOccurred(), "Unable to fetch Master Node IPs")

		By("Getting ip addresses of worker nodes")
		workerNodesAddresses, err = netbfdhelper.NodeIPs(netinittools.APIClient, netinittools.NetConfig.WorkerLabel)
		Expect(err).ToNot(HaveOccurred(), "Unable to fetch Master Node IPs")

		By("Creating configmaps")
		_, err = configmap.
			NewBuilder(netinittools.APIClient, tsparams.WorkerConfigMapName, tsparams.TestNamespace).
			WithData(netbfdhelper.DefineBFDConfigMapData(masterNodesAddresses[0:1])).
			Create()
		Expect(err).ToNot(HaveOccurred(), "Failed to create ConfigMap for Worker Node FRR Pod")
		_, err = configmap.
			NewBuilder(netinittools.APIClient, tsparams.MasterConfigMapName, tsparams.TestNamespace).
			WithData(netbfdhelper.DefineBFDConfigMapData(workerNodesAddresses)).
			Create()
		Expect(err).ToNot(HaveOccurred(), "Failed to create ConfigMap for Master Node FRR Pod")

		By("Creating a role")
		_, err = rbac.
			NewRoleBuilder(netinittools.APIClient, tsparams.RoleName, tsparams.TestNamespace, netbfdhelper.DefineRolePolicy()).
			Create()
		Expect(err).ToNot(HaveOccurred(), "Failed to create Role")

		By("Creating a role binding")
		_, err = rbac.
			NewRoleBindingBuilder(netinittools.APIClient,
				tsparams.RoleName, tsparams.TestNamespace, tsparams.RoleName, netbfdhelper.DefineRoleBindingSubject()).
			Create()
		Expect(err).ToNot(HaveOccurred(), "Failed to create RoleBinding")

		By("Generating Test Container Spec")
		containerSpec, err := netbfdhelper.DefineTestContainerSpec()
		Expect(err).ToNot(HaveOccurred(), "Failed to create Container Spec")

		By("Creating a daemonset")
		frrDaemonset, err := daemonset.
			NewBuilder(netinittools.APIClient,
				tsparams.AppName, tsparams.TestNamespace, map[string]string{"pod": "frr"}, containerSpec).
			WithHostNetwork().
			WithVolume(netbfdhelper.DefineCmVolume()).
			Create()
		Expect(err).ToNot(HaveOccurred(), "Failed to create FRR Daemonset")

		By("Checking the daemonset is in running state")
		Expect(frrDaemonset.IsReady(2*time.Minute)).Should(BeTrue(), "Daemonset is not Running")

		By("Creating FRR container on a Master node")
		masterNodes, err := nodes.
			List(netinittools.APIClient, metav1.ListOptions{LabelSelector: netinittools.NetConfig.ControlPlaneLabel})
		Expect(err).ToNot(HaveOccurred(), "Failed to fetch Master Nodes")
		masterNodePod, err = pod.
			NewBuilder(netinittools.APIClient,
				tsparams.MasterNodeFRRPodName, tsparams.TestNamespace, netinittools.NetConfig.FrrImage).
			RedefineDefaultCMD([]string{}).
			DefineOnNode(masterNodes[0].Object.Name).
			WithTolerationToMaster().
			WithHostNetwork().
			WithPrivilegedFlag().
			WithLocalVolume(tsparams.MasterConfigMapName, tsparams.MountPath).
			CreateAndWaitUntilRunning(2 * time.Minute)
		Expect(err).ToNot(HaveOccurred(), "Failed to create FRR pod on master node")

	})

	It("Should have BFD status up", reportxml.ID("61337"), func() {
		Eventually(func() error {
			return netbfdhelper.IsBFDStatusUp(masterNodePod, workerNodesAddresses)
		},
			2*time.Minute, 2*time.Second).ShouldNot(HaveOccurred(), "BFD Status is not up")
	})
})
