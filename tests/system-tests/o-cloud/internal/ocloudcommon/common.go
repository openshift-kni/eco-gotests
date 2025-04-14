package ocloudcommon

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	. "github.com/openshift-kni/eco-gotests/tests/system-tests/o-cloud/internal/ocloudinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/o-cloud/internal/ocloudparams"

	"github.com/google/uuid"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/ocm"
	"github.com/openshift-kni/eco-goinfra/pkg/olm"
	"github.com/openshift-kni/eco-goinfra/pkg/oran"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-goinfra/pkg/siteconfig"

	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/csv"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/shell"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	policiesv1 "open-cluster-management.io/governance-policy-propagator/api/v1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// VerifyCsvSuccessful verifies that a specific subscription exists.
func VerifyCsvSuccessful(apiClient *clients.Settings, subscriptionName string, nsName string) {
	By(fmt.Sprintf("Verifying that csv %s is successful", subscriptionName))

	csvName, err := csv.GetCurrentCSVNameFromSubscription(apiClient, subscriptionName, nsName)
	if err != nil {
		Skip(fmt.Sprintf("csv %s not found in namespace %s", csvName, nsName))
	}

	csvObj, err := olm.PullClusterServiceVersion(apiClient, csvName, nsName)
	if err != nil {
		Skip(fmt.Sprintf("failed to pull %q csv from the %s namespace", csvName, nsName))
	}

	_, err = csvObj.IsSuccessful()
	Expect(err).ToNot(HaveOccurred(), "failed to verify csv %s in the namespace %s status", csvName, nsName)

	glog.V(ocloudparams.OCloudLogLevel).Infof("csv %s is successful", subscriptionName)
}

// VerifyAllPodsRunningInNamespace verifies that all the pods in a given namespace are running.
func VerifyAllPodsRunningInNamespace(apiClient *clients.Settings, nsName string) {
	By(fmt.Sprintf("Verifying that pods exist in %s namespace", nsName))

	pods, err := pod.List(apiClient, nsName)
	Expect(err).NotTo(HaveOccurred(),
		fmt.Sprintf("Error while listing pods in %s namespace", nsName))
	Expect(len(pods) > 0).To(BeTrue(),
		fmt.Sprintf("Error: did not find any pods in the %s namespace", nsName))

	By(fmt.Sprintf("Verifying that pods are running correctly in %s namespace", nsName))

	running, err := pod.WaitForAllPodsInNamespaceRunning(apiClient, nsName, time.Minute)
	Expect(err).NotTo(HaveOccurred(),
		fmt.Sprintf("Error occurred while waiting for %s pods to be in Running state", nsName))
	Expect(running).To(BeTrue(),
		fmt.Sprintf("Some %s pods are not in Running state", nsName))

	glog.V(ocloudparams.OCloudLogLevel).Infof("all pods running in %s namespace", nsName)
}

// VerifyNamespaceExists verifies that a specific namespace exists.
func VerifyNamespaceExists(nsName string) *namespace.Builder {
	By(fmt.Sprintf("Verifying that %s namespace exists", nsName))

	namespace, err := namespace.Pull(HubAPIClient, nsName)
	Expect(err).ToNot(HaveOccurred(), "Failed to pull namespace %q; %v", nsName, err)

	glog.V(ocloudparams.OCloudLogLevel).Infof("namespace %s exists", nsName)

	return namespace
}

// VerifyNamespaceDoesNotExist verifies that a given namespace does not exist.
func VerifyNamespaceDoesNotExist(namespace *namespace.Builder, waitGroup *sync.WaitGroup, ctx SpecContext) {
	defer waitGroup.Done()
	defer GinkgoRecover()

	nsName := namespace.Object.Name

	By(fmt.Sprintf("Verifying that namespace %s does not exist", nsName))

	Eventually(func(ctx context.Context) bool {
		return !namespace.Exists()
	}).WithTimeout(30*time.Minute).WithPolling(time.Second).WithContext(ctx).Should(BeTrue(),
		fmt.Sprintf("Namespace %s still exists", nsName))

	glog.V(ocloudparams.OCloudLogLevel).Infof("namespace %s does not exist", nsName)
}

