// CNF-16889 story.
package tests

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift-kni/eco-goinfra/pkg/argocd"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/hive"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/ocm"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran-deployment/deploymenttypes/internal/gitdetails"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran-deployment/deploymenttypes/internal/tsparams"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/ran-deployment/internal/raninittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran-deployment/internal/ranparam"
	"gopkg.in/yaml.v3"
	policiesv1 "open-cluster-management.io/governance-policy-propagator/api/v1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type (
	deploymentType   string
	policyType       string
	clusterType      string
	multiClusterType string
)

const (
	siteconfigKind       deploymentType = "SiteConfig"
	clusterInstImageKind deploymentType = "ClusterInstance ImageClusterInstall"
	clusterInstAgentKind deploymentType = "ClusterInstance AgentClusterInstall"

	pgtKind           policyType = "PolicyGenTemplate"
	acmpgKind         policyType = "ACM PolicyGenerator"
	pgtHubSideTempl   policyType = "PolicyGenTemplate with hub-side templating"
	acmpgHubSideTempl policyType = "ACM PolicyGenerator with hub-side templating"

	multiCluster  multiClusterType = "multi cluster"
	singleCluster multiClusterType = "single cluster"

	snoCluster       clusterType = "SNO Cluster"
	snoPlusWorker    clusterType = "SNO+Worker Cluster"
	threeNodeCluster clusterType = "3 Node Cluster"
	standardCluster  clusterType = "Standard Cluster"

	kustSubstring              string = "/kustomization.y"
	kustKind                   string = "Kustomization"
	gitSiteConfigCloneDir      string = "ztp-deployment-siteconfig"
	gitPolicyTemplatesCloneDir string = "ztp-deployment-policy-templates"
)

var (
	reHubSideTemplate = regexp.MustCompile(`\{\{\s*hub[^\r\n]+hub\s*\}\}`)

	deploymentMethod deploymentType
	policyTemplate   policyType
	isMultiCluster   multiClusterType
	clusterKind      clusterType

	ignorePaths [3]string = [3]string{"source-crs/", "custom-crs/", "extra-manifest/"}
	policiesApp *argocd.ApplicationBuilder
	clustersApp *argocd.ApplicationBuilder
)

