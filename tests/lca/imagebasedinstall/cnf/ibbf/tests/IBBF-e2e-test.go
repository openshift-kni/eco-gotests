package ibbftests

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/configmap"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	siteconfigv1alpha1 "github.com/openshift-kni/eco-goinfra/pkg/schemes/siteconfig/v1alpha1"
	"github.com/openshift-kni/eco-goinfra/pkg/siteconfig"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedinstall/cnf/ibbf/internal/gitdetails"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedinstall/cnf/ibbf/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedinstall/cnf/internal/cnfinittools"

	"time"
)

const (
	trueStatus  = "True"
	falseStatus = "False"
)

var _ = Describe(
	"Performing Image-Based Break/Fix Flow",
	Label(tsparams.LabelIBBFe2e), func() {

		It("IBBF Flow", reportxml.ID("78333"), func() {

			By("enabling cluster reinstallation in SiteconfigOperator")

			scoConfig, err := configmap.Pull(cnfinittools.TargetHubAPIClient, "siteconfig-operator-configuration",
				tsparams.RHACMNamespace)
			Expect(err).NotTo(HaveOccurred(), "error pulling siteconfig-operator-configuration configmap")

			scoConfig.Definition.Data["allowReinstalls"] = "true"
			_, err = scoConfig.Update()
			Expect(err).NotTo(HaveOccurred(), "error updating siteconfig-operator-configuration configmap to allow reinstalls")

			// Create test configmap in spoke namespace on hub
			configMap, err := configmap.NewBuilder(
				cnfinittools.TargetHubAPIClient, tsparams.TestCMName, tsparams.SpokeNamespace).
				WithData(map[string]string{"testValue": "true"}).Create()
			Expect(err).NotTo(HaveOccurred(), "error creating configmap")

			// Apply preservation label to configmap
			configMap.Definition.ObjectMeta.SetLabels(map[string]string{"siteconfig.open-cluster-management.io/preserve": ""})

			// Update configmap
			_, err = configMap.Update()
			Expect(err).NotTo(HaveOccurred(), "error updating configmap")

			clusterInstace, err := siteconfig.PullClusterInstance(
				cnfinittools.TargetHubAPIClient, "helix54", tsparams.SpokeNamespace)
			Expect(err).NotTo(HaveOccurred(), "error pulling clusterinstance")

			By("changing policies app to point to IBBF test target directory  ", func() {
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

		It("verifying test configmap was preserved post-IBBF", reportxml.ID("TBF"), func() {

			_, err := configmap.Pull(cnfinittools.TargetHubAPIClient, tsparams.TestCMName, tsparams.SpokeNamespace)
			Expect(err).NotTo(HaveOccurred(), "Preserved configmap is missing after IBBF")

		})

		AfterEach(func() {

		})
		It("IBBF Workflow", reportxml.ID("78333"), func() {

		})
	})
