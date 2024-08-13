package deploy_test

import (
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/bmh"
	"github.com/openshift-kni/eco-goinfra/pkg/hive"
	"github.com/openshift-kni/eco-goinfra/pkg/ibi"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	ibiv1alpha1 "github.com/openshift-kni/eco-goinfra/pkg/schemes/imagebasedinstall/api/hiveextensions/v1alpha1"
	"github.com/openshift-kni/eco-goinfra/pkg/secret"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedinstall/mgmt/deploy/internal/tsparams"
	. "github.com/openshift-kni/eco-gotests/tests/lca/imagebasedinstall/mgmt/internal/mgmtinittools"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe(
	"Performing image based installation",
	Ordered,
	Label(tsparams.LabelEndToEndDeployment), func() {
		var (
			ibiImageSetName string
		)
		BeforeAll(func() {
			if MGMTConfig.Cluster == nil {
				Skip("Failed to collect cluster info")
			}

			if MGMTConfig.SeedClusterInfo == nil || MGMTConfig.SeedClusterInfo.SeedClusterOCPVersion == "" {
				Skip("Seed clusterinfo not supplied")
			}

			splitOCPVersion := strings.Split(MGMTConfig.SeedClusterInfo.SeedClusterOCPVersion, ".")

			if len(splitOCPVersion) < 2 {
				Skip("Could not determine short OCP version from seed cluster ocp version")
			}

			ibiImageSetName = strings.Join(splitOCPVersion[:2], ".")
		})

		It("is successful in connected environment", func() {

			tsparams.ReporterNamespacesToDump[MGMTConfig.Cluster.Info.ClusterName] = "spoke namespace"

			By("Create namespace for IBI installation")
			_, err := namespace.NewBuilder(APIClient, MGMTConfig.Cluster.Info.ClusterName).Create()
			Expect(err).NotTo(HaveOccurred(), "error creating namespace")

			By("Get pull secret from cluster")
			spokePullSecret, err := secret.Pull(APIClient, "pull-secret", "openshift-config")
			Expect(err).NotTo(HaveOccurred(), "error getting pull-secret from hub cluster")

			By("Create pull secret for spoke cluster")
			spokePullSecret.Definition.Name = MGMTConfig.Cluster.Info.ClusterName
			spokePullSecret.Definition.Namespace = MGMTConfig.Cluster.Info.ClusterName
			spokePullSecret.Definition.ResourceVersion = ""
			_, err = spokePullSecret.Create()
			Expect(err).NotTo(HaveOccurred(), "error creating spoke pull-secret")

			for host, info := range MGMTConfig.Cluster.Info.Hosts {
				By("Create baremetalhost secret for " + host)
				_, err = secret.NewBuilder(
					APIClient, host, MGMTConfig.Cluster.Info.ClusterName, v1.SecretTypeOpaque).WithData(map[string][]byte{
					"username": []byte(info.BMC.User),
					"password": []byte(info.BMC.Password),
				}).Create()
				Expect(err).NotTo(HaveOccurred(), "error creating bmh secret")

				By("Create baremetalhost for " + host)
				hostBMH := bmh.NewBuilder(
					APIClient, host, MGMTConfig.Cluster.Info.ClusterName, info.BMC.URLv4, host, info.BMC.MACAddress, "UEFI")
				hostBMH.Definition.Spec.AutomatedCleaningMode = "disabled"

				_, err = hostBMH.Create()
				Expect(err).NotTo(HaveOccurred(), "error creating baremetalhost")
			}

			By("Create cluster deployment for IBI installation")
			_, err = hive.NewABMClusterDeploymentBuilder(
				APIClient, MGMTConfig.Cluster.Info.ClusterName,
				MGMTConfig.Cluster.Info.ClusterName, MGMTConfig.Cluster.Info.ClusterName,
				MGMTConfig.Cluster.Info.BaseDomain, MGMTConfig.Cluster.Info.ClusterName, metav1.LabelSelector{
					MatchLabels: map[string]string{
						"dummy": "label",
					},
				}).WithPullSecret(MGMTConfig.Cluster.Info.ClusterName).Create()
			Expect(err).NotTo(HaveOccurred(), "error creating cluster deployment")

			var snoNodeName string

			for hostname := range MGMTConfig.Cluster.Info.Hosts {
				snoNodeName = hostname

				break
			}

			By("Create imageclusterinstall for IBI installation")
			imageClusterInstall := ibi.NewImageClusterInstallBuilder(
				APIClient, MGMTConfig.Cluster.Info.ClusterName, MGMTConfig.Cluster.Info.ClusterName, ibiImageSetName).
				WithClusterDeployment(MGMTConfig.Cluster.Info.ClusterName).WithHostname(snoNodeName).
				WithVersion(MGMTConfig.SeedClusterInfo.SeedClusterOCPVersion).
				WithMachineNetwork(MGMTConfig.Cluster.Info.MachineCIDR.IPv4)

			if MGMTConfig.PublicSSHKey != "" {
				imageClusterInstall.WithSSHKey(MGMTConfig.PublicSSHKey)
			}

			imageClusterInstall.Definition.Spec.BareMetalHostRef = &ibiv1alpha1.BareMetalHostReference{}
			imageClusterInstall.Definition.Spec.BareMetalHostRef.Name = snoNodeName
			imageClusterInstall.Definition.Spec.BareMetalHostRef.Namespace = MGMTConfig.Cluster.Info.ClusterName
			_, err = imageClusterInstall.Create()
			Expect(err).NotTo(HaveOccurred(), "error creating imageclusterinstall")

			Eventually(func() (bool, error) {
				imageClusterInstall.Object, err = imageClusterInstall.Get()
				if err != nil {
					return false, err
				}

				condition, err := imageClusterInstall.GetCompletedCondition()
				if err != nil {
					return false, err
				}

				return condition.Status == "True" && condition.Reason == ibiv1alpha1.InstallSucceededReason, nil

			}).WithTimeout(time.Minute*20).WithPolling(time.Second*5).Should(
				BeTrue(), "error waiting for imageclusterinstall to complete")
		})
	})
