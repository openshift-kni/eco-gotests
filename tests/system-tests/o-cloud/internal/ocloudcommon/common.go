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
	"sigs.k8s.io/controller-runtime/pkg/client"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// VerifyCsvSuccessful verifies that a specific subscription exists.
func VerifyCsvSuccessful(apiClient *clients.Settings, subscriptionName string, nsname string) {
	By(fmt.Sprintf("Verifying that csv %s is successful", subscriptionName))

	csvName, err := csv.GetCurrentCSVNameFromSubscription(apiClient, subscriptionName, nsname)
	if err != nil {
		Skip(fmt.Sprintf("csv %s not found in namespace %s", csvName, nsname))
	}

	csvObj, err := olm.PullClusterServiceVersion(apiClient, csvName, nsname)
	if err != nil {
		Skip(fmt.Sprintf("failed to pull %q csv from the %s namespace", csvName, nsname))
	}

	_, err = csvObj.IsSuccessful()
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("failed to verify csv %s in the namespace %s status", csvName, nsname))
	
	glog.V(ocloudparams.OCloudLogLevel).Infof("csv %s is successful", subscriptionName)
}

// VerifyAllPodsRunningInNamespace verifies that all the pods in a given namespace are running.
func VerifyAllPodsRunningInNamespace(apiClient *clients.Settings, nsname string) {
	By(fmt.Sprintf("Verifying that pods exist in %s namespace", nsname))
	pods, err := pod.List(apiClient, nsname)
	Expect(err).NotTo(HaveOccurred(),
		fmt.Sprintf("Error while listing pods in %s namespace", nsname))
	Expect(len(pods) > 0).To(BeTrue(),
		fmt.Sprintf("Error: did not find any pods in the %s namespace", nsname))

	By(fmt.Sprintf("Verifying that pods are running correctly in %s namespace", nsname))
	running, err := pod.WaitForAllPodsInNamespaceRunning(apiClient, nsname, time.Minute)
	Expect(err).NotTo(HaveOccurred(),
		fmt.Sprintf("Error occurred while waiting for %s pods to be in Running state", nsname))
	Expect(running).To(BeTrue(),
		fmt.Sprintf("Some %s pods are not in Running state", nsname))
	
	glog.V(ocloudparams.OCloudLogLevel).Infof("all pods running in %s namespace", nsname)
}

// VerifyNamespaceExists verifies that a specific namespace exists.
func VerifyNamespaceExists(nsname string) *namespace.Builder {
	By(fmt.Sprintf("Verifying that %s namespace exists", nsname))
	//err := apiobjectshelper.VerifyNamespaceExists(HubAPIClient, nsname, time.Second)
	ns, err := namespace.Pull(HubAPIClient, nsname)
	Expect(err).ToNot(HaveOccurred(),
	fmt.Sprintf("Failed to pull namespace %q; %v", nsname, err))

	glog.V(ocloudparams.OCloudLogLevel).Infof("namespace %s exists", nsname)

	return ns
}

