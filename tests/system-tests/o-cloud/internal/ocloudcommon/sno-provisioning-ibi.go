package ocloudcommon

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/golang/glog"
	
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	bmhv1alpha1 "github.com/metal3-io/baremetal-operator/apis/metal3.io/v1alpha1"
	"github.com/openshift-kni/eco-goinfra/pkg/bmh"
	"github.com/openshift-kni/eco-goinfra/pkg/ibi"
	corev1 "k8s.io/api/core/v1"

	"github.com/Masterminds/sprig/v3"
	"github.com/openshift-kni/eco-goinfra/pkg/bmc"
	"github.com/openshift-kni/eco-goinfra/pkg/lca"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/ocm"
	"github.com/openshift-kni/eco-goinfra/pkg/oran"
	"github.com/openshift-kni/eco-goinfra/pkg/secret"
	"github.com/openshift-kni/eco-goinfra/pkg/sriov"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/files"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/shell"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/sshcommand"
	. "github.com/openshift-kni/eco-gotests/tests/system-tests/o-cloud/internal/ocloudinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/o-cloud/internal/ocloudparams"
)

type ImageBasedInstallConfigData struct {
	BaseImageName       string
	SeedImage           string
	SeedVersion         string
	Registry            string
	PullSecret          string
	SshKey              string
	RegistryCertificate string
	InterfaceName       string
	InterfaceIpv6       string
	DnsIpv6             string
	NextHopIpv6         string
	NextHopInterface    string
}

// VerifySuccessfulIbiSnoProvisioning verifies the successful provisioning of a SNO cluster using
// the Image Based Installer
func VerifySuccessfulIbiSnoProvisioning(ctx SpecContext) {
	if OCloudConfig.GenerateSeedImage && !baseImageExists() {
		generateBaseImage(ctx)
	}
	
	installBaseImage(
		OCloudConfig.Spoke2BMC, 
		OCloudConfig.IbiBaseImageUrl, 
		OCloudConfig.VirtualMediaID, 
		ocloudparams.SshCluster2, 
		ocloudparams.SpokeSshUser, 
		ocloudparams.SpokeSshPasskeyPath, 
		ctx, 
		time.Minute)

	pr := VerifyProvisionSnoCluster(
		ocloudparams.TemplateName,
		ocloudparams.TemplateVersion4,
		ocloudparams.NodeClusterName2,
		ocloudparams.OCloudSiteId,
		ocloudparams.PolicyTemplateParameters,
		ocloudparams.ClusterInstanceParameters2)

	node, nodePool, ns, bmh, ici := verifyAndRetrieveAssociatedCRsForIBI(
		ocloudparams.ClusterName2,
		ocloudparams.ClusterName2,
		ocloudparams.ClusterName2,
		ocloudparams.HostName2,
		ctx)

	VerifyAllPoliciesInNamespaceAreCompliant(ns.Object.Name, ctx, nil, nil)
	glog.V(ocloudparams.OCloudLogLevel).Infof("All the policies in namespace %s are Complete", ns.Object.Name)

	VerifyProvisioningRequestIsFulfilled(pr)
	glog.V(ocloudparams.OCloudLogLevel).Infof("Provisioning request %s is fulfilled", pr.Object.Name)

	deprovisionIbiSnoCluster(pr, ns, node, nodePool, bmh, ici, ctx)
}

// VerifyFailedIbiSnoProvisioning verifies the failed provisioning of a SNO cluster using
// the Image Based Installer
func VerifyFailedIbiSnoProvisioning(ctx SpecContext) {
	if OCloudConfig.GenerateSeedImage && !baseImageExists() {
		generateBaseImage(ctx)
	}

	installBaseImage(
		OCloudConfig.Spoke2BMC, 
		OCloudConfig.IbiBaseImageUrl, 
		OCloudConfig.VirtualMediaID, 
		ocloudparams.SshCluster2, 
		ocloudparams.SpokeSshUser, 
		ocloudparams.SpokeSshPasskeyPath, 
		ctx, 
		time.Minute)

	pr := VerifyProvisionSnoCluster(
		ocloudparams.TemplateName,
		ocloudparams.TemplateVersion5,
		ocloudparams.NodeClusterName2,
		ocloudparams.OCloudSiteId,
		ocloudparams.PolicyTemplateParameters,
		ocloudparams.ClusterInstanceParameters2)

	node, nodePool, ns, bmh, ici := verifyAndRetrieveAssociatedCRsForIBI(
		ocloudparams.ClusterName2,
		ocloudparams.ClusterName2,
		ocloudparams.ClusterName2,
		ocloudparams.HostName2,
		ctx)

	VerifyProvisioningRequestTimeout(pr)
	glog.V(ocloudparams.OCloudLogLevel).Infof("Provisioning request %s has timed out", pr.Object.Name)

	deprovisionIbiSnoCluster(pr, ns, node, nodePool, bmh, ici, ctx)
}

