package mcm

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/deployment"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"github.com/openshift-kni/eco-goinfra/pkg/serviceaccount"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/internal/define"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/internal/kmmparams"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	. "github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/internal/kmminittools"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/mcm/internal/tsparams"
	. "github.com/openshift-kni/eco-gotests/tests/internal/inittools"
	"github.com/openshift-kni/eco-gotests/tests/internal/reporter"

	_ "github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/mcm/tests"
)

var (
	prereqName = "kmm-tests-executor"
)

var _, currentFile, _, _ = runtime.Caller(0)

func TestManagedClusterModules(t *testing.T) {
	_, reporterConfig := GinkgoConfiguration()
	reporterConfig.JUnitReport = GeneralConfig.GetJunitReportPath(currentFile)

	RegisterFailHandler(Fail)
	RunSpecs(t, "KMM-HUB", Label(tsparams.Labels...), reporterConfig)
}

var _ = ReportAfterSuite("", func(report Report) {
	reportxml.Create(
		report, GeneralConfig.GetReportPath(), GeneralConfig.TCPrefix)
})

var _ = JustAfterEach(func() {
	reporter.ReportIfFailed(
		CurrentSpecReport(), currentFile, tsparams.ReporterNamespacesToDump, tsparams.ReporterCRDsToDump, clients.SetScheme)
})

var _ = BeforeSuite(func() {
	By("Prepare environment spoke for KMM-HUB tests execution")
	if ModulesConfig.SpokeClusterName == "" || ModulesConfig.SpokeKubeConfig == "" {
		Skip("Skipping test. No Spoke environment variables defined.")
	}

	By("Create helper ServiceAccount")
	svcAccount, err := serviceaccount.
		NewBuilder(ModulesConfig.SpokeAPIClient, prereqName, kmmparams.KmmOperatorNamespace).Create()
	Expect(err).ToNot(HaveOccurred(), "error creating serviceaccount")

	By("Create helper ClusterRoleBinding")
	crb := define.ModuleCRB(*svcAccount, prereqName)
	_, err = crb.Create()
	Expect(err).ToNot(HaveOccurred(), "error creating clusterrolebinding")

	By("Create helper Deployments")
	nodeList, err := nodes.List(
		ModulesConfig.SpokeAPIClient, metav1.ListOptions{LabelSelector: labels.Set(GeneralConfig.WorkerLabelMap).String()})

	if err != nil {
		Skip(fmt.Sprintf("Error listing worker nodes. Got error: '%v'", err))
	}
	for _, node := range nodeList {
		glog.V(kmmparams.KmmLogLevel).Infof("Creating privileged deployment on node '%v'", node.Object.Name)

		deploymentName := fmt.Sprintf("%s-%s", kmmparams.KmmTestHelperLabelName, node.Object.Name)
		containerCfg, _ := pod.NewContainerBuilder("test", kmmparams.DTKImage,
			[]string{"/bin/bash", "-c", "sleep INF"}).
			WithSecurityContext(kmmparams.PrivilegedSC).GetContainerCfg()

		deploymentCfg := deployment.NewBuilder(ModulesConfig.SpokeAPIClient, deploymentName, kmmparams.KmmOperatorNamespace,
			map[string]string{kmmparams.KmmTestHelperLabelName: ""}, *containerCfg)
		deploymentCfg.WithLabel(kmmparams.KmmTestHelperLabelName, "").
			WithNodeSelector(map[string]string{"kubernetes.io/hostname": node.Object.Name}).
			WithServiceAccountName("kmm-operator-module-loader")

		_, err = deploymentCfg.CreateAndWaitUntilReady(10 * time.Minute)

		if err != nil {
			Skip(fmt.Sprintf("Could not create deploymentCfg on %s. Got error : %v", node.Object.Name, err))
		}

	}
})

var _ = AfterSuite(func() {
	By("Cleanup environment after KMM tests execution")
	glog.V(kmmparams.KmmLogLevel).Infof("Deleting test deployments")

	By("Delete helper deployments")
	testDeployments, err := deployment.List(ModulesConfig.SpokeAPIClient,
		kmmparams.KmmOperatorNamespace, metav1.ListOptions{})

	if err != nil {
		Fail(fmt.Sprintf("Error cleaning up environment. Got error: %v", err))
	}

	for _, deploymentObj := range testDeployments {
		glog.V(kmmparams.KmmLogLevel).Infof("Deployment: '%s'\n", deploymentObj.Object.Name)
		if strings.Contains(deploymentObj.Object.Name, kmmparams.KmmTestHelperLabelName) {
			glog.V(kmmparams.KmmLogLevel).Infof("Deleting deployment: '%s'\n", deploymentObj.Object.Name)
			err = deploymentObj.DeleteAndWait(time.Minute)

			Expect(err).ToNot(HaveOccurred(), "error deleting helper deployment")
		}
	}

	By("Delete helper clusterrolebinding")
	svcAccount := serviceaccount.NewBuilder(ModulesConfig.SpokeAPIClient, prereqName, kmmparams.KmmOperatorNamespace)
	svcAccount.Exists()

	crb := define.ModuleCRB(*svcAccount, prereqName)
	err = crb.Delete()
	Expect(err).ToNot(HaveOccurred(), "error deleting helper clusterrolebinding")

	By("Delete helper service account")
	err = svcAccount.Delete()
	Expect(err).ToNot(HaveOccurred(), "error deleting helper serviceaccount")
})