// VerifyNamespaceDoesNotExist verifies that a given namespace does not exist.
func VerifyNamespaceDoesNotExist(ns *namespace.Builder, wg *sync.WaitGroup, ctx SpecContext) {
	defer wg.Done()
	defer GinkgoRecover()

	nsName := ns.Object.Name
	By(fmt.Sprintf("Verifying that namespace %s does not exist", nsName))
	Eventually(func(ctx context.Context) bool {
		return !ns.Exists()
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
	oCloudSiteId string,
	policyTemplateParameters map[string]any,
	clusterInstanceParameters map[string]any) *oran.ProvisioningRequestBuilder {
	prName := uuid.New().String()

	By(fmt.Sprintf("Verifing the successful creation of the %s PR", prName))

	pr := oran.NewPRBuilder(HubAPIClient, prName, templateName, templateVersion)
	pr.WithTemplateParameter("nodeClusterName", nodeClusterName)
	pr.WithTemplateParameter("oCloudSiteId", oCloudSiteId)
	pr.WithTemplateParameter("policyTemplateParameters", policyTemplateParameters)
	pr.WithTemplateParameter("clusterInstanceParameters", clusterInstanceParameters)
	pr, err := pr.Create()
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to create PR %s", prName))

	condition := metav1.Condition{
		Type:   "ClusterInstanceProcessed",
		Reason: "Completed",
	}
	pr.WaitForCondition(condition, time.Minute*5)
	glog.V(ocloudparams.OCloudLogLevel).Infof("provisioning request %s has been created", prName)

	return pr
}

func VerifyProvisioningRequestIsFulfilled(pr *oran.ProvisioningRequestBuilder) {
	By(fmt.Sprintf("Verifing that PR %s is fulfilled", pr.Object.Name))
	condition := metav1.Condition{
		Type:   "ClusterProvisioned",
		Reason: "Completed",
	}
	pr.WaitForCondition(condition, time.Minute*10)

	condition = metav1.Condition{
		Type:   "ConfigurationApplied",
		Reason: "Completed",
	}
	pr.WaitForCondition(condition, time.Minute*10)

	glog.V(ocloudparams.OCloudLogLevel).Infof("provisioningrequest %s is fulfilled", pr.Object.Name)
}

// VerifyProvisioningRequestTimeout
func VerifyProvisioningRequestTimeout(pr *oran.ProvisioningRequestBuilder) {
	condition := metav1.Condition{
		Type:   "ConfigurationApplied",
		Reason: "TimedOut",
		Status: "False",
	}
	pr.WaitForCondition(condition, time.Minute*100)

	glog.V(ocloudparams.OCloudLogLevel).Infof("provisioningrequest %s has failed (timeout)", pr.Object.Name)
}

// VerifyProvisioningRequestIsDeleted verifies that a given provisioning request is deleted.
func VerifyProvisioningRequestIsDeleted(pr *oran.ProvisioningRequestBuilder, wg *sync.WaitGroup, ctx SpecContext) {
	defer wg.Done()
	defer GinkgoRecover()

	prName := pr.Object.Name
	err := pr.DeleteAndWait(30*time.Minute)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to delete PR %s: %v", prName, err))
	
	glog.V(ocloudparams.OCloudLogLevel).Infof("provisioningrequest %s deleted", prName)
}

func GetProvisioningRequestName(clusterName string) string {
	nodePool, err := oran.PullNodePool(HubAPIClient, clusterName , ocloudparams.OCloudHardwareManagerPluginNamespace)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to retrieve node pool %s", ocloudparams.ClusterName1))

	for _, ownerReference := range nodePool.Object.OwnerReferences {
		if ownerReference.Kind == "ProvisioningRequest" {
			return ownerReference.Name
		}
	}

	return ""
}

// VerifyClusterInstanceCompleted verifies that a cluster instance exists, that it is provisioned and
// that it is associated to a given provisioning request.
func VerifyClusterInstanceCompleted(prName string, ns string, ciName string, ctx SpecContext) *siteconfig.CIBuilder {
	By(fmt.Sprintf("Verifying that %s PR has a Cluster Instance CR associated in namespace %s", prName, ns))

	ci, err := siteconfig.PullClusterInstance(HubAPIClient, ciName, ns)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to pull Cluster Instance %q; %v", ns, err))

	found := false
	for _, value := range ci.Object.ObjectMeta.Labels {
		if value == prName {
			found = true
			break
		}
	}
	Expect(found).To(BeTrue(),
		fmt.Sprintf("Failed to verify that Cluster Instance %s is associated to PR %s", ciName, prName))

	Eventually(func(ctx context.Context) bool {
		ci, err = siteconfig.PullClusterInstance(HubAPIClient, ciName, ns)
		Expect(err).ToNot(HaveOccurred(),
			fmt.Sprintf("Failed to pull Cluster Instance %q; %v", ns, err))
		for _, value := range ci.Object.Status.Conditions {
			if value.Type == "Provisioned" && value.Status == "True" {
				return true
			}
		}
		return false
	}).WithTimeout(80*time.Minute).WithPolling(time.Minute).WithContext(ctx).Should(BeTrue(),
		fmt.Sprintf("ClusterInstance %s is not Completed", ciName))

	glog.V(ocloudparams.OCloudLogLevel).Infof("clusterinstance %s is completed", ciName)

	return ci
}

// VerifyClusterInstanceDoesNotExist verifies that a given cluster instance does not exist
func VerifyClusterInstanceDoesNotExist(ci *siteconfig.CIBuilder, wg *sync.WaitGroup, ctx SpecContext) {
	defer wg.Done()
	defer GinkgoRecover()

	ciName := ci.Object.Name
	By(fmt.Sprintf("Verifying that clusterinstance %s does not exist", ciName))
	Eventually(func(ctx context.Context) bool {
		return !ci.Exists()
	}).WithTimeout(30*time.Minute).WithPolling(time.Second).WithContext(ctx).Should(BeTrue(),
		fmt.Sprintf("ClusterInstance %s still exists", ciName))
	
	glog.V(ocloudparams.OCloudLogLevel).Infof("clusterinstance %s does not exist", ciName)
}

// VerifyAllPoliciesInNamespaceAreCompliant verifies that all the policies in a given namespace
// report compliant.
func VerifyAllPoliciesInNamespaceAreCompliant(
	nsName string, ctx SpecContext, wg *sync.WaitGroup, mu *sync.Mutex) {
	if wg != nil {
		defer wg.Done()
		defer GinkgoRecover()
	}

	By(fmt.Sprintf("Verifying that all the policies in namespace %s are Compliant", nsName))

	Eventually(func(ctx context.Context) bool {
		if mu != nil {
			mu.Lock()
		}
		policies, err := ocm.ListPoliciesInAllNamespaces(HubAPIClient, runtimeclient.ListOptions{Namespace: nsName})
		Expect(err).ToNot(HaveOccurred(),
			fmt.Sprintf("Failed to pull policies from namespaces %s: %v", nsName, err))
		if mu != nil {
			mu.Unlock()
		}
		for _, policy := range policies {

			if policy.Object.Status.ComplianceState != "Compliant" {
				return false
			}

		}
		return true
	}).WithTimeout(100*time.Minute).WithPolling(30*time.Second).WithContext(ctx).Should(BeTrue(),
		fmt.Sprintf("Failed to verify that all the policies in namespace %s are Compliant", nsName))
	
	glog.V(ocloudparams.OCloudLogLevel).Infof("all the policies in namespace %s are compliant", nsName)
}

// VerifyPoliciesAreNotCompliant verifies that not all the policies in a given namespace
// report compliant.
func VerifyPoliciesAreNotCompliant(
	pr *oran.ProvisioningRequestBuilder,
	nsName string,
	ctx SpecContext,
	wg *sync.WaitGroup,
	mu *sync.Mutex) {
	defer wg.Done()
	defer GinkgoRecover()

	By(fmt.Sprintf("Verifying that not all the policies in namespace %s are Compliant", nsName))
	Eventually(func(ctx context.Context) bool {
		if mu != nil {
			mu.Lock()
		}
		policies, err := ocm.ListPoliciesInAllNamespaces(HubAPIClient, runtimeclient.ListOptions{Namespace: nsName})
		Expect(err).ToNot(HaveOccurred(),
			fmt.Sprintf("Failed to pull policies from namespace %s: %v", nsName, err))
		if mu != nil {
			mu.Unlock()
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
func VerifyOranNodeExistsInNamespace(nodeId string, nsName string) *oran.NodeBuilder {
	By(fmt.Sprintf("Verifying that ORAN node %s exists in namespace %s ", nodeId, nsName))

	listOptions := &client.ListOptions{}
	listOptions.Namespace = nsName
	oranNodes, err := oran.ListNodes(HubAPIClient, *listOptions)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to pull oran node list from namespace %s: %v", nsName, err))

	nodeFound := false
	i := 0
	for index, node := range oranNodes {
		if nodeId == node.Object.Spec.NodePool {
			nodeFound = true
			i = index
			break
		}
	}

	Expect(nodeFound).To(BeTrue(),
		fmt.Sprintf("Failed to pull the oran node with the HW MGR ID %s from namespace %s", nodeId, nsName))

	if nodeFound {
		glog.V(ocloudparams.OCloudLogLevel).Infof("oran node %s exists in namespace %s", nodeId, nsName)

		return oranNodes[i]
	} else {
		glog.V(ocloudparams.OCloudLogLevel).Infof("oran node %s does not exists in namespace %s", nodeId, nsName)

		return nil
	}
}

// VerifyOranNodeDoesNotExist verifies that a given ORAN node does not exist.
func VerifyOranNodeDoesNotExist(node *oran.NodeBuilder, wg *sync.WaitGroup, ctx SpecContext) {
	defer wg.Done()
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
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to pull oran node pool %s from namespace %s: %v", nodePoolName, nsName, err))
	
	glog.V(ocloudparams.OCloudLogLevel).Infof("oran node pool %s exists in namespace %s", nodePoolName, nsName)
	
	return oranNodePool
}

// VerifyOranNodePoolDoesNotExist verifies that a given ORAN node pool does not exist.
func VerifyOranNodePoolDoesNotExist(nodePool *oran.NodePoolBuilder, wg *sync.WaitGroup, ctx SpecContext) {
	defer wg.Done()
	defer GinkgoRecover()

	nodePoolName := nodePool.Object.Name
	By(fmt.Sprintf("Verifying that oran node pool %s does not exist", nodePoolName))
	Eventually(func(ctx context.Context) bool {
		return !nodePool.Exists()
	}).WithTimeout(30*time.Minute).WithPolling(time.Second).WithContext(ctx).Should(BeTrue(),
		fmt.Sprintf("Oran node pool %s still exists", nodePoolName))
	
	glog.V(ocloudparams.OCloudLogLevel).Infof("oran node pool %s does not exist", nodePoolName)
}

func CreateSnoApiClient(nodeName string) *clients.Settings {
	path := fmt.Sprintf("tmp/%s/auth", nodeName)
	err := os.MkdirAll(path, 0750)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Error creating directory %s", path))

	createSnoKubeconfig := fmt.Sprintf(ocloudparams.SnoKubeconfigCreate, nodeName, nodeName, nodeName)
	_, err = shell.ExecuteCmd(createSnoKubeconfig)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Error creating %s kubeconfig", nodeName))

	snoKubeconfigPath := fmt.Sprintf("tmp/%s/auth/kubeconfig", nodeName)
	snoApiClient := clients.New(snoKubeconfigPath)
	return snoApiClient
}