var _ = Describe("Cluster Deployment Types Tests", Ordered, Label(tsparams.LabelDeploymentTypeTestCases), func() {
	var (
		siteconfigRepo         *git.Repository
		policiesRepo           *git.Repository
		pathSiteConfig         string
		pathPolicies           string
		clusterDeploymentsList []*hive.ClusterDeploymentBuilder
	)

	BeforeAll(func() {

		// Determine if cluster deployments were successful, check for compliant policies for each cluster
		Expect(Spoke1APIClient.KubeconfigPath).ToNot(BeEmpty(), "KUBECONFIG for first cluster is not provided.")
		getClusterType(Spoke1APIClient)

		err := ocm.WaitForAllPoliciesComplianceState(
			HubAPIClient, policiesv1.Compliant, time.Minute, runtimeclient.ListOptions{Namespace: RANConfig.Spoke1Name})
		if err != nil {
			glog.V(tsparams.LogLevel).Infof(
				"Failed to verify all policies are compliant for spoke %s: %v", RANConfig.Spoke1Name, err)
			Expect(err).ToNot(HaveOccurred(), "Failed to verify all policies are compliant for spoke %s", RANConfig.Spoke1Name)
		}

		getClusterType(Spoke1APIClient)

		clusterDeploymentsList, err = hive.ListClusterDeploymentsInAllNamespaces(HubAPIClient)
		Expect(err).ToNot(HaveOccurred(), "Failed to Get ClusterDeployments list")
		getDeploymentType(Spoke1APIClient, clusterDeploymentsList)

		isMultiCluster = singleCluster

		if Spoke2APIClient.KubeconfigPath != "" {
			err = ocm.WaitForAllPoliciesComplianceState(
				HubAPIClient, policiesv1.Compliant, time.Minute, runtimeclient.ListOptions{Namespace: RANConfig.Spoke2Name})
			if err != nil {
				glog.V(tsparams.LogLevel).Infof(
					"Failed to verify all policies are compliant for spoke %s: %v", RANConfig.Spoke2Name, err)
				Expect(err).ToNot(HaveOccurred(), "Failed to verify all policies are compliant for spoke %s", RANConfig.Spoke2Name)
			} else {
				isMultiCluster = multiCluster
				getClusterType(Spoke2APIClient)
			}
		} else {
			glog.V(tsparams.LogLevel).Infof("Second cluster KUBECONFIG not available")
		}

		policiesApp, err = argocd.PullApplication(
			HubAPIClient, tsparams.ArgoCdPoliciesAppName, ranparam.OpenshiftGitOpsNamespace)
		Expect(err).ToNot(HaveOccurred(), "Failed to get the policies app")

		policiesSource, err := gitdetails.GetGitSource(policiesApp)
		Expect(err).ToNot(HaveOccurred(), "Failed to get the policies app git source")

		pathPolicies = policiesSource.Path

		clustersApp, err = argocd.PullApplication(
			HubAPIClient, tsparams.ArgoCdClustersAppName, ranparam.OpenshiftGitOpsNamespace)
		Expect(err).ToNot(HaveOccurred(), "Failed to get the clusters app")

		clustersSource, err := gitdetails.GetGitSource(clustersApp)
		Expect(err).ToNot(HaveOccurred(), "Failed to get the clusters app git source")

		pathSiteConfig = clustersSource.Path

		glog.V(tsparams.LogLevel).Infof("Successful retreival of apps git details")

		mkGitCloneDirs()

		siteconfigRepo, policiesRepo = gitCloneToDirs()

		// Examine files in repos
		getFilesInfo(siteconfigRepo, pathSiteConfig)
		getFilesInfo(policiesRepo, pathPolicies)

	})

	AfterAll(func() {

		// Clean up git clone directories
		rmGitCloneDirs()
	})

	FIt(fmt.Sprintf("When deployment is %s", multiCluster), reportxml.ID("80498"), func() {
		multiple := &isMultiCluster

		Expect(*multiple == singleCluster || *multiple == multiCluster).To(BeTrueBecause(
			"Deployment must either be single cluster or multi cluster"))

		if *multiple == singleCluster {
			Skip(fmt.Sprintf("Not %s deployment", multiCluster))
		}

		By("Deployment is multi cluster")
	})

	PDescribeTable("Checking install method",
		func(methodValue *deploymentType, kindValue deploymentType) {

			Expect(*methodValue).ToNot(BeEmpty(), "deployMethod should not be empty")

			if *methodValue != kindValue {
				Skip(fmt.Sprintf("Not %s install method", kindValue))
			}

			By(fmt.Sprintf("Install method is %s", kindValue))

		},
		func(methodValue *deploymentType, kindValue deploymentType) string {
			return fmt.Sprintf("When deployment method is %s", kindValue)
		},
		Entry(nil, &deploymentMethod, clusterInstImageKind, reportxml.ID("80495")),
		Entry(nil, &deploymentMethod, clusterInstAgentKind, reportxml.ID("80494")),
		Entry(nil, &deploymentMethod, siteconfigKind, reportxml.ID("80493")),
	)

	PDescribeTable("Checking policy kind",
		func(polcyValue *policyType, kindValue policyType) {

			Expect(*polcyValue).ToNot(BeEmpty(), "polcyTemplate should not be empty")

			if *polcyValue != kindValue {
				Skip(fmt.Sprintf("Not %s policy type", kindValue))
			}

			By(fmt.Sprintf("Polcy type  is %s", kindValue))

		},
		func(polcyValue *policyType, kindValue policyType) string {
			return fmt.Sprintf("When policy type is %s", kindValue)
		},
		Entry(nil, &policyTemplate, pgtKind, reportxml.ID("80496")),
		Entry(nil, &policyTemplate, acmpgKind, reportxml.ID("80502")),
		Entry(nil, &policyTemplate, pgtHubSideTempl, reportxml.ID("80501")),
		Entry(nil, &policyTemplate, acmpgHubSideTempl, reportxml.ID("80503")),
	)

	PDescribeTable("Checking cluster type",
		func(clusterValue *clusterType, kindValue clusterType) {

			Expect(*clusterValue).ToNot(BeEmpty(), "polcyTemplate should not be empty")

			if *clusterValue != kindValue {
				Skip(fmt.Sprintf("Not %s cluster type", kindValue))
			}

			By(fmt.Sprintf("Cluster type is %s", kindValue))

		},
		func(clusterValue *clusterType, kindValue clusterType) string {
			return fmt.Sprintf("When cluster type is %s", kindValue)
		},
		Entry(nil, &clusterKind, snoCluster, reportxml.ID("80497")),
		Entry(nil, &clusterKind, snoPlusWorker, reportxml.ID("81679")),
		Entry(nil, &clusterKind, threeNodeCluster, reportxml.ID("80499")),
		Entry(nil, &clusterKind, standardCluster, reportxml.ID("80500")),
	)

})

