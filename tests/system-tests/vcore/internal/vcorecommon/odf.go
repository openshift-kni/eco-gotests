package vcorecommon

import (
	"fmt"
	"time"

	"github.com/openshift-kni/eco-goinfra/pkg/console"
	"github.com/openshift-kni/eco-goinfra/pkg/deployment"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"github.com/openshift-kni/eco-goinfra/pkg/storage"
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
	ocsLocalvolumesetName  = "ocs-deviceset"
	storageclusterName     = "ocs-storagecluster"
	odfConsolePlugin       = "odf-console"
	odfOperatorDeployments = []string{"csi-addons-controller-manager", "noobaa-operator",
		"ocs-operator", "odf-console", "odf-operator-controller-manager", "rook-ceph-operator"}

	storageClassName = vcoreparams.StorageClassName
	volumeMode       = corev1.PersistentVolumeBlock
)

// VerifyODFNamespaceExists asserts namespace for ODF exists.
func VerifyODFNamespaceExists(ctx SpecContext) {
	err := apiobjectshelper.VerifyNamespaceExists(APIClient, vcoreparams.ODFNamespace, time.Second)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to pull %q namespace", vcoreparams.ODFNamespace))
} // func VerifyODFNamespaceExists (ctx SpecContext)

// VerifyODFDeployment asserts ODF successfully installed.
func VerifyODFDeployment(ctx SpecContext) {
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
} // func VerifyODFDeployment (ctx SpecContext)

// VerifyODFConfig asserts ODF successfully configured.
func VerifyODFConfig(ctx SpecContext) {
	glog.V(vcoreparams.VCoreLogLevel).Infof("Enable odf-console")

	consoleOperatorObj, err := console.PullConsoleOperator(APIClient, "cluster")
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("consoleOperator 'cluster' not found; %v", err))

	consoleOperatorObj, err = consoleOperatorObj.WithPlugins([]string{odfConsolePlugin}, false).Update()
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to add plugins %v to the consoleOperator 'cluster' due to: %v",
		odfConsolePlugin, err))

	newPluginsList, err := consoleOperatorObj.GetPlugins()
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to get plugins value from the consoleOperator 'cluster'; %v",
		err))
	Expect(slices.Contains(*newPluginsList, odfConsolePlugin),
		fmt.Sprintf("Failed to add new plugin %s to the consoleOperator plugins list: %v",
			odfConsolePlugin, newPluginsList))

	glog.V(vcoreparams.VCoreLogLevel).Infof("Start to configure ODF StorageCluster")

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

	multiCloudGateway := ocsoperatorv1.MultiCloudGatewaySpec{
		ReconcileStrategy: "managed",
	}

	resourceListMap := make(map[corev1.ResourceName]resource.Quantity)
	resourceListMap[corev1.ResourceStorage] = resource.MustParse("1")

	storageDeviceSet := ocsoperatorv1.StorageDeviceSet{
		Count:    3,
		Replica:  1,
		Portable: false,
		Name:     ocsLocalvolumesetName,
		DataPVCTemplate: corev1.PersistentVolumeClaim{
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				Resources: corev1.VolumeResourceRequirements{
					Requests: resourceListMap,
				},
				StorageClassName: &storageClassName,
				VolumeMode:       &volumeMode,
			},
		},
	}

	storageclusterObj, err := storage.NewStorageClusterBuilder(APIClient, storageclusterName, vcoreparams.ODFNamespace).
		WithManageNodes(false).
		WithManagedResources(managedResources).
		WithMonDataDirHostPath("/var/lib/rook").
		WithMultiCloudGateway(multiCloudGateway).WithStorageDeviceSet(storageDeviceSet).Create()
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to create storageCluster %s instance in %s namespace; "+
		"%v", storageclusterName, vcoreparams.ODFNamespace, err))
	Expect(storageclusterObj.Exists()).To(Equal(true),
		fmt.Sprintf("Failed to createstorageCluster %s instance in %s namespace; "+
			"%v", storageclusterName, vcoreparams.ODFNamespace, err))
} // func VerifyODFConfig (ctx SpecContext)

// VerifyODFSuite container that contains tests for ODF verification.
func VerifyODFSuite() {
	Describe(
		"ODF validation",
		Label(vcoreparams.LabelVCoreOperators), func() {
			It(fmt.Sprintf("Verifies %s namespace exists", vcoreparams.ODFNamespace),
				Label("odf"), VerifyODFNamespaceExists)

			It("Verify ODF successfully installed",
				Label("odf"), reportxml.ID("63844"), VerifyODFDeployment)

			It("Verify ODF operator configuration procedure",
				Label("odf"), reportxml.ID("59487"), VerifyODFConfig)
		})
}
