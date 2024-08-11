package vcorecommon

import (
	"fmt"

	"github.com/openshift-kni/eco-goinfra/pkg/configmap"
	"github.com/openshift-kni/eco-goinfra/pkg/storage"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/remote"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/vcore/internal/ocpcli"
	lsov1 "github.com/openshift/local-storage-operator/api/v1"
	lsov1alpha1 "github.com/openshift/local-storage-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"time"

	"github.com/openshift-kni/eco-goinfra/pkg/lso"

	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/await"

	"github.com/openshift-kni/eco-goinfra/pkg/console"
	"github.com/openshift-kni/eco-goinfra/pkg/deployment"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	ocsoperatorv1 "github.com/red-hat-storage/ocs-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/strings/slices"

	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/apiobjectshelper"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/openshift-kni/eco-gotests/tests/system-tests/vcore/internal/vcoreinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/vcore/internal/vcoreparams"
)

var (
	odfConsolePlugin       = "odf-console"
	odfOperatorDeployments = []string{"csi-addons-controller-manager", "noobaa-operator",
		"ocs-operator", "odf-console", "odf-operator-controller-manager", "rook-ceph-operator"}

	odfStorageClassName = vcoreparams.ODFStorageClassName
	volumeMode          = corev1.PersistentVolumeBlock
)

// VerifyODFSuite container that contains tests for ODF verification.
func VerifyODFSuite() {
	Describe(
		"ODF validation",
		Label(vcoreparams.LabelVCoreOdf), func() {
			It(fmt.Sprintf("Verifies %s namespace exists", vcoreparams.ODFNamespace),
				Label("odf"), VerifyODFNamespaceExists)

			It("Verify ODF successfully installed",
				Label("odf"), reportxml.ID("63844"), VerifyODFOperatorDeployment)

			It("Apply taints to the ODF nodes",
				Label("odf"), reportxml.ID("74916"), VerifyODFTaints)

			It("Verify ODF console enabled",
				Label("odf"), reportxml.ID("74917"), VerifyODFConsoleConfig)

			// It("Verify localvolumediscovery instance exists",
			//	 Label("odf"), reportxml.ID("74920"), VerifyLocalVolumeDiscovery)

			It("Verify localvolumeset instance exists",
				Label("odf"), reportxml.ID("74918"), VerifyLocalVolumeSet)

			It("Verify ODF operator StorageSystem configuration procedure",
				Label("odf"), reportxml.ID("59487"), VerifyODFStorageSystemConfig)

			It("Apply operators config for the ODF nodes",
				Label("odf"), reportxml.ID("74919"), VerifyOperatorsConfigForODFNodes)
		})
}

// VerifyODFNamespaceExists asserts namespace for ODF exists.
func VerifyODFNamespaceExists(ctx SpecContext) {
	err := apiobjectshelper.VerifyNamespaceExists(APIClient, vcoreparams.ODFNamespace, time.Second)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to pull namespace %q; %v",
		vcoreparams.ODFNamespace, err))
} // func VerifyODFNamespaceExists (ctx SpecContext)

// VerifyODFOperatorDeployment asserts ODF successfully installed.
func VerifyODFOperatorDeployment(ctx SpecContext) {
	for _, operatorPod := range odfOperatorDeployments {
		glog.V(vcoreparams.VCoreLogLevel).Infof("Confirm that odf %s pod was deployed and running in %s namespace",
			operatorPod, vcoreparams.ODFNamespace)

		odfPods, err := pod.ListByNamePattern(APIClient, operatorPod, vcoreparams.ODFNamespace)
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("No %s pods were found in %s namespace; %v",
			operatorPod, vcoreparams.ODFNamespace, err))
		Expect(len(odfPods)).ToNot(Equal(0), fmt.Sprintf("No %s pods were found in %s namespace; %v",
			operatorPod, vcoreparams.ODFNamespace, err))

		odfPod := odfPods[0]
		odfPodName := odfPod.Object.Name

		err = odfPod.WaitUntilReady(time.Second)
		if err != nil {
			odfPodLog, _ := odfPod.GetLog(600*time.Second, operatorPod)
			glog.Fatalf("%s pod in %s namespace in a bad state: %s",
				odfPodName, vcoreparams.ODFNamespace, odfPodLog)
		}
	}

	for _, operatorDeployment := range odfOperatorDeployments {
		glog.V(vcoreparams.VCoreLogLevel).Infof("Confirm that %s deployment is running in %s namespace",
			operatorDeployment, vcoreparams.ODFNamespace)

		odfDeployment, err := deployment.Pull(APIClient, operatorDeployment, vcoreparams.ODFNamespace)
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("%s deployment not found in %s namespace; %v",
			operatorDeployment, vcoreparams.ODFNamespace, err))
		Expect(odfDeployment.IsReady(5*time.Second)).To(Equal(true),
			fmt.Sprintf("Bad state for %s deployment in %s namespace",
				operatorDeployment, vcoreparams.ODFNamespace))
	}
} // func VerifyODFOperatorDeployment (ctx SpecContext)

