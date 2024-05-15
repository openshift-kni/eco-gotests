package vcorecommon

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/openshift-kni/eco-goinfra/pkg/console"
	"github.com/openshift-kni/eco-goinfra/pkg/mco"
	"github.com/openshift-kni/eco-goinfra/pkg/olm"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/await"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/csv"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/ocpcli"

	clusterlogging "github.com/openshift-kni/eco-goinfra/pkg/clusterlogging"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	. "github.com/openshift-kni/eco-gotests/tests/system-tests/vcore/internal/vcoreinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/vcore/internal/vcoreparams"
)

var (
	eskCdmPodNamePattern = "elasticsearch-cdm"
	eskImPodsNamePattern = []string{
		"elasticsearch-im-app",
		"elasticsearch-im-audit",
		"elasticsearch-im-infra"}
	kibanaPodNamePattern = "kibana"
	esoInstances         = "3"
)

// VerifyESKNamespaceExists asserts namespace for ElasticSearch Operator exists.
func VerifyESKNamespaceExists(ctx SpecContext) {
	glog.V(vcoreparams.VCoreLogLevel).Infof("Verify namespace %q exists",
		vcoreparams.ESKNamespace)

	err := wait.PollUntilContextTimeout(ctx, 5*time.Second, 1*time.Minute, true,
		func(ctx context.Context) (bool, error) {
			_, pullErr := namespace.Pull(APIClient, vcoreparams.ESKNamespace)
			if pullErr != nil {
				glog.V(vcoreparams.VCoreLogLevel).Infof(
					fmt.Sprintf("Failed to pull in namespace %q - %v",
						vcoreparams.ESKNamespace, pullErr))

				return false, pullErr
			}

			return true, nil
		})

	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to pull %q namespace", vcoreparams.ESKNamespace))
} // func VerifyESKNamespaceExists (ctx SpecContext)

// VerifyCLONamespaceExists asserts namespace for ClusterLogging Operator exists.
func VerifyCLONamespaceExists(ctx SpecContext) {
	glog.V(vcoreparams.VCoreLogLevel).Infof("Verify namespace %q exists",
		vcoreparams.CLONamespace)

	err := wait.PollUntilContextTimeout(ctx, 5*time.Second, 1*time.Minute, true,
		func(ctx context.Context) (bool, error) {
			_, pullErr := namespace.Pull(APIClient, vcoreparams.CLONamespace)
			if pullErr != nil {
				glog.V(vcoreparams.VCoreLogLevel).Infof(
					fmt.Sprintf("Failed to pull in namespace %q - %v",
						vcoreparams.CLONamespace, pullErr))

				return false, pullErr
			}

			return true, nil
		})
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to pull %q namespace", vcoreparams.CLONamespace))
} // func VerifyCLONamespaceExists (ctx SpecContext)

// VerifyESKDeployment asserts ElasticSearch Operator successfully installed.
func VerifyESKDeployment(ctx SpecContext) {
	glog.V(100).Infof("Confirm that ElasticSearch operator %s csv was deployed and running in %s namespace",
		vcoreparams.ESKOperatorName, vcoreparams.ESKNamespace)

	eskCSVName, err := csv.GetCurrentCSVNameFromSubscription(APIClient,
		vcoreparams.ESKOperatorName,
		vcoreparams.ESKNamespace)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to get ElasticSearch %s csv name from the %s namespace",
		vcoreparams.ESKOperatorName, vcoreparams.ESKNamespace))

	eskBCSVObj, err := olm.PullClusterServiceVersion(APIClient, eskCSVName, vcoreparams.ESKNamespace)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to pull %q csv from the %s namespace",
		eskCSVName, vcoreparams.ESKNamespace))

	isSuccessful, err := eskBCSVObj.IsSuccessful()
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to verify ElasticSearch csv %s in the namespace %s status",
			eskCSVName, vcoreparams.ESKNamespace))
	Expect(isSuccessful).To(Equal(true),
		fmt.Sprintf("Failed to deploy ElasticSearch operator; the csv %s in the namespace %s status %v",
			eskCSVName, vcoreparams.ESKNamespace, isSuccessful))

	glog.V(100).Infof("Confirm that %s deployment for the ElasticSearch operator is running in %s namespace",
		vcoreparams.ESKOperatorName, vcoreparams.CLONamespace)

	err = await.WaitUntilDeploymentReady(APIClient,
		vcoreparams.ESKOperatorName,
		vcoreparams.ESKNamespace,
		5*time.Second)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("%s deployment not found in %s namespace; %v",
		vcoreparams.ESKOperatorName, vcoreparams.ESKNamespace, err))
} // func VerifyESKDeployment (ctx SpecContext)

