package vcorecommon

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"

	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/apiobjectshelper"

	"github.com/openshift-kni/eco-goinfra/pkg/console"
	"github.com/openshift-kni/eco-goinfra/pkg/mco"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/ocpcli"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	clusterlogging "github.com/openshift-kni/eco-goinfra/pkg/clusterlogging"
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

			It(fmt.Sprintf("Verify Cluster Logging instance %s is running in namespace %s",
				vcoreparams.CLOInstanceName, vcoreparams.CLONamespace),
				Label("clo"), reportxml.ID("59494"), CreateCLOInstance)

			It("Verify ClusterLogging Operator successfully installed",
				Label("clo"), reportxml.ID("73678"), VerifyCLODeployment)
		})
}

// VerifyESKNamespaceExists asserts namespace for ElasticSearch Operator exists.
func VerifyESKNamespaceExists(ctx SpecContext) {
	err := apiobjectshelper.VerifyNamespaceExists(APIClient, vcoreparams.ESKNamespace, time.Second)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to pull %q namespace", vcoreparams.ESKNamespace))
} // func VerifyESKNamespaceExists (ctx SpecContext)

// VerifyCLONamespaceExists asserts namespace for ClusterLogging Operator exists.
func VerifyCLONamespaceExists(ctx SpecContext) {
	err := apiobjectshelper.VerifyNamespaceExists(APIClient, vcoreparams.CLONamespace, time.Second)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to pull %q namespace", vcoreparams.CLONamespace))
} // func VerifyCLONamespaceExists (ctx SpecContext)

// VerifyESKDeployment asserts ElasticSearch Operator successfully installed.
func VerifyESKDeployment(ctx SpecContext) {
	err := apiobjectshelper.VerifyOperatorDeployment(APIClient,
		vcoreparams.ESKOperatorName,
		vcoreparams.ESKOperatorName,
		vcoreparams.ESKNamespace,
		time.Minute)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("ElasticSearch operator deployment %s failure in the namespace %s; %v",
			vcoreparams.ESKOperatorName, vcoreparams.ESKNamespace, err))
} // func VerifyESKDeployment (ctx SpecContext)

// VerifyCLODeployment asserts ClusterLogging Operator successfully installed.
func VerifyCLODeployment(ctx SpecContext) {
	err := apiobjectshelper.VerifyOperatorDeployment(APIClient,
		vcoreparams.CLOName,
		vcoreparams.CLODeploymentName,
		vcoreparams.CLONamespace,
		time.Minute)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("operator deployment %s failure in the namespace %s; %v",
			vcoreparams.CLOName, vcoreparams.CLONamespace, err))
} // func VerifyCLODeployment (ctx SpecContext)

// CreateCLOInstance asserts ClusterLogging instance can be created and running.
//
//nolint:funlen
func CreateCLOInstance(ctx SpecContext) {
	glog.V(vcoreparams.VCoreLogLevel).Infof("Verify Cluster Logging instance %s is running in namespace %s",
		vcoreparams.CLOInstanceName, vcoreparams.CLONamespace)

	cloInstance := clusterlogging.NewBuilder(APIClient, vcoreparams.CLOInstanceName, vcoreparams.CLONamespace)

	if cloInstance.Exists() {
		err := cloInstance.Delete()
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to delete ClusterLogging %s csv name from the %s namespace",
			vcoreparams.CLOInstanceName, vcoreparams.CLONamespace))
	}

	glog.V(vcoreparams.VCoreLogLevel).Infof("Create new Cluster Logging instance %s in namespace %s",
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

	err = ocpcli.ApplyConfig(
		templateDir,
		clusterLoggingTemplateName,
		destinationDirectoryPath,
		clusterLoggingTemplateName,
		varsToReplace)
	Expect(err).To(BeNil(), fmt.Sprint(err))

	Expect(cloInstance.Exists()).To(Equal(true),
		fmt.Sprintf("failed to create ClusterLogging %s in namespace %s",
			vcoreparams.CLOInstanceName, vcoreparams.CLONamespace))

	glog.V(100).Infof("Check clusterLogging pods")

	podsList, err := pod.ListByNamePattern(APIClient, vcoreparams.CLOName, vcoreparams.CLONamespace)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("pod %s not found in namespace %s; %v",
		vcoreparams.CLOName, vcoreparams.CLONamespace, err))
	Expect(len(podsList)).To(Equal(1), fmt.Sprintf("pod %s not found in namespace %s",
		vcoreparams.CLOName, vcoreparams.CLONamespace))

	err = podsList[0].WaitUntilReady(time.Second)
	Expect(err).ToNot(HaveOccurred(), "pod %s in namespace %s is not ready",
		podsList[0].Definition.Name, vcoreparams.CLONamespace)

	err = podsList[0].WaitUntilRunning(5 * time.Second)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("pod %s in namespace %s failed to run; %v",
		podsList[0].Definition.Name, vcoreparams.CLONamespace, err))

	odfMcp := mco.NewMCPBuilder(APIClient, VCoreConfig.OdfMCPName)
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

	glog.V(vcoreparams.VCoreLogLevel).Infof("Enable logging-view-plugin")

	consoleoperatorObj, err := console.PullConsoleOperator(APIClient, "cluster")
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("consoleoperator is unavailable: %v", err))

	_, err = consoleoperatorObj.WithPlugins([]string{"logging-view-plugin"}, false).Update()
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to enable logging-view-pluggin due to %v", err))

	// eskInstance := clusterlogging.NewElasticsearchBuilder(APIClient,
	//   vcoreparams.ESKInstanceName,
	//	 vcoreparams.CLONamespace)
	// Expect(eskInstance.Exists()).To(Equal(true), fmt.Sprintf("Failed to create ElasticSearch %s "+
	//	"instance in %s namespace",
	//	vcoreparams.ESKOperatorName, vcoreparams.CLONamespace))

	_, err = consoleoperatorObj.WithPlugins([]string{"logging-view-plugin"}, false).Update()
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to enable logging-view-pluggin due to %v", err))
} // func CreateCLOInstance (ctx SpecContext)