// VerifyProvisionSnoCluster verifies the successful creation or provisioning request and
// that the provisioning request is progressing.
func VerifyProvisionSnoCluster(
	templateName string,
	templateVersion string,
	nodeClusterName string,
	oCloudSiteID string,
	policyTemplateParameters map[string]any,
	clusterInstanceParameters map[string]any) *oran.ProvisioningRequestBuilder {
	prName := uuid.New().String()

	By(fmt.Sprintf("Verifing the successful creation of the %s PR", prName))

	provisioningRequest := oran.NewPRBuilder(HubAPIClient, prName, templateName, templateVersion).
		WithTemplateParameter("nodeClusterName", nodeClusterName).
		WithTemplateParameter("oCloudSiteId", oCloudSiteID).
		WithTemplateParameter("policyTemplateParameters", policyTemplateParameters).
		WithTemplateParameter("clusterInstanceParameters", clusterInstanceParameters)
	provisioningRequest, err := provisioningRequest.Create()
	Expect(err).ToNot(HaveOccurred(), "Failed to create PR %s", prName)

	condition := metav1.Condition{
		Type:   "ClusterInstanceProcessed",
		Reason: "Completed",
	}

	_, err = provisioningRequest.WaitForCondition(condition, time.Minute*5)
	Expect(err).ToNot(HaveOccurred(), "PR %s is not getting processing", prName)

	glog.V(ocloudparams.OCloudLogLevel).Infof("provisioning request %s has been created", prName)

	return provisioningRequest
}

// VerifyProvisioningRequestIsFulfilled verifies that a given provisioning request is fulfilled.
func VerifyProvisioningRequestIsFulfilled(provisioningRequest *oran.ProvisioningRequestBuilder) {
	By(fmt.Sprintf("Verifing that PR %s is fulfilled", provisioningRequest.Object.Name))

	_, err := provisioningRequest.WaitUntilFulfilled(time.Minute * 10)
	Expect(err).ToNot(HaveOccurred(), "PR %s is not fulfilled", provisioningRequest.Object.Name)

	glog.V(ocloudparams.OCloudLogLevel).Infof("provisioningrequest %s is fulfilled", provisioningRequest.Object.Name)
}

// VerifyProvisioningRequestTimeout verifies that a provisioning request has timed out.
func VerifyProvisioningRequestTimeout(provisioningRequest *oran.ProvisioningRequestBuilder) {
	By(fmt.Sprintf("Verifing that PR %s has timed out", provisioningRequest.Object.Name))

	condition := metav1.Condition{
		Type:   "ConfigurationApplied",
		Reason: "TimedOut",
		Status: "False",
	}
	_, err := provisioningRequest.WaitForCondition(condition, time.Minute*100)
	Expect(err).ToNot(HaveOccurred(), "PR %s failed to report timeout", provisioningRequest.Object.Name)

	glog.V(ocloudparams.OCloudLogLevel).
		Infof("provisioningrequest %s has failed (timeout)", provisioningRequest.Object.Name)
}

// VerifyProvisioningRequestIsDeleted verifies that a given provisioning request is deleted.
func VerifyProvisioningRequestIsDeleted(
	provisioningRequest *oran.ProvisioningRequestBuilder,
	waitGroup *sync.WaitGroup,
	ctx SpecContext) {
	defer waitGroup.Done()
	defer GinkgoRecover()

	prName := provisioningRequest.Object.Name
	err := provisioningRequest.DeleteAndWait(30 * time.Minute)
	Expect(err).ToNot(HaveOccurred(), "Failed to delete PR %s: %v", prName, err)

	glog.V(ocloudparams.OCloudLogLevel).Infof("provisioningrequest %s deleted", prName)
}

// VerifyClusterInstanceCompleted verifies that a cluster instance exists, that it is provisioned and
// that it is associated to a given provisioning request.
func VerifyClusterInstanceCompleted(
	prName string, nsName string, ciName string, ctx SpecContext) *siteconfig.CIBuilder {
	By(fmt.Sprintf("Verifying that %s PR has a Cluster Instance CR associated in namespace %s", prName, nsName))

	clusterInstance, err := siteconfig.PullClusterInstance(HubAPIClient, ciName, nsName)
	Expect(err).ToNot(HaveOccurred(), "Failed to pull Cluster Instance %q; %v", nsName, err)

	found := false

	for _, value := range clusterInstance.Object.ObjectMeta.Labels {
		if value == prName {
			found = true

			break
		}
	}

	Expect(found).To(BeTrue(),
		fmt.Sprintf("Failed to verify that Cluster Instance %s is associated to PR %s", ciName, prName))

	condition := metav1.Condition{
		Type:   "Provisioned",
		Status: "True",
	}

	clusterInstance, err = clusterInstance.WaitForCondition(condition, 80*time.Minute)
	Expect(err).ToNot(HaveOccurred(), "Clusterinstance is not provisioned %s: %v", ciName, err)

	glog.V(ocloudparams.OCloudLogLevel).Infof("clusterinstance %s is completed", ciName)

	return clusterInstance
}

