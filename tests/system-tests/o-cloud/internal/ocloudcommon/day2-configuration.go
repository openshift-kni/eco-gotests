package ocloudcommon

import (
	"fmt"
	"os"

	"sync"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/olm"
	"github.com/openshift-kni/eco-goinfra/pkg/oran"

	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/csv"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/shell"
	. "github.com/openshift-kni/eco-gotests/tests/system-tests/o-cloud/internal/ocloudinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/o-cloud/internal/ocloudparams"
)

// VerifySuccessfulOperatorUpgrade verifies the test case of the successful upgrade of the operators in all
// the SNOs.
func VerifySuccessfulOperatorUpgrade(ctx SpecContext) {
	downgradeOperatorImages()

	pr1 := VerifyProvisionSnoCluster(
		ocloudparams.TemplateName,
		ocloudparams.TemplateVersion6,
		ocloudparams.NodeClusterName1,
		ocloudparams.OCloudSiteId,
		ocloudparams.PolicyTemplateParameters,
		ocloudparams.ClusterInstanceParameters1)

	pr2 := VerifyProvisionSnoCluster(
		ocloudparams.TemplateName,
		ocloudparams.TemplateVersion6,
		ocloudparams.NodeClusterName2,
		ocloudparams.OCloudSiteId,
		ocloudparams.PolicyTemplateParameters,
		ocloudparams.ClusterInstanceParameters2)

	node1, nodePool1, ns1, ci1 := VerifyAndRetrieveAssociatedCRsForAI(pr1.Object.Name, ocloudparams.ClusterName1, ctx)
	node2, nodePool2, ns2, ci2 := VerifyAndRetrieveAssociatedCRsForAI(pr2.Object.Name, ocloudparams.ClusterName2, ctx)

	VerifyAllPoliciesInNamespaceAreCompliant(ns1.Object.Name, ctx, nil, nil)
	VerifyAllPoliciesInNamespaceAreCompliant(ns2.Object.Name, ctx, nil, nil)

	pr1, err := oran.PullPR(HubAPIClient, pr1.Object.Name)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to retrieve PR %s", pr1.Object.Name))

	VerifyProvisioningRequestIsFulfilled(pr1)
	glog.V(ocloudparams.OCloudLogLevel).Infof("Provisioning request %s is fulfilled", pr1.Object.Name)

	pr2, err = oran.PullPR(HubAPIClient, pr2.Object.Name)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to retrieve PR %s", pr2.Object.Name))

	VerifyProvisioningRequestIsFulfilled(pr2)
	glog.V(ocloudparams.OCloudLogLevel).Infof("Provisioning request %s is fulfilled", pr2.Object.Name)

	sno1ApiClient := CreateSnoApiClient(ocloudparams.ClusterName1)
	sno2ApiClient := CreateSnoApiClient(ocloudparams.ClusterName2)

	verifyPtpOperatorVersionInSno(
		sno1ApiClient,
		ocloudparams.PTPVersionMajorOld,
		ocloudparams.PTPVersionMinorOld,
		ocloudparams.PTPVersionPatchOld,
		ocloudparams.PTPVersionPrereleaseOld)

	verifyPtpOperatorVersionInSno(
		sno2ApiClient,
		ocloudparams.PTPVersionMajorOld,
		ocloudparams.PTPVersionMinorOld,
		ocloudparams.PTPVersionPatchOld,
		ocloudparams.PTPVersionPrereleaseOld)

	upgradeOperatorImages()

	var wg1 sync.WaitGroup
	var mu1 sync.Mutex
	wg1.Add(2)
	go VerifyPoliciesAreNotCompliant(pr1, ocloudparams.ClusterName1, ctx, &wg1, &mu1)
	go VerifyPoliciesAreNotCompliant(pr2, ocloudparams.ClusterName2, ctx, &wg1, &mu1)
	wg1.Wait()

	VerifyAllPoliciesInNamespaceAreCompliant(ns1.Object.Name, ctx, nil, nil)
	VerifyAllPoliciesInNamespaceAreCompliant(ns2.Object.Name, ctx, nil, nil)

	verifyPtpOperatorVersionInSno(
		sno1ApiClient,
		ocloudparams.PTPVersionMajorNew,
		ocloudparams.PTPVersionMinorNew,
		ocloudparams.PTPVersionPatchNew,
		ocloudparams.PTPVersionPrereleaseNew)

	verifyPtpOperatorVersionInSno(
		sno2ApiClient,
		ocloudparams.PTPVersionMajorNew,
		ocloudparams.PTPVersionMinorNew,
		ocloudparams.PTPVersionPatchNew,
		ocloudparams.PTPVersionPrereleaseNew)

	pr1, err = oran.PullPR(HubAPIClient, pr1.Object.Name)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to retrieve PR %s", pr1.Object.Name))

	VerifyProvisioningRequestIsFulfilled(pr1)
	glog.V(ocloudparams.OCloudLogLevel).Infof("Provisioning request %s is fulfilled", pr1.Object.Name)

	pr2, err = oran.PullPR(HubAPIClient, pr2.Object.Name)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to retrieve PR %s", pr2.Object.Name))

	VerifyProvisioningRequestIsFulfilled(pr2)
	glog.V(ocloudparams.OCloudLogLevel).Infof("Provisioning request %s is fulfilled", pr2.Object.Name)

	err = os.RemoveAll("tmp/")
	Expect(err).NotTo(HaveOccurred(), "Error removing directory /tmp")

	var wg2 sync.WaitGroup
	wg2.Add(2)
	go DeprovisionAiSnoCluster(pr1, ns1, ci1, node1, nodePool1, ctx, &wg2)
	go DeprovisionAiSnoCluster(pr2, ns2, ci2, node2, nodePool2, ctx, &wg2)
	wg2.Wait()
}

