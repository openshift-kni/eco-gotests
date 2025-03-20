package ocloudcommon

import (
	"fmt"
	"sync"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/oran"
	"github.com/openshift-kni/eco-goinfra/pkg/siteconfig"
	. "github.com/openshift-kni/eco-gotests/tests/system-tests/o-cloud/internal/ocloudinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/o-cloud/internal/ocloudparams"
)

// VerifySuccessfulSnoProvisioning verifies the successful provisioning of a SNO cluster using
// Assisted Installer
func VerifySuccessfulSnoProvisioning(ctx SpecContext) {
	pr := VerifyProvisionSnoCluster(
		ocloudparams.TemplateName,
		ocloudparams.TemplateVersion1,
		ocloudparams.NodeClusterName1,
		ocloudparams.OCloudSiteId,
		ocloudparams.PolicyTemplateParameters,
		ocloudparams.ClusterInstanceParameters1)

	node, nodePool, ns, ci := VerifyAndRetrieveAssociatedCRsForAI(pr.Object.Name, ocloudparams.ClusterName1, ctx)

	VerifyAllPoliciesInNamespaceAreCompliant(ns.Object.Name, ctx, nil, nil)
	glog.V(ocloudparams.OCloudLogLevel).Infof("All the policies in namespace %s are Complete", ns.Object.Name)

	VerifyProvisioningRequestIsFulfilled(pr)
	glog.V(ocloudparams.OCloudLogLevel).Infof("Provisioning request %s is fulfilled", pr.Object.Name)

	DeprovisionAiSnoCluster(pr, ns, ci, node, nodePool, ctx, nil)
}

// VerifyFailedSnoProvisioning verifies that the provisioning of a SNO cluster using
// Assisted Installer fails
func VerifyFailedSnoProvisioning(ctx SpecContext) {
	pr := VerifyProvisionSnoCluster(
		ocloudparams.TemplateName,
		ocloudparams.TemplateVersion2,
		ocloudparams.NodeClusterName1,
		ocloudparams.OCloudSiteId,
		ocloudparams.PolicyTemplateParameters,
		ocloudparams.ClusterInstanceParameters1)

	node, nodePool, ns, ci := VerifyAndRetrieveAssociatedCRsForAI(pr.Object.Name, ocloudparams.ClusterName1, ctx)

	VerifyProvisioningRequestTimeout(pr)

	DeprovisionAiSnoCluster(pr, ns, ci, node, nodePool, ctx, nil)
}

// VerifySimultaneousSnoProvisioningSameClusterTemplate verifies the successful provisioning of two SNO clusters
// simultaneously with the same cluster templates.
func VerifySimultaneousSnoProvisioningSameClusterTemplate(ctx SpecContext) {
	pr1 := VerifyProvisionSnoCluster(
		ocloudparams.TemplateName,
		ocloudparams.TemplateVersion1,
		ocloudparams.NodeClusterName1,
		ocloudparams.OCloudSiteId,
		ocloudparams.PolicyTemplateParameters,
		ocloudparams.ClusterInstanceParameters1)
	pr2 := VerifyProvisionSnoCluster(
		ocloudparams.TemplateName,
		ocloudparams.TemplateVersion1,
		ocloudparams.NodeClusterName2,
		ocloudparams.OCloudSiteId,
		ocloudparams.PolicyTemplateParameters,
		ocloudparams.ClusterInstanceParameters2)

	_, _, ns1, _ := VerifyAndRetrieveAssociatedCRsForAI(pr1.Object.Name, ocloudparams.ClusterName1, ctx)
	_, _, ns2, _ := VerifyAndRetrieveAssociatedCRsForAI(pr2.Object.Name, ocloudparams.ClusterName2, ctx)

	var wg2 sync.WaitGroup
	var mu2 sync.Mutex
	wg2.Add(2)
	go VerifyAllPoliciesInNamespaceAreCompliant(ns1.Object.Name, ctx, &wg2, &mu2)
	go VerifyAllPoliciesInNamespaceAreCompliant(ns2.Object.Name, ctx, &wg2, &mu2)
	wg2.Wait()

	VerifyProvisioningRequestIsFulfilled(pr1)
	VerifyProvisioningRequestIsFulfilled(pr2)
}