// installBaseImage boots a given spoke cluster from CD using the specified base image and virtual media ID,
// and uses ssh to verify that the installation of the base image has finished before a given time.
func installBaseImage(
	spoke *bmc.BMC,
	isoUrl, virtualMediaID, sshHost, sshUser, sshPassKey string, 
	ctx context.Context, 
	timeout time.Duration) {
	By("Installing base image in target SNO")
	err := spoke.BootFromCD(isoUrl, virtualMediaID)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Error setting virtual media to %s", isoUrl))

	powerState, err := spoke.SystemPowerState()
	Expect(err).NotTo(HaveOccurred(), "Error getting system power state")

	if powerState != "Off" {
		err = spoke.SystemForceReset()
		Expect(err).NotTo(HaveOccurred(), "Error resetting the system")
	} else {
		err = spoke.SystemPowerOn()
		Expect(err).NotTo(HaveOccurred(), "Error powering on the system")
	}

	By("Checking if image based install finished in target SNO")
	Eventually(func(ctx context.Context) bool {
		output := sshcommand.SSHCommand(ocloudparams.CheckIbiCompleted, sshHost, sshUser, sshPassKey)
		if output.Err == nil && output.SSHOutput != "" {
			return true
		}
		return false
	}).WithTimeout(80*time.Minute).WithPolling(timeout).WithContext(ctx).Should(BeTrue(),
		"Image based install did not completed")
}