// VerifyODFTaints asserts ODF nodes taints configuration.
func VerifyODFTaints(ctx SpecContext) {
	glog.V(vcoreparams.VCoreLogLevel).Infof("Apply taints to the ODF nodes")

	odfNodesList, err := nodes.List(APIClient, VCoreConfig.OdfLabelListOption)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to get cluster nodes list due to %v", err))

	for _, odfNode := range odfNodesList {
		glog.V(vcoreparams.VCoreLogLevel).Infof("Insure taints applyed to the %s node", odfNode.Definition.Name)
		applyTaintsCmd := fmt.Sprintf(
			"oc adm taint node %s node.ocs.openshift.io/storage=true:NoSchedule --overwrite=true --kubeconfig=%s",
			odfNode.Definition.Name, VCoreConfig.KubeconfigPath)
		_, err = remote.ExecCmdOnHost(VCoreConfig.Host, VCoreConfig.User, VCoreConfig.Pass, applyTaintsCmd)
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to execute %s script due to %v",
			applyTaintsCmd, err))
	}
} // func VerifyODFTaints (ctx SpecContext)

// VerifyODFConsoleConfig asserts ODF console enabled.
func VerifyODFConsoleConfig(ctx SpecContext) {
	glog.V(vcoreparams.VCoreLogLevel).Infof("Enable odf-console")

	consoleOperatorObj, err := console.PullConsoleOperator(APIClient, "cluster")
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("consoleOperator 'cluster' not found; %v", err))

	consoleOperatorObj, err = consoleOperatorObj.WithPlugins([]string{odfConsolePlugin}, false).Update()
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to add plugins %v to the consoleOperator 'cluster' "+
		"due to: %v", odfConsolePlugin, err))

	newPluginsList, err := consoleOperatorObj.GetPlugins()
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to get plugins value from the consoleOperator "+
		"'cluster'; %v", err))
	Expect(slices.Contains(*newPluginsList, odfConsolePlugin),
		fmt.Sprintf("Failed to add new plugin %s to the consoleOperator plugins list: %v",
			odfConsolePlugin, newPluginsList))
} // func VerifyODFConsoleConfig (ctx SpecContext)