// VerifySimultaneousSnoDeprovisioningSameClusterTemplate verifies the successful deletion of
// two SNO clusters with the same cluster template.
func VerifySimultaneousSnoDeprovisioningSameClusterTemplate(ctx SpecContext) {
	prName1 := GetProvisioningRequestName(ocloudparams.ClusterName1)
	prName2 := GetProvisioningRequestName(ocloudparams.ClusterName2)

	pr1, err := oran.PullPR(HubAPIClient, prName1)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to get PR %s", prName1))
	VerifyProvisioningRequestIsFulfilled(pr1)

	pr2, err := oran.PullPR(HubAPIClient, prName2)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to get PR %s", prName2))
	VerifyProvisioningRequestIsFulfilled(pr2)

	By(fmt.Sprintf("Verify that %s PR and %s PR are using the same template version", prName1, prName2))
	pr1TemplateVersion := pr1.Object.Spec.TemplateName + pr1.Object.Spec.TemplateVersion
	pr2TemplateVersion := pr2.Object.Spec.TemplateName + pr1.Object.Spec.TemplateVersion
	Expect(pr1TemplateVersion).To(Equal(pr2TemplateVersion),
		fmt.Sprintf("PR %s and %s are not using the same cluster template", prName1, prName2))

	node1, nodePool1, ns1, ci1 := VerifyAndRetrieveAssociatedCRsForAI(prName1, ocloudparams.ClusterName1, ctx)
	node2, nodePool2, ns2, ci2 := VerifyAndRetrieveAssociatedCRsForAI(prName2, ocloudparams.ClusterName2, ctx)

	var wg sync.WaitGroup
	wg.Add(10)
	go VerifyProvisioningRequestIsDeleted(pr1, &wg, ctx)
	go VerifyProvisioningRequestIsDeleted(pr2, &wg, ctx)
	go VerifyNamespaceDoesNotExist(ns1, &wg, ctx)
	go VerifyNamespaceDoesNotExist(ns2, &wg, ctx)
	go VerifyClusterInstanceDoesNotExist(ci1, &wg, ctx)
	go VerifyClusterInstanceDoesNotExist(ci2, &wg, ctx)
	go VerifyOranNodeDoesNotExist(node1, &wg, ctx)
	go VerifyOranNodeDoesNotExist(node2, &wg, ctx)
	go VerifyOranNodePoolDoesNotExist(nodePool1, &wg, ctx)
	go VerifyOranNodePoolDoesNotExist(nodePool2, &wg, ctx)
	wg.Wait()
}

// VerifySimultaneousSnoProvisioningDifferentClusterTemplates verifies the successful provisioning of
// two SNO clusters simultaneously with different cluster templates.
func VerifySimultaneousSnoProvisioningDifferentClusterTemplates(ctx SpecContext) {
	pr1 := VerifyProvisionSnoCluster(
		ocloudparams.TemplateName,
		ocloudparams.TemplateVersion1,
		ocloudparams.NodeClusterName1,
		ocloudparams.OCloudSiteId,
		ocloudparams.PolicyTemplateParameters,
		ocloudparams.ClusterInstanceParameters1)

	pr2 := VerifyProvisionSnoCluster(
		ocloudparams.TemplateName,
		ocloudparams.TemplateVersion3,
		ocloudparams.NodeClusterName2,
		ocloudparams.OCloudSiteId,
		ocloudparams.PolicyTemplateParameters,
		ocloudparams.ClusterInstanceParameters2)

	_, _, ns1, _ := VerifyAndRetrieveAssociatedCRsForAI(pr1.Object.Name, ocloudparams.ClusterName1, ctx)
	_, _, ns2, _ := VerifyAndRetrieveAssociatedCRsForAI(pr2.Object.Name, ocloudparams.ClusterName2, ctx)

	var wg2 sync.WaitGroup
	var mu2 sync.Mutex
	wg2.Add(2)
	go VerifyAllPoliciesInNamespaceAreCompliant(ns1.Object.Name, ctx, &wg2, &mu2)
	go VerifyAllPoliciesInNamespaceAreCompliant(ns2.Object.Name, ctx, &wg2, &mu2)
	wg2.Wait()
}

