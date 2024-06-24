package vcorecommon

import (
	"fmt"
	"github.com/openshift-kni/eco-goinfra/pkg/clusteroperator"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"time"

	"github.com/openshift-kni/eco-goinfra/pkg/mco"
	"github.com/openshift-kni/eco-goinfra/pkg/nrop"
	//nolint:misspell
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/apiobjectshelper"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/vcore/internal/vcoreparams"
	kubeletconfigv1beta1 "k8s.io/kubelet/config/v1beta1"

	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/openshift-kni/eco-gotests/tests/system-tests/vcore/internal/vcoreinittools"
)

// VerifyNROPSuite container that contains tests for Numa Resources Operator verification.
func VerifyNROPSuite() {
	Describe(
		"NROP validation", //nolint:misspell
		Label(vcoreparams.LabelVCoreOperators), func() {
			It(fmt.Sprintf("Verifies %s namespace exists", vcoreparams.NROPNamespace),
				Label("numa"), VerifyNROPNamespaceExists)

			It("Verify Numa Resources Operator successfully installed",
				Label("numa"), reportxml.ID("66337"), VerifyNROPDeployment)

			It("Verify Numa Resources Operator Custom Resource deployment",
				Label("numa"), reportxml.ID("66343"), VerifyNROPCustomResources)

			It("Create new nodes tuning",
				Label("nto"), reportxml.ID("63740"), CreateNodesTuning) //nolint:misspell

			It("Verify CPU Manager config",
				Label("nto"), reportxml.ID("63809"), VerifyCPUManagerConfig) //nolint:misspell

			It("Verify Node Tuning Operator Huge Pages configuration",
				Label("nto"), reportxml.ID("60062"), VerifyHugePagesConfig) //nolint:misspell

			It("Verify System Reserved memory for user-plane-worker nodes configuration",
				Label("nto"), //nolint:misspell
				reportxml.ID("60047"), SetSystemReservedMemoryForUserPlaneNodes)
		})
}

// VerifyNROPNamespaceExists asserts namespace for Numa Resources Operator exists.
func VerifyNROPNamespaceExists(ctx SpecContext) {
	err := apiobjectshelper.VerifyNamespaceExists(APIClient, vcoreparams.NROPNamespace, time.Second)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to pull %q namespace", vcoreparams.NROPNamespace))
} // func VerifyNROPNamespaceExists (ctx SpecContext)

// VerifyNROPDeployment asserts Numa Resources Operator successfully installed.
func VerifyNROPDeployment(ctx SpecContext) {
	err := apiobjectshelper.VerifyOperatorDeployment(APIClient,
		vcoreparams.NROPSubscriptionName,
		vcoreparams.NROPDeploymentName,
		vcoreparams.NROPNamespace,
		time.Minute)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("operator deployment %s failure in the namespace %s; %v",
			vcoreparams.NROPDeploymentName, vcoreparams.NROPNamespace, err))
} // func VerifyNROPDeployment (ctx SpecContext)

// VerifyNROPCustomResources asserts Numa Resources Operator custom resource deployment.
func VerifyNROPCustomResources(ctx SpecContext) {
	glog.V(vcoreparams.VCoreLogLevel).Infof("Verify NUMAResourcesOperator %s configuration",
		vcoreparams.NROPInstanceName)

	nropCustomResource := nrop.NewBuilder(APIClient, vcoreparams.NROPInstanceName)

	if !nropCustomResource.Exists() {
		glog.V(vcoreparams.VCoreLogLevel).Infof("Create new NUMAResourcesOperator %s",
			vcoreparams.NROPInstanceName)

		var err error

		nropCustomResource, err = nropCustomResource.WithMCPSelector(VCoreConfig.WorkerLabelMap).Create()
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to create NUMAResourcesOperator %s; %v",
			vcoreparams.NROPInstanceName, err))

		glog.V(vcoreparams.VCoreLogLevel).Infof(
			"Wait for all nodes rebooting after applying NUMAResourcesOperator %s",
			vcoreparams.NROPInstanceName)

		_, err = nodes.WaitForAllNodesToReboot(
			APIClient,
			40*time.Minute,
			VCoreConfig.VCorePpLabelListOption)
		Expect(err).ToNot(HaveOccurred(),
			fmt.Sprintf("Nodes failed to reboot after applying NUMAResourcesOperator %s config; %v",
				vcoreparams.NROPInstanceName, err))

		glog.V(vcoreparams.VCoreLogLevel).Info("Wait for all clusteroperators availability after nodes reboot")

		_, err = clusteroperator.WaitForAllClusteroperatorsAvailable(APIClient, 60*time.Second)
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Error waiting for all available clusteroperators: %v", err))
	}

	glog.V(vcoreparams.VCoreLogLevel).Info("Verify NUMA Topology Manager")

	performanceKubeletConfigObj, err := mco.PullKubeletConfig(APIClient, performanceKubeletConfigName)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to get kubeletconfigs %s due to %v",
		performanceKubeletConfigName, err))

	kubeletConfig := performanceKubeletConfigObj.Object.Spec.KubeletConfig.Raw

	currentTopologyManagerPolicy :=
		unmarshalRaw[kubeletconfigv1beta1.KubeletConfiguration](kubeletConfig).TopologyManagerPolicy
	Expect(currentTopologyManagerPolicy).To(Equal(vcoreparams.TopologyConfig),
		fmt.Sprintf("incorrect topology manager policy found; expected: %s, found: %s",
			vcoreparams.TopologyConfig, currentTopologyManagerPolicy))
} // func CreatePerformanceProfile (ctx SpecContext)