// VerifyFailedOperatorUpgradeAllSnos verifies the test case where the upgrade of the operators fails in all
// the SNOs.
func VerifyFailedOperatorUpgradeAllSnos(ctx SpecContext) {
	downgradeOperatorImages()

	pr1 := VerifyProvisionSnoCluster(
		ocloudparams.TemplateName,
		ocloudparams.TemplateVersion6,
		ocloudparams.NodeClusterName1,
		ocloudparams.OCloudSiteId,
		ocloudparams.PolicyTemplateParameters,
		ocloudparams.ClusterInstanceParameters1)

	pr2 := VerifyProvisionSnoCluster(
		ocloudparams.TemplateName,
		ocloudparams.TemplateVersion6,
		ocloudparams.NodeClusterName2,
		ocloudparams.OCloudSiteId,
		ocloudparams.PolicyTemplateParameters,
		ocloudparams.ClusterInstanceParameters2)

	node1, nodePool1, ns1, ci1 := VerifyAndRetrieveAssociatedCRsForAI(pr1.Object.Name, ocloudparams.ClusterName1, ctx)
	node2, nodePool2, ns2, ci2 := VerifyAndRetrieveAssociatedCRsForAI(pr2.Object.Name, ocloudparams.ClusterName2, ctx)

	var wg1 sync.WaitGroup
	var mu1 sync.Mutex
	wg1.Add(2)
	go VerifyAllPoliciesInNamespaceAreCompliant(ns1.Object.Name, ctx, &wg1, &mu1)
	go VerifyAllPoliciesInNamespaceAreCompliant(ns2.Object.Name, ctx, &wg1, &mu1)
	wg1.Wait()

	pr1, err := oran.PullPR(HubAPIClient, pr1.Object.Name)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to retrieve PR %s", pr1.Object.Name))

	VerifyProvisioningRequestIsFulfilled(pr1)
	glog.V(ocloudparams.OCloudLogLevel).Infof("Provisioning request %s is fulfilled", pr1.Object.Name)

	pr2, err = oran.PullPR(HubAPIClient, pr2.Object.Name)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to retrieve PR %s", pr2.Object.Name))

	VerifyProvisioningRequestIsFulfilled(pr2)
	glog.V(ocloudparams.OCloudLogLevel).Infof("Provisioning request %s is fulfilled", pr2.Object.Name)

	sno1ApiClient := CreateSnoApiClient(ocloudparams.ClusterName1)
	sno2ApiClient := CreateSnoApiClient(ocloudparams.ClusterName2)

	verifyPtpOperatorVersionInSno(
		sno1ApiClient,
		ocloudparams.PTPVersionMajorOld,
		ocloudparams.PTPVersionMinorOld,
		ocloudparams.PTPVersionPatchOld,
		ocloudparams.PTPVersionPrereleaseOld)

	verifyPtpOperatorVersionInSno(
		sno2ApiClient,
		ocloudparams.PTPVersionMajorOld,
		ocloudparams.PTPVersionMinorOld,
		ocloudparams.PTPVersionPatchOld,
		ocloudparams.PTPVersionPrereleaseOld)

	VerifyAllPodsRunningInNamespace(sno1ApiClient, ocloudparams.PtpNamespace)
	VerifyAllPodsRunningInNamespace(sno2ApiClient, ocloudparams.PtpNamespace)

	upgradeOperatorImages()

	modifyDeploymentResources(
		sno1ApiClient,
		ocloudparams.PtpOperatorSubscriptionName,
		ocloudparams.PtpNamespace,
		ocloudparams.PtpDeploymentName,
		ocloudparams.PtpContainerName,
		ocloudparams.PtpCpuRequest,
		ocloudparams.PtpMemoryRequest,
		ocloudparams.PtpCpuLimit,
		ocloudparams.PtpMemoryLimit)

	modifyDeploymentResources(
		sno2ApiClient,
		ocloudparams.PtpOperatorSubscriptionName,
		ocloudparams.PtpNamespace,
		ocloudparams.PtpDeploymentName,
		ocloudparams.PtpContainerName,
		ocloudparams.PtpCpuRequest,
		ocloudparams.PtpMemoryRequest,
		ocloudparams.PtpCpuLimit,
		ocloudparams.PtpMemoryLimit)

	var wg2 sync.WaitGroup
	var mu2 sync.Mutex
	wg2.Add(2)
	go VerifyPoliciesAreNotCompliant(pr1, ocloudparams.ClusterName1, ctx, &wg2, &mu2)
	go VerifyPoliciesAreNotCompliant(pr2, ocloudparams.ClusterName2, ctx, &wg2, &mu2)
	wg2.Wait()

	pr1, err = oran.PullPR(HubAPIClient, pr1.Object.Name)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to retrieve PR %s", pr1.Object.Name))

	VerifyProvisioningRequestTimeout(pr1)
	glog.V(ocloudparams.OCloudLogLevel).Infof("Provisioning request %s is timeout", pr1.Object.Name)

	pr2, err = oran.PullPR(HubAPIClient, pr2.Object.Name)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to retrieve PR %s", pr2.Object.Name))

	VerifyProvisioningRequestTimeout(pr2)
	glog.V(ocloudparams.OCloudLogLevel).Infof("Provisioning request %s is timeout", pr2.Object.Name)

	err = os.RemoveAll("tmp/")
	Expect(err).NotTo(HaveOccurred(), "Error removing directory /tmp")

	var wg3 sync.WaitGroup
	wg3.Add(2)
	go DeprovisionAiSnoCluster(pr1, ns1, ci1, node1, nodePool1, ctx, &wg3)
	go DeprovisionAiSnoCluster(pr2, ns2, ci2, node2, nodePool2, ctx, &wg3)
	wg3.Wait()
}