// VerifyLocalVolumeDiscovery asserts localvolumediscovery instance exists.
func VerifyLocalVolumeDiscovery(ctx SpecContext) {
	glog.V(vcoreparams.VCoreLogLevel).Infof("Create localvolumediscovery instance %s in namespace %s if not found",
		vcoreparams.ODFLocalVolumeDiscoveryName, vcoreparams.LSONamespace)

	var err error

	localVolumeDiscoveryObj := lso.NewLocalVolumeDiscoveryBuilder(APIClient,
		vcoreparams.ODFLocalVolumeDiscoveryName,
		vcoreparams.LSONamespace)

	if localVolumeDiscoveryObj.Exists() {
		err = localVolumeDiscoveryObj.Delete()
		Expect(err).ToNot(HaveOccurred(),
			fmt.Sprintf("failed to delete localvolumediscovery %s from namespace %s; %v",
				vcoreparams.ODFLocalVolumeDiscoveryName, vcoreparams.LSONamespace, err))
	}

	nodeSelector := corev1.NodeSelector{NodeSelectorTerms: []corev1.NodeSelectorTerm{{
		MatchExpressions: []corev1.NodeSelectorRequirement{{
			Key:      "kubernetes.io/hostname",
			Operator: "In",
			Values:   []string{"worker-2", "worker-3", "worker-4"},
		}}},
	}}

	tolerations := []corev1.Toleration{{
		Key:      "node.ocs.openshift.io/storage",
		Operator: "Equal",
		Value:    "true",
		Effect:   "NoSchedule",
	}}

	localVolumeDiscoveryObj, err = localVolumeDiscoveryObj.
		WithNodeSelector(nodeSelector).WithTolerations(tolerations).Create()
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to retrieve localVolumeDiscovery %s in namespace %s "+
		"status due to %v", vcoreparams.ODFLocalVolumeDiscoveryName, vcoreparams.LSONamespace, err))
	Expect(await.WaitUntilLVDIsDiscovering(APIClient,
		localVolumeDiscoveryObj.Definition.Name,
		localVolumeDiscoveryObj.Definition.Namespace,
		5*time.Minute)).ToNot(HaveOccurred(),
		fmt.Sprintf("localvolumediscovery %s in namespace %s failed to discover",
			vcoreparams.ODFLocalVolumeDiscoveryName, vcoreparams.LSONamespace))
} // func VerifyLocalVolumeDiscovery (ctx SpecContext)

// VerifyLocalVolumeSet asserts localvolumeset instance exists.
func VerifyLocalVolumeSet(ctx SpecContext) {
	glog.V(vcoreparams.VCoreLogLevel).Infof("Create localvolumeset instance %s in namespace %s if not found",
		vcoreparams.ODFLocalVolumeSetName, vcoreparams.LSONamespace)

	var err error

	localVolumeSetObj := lso.NewLocalVolumeSetBuilder(APIClient,
		vcoreparams.ODFLocalVolumeSetName,
		vcoreparams.LSONamespace)

	nodeSelector := corev1.NodeSelector{NodeSelectorTerms: []corev1.NodeSelectorTerm{{
		MatchExpressions: []corev1.NodeSelectorRequirement{{
			Key:      "kubernetes.io/hostname",
			Operator: "In",
			Values:   []string{"worker-2", "worker-3", "worker-4"},
		}}},
	}}

	deviceInclusionSpec := lsov1alpha1.DeviceInclusionSpec{
		DeviceTypes:                []lsov1alpha1.DeviceType{lsov1alpha1.RawDisk},
		DeviceMechanicalProperties: []lsov1alpha1.DeviceMechanicalProperty{lsov1alpha1.NonRotational},
	}

	tolerations := []corev1.Toleration{{
		Key:      "node.ocs.openshift.io/storage",
		Operator: "Equal",
		Value:    "true",
		Effect:   "NoSchedule",
	}}

	_, err = localVolumeSetObj.WithNodeSelector(nodeSelector).
		WithStorageClassName(vcoreparams.ODFStorageClassName).
		WithVolumeMode(lsov1.PersistentVolumeBlock).
		WithFSType("ext4").
		WithMaxDeviceCount(int32(10)).
		WithDeviceInclusionSpec(deviceInclusionSpec).
		WithTolerations(tolerations).Create()
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to create localvolumeset %s in namespace %s "+
		"due to %v", vcoreparams.ODFLocalVolumeSetName, vcoreparams.LSONamespace, err))

	pvLabel := fmt.Sprintf("storage.openshift.com/owner-name=%s", vcoreparams.ODFLocalVolumeSetName)

	err = await.WaitUntilPersistentVolumeCreated(APIClient,
		3,
		5*time.Minute,
		metav1.ListOptions{LabelSelector: pvLabel})
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to create persistentVolumes due to %v", err))
} // func VerifyLocalVolumeSet (ctx SpecContext)