// Clean up git clone dirs if they exist and create empty dirctories for git clone targets.
func mkGitCloneDirs() {
	rmGitCloneDirs()

	err := os.MkdirAll(gitSiteConfigCloneDir, 0755)
	Expect(err).ToNot(HaveOccurred(), "Failed to create %s directory", gitSiteConfigCloneDir)

	err = os.MkdirAll(gitPolicyTemplatesCloneDir, 0755)
	Expect(err).ToNot(HaveOccurred(), "Failed to create %s directory", gitPolicyTemplatesCloneDir)
}

// Delete git clone directories.
func rmGitCloneDirs() {
	err := os.RemoveAll(gitSiteConfigCloneDir)
	Expect(err).ToNot(HaveOccurred(), "Failed to remove %s directory", gitSiteConfigCloneDir)

	err = os.RemoveAll(gitPolicyTemplatesCloneDir)
	Expect(err).ToNot(HaveOccurred(), "Failed to remove %s directory", gitPolicyTemplatesCloneDir)
}

// git clone siteconfig and policy templates to target directories.
// clusters and policies apps are cloned separately to allow for
// the case where they point to different repos/branches/paths.
func gitCloneToDirs() (siteconfigRepo *git.Repository, policiesRepo *git.Repository) {
	clustersSource, err := gitdetails.GetGitSource(clustersApp)
	Expect(err).ToNot(HaveOccurred(), "Failed to get clusters app git source details")

	policiesSource, err := gitdetails.GetGitSource(policiesApp)
	Expect(err).ToNot(HaveOccurred(), "Failed to get policies app git source details")

	siteconfigRepo, err = git.PlainClone(gitSiteConfigCloneDir, false, &git.CloneOptions{
		URL:           clustersSource.RepoURL,
		Tags:          git.NoTags,
		ReferenceName: plumbing.ReferenceName(clustersSource.TargetRevision),
		Depth:         1,
		SingleBranch:  true,
		Progress:      nil,
	})
	Expect(err).ToNot(HaveOccurred(), "Failed to git clone siteconfig repo %s branch %s to directory %s",
		clustersSource.RepoURL, clustersSource.TargetRevision, gitSiteConfigCloneDir)
	glog.V(tsparams.LogLevel).Infof("Successful git clone of sitconfig repo %s branch %s",
		clustersSource.RepoURL, clustersSource.TargetRevision)
	glog.V(tsparams.LogLevel).Infof("Path in worktree: %s", clustersSource.Path)

	policiesRepo, err = git.PlainClone(gitPolicyTemplatesCloneDir, false, &git.CloneOptions{
		URL:           policiesSource.RepoURL,
		Tags:          git.NoTags,
		ReferenceName: plumbing.ReferenceName(policiesSource.TargetRevision),
		Depth:         1,
		SingleBranch:  true,
		Progress:      nil,
	})
	Expect(err).ToNot(HaveOccurred(), "Failed to git clone policies repo %s branch %s to directory %s",
		policiesSource.RepoURL, policiesSource.TargetRevision, gitPolicyTemplatesCloneDir)
	glog.V(tsparams.LogLevel).Infof("Successful git clone of policies repo %s branch %s",
		policiesSource.RepoURL, policiesSource.TargetRevision)
	glog.V(tsparams.LogLevel).Infof("Path in worktree: %s", policiesSource.Path)

	return siteconfigRepo, policiesRepo
}