// VerifyCLODeployment asserts ClusterLogging Operator successfully installed.
func VerifyCLODeployment(ctx SpecContext) {
	glog.V(100).Infof("Confirm that ClusterLogging operator %s csv was deployed and running in %s namespace",
		vcoreparams.CLOOperatorName, vcoreparams.CLONamespace)

	cloCSVName, err := csv.GetCurrentCSVNameFromSubscription(APIClient,
		vcoreparams.CLOOperatorName,
		vcoreparams.CLONamespace)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to get ClusterLogging %s csv name from the %s namespace",
		vcoreparams.CLOOperatorName, vcoreparams.CLONamespace))

	cloBCSVObj, err := olm.PullClusterServiceVersion(APIClient, cloCSVName, vcoreparams.CLONamespace)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to pull %q csv from the %s namespace",
		cloCSVName, vcoreparams.CLONamespace))

	isSuccessful, err := cloBCSVObj.IsSuccessful()
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to verify ClusterLogging csv %s in the namespace %s status",
			cloCSVName, vcoreparams.CLONamespace))
	Expect(isSuccessful).To(Equal(true),
		fmt.Sprintf("Failed to deploy ClusterLogging operator; the csv %s in the namespace %s status %v",
			cloCSVName, vcoreparams.CLONamespace, isSuccessful))
} // func VerifyCLODeployment (ctx SpecContext)

