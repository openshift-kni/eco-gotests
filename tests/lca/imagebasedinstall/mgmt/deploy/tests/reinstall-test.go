package deploy_test

import (
	"encoding/json"
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/clusterversion"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"

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

				if MGMTConfig.ReinstallManifestDir == "" {
					Skip("Reinstall manifest directory not supplied")
				}

				tsparams.ReporterNamespacesToDump[MGMTConfig.Cluster.Info.ClusterName] = reporterNamespaceToDump

				createIBIOResouces(ipv4AddrFamily)

				By("Load admin kubeconfig secret")
				var adminKubeconfigSecret *v1.Secret
				adminKubeconfigFile, err := os.ReadFile(MGMTConfig.ReinstallManifestDir + "/admin-kubeconfig.json")
				Expect(err).NotTo(HaveOccurred(), "error reading admin-kubeconfig.json")

				err = json.Unmarshal(adminKubeconfigFile, &adminKubeconfigSecret)
				Expect(err).NotTo(HaveOccurred(), "error unmarshalling admin-kubeconfig.json to secret")

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
			})
	})