// Get information from the files in the repo, filtering files by extensions, path, and "kind".
func getFilesInfo(repo *git.Repository, path string) {
	remotes, err := repo.Remotes()

	Expect(err).ToNot(HaveOccurred(), "Failed to get list of remotes")
	glog.V(tsparams.LogLevel).Infof("Remote: %s", remotes[0].Config().URLs[0])

	head, err := repo.Head()
	Expect(err).ToNot(HaveOccurred(), "Failed to get branch head")

	commit, err := repo.CommitObject(head.Hash())
	Expect(err).ToNot(HaveOccurred(), "Failed to get commit")

	tree, err := commit.Tree()
	Expect(err).ToNot(HaveOccurred(), "Failed to get file tree")

	err = tree.Files().ForEach(func(fileEntry *object.File) error {
		if !strings.HasPrefix(fileEntry.Name, path) {
			glog.V(tsparams.LogLevel).Infof("Skipping file: %s (outside of path: %s)", fileEntry.Name, path)

			return nil
		}

		for _, ignorePath := range ignorePaths {
			if strings.Contains(fileEntry.Name, ignorePath) {
				glog.V(tsparams.LogLevel).Infof("Skipping reference CR file: %s", fileEntry.Name)

				return nil
			}
		}

		if strings.HasSuffix(fileEntry.Name, ".yaml") || strings.HasSuffix(fileEntry.Name, ".yml") {
			glog.V(tsparams.LogLevel).Infof("Path: %s", fileEntry.Name)

			content, err := fileEntry.Contents()
			contentBytes := []byte(content)

			Expect(err).ToNot(HaveOccurred(), "Failed to get file content")

			// Get YAML Kind value.
			kind := getYAMLKind(contentBytes, fileEntry.Name)

			glog.V(tsparams.LogLevel).Infof("Kind from YAML: %s", kind)

			// Determine deployment and policy types
			switch kind {
			case string(siteconfigKind):
				deploymentMethod = siteconfigKind
			case string(pgtKind):
				hasHST := checkForHubSideTemplate(contentBytes)

				if !hasHST && policyTemplate != pgtHubSideTempl {
					policyTemplate = pgtKind
				} else if hasHST {
					policyTemplate = pgtHubSideTempl
				}
			case string(acmpgKind):
				hasHST := checkForHubSideTemplate(contentBytes)

				if !hasHST && policyTemplate != acmpgHubSideTempl {
					policyTemplate = acmpgKind
				} else if hasHST {
					policyTemplate = acmpgHubSideTempl
				}
			}

			return nil
		}

		glog.V(tsparams.LogLevel).Infof("Skipping non-YAML file: %s", fileEntry.Name)

		return nil
	})
	Expect(err).ToNot(HaveOccurred(), "Failed to get file iterator")
}

// unmarshal YAML and get CR kind. Return empty string if kind is not found in YAML.
// Handle kustomization.yaml for cases where Kind is not specified.
func getYAMLKind(fileData []byte, fileName string) string {
	fileContent := make(map[string]any)
	err := yaml.Unmarshal(fileData, &fileContent)
	Expect(err).ToNot(HaveOccurred(), "Failed to unmarshal file %s as yaml", fileName)

	kind, result := fileContent["kind"].(string)
	if !result {
		glog.V(tsparams.LogLevel).Infof("Failed to determine kind from file %s", fileName)

		if strings.Contains(fileName, kustSubstring) {
			glog.V(tsparams.LogLevel).Infof("Assuming Kind: %s for file %#s", kustKind, fileName)

			return kustKind
		}

		return ""
	}

	return kind
}