// VerifyFailedOperatorUpgradeSubsetSnos verifies the test case where the upgrade of the operators fails in a
// subset of the SNOs.
func VerifyFailedOperatorUpgradeSubsetSnos(ctx SpecContext) {
	downgradeOperatorImages()

	pr1 := VerifyProvisionSnoCluster(
		ocloudparams.TemplateName,
		ocloudparams.TemplateVersion6,
		ocloudparams.NodeClusterName1,
		ocloudparams.OCloudSiteId,
		ocloudparams.PolicyTemplateParameters,
		ocloudparams.ClusterInstanceParameters1)

	pr2 := VerifyProvisionSnoCluster(
		ocloudparams.TemplateName,
		ocloudparams.TemplateVersion6,
		ocloudparams.NodeClusterName2,
		ocloudparams.OCloudSiteId,
		ocloudparams.PolicyTemplateParameters,
		ocloudparams.ClusterInstanceParameters2)

	node1, nodePool1, ns1, ci1 := VerifyAndRetrieveAssociatedCRsForAI(pr1.Object.Name, ocloudparams.ClusterName1, ctx)
	node2, nodePool2, ns2, ci2 := VerifyAndRetrieveAssociatedCRsForAI(pr2.Object.Name, ocloudparams.ClusterName2, ctx)

	var wg1 sync.WaitGroup
	var mu1 sync.Mutex
	wg1.Add(2)
	go VerifyAllPoliciesInNamespaceAreCompliant(ns1.Object.Name, ctx, &wg1, &mu1)
	go VerifyAllPoliciesInNamespaceAreCompliant(ns2.Object.Name, ctx, &wg1, &mu1)
	wg1.Wait()

	pr1, err := oran.PullPR(HubAPIClient, pr1.Object.Name)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to retrieve PR %s", pr1.Object.Name))

	VerifyProvisioningRequestIsFulfilled(pr1)
	glog.V(ocloudparams.OCloudLogLevel).Infof("Provisioning request %s is fulfilled", pr1.Object.Name)

	pr2, err = oran.PullPR(HubAPIClient, pr2.Object.Name)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to retrieve PR %s", pr2.Object.Name))

	VerifyProvisioningRequestIsFulfilled(pr2)
	glog.V(ocloudparams.OCloudLogLevel).Infof("Provisioning request %s is fulfilled", pr2.Object.Name)

	sno1ApiClient := CreateSnoApiClient(ocloudparams.ClusterName1)
	sno2ApiClient := CreateSnoApiClient(ocloudparams.ClusterName2)

	verifyPtpOperatorVersionInSno(
		sno1ApiClient,
		ocloudparams.PTPVersionMajorOld,
		ocloudparams.PTPVersionMinorOld,
		ocloudparams.PTPVersionPatchOld,
		ocloudparams.PTPVersionPrereleaseOld)

	verifyPtpOperatorVersionInSno(
		sno2ApiClient,
		ocloudparams.PTPVersionMajorOld,
		ocloudparams.PTPVersionMinorOld,
		ocloudparams.PTPVersionPatchOld,
		ocloudparams.PTPVersionPrereleaseOld)

	VerifyAllPodsRunningInNamespace(sno1ApiClient, ocloudparams.PtpNamespace)
	VerifyAllPodsRunningInNamespace(sno2ApiClient, ocloudparams.PtpNamespace)

	upgradeOperatorImages()

	modifyDeploymentResources(
		sno1ApiClient,
		ocloudparams.PtpOperatorSubscriptionName,
		ocloudparams.PtpNamespace,
		ocloudparams.PtpDeploymentName,
		ocloudparams.PtpContainerName,
		ocloudparams.PtpCpuRequest,
		ocloudparams.PtpMemoryRequest,
		ocloudparams.PtpCpuLimit,
		ocloudparams.PtpMemoryLimit)

	var wg2 sync.WaitGroup
	var mu2 sync.Mutex
	wg2.Add(2)
	go VerifyPoliciesAreNotCompliant(pr1, ocloudparams.ClusterName1, ctx, &wg2, &mu2)
	go VerifyPoliciesAreNotCompliant(pr2, ocloudparams.ClusterName2, ctx, &wg2, &mu2)
	wg2.Wait()

	VerifyAllPoliciesInNamespaceAreCompliant(ns2.Object.Name, ctx, nil, nil)

	verifyPtpOperatorVersionInSno(
		sno2ApiClient,
		ocloudparams.PTPVersionMajorNew,
		ocloudparams.PTPVersionMinorNew,
		ocloudparams.PTPVersionPatchNew,
		ocloudparams.PTPVersionPrereleaseNew)

	pr1, err = oran.PullPR(HubAPIClient, pr1.Object.Name)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to retrieve PR %s", pr1.Object.Name))

	VerifyProvisioningRequestTimeout(pr1)
	glog.V(ocloudparams.OCloudLogLevel).Infof("Provisioning request %s timedout", pr1.Object.Name)

	pr2, err = oran.PullPR(HubAPIClient, pr2.Object.Name)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to retrieve PR %s", pr2.Object.Name))

	VerifyProvisioningRequestIsFulfilled(pr2)
	glog.V(ocloudparams.OCloudLogLevel).Infof("Provisioning request %s is fulfilled", pr2.Object.Name)

	err = os.RemoveAll("tmp/")
	Expect(err).NotTo(HaveOccurred(), "Error removing directory /tmp")

	var wg4 sync.WaitGroup
	wg4.Add(2)
	go DeprovisionAiSnoCluster(pr1, ns1, ci1, node1, nodePool1, ctx, &wg4)
	go DeprovisionAiSnoCluster(pr2, ns2, ci2, node2, nodePool2, ctx, &wg4)
	wg4.Wait()
}

