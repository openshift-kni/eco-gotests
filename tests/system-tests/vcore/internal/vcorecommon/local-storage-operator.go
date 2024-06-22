package vcorecommon

import (
	"fmt"
	"time"

	"github.com/openshift-kni/eco-goinfra/pkg/lso"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/apiobjectshelper"
	. "github.com/openshift-kni/eco-gotests/tests/system-tests/vcore/internal/vcoreinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/vcore/internal/vcoreparams"
)

var (
	lvdName = "auto-discover-devices"
	lvsName = "ocs-deviceset"
)

// VerifyLSOSuite container that contains tests for LSO verification.
func VerifyLSOSuite() {
	Describe(
		"LSO validation",
		Label(vcoreparams.LabelVCoreOperators), func() {
			It(fmt.Sprintf("Verifies %s namespace exists", vcoreparams.LSONamespace),
				Label("lso"), VerifyLSONamespaceExists)

			It("Verify Local Storage Operator successfully installed",
				Label("lso"), reportxml.ID("59491"), VerifyLSODeployment)
		})
}

// VerifyLSONamespaceExists asserts namespace for Local Storage Operator exists.
func VerifyLSONamespaceExists(ctx SpecContext) {
	err := apiobjectshelper.VerifyNamespaceExists(APIClient, vcoreparams.LSONamespace, time.Second)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to pull %q namespace", vcoreparams.LSONamespace))
} // func VerifyLSONamespaceExists (ctx SpecContext)

// VerifyLSODeployment asserts Local Storage Operator successfully installed.
func VerifyLSODeployment(ctx SpecContext) {
	err := apiobjectshelper.VerifyOperatorDeployment(APIClient,
		vcoreparams.LSOName,
		vcoreparams.LSOName,
		vcoreparams.LSONamespace,
		time.Minute)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("operator deployment %s failure in the namespace %s; %v",
			vcoreparams.LSOName, vcoreparams.LSONamespace, err))

	glog.V(100).Infof("Confirm that LSO %s pod was deployed and running in %s namespace",
		vcoreparams.LSOName, vcoreparams.LSONamespace)

	lsoPods, err := pod.ListByNamePattern(APIClient, vcoreparams.LSOName, vcoreparams.LSONamespace)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("No %s pods were found in %s namespace due to %s",
		vcoreparams.LSOName, vcoreparams.LSONamespace, err))
	Expect(len(lsoPods)).ToNot(Equal(0), fmt.Sprintf("The list of pods %s found in namespace %s is empty",
		vcoreparams.LSOName, vcoreparams.LSONamespace))

	lsoPod := lsoPods[0]
	lsoPodName := lsoPod.Object.Name

	err = lsoPod.WaitUntilReady(time.Second)
	if err != nil {
		lsoPodLog, _ := lsoPod.GetLog(600*time.Second, vcoreparams.LSOName)
		glog.Fatalf("%s pod in %s namespace in a bad state: %s",
			lsoPodName, vcoreparams.LSONamespace, lsoPodLog)
	}

	glog.V(vcoreparams.VCoreLogLevel).Info("Verify auto-discover CRD is created")

	lvdInstance := lso.NewLocalVolumeDiscoveryBuilder(APIClient, lvdName, vcoreparams.LSONamespace)
	Expect(lvdInstance.Exists()).To(Equal(true), fmt.Sprintf("%s auto-discover-devices not found "+
		"in %s namespace", lvdName, vcoreparams.LSONamespace))

	glog.V(vcoreparams.VCoreLogLevel).Info("Verify localvolumeset CRD is created")

	lvsInstance := lso.NewLocalVolumeSetBuilder(APIClient, lvsName, vcoreparams.LSONamespace)
	Expect(lvsInstance.Exists()).To(Equal(true), fmt.Sprintf("%s localvolumeset not found in %s namespace",
		lvsName, vcoreparams.LSONamespace))
} // func VerifyLSODeployment (ctx SpecContext)