// VerifyClusterInstanceDoesNotExist verifies that a given cluster instance does not exist.
func VerifyClusterInstanceDoesNotExist(
	clusterInstance *siteconfig.CIBuilder, waitGroup *sync.WaitGroup, ctx SpecContext) {
	defer waitGroup.Done()
	defer GinkgoRecover()

	ciName := clusterInstance.Object.Name
	By(fmt.Sprintf("Verifying that clusterinstance %s does not exist", ciName))
	Eventually(func(ctx context.Context) bool {
		return !clusterInstance.Exists()
	}).WithTimeout(30*time.Minute).WithPolling(time.Second).WithContext(ctx).Should(BeTrue(),
		fmt.Sprintf("ClusterInstance %s still exists", ciName))

	glog.V(ocloudparams.OCloudLogLevel).Infof("clusterinstance %s does not exist", ciName)
}

// VerifyAllPoliciesInNamespaceAreCompliant verifies that all the policies in a given namespace
// report compliant.
func VerifyAllPoliciesInNamespaceAreCompliant(
	nsName string, ctx SpecContext, waitGroup *sync.WaitGroup, mutex *sync.Mutex) {
	if waitGroup != nil {
		defer waitGroup.Done()
		defer GinkgoRecover()
	}

	By(fmt.Sprintf("Verifying that all the policies in namespace %s are Compliant", nsName))

	err := ocm.WaitForAllPoliciesComplianceState(
		HubAPIClient, policiesv1.Compliant, 100*time.Minute, runtimeclient.ListOptions{Namespace: nsName})
	Expect(err).ToNot(HaveOccurred(), "Failed to verify that all the policies in namespace %s are Compliant", nsName)

	glog.V(ocloudparams.OCloudLogLevel).Infof("all the policies in namespace %s are compliant", nsName)
}

// VerifyPoliciesAreNotCompliant verifies that not all the policies in a given namespace
// report compliant.
func VerifyPoliciesAreNotCompliant(
	nsName string,
	ctx SpecContext,
	waitGroup *sync.WaitGroup,
	mutex *sync.Mutex) {
	defer waitGroup.Done()
	defer GinkgoRecover()

	By(fmt.Sprintf("Verifying that not all the policies in namespace %s are Compliant", nsName))

	Eventually(func(ctx context.Context) bool {
		if mutex != nil {
			mutex.Lock()
		}
		policies, err := ocm.ListPoliciesInAllNamespaces(
			HubAPIClient, runtimeclient.ListOptions{Namespace: nsName})
		Expect(err).ToNot(HaveOccurred(), "Failed to pull policies from namespace %s: %v", nsName, err)
		if mutex != nil {
			mutex.Unlock()
		}
		for _, policy := range policies {
			if policy.Object.Status.ComplianceState != "Compliant" {
				return true
			}
		}

		return false
	}).WithTimeout(30*time.Minute).WithPolling(3*time.Second).WithContext(ctx).Should(BeTrue(),
		fmt.Sprintf("Failed to verify that not all the policies in namespace %s are Compliant", nsName))

	glog.V(ocloudparams.OCloudLogLevel).Infof("all the policies in namespace %s are not compliant", nsName)
}

// VerifyOranNodeExistsInNamespace verifies that a given ORAN node exists in a given namespace.
func VerifyOranNodeExistsInNamespace(nodeID string, nsName string) *oran.NodeBuilder {
	By(fmt.Sprintf("Verifying that ORAN node %s exists in namespace %s ", nodeID, nsName))

	listOptions := &runtimeclient.ListOptions{}
	listOptions.Namespace = nsName
	oranNodes, err := oran.ListNodes(HubAPIClient, *listOptions)
	Expect(err).ToNot(HaveOccurred(), "Failed to pull oran node list from namespace %s: %v", nsName, err)

	nodeFound := false
	nodeFoundIndex := 0

	for index, node := range oranNodes {
		if nodeID == node.Object.Spec.NodePool {
			nodeFound = true
			nodeFoundIndex = index

			break
		}
	}

	Expect(nodeFound).To(BeTrue(),
		fmt.Sprintf("Failed to pull the oran node with the HW MGR ID %s from namespace %s", nodeID, nsName))

	if nodeFound {
		glog.V(ocloudparams.OCloudLogLevel).Infof("oran node %s exists in namespace %s", nodeID, nsName)

		return oranNodes[nodeFoundIndex]
	}

	glog.V(ocloudparams.OCloudLogLevel).Infof("oran node %s does not exists in namespace %s", nodeID, nsName)

	return nil
}