// verifyPtpOperatorVersionInSno verifies that the PTP operator is in the specified version.
func verifyPtpOperatorVersionInSno(sno1ApiClient *clients.Settings,
	major uint64, minor uint64, patch uint64, prerelease uint64) {
	By(fmt.Sprintf("Verifing that PTP Operator version is %d.%d.%d-%d", major, minor, patch, prerelease))
	csvName, err := csv.GetCurrentCSVNameFromSubscription(sno1ApiClient,
		ocloudparams.PtpOperatorSubscriptionName, ocloudparams.PtpNamespace)
	Expect(err).NotTo(HaveOccurred(),
		fmt.Sprintf("csv %s not found in namespace %s", csvName, ocloudparams.PtpNamespace))

	csvObj, err := olm.PullClusterServiceVersion(sno1ApiClient, csvName, ocloudparams.PtpNamespace)
	Expect(err).NotTo(HaveOccurred(),
		fmt.Sprintf("failed to pull %q csv from the %s namespace", csvName, ocloudparams.PtpNamespace))

	versionOk := false
	ptpVersion := csvObj.Object.Spec.Version
	if ptpVersion.Major == major &&
		ptpVersion.Minor == minor &&
		ptpVersion.Patch == patch {
		for _, pre := range csvObj.Object.Spec.Version.Pre {
			if pre.VersionNum == prerelease {
				versionOk = true
			}
		}
	}

	Expect(versionOk).To(BeTrue(), fmt.Sprintf("PTP version %s is not the expected one", ptpVersion))
}