// VerifyODFStorageSystemConfig asserts ODF storage cluster system successfully configured.
func VerifyODFStorageSystemConfig(ctx SpecContext) {
	glog.V(vcoreparams.VCoreLogLevel).Infof("Cleanup StorageSystem and StorageCluster config")

	storageSystemObj := storage.NewSystemODFBuilder(APIClient,
		vcoreparams.ODFStorageSystemName, vcoreparams.ODFNamespace)

	if storageSystemObj.Exists() {
		err := storageSystemObj.Delete()
		Expect(err).ToNot(HaveOccurred(),
			fmt.Sprintf("failed to delete ODF StorageSystem %s from namespace %s; %v",
				vcoreparams.ODFStorageSystemName, vcoreparams.ODFNamespace, err))
	}

	storageclusterObj := storage.NewStorageClusterBuilder(APIClient,
		vcoreparams.StorageClusterName, vcoreparams.ODFNamespace)

	if storageclusterObj.Exists() {
		err := storageclusterObj.Delete()
		Expect(err).ToNot(HaveOccurred(),
			fmt.Sprintf("failed to delete ODF StorageCluster %s from namespace %s; %v",
				vcoreparams.StorageClusterName, vcoreparams.ODFNamespace, err))
	}

	glog.V(vcoreparams.VCoreLogLevel).Infof("Start to configure ODF StorageSystem")

	_, err := storageSystemObj.WithSpec("storagecluster.ocs.openshift.io/v1",
		vcoreparams.StorageClusterName, vcoreparams.ODFNamespace).Create()
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to create storageSystem %s instance in %s namespace; "+
		"%v", vcoreparams.ODFStorageSystemName, vcoreparams.ODFNamespace, err))

	glog.V(vcoreparams.VCoreLogLevel).Infof("Start to configure ODF StorageCluster")

	annotations := map[string]string{
		"cluster.ocs.openshift.io/local-devices":    "true",
		"uninstall.ocs.openshift.io/cleanup-policy": "delete",
		"uninstall.ocs.openshift.io/mode":           "graceful",
	}

	managedResources := ocsoperatorv1.ManagedResourcesSpec{
		CephBlockPools: ocsoperatorv1.ManageCephBlockPools{
			ReconcileStrategy: "managed",
		},
		CephFilesystems: ocsoperatorv1.ManageCephFilesystems{
			ReconcileStrategy: "managed",
		},
		CephObjectStoreUsers: ocsoperatorv1.ManageCephObjectStoreUsers{
			ReconcileStrategy: "managed",
		},
		CephObjectStores: ocsoperatorv1.ManageCephObjectStores{
			ReconcileStrategy: "managed",
		},
	}

	resourceListMap := make(map[corev1.ResourceName]resource.Quantity)
	resourceListMap[corev1.ResourceStorage] = resource.MustParse("1")

	storageDeviceSet := ocsoperatorv1.StorageDeviceSet{
		Count:    3,
		Replica:  1,
		Portable: false,
		Name:     vcoreparams.ODFLocalVolumeSetName,
		DataPVCTemplate: corev1.PersistentVolumeClaim{
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				Resources: corev1.VolumeResourceRequirements{
					Requests: resourceListMap,
				},
				StorageClassName: &odfStorageClassName,
				VolumeMode:       &volumeMode,
			},
		},
	}

	storageclusterObj, err = storageclusterObj.
		WithAnnotations(annotations).
		WithManageNodes(false).
		WithManagedResources(managedResources).
		WithMonDataDirHostPath("/var/lib/rook").
		WithStorageDeviceSet(storageDeviceSet).Create()
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to create storageCluster %s instance in namespace %s; "+
		"%v", vcoreparams.StorageClusterName, vcoreparams.ODFNamespace, err))
	Expect(storageclusterObj.Exists()).To(Equal(true),
		fmt.Sprintf("Failed to createstorageCluster %s instance in namespace %s due",
			vcoreparams.StorageClusterName, vcoreparams.ODFNamespace))
} // func VerifyODFStorageSystemConfig (ctx SpecContext)