// VerifyOranNodeDoesNotExist verifies that a given ORAN node does not exist.
func VerifyOranNodeDoesNotExist(node *oran.NodeBuilder, waitGroup *sync.WaitGroup, ctx SpecContext) {
	defer waitGroup.Done()
	defer GinkgoRecover()

	nodeName := node.Object.Name

	By(fmt.Sprintf("Verifying that oran node %s does not exist", nodeName))

	Eventually(func(ctx context.Context) bool {
		return !node.Exists()
	}).WithTimeout(30*time.Minute).WithPolling(time.Second).WithContext(ctx).Should(BeTrue(),
		fmt.Sprintf("Oran node %s still exists", nodeName))

	glog.V(ocloudparams.OCloudLogLevel).Infof("oran node %s does not exists", nodeName)
}

// VerifyOranNodePoolExistsInNamespace verifies that a given ORAN node pool exists in a given namespace.
func VerifyOranNodePoolExistsInNamespace(nodePoolName string, nsName string) *oran.NodePoolBuilder {
	By(fmt.Sprintf("Verifying that ORAN node pool %s exists in namespace %s", nodePoolName, nsName))

	oranNodePool, err := oran.PullNodePool(HubAPIClient, nodePoolName, nsName)
	Expect(err).ToNot(HaveOccurred(), "Failed to pull oran node pool %s from namespace %s: %v", nodePoolName, nsName, err)

	glog.V(ocloudparams.OCloudLogLevel).Infof("oran node pool %s exists in namespace %s", nodePoolName, nsName)

	return oranNodePool
}

// VerifyOranNodePoolDoesNotExist verifies that a given ORAN node pool does not exist.
func VerifyOranNodePoolDoesNotExist(nodePool *oran.NodePoolBuilder, waitGroup *sync.WaitGroup, ctx SpecContext) {
	defer waitGroup.Done()
	defer GinkgoRecover()

	nodePoolName := nodePool.Object.Name

	By(fmt.Sprintf("Verifying that oran node pool %s does not exist", nodePoolName))

	Eventually(func(ctx context.Context) bool {
		return !nodePool.Exists()
	}).WithTimeout(30*time.Minute).WithPolling(time.Second).WithContext(ctx).Should(BeTrue(),
		fmt.Sprintf("Oran node pool %s still exists", nodePoolName))

	glog.V(ocloudparams.OCloudLogLevel).Infof("oran node pool %s does not exist", nodePoolName)
}

// CreateSnoAPIClient creates a new client for the given node.
func CreateSnoAPIClient(nodeName string) *clients.Settings {
	path := fmt.Sprintf("tmp/%s/auth", nodeName)
	err := os.MkdirAll(path, 0750)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Error creating directory %s", path))

	createSnoKubeconfig := fmt.Sprintf(ocloudparams.SnoKubeconfigCreate, nodeName, nodeName, nodeName)
	_, err = shell.ExecuteCmd(createSnoKubeconfig)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Error creating %s kubeconfig", nodeName))

	snoKubeconfigPath := fmt.Sprintf("tmp/%s/auth/kubeconfig", nodeName)
	snoAPIClient := clients.New(snoKubeconfigPath)

	return snoAPIClient
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

// DeprovisionAiSnoCluster deprovisions a SNO cluster.
func DeprovisionAiSnoCluster(
	provisioningRequest *oran.ProvisioningRequestBuilder,
	namespace *namespace.Builder,
	clusterInstance *siteconfig.CIBuilder,
	node *oran.NodeBuilder,
	nodePool *oran.NodePoolBuilder,
	ctx SpecContext,
	waitGroup *sync.WaitGroup) {
	if waitGroup != nil {
		defer waitGroup.Done()
		defer GinkgoRecover()
	}

	prName := provisioningRequest.Object.Name
	By(fmt.Sprintf("Tearing down PR %s", prName))

	var tearDownWg sync.WaitGroup

	tearDownWg.Add(5)

	go VerifyProvisioningRequestIsDeleted(provisioningRequest, &tearDownWg, ctx)
	go VerifyNamespaceDoesNotExist(namespace, &tearDownWg, ctx)
	go VerifyClusterInstanceDoesNotExist(clusterInstance, &tearDownWg, ctx)
	go VerifyOranNodeDoesNotExist(node, &tearDownWg, ctx)
	go VerifyOranNodePoolDoesNotExist(nodePool, &tearDownWg, ctx)

	tearDownWg.Wait()

	glog.V(ocloudparams.OCloudLogLevel).Infof("Provisioning request %s has been removed", prName)
}