// modifyDeploymentResources modifies the cpu and memory resources available for a given container, in a given
// deployment in a given subscription.
func modifyDeploymentResources(
	apiClient *clients.Settings,
	subscriptionName string,
	nsname string,
	deploymentName string,
	containerName string,
	cpuRequest string,
	memoryRequest string,
	cpuLimit string,
	memoryLimit string) {
	csvName, err := csv.GetCurrentCSVNameFromSubscription(apiClient, subscriptionName, nsname)
	if err != nil {
		Skip(fmt.Sprintf("csv %s not found in namespace %s", csvName, nsname))
	}

	csvObj, err := olm.PullClusterServiceVersion(apiClient, csvName, nsname)
	if err != nil {
		Skip(fmt.Sprintf("failed to pull %q csv from the %s namespace", csvName, nsname))
	}

	for i, deployment := range csvObj.Object.Spec.InstallStrategy.StrategySpec.DeploymentSpecs {
		if deployment.Name == deploymentName {
			for j, container := range deployment.Spec.Template.Spec.Containers {
				if container.Name == containerName {
					csvObj.Object.Spec.InstallStrategy.StrategySpec.DeploymentSpecs[i].Spec.Template.Spec.Containers[j].Resources = corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							"cpu":    resource.MustParse(cpuLimit),
							"memory": resource.MustParse(memoryLimit),
						},
						Requests: corev1.ResourceList{
							"cpu":    resource.MustParse(cpuRequest),
							"memory": resource.MustParse(memoryRequest),
						},
					}
					csvObj.Update()
				}
			}
		}
	}
}

// upgradeOperatorImages upgrades the operator images.
func upgradeOperatorImages() {
	_, err := shell.ExecuteCmd(ocloudparams.PodmanTagOperatorUpgrade)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Error tagging redhat-operators image for upgrade: %v", err))

	_, err = shell.ExecuteCmd(ocloudparams.PodmanTagSriovUpgrade)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Error tagging far-edge-sriov-fec image for upgrade: %v", err))

	_, err = shell.ExecuteCmd(ocloudparams.PodmanPushOperatorUpgrade)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Error pushing redhat-operators image for upgrade: %v", err))

	_, err = shell.ExecuteCmd(ocloudparams.PodmanPushSriovUpgrade)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Error pushing far-edge-sriov-fec image for upgrade: %v", err))
}

// downgradeOperatorImages downgrades the operator images.
func downgradeOperatorImages() {
	_, err := shell.ExecuteCmd(ocloudparams.PodmanTagOperatorDowngrade)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Error tagging redhat-operators image for downgrade: %v", err))

	_, err = shell.ExecuteCmd(ocloudparams.PodmanTagSriovDowngrade)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Error tagging far-edge-sriov-fec image for downgrade: %v", err))

	_, err = shell.ExecuteCmd(ocloudparams.PodmanPushOperatorDowngrade)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Error pushing redhat-operators image for downgrade: %v", err))

	_, err = shell.ExecuteCmd(ocloudparams.PodmanPushSriovDowngrade)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Error pushing far-edge-sriov-fec image for downgrade: %v", err))
}