// Check file for hub-side templating syntax.
func checkForHubSideTemplate(content []byte) bool {
	if reHubSideTemplate.Match(content) {
		return true
	}
	return false
}

// getCluterType determines the cluster type as one of: standard, 3node, SNO, SNO+Worker.
func getClusterType(cluster *clients.Settings) {
	var (
		workerCount       = 0
		controlPlaneCount = 0
	)

	klusterlet, err := ocm.PullKlusterlet(cluster, ocm.KlusterletName)
	Expect(err).ToNot(HaveOccurred(), "Failed to get klusterlet")

	clusterName := klusterlet.Object.Spec.ClusterName
	Expect(clusterName).ToNot(BeEmpty(), "Failed to get clustername")

	if cluster.KubeconfigPath == "" {
		glog.V(tsparams.LogLevel).Infof("Cluster %s KUBECONFIG is not availabled", clusterName)

		return
	}

	nodes, err := nodes.List(cluster)
	Expect(err).ToNot(HaveOccurred(), "Failed to get nodes list")

	for nodeNum := range nodes {
		nodeName := nodes[nodeNum].Definition.Name
		// isControlPlane, isWorker := false, false

		_, isControlPlane := nodes[nodeNum].Object.Labels[RANConfig.ControlPlaneLabel]
		_, isWorker := nodes[nodeNum].Object.Labels[RANConfig.WorkerLabel]

		Expect(isWorker || isControlPlane).To(BeTrueBecause("Node %s has neither control-plane nor worker label?", nodeName))

		switch {
		case isControlPlane:
			controlPlaneCount++
		case isWorker:
			workerCount++
		}
	}

	glog.V(tsparams.LogLevel).Infof(
		"controlPlaneCount: %d\nworkerCount: %d", controlPlaneCount, workerCount)

	switch {
	case (controlPlaneCount == 3) && (workerCount == 2):
		clusterKind = standardCluster
	case (controlPlaneCount == 3) && (workerCount == 3):
		clusterKind = threeNodeCluster
	case (controlPlaneCount == 1) && (workerCount == 2):
		clusterKind = snoPlusWorker
	case (controlPlaneCount == 1) && (workerCount == 1):
		clusterKind = snoCluster
	}
}

// getDeploymentType determines the deployment type as one of:
// SiteConfig with AgentClusterInstall, ClusterInstance with AgentClusterInstall, or ClusterInstance with ImageClusterInstall.
func getDeploymentType(cluster *clients.Settings, clusterDeploymentsList []*hive.ClusterDeploymentBuilder) {
	if deploymentMethod != siteconfigKind {
		klusterlet, err := ocm.PullKlusterlet(cluster, ocm.KlusterletName)
		Expect(err).ToNot(HaveOccurred(), "Failed to get klusterlet")

		clusterName := klusterlet.Object.Spec.ClusterName
		Expect(clusterName).ToNot(BeEmpty(), "Failed to get clustername")

		for _, clusterDeployment := range clusterDeploymentsList {
			deploymentClusterName := clusterDeployment.Object.Spec.ClusterName
			clusterDeploymentName := clusterDeployment.Object.GetObjectMeta().GetName()

			Expect(clusterName).ToNot(BeEmpty(),
				fmt.Sprintf("clusterdeployment %s does not have ClusterName value",
					clusterDeploymentName))

			if clusterName != deploymentClusterName {
				continue
			}

			installKind := clusterDeployment.Object.Spec.ClusterInstallRef.Kind
			Expect(installKind).ToNot(BeEmpty(),
				fmt.Sprintf("clusterdeployment %s does not have ClusterInstallRef.Kind value",
					clusterDeploymentName))

			switch installKind {
			case "ImageClusterInstall":
				deploymentMethod = clusterInstImageKind
			case "AgentClusterInstall":
				deploymentMethod = clusterInstAgentKind
			}
		}
	}
}
