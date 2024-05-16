package vcorecommon

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/csv"

	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/await"

	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/metallb"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/olm"

	. "github.com/openshift-kni/eco-gotests/tests/system-tests/vcore/internal/vcoreinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/vcore/internal/vcoreparams"
)

// VerifyMetalLBNamespaceExists asserts namespace for NMState operator exists.
func VerifyMetalLBNamespaceExists(ctx SpecContext) {
	glog.V(vcoreparams.VCoreLogLevel).Infof("Verify namespace %q exists",
		vcoreparams.MetalLBOperatorNamespace)

	err := wait.PollUntilContextTimeout(ctx, 5*time.Second, 1*time.Minute, true,
		func(ctx context.Context) (bool, error) {
			_, pullErr := namespace.Pull(APIClient, vcoreparams.MetalLBOperatorNamespace)
			if pullErr != nil {
				glog.V(vcoreparams.VCoreLogLevel).Infof(
					fmt.Sprintf("Failed to pull in namespace %q - %v",
						vcoreparams.MetalLBOperatorNamespace, pullErr))

				return false, pullErr
			}

			return true, nil
		})

	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to pull %q namespace", vcoreparams.MetalLBOperatorNamespace))
} // func VerifyMetalLBNamespaceExists (ctx SpecContext)

// VerifyMetalLBOperatorDeployment asserts MetalLB operator successfully installed.
func VerifyMetalLBOperatorDeployment(ctx SpecContext) {
	glog.V(100).Infof("Confirm that the %s operator is available", vcoreparams.MetalLBOperatorName)
	_, err := olm.PullPackageManifest(APIClient,
		vcoreparams.MetalLBOperatorName,
		vcoreparams.OperatorsNamespace)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("%s operator not found deployed in %s namespace; %v",
		vcoreparams.MetalLBOperatorName, vcoreparams.OperatorsNamespace, err))

	glog.V(100).Infof("Confirm the install plan is in the %s namespace", vcoreparams.MetalLBOperatorNamespace)
	installPlanList, err := olm.ListInstallPlan(APIClient, vcoreparams.MetalLBOperatorNamespace)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("metalLB installPlan not found in %s namespace; %v",
		vcoreparams.MetalLBOperatorNamespace, err))
	Expect(len(installPlanList)).To(Equal(1),
		fmt.Sprintf("metalLB installPlan not found in %s namespace; found: %v",
			vcoreparams.MetalLBOperatorNamespace, installPlanList))

	glog.V(100).Infof("Confirm that the deployment for the metalLB operator is running in %s namespace",
		vcoreparams.MetalLBOperatorNamespace)

	metalLBCSVName, err := csv.GetCurrentCSVNameFromSubscription(APIClient,
		vcoreparams.MetalLBSubscriptionName,
		vcoreparams.MetalLBOperatorNamespace)

	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to get metalLB %s csv name from the %s namespace",
		vcoreparams.MetalLBOperatorName, vcoreparams.MetalLBOperatorNamespace))

	metalLBCSVObj, err := olm.PullClusterServiceVersion(APIClient, metalLBCSVName, vcoreparams.MetalLBOperatorNamespace)

	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to pull %q csv from the %s namespace",
		metalLBCSVName, vcoreparams.MetalLBOperatorNamespace))

	isSuccessful, err := metalLBCSVObj.IsSuccessful()

	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to verify metalLB csv %s in the namespace %s status",
			metalLBCSVName, vcoreparams.MetalLBOperatorNamespace))
	Expect(isSuccessful).To(Equal(true),
		fmt.Sprintf("Failed to deploy metalLB operator; the csv %s in the namespace %s status %v",
			metalLBCSVName, vcoreparams.MetalLBOperatorNamespace, isSuccessful))

	glog.V(100).Infof("Create a single instance of a metalLB custom resource in %s namespace",
		vcoreparams.MetalLBOperatorNamespace)

	metallbInstance := metallb.NewBuilder(APIClient,
		vcoreparams.MetalLBInstanceName,
		vcoreparams.MetalLBOperatorNamespace,
		VCoreConfig.WorkerLabelMap)

	if !metallbInstance.Exists() {
		metallbInstance, err = metallbInstance.Create()
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to create custom %s metallb instance in %s namespace; "+
			"%v", vcoreparams.MetalLBInstanceName, vcoreparams.MetalLBOperatorNamespace, err))

		glog.V(100).Infof("Confirm that %s deployment for the MetalLB operator is running in %s namespace",
			vcoreparams.MetalLBOperatorDeploymentName, vcoreparams.MetalLBOperatorNamespace)

		err = await.WaitUntilDeploymentReady(APIClient,
			vcoreparams.MetalLBOperatorDeploymentName,
			vcoreparams.MetalLBOperatorNamespace,
			5*time.Second)
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("%s deployment not found in %s namespace; %v",
			vcoreparams.MetalLBOperatorDeploymentName, vcoreparams.MetalLBOperatorNamespace, err))
		Expect(metallbInstance.Exists()).To(Equal(true), fmt.Sprintf("Failed to create custom %s metallb "+
			"instance in %s namespace",
			vcoreparams.MetalLBInstanceName, vcoreparams.MetalLBOperatorNamespace))

		glog.V(100).Info("Check that the daemon set for the speaker is running")
		time.Sleep(5 * time.Second)
	}

	err = await.WaitUntilNewMetalLbDaemonSetIsRunning(APIClient, 5*time.Minute)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("daemonset for %s deployment speaker not found in %s namespace; %v",
		vcoreparams.MetalLBOperatorDeploymentName, vcoreparams.MetalLBOperatorNamespace, err))
} // func VerifyMetalLBOperatorDeployment (ctx SpecContext)

// VerifyMetaLBSuite container that contains tests for MetalLB verification.
func VerifyMetaLBSuite() {
	Describe(
		"NMState validation",
		Label(vcoreparams.LabelVCoreOperators), func() {
			BeforeAll(func() {
				By(fmt.Sprintf("Asserting %s folder exists", vcoreparams.ConfigurationFolderName))

				homeDir, err := os.UserHomeDir()
				Expect(err).To(BeNil(), fmt.Sprint(err))

				vcoreConfigsFolder := filepath.Join(homeDir, vcoreparams.ConfigurationFolderName)

				glog.V(100).Infof("samsungConfigsFolder: %s", vcoreConfigsFolder)

				if err := os.Mkdir(vcoreConfigsFolder, 0755); os.IsExist(err) {
					glog.V(100).Infof("%s folder already exists", vcoreConfigsFolder)
				}
			})
			It(fmt.Sprintf("Verifies %s namespace exists", vcoreparams.MetalLBOperatorNamespace),
				Label("metallb"), VerifyMetalLBNamespaceExists)

			It("Verify MetalLB operator successfully installed",
				Label("metallb"), reportxml.ID("60036"), VerifyMetalLBOperatorDeployment)
		})
}