// generateBaseImage provisions a seed SNO cluster and generates a base image to be used with Image Based Installation.
func generateBaseImage(ctx SpecContext) {
	By("Generating a base image from seed SNO")
	pr := VerifyProvisionSnoCluster(
		ocloudparams.TemplateName,
		ocloudparams.TemplateVersionSeed,
		ocloudparams.NodeClusterName1,
		ocloudparams.OCloudSiteId,
		ocloudparams.PolicyTemplateParameters,
		ocloudparams.ClusterInstanceParameters1)

	node, nodePool, ns, ci := VerifyAndRetrieveAssociatedCRsForAI(pr.Object.Name, ocloudparams.ClusterName1, ctx)

	VerifyAllPoliciesInNamespaceAreCompliant(ns.Object.Name, ctx, nil, nil)
	glog.V(ocloudparams.OCloudLogLevel).Infof("All the policies are compliant")

	VerifyProvisioningRequestIsFulfilled(pr)
	glog.V(ocloudparams.OCloudLogLevel).Infof("Provisioning request %s is fulfilled", pr.Object.Name)

	snoApiClient := CreateSnoApiClient(ocloudparams.ClusterName1)

	By("Verifying the SR-IOV network node states")
	networkNodeStates, err := sriov.ListNetworkNodeState(snoApiClient, ocloudparams.SriovNamespace)
	Expect(err).NotTo(HaveOccurred(),
		fmt.Sprintf("Error getting the list of SR-IOV network node states in namespace %s", ocloudparams.SriovNamespace))

	for _, networkNodeState := range networkNodeStates {
		networkNodeState.WaitUntilSyncStatus("Succeeded", 30*time.Minute)
	}

	By("Detaching the seed SNO from the hub")
	cluster, err := ocm.PullManagedCluster(HubAPIClient, ocloudparams.ClusterName1)
	if err == nil {
		err = cluster.DeleteAndWait(time.Minute * 10)
		Expect(err).NotTo(HaveOccurred(),
			fmt.Sprintf("Error deleting managed cluster %s", ocloudparams.ClusterName1))
	}

	By("Creating a seedgen secret in the LCA namespace")
	secret := secret.NewBuilder(snoApiClient, "seedgen", ocloudparams.LifecycleAgentNamespace, corev1.SecretTypeOpaque)
	data := make(map[string][]byte)
	data["seedAuth"] = []byte(OCloudConfig.LocalRegistryAuth)
	secret.WithData(data)
	_, err = secret.Create()
	Expect(err).NotTo(HaveOccurred(),
		fmt.Sprintf("Error creating seedgen secret in namespace %s: %v", ocloudparams.LifecycleAgentNamespace, err))

	By("Creating a seed generator")
	seedGenerator := lca.NewSeedGeneratorBuilder(snoApiClient, ocloudparams.SeedGeneratorName)
	seedGenerator.WithSeedImage(OCloudConfig.SeedImage)
	seedGenerator, err = seedGenerator.Create()
	Expect(err).NotTo(HaveOccurred(),
		fmt.Sprintf("Error creating seedgenerator seedimage: %v", err))
	_, err = seedGenerator.WaitUntilComplete(30 * time.Minute)
	Expect(err).NotTo(HaveOccurred(),
		fmt.Sprintf("Seedgenerator seedimage did not completed: %v", err))

	By("Creating the base image")
	_, err = shell.ExecuteCmd(ocloudparams.CreateImageBasedInstallationConfig)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Error creating image-based-installation-config.yaml file: %v", err))

	content, err := os.ReadFile(ocloudparams.IbiConfigTemplate)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Error reading config template: %v", err))

	templateText := string(content)
	tmpl, _ := template.New("config").Funcs(sprig.FuncMap()).Parse(templateText)

	certContent, err := os.ReadFile(ocloudparams.RegistryCertPath)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Error reading certificate file: %v", err))
	registryCertificate := removeLastNewline(string(certContent))

	imageBasedConfigData := ImageBasedInstallConfigData{
		BaseImageName:       OCloudConfig.BaseImageName,
		SeedImage:           OCloudConfig.SeedImage,
		SeedVersion:         OCloudConfig.SeedVersion,
		Registry:            OCloudConfig.Registry,
		PullSecret:          OCloudConfig.PullSecret,
		SshKey:              OCloudConfig.SshKey,
		RegistryCertificate: registryCertificate,
		InterfaceName:       OCloudConfig.InterfaceName,
		InterfaceIpv6:       OCloudConfig.InterfaceIpv6,
		DnsIpv6:             OCloudConfig.DnsIpv6,
		NextHopIpv6:         OCloudConfig.NextHopIpv6,
		NextHopInterface:    OCloudConfig.NextHopInterface,
	}

	var rendered bytes.Buffer
	err = tmpl.Execute(&rendered, imageBasedConfigData)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Error executing template file: %v", err))

	err = os.WriteFile(ocloudparams.IbiConfigTemplateYaml, rendered.Bytes(), 0644)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Error writing YAML file: %v", err))

	_, err = shell.ExecuteCmd(ocloudparams.CreateIsoImage)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Error creating rhcos-ibi.iso image: %v", err))

	err = files.CopyFile(ocloudparams.IbiBasedImageSourcePath, OCloudConfig.IbiBaseImagePath)
	Expect(err).NotTo(HaveOccurred(),
		fmt.Sprintf("Error copying rhcos-ibi.iso image to %s: %v", OCloudConfig.IbiBaseImagePath, err))

	DeprovisionAiSnoCluster(pr, ns, ci, node, nodePool, ctx, nil)
}
// verifyBmhDoesNotExist verifies that a given ORAN node does not exist.
func verifyBmhDoesNotExist(bmh *bmh.BmhBuilder, wg *sync.WaitGroup, ctx SpecContext) {
	if wg != nil {
		defer wg.Done()
		defer GinkgoRecover()
	}

	By(fmt.Sprintf("Verifying that BMH %s does not exist", bmh.Object.Name))
	Eventually(func(ctx context.Context) bool {
		return !bmh.Exists()
	}).WithTimeout(5*time.Second).WithPolling(time.Second).WithContext(ctx).Should(BeTrue(),
		fmt.Sprintf("BMH %s still exists", bmh.Object.Name))
}

// verifyImageClusterInstallDoesNotExist verifies that a given ORAN node pool does not exist.
func verifyImageClusterInstallDoesNotExist(ici *ibi.ImageClusterInstallBuilder, wg *sync.WaitGroup, ctx SpecContext) {
	if wg != nil {
		defer wg.Done()
		defer GinkgoRecover()
	}

	By(fmt.Sprintf("Verifying that image cluster install %s does not exist", ici.Object.Name))
	Eventually(func(ctx context.Context) bool {
		return !ici.Exists()
	}).WithTimeout(5*time.Second).WithPolling(time.Second).WithContext(ctx).Should(BeTrue(),
		fmt.Sprintf("Image cluster install %s still exists", ici.Object.Name))
}

