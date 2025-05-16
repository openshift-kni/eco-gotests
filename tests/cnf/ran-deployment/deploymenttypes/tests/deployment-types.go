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

const (
	gitSiteConfigCloneDir      string = "ztp-deployment-siteconfig"
	gitPolicyTemplatesCloneDir string = "ztp-deployment-policy-templates"
	siteconfigKind             string = "SiteConfig"
	clusterinstanceKind        string = "ClusterInstance"
	pgtKind                    string = "PolicyGenTemplate"
	acmpgKind                  string = "PolicyGenerator"
	multiCluster               string = "Mutiple spoke clusters"
	snoCluster                 string = "SNO Cluster"
	snoPlusWorker              string = "SNO+Worker Cluster"
	threeNodeCluster           string = "3 Node Cluster"
	standardCluster            string = "Standard Cluster"
	kustSubstring              string = "/kustomization.y"
	kustKind                   string = "Kustomization"
	imageInstallKind           string = "ImageClusterInstall"
	agentInstallKind           string = "AgentClusterInstall"
)

var (
	reHubSideTemplate  = regexp.MustCompile(`\{\{\s*hub[^\r\n]+hub\s*\}\}`)
	hasHubSideTemplate = false
	isClusterInstance  = false
	isSiteConfig       = false
	isPGT              = false
	isACMPG            = false
	isMultiCluster     = false
	isSNO              = false
	isSnoPlusWorker    = false
	isThreeNodeCluster = false
	isStandardCluster  = false

	ignorePaths    [3]string = [3]string{"source-crs/", "custom-crs/", "extra-manifest/"}
	isAgentInstall           = false
	isImageInstall           = false

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
		err := ocm.WaitForAllPoliciesComplianceState(
			HubAPIClient, policiesv1.Compliant, time.Minute, runtimeclient.ListOptions{Namespace: RANConfig.Spoke1Name})
		if err != nil {
			glog.V(tsparams.LogLevel).Infof(
				"Failed to verify all policies are compliant for spoke %s: %v", RANConfig.Spoke1Name, err)
			Expect(err).ToNot(HaveOccurred(), "Failed to verify all policies are compliant for spoke %s", RANConfig.Spoke1Name)
		}
		Expect(Spoke1APIClient.KubeconfigPath).ToNot(BeEmpty(), "KUBECONFIG for first cluster is not provided.")
		getClusterType(Spoke1APIClient)

		clusterDeploymentsList, err = hive.ListClusterDeploymentsInAllNamespaces(HubAPIClient)
		Expect(err).ToNot(HaveOccurred(), "Failed to Get ClusterDeployments list")
		getDeploymentType(Spoke1APIClient, clusterDeploymentsList)

		if Spoke2APIClient.KubeconfigPath != "" {
			err = ocm.WaitForAllPoliciesComplianceState(
				HubAPIClient, policiesv1.Compliant, time.Minute, runtimeclient.ListOptions{Namespace: RANConfig.Spoke2Name})
			if err != nil {
				glog.V(tsparams.LogLevel).Infof(
					"Failed to verify all policies are compliant for spoke %s: %v", RANConfig.Spoke2Name, err)
				Expect(err).ToNot(HaveOccurred(), "Failed to verify all policies are compliant for spoke %s", RANConfig.Spoke2Name)
			} else {
				isMultiCluster = true
				getClusterType(Spoke2APIClient)
				getDeploymentType(Spoke2APIClient, clusterDeploymentsList)
			}
		} else {
			glog.V(tsparams.LogLevel).Infof("Second cluster KUBECONFIG not available")
		}

		policiesApp, err = argocd.PullApplication(
			HubAPIClient, tsparams.ArgoCdPoliciesAppName, ranparam.OpenshiftGitOpsNamespace)
		Expect(err).ToNot(HaveOccurred(), "Failed to get the policies app")

		pathPolicies, err = gitdetails.GetGitPath(policiesApp)
		Expect(err).ToNot(HaveOccurred(), "Failed to get the policies app git path")

		clustersApp, err = argocd.PullApplication(
			HubAPIClient, tsparams.ArgoCdClustersAppName, ranparam.OpenshiftGitOpsNamespace)
		Expect(err).ToNot(HaveOccurred(), "Failed to get the clusters app")

		pathSiteConfig, err = gitdetails.GetGitPath(clustersApp)
		Expect(err).ToNot(HaveOccurred(), "Failed to get the clusters app git path")

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

	DescribeTable("Checking install methods",
		func(methodKind *bool, methodValue string, foundKind *bool, kindValue string) {
			if !*foundKind {
				glog.V(tsparams.LogLevel).Infof("Not %s install kind", kindValue)
				Skip(fmt.Sprintf("Not %s install kind", kindValue))
			}
			if !*methodKind {
				glog.V(tsparams.LogLevel).Infof("Not %s install method", methodValue)
				Skip(fmt.Sprintf("Not %s install method", methodValue))
			} else {
				glog.V(tsparams.LogLevel).Infof("%s install method via %s", methodValue, kindValue)
			}
		},
		func(methodKind *bool, methodValue string, foundKind *bool, kindValue string) string {
			return fmt.Sprintf("When deployment method is %s via %s", methodValue, kindValue)
		},
		Entry(nil, &isImageInstall, imageInstallKind,
			&isClusterInstance, clusterinstanceKind, reportxml.ID("80495")),
		Entry(nil, &isAgentInstall, agentInstallKind,
			&isClusterInstance, clusterinstanceKind, reportxml.ID("80494")),
	)

	DescribeTable("Checking deployment kinds",
		func(foundKind *bool, kindValue string) {
			if !*foundKind {
				glog.V(tsparams.LogLevel).Infof("Not %s spoke deployment", kindValue)
				Skip(fmt.Sprintf("Not %s spoke deployment", kindValue))
			} else {
				glog.V(tsparams.LogLevel).Infof("%s spoke deployment found", kindValue)
			}
		},
		Entry(fmt.Sprintf("When deployment is %s", multiCluster),
			&isMultiCluster, multiCluster, reportxml.ID("80498")),
		Entry(fmt.Sprintf("When deployment method is %s", siteconfigKind),
			&isSiteConfig, siteconfigKind, reportxml.ID("80493")),
	)

	DescribeTable("Checking policy kinds",
		func(foundKind *bool, foundHST *bool, checkForHST bool, kindValue string) {
			if !*foundKind {
				glog.V(tsparams.LogLevel).Infof("Not %s spoke deployment", kindValue)
				Skip(fmt.Sprintf("Not %s spoke deployment", kindValue))
			} else {
				glog.V(tsparams.LogLevel).Infof("%s spoke deployment found", kindValue)

				switch {
				case checkForHST && *foundHST:
					glog.V(tsparams.LogLevel).Infof("Hub-side templating found (expected)")
				case checkForHST && !*foundHST:
					glog.V(tsparams.LogLevel).Infof("Hub-side templating not found")
					Skip("Hub-side templating not found")
				case !checkForHST && *foundHST:
					glog.V(tsparams.LogLevel).Infof("Hub-side templating found")
					Skip("Hub-side templating found")
				case !checkForHST && !*foundHST:
					glog.V(tsparams.LogLevel).Infof("Hub-side templating not found (expected)")
				}
			}
		},
		func(foundKind *bool, foundHST *bool, checkForHST bool, kindValue string) string {
			if checkForHST {
				return fmt.Sprintf("When policy template is %s with hub-side templating", kindValue)
			}

			return fmt.Sprintf("When policy template is %s without hub-side templating", kindValue)
		},
		Entry(nil, &isPGT, &hasHubSideTemplate, false, pgtKind, reportxml.ID("80496")),
		Entry(nil, &isACMPG, &hasHubSideTemplate, false, acmpgKind, reportxml.ID("80502")),
		Entry(nil, &isPGT, &hasHubSideTemplate, true, pgtKind, reportxml.ID("80501")),
		Entry(nil, &isACMPG, &hasHubSideTemplate, true, acmpgKind, reportxml.ID("80503")),
	)

	DescribeTable("Checking cluster types",
		func(foundType *bool, typeValue string) {
			if !*foundType {
				glog.V(tsparams.LogLevel).Infof("Not cluster type %s", typeValue)
				Skip(fmt.Sprintf("Not cluster type %s", typeValue))
			} else {
				glog.V(tsparams.LogLevel).Infof("Spoke cluster type %s found", typeValue)
			}
		},
		func(foundType *bool, typeValue string) string {
			return fmt.Sprintf("When cluster type is %s", typeValue)
		},
		Entry(nil, &isSNO, snoCluster, reportxml.ID("80497")),
		Entry(nil, &isSnoPlusWorker, snoPlusWorker, reportxml.ID("81679")),
		Entry(nil, &isThreeNodeCluster, threeNodeCluster, reportxml.ID("80499")),
		Entry(nil, &isStandardCluster, standardCluster, reportxml.ID("80500")),
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
	remoteSiteConfig, err := gitdetails.GetGitRepoUrl(clustersApp)
	Expect(err).ToNot(HaveOccurred(), "Failed to get clusters app git URL")

	branchSiteConfig, err := gitdetails.GetGitTargetRevision(clustersApp)
	Expect(err).ToNot(HaveOccurred(), "Failed to get clusters app git branch")

	pathSiteConfig, err := gitdetails.GetGitPath(clustersApp)
	Expect(err).ToNot(HaveOccurred(), "Failed to get clusters app git path")

	remotePolicies, err := gitdetails.GetGitRepoUrl(policiesApp)
	Expect(err).ToNot(HaveOccurred(), "Failed to get policies app git URL")

	branchPolicies, err := gitdetails.GetGitTargetRevision(policiesApp)
	Expect(err).ToNot(HaveOccurred(), "Failed to get policies app git branch")

	pathPolicies, err := gitdetails.GetGitPath(policiesApp)
	Expect(err).ToNot(HaveOccurred(), "Failed to get policies app git path")

	siteconfigRepo, err = git.PlainClone(gitSiteConfigCloneDir, false, &git.CloneOptions{
		URL:           remoteSiteConfig,
		Tags:          git.NoTags,
		ReferenceName: plumbing.ReferenceName(branchSiteConfig),
		Depth:         1,
		SingleBranch:  true,
		Progress:      nil,
	})
	Expect(err).ToNot(HaveOccurred(), "Failed to git clone siteconfig repo %s branch %s to directory %s",
		remoteSiteConfig, branchSiteConfig, gitSiteConfigCloneDir)
	glog.V(tsparams.LogLevel).Infof("Successful git clone of sitconfig repo %s branch %s",
		remoteSiteConfig, branchSiteConfig)
	glog.V(tsparams.LogLevel).Infof("Path in worktree: %s", pathSiteConfig)

	policiesRepo, err = git.PlainClone(gitPolicyTemplatesCloneDir, false, &git.CloneOptions{
		URL:           remotePolicies,
		Tags:          git.NoTags,
		ReferenceName: plumbing.ReferenceName(branchPolicies),
		Depth:         1,
		SingleBranch:  true,
		Progress:      nil,
	})
	Expect(err).ToNot(HaveOccurred(), "Failed to git clone policies repo %s branch %s to directory %s",
		remotePolicies, remotePolicies, gitPolicyTemplatesCloneDir)
	glog.V(tsparams.LogLevel).Infof("Successful git clone of policies repo %s branch %s", remotePolicies, branchPolicies)
	glog.V(tsparams.LogLevel).Infof("Path in worktree: %s", pathPolicies)

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
			case siteconfigKind:
				isSiteConfig = true
			case clusterinstanceKind:
				isClusterInstance = true
			case pgtKind:
				isPGT = true

				checkForHubSideTemplate(contentBytes)
			case acmpgKind:
				isACMPG = true

				checkForHubSideTemplate(contentBytes)
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
func checkForHubSideTemplate(content []byte) {
	if reHubSideTemplate.Match(content) {
		hasHubSideTemplate = true
	}
}

// Determine the cluster type as one of: standard, 3node, SNO, SNO+Worker.
func getClusterType(cluster *clients.Settings) {
	var (
		bothCount         = 0
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
		case isControlPlane && isWorker:
			bothCount++
		case isControlPlane:
			controlPlaneCount++
		case isWorker:
			workerCount++
		}
	}

	glog.V(tsparams.LogLevel).Infof(
		"bothCount: %d\ncontrolPlaneCount: %d\nworkerCount: %d", bothCount, controlPlaneCount, workerCount)

	switch {
	case (controlPlaneCount == 3) && (workerCount == 2):
		isStandardCluster = true
	case (bothCount == 3) && (workerCount == 0):
		isThreeNodeCluster = true
	case (bothCount == 1) && (workerCount == 1):
		isSnoPlusWorker = true
	case bothCount == 1:
		isSNO = true
	}
}

func getDeploymentType(cluster *clients.Settings, clusterDeploymentsList []*hive.ClusterDeploymentBuilder) {
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
		case imageInstallKind:
			isImageInstall = true
		case agentInstallKind:
			isAgentInstall = true
		}
	}
}
