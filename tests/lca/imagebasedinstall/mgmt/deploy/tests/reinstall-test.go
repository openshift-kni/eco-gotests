package deploy_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/bmh"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/clusterversion"
	"github.com/openshift-kni/eco-goinfra/pkg/configmap"
	"github.com/openshift-kni/eco-goinfra/pkg/hive"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	siteconfigv1alpha1 "github.com/openshift-kni/eco-goinfra/pkg/schemes/siteconfig/v1alpha1"
	"github.com/openshift-kni/eco-goinfra/pkg/siteconfig"

	v1 "k8s.io/api/core/v1"

	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedinstall/mgmt/deploy/internal/tsparams"
	. "github.com/openshift-kni/eco-gotests/tests/lca/imagebasedinstall/mgmt/internal/mgmtinittools"
)

var _ = Describe(
	"Performing re-install with image based installation",
	Ordered,
	Label(tsparams.LabelReinstall), func() {
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

		It("through IBI operator is successful in a connected environment with static networking",
			reportxml.ID("no-test"), func() {
				if !MGMTConfig.StaticNetworking {
					Skip("Cluster is deployed without static networking")
				}

				if MGMTConfig.SiteConfig {
					Skip("Cluster is deployed with siteconfig operator")
				}

				if MGMTConfig.ReinstallConfigFile == "" {
					Skip("Reinstall configuration not supplied")
				}

				tsparams.ReporterNamespacesToDump[MGMTConfig.Cluster.Info.ClusterName] = reporterNamespaceToDump

				createIBIOResouces(ipv4AddrFamily)

				By("Load admin kubeconfig secret")
				var adminKubeconfigSecret *v1.Secret
				adminKubeconfigFile, err := os.ReadFile(MGMTConfig.Reinstall.AdminKubeConfigSecretFile)
				Expect(err).NotTo(HaveOccurred(), "error reading %s", MGMTConfig.Reinstall.AdminKubeConfigSecretFile)

				err = json.Unmarshal(adminKubeconfigFile, &adminKubeconfigSecret)
				Expect(err).NotTo(HaveOccurred(),
					"error unmarshalling %s to secret", MGMTConfig.Reinstall.AdminKubeConfigSecretFile)

				adminKubeconfigContent, ok := adminKubeconfigSecret.Data["kubeconfig"]
				Expect(ok).To(BeTrue(), "error checking for kubeconfig key from admin kubeconfig secret")

				By("Writing spoke admin kubeconfig to file")

				err = os.WriteFile("/tmp/spoke-reinstall-kubeconfig", adminKubeconfigContent, 0644)
				Expect(err).NotTo(HaveOccurred(), "error writing spoke kubeconfig to file")

				spokeClient = clients.New("/tmp/spoke-reinstall-kubeconfig")
				Expect(spokeClient).NotTo(BeNil(), "error creating client from spoke kubeconfig file")

				targetClusterVersion, err := clusterversion.Pull(spokeClient)
				Expect(err).NotTo(HaveOccurred(), "error pulling target cluster OCP version")
				Expect(targetClusterVersion.Object.Status.Desired.Version).To(
					Equal(MGMTConfig.SeedClusterInfo.SeedClusterOCPVersion),
					"error: target cluster version does not match seedimage cluster version")
				Expect(MGMTConfig.Reinstall.ClusterIdentity).To(Equal(string(targetClusterVersion.Object.Spec.ClusterID)),
					"error: reinstalled cluster has different cluster identity than original cluster")
			})

		It("through siteconfig operator is successful in an IPv4 environment with DHCP networking",
			reportxml.ID("no-testcase"), func() {
				if MGMTConfig.StaticNetworking {
					Skip("Cluster is deployed with static networking")
				}

				if !MGMTConfig.SiteConfig {
					Skip("Cluster is deployed without siteconfig operator")
				}

				if MGMTConfig.SeedClusterInfo.Proxy.HTTPProxy != "" || MGMTConfig.SeedClusterInfo.Proxy.HTTPSProxy != "" {
					Skip("Cluster installed with proxy")
				}

				tsparams.ReporterNamespacesToDump[MGMTConfig.Cluster.Info.ClusterName] = reporterNamespaceToDump

				By("Load original spoke api client")
				spokeClient := getSpokeClient()

				By("Enable cluster reinstallation")
				scoConfig, err := configmap.Pull(APIClient, "siteconfig-operator-configuration", tsparams.RHACMNamespace)
				Expect(err).NotTo(HaveOccurred(), "error pulling siteconfig-operator-configuration configmap")

				scoConfig.Definition.Data["allowReinstalls"] = "true"
				_, err = scoConfig.Update()
				Expect(err).NotTo(HaveOccurred(), "error updating siteconfig-operator-configuration configmap to allow reinstalls")

				By("Pulling existing clusterdeployment")
				clusterDeployment, err := hive.PullClusterDeployment(APIClient, MGMTConfig.Cluster.Info.ClusterName,
					MGMTConfig.Cluster.Info.ClusterName)
				Expect(err).NotTo(HaveOccurred(), "error pulling clusterdeployment")

				originalClusterID := clusterDeployment.Object.Spec.ClusterMetadata.ClusterID

				By("Pulling existing clusterinstance")
				clusterInstace, err := siteconfig.PullClusterInstance(APIClient, MGMTConfig.Cluster.Info.ClusterName,
					MGMTConfig.Cluster.Info.ClusterName)
				Expect(err).NotTo(HaveOccurred(), "error pulling clusterinstance")

				for idx := range clusterInstace.Definition.Spec.Nodes {
					for _, host := range MGMTConfig.Cluster.Info.Hosts {
						clusterInstace.Definition.Spec.Nodes[idx].BmcAddress = host.BMC.URLv4
						clusterInstace.Definition.Spec.Nodes[idx].BootMACAddress = host.BMC.MACAddress
					}
				}

				if clusterInstace.Definition.Spec.Reinstall == nil {
					clusterInstace.Definition.Spec.Reinstall = new(siteconfigv1alpha1.ReinstallSpec)
				}

				clusterInstace.Definition.Spec.Reinstall.Generation = MGMTConfig.ReinstallGenerationLabel
				clusterInstace.Definition.Spec.Reinstall.PreservationMode = "ClusterIdentity"

				By("Updating clusterinstance for reinstallation")
				_, err = clusterInstace.Update(false)
				Expect(err).NotTo(HaveOccurred(), "error updating clusterinstance")

				for host := range MGMTConfig.Cluster.Info.Hosts {
					dataImage, err := bmh.PullDataImage(APIClient, host,
						MGMTConfig.Cluster.Info.ClusterName)
					Expect(err).NotTo(HaveOccurred(), "error pulling dataimage from cluster")

					err = removeDataImageFinalizer(APIClient, dataImage)
					Expect(err).NotTo(HaveOccurred(), "error removing dataimage finalizer")
				}

				By("Waiting for clusterinstance re-installation to trigger")
				Eventually(func() (bool, error) {
					clusterInstace.Object, err = clusterInstace.Get()
					if err != nil {
						return false, err
					}

					if clusterInstace.Object.Status.Reinstall == nil || clusterInstace.Object.Status.Reinstall.Conditions == nil {
						return false, nil
					}

					for _, condition := range clusterInstace.Object.Status.Reinstall.Conditions {
						if condition.Type == string(siteconfigv1alpha1.ReinstallRequestProcessed) {
							return condition.Status == "True" && condition.Reason == string(siteconfigv1alpha1.Completed), nil
						}
					}

					return false, nil
				}).WithTimeout(time.Minute*40).WithPolling(time.Second*10).Should(
					BeTrue(), "error waiting for clusterinstance to begin re-install")

				By("Waiting for clusterinstance to start provisioning")
				Eventually(func() (bool, error) {
					clusterInstace.Object, err = clusterInstace.Get()
					if err != nil {
						return false, err
					}

					for _, condition := range clusterInstace.Object.Status.Conditions {
						if condition.Type == string(siteconfigv1alpha1.ClusterProvisioned) {
							return condition.Status == falseStatus && condition.Reason == string(siteconfigv1alpha1.InProgress), nil

						}
					}

					return false, nil
				}).WithTimeout(time.Minute*5).WithPolling(time.Second*10).Should(
					BeTrue(), "error waiting for clusterinstance to start provisioning")

				By("Waiting for clusterinstance to finish provisioning")
				Eventually(func() (bool, error) {
					clusterInstace.Object, err = clusterInstace.Get()
					if err != nil {
						return false, err
					}

					for _, condition := range clusterInstace.Object.Status.Conditions {
						if condition.Type == string(siteconfigv1alpha1.ClusterProvisioned) {
							return condition.Status == trueStatus && condition.Reason == string(siteconfigv1alpha1.Completed), nil

						}
					}

					return false, nil
				}).WithTimeout(time.Minute*30).WithPolling(time.Second*10).Should(
					BeTrue(), "error waiting for clusterinstance to finish provisioning")

				By("Pull spoke cluster version using original client")
				targetClusterVersion, err := clusterversion.Pull(spokeClient)
				Expect(err).NotTo(HaveOccurred(), "error pulling target cluster OCP version")
				Expect(targetClusterVersion.Object.Status.Desired.Version).To(
					Equal(MGMTConfig.SeedClusterInfo.SeedClusterOCPVersion),
					"error: target cluster version does not match seedimage cluster version")
				Expect(originalClusterID).To(Equal(string(targetClusterVersion.Object.Spec.ClusterID)),
					"error: reinstalled cluster has different cluster identity than original cluster")
			})
	})

func removeDataImageFinalizer(apiClient *clients.Settings, dataImage *bmh.DataImageBuilder) error {
	if apiClient == nil {
		return fmt.Errorf("cannot use nil apiClient to remove dataImage finalizer")
	}

	if dataImage == nil {
		return fmt.Errorf("cannot update nil dataImage")
	}

	dataImage.Definition.ObjectMeta.Finalizers = nil

	err := apiClient.Update(context.TODO(), dataImage.Definition)

	if err != nil {
		return err
	}

	dataImage.Object = dataImage.Definition

	return nil
}