// VerifySimultaneousSnoDeprovisioningDifferentClusterTemplates verifies the successful deletion of
// two SNO clusters with different cluster templates.
func VerifySimultaneousSnoDeprovisioningDifferentClusterTemplates(ctx SpecContext) {
	prName1 := GetProvisioningRequestName(ocloudparams.ClusterName1)
	prName2 := GetProvisioningRequestName(ocloudparams.ClusterName2)

	pr1, err := oran.PullPR(HubAPIClient, prName1)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to get PR %s", prName1))
	VerifyProvisioningRequestIsFulfilled(pr1)

	pr2, err := oran.PullPR(HubAPIClient, prName2)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to get PR %s", prName2))
	VerifyProvisioningRequestIsFulfilled(pr2)

	By(fmt.Sprintf("Verify that %s PR and %s PR are using different cluster template versions", prName1, prName2))
	pr1TemplateVersion := pr1.Object.Spec.TemplateName + pr1.Object.Spec.TemplateVersion
	pr2TemplateVersion := pr2.Object.Spec.TemplateName + pr2.Object.Spec.TemplateVersion
	Expect(pr1TemplateVersion).NotTo(Equal(pr2TemplateVersion),
		fmt.Sprintf("PR %s and %s are using the same cluster template", prName1, prName2))

	node1, nodePool1, ns1, ci1 := VerifyAndRetrieveAssociatedCRsForAI(prName1, ocloudparams.ClusterName1, ctx)
	node2, nodePool2, ns2, ci2 := VerifyAndRetrieveAssociatedCRsForAI(prName2, ocloudparams.ClusterName2, ctx)

	var wg sync.WaitGroup
	wg.Add(10)
	go VerifyProvisioningRequestIsDeleted(pr1, &wg, ctx)
	go VerifyProvisioningRequestIsDeleted(pr2, &wg, ctx)
	go VerifyNamespaceDoesNotExist(ns1, &wg, ctx)
	go VerifyNamespaceDoesNotExist(ns2, &wg, ctx)
	go VerifyClusterInstanceDoesNotExist(ci1, &wg, ctx)
	go VerifyClusterInstanceDoesNotExist(ci2, &wg, ctx)
	go VerifyOranNodeDoesNotExist(node1, &wg, ctx)
	go VerifyOranNodeDoesNotExist(node2, &wg, ctx)
	go VerifyOranNodePoolDoesNotExist(nodePool1, &wg, ctx)
	go VerifyOranNodePoolDoesNotExist(nodePool2, &wg, ctx)
	wg.Wait()
}

// DeprovisionSnoCluster deprovisions a SNO cluster.
func DeprovisionAiSnoCluster(
	pr *oran.ProvisioningRequestBuilder,
	ns *namespace.Builder,
	ci *siteconfig.CIBuilder,
	node *oran.NodeBuilder,
	nodePool *oran.NodePoolBuilder,
	ctx SpecContext,
	wg *sync.WaitGroup) {

	if wg != nil {
		defer wg.Done()
		defer GinkgoRecover()
	}

	prName := pr.Object.Name
	By(fmt.Sprintf("Tearing down PR %s", prName))

	var tearDownWg sync.WaitGroup
	tearDownWg.Add(5)
	go VerifyProvisioningRequestIsDeleted(pr, &tearDownWg, ctx)
	go VerifyNamespaceDoesNotExist(ns, &tearDownWg, ctx)
	go VerifyClusterInstanceDoesNotExist(ci, &tearDownWg, ctx)
	go VerifyOranNodeDoesNotExist(node, &tearDownWg, ctx)
	go VerifyOranNodePoolDoesNotExist(nodePool, &tearDownWg, ctx)
	tearDownWg.Wait()

	glog.V(ocloudparams.OCloudLogLevel).Infof("Provisioning request %s has been removed", prName)
	downgradeOperatorImages()
}

// VerifyAndRetrieveAssociatedCRsForAI verifies that a given ORAN node, a given ORAN node pool, a given namespace
// and a given cluster instance exist and retrieves them.
func VerifyAndRetrieveAssociatedCRsForAI(
	prName string,
	clusterName string,
	ctx SpecContext) (*oran.NodeBuilder, *oran.NodePoolBuilder, *namespace.Builder, *siteconfig.CIBuilder) {
	node := VerifyOranNodeExistsInNamespace(clusterName, ocloudparams.OCloudHardwareManagerPluginNamespace)
	nodePool := VerifyOranNodePoolExistsInNamespace(
		clusterName, ocloudparams.OCloudHardwareManagerPluginNamespace)
	ns := VerifyNamespaceExists(clusterName)
	ci := VerifyClusterInstanceCompleted(prName, clusterName, clusterName, ctx)

	return node, nodePool, ns, ci
}