// deprovisionIbiSnoCluster deprovisions a SNO cluster that has been deployed using IBI.
func deprovisionIbiSnoCluster(
	pr *oran.ProvisioningRequestBuilder,
	ns *namespace.Builder,
	node *oran.NodeBuilder,
	nodePool *oran.NodePoolBuilder,
	bmh *bmh.BmhBuilder,
	ici *ibi.ImageClusterInstallBuilder,
	ctx SpecContext) {

	By(fmt.Sprintf("Tearing down PR %s", pr.Object.Name))

	var tearDownWg sync.WaitGroup
	tearDownWg.Add(5)
	go VerifyProvisioningRequestIsDeleted(pr, &tearDownWg, ctx)
	go VerifyNamespaceDoesNotExist(ns, &tearDownWg, ctx)
	go VerifyOranNodeDoesNotExist(node, &tearDownWg, ctx)
	go VerifyOranNodePoolDoesNotExist(nodePool, &tearDownWg, ctx)
	go verifyBmhDoesNotExist(bmh, &tearDownWg, ctx)
	go verifyImageClusterInstallDoesNotExist(ici, &tearDownWg, ctx)
	tearDownWg.Wait()

	glog.V(ocloudparams.OCloudLogLevel).Infof("Provisioning request %s has been removed", pr.Object.Name)
	downgradeOperatorImages()
}

// verifyAndRetrieveAssociatedCRsForIBI verifies that a given ORAN node, a given ORAN node pool, a given namespace
// and a given cluster instance exist and retrieves them.
func verifyAndRetrieveAssociatedCRsForIBI(nodeId string,
	nodePoolName string,
	nsName string,
	hostName string,
	ctx SpecContext) (*oran.NodeBuilder, *oran.NodePoolBuilder, *namespace.Builder, *bmh.BmhBuilder, *ibi.ImageClusterInstallBuilder) {
	By(fmt.Sprintf("Verifying that BMH %s exists in namespace %s", hostName, nsName))
	bmh, err := bmh.Pull(HubAPIClient, hostName, nsName)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to pull BMH %s from namespace %s: %v", hostName, nsName, err))

	err = bmh.WaitUntilInStatus(bmhv1alpha1.StateExternallyProvisioned, 10*time.Minute)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to verify that BMH %s is externally provisioned", hostName))

	glog.V(ocloudparams.OCloudLogLevel).Infof("BMH %s is externally provisioned", hostName)

	By(fmt.Sprintf("Verifying that Image Cluster Install %s in namespace %s has succeeded", nodeId, nsName))

	ici, err := ibi.PullImageClusterInstall(HubAPIClient, nodeId, nsName)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to pull Image Cluster install %s from namespace %s; %v", nodeId, nsName, err))

	Eventually(func(ctx context.Context) bool {
		condition, _ := ici.GetCompletedCondition()
		return condition.Status == "True"
	}).WithTimeout(60*time.Minute).WithPolling(20*time.Second).WithContext(ctx).Should(BeTrue(),
		fmt.Sprintf("Image Cluster Install %s is not Completed", nodeId))

	glog.V(ocloudparams.OCloudLogLevel).Infof("Cluster installation %s has succeeded ", nodeId)

	ns := VerifyNamespaceExists(nsName)
	node := VerifyOranNodeExistsInNamespace(nodeId, ocloudparams.OCloudHardwareManagerPluginNamespace)
	nodePool := VerifyOranNodePoolExistsInNamespace(nodePoolName, ocloudparams.OCloudHardwareManagerPluginNamespace)

	return node, nodePool, ns, bmh, ici
}

// baseImageExists returns true if the IBI base image exists false otherwise.
func baseImageExists() bool {
	By(fmt.Sprintf("Verifying that file %s exists", OCloudConfig.IbiBaseImagePath))
	_, err := os.Stat(OCloudConfig.IbiBaseImagePath)
	return !os.IsNotExist(err)
}

// removeLastNewline removes the last new line.
func removeLastNewline(s string) string {
	lastNewline := strings.LastIndex(s, "\n")
	if lastNewline == -1 {
		return s
	}
	return s[:lastNewline] + s[lastNewline+1:]
}