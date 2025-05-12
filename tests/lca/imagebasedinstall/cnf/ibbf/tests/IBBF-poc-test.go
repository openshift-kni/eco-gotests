package tests

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/configmap"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	siteconfigv1alpha1 "github.com/openshift-kni/eco-goinfra/pkg/schemes/siteconfig/v1alpha1"
	"github.com/openshift-kni/eco-goinfra/pkg/siteconfig"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedinstall/cnf/ibbf/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedinstall/cnf/internal/cnfclusterinfo"
	. "github.com/openshift-kni/eco-gotests/tests/lca/imagebasedinstall/cnf/internal/cnfinittools"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedinstall/cnf/internal/gitdetails"
	"time"
)

const (
	trueStatus  = "True"
	falseStatus = "False"
)

var _ = Describe(
	"Performing upgrade prep abort flow",
	Label(tsparams.LabelPrepAbortFlow), func() {

		BeforeEach(func() {

			By("Fetching target sno cluster name", func() {
				err := cnfclusterinfo.PreUpgradeClusterInfo.SaveClusterInfo()
				Expect(err).ToNot(HaveOccurred(), "Failed to extract target sno cluster name")

				tsparams.TargetSnoClusterName = cnfclusterinfo.PreUpgradeClusterInfo.Name
			})
		})

		It("IBBF Flow", reportxml.ID("68956"), func() {

			By("Saving target sno cluster info prior to IBBF", func() {
				err := cnfclusterinfo.PostUpgradeClusterInfo.SaveClusterInfo()
				Expect(err).ToNot(HaveOccurred(), "Failed to collect and save target sno cluster before IBBF")
			})

			// Enalble reinstall in SiteconfigOperator
			By("Enable cluster reinstallation")

			scoConfig, err := configmap.Pull(TargetHubAPIClient, "siteconfig-operator-configuration", tsparams.RHACMNamespace)
			Expect(err).NotTo(HaveOccurred(), "error pulling siteconfig-operator-configuration configmap")

			scoConfig.Definition.Data["allowReinstalls"] = "true"
			_, err = scoConfig.Update()
			Expect(err).NotTo(HaveOccurred(), "error updating siteconfig-operator-configuration configmap to allow reinstalls")

			// Create test configmap in spoke namespace on hub
			cm, err := configmap.NewBuilder(TargetHubAPIClient, "test", tsparams.SpokeNamespace).
				WithData(map[string]string{"testValue": "true"}).Create()
			Expect(err).NotTo(HaveOccurred())
			// Apply preservation label to configmap
			cm.Definition.ObjectMeta.SetLabels(map[string]string{"siteconfig.open-cluster-management.io/preserve": ""})
			//Update configmap
			cm.Update()

			clusterInstace, err := siteconfig.PullClusterInstance(TargetHubAPIClient, "helix54", tsparams.SpokeNamespace)
			Expect(err).NotTo(HaveOccurred(), "error pulling clusterinstance")

			By("Change Policies App to IBBF  ", func() {
				exists, err := gitdetails.UpdateArgoCdAppGitPath(
					tsparams.ArgoCdClustersAppName, tsparams.IBBFTestPath, true)
				if !exists {
					Skip(err.Error())
				}
			})

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
		})

		It("Tests for preserved configmap", reportxml.ID("TBF"), func() {

			_, err := configmap.Pull(TargetHubAPIClient, "test", tsparams.SpokeNamespace)
			Expect(err).NotTo(HaveOccurred(), "Preserved configmap is missing after IBBF")

		})

		AfterEach(func() {

		})
		//Update report ID
		It("IBBF Workflow", reportxml.ID("TBD"), func() {

		})
	})