// VerifyOperatorsConfigForODFNodes asserts operators configuration for ODF nodes.
//
//nolint:goconst
func VerifyOperatorsConfigForODFNodes(ctx SpecContext) {
	glog.V(vcoreparams.VCoreLogLevel).Infof("Configuring operators for ODF nodes")

	glog.V(vcoreparams.VCoreLogLevel).Infof("DNS operator")

	patchStr :=
		"spec:\n" +
			"  nodePlacement:\n" +
			"    tolerations:\n" +
			"    - operator: Exists\n"

	// patchStr := `{"spec": {"nodePlacement": {"tolerations": ["operator": "Exists"]}}}`

	err := ocpcli.PatchAPIObject("default", "openshift-dns",
		"dns.operator", "merge", patchStr)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to patch dns.operator default in namespace openshift-dns"+
		" due to %v", err))

	glog.V(vcoreparams.VCoreLogLevel).Infof("Configuring selectors and tolerations for ODF components")

	rookCephConfigMap, err := configmap.Pull(APIClient, vcoreparams.RookCephConfigMapName, vcoreparams.ODFNamespace)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to retrieve configmap %s from "+
		"namespace %s due to %v", vcoreparams.RookCephConfigMapName, vcoreparams.ODFNamespace, err))

	rookData := map[string]string{
		"CSI_PLUGIN_TOLERATIONS": "" +
			"- operator: \"Exists\"\n" +
			"  key: node-role.kubernetes.io/infra\n" +
			"- key: node.ocs.openshift.io/storage\n" +
			"  operator: Equal\n" +
			"  value: \"true\"\n" +
			"  effect: NoSchedule",
		"CSI_PROVISIONER_TOLERATIONS": "" +
			"- key: node.ocs.openshift.io/storage\n" +
			"  operator: Equal\n" +
			"  value: \"true\"\n" +
			"  effect: NoSchedule\n",
		"CSI_PROVISIONER_NODE_AFFINITY": "" +
			"requiredDuringSchedulingIgnoredDuringExecution:\n" +
			"  nodeSelectorTerms:\n" +
			"  - matchExpressions:\n" +
			"    - key: cluster.ocs.openshift.io/openshift-storage\n" +
			"      operator: Exists",
		"CSI_CEPHFS_PROVISIONER_NODE_AFFINITY": "" +
			"requiredDuringSchedulingIgnoredDuringExecution:\n" +
			"  nodeSelectorTerms:\n" +
			"  - matchExpressions:\n" +
			"    - key: cluster.ocs.openshift.io/openshift-storage\n" +
			"      operator: Exists",
		"CSI_NFS_PROVISIONER_NODE_AFFINITY": "" +
			"requiredDuringSchedulingIgnoredDuringExecution:\n" +
			"  nodeSelectorTerms:\n" +
			"  - matchExpressions:\n" +
			"    - key: cluster.ocs.openshift.io/openshift-storage\n" +
			"      operator: Exists",
		"CSI_RBD_PROVISIONER_NODE_AFFINITY": "" +
			"requiredDuringSchedulingIgnoredDuringExecution:\n" +
			"  nodeSelectorTerms:\n" +
			"  - matchExpressions:\n" +
			"    - key: cluster.ocs.openshift.io/openshift-storage\n" +
			"      operator: Exists",
	}

	_, err = rookCephConfigMap.WithData(rookData).Update()
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to update configMap %s in namespace %s due to %v",
		vcoreparams.RookCephConfigMapName, vcoreparams.ODFNamespace, err))

	glog.V(vcoreparams.VCoreLogLevel).Info("Restarting storage pods")

	restartStoragePodsCmd := fmt.Sprintf("oc delete pods -n %s --all --kubeconfig=%s",
		vcoreparams.ODFNamespace, VCoreConfig.KubeconfigPath)
	_, err = remote.ExecCmdOnHost(VCoreConfig.Host, VCoreConfig.User, VCoreConfig.Pass, restartStoragePodsCmd)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to execute %s script due to %v",
		restartStoragePodsCmd, err))

	_, err = pod.WaitForAllPodsInNamespaceRunning(APIClient, vcoreparams.ODFNamespace, 7*time.Minute)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("not all storage pods in namespace %s succeeded to "+
		"recover after deletion due to %v", vcoreparams.ODFNamespace, err))
} // func VerifyOperatorsConfigForODFNodes (ctx SpecContext)
