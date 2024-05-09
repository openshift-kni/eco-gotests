package vcorecommon

import (
	"context"
	"fmt"
	"time"

	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/lso"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	. "github.com/openshift-kni/eco-gotests/tests/system-tests/vcore/internal/vcoreinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/vcore/internal/vcoreparams"
)

var (
	lvdName = "auto-discover-devices"
	lvsName = "ocs-deviceset"
)

// VerifyLSONamespaceExists asserts namespace for Local Storage Operator exists.
func VerifyLSONamespaceExists(ctx SpecContext) {
	glog.V(vcoreparams.VCoreLogLevel).Infof("Verify namespace %q exists",
		vcoreparams.LSONamespace)

	err := wait.PollUntilContextTimeout(ctx, 5*time.Second, 1*time.Minute, true,
		func(ctx context.Context) (bool, error) {
			_, pullErr := namespace.Pull(APIClient, vcoreparams.LSONamespace)
			if pullErr != nil {
				glog.V(vcoreparams.VCoreLogLevel).Infof(
					fmt.Sprintf("Failed to pull in namespace %q - %v",
						vcoreparams.LSONamespace, pullErr))

				return false, pullErr
			}

			return true, nil
		})

	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to pull %q namespace", vcoreparams.LSONamespace))
} // func VerifyLSONamespaceExists (ctx SpecContext)

// VerifyLSODeployment asserts Local Storage Operator successfully installed.
func VerifyLSODeployment(ctx SpecContext) {
	glog.V(100).Infof("Confirm that LSO %s pod was deployed and running in %s namespace",
		vcoreparams.LSOInstanceNamePattern, vcoreparams.LSONamespace)

	lsoPods, err := pod.ListByNamePattern(APIClient, vcoreparams.LSOInstanceNamePattern, vcoreparams.LSONamespace)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("No %s pods were found in %s namespace due to %s",
		vcoreparams.LSOInstanceNamePattern, vcoreparams.LSONamespace, err))
	Expect(len(lsoPods)).ToNot(Equal(0), fmt.Sprintf("The list of pods %s found in namespace %s is empty",
		vcoreparams.LSOInstanceNamePattern, vcoreparams.LSONamespace))

	lsoPod := lsoPods[0]
	lsoPodName := lsoPod.Object.Name

	err = lsoPod.WaitUntilReady(time.Second)
	if err != nil {
		lsoPodLog, _ := lsoPod.GetLog(600*time.Second, vcoreparams.LSOInstanceNamePattern)
		glog.Fatalf("%s pod in %s namespace in a bad state: %s",
			lsoPodName, vcoreparams.LSONamespace, lsoPodLog)
	}

	glog.V(100).Info("Verify auto-discover CRD is created")

	lvdInstance := lso.NewLocalVolumeDiscoveryBuilder(APIClient, lvdName, vcoreparams.LSONamespace)
	Expect(lvdInstance.Exists()).To(Equal(true), fmt.Sprintf("%s auto-discover-devices not found "+
		"in %s namespace", lvdName, vcoreparams.LSONamespace))

	glog.V(100).Info("Verify localvolumeset CRD is created")

	lvsInstance := lso.NewLocalVolumeSetBuilder(APIClient, lvsName, vcoreparams.LSONamespace)
	Expect(lvsInstance.Exists()).To(Equal(true), fmt.Sprintf("%s localvolumeset not found in %s namespace",
		lvsName, vcoreparams.LSONamespace))
} // func VerifyLSODeployment (ctx SpecContext)

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