// CreateCLOInstance asserts ClusterLogging instance can be created and running.
//
//nolint:funlen
func CreateCLOInstance(ctx SpecContext) {
	glog.V(100).Infof("Verify Cluster Logging instance %s is running in namespace %s",
		vcoreparams.CLOInstanceName, vcoreparams.CLONamespace)

	cloInstance := clusterlogging.NewBuilder(APIClient, vcoreparams.CLOInstanceName, vcoreparams.CLONamespace)

	if cloInstance.Exists() {
		err := cloInstance.Delete()
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to delete ClusterLogging %s csv name from the %s namespace",
			vcoreparams.CLOInstanceName, vcoreparams.CLONamespace))
	}

	glog.V(100).Infof("Create new Cluster Logging instance %s in namespace %s",
		vcoreparams.CLOInstanceName, vcoreparams.CLONamespace)

	homeDir, err := os.UserHomeDir()
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to get home directory due to %s", err))

	clusterLoggingTemplateName := "clo-instance.yaml"
	varsToReplace := make(map[string]interface{})
	varsToReplace["ClusterLoggingName"] = vcoreparams.CLOInstanceName
	varsToReplace["ClusterLoggingNamespace"] = vcoreparams.CLONamespace
	varsToReplace["NodeCount"] = esoInstances

	destinationDirectoryPath := filepath.Join(homeDir, vcoreparams.ConfigurationFolderName)

	workingDir, err := os.Getwd()
	Expect(err).ToNot(HaveOccurred(), err)

	templateDir := filepath.Join(workingDir, vcoreparams.TemplateFilesFolder)

	err = ocpcli.ApplyConfigFile(
		templateDir,
		clusterLoggingTemplateName,
		destinationDirectoryPath,
		clusterLoggingTemplateName,
		varsToReplace)
	Expect(err).To(BeNil(), fmt.Sprint(err))

	Expect(cloInstance.Exists()).To(Equal(true),
		fmt.Sprintf("failed to create ClusterLogging %s in namespace %s",
			vcoreparams.CLOInstanceName, vcoreparams.CLONamespace))

	glog.V(100).Infof("Confirm that %s deployment for the ClusterLogging operator is running in %s namespace",
		vcoreparams.CLOOperatorName, vcoreparams.CLONamespace)

	err = await.WaitUntilDeploymentReady(APIClient,
		vcoreparams.CLOOperatorName,
		vcoreparams.CLONamespace,
		5*time.Second)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("%s deployment not found in %s namespace; %v",
		vcoreparams.CLOOperatorName, vcoreparams.CLONamespace, err))

	glog.V(100).Infof("Check clusterLogging pods")

	podsList, err := pod.ListByNamePattern(APIClient, vcoreparams.CLOOperatorName, vcoreparams.CLONamespace)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("pod %s not found in namespace %s; %v",
		vcoreparams.CLOOperatorName, vcoreparams.CLONamespace, err))
	Expect(len(podsList)).To(Equal(1), fmt.Sprintf("pod %s not found in namespace %s",
		vcoreparams.CLOOperatorName, vcoreparams.CLONamespace))

	err = podsList[0].WaitUntilReady(time.Second)
	Expect(err).ToNot(HaveOccurred(), "pod %s in namespace %s is not ready",
		podsList[0].Definition.Name, vcoreparams.CLONamespace)

	err = podsList[0].WaitUntilRunning(5 * time.Second)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("pod %s in namespace %s failed to run; %v",
		podsList[0].Definition.Name, vcoreparams.CLONamespace, err))

	odfMcp := mco.NewMCPBuilder(APIClient, VCoreConfig.OdfLabel)
	if odfMcp.IsInCondition("Updated") {
		podsList, err := pod.ListByNamePattern(APIClient, eskCdmPodNamePattern, vcoreparams.CLONamespace)
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("pod %s not found in namespace %s; %v",
			eskCdmPodNamePattern, vcoreparams.CLONamespace, err))
		Expect(len(podsList)).To(Equal(3), fmt.Sprintf("not all pods %s found in namespace %s: %v",
			eskCdmPodNamePattern, vcoreparams.CLONamespace, podsList))

		for _, pod := range podsList {
			err = pod.WaitUntilReady(time.Second)
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("pod %s in namespace %s is not ready",
				pod.Definition.Name, vcoreparams.CLONamespace))

			err = pod.WaitUntilRunning(5 * time.Second)
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("pod %s in namespace %s failed to run; %v",
				pod.Definition.Name, vcoreparams.CLONamespace, err))
		}

		for _, eskImPodNamePattern := range eskImPodsNamePattern {
			podsList, err = pod.ListByNamePattern(APIClient, eskImPodNamePattern, vcoreparams.CLONamespace)
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("pod %s not found in namespace %s; %v",
				eskCdmPodNamePattern, vcoreparams.CLONamespace, err))
			Expect(len(podsList)).To(Equal(1), fmt.Sprintf("pods %s not found in namespace %s",
				eskCdmPodNamePattern, vcoreparams.CLONamespace))

			err = podsList[0].WaitUntilReady(time.Second)
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("pod %s in namespace %s is not ready",
				podsList[0].Definition.Name, vcoreparams.CLONamespace))

			err = podsList[0].WaitUntilRunning(5 * time.Second)
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("pod %s in namespace %s failed to run; %v",
				podsList[0].Definition.Name, vcoreparams.CLONamespace, err))
		}
	}

	podsList, err = pod.ListByNamePattern(APIClient, kibanaPodNamePattern, vcoreparams.CLONamespace)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("pod %s not found in namespace %s; %v",
		kibanaPodNamePattern, vcoreparams.CLONamespace, err))
	Expect(len(podsList)).To(Equal(1), fmt.Sprintf("pod %s not found in namespace %s",
		kibanaPodNamePattern, vcoreparams.CLONamespace))

	err = podsList[0].WaitUntilReady(time.Second)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("pod %s in namespace %s is not ready",
		podsList[0].Definition.Name, vcoreparams.CLONamespace))

	err = podsList[0].WaitUntilRunning(5 * time.Second)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("pod %s in namespace %s failed to run; %v",
		podsList[0].Definition.Name, vcoreparams.CLONamespace, err))

	glog.V(100).Infof("Enable logging-view-plugin")

	consoleoperatorObj, err := console.PullConsoleOperator(APIClient, "cluster")
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("consoleoperator is unavailable: %v", err))

	_, err = consoleoperatorObj.WithPlugins([]string{"logging-view-plugin"}, false).Update()
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to enable logging-view-pluggin due to %v", err))

	eskInstance := clusterlogging.NewElasticsearchBuilder(APIClient,
		vcoreparams.ESKInstanceName,
		vcoreparams.CLONamespace)
	Expect(eskInstance.Exists()).To(Equal(true), fmt.Sprintf("Failed to create ElasticSearch %s "+
		"instance in %s namespace",
		vcoreparams.ESKOperatorName, vcoreparams.CLONamespace))

	_, err = consoleoperatorObj.WithPlugins([]string{"logging-view-plugin"}, false).Update()
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to enable logging-view-pluggin due to %v", err))
} // func VerifyCLODeployment (ctx SpecContext)

// VerifyESKAndCLOSuite container that contains tests for ElasticSearch and ClusterLogging verification.
func VerifyESKAndCLOSuite() {
	Describe(
		"ElasticSearch and Cluster Logging validation",
		Label(vcoreparams.LabelVCoreOperators), func() {
			It(fmt.Sprintf("Verifies %s namespace exists", vcoreparams.ESKNamespace),
				Label("clo"), VerifyESKNamespaceExists)

			It(fmt.Sprintf("Verifies %s namespace exists", vcoreparams.CLONamespace),
				Label("clo"), VerifyCLONamespaceExists)

			It("Verify ElasticSearch Operator successfully installed",
				Label("clo"), reportxml.ID("59493"), VerifyESKDeployment)

			It("Verify ClusterLogging Operator successfully installed",
				Label("clo"), reportxml.ID("73678"), VerifyCLODeployment)
		})
}
